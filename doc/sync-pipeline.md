# Sync Pipeline

## Обзор

Сканирование папки `tags/`, парсинг CSV/TXT файлов, хеширование, синхронизация с SQLite.

```
tags/Danbooru/
├── info.pack          ← метаданные пака (JSON)
├── 1_general.txt      ← Danbooru category ID_name.txt
├── 3_copyright.txt
├── 4_character.txt
├── img/               ← превью тегов
│   ├── animalgirl/
│   ├── armor/
│   └── ...
```

## Формат info.pack

```json
{
  "name": "Danbooru",
  "name_ru": "Данбору",
  "description": "Full Danbooru tag set",
  "description_ru": "Полный набор тегов Danbooru",
  "version": "1.0",
  "author": "DmitriusFalse",
  "icon": "📦",
  "categories": [
    { "name": "general", "name_ru": "Общие", "file": "0_general.txt" },
    { "name": "artist", "name_ru": "Художник", "file": "1_artist.txt", "block_id": 4 },
    ...
  ]
}
```

`block_id` (опционально) — номер блока для группировки (1-7, по умолчанию 4).

## Парсинг (`sync/parser.go`)

### CSV

Файлы с расширением `.csv`. Формат:
```
tag_name, category_name, subcategory_name, aliases
```

Первая колонка с trim ведущих пробелов. Минимум 4 колонки. Пустые tag_name пропускаются.

### TXT

Файлы с расширением `.txt`. Простой список:
```
tag1
tag2
# комментарий
tag3
```

Пустые строки и строки с `#` пропускаются. Все теги получают одинаковые category_name и subcategory_name (из параметров).

## Хеширование (`sync/hasher.go`)

SHA-256 полного содержимого файла. Используется для детекта изменений при синхронизации.

## Сканирование (`sync/scanner.go`)

### Scanner.Scan(tagsPath)

1. Читает все поддиректории `tagsPath/`
2. Для каждой → `scanPack(name, path)`

### scanPack

1. Загружает/генерирует `info.pack` через `SaveGeneratedPackInfo`
2. Для каждой категории → `scanFile`
3. Возвращает `PackResult`

### scanFile

Маршрутизация: `.csv` → `scanCSVFile`, `.txt` → `scanTXTFile`

### parseFilename

Для CSV: разбирает `{id}_{category}_{subcategory}.csv`. Валидирует, что числовой ID соответствует ожидаемому имени из `categoryNameMap`:
- 0 → general
- 1 → artist
- 3 → copyright
- 4 → character
- 5 → meta

## SaveGeneratedPackInfo (`sync/packinfo.go`)

- Если `info.pack` не существует → генерирует и сохраняет
- Если существует → читает (не перезаписывает!)
- Если чтение сломанного файла провалилось → использует сгенерированный in-memory (не трогает битый файл на диске)

## Sync Service (`sync/sync.go`)

### Service.Sync(tagsPath)

1. Сканирует папку → получает `[]PackResult`
2. Загружает существующие паки из БД
3. Для каждого сканированного пака:
   - Если новый → создаёт в БД
   - Вызывает `syncFiles(dbPack, files)`
   - Обновляет метаданные пака
4. Удаляет паки, которых больше нет на диске

### syncFiles

1. Строит `set` имён файлов со сканера
2. Удаляет из БД файлы, которых нет на диске (каскадно удаляются и теги)
3. Для каждого сканированного файла:
   - Хеш совпадает → пропуск
   - Хеш отличается → обновить мету, diff тегов (insert new, delete removed)
   - Новый файл → insert файла + batch insert тегов
