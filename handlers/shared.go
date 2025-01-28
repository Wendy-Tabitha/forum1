package handlers

import (
	"database/sql"
	"html/template"
)

// Shared variables for the handlers package
var (
	db        *sql.DB
	templates *template.Template
)
