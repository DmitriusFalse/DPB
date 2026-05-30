package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/handler"
	syncsvc "danbooru-prompt-builder/sync"
	"danbooru-prompt-builder/tray"

	"github.com/getlantern/systray"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
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

	// HTTP server in background
	go func() {
		log.Printf("Server starting on http://127.0.0.1:%d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// On Ctrl+C → tell systray to quit so main goroutine can proceed
	go func() {
		<-ctx.Done()
		log.Println("Signal received, shutting down...")
		systray.Quit()
	}()

	// Systray must run on main goroutine
	configPath, _ := filepath.Abs("config.json")
	tray.Run(cfg.Port, tray.Actions{
		PacksPath:  cfg.TagsPath,
		ConfigPath: configPath,
	})

	// After systray.Run returns (user clicked Exit or Ctrl+C)
	log.Println("Shutting down server...")
	server.Shutdown(context.Background())
}
