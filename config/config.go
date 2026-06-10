package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port          int    `json:"port"`
	TagsPath      string `json:"tags_path"`
	DBPath        string `json:"db_path"`
	LogLevel      string `json:"log_level"`
	LogsDir       string `json:"logs_dir"`
	StaticImgPath string `json:"static_img_path"`
	ComfyEnabled  bool   `json:"comfy_enabled"`
	ComfyAddress  string `json:"comfy_address"`
	SavePath      string `json:"save_path"`
	Resolutions   string `json:"resolutions"`
	WorkflowsPath string `json:"-"`
}

func defaultConfig() *Config {
	return &Config{
		Port:          8080,
		TagsPath:      "./tags",
		DBPath:        "./data.db",
		LogsDir:       "./logs",
		LogLevel:      "error",
		StaticImgPath: "./img",
		ComfyEnabled:  false,
		ComfyAddress:  "http://127.0.0.1:8188",
		SavePath:      "./output",
		Resolutions:   "Square 1:1#512x512\nSquare HD 1:1#768x768\nSquare XL 1:1#1024x1024\nPortrait 2:3#768x1152\nLandscape 3:2#1152x768\nPortrait 3:4#768x1024\nLandscape 4:3#1024x768\nPortrait Tall 4:7#768x1344\nUltra Wide 7:4#1344x768\nPortrait 9:16#720x1280\nPortrait Wide 13:19#832x1216\nWidescreen 16:9#1280x720\nLandscape Wide 19:13#1216x832",
	}
}

func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig()
			cfgDir := filepath.Dir(absPath)
			cfg.TagsPath = resolvePath(cfgDir, cfg.TagsPath)
			cfg.DBPath = resolvePath(cfgDir, cfg.DBPath)
			cfg.LogsDir = resolvePath(cfgDir, cfg.LogsDir)
			cfg.StaticImgPath = resolvePath(cfgDir, cfg.StaticImgPath)
					cfg.SavePath = resolvePath(cfgDir, cfg.SavePath)
			if err := cfg.Save(absPath); err != nil {
				return nil, fmt.Errorf("create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfgDir := filepath.Dir(absPath)

	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.TagsPath == "" {
		cfg.TagsPath = "./tags"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "./data.db"
	}
	if cfg.LogsDir == "" {
		cfg.LogsDir = "./logs"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}
	if cfg.StaticImgPath == "" {
		cfg.StaticImgPath = "./img"
	}

	cfg.TagsPath = resolvePath(cfgDir, cfg.TagsPath)
	cfg.DBPath = resolvePath(cfgDir, cfg.DBPath)
	cfg.LogsDir = resolvePath(cfgDir, cfg.LogsDir)
		cfg.StaticImgPath = resolvePath(cfgDir, cfg.StaticImgPath)
	if cfg.ComfyAddress == "" {
		cfg.ComfyAddress = "http://127.0.0.1:8188"
	}
	if cfg.SavePath == "" {
		cfg.SavePath = "./output"
	}
	if cfg.Resolutions == "" {
		cfg.Resolutions = "Square 1:1#512x512\nSquare HD 1:1#768x768\nSquare XL 1:1#1024x1024\nPortrait 2:3#768x1152\nLandscape 3:2#1152x768\nPortrait 3:4#768x1024\nLandscape 4:3#1024x768\nPortrait Tall 4:7#768x1344\nUltra Wide 7:4#1344x768\nPortrait 9:16#720x1280\nPortrait Wide 13:19#832x1216\nWidescreen 16:9#1280x720\nLandscape Wide 19:13#1216x832"
	}
	cfg.SavePath = resolvePath(cfgDir, cfg.SavePath)

	return cfg, nil
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func resolvePath(baseDir, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(baseDir, target)
}

