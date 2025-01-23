package handlers

import (
    "net/http"
    "html/template"
)

// UserHandler handles the user page
func UserHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("templates/user.html")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    tmpl.Execute(w, nil)
}
