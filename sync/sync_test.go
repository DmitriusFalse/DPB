package sync

import (
	"os"
	"path/filepath"
	"testing"

	"danbooru-prompt-builder/database"
)

func TestSync_NewPack(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	packDir := filepath.Join(dir, "tags", "testpack")
	os.MkdirAll(packDir, 0755)
	os.WriteFile(filepath.Join(packDir, "0_general_test.csv"), []byte("t1,general,test,\nt2,general,test,\n"), 0644)

	db, err := database.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewService(db)
	err = svc.Sync(filepath.Join(dir, "tags"))
	if err != nil {
		t.Fatal(err)
	}

	repo := database.NewRepo(db)
	packs, _ := repo.GetPacks()
	if len(packs) != 1 {
		t.Fatalf("got %d packs, want 1", len(packs))
	}
	if packs[0].Name != "testpack" {
		t.Errorf("Name = %q", packs[0].Name)
	}

	results, _ := repo.SearchTags(packs[0].ID, "t1", 10)
	if len(results) != 1 {
		t.Errorf("expected tag t1, got %d results", len(results))
	}
}

func TestSync_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	packDir := filepath.Join(dir, "tags", "testpack")
	os.MkdirAll(packDir, 0755)
	os.WriteFile(filepath.Join(packDir, "0_general_test.csv"), []byte("t1,general,test,\n"), 0644)

	db, _ := database.Init(dbPath)
	svc := NewService(db)
	svc.Sync(filepath.Join(dir, "tags"))

	repo := database.NewRepo(db)
	results, _ := repo.SearchTags(1, "t1", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 tag after first sync, got %d", len(results))
	}

	svc.Sync(filepath.Join(dir, "tags"))
	results, _ = repo.SearchTags(1, "t1", 10)
	if len(results) != 1 {
		t.Errorf("expected 1 tag after second sync (idempotent), got %d", len(results))
	}
	db.Close()
}

func TestSync_ModifiedFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	packDir := filepath.Join(dir, "tags", "testpack")
	os.MkdirAll(packDir, 0755)
	csvPath := filepath.Join(packDir, "0_general_test.csv")

	os.WriteFile(csvPath, []byte("t1,general,test,\n"), 0644)
	db, _ := database.Init(dbPath)
	svc := NewService(db)
	svc.Sync(filepath.Join(dir, "tags"))

	os.WriteFile(csvPath, []byte("t1,general,test,\nt2,general,test,\n"), 0644)
	svc.Sync(filepath.Join(dir, "tags"))

	repo := database.NewRepo(db)
	results, _ := repo.SearchTags(1, "", 100)
	if len(results) != 2 {
		t.Fatalf("expected 2 tags after modification, got %d", len(results))
	}
	db.Close()
}

func TestSync_StalePackRemoved(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	packDir := filepath.Join(dir, "tags", "testpack")
	os.MkdirAll(packDir, 0755)
	os.WriteFile(filepath.Join(packDir, "0_general_test.csv"), []byte("t1,general,test,\n"), 0644)

	db, _ := database.Init(dbPath)
	svc := NewService(db)
	svc.Sync(filepath.Join(dir, "tags"))

	os.RemoveAll(packDir)
	svc.Sync(filepath.Join(dir, "tags"))

	repo := database.NewRepo(db)
	packs, _ := repo.GetPacks()
	if len(packs) != 0 {
		t.Errorf("expected 0 packs after removal, got %d", len(packs))
	}
	db.Close()
}

func TestSync_MultiplePacks(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	for _, name := range []string{"pack_a", "pack_b"} {
		pd := filepath.Join(dir, "tags", name)
		os.MkdirAll(pd, 0755)
		os.WriteFile(filepath.Join(pd, "0_general_test.csv"), []byte("t1,general,test,\n"), 0644)
	}

	db, _ := database.Init(dbPath)
	svc := NewService(db)
	svc.Sync(filepath.Join(dir, "tags"))

	repo := database.NewRepo(db)
	packs, _ := repo.GetPacks()
	if len(packs) != 2 {
		t.Errorf("expected 2 packs, got %d", len(packs))
	}
	db.Close()
}

func TestSync_EmptyTagsDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	os.MkdirAll(filepath.Join(dir, "tags"), 0755)

	db, _ := database.Init(dbPath)
	svc := NewService(db)
	err := svc.Sync(filepath.Join(dir, "tags"))
	if err != nil {
		t.Fatal(err)
	}

	repo := database.NewRepo(db)
	packs, _ := repo.GetPacks()
	if len(packs) != 0 {
		t.Errorf("expected 0 packs, got %d", len(packs))
	}
	db.Close()
}
