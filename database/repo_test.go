package database

import (
	"os"
	"testing"
)

func testRepo(t *testing.T) (*Repo, func()) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/test.db"
	db, err := Init(path)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return NewRepo(db), func() {
		db.Close()
		os.Remove(path)
	}
}

func TestPacks_CreateAndGet(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, err := repo.CreatePack("test", "/path/to/test")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "test" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.ID == 0 {
		t.Error("ID is 0")
	}

	packs, err := repo.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 {
		t.Fatalf("got %d packs, want 1", len(packs))
	}
	if packs[0].Name != "test" {
		t.Errorf("Name = %q", packs[0].Name)
	}
}

func TestPacks_GetByName(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	repo.CreatePack("test", "/path")

	p, err := repo.GetPackByName("test")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("pack is nil")
	}
	if p.Name != "test" {
		t.Errorf("Name = %q", p.Name)
	}

	p, err = repo.GetPackByName("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Error("expected nil for nonexistent pack")
	}
}

func TestPacks_Delete(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")
	repo.DeletePack(p.ID)

	packs, _ := repo.GetPacks()
	if len(packs) != 0 {
		t.Error("pack not deleted")
	}
}

func TestPacks_CascadeDelete(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")
	fid, _ := repo.UpsertFile(p.ID, "file.csv", 0, "general", "sub", "hash123")
	repo.InsertTags([]Tag{
		{FileID: fid, PackID: p.ID, TagName: "tag1", CategoryName: "general", SubcategoryName: "sub"},
	})

	repo.DeletePack(p.ID)

	var count int
	repo.db.QueryRow(`SELECT COUNT(*) FROM tags`).Scan(&count)
	if count != 0 {
		t.Error("tags not cascade deleted")
	}

	repo.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&count)
	if count != 0 {
		t.Error("files not cascade deleted")
	}
}

func TestFiles_UpsertAndGet(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")

	fid, err := repo.UpsertFile(p.ID, "file.csv", 0, "general", "appearance", "hash1")
	if err != nil {
		t.Fatal(err)
	}
	if fid == 0 {
		t.Error("file ID is 0")
	}

	f, err := repo.GetFileByPackAndName(p.ID, "file.csv")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("file is nil")
	}
	if f.CategoryName != "general" {
		t.Errorf("CategoryName = %q", f.CategoryName)
	}
	if f.SubcategoryName != "appearance" {
		t.Errorf("SubcategoryName = %q", f.SubcategoryName)
	}

	// Upsert again with different hash -> updates in place
	fid2, _ := repo.UpsertFile(p.ID, "file.csv", 0, "general", "appearance", "hash2")
	if fid2 != fid {
		t.Errorf("expected same file ID %d, got %d", fid, fid2)
	}

	updated, _ := repo.GetFileByPackAndName(p.ID, "file.csv")
	if updated.FileHash != "hash2" {
		t.Errorf("FileHash = %q, want hash2", updated.FileHash)
	}
}

func TestFiles_DeleteByPack(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")
	repo.UpsertFile(p.ID, "f1.csv", 0, "gen", "sub", "h1")
	repo.UpsertFile(p.ID, "f2.csv", 0, "gen", "sub", "h2")
	repo.DeleteFilesByPack(p.ID)

	f, _ := repo.GetFileByPackAndName(p.ID, "f1.csv")
	if f != nil {
		t.Error("file not deleted")
	}
}

