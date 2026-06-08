package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/sync"
)

func handleGetPackByID(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.Atoi(idStr)
		if id <= 0 {
			jsonError(w, "id required", http.StatusBadRequest)
			return
		}
		pack, err := repo.GetPackByID(id)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if pack == nil {
			jsonError(w, "pack not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pack)
	}
}

func handleReadPackInfoFromReader(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.Atoi(idStr)
		if id <= 0 {
			jsonError(w, "id required", http.StatusBadRequest)
			return
		}
		pack, err := repo.GetPackByID(id)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if pack == nil {
			jsonError(w, "pack not found", http.StatusNotFound)
			return
		}
		infoPath := filepath.Join(pack.Path, "info.pack")
		f, err := os.Open(infoPath)
		if err != nil {
			jsonError(w, "info.pack not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		info, err := sync.ReadPackInfoFromReader(f)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

func handlePacks(repo *database.Repo) http.HandlerFunc {
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
