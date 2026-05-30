package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"danbooru-prompt-builder/database"
)

func handleFavorites(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packID, _ := strconv.Atoi(r.URL.Query().Get("pack_id"))

		switch r.Method {
		case http.MethodGet:
			tags, err := repo.GetFavorites(packID)
			if err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if tags == nil {
				tags = []database.Tag{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tags)

		case http.MethodPost:
			var body struct {
				PackID  int    `json:"pack_id"`
				TagName string `json:"tag_name"`
				Add     bool   `json:"add"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				jsonError(w, "invalid body", http.StatusBadRequest)
				return
			}
			if body.PackID > 0 {
				packID = body.PackID
			}

			var err error
			if body.Add {
				err = repo.AddFavorite(packID, body.TagName)
			} else {
				err = repo.RemoveFavorite(packID, body.TagName)
			}
			if err != nil {
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
