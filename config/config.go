package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port     int    `json:"port"`
	TagsPath string `json:"tags_path"`
	DBPath   string `json:"db_path"`
}

func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
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

	cfg.TagsPath = resolvePath(cfgDir, cfg.TagsPath)
	cfg.DBPath = resolvePath(cfgDir, cfg.DBPath)

	return cfg, nil
}

func resolvePath(baseDir, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(baseDir, target)
}

