package main

import (
	"context"
	"encoding/binary"
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
	"time"
	"unsafe"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/handler"
	"danbooru-prompt-builder/logger"
	syncsvc "danbooru-prompt-builder/sync"
	"danbooru-prompt-builder/tray"

	webview "github.com/webview/webview_go"
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	procFindWindow             = user32.NewProc("FindWindowW")
	procShowWindow             = user32.NewProc("ShowWindow")
	procSendMessageW           = user32.NewProc("SendMessageW")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
)

const (
	SW_MAXIMIZE = 3
	WM_SETICON  = 0x0080
	ICON_SMALL  = 0
	ICON_BIG    = 1
)

func maximizeWindow(title string) {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	hwnd, _, _ := syscall.Syscall(procFindWindow.Addr(), 2, 0, uintptr(unsafe.Pointer(titlePtr)), 0)
	if hwnd != 0 {
		syscall.Syscall(procShowWindow.Addr(), 2, hwnd, SW_MAXIMIZE, 0)
		setWindowIcon(hwnd)
	}
}

func setWindowIcon(hwnd uintptr) {
	icoData := tray.IconData
	if len(icoData) < 6 {
		return
	}
	count := int(binary.LittleEndian.Uint16(icoData[4:6]))
	if count == 0 {
		return
	}
	// Find the largest image
	bestIdx := 0
	bestW := 0
	for i := 0; i < count && 6+(i+1)*16 <= len(icoData); i++ {
		w := int(icoData[6+i*16])
		if w == 0 {
			w = 256
		}
		if w > bestW {
			bestW = w
			bestIdx = i
		}
	}
	if 6+(bestIdx+1)*16 > len(icoData) {
		return
	}
	dirEntry := icoData[6+bestIdx*16:]
	imgOffset := binary.LittleEndian.Uint32(dirEntry[12:16])
	imgSize := binary.LittleEndian.Uint32(dirEntry[8:12])
	if int(imgOffset+imgSize) > len(icoData) {
		return
	}
	imgData := icoData[imgOffset : imgOffset+imgSize]
	hicon, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&imgData[0])),
		uintptr(imgSize),
		1,              // fIcon = TRUE
		0x00030000,     // dwVer = 3.00
		0, 0, 0,        // default size, no flags
	)
	if hicon == 0 {
		return
	}
	procSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, hicon)
	procSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, hicon)
}

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
	w.SetSize(1280, 900, webview.HintNone)
	w.Navigate(addr)

	go func() {
		<-destroyWebview
		w.Destroy()
	}()

	go func() {
		time.Sleep(150 * time.Millisecond)
		maximizeWindow("Danbooru Prompt Builder")
	}()

	w.Run()

	tray.Quit()
	logger.Debug("Shutting down server...")
	server.Shutdown(context.Background())
}
