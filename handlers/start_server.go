package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // Import the sqlite3 driver
	"html/template"
	"io"
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
var tmpl_profile *template.Template
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

	tmpl_profile, err = template.New("profile").ParseFiles(filepath.Join(wd, "Static", "Templates", "profile.html"))
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

	http.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/profile" {
			data := handleProfile(w, r)
			tmpl_profile.Execute(w, data)
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
	/*dropPostTableSQL := `DROP TABLE IF EXISTS Post;`
	_, err := db.Exec(dropPostTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	dropLikeTableSQL := `DROP TABLE IF EXISTS PostLikes;`
	_, err = db.Exec(dropLikeTableSQL)
	if err != nil {
		log.Fatal(err)
	}*/

	createAccountTableSQL := `CREATE TABLE IF NOT EXISTS Account (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL,
        password TEXT NOT NULL,
        mail TEXT NOT NULL,
        rank TEXT NOT NULL
    );`
	_, err := db.Exec(createAccountTableSQL)
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

	var dbUsername, dbPassword string
	query := `SELECT username, password FROM Account WHERE username = ?`
	err := db.QueryRow(query, username).Scan(&dbUsername, &dbPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No user found with username:", username)
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		} else {
			log.Println("Error querying database:", err)
			http.Error(w, "Error querying database", http.StatusInternalServerError)
		}
		return
	}

	if dbPassword != password {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	sessionToken := fmt.Sprintf("%d", time.Now().UnixNano())
	sessionsMutex.Lock()
	sessions[sessionToken] = username
	sessionsMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})
	http.Redirect(w, r, "/Forum?username="+username, http.StatusSeeOther)
}

func handleHome(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	username := getSessionUsername(r)
	rank, err := getSessionRank(r)
	if err != nil {
		http.Error(w, "Error getting rank", http.StatusInternalServerError)
		return nil
	}

	// Récupérer les postes de la base de données
	rows, err := db.Query(`SELECT 
            p.id, 
            p.post_name, 
            p.creator_id, 
            p.post_message, 
            p.category_name, 
            IFNULL(likeCount, 0) as likeCount, 
            IFNULL(dislikeCount, 0) as dislikeCount
        FROM 
            Post p 
        LEFT JOIN (
            SELECT 
                post_id, 
                SUM(CASE WHEN liked = 1 THEN 1 ELSE 0 END) as likeCount, 
                SUM(CASE WHEN liked = 0 THEN 1 ELSE 0 END) as dislikeCount 
            FROM 
                PostLikes 
            GROUP BY 
                post_id
        ) pl 
        ON p.id = pl.post_id 
        ORDER BY p.id DESC`)
	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var id, likes, dislikes int
		var postName, creatorID, postMessage, category_name string
		err := rows.Scan(&id, &postName, &creatorID, &postMessage, &category_name, &likes, &dislikes)
		if err != nil {
			http.Error(w, "Error scanning post", http.StatusInternalServerError)
			return nil
		}

		post := map[string]interface{}{
			"ID":           id,
			"PostName":     postName,
			"CreatorID":    creatorID,
			"PostMessage":  postMessage,
			"categoryName": category_name,
			"LikeCount":    likes,
			"DislikeCount": dislikes,
		}
		posts = append(posts, post)
	}

	isAdmin := false
	if rank == "admin" {
		isAdmin = true
	}

	data := map[string]interface{}{
		"Username": username,
		"Posts":    posts,
		"Rank":     isAdmin,
	}

	return data
}

