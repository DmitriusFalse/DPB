package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// ─── Packs ───

func (r *Repo) GetPacks() ([]Pack, error) {
	rows, err := r.db.Query(`SELECT id, name, path, description, description_ru, version, author, icon, name_ru, categories, created_at, updated_at FROM packs ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packs []Pack
	for rows.Next() {
		var p Pack
		if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.DescriptionRu, &p.Version, &p.Author, &p.Icon, &p.NameRu, &p.Categories, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		packs = append(packs, p)
	}
	return packs, rows.Err()
}

func (r *Repo) GetPackByID(id int) (*Pack, error) {
	p := &Pack{}
	err := r.db.QueryRow(`SELECT id, name, path, description, description_ru, version, author, icon, name_ru, categories, created_at, updated_at FROM packs WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.DescriptionRu, &p.Version, &p.Author, &p.Icon, &p.NameRu, &p.Categories, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repo) GetPackByName(name string) (*Pack, error) {
	p := &Pack{}
	err := r.db.QueryRow(`SELECT id, name, path, description, description_ru, version, author, icon, name_ru, categories, created_at, updated_at FROM packs WHERE name = ?`, name).
		Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.DescriptionRu, &p.Version, &p.Author, &p.Icon, &p.NameRu, &p.Categories, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repo) CreatePack(name, path string) (*Pack, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Exec(`INSERT INTO packs (name, path, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		name, path, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Pack{ID: int(id), Name: name, Path: path, CreatedAt: now, UpdatedAt: now}, nil
}

func (r *Repo) UpdatePackMeta(id int, desc, descRu, version, author, icon, nameRu string, categories []byte) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`
		UPDATE packs SET description=?, description_ru=?, version=?, author=?, icon=?, name_ru=?, categories=?, updated_at=?
		WHERE id=?
	`, desc, descRu, version, author, icon, nameRu, string(categories), now, id)
	return err
}

func (r *Repo) DeletePack(id int) error {
	_, err := r.db.Exec(`DELETE FROM packs WHERE id = ?`, id)
	return err
}

// ─── Files ───

func (r *Repo) InsertFile(packID int, fileName string, categoryID int, categoryName, subcategoryName, hash string) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Exec(`
		INSERT INTO files (pack_id, file_name, category_id, category_name, subcategory_name, file_hash, last_synced)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, packID, fileName, categoryID, categoryName, subcategoryName, hash, now)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (r *Repo) UpdateFile(id, packID int, fileName string, categoryID int, categoryName, subcategoryName, hash string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`
		UPDATE files SET pack_id=?, file_name=?, category_id=?, category_name=?, subcategory_name=?, file_hash=?, last_synced=?
		WHERE id=?
	`, packID, fileName, categoryID, categoryName, subcategoryName, hash, now, id)
	return err
}

func (r *Repo) UpsertFile(packID int, fileName string, categoryID int, categoryName, subcategoryName, hash string) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`
		INSERT INTO files (pack_id, file_name, category_id, category_name, subcategory_name, file_hash, last_synced)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(pack_id, file_name) DO UPDATE SET
			category_id = excluded.category_id,
			category_name = excluded.category_name,
			subcategory_name = excluded.subcategory_name,
			file_hash = excluded.file_hash,
			last_synced = excluded.last_synced
	`, packID, fileName, categoryID, categoryName, subcategoryName, hash, now)
	if err != nil {
		return 0, err
	}
	var fid int
	err = r.db.QueryRow(`SELECT id FROM files WHERE pack_id = ? AND file_name = ?`, packID, fileName).Scan(&fid)
	return fid, err
}

func (r *Repo) DeleteFilesByPack(packID int) error {
	_, err := r.db.Exec(`DELETE FROM files WHERE pack_id = ?`, packID)
	return err
}

func (r *Repo) DeleteFile(id int) error {
	_, err := r.db.Exec(`DELETE FROM files WHERE id = ?`, id)
	return err
}

func (r *Repo) GetFilesByPack(packID int) ([]File, error) {
	rows, err := r.db.Query(`
		SELECT id, pack_id, file_name, category_id, category_name, subcategory_name, file_hash, last_synced
		FROM files WHERE pack_id = ?
	`, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.PackID, &f.FileName, &f.CategoryID, &f.CategoryName, &f.SubcategoryName, &f.FileHash, &f.LastSynced); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *Repo) GetFileByPackAndName(packID int, fileName string) (*File, error) {
	f := &File{}
	err := r.db.QueryRow(`
		SELECT id, pack_id, file_name, category_id, category_name, subcategory_name, file_hash, last_synced
		FROM files WHERE pack_id = ? AND file_name = ?
	`, packID, fileName).
		Scan(&f.ID, &f.PackID, &f.FileName, &f.CategoryID, &f.CategoryName, &f.SubcategoryName, &f.FileHash, &f.LastSynced)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ─── Tags ───

func (r *Repo) InsertTags(tags []Tag) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO tags (file_id, pack_id, tag_name, category_name, subcategory_name, aliases)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range tags {
		if _, err := stmt.Exec(t.FileID, t.PackID, t.TagName, t.CategoryName, t.SubcategoryName, t.Aliases); err != nil {
			return fmt.Errorf("insert tag %s: %w", t.TagName, err)
		}
	}

	return tx.Commit()
}

