package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	googleOauthConfig *oauth2.Config
	githubOauthConfig *oauth2.Config
	oauthStateString  = "pseudo-random"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	sessions          = map[string]string{}
	sessionsMutex     sync.Mutex
)

func init() {
	loadEnv()

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if googleClientID == "" || googleClientSecret == "" {
		log.Fatal("Google OAuth credentials are not set in environment variables")
	}

	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}

	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	if githubClientID == "" || githubClientSecret == "" {
		log.Fatal("GitHub OAuth credentials are not set in environment variables")
	}

	githubOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/auth/github/callback",
		ClientID:     githubClientID,
		ClientSecret: githubClientSecret,
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		log.Println("État OAuth invalide")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	log.Println("Code reçu:", code)

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Println("Échange de code échoué:", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	log.Println("Jeton reçu:", token)

	response, err := http.Get(googleUserInfoURL + "?access_token=" + token.AccessToken)
	if err != nil {
		log.Println("Échec de l'obtention des informations utilisateur:", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer response.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&userInfo); err != nil {
		log.Println("Échec du décodage JSON:", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	log.Println("Informations utilisateur Google:", userInfo)
	handleLoginGoogle(w, r, userInfo)
}

func handleLoginGoogle(w http.ResponseWriter, r *http.Request, userInfo map[string]interface{}) {
	username, ok := userInfo["given_name"].(string)
	if !ok {
		log.Println("Erreur: le nom d'utilisateur n'est pas une chaîne")
		http.Error(w, "Erreur lors de l'obtention du nom d'utilisateur", http.StatusInternalServerError)
		return
	}

	email, ok := userInfo["email"].(string)
	if !ok {
		log.Println("Erreur: l'email n'est pas une chaîne")
		http.Error(w, "Erreur lors de l'obtention de l'email", http.StatusInternalServerError)
		return
	}

	log.Println("Nom d'utilisateur:", username)
	log.Println("Email:", email)

	var dbUsername string
	checkUser := true
	for checkUser {
		query := `SELECT username FROM Account WHERE username = ?`
		err := db.QueryRow(query, username).Scan(&dbUsername)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Println("Aucun utilisateur trouvé avec le nom d'utilisateur:", username)
				insertUserSQL := `INSERT INTO Account (username, password, mail, rank) VALUES (?, ?, ?, ?)`
				statement, err := db.Prepare(insertUserSQL)
				if err != nil {
					http.Error(w, "Erreur lors de la préparation de l'instruction", http.StatusInternalServerError)
					return
				}
				defer statement.Close()

				_, err = statement.Exec(username, nil, email, "user")
				if err != nil {
					http.Error(w, "Erreur lors de l'insertion de l'utilisateur dans la base de données", http.StatusInternalServerError)
					return
				}
				log.Println("Utilisateur inséré dans la base de données")
			} else {
				log.Println("Erreur lors de la requête à la base de données:", err)
				http.Error(w, "Erreur lors de la requête à la base de données", http.StatusInternalServerError)
				return
			}
			checkUser = false
		} else {
			checkUser = false
		}
	}

	sessionToken := fmt.Sprintf("%d", time.Now().UnixNano())
	log.Println("Création d'une nouvelle session avec le jeton:", sessionToken)

	sessionsMutex.Lock()
	sessions[sessionToken] = username
	sessionsMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	log.Println("Jeton de session défini:", sessionToken)
	http.Redirect(w, r, "/Forum?username="+username, http.StatusSeeOther)
}

func isUserLoggedIn(r *http.Request) (bool, string) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			log.Println("Aucun cookie de jeton de session trouvé")
			return false, ""
		}
		log.Println("Erreur lors de la récupération du cookie:", err)
		return false, ""
	}

	sessionToken := cookie.Value
	log.Println("Jeton de session récupéré du cookie:", sessionToken)

	sessionsMutex.Lock()
	username, exists := sessions[sessionToken]
	sessionsMutex.Unlock()

	if !exists {
		log.Println("Jeton de session non trouvé dans la map des sessions:", sessionToken)
		return false, ""
	}

	return true, username
}

func HandleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	url := githubOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func HandleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		log.Println("invalid oauth state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := githubOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Println("code exchange failed: ", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Créer une nouvelle requête HTTP avec le jeton d'accès dans l'en-tête Authorization
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		log.Println("failed to create request: ", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	req.Header.Set("Authorization", "token "+token.AccessToken)

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Println("failed getting user info: ", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer response.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&userInfo); err != nil {
		log.Println("failed to decode JSON: ", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	fmt.Println("GitHub User Info:", userInfo)
	// Handle the user authentication or registration using GitHub account
	handleLoginGithub(w, r, userInfo)
}

func handleLoginGithub(w http.ResponseWriter, r *http.Request, userInfo map[string]interface{}) {
	username, ok := userInfo["login"].(string)
	if !ok {
		log.Println("Erreur: le nom d'utilisateur n'est pas une chaîne")
		http.Error(w, "Erreur lors de l'obtention du nom d'utilisateur", http.StatusInternalServerError)
		return
	}

	email, ok := userInfo["email"]
	if !ok {
		log.Println("Erreur: l'email n'est pas une chaîne")
		http.Error(w, "Erreur lors de l'obtention de l'email", http.StatusInternalServerError)
		return
	}

	log.Println("Nom d'utilisateur:", username)
	log.Println("Email:", email)

	var dbUsername string
	checkUser := true
	for checkUser {
		query := `SELECT username FROM Account WHERE username = ?`
		err := db.QueryRow(query, username).Scan(&dbUsername)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Println("Aucun utilisateur trouvé avec le nom d'utilisateur:", username)
				insertUserSQL := `INSERT INTO Account (username, password, mail, rank) VALUES (?, ?, ?, ?)`
				statement, err := db.Prepare(insertUserSQL)
				if err != nil {
					http.Error(w, "Erreur lors de la préparation de l'instruction", http.StatusInternalServerError)
					return
				}
				defer statement.Close()

				_, err = statement.Exec(username, nil, email, "user")
				if err != nil {
					http.Error(w, "Erreur lors de l'insertion de l'utilisateur dans la base de données", http.StatusInternalServerError)
					return
				}
				log.Println("Utilisateur inséré dans la base de données")
			} else {
				log.Println("Erreur lors de la requête à la base de données:", err)
				http.Error(w, "Erreur lors de la requête à la base de données", http.StatusInternalServerError)
				return
			}
			checkUser = false
		} else {
			checkUser = false
		}
	}

	sessionToken := fmt.Sprintf("%d", time.Now().UnixNano())
	log.Println("Création d'une nouvelle session avec le jeton:", sessionToken)

	sessionsMutex.Lock()
	sessions[sessionToken] = username
	sessionsMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	log.Println("Jeton de session défini:", sessionToken)
	http.Redirect(w, r, "/Forum?username="+username, http.StatusSeeOther)
}

func SetupRoutes() {
	http.HandleFunc("/auth/google/login", HandleGoogleLogin)
	http.HandleFunc("/auth/google/callback", HandleGoogleCallback)
	http.HandleFunc("/auth/github/login", HandleGitHubLogin)
	http.HandleFunc("/auth/github/callback", HandleGitHubCallback)
}