func handleProfile(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	username := getSessionUsername(r)
	userID := getUserIDByUsername(username)

	// Requête SQL pour récupérer les posts aimés par l'utilisateur
	query := `
        SELECT 
            p.id, 
            p.post_name, 
            p.creator_id, 
            p.post_message, 
            p.category_name, 
            p.likes, 
            p.dislikes
        FROM 
            Post p
        LEFT JOIN 
            PostLikes pl ON p.id = pl.post_id
        WHERE 
            pl.user_id = ? AND pl.liked = 1
        ORDER BY 
            p.id DESC`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	var postsLike []map[string]interface{}
	for rows.Next() {
		var id, likes, dislikes int
		var postName, postMessage, categoryName string
		var creatorID int // creatorID should be an int as it references Account(id)
		err := rows.Scan(&id, &postName, &creatorID, &postMessage, &categoryName, &likes, &dislikes)
		if err != nil {
			log.Printf("Error scanning post: %v", err)
			http.Error(w, "Error scanning post", http.StatusInternalServerError)
			return nil
		}

		post := map[string]interface{}{
			"ID":           id,
			"PostName":     postName,
			"CreatorID":    creatorID,
			"PostMessage":  postMessage,
			"CategoryName": categoryName,
			"Likes":        likes,
			"Dislikes":     dislikes,
		}
		postsLike = append(postsLike, post)
	}

	// Récupérer les postes de la base de données
	rows, err = db.Query(`SELECT 
            p.id, 
            p.post_name, 
            p.creator_id, 
            p.post_message, 
            p.category_name, 
            IFNULL(likeCount, 0) as likeCount, 
            IFNULL(dislikeCount, 0) as dislikeCount
        FROM 
            Post p 
        LEFT JOIN (
            SELECT 
                post_id, 
                SUM(CASE WHEN liked = 1 THEN 1 ELSE 0 END) as likeCount, 
                SUM(CASE WHEN liked = 0 THEN 1 ELSE 0 END) as dislikeCount 
            FROM 
                PostLikes 
            GROUP BY 
                post_id
        ) pl 
        ON p.id = pl.post_id 
        ORDER BY p.id DESC`)
	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()
	isEqID := false

	var posts []map[string]interface{}
	for rows.Next() {
		var id, likes, dislikes, creatorID int
		var postName, postMessage, category_name string
		err := rows.Scan(&id, &postName, &creatorID, &postMessage, &category_name, &likes, &dislikes)
		if err != nil {
			http.Error(w, "Error scanning post", http.StatusInternalServerError)
			return nil
		}

		if userID == creatorID {
			isEqID = true
		} else {
			isEqID = false
		}
		post := map[string]interface{}{
			"ID":           id,
			"PostName":     postName,
			"CreatorID":    creatorID,
			"PostMessage":  postMessage,
			"categoryName": category_name,
			"LikeCount":    likes,
			"DislikeCount": dislikes,
			"IsEqID":       isEqID,
		}
		posts = append(posts, post)
	}

	data := map[string]interface{}{
		"Username":  username,
		"Posts":     posts,
		"PostsLike": postsLike,
		"UserId":    userID,
	}

	return data
}

func handleLikePost(w http.ResponseWriter, r *http.Request) {
	userID := getSessionUserID(r)
	if userID == 0 {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}

	postID := r.URL.Query().Get("postID")
	like := r.URL.Query().Get("like") // "true" for like, "false" for dislike

	if postID == "" || like == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	liked := like == "true"

	// Check if the user has already liked/disliked the post
	var existingLikeID int
	var existingLiked bool
	err := db.QueryRow(`SELECT id, liked FROM PostLikes WHERE post_id = ? AND user_id = ?`, postID, userID).Scan(&existingLikeID, &existingLiked)

	if err == sql.ErrNoRows {
		// No existing like/dislike, insert a new one
		_, err = db.Exec(`INSERT INTO PostLikes (post_id, user_id, liked) VALUES (?, ?, ?)`, postID, userID, liked)
		if err != nil {
			http.Error(w, "Error liking post", http.StatusInternalServerError)
			return
		}
	} else if err == nil {
		if existingLiked == liked {
			// Existing like/dislike found and matches the current action, remove it (toggle off)
			_, err = db.Exec(`DELETE FROM PostLikes WHERE id = ?`, existingLikeID)
			if err != nil {
				http.Error(w, "Error unliking post", http.StatusInternalServerError)
				return
			}
		} else {
			// Existing like/dislike found but does not match the current action, update it (toggle switch)
			_, err = db.Exec(`UPDATE PostLikes SET liked = ? WHERE id = ?`, liked, existingLikeID)
			if err != nil {
				http.Error(w, "Error updating like status", http.StatusInternalServerError)
				return
			}
		}
	} else {
		http.Error(w, "Error checking like status", http.StatusInternalServerError)
		return
	}

	// Update the likes and dislikes in the Post table
	updatePostLikes(postID)

	// Return the updated like/dislike count for the post
	var likeCount, dislikeCount int
	err = db.QueryRow(`SELECT 
                        (SELECT COUNT(*) FROM PostLikes WHERE post_id = ? AND liked = 1) as likeCount,
                        (SELECT COUNT(*) FROM PostLikes WHERE post_id = ? AND liked = 0) as dislikeCount`,
		postID, postID).Scan(&likeCount, &dislikeCount)
	if err != nil {
		http.Error(w, "Error fetching like/dislike counts", http.StatusInternalServerError)
		return
	}

	response := map[string]int{
		"likeCount":    likeCount,
		"dislikeCount": dislikeCount,
	}
	jsonResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func updatePostLikes(postID string) {
	_, err := db.Exec(`
        UPDATE Post
        SET likes = (SELECT COUNT(*) FROM PostLikes WHERE post_id = ? AND liked = 1),
            dislikes = (SELECT COUNT(*) FROM PostLikes WHERE post_id = ? AND liked = 0)
        WHERE id = ?
    `, postID, postID, postID)
	if err != nil {
		log.Printf("Error updating post likes: %v", err)
	}
}

func getSessionUserID(r *http.Request) int {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0
	}

	sessionToken := cookie.Value
	sessionsMutex.Lock()
	username := sessions[sessionToken]
	sessionsMutex.Unlock()

	if username == "" {
		return 0
	}

	var userID int
	err = db.QueryRow(`SELECT id FROM Account WHERE username = ?`, username).Scan(&userID)
	if err != nil {
		return 0
	}
	return userID
}

