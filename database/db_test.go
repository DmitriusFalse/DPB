package database

import (
	"database/sql"
	"os"
	"testing"
)

func testDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/test.db"
	db, err := Init(path)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return db, func() {
		db.Close()
		os.Remove(path)
	}
}

func TestInit_CreatesTables(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	expected := []string{
		"packs", "files", "tags", "favorite_tags",
		"saved_prompts", "tag_presets",
	}

	for _, name := range expected {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count == 0 {
			t.Errorf("table %s not found", name)
		}
	}
}

func TestInit_SeedsDefaultPreset(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM tag_presets WHERE name='Pony Quality'`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Error("Pony Quality preset not seeded")
	}
}

func TestInit_SeedIsIdempotent(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	var count1 int
	db.QueryRow(`SELECT COUNT(*) FROM tag_presets`).Scan(&count1)

	repo := NewRepo(db)
	repo.SeedDefaultPreset()
	repo.SeedDefaultPreset()

	var count2 int
	db.QueryRow(`SELECT COUNT(*) FROM tag_presets`).Scan(&count2)

	if count1 != count2 {
		t.Errorf("seed not idempotent: %d -> %d", count1, count2)
	}
}

func TestInit_ForeignKeysEnabled(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.db"
	db, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var enabled int
	db.QueryRow(`PRAGMA foreign_keys`).Scan(&enabled)
	if enabled != 1 {
		t.Error("foreign keys not enabled")
	}
}

func TestInit_WALMode(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.db"
	db, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var journal string
	db.QueryRow(`PRAGMA journal_mode`).Scan(&journal)
	if journal != "wal" {
		t.Errorf("journal_mode = %q, want wal", journal)
	}
}