func (r *Repo) DeleteTagsByFile(fileID int) error {
	_, err := r.db.Exec(`DELETE FROM tags WHERE file_id = ?`, fileID)
	return err
}

func (r *Repo) GetTagsByFile(fileID int) ([]Tag, error) {
	rows, err := r.db.Query(`
		SELECT id, file_id, pack_id, tag_name, category_name, subcategory_name, aliases
		FROM tags WHERE file_id = ?
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.FileID, &t.PackID, &t.TagName, &t.CategoryName, &t.SubcategoryName, &t.Aliases); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *Repo) DeleteTag(fileID int, tagName string) error {
	_, err := r.db.Exec(`DELETE FROM tags WHERE file_id = ? AND tag_name = ?`, fileID, tagName)
	return err
}

// ─── Search / Tree ───

func (r *Repo) GetTagCategory(packID int, tagName string) (string, error) {
	var category string
	err := r.db.QueryRow(`SELECT subcategory_name FROM tags WHERE pack_id = ? AND tag_name = ? LIMIT 1`, packID, tagName).Scan(&category)
	if err != nil {
		return "", err
	}
	return category, nil
}

func (r *Repo) SearchTags(packID int, query string, limit int) ([]Tag, error) {
	if limit <= 0 {
		limit = 50
	}

	q := strings.ReplaceAll(query, "%", "\\%")
	q = strings.ReplaceAll(q, "_", "\\_")

	rows, err := r.db.Query(`
		SELECT id, file_id, pack_id, tag_name, category_name, subcategory_name, aliases
		FROM tags
		WHERE pack_id = ? AND (tag_name LIKE ? OR aliases LIKE ?)
		ORDER BY tag_name
		LIMIT ?
	`, packID, "%"+q+"%", "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.FileID, &t.PackID, &t.TagName, &t.CategoryName, &t.SubcategoryName, &t.Aliases); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *Repo) GetCategoryTree(packID int) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT category_name FROM tags WHERE pack_id = ? ORDER BY category_name
	`, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (r *Repo) GetCategoryCounts(packID int) (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT category_name, COUNT(*) FROM tags
		WHERE pack_id = ?
		GROUP BY category_name
		ORDER BY category_name
	`, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var cat string
		var count int
		if err := rows.Scan(&cat, &count); err != nil {
			return nil, err
		}
		counts[cat] = count
	}
	return counts, rows.Err()
}

func (r *Repo) GetTagsByCategory(packID int, categoryName string, offset, limit int) (tags []Tag, total int, err error) {
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM tags
		WHERE pack_id = ? AND category_name = ?
	`, packID, categoryName).Scan(&total)
	if err != nil {
		return
	}

	rows, err := r.db.Query(`
		SELECT id, file_id, pack_id, tag_name, category_name, subcategory_name, aliases
		FROM tags
		WHERE pack_id = ? AND category_name = ?
		ORDER BY tag_name
		LIMIT ? OFFSET ?
	`, packID, categoryName, limit, offset)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.FileID, &t.PackID, &t.TagName, &t.CategoryName, &t.SubcategoryName, &t.Aliases); err != nil {
			return nil, 0, err
		}
		tags = append(tags, t)
	}
	err = rows.Err()
	return
}

// ─── Favorite Tags ───

