package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"danbooru-prompt-builder/config"
	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/sync"
)

type testEnv struct {
	db     *sql.DB
	repo   *database.Repo
	cfg    *config.Config
	syncSvc *sync.Service
	mux    *http.ServeMux
}

func setupTest(t *testing.T) *testEnv {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	cfg := &config.Config{
		Port:     8080,
		TagsPath: filepath.Join(dir, "tags"),
		DBPath:   dbPath,
	}

	syncSvc := sync.NewService(db)

	mux := http.NewServeMux()
	configPath := filepath.Join(dir, "config.json")
	RegisterRoutes(mux, db, cfg, syncSvc, configPath)

	return &testEnv{
		db:      db,
		repo:    database.NewRepo(db),
		cfg:     cfg,
		syncSvc: syncSvc,
		mux:     mux,
	}
}

func (e *testEnv) seed(t *testing.T) {
	t.Helper()

	p, err := e.repo.CreatePack("testpack", e.cfg.TagsPath+"/testpack")
	if err != nil {
		t.Fatalf("seed CreatePack: %v", err)
	}

	fid, err := e.repo.UpsertFile(p.ID, "0_general_test.csv", 0, "general", "general", "hash123")
	if err != nil {
		t.Fatalf("seed UpsertFile: %v", err)
	}

	e.repo.InsertTags([]database.Tag{
		{FileID: fid, PackID: p.ID, TagName: "tag1", CategoryName: "general", SubcategoryName: "general", Aliases: "alias1"},
		{FileID: fid, PackID: p.ID, TagName: "tag2", CategoryName: "general", SubcategoryName: "general", Aliases: ""},
	})

	e.repo.AddFavorite(p.ID, "tag1")
}

func (e *testEnv) close() {
	e.db.Close()
}

