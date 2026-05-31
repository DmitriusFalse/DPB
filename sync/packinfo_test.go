package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCategoryNameFromFile_CSV(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0_general_appearance.csv", "general"},
		{"1_artist_individual.csv", "artist"},
		{"3_copyright_anime.csv", "copyright"},
		{"4_character_canonical.csv", "character"},
		{"5_meta_quality.csv", "meta"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := categoryNameFromFile(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCategoryNameFromFile_TXT(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"armor.txt", "armor"},
		{"hair_ornament.txt", "hair_ornament"},
		{"blood_on.txt", "blood_on"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := categoryNameFromFile(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCategoryNameFromFile_NonStandard(t *testing.T) {
	got := categoryNameFromFile("readme.md")
	if got != "readme" {
		t.Errorf("got %q, want %q", got, "readme")
	}
}

func TestReadPackInfo_Valid(t *testing.T) {
	dir := t.TempDir()
	info := `{
		"name": "TestPack",
		"name_ru": "ТестПак",
		"description": "A test pack",
		"description_ru": "Тестовый набор",
		"version": "2.0",
		"author": "testuser",
		"icon": "📦",
		"categories": [
			{"name": "armor", "file": "armor.txt"},
			{"name": "weapons", "file": "weapons.txt"}
		]
	}`
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte(info), 0644)

	p, err := ReadPackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "TestPack" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.NameRu != "ТестПак" {
		t.Errorf("NameRu = %q", p.NameRu)
	}
	if p.Description != "A test pack" {
		t.Errorf("Description = %q", p.Description)
	}
	if p.DescriptionRu != "Тестовый набор" {
		t.Errorf("DescriptionRu = %q", p.DescriptionRu)
	}
	if p.Version != "2.0" {
		t.Errorf("Version = %q", p.Version)
	}
	if p.Author != "testuser" {
		t.Errorf("Author = %q", p.Author)
	}
	if p.Icon != "📦" {
		t.Errorf("Icon = %q", p.Icon)
	}
	if len(p.Categories) != 2 {
		t.Fatalf("got %d categories, want 2", len(p.Categories))
	}
	if p.Categories[0].Name != "armor" {
		t.Errorf("Category[0].Name = %q", p.Categories[0].Name)
	}
	if p.Categories[0].File != "armor.txt" {
		t.Errorf("Category[0].File = %q", p.Categories[0].File)
	}
}

func TestReadPackInfo_Minimal(t *testing.T) {
	dir := t.TempDir()
	info := `{"name": "Minimal", "categories": [{"name": "test", "file": "test.txt"}]}`
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte(info), 0644)

	p, err := ReadPackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Minimal" {
		t.Errorf("Name = %q", p.Name)
	}
}

func TestReadPackInfo_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadPackInfo(dir)
	if err == nil {
		t.Fatal("expected error for missing info.pack")
	}
	if !strings.Contains(err.Error(), "info.pack") {
		t.Errorf("error should mention info.pack, got: %v", err)
	}
}

func TestReadPackInfo_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte("{invalid"), 0644)

	_, err := ReadPackInfo(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadPackInfo_MissingName(t *testing.T) {
	dir := t.TempDir()
	info := `{"categories": [{"name": "test", "file": "test.txt"}]}`
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte(info), 0644)

	_, err := ReadPackInfo(dir)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestReadPackInfo_EmptyCategories(t *testing.T) {
	dir := t.TempDir()
	info := `{"name": "Empty", "categories": []}`
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte(info), 0644)

	_, err := ReadPackInfo(dir)
	if err == nil {
		t.Fatal("expected error for empty categories")
	}
}

func TestReadPackInfo_CategoryNoName(t *testing.T) {
	dir := t.TempDir()
	info := `{"name": "Bad", "categories": [{"file": "test.txt"}]}`
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte(info), 0644)

	_, err := ReadPackInfo(dir)
	if err == nil {
		t.Fatal("expected error for category without name")
	}
}

