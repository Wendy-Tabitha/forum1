package main

import (
	"fmt"
	"log"
	"net/http"

	"forum/handlers"
	"forum/models"
)

func main() {
	// Initialize database
	err := models.InitDB("forum.db")
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Register routes
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/user", handlers.UserHandler)

	// Update the posts route
	http.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreatePostHandler(w, r)
		case http.MethodGet:
			handlers.GetPostsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Start the server
	port := ":8082"
	fmt.Printf("Server is running on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
