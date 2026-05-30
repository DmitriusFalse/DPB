package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFilename_Valid(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		cat      string
		sub      string
	}{
		{"0_general_appearance.csv", 0, "general", "appearance"},
		{"1_artist_individual_artists.csv", 1, "artist", "individual_artists"},
		{"3_copyright_anime_manga.csv", 3, "copyright", "anime_manga"},
		{"4_character_canonical.csv", 4, "character", "canonical"},
		{"5_meta_quality_resolution.csv", 5, "meta", "quality_resolution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, cat, sub, err := parseFilename(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if id != tt.id {
				t.Errorf("id = %d, want %d", id, tt.id)
			}
			if cat != tt.cat {
				t.Errorf("cat = %q, want %q", cat, tt.cat)
			}
			if sub != tt.sub {
				t.Errorf("sub = %q, want %q", sub, tt.sub)
			}
		})
	}
}

func TestParseFilename_NoExtension(t *testing.T) {
	_, _, _, err := parseFilename("0_general_appearance")
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseFilename_InvalidFormat(t *testing.T) {
	_, _, _, err := parseFilename("invalid.csv")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFilename_MissingSubcategory(t *testing.T) {
	_, _, _, err := parseFilename("0_general.csv")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFilename_BadCategoryID(t *testing.T) {
	_, _, _, err := parseFilename("abc_general_test.csv")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFilename_UnknownCategoryID(t *testing.T) {
	id, cat, sub, err := parseFilename("9_unknown_test.csv")
	if err != nil {
		t.Fatal(err)
	}
	if id != 9 {
		t.Errorf("id = %d, want 9", id)
	}
	if cat != "unknown" {
		t.Errorf("cat = %q", cat)
	}
	if sub != "test" {
		t.Errorf("sub = %q", sub)
	}
}

func TestParseFilename_MismatchExpectedName(t *testing.T) {
	_, _, _, err := parseFilename("0_artist_test.csv")
	if err == nil {
		t.Error("expected error for mismatched category name")
	}
}

func TestParseFilename_ValidNonStandardNames(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		cat      string
		sub      string
	}{
		{"1_artist_circles.csv", 1, "artist", "circles"},
		{"3_copyright_video_games.csv", 3, "copyright", "video_games"},
		{"5_meta_commentary.csv", 5, "meta", "commentary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, cat, sub, err := parseFilename(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if id != tt.id {
				t.Errorf("id = %d, want %d", id, tt.id)
			}
			if cat != tt.cat {
				t.Errorf("cat = %q, want %q", cat, tt.cat)
			}
			if sub != tt.sub {
				t.Errorf("sub = %q, want %q", sub, tt.sub)
			}
		})
	}
}

func TestScan_NoTagsDir(t *testing.T) {
	s := NewScanner()
	_, err := s.Scan(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
}

func TestScan_WithFiles(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "testpack")
	os.MkdirAll(packDir, 0755)

	csvContent := "tag1,general,appearance,\ntag2,copyright,anime_manga,\n"
	os.WriteFile(filepath.Join(packDir, "0_general_appearance.csv"), []byte(csvContent), 0644)

	s := NewScanner()
	packs, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(packs) != 1 {
		t.Fatalf("got %d packs, want 1", len(packs))
	}
	if packs[0].Name != "testpack" {
		t.Errorf("Name = %q, want testpack", packs[0].Name)
	}
	if len(packs[0].Files) != 1 {
		t.Fatalf("got %d files, want 1", len(packs[0].Files))
	}

	f := packs[0].Files[0]
	if f.CategoryName != "general" {
		t.Errorf("CategoryName = %q", f.CategoryName)
	}
	if f.SubcategoryName != "general" {
		t.Errorf("SubcategoryName = %q, want general (same as category name)", f.SubcategoryName)
	}
	if len(f.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(f.Tags))
	}
	if f.Tags[0].TagName != "tag1" {
		t.Errorf("TagName = %q", f.Tags[0].TagName)
	}
}

func TestScan_TXTFileAccepted(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "testpack")
	os.MkdirAll(packDir, 0755)
	os.WriteFile(filepath.Join(packDir, "armor.txt"), []byte("shoulder_armor\njapanese_armor\n"), 0644)

	s := NewScanner()
	packs, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(packs) != 1 {
		t.Fatalf("got %d packs", len(packs))
	}
	if len(packs[0].Files) != 1 {
		t.Fatalf("got %d files, want 1", len(packs[0].Files))
	}

	f := packs[0].Files[0]
	if f.CategoryName != "armor" {
		t.Errorf("CategoryName = %q, want armor", f.CategoryName)
	}
	if f.SubcategoryName != "armor" {
		t.Errorf("SubcategoryName = %q, want the same as category name", f.SubcategoryName)
	}
	if len(f.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(f.Tags))
	}
	if f.Tags[0].TagName != "shoulder_armor" {
		t.Errorf("TagName = %q", f.Tags[0].TagName)
	}
}

func TestScan_NonTagFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "testpack")
	os.MkdirAll(packDir, 0755)
	os.WriteFile(filepath.Join(packDir, "readme.md"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(packDir, "0_general_test.csv"), []byte("t,general,sub,\n"), 0644)

	s := NewScanner()
	packs, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(packs) != 1 {
		t.Fatalf("got %d packs", len(packs))
	}
	if len(packs[0].Files) != 1 {
		t.Fatalf("got %d files (non-tag files should be ignored)", len(packs[0].Files))
	}
}

func TestScan_MultiplePacks(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"pack_a", "pack_b"} {
		pd := filepath.Join(dir, name)
		os.MkdirAll(pd, 0755)
		os.WriteFile(filepath.Join(pd, "0_general_test.csv"), []byte("t,general,sub,\n"), 0644)
	}

	s := NewScanner()
	packs, err := s.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 2 {
		t.Fatalf("got %d packs, want 2", len(packs))
	}
}
