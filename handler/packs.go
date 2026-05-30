package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
)

func handlePacks(repo *database.Repo, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			packs, err := repo.GetPacks()
			if err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if packs == nil {
				packs = []database.Pack{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(packs)

		case http.MethodDelete:
			idStr := r.URL.Query().Get("id")
			id, _ := strconv.Atoi(idStr)
			if id <= 0 {
				jsonError(w, "id required", http.StatusBadRequest)
				return
			}
			if err := repo.DeletePack(id); err != nil {
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
