# Фронтенд (SPA)

## Технологии

- **Alpine.js 3.x** — реактивность через HTML-атрибуты (CDN, без сборки)
- **CSS custom properties** — утилитарные классы (Tailwind-like), светлая/тёмная тема
- **PWA** — Service Worker (cache-first для статики, network-first для API), manifest.json
- **i18n** — JSON-файлы с ключами, переключение RU/EN

## Файлы

| Файл | Назначение |
|------|------------|
| `index.html` | Основная SPA-страница (558 строк) |
| `settings.html` | Страница настроек (280 строк) |
| `js/app.js` | Логика приложения (990 строк) |
| `styles.css` | Все стили (610+ строк) |
| `constants.json` | Статические теги (8 категорий, ~400+ тегов) |
| `presets.json` | 7 готовых пресетов |
| `i18n/ru.json` | Русские переводы (121 строка) |
| `i18n/en.json` | Английские переводы (121 строка) |
| `sw.js` | Service Worker (67 строк) |
| `manifest.json` | PWA-манифест |

## index.html — структура

### `<head>`

- Мета-теги, viewport
- PWA manifest + иконка
- Service Worker регистрация
- `beforeinstallprompt` обработчик → `pwaInstallable`
- Alpine.js CDN (3.x)
- `installPwa()` — глобальная функция

### `<header>`

- Заголовок с версией: `t('app.title') + (version ? ' v' + version : '')`
- Pack-селектор (Alpine dropdown)
- Кнопка синхронизации
- Donate ссылка (Boosty, анимированное сердечко)
- Theme switcher (auto/dark/light)
- Language toggle (RU/EN)
- PWA install (показывается при `pwaInstallable`)
- Settings link

### Sidebar (flex-[7])

- **Пресеты** — кнопки из `presetData`, применяют `applyPreset(name)`
- **Поиск** — input с debounce, результаты с fav/+/− кнопками
- **Избранное** — collapsible список избранных тегов
- **Constant tags** — 7 категорий из `constants.json`, collapsible, каждая с тегами + подкатегориями
- **Tree tags** — дерево категорий из API, lazy-load тегов при раскрытии

### Workspace (flex-[3])

- **Positive chips** — grouped by `block_id` (1-7) с заголовками, draggable
- **Negative chips** — flat список, без блоков
- **Custom tag input** — текстовое поле + кнопки +/−
- **Prompt textareas** — collapsible (toggle через `promptExpanded`), readonly, font-mono, кнопка копирования
- **Action buttons** — Save, Clear (с подтверждением), History
- **Tabs** — переключение между Tag Images и Generation (показываются только при `comfyEnabled`)
- **Tag images gallery** — lazy-load изображения всех чипов (вкладка Tags)
- **Generation UI** (вкладка Generation):
  - Workflow selector (из папки Workflows/)
  - Save path input
  - Checkpoint / Steps / CFG / Sampler / Scheduler / Resolution
  - Seed input + «Fix» checkbox
  - Кнопка Generate → прогресс-бар, статус, результат

### Drawer

- Slide-over панель с двумя табами: History / Favorites
- History — список сохранённых промптов
- Favorites — избранные теги + избранные промпты

### Modals

- Save prompt — название + кнопки
- Tree loading — progress bar (0-100%) при загрузке тегов категории

### Toast

- Фиксированное уведомление внизу (copy confirm, error)

### Tag image preview

- Абсолютный элемент, появляется при hover на тег

## app.js — Alpine.js компонент

### Глобальные константы

```js
const BLOCK_IDS = {
  'quality': 1, 'sources': 2, 'rating': 3,
  'appearance': 4,
  'pose': 5, 'scene': 6, 'style': 7
};
```

### Данные компонента (ключевые)

| Поле | Начальное | Описание |
|------|-----------|----------|
| `packs` | `[]` | Список паков |
| `selectedPackId` | `''` | ID выбранного пака |
| `constantTags` | `[]` | Статические теги из constants.json |
| `tagBlockMap` | `{}` | `tag → block_id` для констант |
| `positiveChips` | `[]` | Чипы позитива `{ name, category, subcategory, block_id }` |
| `negativeChips` | `[]` | Чипы негатива |
| `version` | `''` | Версия приложения |
| `tree` | `[]` | Дерево категорий |
| `searchResults` | `[]` | Результаты поиска |
| `favorites` | `[]` | Избранные теги |
| `history` | `[]` | История промптов |
| `dragState` | `null` | Состояние drag |
| `dropTarget` | `null` | Позиция drop |
| `comfyEnabled` | `false` | Включена ли интеграция ComfyUI |
| `comfyAddress` | `http://127.0.0.1:8188` | Адрес ComfyUI |
| `savePath` | `''` | Путь сохранения результатов |
| `resolutions` | `[]` | Список разрешений из настроек |
| `activeTab` | `'tags'` | Активная вкладка (`tags`/`generation`) |
| `promptExpanded` | `false` | Развёрнуты ли текстовые поля промптов |
| `workflows` | `[]` | Список воркфлоу из `Workflows/` |
| `checkpoints` | `[]` | Список чекпоинтов из ComfyUI |
| `samplers` | `[]` | Список семплеров |
| `schedulers` | `[]` | Список шедулеров |
| `seed` | `0` | Seed для генерации |
| `seedFixed` | `false` | Фиксировать seed (не менять при следующей генерации) |
| `generating` | `false` | Идёт ли генерация |
| `generationProgress` | `0` | Прогресс 0-100 |
| `generationStatus` | `''` | Текстовый статус |
| `generationResult` | `null` | URL результата (`/api/comfy/image?...`) |

