package sync

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CategoryInfo struct {
	Name    string `json:"name"`
	NameRu  string `json:"name_ru,omitempty"`
	File    string `json:"file"`
	BlockID int    `json:"block_id,omitempty"`
}

type PackInfo struct {
	Name        string         `json:"name"`
	NameRu      string         `json:"name_ru,omitempty"`
	Description string         `json:"description,omitempty"`
	DescriptionRu string       `json:"description_ru,omitempty"`
	Version     string         `json:"version,omitempty"`
	Author      string         `json:"author,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	Categories  []CategoryInfo `json:"categories"`
}

func ReadPackInfoFromReader(r io.Reader) (*PackInfo, error) {
	var info PackInfo
	if err := json.NewDecoder(r).Decode(&info); err != nil {
		return nil, fmt.Errorf("parse info.pack: %w", err)
	}
	if info.Name == "" {
		return nil, fmt.Errorf("info.pack: name is required")
	}
	if len(info.Categories) == 0 {
		return nil, fmt.Errorf("info.pack: no categories")
	}
	for _, c := range info.Categories {
		if c.Name == "" {
			return nil, fmt.Errorf("info.pack: category name is required")
		}
		if c.File == "" {
			return nil, fmt.Errorf("info.pack: category %q has no file", c.Name)
		}
	}
	return &info, nil
}

func ReadPackInfo(packPath string) (*PackInfo, error) {
	f, err := os.Open(filepath.Join(packPath, "info.pack"))
	if err != nil {
		return nil, fmt.Errorf("read info.pack: %w", err)
	}
	defer f.Close()
	return ReadPackInfoFromReader(f)
}

func categoryNameFromFile(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	base := strings.TrimSuffix(name, ext)
	if ext == ".csv" {
		parts := strings.SplitN(base, "_", 3)
		if len(parts) == 3 {
			return parts[1]
		}
	}
	return base
}

func GeneratePackInfo(packPath string) (*PackInfo, error) {
	entries, err := os.ReadDir(packPath)
	if err != nil {
		return nil, err
	}
	info := &PackInfo{
		Name: filepath.Base(packPath),
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".csv" && ext != ".txt" {
			continue
		}
		catName := categoryNameFromFile(e.Name())
		info.Categories = append(info.Categories, CategoryInfo{
			Name: catName,
			File: e.Name(),
		})
	}
	if len(info.Categories) == 0 {
		return nil, fmt.Errorf("no tag files found in %s", packPath)
	}
	return info, nil
}

func WritePackInfo(packPath string, info *PackInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal info.pack: %w", err)
	}
	path := filepath.Join(packPath, "info.pack")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write info.pack: %w", err)
	}
	return nil
}

func SaveGeneratedPackInfo(packPath string) (*PackInfo, error) {
	infoPath := filepath.Join(packPath, "info.pack")
	if _, err := os.Stat(infoPath); err != nil {
		// info.pack does not exist — generate and write
		info, genErr := GeneratePackInfo(packPath)
		if genErr != nil {
			return nil, genErr
		}
		if err := WritePackInfo(packPath, info); err != nil {
			return nil, err
		}
		return info, nil
	}
	// info.pack exists — read it
	info, err := ReadPackInfo(packPath)
	if err != nil {
		// Corrupted or incompatible — use generated fallback in memory only, do NOT overwrite the file
		fallback, genErr := GeneratePackInfo(packPath)
		if genErr != nil {
			return nil, genErr
		}
		return fallback, nil
	}
	return info, nil
}
