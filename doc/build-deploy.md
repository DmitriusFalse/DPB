# Сборка и деплой

## build.bat

Полный пайплайн сборки Windows-бинарника.

### Этапы

1. **Проверка Go** — `go version`
2. **Установка зависимостей** — `go mod download`
3. **Инкремент версии** — читает `version.txt`, парсит X1.X2.X3, увеличивает X3 на 1
4. **Компиляция** — `go build -ldflags="-s -w -H=windowsgui" -trimpath -o build/DanbooruPromptBuilder.exe`
5. **Копирование config.json** — `copy /Y config.json build/config.json`
6. **Копирование tags/** — перезаписывает `build/tags/`
7. **Копирование img/** — перезаписывает `build/img/`
10. **Копирование Workflows/** — перезаписывает `build/Workflows/`
11. **Очистка** — удаляет `rsrc.syso`, `.log` файлы

### Параметры сборки

- `GOOS=windows`, `GOARCH=amd64`
- `CGO_ENABLED=1` (необходим для sqlite3)
- `-s -w` — strip debug info
- `-H=windowsgui` — без консоли
- `-trimpath` — удалить пути сборки

### Результат

`build/DanbooruPromptBuilder.exe` (~9.5 MB) + папки `tags/` и `img/` рядом с exe.

## Версионирование

### version.txt

```
1.0.31
```

Формат: `X1.X2.X3`

- **X1, X2** — меняются вручную
- **X3** — автоматический инкремент при каждой сборке (начиная с 30)

### Отображение

Версия встраивается в бинарник через `//go:embed version.txt` и экспонируется через `/api/version`. Фронтенд подхватывает её и показывает в `<title>` и `<h1>`.

## Коммит и релиз

- Изменения в `version.txt` (инкремент) не коммитятся — это build artifact
- Для нового релиза: вручную выставить X1.X2, запушить, создать GitHub release
- `tmpcov` в `.gitignore` — не коммитить
- `data.db*` в `.gitignore` — не коммитить
- `build/` в `.gitignore` — не коммитить
- `*.exe` в `.gitignore`

## Тестирование

```bash
go test ./...

# vet
go vet ./...
```

Тесты: config, database (repo, db), handler, sync (packinfo, parser, scanner, hasher, sync).
