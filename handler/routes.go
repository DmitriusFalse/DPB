package handler

import (
	"database/sql"
	"net/http"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/sync"
)

func RegisterRoutes(mux *http.ServeMux, db *sql.DB, cfg *config.Config, syncSvc *sync.Service, configPath string) {
	repo := database.NewRepo(db)

	mux.Handle("/static/", http.StripPrefix("/static/", StaticHandler()))

	mux.HandleFunc("/", handleIndex(cfg))
	mux.HandleFunc("/settings", handleSettingsPage())

	api := func(h http.HandlerFunc) http.HandlerFunc {
		return apiMiddleware(h)
	}

	mux.HandleFunc("/api/config", api(handleConfig(cfg, configPath)))
	mux.HandleFunc("/api/packs", api(handlePacks(repo, cfg)))
	mux.HandleFunc("/api/sync", api(handleSync(syncSvc, cfg)))
	mux.HandleFunc("/api/tags/search", api(handleSearch(repo)))
	mux.HandleFunc("/api/tags/tree", api(handleTree(repo)))
	mux.HandleFunc("/api/favorites", api(handleFavorites(repo)))
	mux.HandleFunc("/api/presets", api(handlePresets(repo)))
	mux.HandleFunc("/api/tags/image", api(handleTagImage(repo, cfg)))
	mux.HandleFunc("/api/prompts", api(handlePrompts(repo)))
}
