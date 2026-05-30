package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

type PackResult struct {
	Name  string
	Path  string
	Files []FileResult
}

type FileResult struct {
	FileName        string
	CategoryID      int
	CategoryName    string
	SubcategoryName string
	Hash            string
	Tags            []TagResult
}

type TagResult struct {
	TagName         string
	CategoryName    string
	SubcategoryName string
	Aliases         string
}

func (s *Scanner) Scan(tagsPath string) ([]PackResult, error) {
	entries, err := os.ReadDir(tagsPath)
	if err != nil {
		return nil, fmt.Errorf("read tags dir: %w", err)
	}

	var packs []PackResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packPath := filepath.Join(tagsPath, entry.Name())
		pack, err := s.scanPack(entry.Name(), packPath)
		if err != nil {
			return nil, fmt.Errorf("scan pack %s: %w", entry.Name(), err)
		}
		packs = append(packs, pack)
	}

	return packs, nil
}

func (s *Scanner) scanPack(name, path string) (PackResult, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return PackResult{}, err
	}

	pack := PackResult{
		Name: name,
		Path: path,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".csv" && ext != ".txt" {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		fr, err := s.scanFile(filePath, entry.Name())
		if err != nil {
			return PackResult{}, fmt.Errorf("scan file %s: %w", entry.Name(), err)
		}
		pack.Files = append(pack.Files, fr)
	}

	return pack, nil
}

func (s *Scanner) scanFile(filePath, fileName string) (FileResult, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".csv":
		return s.scanCSVFile(filePath, fileName)
	case ".txt":
		return s.scanTXTFile(filePath, fileName)
	default:
		return FileResult{}, fmt.Errorf("unsupported file type: %s", fileName)
	}
}

func (s *Scanner) scanCSVFile(filePath, fileName string) (FileResult, error) {
	catID, catName, _, err := parseFilename(fileName)
	if err != nil {
		return FileResult{}, err
	}

	hash, err := FileHash(filePath)
	if err != nil {
		return FileResult{}, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return FileResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	tags, err := ParseCSV(f)
	if err != nil {
		return FileResult{}, fmt.Errorf("parse csv: %w", err)
	}

	// Override subcategory with category name (no subcategories)
	for i := range tags {
		tags[i].SubcategoryName = catName
	}

	return FileResult{
		FileName:        fileName,
		CategoryID:      catID,
		CategoryName:    catName,
		SubcategoryName: catName,
		Hash:            hash,
		Tags:            tags,
	}, nil
}

func (s *Scanner) scanTXTFile(filePath, fileName string) (FileResult, error) {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	catName := strings.ReplaceAll(base, "_", " ")
	subName := catName

	hash, err := FileHash(filePath)
	if err != nil {
		return FileResult{}, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return FileResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	tags, err := ParseTXT(f, catName, subName)
	if err != nil {
		return FileResult{}, fmt.Errorf("parse txt: %w", err)
	}

	return FileResult{
		FileName:        fileName,
		CategoryID:      0,
		CategoryName:    catName,
		SubcategoryName: subName,
		Hash:            hash,
		Tags:            tags,
	}, nil
}

var categoryNameMap = map[int]string{
	0: "general",
	1: "artist",
	3: "copyright",
	4: "character",
	5: "meta",
}

func parseFilename(name string) (int, string, string, error) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.SplitN(base, "_", 2)
	if len(parts) < 2 {
		return 0, "", "", fmt.Errorf("invalid filename format: %s", name)
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid category ID in filename %s: %w", name, err)
	}

	rest := parts[1]
	restParts := strings.SplitN(rest, "_", 2)
	if len(restParts) < 2 {
		return 0, "", "", fmt.Errorf("invalid filename format (missing subcategory): %s", name)
	}

	catName := restParts[0]
	subName := restParts[1]

	if expected, ok := categoryNameMap[id]; ok && expected != catName {
		return 0, "", "", fmt.Errorf("filename %s: category ID %d does not match name %s (expected %s)", name, id, catName, expected)
	}

	return id, catName, subName, nil
}
