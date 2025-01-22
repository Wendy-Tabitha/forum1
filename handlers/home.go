package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type PageData struct {
	User       *User
	Posts      []Post
	Categories []Category
}

type User struct {
	ID       string
	Email    string
	Username string
	Password string
}

type Post struct {
	ID            int
	Title         string
	Content       string
	UserID        string
	Username      string
	Score         int
	CommentCount  int
	UserVoted     bool
	UserDownvoted bool
	Categories    []string
	CreatedAt     time.Time
}

type Category struct {
	ID          int
	Name        string
	Description string
	PostCount   int
	Color       string
}

var (
	templates *template.Template
	db        *sql.DB
)

func init() {
	var err error
	// Initialize database
	db, err = sql.Open("sqlite3", "./forum.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create tables if they don't exist
	CreateTables()

	// Parse templates
	templates = template.New("").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 02, 2006")
		},
	})
	templates = template.Must(templates.ParseGlob("templates/*.html"))

	// Insert default categories if they don't exist
	insertDefaultCategories()
}

func CreateTables() {
	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create categories table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			color TEXT
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create posts table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			user_id TEXT NOT NULL,
			score INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create post_categories table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS post_categories (
			post_id INTEGER,
			category_id INTEGER,
			PRIMARY KEY (post_id, category_id),
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create comments table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			user_id TEXT NOT NULL,
			post_id INTEGER NOT NULL,
			score INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (post_id) REFERENCES posts(id)
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create votes table for both posts and comments
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS votes (
			user_id TEXT NOT NULL,
			target_id INTEGER NOT NULL,
			target_type TEXT NOT NULL,
			value INTEGER NOT NULL,
			PRIMARY KEY (user_id, target_id, target_type),
			FOREIGN KEY (user_id) REFERENCES users(id),
			CHECK (value IN (-1, 1))
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func insertDefaultCategories() {
	categories := []struct {
		name        string
		description string
		color       string
	}{
		{"Technology", "Discussion about latest tech trends and innovations", "#3498db"},
		{"Science", "Scientific discoveries and research", "#2ecc71"},
		{"Entertainment", "Movies, TV shows, and pop culture", "#e74c3c"},
		{"Gaming", "Video games and gaming culture", "#9b59b6"},
		{"Art", "Visual arts, music, and creative works", "#e67e22"},
	}

	for _, cat := range categories {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO categories (name, description, color)
			VALUES (?, ?, ?)
		`, cat.name, cat.description, cat.color)
		if err != nil {
			log.Printf("Error inserting category %s: %v", cat.name, err)
		}
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Fetch categories with post counts
	categories, err := GetCategories()
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Fetch recent posts with their categories
	posts, err := GetRecentPosts()
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Categories: categories,
		Posts:      posts,
		User:       nil,
	}

	err = templates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func GetCategories() ([]Category, error) {
	rows, err := db.Query(`
		SELECT 
			c.id,
			c.name,
			c.description,
			c.color,
			COUNT(pc.post_id) as post_count
		FROM categories c
		LEFT JOIN post_categories pc ON c.id = pc.category_id
		GROUP BY c.id
		ORDER BY post_count DESC, c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Name, &cat.Description, &cat.Color, &cat.PostCount)
		if err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

func GetRecentPosts() ([]Post, error) {
	rows, err := db.Query(`
		SELECT 
			p.id,
			p.title,
			p.content,
			p.user_id,
			u.username,
			p.score,
			p.created_at,
			(
				SELECT GROUP_CONCAT(c.name)
				FROM post_categories pc
				JOIN categories c ON pc.category_id = c.id
				WHERE pc.post_id = p.id
			) as categories
		FROM posts p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.created_at DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		var categoriesStr sql.NullString
		err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.UserID,
			&p.Username, &p.Score, &p.CreatedAt, &categoriesStr,
		)
		if err != nil {
			return nil, err
		}
		if categoriesStr.Valid {
			p.Categories = strings.Split(categoriesStr.String, ",")
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func GetUserByID(id string) (*User, error) {
	var user User
	err := db.QueryRow("SELECT id, username, email FROM users WHERE id = ?", id).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
