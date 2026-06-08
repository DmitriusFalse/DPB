package handler

import (
	"net/http"
)

func handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		serveEmbedded(w, r, "/index.html")
	}
}

func handleSettingsPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveEmbedded(w, r, "/settings.html")
	}
}
