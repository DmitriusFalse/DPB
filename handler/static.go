package handler

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:static
var staticFS embed.FS

func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}

func serveEmbedded(w http.ResponseWriter, r *http.Request, path string) {
	data, err := staticFS.ReadFile("static" + path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := detectContentType(path)
	w.Header().Set("Content-Type", ct)
	w.Write(data)
}

func detectContentType(path string) string {
	switch {
	case len(path) >= 5 && path[len(path)-5:] == ".html":
		return "text/html; charset=utf-8"
	case len(path) >= 4 && path[len(path)-4:] == ".css":
		return "text/css; charset=utf-8"
	case len(path) >= 3 && path[len(path)-3:] == ".js":
		return "application/javascript; charset=utf-8"
	case len(path) >= 5 && path[len(path)-5:] == ".json":
		return "application/json"
	case len(path) >= 4 && path[len(path)-4:] == ".ico":
		return "image/x-icon"
	case len(path) >= 4 && path[len(path)-4:] == ".png":
		return "image/png"
	default:
		return "text/plain; charset=utf-8"
	}
}
