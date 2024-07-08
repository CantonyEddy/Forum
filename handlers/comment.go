package handlers

import "net/http"

func handleAddComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode de requête non valide", http.StatusMethodNotAllowed)
		return
	}

	userID := getSessionUserID(r)
	if userID == 0 {
		http.Error(w, "Utilisateur non connecté", http.StatusUnauthorized)
		return
	}

	postID := r.FormValue("postID")
	message := r.FormValue("message")

	if postID == "" || message == "" {
		http.Error(w, "Paramètres manquants", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`INSERT INTO Commentaire (post_id, user_id, message) VALUES (?, ?, ?)`, postID, userID, message)
	if err != nil {
		http.Error(w, "Erreur lors de l'ajout du commentaire", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}
