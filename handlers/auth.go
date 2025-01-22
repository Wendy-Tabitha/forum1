package handlers

import (
	"net/http"
	"time"

	"forum/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthData struct {
	Error string
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates.ExecuteTemplate(w, "login.html", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := models.GetUserByEmail(email)
	if err != nil {
		templates.ExecuteTemplate(w, "login.html", AuthData{Error: "Login failed"})
		return
	}

	if user == nil {
		templates.ExecuteTemplate(w, "login.html", AuthData{Error: "Invalid email or password"})
		return
	}

	if !models.CheckPasswordHash(password, user.Password) {
		templates.ExecuteTemplate(w, "login.html", AuthData{Error: "Invalid email or password"})
		return
	}

	// Create session
	sessionID := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates.ExecuteTemplate(w, "register.html", nil)
		return
	}

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate input
	if email == "" || username == "" || password == "" {
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "All fields are required"})
		return
	}

	// Check if email exists
	existingUser, err := models.GetUserByEmail(email)
	if err != nil {
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Registration failed"})
		return
	}
	if existingUser != nil {
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Email already exists"})
		return
	}

	// Check if username exists
	existingUser, err = models.GetUserByUsername(username)
	if err != nil {
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Registration failed"})
		return
	}
	if existingUser != nil {
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Username already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Registration failed"})
		return
	}

	// Create user
	user := models.User{
		ID:        uuid.New().String(),
		Email:     email,
		Username:  username,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	// Save user to database
	err = user.Save()
	if err != nil {
		templates.ExecuteTemplate(w, "register.html", AuthData{Error: "Failed to save user"})
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
