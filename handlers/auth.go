package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// var (
// 	db        *sql.DB
// 	templates *template.Template
// )

func init() {
	// Initialize database connection
	var err error
	db, err = sql.Open("sqlite", "./forum.db")
	if err != nil {
		log.Fatal("Database connection error:", err)
	}

	// Test database connection
	if err = db.Ping(); err != nil {
		log.Fatal("Database ping error:", err)
	}

	// Only create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Table creation error:", err)
	}

	// Create sessions table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Fatal("Sessions table creation error:", err)
	}

	// Parse templates
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Template parsing error:", err)
	}
}

type AuthData struct {
	Error string
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		err := templates.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Get form values
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate input
	if email == "" || password == "" {
		data := AuthData{Error: "Email and password are required"}
		templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	// Get user from database
	var user struct {
		ID       string
		Email    string
		Password string
	}

	err := db.QueryRow(`
		SELECT id, email, password 
		FROM users 
		WHERE email = ?`, email).Scan(&user.ID, &user.Email, &user.Password)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Login attempt with non-existent email: %s", email)
			data := AuthData{Error: "Invalid email or password"}
			templates.ExecuteTemplate(w, "login.html", data)
			return
		}
		log.Printf("Database error: %v", err)
		data := AuthData{Error: "Internal server error"}
		templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("Failed login attempt for email: %s", email)
		data := AuthData{Error: "Invalid email or password"}
		templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	// Create new session
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	// Delete any existing sessions for this user
	_, err = db.Exec("DELETE FROM sessions WHERE user_id = ?", user.ID)
	if err != nil {
		log.Printf("Error clearing old sessions: %v", err)
	}

	// Create new session in database
	_, err = db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at) 
		VALUES (?, ?, ?)`,
		sessionID, user.ID, expiresAt)

	if err != nil {
		log.Printf("Session creation error: %v", err)
		templates.ExecuteTemplate(w, "login.html", AuthData{Error: "Error creating session"})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})

	// Log successful login
	log.Printf("Successful login for user: %s", email)

	// Redirect to user page
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates.ExecuteTemplate(w, "register.html", nil)
		return
	}

	// Add debug logging
	log.Println("Processing registration request")

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Debug input values (remove in production)
	log.Printf("Received registration: email=%s, username=%s", email, username)

	// Validate input
	if email == "" || username == "" || password == "" {
		log.Println("Empty fields detected")
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "All fields are required"})
		return
	}

	// Check if email or username exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR username = ?)",
		email, username).Scan(&exists)
	if err != nil {
		log.Printf("Error checking existing user: %v", err)
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Registration failed"})
		return
	}
	if exists {
		log.Println("User already exists")
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Email or username already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Registration failed"})
		return
	}

	// Create user
	userID := uuid.New().String()
	_, err = db.Exec(`
		INSERT INTO users (id, email, username, password, created_at) 
		VALUES (?, ?, ?, ?, ?)`,
		userID, email, username, string(hashedPassword), time.Now())

	if err != nil {
		log.Printf("User creation error: %v", err)
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Failed to create account"})
		return
	}

	log.Printf("Successfully created user with ID: %s", userID)

	// Create session for new user
	sessionID := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	// Store session
	_, err = db.Exec("INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)",
		sessionID, userID, time.Now().Add(24*time.Hour))
	if err != nil {
		log.Printf("Session creation error: %v", err)
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
	})

	// Delete session from database
	if cookie, err := r.Cookie("session_id"); err == nil {
		db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