### Computed (getters)

- `currentPack` — выбранный пак с локализованным именем
- `positivePrompt` — группировка чипов по block_id, ` BREAK ` между блоками
- `negativePrompt` — плоский список через `, `
- `isDark` — авто/тёмная/светлая тема

### Ключевые методы

**Инициализация:**
- `init()` — загружает пресеты, переводы, константы, паки, версию, ComfyUI-конфиг (`loadComfyConfig`)

**Чипы:**
- `resolveBlockId(category, subcategory)` — const → BLOCK_IDS, иначе 4
- `resolveBlockIdByName(tagName)` — lookup в tagBlockMap
- `makeChip(tag)` — создаёт объект чипа с block_id
- `addTag(tag)` / `addNegativeTag(tag)` — toggle добавление (сначала удаляет из противоположного)
- `removeChip(type, name)` — удаление
- `addCustomTag(negative)` — toggle кастомного тега

**DnD:**
- `onDragStart` — устанавливает dragState
- `onDragOver` — Euclidean distance до центра чипа, определяет before/after
- `onDrop` — перемещает чип между/внутри блоков

**Промпты:**
- `savePrompt()` / `doSavePrompt()` — сохранение
- `autoSavePrompt()` — debounced (150ms) localStorage
- `loadAutoSave()` — восстановление при старте
- `copyPrompt(type)` / `copyTagName(name)` — буфер обмена + toast

**ComfyUI:**
- `loadComfyConfig()` — GET `/api/config`, заполняет `comfyEnabled`, `comfyAddress`, `savePath`, `resolutions`
- `loadWorkflows()` — GET `/api/comfy/workflows`
- `loadCheckpoints()` — GET `/api/comfy/object_info/CheckpointLoaderSimple`
- `loadSamplers()` — GET `/api/comfy/object_info/KSampler`, извлекает `sampler_name` и `scheduler`
- `loadGenerationData()` — lazy-load воркфлоу/чекпоинты/семплеры при первом открытии вкладки Generation
- `generate()` — открывает WebSocket к `/api/comfy/ws`, POST `/api/comfy/generate`, обрабатывает `progress`/`executing`/`executed`/`execution_error` сообщения; после `executed` с изображением — POST `/api/comfy/save-image` для сохранения в целевую папку

**Изображения:**
- `showTagImage(event, tagName, isStatic)` — lazy-load изображения при hover

## styles.css

### Тема

CSS custom properties для светлой темы (`:root`), `.dark` класс переопределяет их.

Ключевые цвета:
- Фон: `#fff` / `#0f172a` (dark)
- Поверхность: `#f9fafb` / `#1e293b`
- Текст: `#111827` / `#f1f5f9`
- Акцент: `#2563eb` / `#93c5fd`
- Зелёный (pos): `#16a34a`
- Красный (neg): `#dc2626`

### Компоненты

- `.chip` / `.chip-positive` / `.chip-negative` — чипы в workspace
- `.tag-pill` — теги в сайдбаре (с `.selected-pos` / `.selected-neg`)
- `.block-section` / `.block-header` — группы чипов
- `.chip-dragging` — opacity 0.4 при drag
- `.drop-before` / `.drop-after` — box-shadow индикаторы
- `.drag-over` — пунктирная рамка
- `.toast` — уведомление
- `.tabs` / `.tab-btn` / `.tab-btn.active` — вкладки Tag Images / Generation
- `.gen-row` / `.gen-label` / `.gen-refresh` — строки управления генерацией
- `.btn-gen` / `.btn-gen-ready` / `.btn-gen-busy` — кнопка Generate (состояния)
- `.progress-bar` / `.progress-bar-fill` — прогресс-бар
- `.gen-result` / `.gen-result img` — контейнер результата
- `.collapse-toggle` — переключатель сворачивания промптов

## i18n

Файлы: `i18n/ru.json`, `i18n/en.json`

~68 ключей в категориях: app, theme, packs, settings, const (названия категорий), presets, search, fav, tag, workspace, custom, prompt, actions, images, drawer, modal, toast, block (заголовки блоков), donate, comfy (18 ключей: enable, address, save_path, resolutions, workflow, checkpoint, steps, cfg, sampler, scheduler, resolution, generate, generating, refresh, error, no_workflow, status, result, seed, seed_fix), tab (tags/generation), prompt (collapse/expand), settings (yes/no).

Переключение: `this.lang` → fetch `/static/i18n/{lang}.json` → `this.translations`

## PWA

**sw.js:**
- Cache name: `danbooru-prompt-builder-v1`
- Install: pre-cache `/`, js, manifest
- Activate: clean old caches
- Fetch: `/api/*` → network first (fallback to cache), всё остальное → cache first

**manifest.json:**
- `display: standalone` — без браузерного хрома
- Тёмные цвета темы
- Иконка `/static/icon.ico`
