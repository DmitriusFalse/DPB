package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS packs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		path TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		description_ru TEXT NOT NULL DEFAULT '',
		version TEXT NOT NULL DEFAULT '',
		author TEXT NOT NULL DEFAULT '',
		icon TEXT NOT NULL DEFAULT '',
		name_ru TEXT NOT NULL DEFAULT '',
		categories TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pack_id INTEGER NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
		file_name TEXT NOT NULL,
		category_id INTEGER NOT NULL,
		category_name TEXT NOT NULL,
		subcategory_name TEXT NOT NULL,
		file_hash TEXT NOT NULL,
		last_synced DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(pack_id, file_name)
	)`,
	`CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		pack_id INTEGER NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
		tag_name TEXT NOT NULL,
		category_name TEXT NOT NULL,
		subcategory_name TEXT NOT NULL,
		aliases TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS favorite_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pack_id INTEGER NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
		tag_name TEXT NOT NULL,
		UNIQUE(pack_id, tag_name)
	)`,
	`CREATE TABLE IF NOT EXISTS saved_prompts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL DEFAULT '',
		positive_text TEXT NOT NULL DEFAULT '',
		negative_text TEXT NOT NULL DEFAULT '',
		is_favorite INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS tag_presets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		positive_tags TEXT NOT NULL DEFAULT '[]',
		negative_tags TEXT NOT NULL DEFAULT '[]'
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tags_pack ON tags(pack_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(tag_name)`,
	`CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category_name, subcategory_name)`,
	`CREATE INDEX IF NOT EXISTS idx_tags_file ON tags(file_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tags_file_tag ON tags(file_id, tag_name)`,
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull sql.NullInt64
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func addColumnIfNotExists(db *sql.DB, table, column, colDef string) error {
	exists, err := columnExists(db, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, colDef))
	return err
}

func Init(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return nil, fmt.Errorf("migration: %w", err)
		}
	}

	if err := addColumnIfNotExists(db, "saved_prompts", "gen_data", "gen_data TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, fmt.Errorf("migration: %w", err)
	}

	repo := NewRepo(db)
	if err := repo.SeedDefaultPreset(); err != nil {
		return nil, fmt.Errorf("seed presets: %w", err)
	}

	return db, nil
}
