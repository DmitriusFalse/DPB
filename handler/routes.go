package handler

import (
	"database/sql"
	"net/http"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/sync"
)

func RegisterRoutes(mux *http.ServeMux, db *sql.DB, cfg *config.Config, syncSvc *sync.Service) {
	repo := database.NewRepo(db)

	mux.Handle("/static/", http.StripPrefix("/static/", StaticHandler()))

	mux.HandleFunc("/", handleIndex(cfg))
	mux.HandleFunc("/packs", handlePacksPage())

	mux.HandleFunc("/api/packs", handlePacks(repo, cfg))
	mux.HandleFunc("/api/sync", handleSync(syncSvc, cfg))
	mux.HandleFunc("/api/tags/search", handleSearch(repo))
	mux.HandleFunc("/api/tags/tree", handleTree(repo))
	mux.HandleFunc("/api/favorites", handleFavorites(repo))
	mux.HandleFunc("/api/presets", handlePresets(repo))
	mux.HandleFunc("/api/tags/image", handleTagImage(repo, cfg))
	mux.HandleFunc("/api/prompts", handlePrompts(repo))
}