func TestReadPackInfo_CategoryNoFile(t *testing.T) {
	dir := t.TempDir()
	info := `{"name": "Bad", "categories": [{"name": "test"}]}`
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte(info), 0644)

	_, err := ReadPackInfo(dir)
	if err == nil {
		t.Fatal("expected error for category without file")
	}
}

func TestGeneratePackInfo_TXTFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "armor.txt"), []byte("tag1\ntag2\n"), 0644)
	os.WriteFile(filepath.Join(dir, "weapons.txt"), []byte("tag3\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("ignore"), 0644)

	info, err := GeneratePackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != filepath.Base(dir) {
		t.Errorf("Name = %q", info.Name)
	}
	if len(info.Categories) != 2 {
		t.Fatalf("got %d categories, want 2", len(info.Categories))
	}
	if info.Categories[0].Name != "armor" {
		t.Errorf("Category[0].Name = %q", info.Categories[0].Name)
	}
	if info.Categories[0].File != "armor.txt" {
		t.Errorf("Category[0].File = %q", info.Categories[0].File)
	}
}

func TestGeneratePackInfo_CSVFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "0_general_appearance.csv"), []byte("t,general,sub,\n"), 0644)
	os.WriteFile(filepath.Join(dir, "1_artist_individual.csv"), []byte("t,artist,sub,\n"), 0644)

	info, err := GeneratePackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Categories) != 2 {
		t.Fatalf("got %d categories, want 2", len(info.Categories))
	}
	if info.Categories[0].Name != "general" {
		t.Errorf("Category[0].Name = %q, want general", info.Categories[0].Name)
	}
	if info.Categories[1].Name != "artist" {
		t.Errorf("Category[1].Name = %q, want artist", info.Categories[1].Name)
	}
}

func TestGeneratePackInfo_Mixed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "armor.txt"), []byte("t1\n"), 0644)
	os.WriteFile(filepath.Join(dir, "0_general_test.csv"), []byte("t,general,sub,\n"), 0644)

	info, err := GeneratePackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Categories) != 2 {
		t.Fatalf("got %d categories", len(info.Categories))
	}
}

func TestGeneratePackInfo_NoTagFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("hello"), 0644)

	_, err := GeneratePackInfo(dir)
	if err == nil {
		t.Fatal("expected error for no tag files")
	}
}

func TestGeneratePackInfo_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := GeneratePackInfo(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestGeneratePackInfo_SkipsSubdirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "armor.txt"), []byte("t1\n"), 0644)

	info, err := GeneratePackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Categories) != 1 {
		t.Fatalf("got %d categories, want 1", len(info.Categories))
	}
}

func TestWritePackInfo(t *testing.T) {
	dir := t.TempDir()
	info := &PackInfo{
		Name:   "Test",
		Author: "me",
		Categories: []CategoryInfo{
			{Name: "cat1", File: "cat1.txt"},
		},
	}

	if err := WritePackInfo(dir, info); err != nil {
		t.Fatal(err)
	}

	read, err := ReadPackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if read.Name != "Test" {
		t.Errorf("Name = %q", read.Name)
	}
	if read.Author != "me" {
		t.Errorf("Author = %q", read.Author)
	}
	if len(read.Categories) != 1 {
		t.Fatalf("got %d categories", len(read.Categories))
	}
}

func TestSaveGeneratedPackInfo_ExistingInfoPack(t *testing.T) {
	dir := t.TempDir()
	info := `{"name": "Existing", "categories": [{"name": "test", "file": "test.txt"}]}`
	os.WriteFile(filepath.Join(dir, "info.pack"), []byte(info), 0644)

	p, err := SaveGeneratedPackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Existing" {
		t.Errorf("Name = %q", p.Name)
	}
}

func TestSaveGeneratedPackInfo_GeneratesAndWrites(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "armor.txt"), []byte("t1\n"), 0644)

	p, err := SaveGeneratedPackInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != filepath.Base(dir) {
		t.Errorf("Name = %q", p.Name)
	}
	if len(p.Categories) != 1 {
		t.Fatalf("got %d categories, want 1", len(p.Categories))
	}

	info, err := os.ReadFile(filepath.Join(dir, "info.pack"))
	if err != nil {
		t.Fatal("info.pack should have been written")
	}
	if !strings.Contains(string(info), "armor") {
		t.Error("info.pack should contain 'armor'")
	}
}

