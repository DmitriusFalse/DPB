package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"danbooru-prompt-builder/database"
)

func handleSearch(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packID, _ := strconv.Atoi(r.URL.Query().Get("pack_id"))
		query := r.URL.Query().Get("q")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 20
		}

		tags, err := repo.SearchTags(packID, query, limit)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tags == nil {
			tags = []database.Tag{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tags)
	}
}
