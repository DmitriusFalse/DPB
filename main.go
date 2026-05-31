package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/handler"
	"danbooru-prompt-builder/logger"
	syncsvc "danbooru-prompt-builder/sync"
	"danbooru-prompt-builder/tray"

	"github.com/getlantern/systray"
)

func main() {
	cfg, err := config.Load("config.json")
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
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, db, cfg, syncSvc)

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

	go func() {
		<-ctx.Done()
		logger.Debug("Signal received, shutting down...")
		systray.Quit()
	}()

	configPath, _ := filepath.Abs("config.json")
	tray.Run(cfg.Port, tray.Actions{
		PacksPath:  cfg.TagsPath,
		ConfigPath: configPath,
	})

	logger.Debug("Shutting down server...")
	server.Shutdown(context.Background())
}
