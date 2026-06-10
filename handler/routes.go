package handler

import (
	"database/sql"
	"net/http"
	"path/filepath"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/sync"
)

func RegisterRoutes(mux *http.ServeMux, db *sql.DB, cfg *config.Config, syncSvc *sync.Service, configPath string) {
	repo := database.NewRepo(db)
	cfg.WorkflowsPath = filepath.Join(filepath.Dir(configPath), "Workflows")

	mux.Handle("/static/", http.StripPrefix("/static/", StaticHandler()))

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/icon.ico", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/", handleIndex())
	mux.HandleFunc("/settings", handleSettingsPage())

	api := func(h http.HandlerFunc) http.HandlerFunc {
		return apiMiddleware(h)
	}

	mux.HandleFunc("/api/config", api(handleConfig(cfg, configPath)))
	mux.HandleFunc("/api/pack", api(handleGetPackByID(repo)))
	mux.HandleFunc("/api/pack/info", api(handleReadPackInfoFromReader(repo)))
	mux.HandleFunc("/api/packs", api(handlePacks(repo)))
	mux.HandleFunc("/api/sync", api(handleSync(syncSvc, cfg)))
	mux.HandleFunc("/api/tags/search", api(handleSearch(repo)))
	mux.HandleFunc("/api/tags/tree", api(handleTree(repo)))
	mux.HandleFunc("/api/favorites", api(handleFavorites(repo)))
	mux.HandleFunc("/api/presets", api(handlePresets(repo)))
	mux.HandleFunc("/api/static/image", api(handleStaticImage(cfg)))
	mux.HandleFunc("/api/tags/image", api(handleTagImage(repo)))
	mux.HandleFunc("/api/prompts", api(handlePrompts(repo)))
	mux.HandleFunc("/api/comfy/workflows", api(handleComfyWorkflows(cfg)))
	mux.HandleFunc("/api/comfy/generate", api(handleComfyGenerate(cfg)))
	mux.HandleFunc("/api/comfy/image", api(handleComfyImage(cfg)))
	mux.HandleFunc("/api/comfy/object_info/", api(handleComfyObjectInfo(cfg)))
	mux.HandleFunc("/api/comfy/save-image", api(handleComfySaveImage(cfg)))
	mux.HandleFunc("/api/comfy/ws", api(handleComfyWS(cfg)))
}
