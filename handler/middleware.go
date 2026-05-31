package handler

import (
	"net/http"

	"danbooru-prompt-builder/logger"
)

func apiMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("Panic in %s %s: %v", r.Method, r.URL.Path, rec)
				jsonError(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}
