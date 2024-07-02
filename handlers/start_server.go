package handlers

import (
	"database/sql"
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
var tmpl_create_poste *template.Template
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

	createTables(db) // Ensure tables are created

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

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
	http.HandleFunc("/addPost", createPost)

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
		FOREIGN KEY (creator_id) REFERENCES Account(id)
	);`
	_, err = db.Exec(createPostTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	createImageTableSQL := `CREATE TABLE IF NOT EXISTS Post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		link TEXT NOT NULL
	);`
	_, err = db.Exec(createImageTableSQL)
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

	// Récupérer les postes de la base de données
	rows, err := db.Query(`SELECT id, post_name, creator_id, post_message, category_name FROM Post ORDER BY id DESC`)
	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var id int
		var postName, creatorID, postMessage, category_name string
		err := rows.Scan(&id, &postName, &creatorID, &postMessage, &category_name)
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
		}
		posts = append(posts, post)
	}

	data := map[string]interface{}{
		"Username": username,
		"Posts":    posts,
	}

	return data
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

	postName := r.FormValue("postName")
	creatorID := getSessionUsername(r)
	postMessage := r.FormValue("postMessage")
	category_name := r.FormValue("category_name")

	// Insert the user into the database
	insertPostSQL := `INSERT INTO Post (post_name, creator_id, post_message, category_name) VALUES (?, ?, ?, ?)`
	statement, err := db.Prepare(insertPostSQL)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(postName, creatorID, postMessage, category_name)
	if err != nil {
		http.Error(w, "Error inserting user into database", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Post %s registered successfully!", postName)
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
