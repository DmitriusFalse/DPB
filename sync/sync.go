package sync

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"danbooru-prompt-builder/database"
	"danbooru-prompt-builder/logger"
)

type Service struct {
	repo    *database.Repo
	scanner *Scanner
}

func NewService(db *sql.DB) *Service {
	return &Service{
		repo:    database.NewRepo(db),
		scanner: NewScanner(),
	}
}

func (s *Service) Sync(tagsPath string) error {
	logger.Debug("Syncing tags from %s", tagsPath)

	packs, err := s.scanner.Scan(tagsPath)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	existingPacks, err := s.repo.GetPacks()
	if err != nil {
		return fmt.Errorf("get existing packs: %w", err)
	}

	existingMap := make(map[string]*database.Pack)
	for i := range existingPacks {
		existingMap[existingPacks[i].Name] = &existingPacks[i]
	}

	for _, pack := range packs {
		dbPack, ok := existingMap[pack.Name]
		if !ok {
			dbPack, err = s.repo.CreatePack(pack.Name, pack.Path)
			if err != nil {
				return fmt.Errorf("create pack %s: %w", pack.Name, err)
			}
			logger.Debug("  New pack: %s", pack.Name)
		}
		delete(existingMap, pack.Name)

		if err := s.syncFiles(dbPack, pack.Files); err != nil {
			return fmt.Errorf("sync files for %s: %w", pack.Name, err)
		}

		categoriesJSON, err := json.Marshal(pack.Categories)
		if err != nil {
			return fmt.Errorf("marshal categories for %s: %w", pack.Name, err)
		}
		if err := s.repo.UpdatePackMeta(dbPack.ID,
			pack.Description, pack.DescriptionRu,
			pack.Version, pack.Author, pack.Icon, pack.NameRu,
			categoriesJSON,
		); err != nil {
			return fmt.Errorf("update pack meta %s: %w", pack.Name, err)
		}
	}

	for _, stalePack := range existingMap {
		logger.Debug("  Removing stale pack: %s", stalePack.Name)
		if err := s.repo.DeletePack(stalePack.ID); err != nil {
			return fmt.Errorf("delete pack %s: %w", stalePack.Name, err)
		}
	}

	logger.Debug("Sync complete")
	return nil
}

func (s *Service) syncFiles(pack *database.Pack, files []FileResult) error {
	// Build set of scanned file names
	scanned := make(map[string]bool)
	for _, file := range files {
		scanned[file.FileName] = true
	}

	// Remove DB files that no longer exist on disk
	dbFiles, err := s.repo.GetFilesByPack(pack.ID)
	if err != nil {
		return err
	}
	for _, dbFile := range dbFiles {
		if !scanned[dbFile.FileName] {
			if err := s.repo.DeleteTagsByFile(dbFile.ID); err != nil {
				return err
			}
			if err := s.repo.DeleteFile(dbFile.ID); err != nil {
				return err
			}
			logger.Debug("    Removed stale file: %s", dbFile.FileName)
		}
	}

	for _, file := range files {
		existing, err := s.repo.GetFileByPackAndName(pack.ID, file.FileName)
		if err != nil {
			return err
		}

		if existing != nil && existing.FileHash == file.Hash {
			logger.Debug("    Unchanged %s", file.FileName)
			continue
		}

		var (
			fileID int
			del    int
			ins    int
		)

		if existing != nil {
			fileID = existing.ID

			oldTags, err := s.repo.GetTagsByFile(existing.ID)
			if err != nil {
				return err
			}
			oldMap := make(map[string]database.Tag, len(oldTags))
			for _, t := range oldTags {
				oldMap[t.TagName] = t
			}

			var inserts []database.Tag
			for _, t := range file.Tags {
				if _, exists := oldMap[t.TagName]; exists {
					delete(oldMap, t.TagName)
				} else {
					inserts = append(inserts, database.Tag{
						FileID: fileID, PackID: pack.ID,
						TagName: t.TagName, CategoryName: t.CategoryName,
						SubcategoryName: t.SubcategoryName, Aliases: t.Aliases,
					})
				}
			}
			if len(inserts) > 0 {
				if err := s.repo.InsertTags(inserts); err != nil {
					return fmt.Errorf("insert new tags: %w", err)
				}
				ins = len(inserts)
			}
			for _, t := range oldMap {
				if err := s.repo.DeleteTag(fileID, t.TagName); err != nil {
					return fmt.Errorf("delete tag %s: %w", t.TagName, err)
				}
				del++
			}

			if err := s.repo.UpdateFile(existing.ID, pack.ID, file.FileName,
				file.CategoryID, file.CategoryName, file.SubcategoryName, file.Hash); err != nil {
				return err
			}
		} else {
			fileID, err = s.repo.InsertFile(
				pack.ID, file.FileName, file.CategoryID,
				file.CategoryName, file.SubcategoryName, file.Hash,
			)
			if err != nil {
				return err
			}

			var tags []database.Tag
			for _, t := range file.Tags {
				tags = append(tags, database.Tag{
					FileID: fileID, PackID: pack.ID,
					TagName: t.TagName, CategoryName: t.CategoryName,
					SubcategoryName: t.SubcategoryName, Aliases: t.Aliases,
				})
			}
			if err := s.repo.InsertTags(tags); err != nil {
				return fmt.Errorf("insert tags for %s: %w", file.FileName, err)
			}
			ins = len(tags)
		}

		logger.Debug("    Synced %s (+%d -%d)", file.FileName, ins, del)
	}

	return nil
}
