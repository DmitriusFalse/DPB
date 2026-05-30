package handler

import (
	"net/http"

	"danbooru-prompt-builder/config"
)

func handleIndex(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		serveEmbedded(w, r, "/index.html")
	}
}

func handlePacksPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveEmbedded(w, r, "/packs.html")
	}
}