func (r *Repo) GetFavorites(packID int) ([]Tag, error) {
	rows, err := r.db.Query(`
		SELECT ft.id, 0, ft.pack_id, ft.tag_name, '', '', ''
		FROM favorite_tags ft
		WHERE ft.pack_id = ?
		ORDER BY ft.tag_name
	`, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.FileID, &t.PackID, &t.TagName, &t.CategoryName, &t.SubcategoryName, &t.Aliases); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *Repo) AddFavorite(packID int, tagName string) error {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO favorite_tags (pack_id, tag_name) VALUES (?, ?)`,
		packID, tagName)
	return err
}

func (r *Repo) RemoveFavorite(packID int, tagName string) error {
	_, err := r.db.Exec(`DELETE FROM favorite_tags WHERE pack_id = ? AND tag_name = ?`,
		packID, tagName)
	return err
}

func (r *Repo) IsFavorite(packID int, tagName string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM favorite_tags WHERE pack_id = ? AND tag_name = ?`,
		packID, tagName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ─── Saved Prompts ───

func (r *Repo) SavePrompt(name, positiveText, negativeText string, isFavorite bool, genData string) (*SavedPrompt, error) {
	fav := 0
	if isFavorite {
		fav = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Exec(`
		INSERT INTO saved_prompts (name, positive_text, negative_text, is_favorite, created_at, gen_data)
		VALUES (?, ?, ?, ?, ?, ?)
	`, name, positiveText, negativeText, fav, now, genData)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &SavedPrompt{
		ID:           int(id),
		Name:         name,
		PositiveText: positiveText,
		NegativeText: negativeText,
		IsFavorite:   isFavorite,
		CreatedAt:    now,
		GenData:      genData,
	}, nil
}

func (r *Repo) GetHistory(limit int) ([]SavedPrompt, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(`
		SELECT id, name, positive_text, negative_text, is_favorite, created_at, gen_data
		FROM saved_prompts
		WHERE is_favorite = 0
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []SavedPrompt
	for rows.Next() {
		var p SavedPrompt
		var fav int
		if err := rows.Scan(&p.ID, &p.Name, &p.PositiveText, &p.NegativeText, &fav, &p.CreatedAt, &p.GenData); err != nil {
			return nil, err
		}
		p.IsFavorite = fav == 1
		prompts = append(prompts, p)
	}
	return prompts, rows.Err()
}

func (r *Repo) GetFavoritesPrompts() ([]SavedPrompt, error) {
	rows, err := r.db.Query(`
		SELECT id, name, positive_text, negative_text, is_favorite, created_at, gen_data
		FROM saved_prompts
		WHERE is_favorite = 1
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []SavedPrompt
	for rows.Next() {
		var p SavedPrompt
		var fav int
		if err := rows.Scan(&p.ID, &p.Name, &p.PositiveText, &p.NegativeText, &fav, &p.CreatedAt, &p.GenData); err != nil {
			return nil, err
		}
		p.IsFavorite = fav == 1
		prompts = append(prompts, p)
	}
	return prompts, rows.Err()
}

func (r *Repo) TrimHistory(max int) error {
	if max <= 0 {
		max = 50
	}
	_, err := r.db.Exec(`
		DELETE FROM saved_prompts
		WHERE id IN (
			SELECT id FROM saved_prompts
			WHERE is_favorite = 0
			ORDER BY created_at DESC
			LIMIT -1 OFFSET ?
		)
	`, max)
	return err
}

func (r *Repo) DeletePrompt(id int) error {
	_, err := r.db.Exec(`DELETE FROM saved_prompts WHERE id = ?`, id)
	return err
}

// ─── Tag Presets ───

func (r *Repo) GetPresets() ([]TagPreset, error) {
	rows, err := r.db.Query(`SELECT id, name, positive_tags, negative_tags FROM tag_presets ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []TagPreset
	for rows.Next() {
		var p TagPreset
		if err := rows.Scan(&p.ID, &p.Name, &p.PositiveTags, &p.NegativeTags); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (r *Repo) SavePreset(name string, positiveTags, negativeTags []string) (*TagPreset, error) {
	posJSON, _ := json.Marshal(positiveTags)
	negJSON, _ := json.Marshal(negativeTags)

	res, err := r.db.Exec(`
		INSERT INTO tag_presets (name, positive_tags, negative_tags)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			positive_tags = excluded.positive_tags,
			negative_tags = excluded.negative_tags
	`, name, string(posJSON), string(negJSON))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &TagPreset{
		ID:           int(id),
		Name:         name,
		PositiveTags: string(posJSON),
		NegativeTags: string(negJSON),
	}, nil
}

func (r *Repo) DeletePreset(id int) error {
	_, err := r.db.Exec(`DELETE FROM tag_presets WHERE id = ?`, id)
	return err
}

// ─── Seed ───

func (r *Repo) SeedDefaultPreset() error {
	var count int
	r.db.QueryRow(`SELECT COUNT(*) FROM tag_presets WHERE name = 'Quality Only'`).Scan(&count)
	if count > 0 {
		return nil
	}

	positive := []string{"score_9", "score_8_up", "score_7_up", "score_6_up", "score_5_up", "BREAK", "best_quality", "high_quality", "high_res", "masterpiece", "detailed"}
	negative := []string{"low_quality", "worst_quality", "bad_art"}

	_, err := r.SavePreset("Quality Only", positive, negative)
	return err
}