func TestFiles_GetNonexistent(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	f, err := repo.GetFileByPackAndName(999, "none.csv")
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestTags_InsertAndSearch(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")
	fid, _ := repo.UpsertFile(p.ID, "file.csv", 0, "general", "appearance", "h")
	repo.InsertTags([]Tag{
		{FileID: fid, PackID: p.ID, TagName: "long-hair", CategoryName: "general", SubcategoryName: "appearance", Aliases: "/lh,longhair"},
		{FileID: fid, PackID: p.ID, TagName: "short-hair", CategoryName: "general", SubcategoryName: "appearance", Aliases: ""},
		{FileID: fid, PackID: p.ID, TagName: "blonde-hair", CategoryName: "general", SubcategoryName: "appearance", Aliases: "blonde,blond"},
	})

	results, err := repo.SearchTags(p.ID, "long", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].TagName != "long-hair" {
		t.Errorf("TagName = %q", results[0].TagName)
	}

	results, err = repo.SearchTags(p.ID, "blond", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected search by alias to find results")
	}
}

func TestTags_DeleteByFile(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")
	fid, _ := repo.UpsertFile(p.ID, "f.csv", 0, "gen", "sub", "h")
	repo.InsertTags([]Tag{
		{FileID: fid, PackID: p.ID, TagName: "t1", CategoryName: "gen", SubcategoryName: "sub"},
	})

	repo.DeleteTagsByFile(fid)
	results, _ := repo.SearchTags(p.ID, "t1", 10)
	if len(results) != 0 {
		t.Error("tags not deleted")
	}
}

func TestTags_CategoryTree(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")
	fid1, _ := repo.UpsertFile(p.ID, "f1.csv", 0, "general", "appearance", "h")
	fid2, _ := repo.UpsertFile(p.ID, "f2.csv", 3, "copyright", "anime_manga", "h")
	repo.InsertTags([]Tag{
		{FileID: fid1, PackID: p.ID, TagName: "t1", CategoryName: "general", SubcategoryName: "appearance"},
		{FileID: fid2, PackID: p.ID, TagName: "touhou", CategoryName: "copyright", SubcategoryName: "anime_manga"},
	})

	cats, err := repo.GetCategoryTree(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 2 {
		t.Fatalf("got %d categories, want 2", len(cats))
	}

	subs, err := repo.GetSubcategories(p.ID, "general")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0] != "appearance" {
		t.Errorf("subs = %v", subs)
	}

	tags, total, err := repo.GetTagsByCategory(p.ID, "general", "appearance", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("total = %d", total)
	}
	if len(tags) != 1 || tags[0].TagName != "t1" {
		t.Errorf("tags = %v", tags)
	}
}

func TestFavorites_CRUD(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")

	repo.AddFavorite(p.ID, "tag1")
	repo.AddFavorite(p.ID, "tag2")
	repo.AddFavorite(p.ID, "tag1") // duplicate

	favs, err := repo.GetFavorites(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(favs) != 2 {
		t.Fatalf("got %d favorites, want 2", len(favs))
	}

	isFav, _ := repo.IsFavorite(p.ID, "tag1")
	if !isFav {
		t.Error("tag1 should be favorite")
	}
	isFav, _ = repo.IsFavorite(p.ID, "nonexistent")
	if isFav {
		t.Error("nonexistent should not be favorite")
	}

	repo.RemoveFavorite(p.ID, "tag1")
	favs, _ = repo.GetFavorites(p.ID)
	if len(favs) != 1 {
		t.Fatalf("got %d favorites, want 1", len(favs))
	}
}

func TestPrompts_SaveAndGet(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, err := repo.SavePrompt("test", "tag1, tag2", "bad1", true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "test" {
		t.Errorf("Name = %q", p.Name)
	}
	if !p.IsFavorite {
		t.Error("IsFavorite should be true")
	}

	favs, err := repo.GetFavoritesPrompts()
	if err != nil {
		t.Fatal(err)
	}
	if len(favs) != 1 {
		t.Fatalf("got %d fav prompts, want 1", len(favs))
	}
}

func TestPrompts_History(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		repo.SavePrompt("", "t1", "n1", false)
	}

	history, err := repo.GetHistory(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Fatalf("got %d history items, want 3", len(history))
	}
}

func TestPrompts_TrimHistory(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		repo.SavePrompt("", "t", "n", false)
	}

	repo.TrimHistory(3)

	history, _ := repo.GetHistory(100)
	if len(history) != 3 {
		t.Fatalf("got %d history items, want 3", len(history))
	}
}

func TestPrompts_Delete(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.SavePrompt("test", "t", "n", false)
	repo.DeletePrompt(p.ID)

	history, _ := repo.GetHistory(100)
	if len(history) != 0 {
		t.Error("prompt not deleted")
	}
}

func TestPresets_CRUD(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, err := repo.SavePreset("My Preset", []string{"tag1", "tag2"}, []string{"bad1"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "My Preset" {
		t.Errorf("Name = %q", p.Name)
	}

	presets, err := repo.GetPresets()
	if err != nil {
		t.Fatal(err)
	}
	if len(presets) == 0 {
		t.Fatal("no presets")
	}

	found := false
	for _, pr := range presets {
		if pr.Name == "My Preset" {
			found = true
			break
		}
	}
	if !found {
		t.Error("My Preset not found")
	}

	repo.DeletePreset(p.ID)
	presets, _ = repo.GetPresets()
	for _, pr := range presets {
		if pr.ID == p.ID {
			t.Error("preset not deleted")
		}
	}
}

func TestPresets_DefaultSeed(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	presets, _ := repo.GetPresets()
	found := false
	for _, p := range presets {
		if p.Name == "Pony Quality" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Pony Quality preset not seeded")
	}
}

func TestPresets_UpdateExisting(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	repo.SavePreset("Test", []string{"a"}, []string{"b"})
	repo.SavePreset("Test", []string{"x", "y"}, []string{"z"})

	presets, _ := repo.GetPresets()
	for _, p := range presets {
		if p.Name == "Test" {
			if p.PositiveTags != `["x","y"]` {
				t.Errorf("PositiveTags = %q", p.PositiveTags)
			}
		}
	}
}

func TestPacks_EmptyList(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	packs, err := repo.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 0 {
		t.Error("expected empty list")
	}
}

func TestTags_SearchNoResults(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	results, err := repo.SearchTags(1, "nonexistent", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Error("expected empty results")
	}
}

func TestTags_SearchLimit(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	p, _ := repo.CreatePack("test", "/path")
	fid, _ := repo.UpsertFile(p.ID, "f.csv", 0, "gen", "sub", "h")
	var tags []Tag
	for i := 0; i < 100; i++ {
		tags = append(tags, Tag{
			FileID: fid, PackID: p.ID,
			TagName:         "tag-" + string(rune('a'+i)),
			CategoryName:    "gen",
			SubcategoryName: "sub",
		})
	}
	repo.InsertTags(tags)

	results, _ := repo.SearchTags(p.ID, "tag", 5)
	if len(results) != 5 {
		t.Errorf("got %d results, want 5", len(results))
	}
}

func TestFavorites_NoPack(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	favs, err := repo.GetFavorites(999)
	if err != nil {
		t.Fatal(err)
	}
	if len(favs) != 0 {
		t.Error("expected empty favorites")
	}
}

func TestPrompts_EmptyHistory(t *testing.T) {
	repo, cleanup := testRepo(t)
	defer cleanup()

	history, err := repo.GetHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Error("expected empty history")
	}
}

func createTestPack(t *testing.T, repo *Repo) (*Pack, *File) {
	t.Helper()
	p, err := repo.CreatePack("test", "/path")
	if err != nil {
		t.Fatalf("CreatePack: %v", err)
	}
	fid, err := repo.UpsertFile(p.ID, "0_general_test.csv", 0, "general", "test", "hash")
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	return p, &File{ID: fid, PackID: p.ID}
}

func TestNewRepo(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.db"
	db, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewRepo(db)
	if repo == nil {
		t.Fatal("repo is nil")
	}
}
