package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"danbooru-prompt-builder/database"
)

func handlePresets(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			presets, err := repo.GetPresets()
			if err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if presets == nil {
				presets = []database.TagPreset{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(presets)

		case http.MethodPost:
			var body struct {
				Name         string   `json:"name"`
				PositiveTags []string `json:"positive_tags"`
				NegativeTags []string `json:"negative_tags"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				jsonError(w, "invalid body", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(body.Name) == "" {
				jsonError(w, "name is required", http.StatusBadRequest)
				return
			}

			preset, err := repo.SavePreset(body.Name, body.PositiveTags, body.NegativeTags)
			if err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(preset)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
