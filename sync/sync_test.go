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

func TestSync_PackMetaSaved(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	packDir := filepath.Join(dir, "tags", "mypack")
	os.MkdirAll(packDir, 0755)

	info := `{
		"name": "MyPack",
		"name_ru": "МойПак",
		"description": "A test",
		"description_ru": "Тест",
		"version": "1.0",
		"author": "me",
		"icon": "📦",
		"categories": [{"name": "test", "file": "0_general_test.csv"}]
	}`
	os.WriteFile(filepath.Join(packDir, "info.pack"), []byte(info), 0644)
	os.WriteFile(filepath.Join(packDir, "0_general_test.csv"), []byte("t1,general,test,\n"), 0644)

	db, err := database.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := NewService(db)
	if err := svc.Sync(filepath.Join(dir, "tags")); err != nil {
		t.Fatal(err)
	}

	repo := database.NewRepo(db)
	packs, _ := repo.GetPacks()
	if len(packs) != 1 {
		t.Fatalf("got %d packs, want 1", len(packs))
	}

	p := packs[0]
	if p.Name != "MyPack" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.NameRu != "МойПак" {
		t.Errorf("NameRu = %q", p.NameRu)
	}
	if p.Description != "A test" {
		t.Errorf("Description = %q", p.Description)
	}
	if p.DescriptionRu != "Тест" {
		t.Errorf("DescriptionRu = %q", p.DescriptionRu)
	}
	if p.Version != "1.0" {
		t.Errorf("Version = %q", p.Version)
	}
	if p.Author != "me" {
		t.Errorf("Author = %q", p.Author)
	}
	if p.Icon != "📦" {
		t.Errorf("Icon = %q", p.Icon)
	}
	if p.Categories == "" {
		t.Error("Categories should not be empty")
	}
}

func TestSync_PackMetaUpdatedOnRescan(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	packDir := filepath.Join(dir, "tags", "testpack")
	os.MkdirAll(packDir, 0755)

	info1 := `{"name": "testpack", "name_ru": "v1", "description": "first", "categories": [{"name": "test", "file": "0_general_test.csv"}]}`
	os.WriteFile(filepath.Join(packDir, "info.pack"), []byte(info1), 0644)
	os.WriteFile(filepath.Join(packDir, "0_general_test.csv"), []byte("t1,general,test,\n"), 0644)

	db, _ := database.Init(dbPath)
	svc := NewService(db)
	svc.Sync(filepath.Join(dir, "tags"))

	info2 := `{"name": "testpack", "name_ru": "v2", "description": "updated", "categories": [{"name": "test", "file": "0_general_test.csv"}]}`
	os.WriteFile(filepath.Join(packDir, "info.pack"), []byte(info2), 0644)

	svc.Sync(filepath.Join(dir, "tags"))

	repo := database.NewRepo(db)
	pack, _ := repo.GetPackByName("testpack")
	if pack == nil {
		t.Fatal("pack not found")
	}
	if pack.NameRu != "v2" {
		t.Errorf("NameRu = %q, want v2", pack.NameRu)
	}
	if pack.Description != "updated" {
		t.Errorf("Description = %q", pack.Description)
	}
	db.Close()
}

func TestSync_AutoGeneratedPackInfo(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	packDir := filepath.Join(dir, "tags", "autopack")
	os.MkdirAll(packDir, 0755)
	os.WriteFile(filepath.Join(packDir, "armor.txt"), []byte("tag1\ntag2\n"), 0644)
	os.WriteFile(filepath.Join(packDir, "weapons.txt"), []byte("tag3\n"), 0644)

	db, _ := database.Init(dbPath)
	svc := NewService(db)
	if err := svc.Sync(filepath.Join(dir, "tags")); err != nil {
		t.Fatal(err)
	}

	repo := database.NewRepo(db)
	packs, _ := repo.GetPacks()
	if len(packs) != 1 {
		t.Fatalf("got %d packs, want 1", len(packs))
	}
	p := packs[0]
	if p.Name != "autopack" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.Categories == "" {
		t.Error("Categories should be auto-generated")
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
