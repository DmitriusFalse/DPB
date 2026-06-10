# База данных (SQLite)

## Подключение

- Драйвер: `github.com/mattn/go-sqlite3`
- Режим: WAL (`_journal_mode=WAL`) + foreign keys (`_foreign_keys=on`)
- Путь по умолчанию: `./data.db` (относительно exe)

## Таблицы

### packs

| Колонка | Тип | Ограничения | Описание |
|---------|-----|-------------|----------|
| id | INTEGER | PK AUTOINCREMENT | |
| name | TEXT | UNIQUE NOT NULL | Название пака |
| path | TEXT | NOT NULL | Путь к файлам на диске |
| description | TEXT | | Описание (EN) |
| description_ru | TEXT | | Описание (RU) |
| version | TEXT | | Версия пака |
| author | TEXT | | Автор |
| icon | TEXT | | Иконка |
| name_ru | TEXT | | Название (RU) |
| categories | TEXT | | JSON с категориями |
| created_at | TEXT | NOT NULL | UTC RFC3339 |
| updated_at | TEXT | NOT NULL | UTC RFC3339 |

### files

| Колонка | Тип | Ограничения | Описание |
|---------|-----|-------------|----------|
| id | INTEGER | PK AUTOINCREMENT | |
| pack_id | INTEGER | FK → packs(id) ON DELETE CASCADE | |
| file_name | TEXT | NOT NULL | Имя CSV/TXT файла |
| category_id | INTEGER | | ID категории Danbooru |
| category_name | TEXT | | Название категории |
| subcategory_name | TEXT | | Название подкатегории |
| file_hash | TEXT | | SHA-256 хеш |
| last_synced | TEXT | | UTC RFC3339 |

**UNIQUE(pack_id, file_name)**

### tags

| Колонка | Тип | Ограничения | Описание |
|---------|-----|-------------|----------|
| id | INTEGER | PK AUTOINCREMENT | |
| file_id | INTEGER | FK → files(id) ON DELETE CASCADE | |
| pack_id | INTEGER | FK → packs(id) ON DELETE CASCADE | |
| tag_name | TEXT | NOT NULL | Имя тега |
| category_name | TEXT | | Категория |
| subcategory_name | TEXT | | Подкатегория |
| aliases | TEXT | | Псевдонимы (через запятую) |

**Индексы:** `idx_tags_pack`, `idx_tags_name`, `idx_tags_category`, `idx_tags_file`, `idx_tags_file_tag`

### favorite_tags

| Колонка | Тип | Ограничения | Описание |
|---------|-----|-------------|----------|
| id | INTEGER | PK AUTOINCREMENT | |
| pack_id | INTEGER | FK → packs(id) ON DELETE CASCADE | |
| tag_name | TEXT | NOT NULL | |

**UNIQUE(pack_id, tag_name)**

### saved_prompts

| Колонка | Тип | Ограничения | Описание |
|---------|-----|-------------|----------|
| id | INTEGER | PK AUTOINCREMENT | |
| name | TEXT | | Название |
| positive_text | TEXT | | Позитивный промпт |
| negative_text | TEXT | | Негативный промпт |
| is_favorite | INTEGER | DEFAULT 0 | 0/1 |
| created_at | TEXT | NOT NULL | UTC RFC3339 |

### tag_presets

| Колонка | Тип | Ограничения | Описание |
|---------|-----|-------------|----------|
| id | INTEGER | PK AUTOINCREMENT | |
| name | TEXT | UNIQUE NOT NULL | Название пресета |
| positive_tags | TEXT | | JSON-массив |
| negative_tags | TEXT | | JSON-массив |

## Repo Layer (`database/repo.go`)

### `Repo` struct

Оборачивает `*sql.DB`. Все методы принимают/возвращают модели из `models.go`.

### Пакеты операций

**Packs:** GetPacks, GetPackByID, GetPackByName, CreatePack, UpdatePackMeta, DeletePack

**Files:** InsertFile, UpdateFile, UpsertFile, DeleteFilesByPack, DeleteFile, GetFilesByPack, GetFileByPackAndName

**Tags:** InsertTags (batch), DeleteTagsByFile, GetTagsByFile, DeleteTag

**Search/Tree:** GetTagCategory, SearchTags, GetCategoryTree, GetSubcategories, GetCategoryCounts, GetTagsByCategory (paginated)

**Favorites:** GetFavorites, AddFavorite, RemoveFavorite, IsFavorite

**Prompts:** SavePrompt, GetHistory (50), GetFavoritesPrompts, TrimHistory (50 max), DeletePrompt

**Presets:** GetPresets, SavePreset (upsert by name), DeletePreset

**Seeding:** SeedDefaultPreset — создаёт пресет "Pony Quality" при первом запуске

### UpsertFile

```sql
INSERT INTO files (...) VALUES (...)
ON CONFLICT(pack_id, file_name) DO UPDATE SET ...
```

Возвращает ID файла (существующего или только что созданного).

### Batch InsertTags

Вставка пачки тегов в одной транзакции через prepared statement.
