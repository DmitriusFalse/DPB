package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/handler"
	"danbooru-prompt-builder/logger"
	syncsvc "danbooru-prompt-builder/sync"
	"danbooru-prompt-builder/tray"

	webview "github.com/webview/webview_go"
)

//go:embed version.txt
var buildVersion string

func main() {
	exe, _ := os.Executable()
	cfgPath := filepath.Join(filepath.Dir(exe), "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logLevel := logger.LevelError
	if cfg.LogLevel == "debug" {
		logLevel = logger.LevelDebug
	}
	logger.Init(logLevel, cfg.LogsDir)

	db, err := database.Init(cfg.DBPath)
	if err != nil {
		logger.Error("Failed to init database: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	syncSvc := syncsvc.NewService(db)
	if err := syncSvc.Sync(cfg.TagsPath); err != nil {
		logger.Error("Initial sync: %v", err)
	}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, db, cfg, syncSvc, cfgPath)

	version := strings.TrimSpace(buildVersion)
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": version})
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.Port),
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Debug("Server starting on http://127.0.0.1:%d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error: %v", err)
		}
	}()

	destroyWebview := make(chan struct{})
	var destroyOnce sync.Once

	go func() {
		<-ctx.Done()
		logger.Debug("Signal received, shutting down...")
		destroyOnce.Do(func() { close(destroyWebview) })
	}()

	go tray.Run(cfg.Port, tray.Actions{
		PacksPath: cfg.TagsPath,
		OnQuit: func() {
			destroyOnce.Do(func() { close(destroyWebview) })
		},
	})

	addr := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
	w := webview.New(false)
	w.SetTitle("Danbooru Prompt Builder")
	w.SetSize(800, 600, webview.HintMax)
	w.Navigate(addr)

	go func() {
		<-destroyWebview
		w.Destroy()
	}()

	w.Run()

	tray.Quit()
	logger.Debug("Shutting down server...")
	server.Shutdown(context.Background())
}
