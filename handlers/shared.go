package handlers

import (
	"database/sql"
	"html/template"
	"time"
)

// Shared variables for the handlers package
var (
	db        *sql.DB
	templates *template.Template
)

// Post represents a forum post
type Post struct {
	ID        string
	UserID    string
	Title     string
	Content   string
	Category  string
	CreatedAt time.Time
	Likes     int
	Dislikes  int
}

// User represents a forum user
type User struct {
	ID       string
	Email    string
	Username string
	Password string
}

// Category represents a forum category
type Category struct {
	ID          int
	Name        string
	Description string
	PostCount   int
	Color       string
}
