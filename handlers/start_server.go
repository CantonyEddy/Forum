package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

var tmpl *template.Template
var tmpl_register *template.Template
var tmpl_login *template.Template
var db *sql.DB

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

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.Dir(wd + "\\web"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, wd+"\\Templates\\index.html")
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

	http.HandleFunc("/registerUser", handleRegister)

	fmt.Println("Pour accéder à la page web -> http://localhost:8080/")
	err1 := http.ListenAndServe(":8080", nil)
	if err1 != nil {
		log.Fatal("ListenAndServe: ", err)
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
