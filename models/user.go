package models

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func InitDB(dataSourceName string) error {
	var err error
	db, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return err
	}

	// Create users table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE,
			username TEXT UNIQUE,
			password TEXT,
			created_at DATETIME
		)
	`)
	return err
}

type User struct {
	ID        string
	Email     string
	Username  string
	Password  string
	CreatedAt time.Time
}

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

func (u *User) Save() error {
	_, err := db.Exec(
		"INSERT INTO users (id, email, username, password, created_at) VALUES (?, ?, ?, ?, ?)",
		u.ID, u.Email, u.Username, u.Password, u.CreatedAt,
	)
	return err
}

func GetUserByEmail(email string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, email, username, password, created_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.Password, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, email, username, password, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Email, &user.Username, &user.Password, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
