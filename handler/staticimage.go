package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"danbooru-prompt-builder/config"
)

var (
	staticTagCatOnce sync.Once
	staticTagCat     map[string]string
)

var tkeyToFolder = map[string]string{
	"const.quality":  "quality",
	"const.sources":  "source",
	"const.rating":   "rating",
	"const.pose":     "pose",
	"const.scene":    "scene",
	"const.style":    "style",
	"const.negative": "negative",
}

func getStaticTagCategory() map[string]string {
	staticTagCatOnce.Do(func() {
		m := map[string]string{}
		data, err := staticFS.ReadFile("static/constants.json")
		if err != nil {
			staticTagCat = m
			return
		}
		var entries []struct {
			TKey          string `json:"tkey"`
			Tags          []string `json:"tags"`
			Subcategories []struct {
				Tags []string `json:"tags"`
			} `json:"subcategories"`
		}
		if err := json.Unmarshal(data, &entries); err != nil {
			staticTagCat = m
			return
		}
		for _, e := range entries {
			folder := tkeyToFolder[e.TKey]
			if folder == "" {
				continue
			}
			for _, tag := range e.Tags {
				m[tag] = folder
			}
			for _, sub := range e.Subcategories {
				for _, tag := range sub.Tags {
					m[tag] = folder
				}
			}
		}
		staticTagCat = m
	})
	return staticTagCat
}

func handleStaticImage(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagName := r.URL.Query().Get("tag")
		if tagName == "" {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}

		cat := getStaticTagCategory()[tagName]
		if cat == "" {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(transparentGIF)
			return
		}

		imgPath := filepath.Join(cfg.StaticImgPath, cat, tagName+".png")
		f, err := os.Open(imgPath)
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
