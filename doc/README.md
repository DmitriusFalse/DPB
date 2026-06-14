# Danbooru Prompt Builder

Desktop Windows-приложение для сборки AI-промптов из тегов Danbooru с группировкой по блокам и drag-and-drop сортировкой.

**Версия:** см. `version.txt` (X1.X2.X3, X3 автоинкремент при сборке)  
**Язык:** Go 1.26.1 (бэкенд) + Alpine.js 3.x (фронтенд)  
**БД:** SQLite (WAL mode)  
**Сборка:** `build.bat` → `build/DanbooruPromptBuilder.exe`

---

## Содержание документации

| Файл | О чём |
|------|-------|
| [architecture.md](architecture.md) | Общая архитектура, компоненты, связи |
| [backend.md](backend.md) | Go-пакеты: main, config, logger, handler |
| [frontend.md](frontend.md) | SPA: Alpine.js, HTML, CSS, i18n, PWA |
| [api.md](api.md) | Все HTTP-эндпоинты |
| [database.md](database.md) | SQLite: схема, модели, репозиторий |
| [sync-pipeline.md](sync-pipeline.md) | Сканирование паков, парсинг, синхронизация |
| [build-deploy.md](build-deploy.md) | Сборка, версионирование, деплой |
