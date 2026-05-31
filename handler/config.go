package handler

import (
	"encoding/json"
	"net/http"

	"danbooru-prompt-builder/config"
)

func handleConfig(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cfg)

		case http.MethodPut:
			var updated config.Config
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				jsonError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			if updated.Port <= 0 {
				updated.Port = 8080
			}
			if updated.TagsPath == "" {
				updated.TagsPath = "./tags"
			}
			if updated.DBPath == "" {
				updated.DBPath = "./data.db"
			}
			if updated.LogsDir == "" {
				updated.LogsDir = "./logs"
			}
			if updated.LogLevel == "" {
				updated.LogLevel = "error"
			}

			*cfg = updated

			if err := cfg.Save(configPath); err != nil {
				jsonError(w, "Save failed: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
