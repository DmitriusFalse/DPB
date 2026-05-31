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
	Name          string
	Path          string
	Description   string
	Version       string
	Author        string
	Icon          string
	NameRu        string
	DescriptionRu string
	Categories    []CategoryInfo
	Files         []FileResult
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
	info, err := SaveGeneratedPackInfo(path)
	if err != nil {
		return PackResult{}, fmt.Errorf("read or generate info.pack: %w", err)
	}

	pack := PackResult{
		Name:          info.Name,
		Path:          path,
		Description:   info.Description,
		DescriptionRu: info.DescriptionRu,
		Version:       info.Version,
		Author:        info.Author,
		Icon:          info.Icon,
		NameRu:        info.NameRu,
		Categories:    info.Categories,
	}

	for _, cat := range info.Categories {
		filePath := filepath.Join(path, cat.File)
		if _, err := os.Stat(filePath); err != nil {
			return PackResult{}, fmt.Errorf("category %q file not found: %s", cat.Name, cat.File)
		}
		fr, err := s.scanFile(filePath, cat.File, cat.Name)
		if err != nil {
			return PackResult{}, fmt.Errorf("scan %s: %w", cat.File, err)
		}
		pack.Files = append(pack.Files, fr)
	}

	return pack, nil
}

func (s *Scanner) scanFile(filePath, fileName, catName string) (FileResult, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".csv":
		return s.scanCSVFile(filePath, fileName, catName)
	case ".txt":
		return s.scanTXTFile(filePath, fileName, catName)
	default:
		return FileResult{}, fmt.Errorf("unsupported file type: %s", fileName)
	}
}

func (s *Scanner) scanCSVFile(filePath, fileName, catName string) (FileResult, error) {
	catID, _, _, err := parseFilename(fileName)
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

func (s *Scanner) scanTXTFile(filePath, fileName, catName string) (FileResult, error) {
	hash, err := FileHash(filePath)
	if err != nil {
		return FileResult{}, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return FileResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	tags, err := ParseTXT(f, catName, catName)
	if err != nil {
		return FileResult{}, fmt.Errorf("parse txt: %w", err)
	}

	return FileResult{
		FileName:        fileName,
		CategoryID:      0,
		CategoryName:    catName,
		SubcategoryName: catName,
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
