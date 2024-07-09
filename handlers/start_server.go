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
)

var tmpl *template.Template
var tmpl_register *template.Template
var tmpl_login *template.Template
var tmpl_main_page *template.Template
var tmpl_admin_pannel *template.Template
var tmpl_create_poste *template.Template
var tmpl_profile *template.Template
var tmpl_post *template.Template
var db *sql.DB

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

	tmpl_post, err = template.New("post").ParseFiles(filepath.Join(wd, "Static", "Templates", "post.html"))
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
			http.Redirect(w, r, "/login", http.StatusSeeOther)
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
			category := r.URL.Query().Get("category")
			data := handleHome(w, r, category)
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
			data := handleHome(w, r, "")
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

	http.HandleFunc("/post/", handlePost)
	http.HandleFunc("/addComment", handleAddComment)
	http.HandleFunc("/registerUser", handleRegister)
	http.HandleFunc("/loginUser", handleLogin)
	http.HandleFunc("/upgradeRank", handleRankUp)
	http.HandleFunc("/addPost", createPost)
	http.HandleFunc("/likePost", handleLikePost)
	http.HandleFunc("/deletePost", deletePostByID)

	http.HandleFunc("/auth/google/login", HandleGoogleLogin)
	http.HandleFunc("/auth/google/callback", HandleGoogleCallback)
	http.HandleFunc("/auth/github/login", HandleGitHubLogin)
	http.HandleFunc("/auth/github/callback", HandleGitHubCallback)

	fmt.Println("Pour accéder à la page web -> http://localhost:8080/")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

func createTables(db *sql.DB) {
	/*dropPostTableSQL := `DROP TABLE IF EXISTS Account;`
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
        password TEXT,
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

	createCommentTableSQL := `CREATE TABLE IF NOT EXISTS Commentaire (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		message TEXT NOT NULL,
		FOREIGN KEY (post_id) REFERENCES Post(id),
		FOREIGN KEY (user_id) REFERENCES Account(id)
	);`
	_, err = db.Exec(createCommentTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	/*
		insertAccountSQL := `INSERT INTO Account (username, password, mail, rank) VALUES (?, ?, ?, ?)`
		_, err = db.Exec(insertAccountSQL, "admin1", "admin1", "john@example.com", "admin")
		if err != nil {
			log.Fatal(err)
		}*/
}
