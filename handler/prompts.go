package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"danbooru-prompt-builder/database"
)

func handlePrompts(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			favoritesOnly := r.URL.Query().Get("favorites") == "1"
			var prompts []database.SavedPrompt
			var err error

			if favoritesOnly {
				prompts, err = repo.GetFavoritesPrompts()
			} else {
				prompts, err = repo.GetHistory(50)
			}
			if err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if prompts == nil {
				prompts = []database.SavedPrompt{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(prompts)

		case http.MethodPost:
			var body struct {
				Name         string `json:"name"`
				PositiveText string `json:"positive_text"`
				NegativeText string `json:"negative_text"`
				IsFavorite   bool   `json:"is_favorite"`
				GenData      string `json:"gen_data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				jsonError(w, "invalid body", http.StatusBadRequest)
				return
			}

			prompt, err := repo.SavePrompt(body.Name, body.PositiveText, body.NegativeText, body.IsFavorite, body.GenData)
			if err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := repo.TrimHistory(50); err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(prompt)

		case http.MethodDelete:
			idStr := r.URL.Query().Get("id")
			id, _ := strconv.Atoi(idStr)
			if id <= 0 {
				jsonError(w, "id required", http.StatusBadRequest)
				return
			}
			if err := repo.DeletePrompt(id); err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
