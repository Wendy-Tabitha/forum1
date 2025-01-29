package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Comment struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	PostID    int       `json:"postId"`
	CreatedAt time.Time `json:"createdAt"`
}

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON body
	var post struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		log.Printf("Session error: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Create posts table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			user_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Printf("Table creation error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	result, err := db.Exec(`
		INSERT INTO posts (title, content, user_id, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		post.Title, post.Content, user.ID)

	if err != nil {
		log.Printf("Error creating post: %v", err)
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	postID, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"postId":  postID,
	})
}

func VotePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from session
	user, err := getUserFromSession(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse post ID and vote value
	var vote struct {
		Value int `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&vote); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	postID := getPostIDFromURL(r.URL.Path)
	if postID == 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Update or insert vote
	_, err = db.Exec(`
		INSERT INTO post_votes (user_id, post_id, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, post_id) DO UPDATE SET value = $3
	`, user.ID, postID, vote.Value)

	if err != nil {
		log.Printf("Error voting on post: %v", err)
		http.Error(w, "Failed to vote", http.StatusInternalServerError)
		return
	}

	// Get updated score
	var newScore int
	err = db.QueryRow("SELECT SUM(value) FROM post_votes WHERE post_id = ?", postID).Scan(&newScore)
	if err != nil {
		log.Printf("Error getting updated score: %v", err)
		http.Error(w, "Failed to get updated score", http.StatusInternalServerError)
		return
	}

	// Return updated score
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"newScore": newScore,
	})
}

func getUserFromSession(r *http.Request) (*User, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}

	var user User
	err = db.QueryRow(`
		SELECT u.id, u.username, u.email 
		FROM users u 
		JOIN sessions s ON u.id = s.user_id 
		WHERE s.id = ? AND s.expires_at > CURRENT_TIMESTAMP`,
		cookie.Value).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func AddCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from session
	user, err := getUserFromSession(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse comment content
	var comment struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	postID := getPostIDFromURL(r.URL.Path)
	if postID == 0 {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Insert comment
	var commentID int64
	err = db.QueryRow(`
		INSERT INTO comments (content, user_id, post_id, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		RETURNING id
	`, comment.Content, user.ID, postID).Scan(&commentID)

	if err != nil {
		log.Printf("Error creating comment: %v", err)
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"comment": map[string]interface{}{
			"id":       commentID,
			"content":  comment.Content,
			"username": user.Username,
		},
	})
}

func getPostIDFromURL(path string) int {
	// Extract post ID from URL path
	// Expected format: /api/posts/{id}/...
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return 0
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0
	}
	return id
}

func GetPostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query(`
		SELECT p.id, p.title, p.content, p.user_id, u.username, p.created_at,
			   COALESCE(SUM(v.value), 0) as score,
			   COUNT(DISTINCT c.id) as comment_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN post_votes v ON p.id = v.post_id
		LEFT JOIN comments c ON p.id = c.post_id
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &post.UserID,
			&post.Username, &post.CreatedAt, &post.Score, &post.CommentCount,
		)
		if err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}
		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}