func handleAccount(w http.ResponseWriter) map[string]interface{} {
	rows, err := db.Query(`SELECT id, username, rank FROM Account ORDER BY id DESC`)
	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	var accounts []map[string]interface{}
	for rows.Next() {
		var id int
		var username, rank string
		err := rows.Scan(&id, &username, &rank)
		if err != nil {
			http.Error(w, "Error scanning post", http.StatusInternalServerError)
			return nil
		}

		account := map[string]interface{}{
			"ID":       id,
			"Username": username,
			"Rank":     rank,
		}
		accounts = append(accounts, account)
	}

	data := map[string]interface{}{
		"Accounts": accounts,
	}

	return data
}

func handleRankUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	updateRankSQL := `UPDATE Account SET rank = ? WHERE username = ?`
	stmt, err := db.Prepare(updateRankSQL)
	if err != nil {
		return
	}
	defer stmt.Close()

	// Exécute la commande avec les valeurs fournies
	_, err = stmt.Exec("Modérateurs", username)
	if err != nil {
		return
	}

	return
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return false
	}

	sessionToken := cookie.Value
	// Vérifiez le token de session ici
	// Par exemple, en le comparant avec une valeur stockée en mémoire ou en base de données

	return sessionToken != ""
}

func createPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	username := getSessionUsername(r)

	postName := r.FormValue("postName")
	creatorID := getUserIDByUsername(username)
	postMessage := r.FormValue("postMessage")
	category_name := r.FormValue("category_name")

	// Insert the post into the database
	insertPostSQL := `INSERT INTO Post (post_name, creator_id, post_message, category_name) VALUES (?, ?, ?, ?)`
	statement, err := db.Prepare(insertPostSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	result, err := statement.Exec(postName, creatorID, postMessage, category_name)
	if err != nil {
		http.Error(w, "Error inserting post into database", http.StatusInternalServerError)
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Error getting last insert ID", http.StatusInternalServerError)
		return
	}

	// Handle the image upload
	createImage(w, r, postID)

	fmt.Fprintf(w, "Post %s registered successfully!", postName)
}

func createImage(w http.ResponseWriter, r *http.Request, postID int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form
	err := r.ParseMultipartForm(20 << 20) // Limite de 20 MB
	if err != nil {
		http.Error(w, "Error parsing multipart form", http.StatusInternalServerError)
		log.Println("Error parsing multipart form:", err)
		return
	}

	// Retrieve the file from form data
	file, handler, err := r.FormFile("postImage")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		log.Println("Error retrieving the file:", err)
		return
	}
	defer file.Close()

	// Save the file to the disk
	filePath := filepath.Join("uploads/images", handler.Filename)
	dest, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		log.Println("Error creating the file:", err)
		return
	}
	defer dest.Close()
	_, err = io.Copy(dest, file)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		log.Println("Error copying the file:", err)
		return
	}

	// Insert the image link into the Image table
	insertImageSQL := `INSERT INTO Image (link, post_id) VALUES (?, ?)`
	statement, err := db.Prepare(insertImageSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		log.Println("Error preparing statement:", err)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(filePath, postID)
	if err != nil {
		http.Error(w, "Error inserting image link into database", http.StatusInternalServerError)
		log.Println("Error inserting image link into database:", err)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully: %s\n", handler.Filename)
}

func ensureUploadDir() error {
	uploadDir := filepath.Join("uploads", "images")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err := os.MkdirAll(uploadDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create upload directory: %w", err)
		}
	}
	return nil
}

func getSessionUsername(r *http.Request) string {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return ""
	}

	sessionToken := cookie.Value
	sessionsMutex.Lock()
	username := sessions[sessionToken]
	sessionsMutex.Unlock()

	return username
}

func getSessionRank(r *http.Request) (string, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return "", err
	}

	sessionToken := cookie.Value
	sessionsMutex.Lock()
	username, exists := sessions[sessionToken]
	sessionsMutex.Unlock()

	if !exists {
		return "", fmt.Errorf("session not found")
	}

	var rank string
	err = db.QueryRow("SELECT rank FROM Account WHERE username = ?", username).Scan(&rank)
	if err != nil {
		return "", err
	}

	return rank, nil
}

func getUserIDByUsername(username string) int {
	var userID int
	err := db.QueryRow("SELECT id FROM Account WHERE username = ?", username).Scan(&userID)
	if err != nil {
		log.Println("Error fetching user ID:", err)
		return 0 // Handle this case appropriately
	}
	return userID
}
