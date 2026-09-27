package webapp

import (
	"embed"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func ServeHomeHTML(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	content, err := staticFS.ReadFile("static/home.html")
	if err != nil {
		http.Error(w, "home page not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func ServeWebAppHTML(w http.ResponseWriter, r *http.Request) {
	content, err := staticFS.ReadFile("static/room.html")
	if err != nil {
		http.Error(w, "web app page not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}
