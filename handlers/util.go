package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func decodeURL(encodedURL string) (string, error) {
	decodedURL, err := url.QueryUnescape(encodedURL)
	if err != nil {
		return "", err
	}
	// Remplacer les barres obliques inverses par des barres obliques
	decodedURL = strings.ReplaceAll(decodedURL, "\\", "/")
	return decodedURL, nil
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
	loggedIn, username := isUserLoggedIn(r)

	if !loggedIn {
		return "", fmt.Errorf("session not found")
	}

	var rank string
	err := db.QueryRow("SELECT rank FROM Account WHERE username = ?", username).Scan(&rank)
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
