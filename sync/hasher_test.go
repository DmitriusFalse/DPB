package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileHash_KnownContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello world\n"
	os.WriteFile(path, []byte(content), 0644)

	hash, err := FileHash(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

func TestFileHash_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0644)

	hash, err := FileHash(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

func TestFileHash_FileNotFound(t *testing.T) {
	_, err := FileHash("/nonexistent/file.csv")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileHash_Consistency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	os.WriteFile(path, []byte("a,b,c,d\n"), 0644)

	h1, _ := FileHash(path)
	h2, _ := FileHash(path)
	if h1 != h2 {
		t.Errorf("hash inconsistent: %q vs %q", h1, h2)
	}
}

func TestFileHash_ChangedContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")

	os.WriteFile(path, []byte("v1\n"), 0644)
	h1, _ := FileHash(path)

	os.WriteFile(path, []byte("v2\n"), 0644)
	h2, _ := FileHash(path)

	if h1 == h2 {
		t.Error("expected different hash after content change")
	}
}
