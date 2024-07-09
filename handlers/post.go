package handlers

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

	http.Redirect(w, r, "/Forum?username="+username, http.StatusSeeOther)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	postIDStr := strings.TrimPrefix(r.URL.Path, "/post/")
	if postIDStr == "" {
		http.Error(w, "Post ID manquant", http.StatusBadRequest)
		return
	}

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var post struct {
		ID              int
		PostName        string
		CreatorUsername string
		PostMessage     string
		CategoryName    string
		Likes           int
		Dislikes        int
		ImageLinks      []string // Add this field to store image links
	}

	query := `SELECT p.id, p.post_name, a.username AS creator_username, p.post_message, p.category_name, 
              (SELECT COUNT(*) FROM PostLikes WHERE post_id = p.id AND liked = 1) AS likes,
              (SELECT COUNT(*) FROM PostLikes WHERE post_id = p.id AND liked = 0) AS dislikes
              FROM Post p
              JOIN Account a ON p.creator_id = a.id
              WHERE p.id = ?`

	err = db.QueryRow(query, postID).Scan(&post.ID, &post.PostName, &post.CreatorUsername, &post.PostMessage, &post.CategoryName, &post.Likes, &post.Dislikes)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Post non trouvé", http.StatusNotFound)
		} else {
			http.Error(w, "Erreur lors de la récupération du post", http.StatusInternalServerError)
		}
		return
	}

	// Récupérer les commentaires associés au post
	rows, err := db.Query(`SELECT c.id, c.message, a.username FROM Commentaire c JOIN Account a ON c.user_id = a.id WHERE c.post_id = ?`, postID)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des commentaires", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Récupérer les images associées au post
	imageRows, err := db.Query(`SELECT link FROM Image WHERE post_id = ?`, postID)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des images", http.StatusInternalServerError)
		return
	}
	defer imageRows.Close()

	var imageLinks []string
	for imageRows.Next() {
		var link string
		err := imageRows.Scan(&link)
		if err != nil {
			http.Error(w, "Erreur lors de l'analyse des images", http.StatusInternalServerError)
			return
		}
		imageLinks = append(imageLinks, link)
	}

	post.ImageLinks = imageLinks

	err = tmpl_post.Execute(w, post)
	if err != nil {
		http.Error(w, "Erreur lors du rendu du template", http.StatusInternalServerError)
	}
}

func deletePostByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	postID := r.FormValue("id")
	if postID == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	username := getSessionUsername(r)
	userId := getUserIDByUsername(username)
	rank, err := getSessionRank(r)
	if err != nil {
		http.Error(w, "Error getting session rank", http.StatusInternalServerError)
		return
	}

	var creatorID string
	err = db.QueryRow("SELECT creator_id FROM Post WHERE id = ?", postID).Scan(&creatorID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Post not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error checking post creator", http.StatusInternalServerError)
		}
		return
	}

	id, err := strconv.Atoi(creatorID)
	if err != nil {
		panic(err)
	}

	if userId != id && rank != "admin" && rank != "Modérateurs" {
		http.Error(w, "Unauthorized to delete this post", http.StatusUnauthorized)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Error beginning transaction", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM PostLikes WHERE post_id = ?", postID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Error deleting post likes", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM Commentaire WHERE post_id = ?", postID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Error deleting comments", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM Image WHERE post_id = ?", postID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Error deleting Image", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM Post WHERE id = ?", postID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Error committing transaction", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/Forum?username="+username, http.StatusSeeOther)
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
	filePath := filepath.Join("Static/uploads/images", handler.Filename)
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
}
