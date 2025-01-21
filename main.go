package main

import (
	"log"
	"net/http"
	"forum/handlers"
)

func main() {
	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Register routes
	http.HandleFunc("/", handlers.HomeHandler)
	// http.HandleFunc("/login", handlers.LoginHandler)
	// http.HandleFunc("/register", handlers.registerHandler)

	// Start the server
	log.Println("Server starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
