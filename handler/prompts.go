package handler

import (
	"encoding/json"
	"net/http"

	"danbooru-prompt-builder/database"
)

func handlePrompts(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
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
	}
}
