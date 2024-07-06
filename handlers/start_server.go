package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // Import the sqlite3 driver
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var tmpl *template.Template
var tmpl_register *template.Template
var tmpl_login *template.Template
var tmpl_main_page *template.Template
var tmpl_admin_pannel *template.Template
var tmpl_create_poste *template.Template
var tmpl_create_image *template.Template
var db *sql.DB
var sessions = map[string]string{}
var sessionsMutex sync.Mutex

func StartServer() {
	var err error

	db, err = sql.Open("sqlite3", "BDD/DBForum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	err = ensureUploadDir()
	if err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("sqlite3", "BDD/DBForum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTables(db) // Ensure tables are created

	tmpl, err = template.New("index").ParseFiles(filepath.Join(wd, "Static", "Templates", "index.html"))
	if err != nil {
		panic(err)
	}

	tmpl_register, err = template.New("register").ParseFiles(filepath.Join(wd, "Static", "Templates", "register.html"))
	if err != nil {
		panic(err)
	}

	tmpl_login, err = template.New("login").ParseFiles(filepath.Join(wd, "Static", "Templates", "login.html"))
	if err != nil {
		panic(err)
	}

	tmpl_main_page, err = template.New("Forum").ParseFiles(filepath.Join(wd, "Static", "Templates", "forumMainPage.html"))
	if err != nil {
		panic(err)
	}

	tmpl_admin_pannel, err = template.New("adminPannel").ParseFiles(filepath.Join(wd, "Static", "Templates", "adminPannel.html"))
	if err != nil {
		panic(err)
	}

	tmpl_create_poste, err = template.New("createPost").ParseFiles(filepath.Join(wd, "Static", "Templates", "createPost.html"))
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.Dir(filepath.Join(wd, "Static")))

	// Handler pour servir les fichiers statiques
	http.Handle("/Static/", http.StripPrefix("/Static/", fileServer))

	// Route pour les fichiers CSS
	http.HandleFunc("/Static/CSS/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}
		http.ServeFile(w, r, filepath.Join(wd, r.URL.Path))
	})

	// Route pour les fichiers JS
	http.HandleFunc("/Static/JavaScript/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		http.ServeFile(w, r, filepath.Join(wd, r.URL.Path))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if !isAuthenticated(r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			} else {
				http.ServeFile(w, r, filepath.Join(wd, "Static", "Templates", "index.html"))
			}
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/register" {
			http.ServeFile(w, r, filepath.Join(wd, "Static", "Templates", "register.html"))
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.ServeFile(w, r, filepath.Join(wd, "Static", "Templates", "login.html"))
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/Forum", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Forum" {
			data := handleHome(w, r)
			tmpl_main_page.Execute(w, data)
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/adminPannel", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/adminPannel" {
			rank, err := getSessionRank(r)
			if err != nil {
				http.Error(w, "Error getting rank", http.StatusInternalServerError)
				return
			}
			if rank == "admin" {
				data := handleAccount(w)
				tmpl_admin_pannel.Execute(w, data)
			} else {
				http.Redirect(w, r, "/Forum", http.StatusSeeOther)
			}
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/createPost", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/createPost" {
			data := handleHome(w, r)
			tmpl_create_poste.Execute(w, data)
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/registerUser", handleRegister)
	http.HandleFunc("/loginUser", handleLogin)
	http.HandleFunc("/upgradeRank", handleRankUp)
	http.HandleFunc("/addPost", createPost)
	http.HandleFunc("/likePost", handleLikePost)

	http.HandleFunc("/auth/google/login", HandleGoogleLogin)
	http.HandleFunc("/auth/google/callback", HandleGoogleCallback)
	http.HandleFunc("/auth/github/login", HandleGitHubLogin)
	http.HandleFunc("/auth/github/callback", HandleGitHubCallback)

	fmt.Println("Pour accéder à la page web -> http://localhost:8080/")
	err1 := http.ListenAndServe(":8080", nil)
	if err1 != nil {
		log.Fatal("ListenAndServe: ", err1)
	}
}

func createTables(db *sql.DB) {
	createAccountTableSQL := `CREATE TABLE IF NOT EXISTS Account (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL,
        password TEXT,
        mail TEXT NOT NULL,
        rank TEXT NOT NULL
    );`
	_, err := db.Exec(createAccountTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	dropPostTableSQL := `DROP TABLE IF EXISTS Post`
	_, err = db.Exec(dropPostTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	createPostTableSQL := `CREATE TABLE IF NOT EXISTS Post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_name TEXT NOT NULL,
		creator_id INTEGER NOT NULL,
		post_message TEXT NOT NULL,
		category_name TEXT,
		likes INTEGER DEFAULT 0,
    	dislikes INTEGER DEFAULT 0,
		FOREIGN KEY (creator_id) REFERENCES Account(id)
	);`
	_, err = db.Exec(createPostTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	createImageTableSQL := `CREATE TABLE IF NOT EXISTS Image (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		link TEXT NOT NULL,
		post_id INTEGER NOT NULL,
		FOREIGN KEY (post_id) REFERENCES Post(id)
	);`

	_, err = db.Exec(createImageTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	createLikeTableSQL := `CREATE TABLE IF NOT EXISTS PostLikes (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		liked BOOLEAN NOT NULL,
		FOREIGN KEY (post_id) REFERENCES Post(id),
		FOREIGN KEY (user_id) REFERENCES Account(id),
		UNIQUE(post_id, user_id)
	);`
	_, err = db.Exec(createLikeTableSQL)
	if err != nil {
		log.Fatal(err)
	}

}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirmPassword")

	if password != confirmPassword {
		http.Error(w, "Passwords do not match", http.StatusBadRequest)
		return
	}

	// Insert the user into the database
	insertUserSQL := `INSERT INTO Account (username, password, mail, rank) VALUES (?, ?, ?, ?)`
	statement, err := db.Prepare(insertUserSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(username, password, email, "user")
	if err != nil {
		http.Error(w, "Error inserting user into database", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "User %s registered successfully!", username)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Query the user from the database
	var dbUsername, dbPassword string
	query := "SELECT username, password FROM Account WHERE username = ?"
	err := db.QueryRow(query, username).Scan(&dbUsername, &dbPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		} else {
			http.Error(w, "Error querying database", http.StatusInternalServerError)
		}
		return
	}

	if password != dbPassword {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate a session token
	sessionToken := generateSessionToken()
	sessionsMutex.Lock()
	sessions[sessionToken] = username
	sessionsMutex.Unlock()

	// Set the session token as a cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(24 * time.Hour),
	})

	fmt.Fprintf(w, "User %s logged in successfully!", username)
}

func generateSessionToken() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func ensureUploadDir() error {
	uploadDir := filepath.Join("Static", "upload")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		return os.MkdirAll(uploadDir, os.ModePerm)
	}
	return nil
}

func handleHome(w http.ResponseWriter, r *http.Request) any {
	db, err := sql.Open("sqlite3", "BDD/DBForum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	posts, err := getAllPosts(db)
	if err != nil {
		http.Error(w, "Failed to get posts", http.StatusInternalServerError)
		return nil
	}

	user, err := getSessionUser(r)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return nil
	}

	data := struct {
		User  string
		Posts []Post
	}{
		User:  user,
		Posts: posts,
	}

	return data
}

func handleAccount(w http.ResponseWriter) any {
	db, err := sql.Open("sqlite3", "BDD/DBForum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	accounts, err := getAllAccounts(db)
	if err != nil {
		http.Error(w, "Failed to get accounts", http.StatusInternalServerError)
		return nil
	}

	data := struct {
		Accounts []Account
	}{
		Accounts: accounts,
	}

	return data
}

func getAllAccounts(db *sql.DB) ([]Account, error) {
	rows, err := db.Query("SELECT id, username, mail, rank FROM Account")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.Username, &account.Mail, &account.Rank); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func getAllPosts(db *sql.DB) ([]Post, error) {
	rows, err := db.Query("SELECT id, post_name, creator_id, post_message, category_name, likes, dislikes FROM Post")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.PostName, &post.CreatorID, &post.PostMessage, &post.CategoryName, &post.Likes, &post.Dislikes); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

type Post struct {
	ID           int
	PostName     string
	CreatorID    int
	PostMessage  string
	CategoryName string
	Likes        int
	Dislikes     int
}

type Account struct {
	ID       int
	Username string
	Mail     string
	Rank     string
}

func getSessionUser(r *http.Request) (string, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return "", err
	}

	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	username, exists := sessions[cookie.Value]
	if !exists {
		return "", fmt.Errorf("session not found")
	}

	return username, nil
}

func getSessionRank(r *http.Request) (string, error) {
	user, err := getSessionUser(r)
	if err != nil {
		return "", err
	}

	var rank string
	query := "SELECT rank FROM Account WHERE username = ?"
	err = db.QueryRow(query, user).Scan(&rank)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user not found")
		}
		return "", err
	}

	return rank, nil
}

func isAuthenticated(r *http.Request) bool {
	_, err := getSessionUser(r)
	return err == nil
}

func handleRankUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	newRank := r.FormValue("rank")

	updateRankSQL := `UPDATE Account SET rank = ? WHERE username = ?`
	statement, err := db.Prepare(updateRankSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(newRank, username)
	if err != nil {
		http.Error(w, "Error updating user rank in database", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "User %s rank updated to %s successfully!", username, newRank)
}

func handleLikePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	postID := r.FormValue("post_id")
	like := r.FormValue("like") == "true"

	username, err := getSessionUser(r)
	if err != nil {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var userID int
	err = db.QueryRow("SELECT id FROM Account WHERE username = ?", username).Scan(&userID)
	if err != nil {
		http.Error(w, "Error getting user ID", http.StatusInternalServerError)
		return
	}

	likeSQL := `INSERT INTO PostLikes (post_id, user_id, liked) VALUES (?, ?, ?)
		ON CONFLICT(post_id, user_id) DO UPDATE SET liked = excluded.liked`
	statement, err := db.Prepare(likeSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(postID, userID, like)
	if err != nil {
		http.Error(w, "Error liking post in database", http.StatusInternalServerError)
		return
	}

	updatePostLikesSQL := `UPDATE Post SET likes = likes + ?, dislikes = dislikes + ? WHERE id = ?`
	statement, err = db.Prepare(updatePostLikesSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	var likeIncrement, dislikeIncrement int
	if like {
		likeIncrement = 1
		dislikeIncrement = 0
	} else {
		likeIncrement = 0
		dislikeIncrement = 1
	}

	_, err = statement.Exec(likeIncrement, dislikeIncrement, postID)
	if err != nil {
		http.Error(w, "Error updating post likes in database", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Post liked successfully!")
}

func createPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	postName := r.FormValue("post_name")
	postMessage := r.FormValue("post_message")
	categoryName := r.FormValue("category_name")

	username, err := getSessionUser(r)
	if err != nil {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var userID int
	err = db.QueryRow("SELECT id FROM Account WHERE username = ?", username).Scan(&userID)
	if err != nil {
		http.Error(w, "Error getting user ID", http.StatusInternalServerError)
		return
	}

	createPostSQL := `INSERT INTO Post (post_name, creator_id, post_message, category_name, likes, dislikes) VALUES (?, ?, ?, ?, 0, 0)`
	statement, err := db.Prepare(createPostSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(postName, userID, postMessage, categoryName)
	if err != nil {
		http.Error(w, "Error creating post in database", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Post created successfully!")
}

// Ajout d'une route pour récupérer les catégories
func handleCategories(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "BDD/DBForum.db")
	if err != nil {
		http.Error(w, "Error opening database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	categories, err := getAllCategories(db)
	if err != nil {
		http.Error(w, "Failed to get categories", http.StatusInternalServerError)
		return
	}

	data := struct {
		Categories []string
	}{
		Categories: categories,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func getAllCategories(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM Category")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		categories = append(categories, name)
	}
	return categories, nil
}

func deletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	postID := r.FormValue("post_id")

	username, err := getSessionUser(r)
	if err != nil {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var userID int
	err = db.QueryRow("SELECT id FROM Account WHERE username = ?", username).Scan(&userID)
	if err != nil {
		http.Error(w, "Error getting user ID", http.StatusInternalServerError)
		return
	}

	// Check if the user is the creator of the post
	var creatorID int
	err = db.QueryRow("SELECT creator_id FROM Post WHERE id = ?", postID).Scan(&creatorID)
	if err != nil {
		http.Error(w, "Error getting post creator ID", http.StatusInternalServerError)
		return
	}

	if userID != creatorID {
		http.Error(w, "User not authorized to delete this post", http.StatusForbidden)
		return
	}

	deletePostSQL := `DELETE FROM Post WHERE id = ?`
	statement, err := db.Prepare(deletePostSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(postID)
	if err != nil {
		http.Error(w, "Error deleting post from database", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Post deleted successfully!")
}

func getPostByID(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("post_id")
	if postID == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite3", "BDD/DBForum.db")
	if err != nil {
		http.Error(w, "Error opening database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var post Post
	query := "SELECT id, post_name, creator_id, post_message, category_name, likes, dislikes FROM Post WHERE id = ?"
	err = db.QueryRow(query, postID).Scan(&post.ID, &post.PostName, &post.CreatorID, &post.PostMessage, &post.CategoryName, &post.Likes, &post.Dislikes)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Post not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error querying database", http.StatusInternalServerError)
		}
		return
	}

	jsonData, err := json.Marshal(post)
	if err != nil {
		http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
