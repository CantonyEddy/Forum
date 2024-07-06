package BDD

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func InitTable() {
	// Ouvrir une connexion à la base de données SQLite
	db, err := sql.Open("sqlite3", "BDD/DBForum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Créer les tables
	createTables(db)

	// Insérer des données exemple (optionnel)
	// insertExampleData(db)

	// Supprimer un compte par ID (optionnel)
	// deleteAccountByID(db, 2)

	// Lire les données
	readData(db)
}

func createTables(db *sql.DB) {
	// Créer la table Account
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

	// Créer la table ForumMainPage
	createForumMainPageTableSQL := `CREATE TABLE IF NOT EXISTS ForumMainPage (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        post_name TEXT NOT NULL,
        creator_name TEXT NOT NULL,
        post_picture TEXT,
        category_name TEXT
    );`
	_, err = db.Exec(createForumMainPageTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	// Créer la table Post
	createPostTableSQL := `CREATE TABLE IF NOT EXISTS Post (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        post_name TEXT NOT NULL,
        creator_id INTEGER NOT NULL,
        category_name TEXT,
        post_message TEXT,
        comments_messages TEXT,
        FOREIGN KEY (creator_id) REFERENCES Account(id)
    );`
	_, err = db.Exec(createPostTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	// Créer la table Image
	createImageTableSQL := `CREATE TABLE IF NOT EXISTS Image (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		link TEXT NOT NULL
	);`
	_, err = db.Exec(createImageTableSQL)
	if err != nil {
		log.Fatal(err)
	}
}

func insertExampleData(db *sql.DB) {
	// Insérer des données dans Account
	insertAccountSQL := `INSERT INTO Account (username, password, mail, rank) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(insertAccountSQL, "john_doe", "securepassword", "john@example.com", "admin")
	if err != nil {
		log.Fatal(err)
	}

	// Insérer des données dans ForumMainPage
	insertForumMainPageSQL := `INSERT INTO ForumMainPage (post_name, creator_name, post_picture, category_name) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(insertForumMainPageSQL, "First Post", "john_doe", "link_to_picture", "General")
	if err != nil {
		log.Fatal(err)
	}

	// Insérer des données dans Post
	insertPostSQL := `INSERT INTO Post (post_name, creator_id, category_name, post_message, comments_messages) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(insertPostSQL, "First Post", 1, "General", "This is the first post message", "")
	if err != nil {
		log.Fatal(err)
	}
}

func deleteAccountByID(db *sql.DB, id int) {
	deleteSQL := `DELETE FROM Account WHERE id = ?`
	statement, err := db.Prepare(deleteSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer statement.Close()

	result, err := statement.Exec(id)
	if err != nil {
		log.Fatal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Deleted %d row(s) from Account table\n", rowsAffected)
}

func readData(db *sql.DB) {
	// Lire les données de Account
	rows, err := db.Query("SELECT id, username, mail, rank FROM Account")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var username, mail, rank string
		err = rows.Scan(&id, &username, &mail, &rank)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Account: ID=%d, Username=%s, Mail=%s, Rank=%s\n", id, username, mail, rank)
	}

	// Lire les données de ForumMainPage
	rows, err = db.Query("SELECT id, post_name, creator_name, post_picture, category_name FROM ForumMainPage")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var postName, creatorName, postPicture, categoryName string
		err = rows.Scan(&id, &postName, &creatorName, &postPicture, &categoryName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ForumMainPage: ID=%d, PostName=%s, CreatorName=%s, PostPicture=%s, CategoryName=%s\n", id, postName, creatorName, postPicture, categoryName)
	}

	// Lire les données de Post
	rows, err = db.Query("SELECT id, post_name, creator_id, category_name, post_message, comments_messages FROM Post")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var postName, categoryName, postMessage, commentsMessages string
		var creatorId int
		err = rows.Scan(&id, &postName, &creatorId, &categoryName, &postMessage, &commentsMessages)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Post: ID=%d, PostName=%s, CreatorID=%d, CategoryName=%s, PostMessage=%s, CommentsMessages=%s\n", id, postName, creatorId, categoryName, postMessage, commentsMessages)
	}
}