func TestGetPacks(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("GET", "/api/packs", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var packs []database.Pack
	json.Unmarshal(w.Body.Bytes(), &packs)
	if len(packs) != 1 {
		t.Fatalf("got %d packs", len(packs))
	}
	if packs[0].Name != "testpack" {
		t.Errorf("Name = %q", packs[0].Name)
	}
}

func TestGetPacks_NewFields(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	p, err := env.repo.CreatePack("metapack", "/path/to/metapack")
	if err != nil {
		t.Fatal(err)
	}
	env.repo.UpdatePackMeta(p.ID,
		"Full description", "Полное описание",
		"2.0", "developer", "🔧", "МетаПак",
		[]byte(`[{"name":"test","file":"test.txt"}]`),
	)

	req := httptest.NewRequest("GET", "/api/packs", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	var packs []database.Pack
	json.Unmarshal(w.Body.Bytes(), &packs)

	var found *database.Pack
	for i := range packs {
		if packs[i].ID == p.ID {
			found = &packs[i]
			break
		}
	}
	if found == nil {
		t.Fatal("pack not found in response")
	}
	if found.Description != "Full description" {
		t.Errorf("Description = %q", found.Description)
	}
	if found.DescriptionRu != "Полное описание" {
		t.Errorf("DescriptionRu = %q", found.DescriptionRu)
	}
	if found.Version != "2.0" {
		t.Errorf("Version = %q", found.Version)
	}
	if found.Author != "developer" {
		t.Errorf("Author = %q", found.Author)
	}
	if found.Icon != "🔧" {
		t.Errorf("Icon = %q", found.Icon)
	}
	if found.NameRu != "МетаПак" {
		t.Errorf("NameRu = %q", found.NameRu)
	}
	if found.Categories != `[{"name":"test","file":"test.txt"}]` {
		t.Errorf("Categories = %q", found.Categories)
	}
}

func TestGetPacks_Empty(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/packs", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	var packs []database.Pack
	json.Unmarshal(w.Body.Bytes(), &packs)
	if len(packs) != 0 {
		t.Errorf("expected empty list, got %d", len(packs))
	}
}

func TestDeletePack(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("DELETE", "/api/packs?id=1", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q", resp["status"])
	}
}

func TestDeletePack_NoID(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("DELETE", "/api/packs", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSync(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	packDir := filepath.Join(env.cfg.TagsPath, "testpack")
	os.MkdirAll(packDir, 0755)
	os.WriteFile(filepath.Join(packDir, "0_general_test.csv"), []byte("t1,general,test,\n"), 0644)

	req := httptest.NewRequest("POST", "/api/sync", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q", resp["status"])
	}

	packs, _ := env.repo.GetPacks()
	if len(packs) != 1 {
		t.Errorf("expected 1 pack after sync, got %d", len(packs))
	}
}

func TestSync_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/sync", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestSearchTags(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("GET", "/api/tags/search?pack_id=1&q=tag1&limit=10", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var tags []database.Tag
	json.Unmarshal(w.Body.Bytes(), &tags)
	if len(tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(tags))
	}
	if tags[0].TagName != "tag1" {
		t.Errorf("TagName = %q", tags[0].TagName)
	}
}

func TestSearchTags_Empty(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("GET", "/api/tags/search?pack_id=1&q=nonexistent", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	var tags []database.Tag
	json.Unmarshal(w.Body.Bytes(), &tags)
	if len(tags) != 0 {
		t.Errorf("expected empty results")
	}
}

func TestSearchTags_NoPackID(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/tags/search?q=tag", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
}

func TestTree(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("GET", "/api/tags/tree?pack_id=1", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	var tree []TreeCategory
	json.Unmarshal(w.Body.Bytes(), &tree)
	if len(tree) == 0 {
		t.Fatal("tree is empty")
	}

	found := false
	for _, c := range tree {
		if c.Name == "general" {
			found = true
			if c.Subcategories != nil {
				t.Errorf("expected no subcategories, got %d", len(c.Subcategories))
			}
		}
	}
	if !found {
		t.Error("category 'general' not found in tree")
	}
}

func TestTree_NoPackID(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/tags/tree", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTree_SubcategoryTags(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("GET", "/api/tags/tree?pack_id=1&category=general&subcategory=general", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	var page struct {
		Tags  []database.Tag `json:"tags"`
		Total int            `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &page)
	if len(page.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(page.Tags))
	}
	if page.Total != 2 {
		t.Errorf("Total = %d", page.Total)
	}
	if page.Tags[0].TagName != "tag1" {
		t.Errorf("TagName = %q", page.Tags[0].TagName)
	}
	if page.Tags[1].TagName != "tag2" {
		t.Errorf("TagName = %q", page.Tags[1].TagName)
	}
}

func TestTree_CategoryOnly(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("GET", "/api/tags/tree?pack_id=1&category=general&offset=0&limit=99999", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	var page struct {
		Tags  []database.Tag `json:"tags"`
		Total int            `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &page)
	if len(page.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(page.Tags))
	}
	if page.Total != 2 {
		t.Errorf("Total = %d", page.Total)
	}
}

func TestFavorites_AddAndGet(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	body := `{"pack_id":1,"tag_name":"tag2","add":true}`
	req := httptest.NewRequest("POST", "/api/favorites", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/favorites?pack_id=1", nil)
	w = httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	var favs []database.Tag
	json.Unmarshal(w.Body.Bytes(), &favs)
	if len(favs) != 2 {
		t.Fatalf("got %d favorites, want 2", len(favs))
	}
}

func TestFavorites_Remove(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	body := `{"pack_id":1,"tag_name":"tag1","add":false}`
	req := httptest.NewRequest("POST", "/api/favorites", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/favorites?pack_id=1", nil)
	w = httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	var favs []database.Tag
	json.Unmarshal(w.Body.Bytes(), &favs)
	if len(favs) != 0 {
		t.Errorf("expected 0 favorites after remove, got %d", len(favs))
	}
}

func TestFavorites_InvalidBody(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("POST", "/api/favorites", bytes.NewBufferString(`invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPresets_Get(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/presets", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var presets []database.TagPreset
	json.Unmarshal(w.Body.Bytes(), &presets)
	if len(presets) == 0 {
		t.Fatal("expected seeded preset")
	}
	found := false
	for _, p := range presets {
		if p.Name == "Quality Only" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Quality Only preset not found in %v", presets)
	}
}

func TestPresets_Create(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	body := `{"name":"Custom","positive_tags":["a","b"],"negative_tags":["c"]}`
	req := httptest.NewRequest("POST", "/api/presets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var preset database.TagPreset
	json.Unmarshal(w.Body.Bytes(), &preset)
	if preset.Name != "Custom" {
		t.Errorf("Name = %q", preset.Name)
	}

	req = httptest.NewRequest("GET", "/api/presets", nil)
	w = httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	var presets []database.TagPreset
	json.Unmarshal(w.Body.Bytes(), &presets)

	found := false
	for _, p := range presets {
		if p.Name == "Custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom preset not found after create")
	}
}

func TestPrompts_Save(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	body := `{"name":"Test Prompt","positive_text":"tag1, tag2","negative_text":"bad","is_favorite":true}`
	req := httptest.NewRequest("POST", "/api/prompts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPrompts_GetHistory(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	env.repo.SavePrompt("", "t1", "n1", false, "")
	env.repo.SavePrompt("", "t2", "n2", false, "")

	req := httptest.NewRequest("GET", "/api/prompts?favorites=0", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var prompts []database.SavedPrompt
	json.Unmarshal(w.Body.Bytes(), &prompts)
	if len(prompts) != 2 {
		t.Errorf("got %d prompts, want 2", len(prompts))
	}
}

func TestPrompts_GetFavorites(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	env.repo.SavePrompt("fav", "t1", "n1", true, "")
	env.repo.SavePrompt("hist", "t2", "n2", false, "")

	req := httptest.NewRequest("GET", "/api/prompts?favorites=1", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	var prompts []database.SavedPrompt
	json.Unmarshal(w.Body.Bytes(), &prompts)
	if len(prompts) != 1 {
		t.Fatalf("got %d promps, want 1", len(prompts))
	}
	if prompts[0].Name != "fav" {
		t.Errorf("Name = %q", prompts[0].Name)
	}
}

func TestPrompts_Delete_Rejected(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("DELETE", "/api/prompts?id=1", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/prompts", nil)
	w = httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestNotFound(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/nonexistent", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestIndex(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestStatic(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/static/manifest.json", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestGetPackByIDHandler(t *testing.T) {
	env := setupTest(t)
	defer env.close()
	env.seed(t)

	req := httptest.NewRequest("GET", "/api/pack?id=1", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	var pack database.Pack
	json.Unmarshal(w.Body.Bytes(), &pack)
	if pack.ID != 1 {
		t.Errorf("ID = %d", pack.ID)
	}
	if pack.Name != "testpack" {
		t.Errorf("Name = %q", pack.Name)
	}
}

func TestGetPackByIDHandler_NotFound(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/pack?id=999", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetPackByIDHandler_NoID(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/pack", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestReadPackInfoFromReaderHandler(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	p, err := env.repo.CreatePack("infopack", filepath.Join(env.cfg.TagsPath, "infopack"))
	if err != nil {
		t.Fatal(err)
	}
	packDir := filepath.Join(env.cfg.TagsPath, "infopack")
	os.MkdirAll(packDir, 0755)
	info := `{"name":"infopack","categories":[{"name":"test","file":"test.txt"}]}`
	os.WriteFile(filepath.Join(packDir, "info.pack"), []byte(info), 0644)

	req := httptest.NewRequest("GET", "/api/pack/info?id="+strconv.Itoa(p.ID), nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Name string `json:"name"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Name != "infopack" {
		t.Errorf("Name = %q", result.Name)
	}
}

func TestReadPackInfoFromReaderHandler_NotFound(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/api/pack/info?id=999", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestReadPackInfoFromReaderHandler_NoInfoPack(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	p, err := env.repo.CreatePack("nopack", filepath.Join(env.cfg.TagsPath, "nopack"))
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(env.cfg.TagsPath, "nopack"), 0755)

	req := httptest.NewRequest("GET", "/api/pack/info?id="+strconv.Itoa(p.ID), nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPacksPage(t *testing.T) {
	env := setupTest(t)
	defer env.close()

	req := httptest.NewRequest("GET", "/settings", nil)
	w := httptest.NewRecorder()
	env.mux.ServeHTTP(w, req)

 	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
