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
	// http.HandleFunc("/post/", handlers.PostHandler)
	http.HandleFunc("/user", handlers.UserHandler)
	// http.HandleFunc("/comment/", handlers.CommentHandler)

	// Start the server
	port := ":8082"
	fmt.Printf("Server is running on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
