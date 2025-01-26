package main

import (
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
	// http.HandleFunc("/post/", handlers.PostHandler)
	http.HandleFunc("/user", handlers.UserHandler)
	// http.HandleFunc("/comment/", handlers.CommentHandler)

	// Start the server
	log.Println("Server starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
