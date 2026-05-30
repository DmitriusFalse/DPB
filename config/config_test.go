package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{"port": 9090, "tags_path": "./mytags", "db_path": "./mydb.db"}`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 9090 {
		t.Errorf("port = %d, want 9090", cfg.Port)
	}
	if cfg.TagsPath != filepath.Join(dir, "mytags") {
		t.Errorf("tags_path = %s", cfg.TagsPath)
	}
	if cfg.DBPath != filepath.Join(dir, "mydb.db") {
		t.Errorf("db_path = %s", cfg.DBPath)
	}
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{}`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Port)
	}
	if cfg.TagsPath != filepath.Join(dir, "tags") {
		t.Errorf("tags_path = %s, want %s", cfg.TagsPath, filepath.Join(dir, "tags"))
	}
	if cfg.DBPath != filepath.Join(dir, "data.db") {
		t.Errorf("db_path = %s, want %s", cfg.DBPath, filepath.Join(dir, "data.db"))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{invalid`), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
}
