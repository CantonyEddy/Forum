package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
)

var (
	googleOauthConfig *oauth2.Config
	githubOauthConfig *oauth2.Config
	oauthStateString  = "pseudo-random"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
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
		log.Println("invalid oauth state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Println("code exchange failed: ", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	response, err := http.Get(googleUserInfoURL + "?access_token=" + token.AccessToken)
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

	fmt.Println("Google User Info:", userInfo)
	// Handle the user authentication or registration using Google account
	http.Redirect(w, r, "/Forum", http.StatusSeeOther)
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

	response, err := http.Get("https://api.github.com/user?access_token=" + token.AccessToken)
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
	http.Redirect(w, r, "/Forum", http.StatusSeeOther)
}

func SetupRoutes() {
	http.HandleFunc("/auth/google/login", HandleGoogleLogin)
	http.HandleFunc("/auth/google/callback", HandleGoogleCallback)
	http.HandleFunc("/auth/github/login", HandleGitHubLogin)
	http.HandleFunc("/auth/github/callback", HandleGitHubCallback)
}
