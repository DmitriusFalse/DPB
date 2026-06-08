package handler

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"danbooru-prompt-builder/database"
)

type gzipWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (gw gzipWriter) Write(b []byte) (int, error) {
	return gw.writer.Write(b)
}

type TreeCategory struct {
	Name          string                     `json:"name"`
	Subcategories []database.SubcategoryInfo `json:"subcategories"`
	Count         int                        `json:"count"`
}

type TagPage struct {
	Tags  []database.Tag `json:"tags"`
	Total int            `json:"total"`
}

func handleTree(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		packID, _ := strconv.Atoi(r.URL.Query().Get("pack_id"))
		if packID <= 0 {
			jsonError(w, "pack_id required", http.StatusBadRequest)
			return
		}

		// Apply gzip after validation — avoid compressing error responses
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gz := gzip.NewWriter(w)
			w.Header().Set("Content-Encoding", "gzip")
			w = gzipWriter{ResponseWriter: w, writer: gz}
			defer gz.Close()
		}

		catName := r.URL.Query().Get("category")

		// If category specified, return paginated tags for that category
		if catName != "" {
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit <= 0 {
				limit = 500
			}

			tags, total, err := repo.GetTagsByCategory(packID, catName, catName, offset, limit)
			if err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if tags == nil {
				tags = []database.Tag{}
			}
			json.NewEncoder(w).Encode(TagPage{Tags: tags, Total: total})
			return
		}

		// Otherwise return categories with counts (no subcategories)
		categories, err := repo.GetCategoryTree(packID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		catCounts, err := repo.GetCategoryCounts(packID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var tree []TreeCategory
		for _, cat := range categories {
			tree = append(tree, TreeCategory{Name: cat, Subcategories: nil, Count: catCounts[cat]})
		}

		json.NewEncoder(w).Encode(tree)
	}
}