func TestSaveGeneratedPackInfo_NoFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := SaveGeneratedPackInfo(dir)
	if err == nil {
		t.Fatal("expected error for directory with no tag files")
	}
}

func TestReadPackInfoFromReader_Valid(t *testing.T) {
	r := strings.NewReader(`{"name":"Test","categories":[{"name":"cat1","file":"cat1.txt"}]}`)
	p, err := ReadPackInfoFromReader(r)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Test" {
		t.Errorf("Name = %q", p.Name)
	}
	if len(p.Categories) != 1 {
		t.Fatalf("got %d categories", len(p.Categories))
	}
	if p.Categories[0].Name != "cat1" {
		t.Errorf("Category[0].Name = %q", p.Categories[0].Name)
	}
}

func TestReadPackInfoFromReader_AllFields(t *testing.T) {
	r := strings.NewReader(`{
		"name": "FullPack",
		"name_ru": "ПолныйПак",
		"description": "English desc",
		"description_ru": "Русское описание",
		"version": "3.0",
		"author": "dev",
		"icon": "⭐",
		"categories": [{"name":"cat1","file":"f1.txt"},{"name":"cat2","name_ru":"кат2","file":"f2.txt"}]
	}`)
	p, err := ReadPackInfoFromReader(r)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "FullPack" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.NameRu != "ПолныйПак" {
		t.Errorf("NameRu = %q", p.NameRu)
	}
	if p.Description != "English desc" {
		t.Errorf("Description = %q", p.Description)
	}
	if p.DescriptionRu != "Русское описание" {
		t.Errorf("DescriptionRu = %q", p.DescriptionRu)
	}
	if p.Version != "3.0" {
		t.Errorf("Version = %q", p.Version)
	}
	if p.Author != "dev" {
		t.Errorf("Author = %q", p.Author)
	}
	if p.Icon != "⭐" {
		t.Errorf("Icon = %q", p.Icon)
	}
	if len(p.Categories) != 2 {
		t.Fatalf("got %d categories", len(p.Categories))
	}
	if p.Categories[1].NameRu != "кат2" {
		t.Errorf("Category[1].NameRu = %q", p.Categories[1].NameRu)
	}
}

func TestReadPackInfoFromReader_InvalidJSON(t *testing.T) {
	_, err := ReadPackInfoFromReader(strings.NewReader("{invalid"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadPackInfoFromReader_MissingName(t *testing.T) {
	_, err := ReadPackInfoFromReader(strings.NewReader(`{"categories":[{"name":"c","file":"f"}]}`))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestReadPackInfoFromReader_EmptyCategories(t *testing.T) {
	_, err := ReadPackInfoFromReader(strings.NewReader(`{"name":"X","categories":[]}`))
	if err == nil {
		t.Fatal("expected error for empty categories")
	}
}

func TestReadPackInfoFromReader_CategoryNoName(t *testing.T) {
	_, err := ReadPackInfoFromReader(strings.NewReader(`{"name":"X","categories":[{"file":"f"}]}`))
	if err == nil {
		t.Fatal("expected error for category without name")
	}
}

func TestReadPackInfoFromReader_CategoryNoFile(t *testing.T) {
	_, err := ReadPackInfoFromReader(strings.NewReader(`{"name":"X","categories":[{"name":"c"}]}`))
	if err == nil {
		t.Fatal("expected error for category without file")
	}
}

func TestReadPackInfoFromReader_CategoryWithNameRu(t *testing.T) {
	r := strings.NewReader(`{"name":"X","categories":[{"name":"c","name_ru":"рус","file":"f.txt"}]}`)
	p, err := ReadPackInfoFromReader(r)
	if err != nil {
		t.Fatal(err)
	}
	if p.Categories[0].NameRu != "рус" {
		t.Errorf("NameRu = %q", p.Categories[0].NameRu)
	}
}
