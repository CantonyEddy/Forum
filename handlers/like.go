package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

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
