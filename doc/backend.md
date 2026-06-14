# Бэкенд (Go)

## main.go

Точка входа. Embed `version.txt`, инициализация всех подсистем, запуск HTTP-сервера и системногоトレя.

```go
//go:embed version.txt
var buildVersion string
```

Экспонирует `/api/version` напрямую (не через `handler.RegisterRoutes`):

```go
mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"version": version})
})
```

**Зависимости:** `github.com/mattn/go-sqlite3`, `github.com/gorilla/websocket`, `github.com/webview/webview_go`

---

## config (`config/config.go`)

### `Config` struct

| Поле | JSON | По умолчанию | Описание |
|------|------|-------------|----------|
| `Port` | `port` | `8080` | Порт HTTP-сервера |
| `TagsPath` | `tags_path` | `./tags` | Путь к папке с паками |
| `DBPath` | `db_path` | `./data.db` | Путь к SQLite |
| `LogLevel` | `log_level` | `"error"` | `"error"` или `"debug"` |
| `LogsDir` | `logs_dir` | `./logs` | Папка логов |
| `StaticImgPath` | `static_img_path` | `./img` | Папка статических изображений тегов |
| `ComfyEnabled` | `comfy_enabled` | `false` | Включить интеграцию с ComfyUI |
| `ComfyAddress` | `comfy_address` | `http://127.0.0.1:8188` | Адрес ComfyUI |
| `SavePath` | `save_path` | `./output` | Папка для сохранения сгенерированных изображений |
| `Resolutions` | `resolutions` | `512x512` | Разрешения (одно на строку) |
| `WorkflowsPath` | — | `{exe_dir}/Workflows` | Вычисляется в `routes.go`, не сериализуется (`json:"-"`) |

### Функции

- `Load(path)` — загружает JSON, создаёт дефолтный при отсутствии, resolve относительных путей от папки конфига
- `Save(path)` — сохраняет indented JSON
- `resolvePath(base, target)` — absolute → as-is, relative → join

---

## logger (`logger/logger.go`)

Логирование с ротацией по дате.

- `Init(level, dir)` — инициализация
- `Error(format, args...)` / `Debug(format, args...)` — уровни
- Файл: `app-YYYY-MM-DD.log` в указанной папке
- Вывод: одновременно в файл и stderr
- Формат: `[HH:MM:SS.mmm] LEVEL message`

---

## handler — HTTP Handlers

### Назначение

Все хендлеры в пакете `handler`. Роуты регистрируются в `RegisterRoutes`.

### Middleware

`apiMiddleware` — panic recovery. Оборачивает все `/api/*` хендлеры.

### Список хендлеров

| Файл | Хендлер | Назначение |
|------|---------|------------|
| `index.go` | `handleIndex()` | Служит `index.html` |
| `index.go` | `handleSettingsPage()` | Служит `settings.html` |
| `config.go` | `handleConfig(cfg, configPath)` | GET/PUT конфига |
| `packs.go` | `handleGetPackByID(repo)` | GET пака по ID |
| `packs.go` | `handleReadPackInfoFromReader(repo)` | Читает `info.pack` с диска |
| `packs.go` | `handlePacks(repo)` | GET список / DELETE пак |
| `tree.go` | `handleTree(repo)` | Дерево категорий + пагинация тегов |
| `search.go` | `handleSearch(repo)` | Поиск тегов |
| `favorites.go` | `handleFavorites(repo)` | GET/POST избранных тегов |
| `presets.go` | `handlePresets(repo)` | GET/POST пресетов |
| `prompts.go` | `handlePrompts(repo)` | GET/POST/DELETE промптов |
| `static.go` | `StaticHandler()` | Раздача встроенных статических файлов |
| `staticimage.go` | `handleStaticImage(cfg)` | Изображения статических тегов |
| `tagimage.go` | `handleTagImage(repo)` | Изображения тегов паков |
| `sync_handler.go` | `handleSync(syncSvc, cfg)` | Триггер ресинхронизации |
| `comfy.go` | `handleComfyWorkflows(cfg)` | GET список/содержимое воркфлоу |
| `comfy.go` | `handleComfyGenerate(cfg)` | POST — макроподстановка, отправка в ComfyUI `/prompt` |
| `comfy.go` | `handleComfyImage(cfg)` | GET — прокси к ComfyUI `/view` |
| `comfy.go` | `handleComfyObjectInfo(cfg)` | GET — прокси к ComfyUI `/object_info` |
| `comfy.go` | `handleComfySaveImage(cfg)` | POST — скачать из ComfyUI `/view`, сохранить в `cfg.SavePath` |
| `comfy.go` | `handleComfyWS(cfg)` | WS — прокси WebSocket (gorilla/websocket), пайп браузер↔ComfyUI |

### comfy.go — макроподстановка

Макросы в JSON воркфлоу заменяются в два прохода:
- `"%KEY%"` (в кавычках) → JSON-escaped строка
- `%KEY%` (без кавычек) → сырое значение (числа, булевы)

### comfy.go — WebSocket proxy

`gorilla/websocket.Upgrader{CheckOrigin: true}` — разрешает любой Origin.  
Пайп в обе стороны через две горутины. Соединение закрывается при первой ошибке в любом направлении.

### staticimage.go — изображения статических тегов

- `tkeyToFolder` map: `const.quality → quality`, `const.appearance → appearance` и т.д.
- `getStaticTagCategory()` — lazy-load `constants.json`, строит `tag → folder`
- Ищет файл: `{StaticImgPath}/{folder}/{tag}.png`
- Возвращает 1x1 transparent GIF при ошибке

### tagimage.go — изображения тегов паков

- Путь: `{pack.Path}/img/{tag}.png` или `{pack.Path}/img/{category}/{tag}.png`
- Прозрачный GIF при отсутствии

### static.go — статика

- `//go:embed all:static` — все файлы `handler/static/` встроены
- `detectContentType` — `.html`/`.css`/`.js`/`.json`/`.ico`
- Используется `http.FileServer` для раздачи embedded FS

### handler_test.go

Тесты для: config (GET/PUT), packs (GET/DELETE), search, tree, presets, favorites, sync, static image, tag image, prompts.
