package handlers

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // Import the sqlite3 driver
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var tmpl *template.Template
var tmpl_register *template.Template
var tmpl_login *template.Template
var tmpl_main_page *template.Template
var tmpl_admin_pannel *template.Template
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

	tmpl, err = template.New("index").ParseFiles("Templates/index.html")
	if err != nil {
		panic(err)
	}

	tmpl_register, err = template.New("register").ParseFiles("Templates/register.html")
	if err != nil {
		panic(err)
	}

	tmpl_login, err = template.New("login").ParseFiles("Templates/login.html")
	if err != nil {
		panic(err)
	}

	tmpl_main_page, err = template.New("Forum").ParseFiles("Templates/forumMainPage.html")
	if err != nil {
		panic(err)
	}

	tmpl_admin_pannel, err = template.New("adminPannel").ParseFiles("Templates/adminPannel.html")
	if err != nil {
		panic(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.Dir(wd + "\\web"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if !isAuthenticated(r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			} else {
				http.ServeFile(w, r, wd+"\\Templates\\index.html")
			}
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/register" {
			http.ServeFile(w, r, wd+"\\Templates\\register.html")
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.ServeFile(w, r, wd+"\\Templates\\login.html")
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

	http.HandleFunc("/registerUser", handleRegister)
	http.HandleFunc("/loginUser", handleLogin)
	http.HandleFunc("/upgradeRank", handleRankUp)

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
