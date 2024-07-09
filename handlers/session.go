package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"
)

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

	http.Redirect(w, r, "/Forum?username="+username, http.StatusSeeOther)
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

func handleHome(w http.ResponseWriter, r *http.Request, category string) map[string]interface{} {
	username := getSessionUsername(r)
	fmt.Println(username)
	rank, err := getSessionRank(r)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error getting rank", http.StatusInternalServerError)
		return nil
	}

	// Récupérer le paramètre de tri de l'URL
	sort := r.URL.Query().Get("sort")
	orderBy := "p.id DESC" // Tri par défaut

	if sort == "likes_asc" {
		orderBy = "likeCount ASC"
	} else if sort == "likes_desc" {
		orderBy = "likeCount DESC"
	}

	// Construire la requête SQL avec le tri spécifié
	query := `SELECT 
            p.id, 
            p.post_name, 
            p.creator_id, 
            p.post_message, 
            p.category_name, 
            IFNULL(likeCount, 0) as likeCount, 
            IFNULL(dislikeCount, 0) as dislikeCount,
            IFNULL(i.link, '') as imageLink
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
        LEFT JOIN (
            SELECT 
                post_id, 
                link 
            FROM 
                Image
        ) i
        ON p.id = i.post_id`

	var rows *sql.Rows

	if category != "" {
		query += " WHERE p.category_name = ? ORDER BY " + orderBy
		rows, err = db.Query(query, category)
	} else {
		query += " ORDER BY " + orderBy
		rows, err = db.Query(query)
	}

	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		return nil
	}
	defer rows.Close()

	isAdmin := false
	if rank == "admin" {
		isAdmin = true
	}

	isModo := false
	if rank == "Modérateurs" || rank == "admin" {
		isModo = true
	}

	var posts []map[string]interface{}
	for rows.Next() {
		var id, likes, dislikes int
		var postName, creatorID, postMessage, categoryName, imageLink string
		err := rows.Scan(&id, &postName, &creatorID, &postMessage, &categoryName, &likes, &dislikes, &imageLink)
		if err != nil {
			http.Error(w, "Error scanning post", http.StatusInternalServerError)
			return nil
		}

		// Décoder les liens d'image
		decodedImageLink, err := decodeURL(imageLink)
		if err != nil {
			http.Error(w, "Error decoding image link", http.StatusInternalServerError)
			return nil
		}

		post := map[string]interface{}{
			"ID":           id,
			"PostName":     postName,
			"CreatorID":    creatorID,
			"PostMessage":  postMessage,
			"CategoryName": categoryName,
			"LikeCount":    likes,
			"DislikeCount": dislikes,
			"ImageLink":    decodedImageLink,
			"Rank":         isModo,
		}
		posts = append(posts, post)
	}

	data := map[string]interface{}{
		"Username": username,
		"Posts":    posts,
		"Rank":     isAdmin,
		"Category": category,
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
