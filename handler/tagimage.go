package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"danbooru-prompt-builder/database"
)

var transparentGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61,
	0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00,
	0x21, 0xF9, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x2C, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01,
	0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3B,
}

func handleTagImage(repo *database.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packID, _ := strconv.Atoi(r.URL.Query().Get("pack_id"))
		tagName := r.URL.Query().Get("tag")
		if packID <= 0 || tagName == "" {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}

		pack, err := repo.GetPackByID(packID)
		if err != nil || pack == nil {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}
		packPath := pack.Path

		tagPath := filepath.Join(packPath, "img", tagName+".png")
		if category, _ := repo.GetTagCategory(packID, tagName); category != "" {
			catPath := filepath.Join(packPath, "img", category, tagName+".png")
			if _, err2 := os.Stat(catPath); err2 == nil {
				tagPath = catPath
			}
		}

		f, err := os.Open(tagPath)
		if err != nil {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=86400")
		http.ServeContent(w, r, tagName+".png", stat.ModTime(), f)
	}
}
