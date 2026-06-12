const BLOCK_IDS = { 'quality': 1, 'sources': 2, 'rating': 3, 'appearance': 4, 'pose': 5, 'scene': 6, 'style': 7 };
const BLOCK_COLORS = [null, '#60cdff', '#87b6ff', '#aa7be0', '#d48ebd', '#6ccb6c', '#e8b84a', '#f59a44'];

function app() {
  let _tagImgId = 0;
  return {
    pwaInstallable: pwaInstallable,

    // Theme: 'auto', 'dark', 'light'
    theme: localStorage.getItem('theme') || 'auto',

    // i18n
    lang: localStorage.getItem('lang') || ((navigator.language || '').startsWith('ru') ? 'ru' : 'en'),
    translations: {},

    t(key) {
      return this.translations[key] || key;
    },

    async loadTranslations() {
      try {
        const res = await fetch(`/static/i18n/${this.lang}.json`);
        this.translations = await res.json();
      } catch (e) {
        console.error('Failed to load translations:', e);
        this.translations = {};
      }
    },

    async loadPresets() {
      try {
        const res = await fetch('/static/presets.json');
        this.presetData = await res.json();
      } catch (e) {
        console.error('Failed to load presets:', e);
        this.presetData = {};
      }
    },

    async loadConstants() {
      this.tagBlockMap = {};
      this.tagInfoMap = {};
      try {
        const res = await fetch('/static/constants.json');
        if (!res.ok) { console.error('loadConstants status:', res.status); return; }
        this.constantTags = await res.json();
        for (const group of this.constantTags) {
          const tkey = group.tkey || '';
          const parts = tkey.split('.');
          const cat = parts.length >= 2 ? parts[1] : '';
          const blockId = BLOCK_IDS[cat];
          if (!blockId) continue;
          const subcatKey = group.subcat || cat;
          if (group.tags) {
            for (const tag of group.tags) {
              this.tagBlockMap[tag] = blockId;
              this.tagInfoMap[tag] = { category: 'const', subcategory: subcatKey };
            }
          }
          if (group.subcategories) {
            for (const sub of group.subcategories) {
              if (sub.tags) {
                for (const tag of sub.tags) {
                  this.tagBlockMap[tag] = blockId;
                  this.tagInfoMap[tag] = { category: 'const', subcategory: subcatKey };
                }
              }
            }
          }
        }
      } catch (e) {
        console.error('Failed to load constants:', e);
        this.constantTags = [];
      }
    },

    // Toast
    toastText: '',
    toastVisible: false,
    toastTimer: null,

    // Fast lookup sets (rebuilt on chip mutations)
    posNames: {},
    negNames: {},
    _autoSaveTimer: null,

    // Packs
    packs: [],
    selectedPackId: '',
    syncing: false,

    get currentPack() {
      const p = this.packs.find(p => p.id === this.selectedPackId);
      if (!p) return { name: '', icon: '📦' };
      return {
        ...p,
        name: this.lang === 'ru' && p.name_ru ? p.name_ru : p.name
      };
    },

    tCat(categoryName) {
      const p = this.packs.find(p => p.id === this.selectedPackId);
      if (!p || !p.categories_list) return categoryName;
      const cat = p.categories_list.find(c => c.name === categoryName);
      if (this.lang === 'ru' && cat && cat.name_ru) return cat.name_ru;
      return categoryName;
    },

    // Search
    searchQuery: '',
    searchResults: [],

    // Tree
    tree: [],
    treeOpen: {},

    treeModal: false,
    treeModalProgress: 0,
    _treeLoading: null,

    // Constant tags
    constOpen: {},
    constSubOpen: {},
    constantTags: [],
    categoryColor: {},
    tagBlockMap: {},
    tagInfoMap: {},
    version: '',

    // ComfyUI
    comfyEnabled: false,
    resolutions: [],
    activeTab: 'tags',
    promptsOpen: false,
    workflows: [],
    selectedWorkflow: '',
    checkpoints: [],
    selectedCheckpoint: '',
    steps: 20,
    cfg: 7,
    samplers: [],
    selectedSampler: 'euler',
    schedulers: [],
    selectedScheduler: 'normal',
    selectedResolution: '512x512',
    generating: false,
    generationProgress: 0,
    generationStatus: '',
    generationResult: null,
    generationHistory: [],
    viewerImage: null,
    viewerIndex: -1,
    _genDataLoaded: false,
    seed: 0,
    seedFixed: false,
    nodeTitles: {},
    tagToCategory: {},

    // Layout
    leftRatio: parseInt(localStorage.getItem('layout_left') || '20'),
    rightWidth: parseInt(localStorage.getItem('layout_right') || '400'),
    workNoComfyRatio: parseInt(localStorage.getItem('layout_work_nc') || '75'),
    workComfyRatio: parseInt(localStorage.getItem('layout_work_c') || '30'),
    resizing: null,

    // Favorites
    favorites: [],
    favOpen: false,

    // Chips as objects: { name, category, subcategory, block_id }
    positiveChips: [],
    negativeChips: [],
    dragState: null,
    dropTarget: null,
    _ignoreNextClick: false,

    // Custom input
    customTag: '',

    // Presets
    presetData: {},

    // Preview panel
    previewTab: 'images',
    history: [],
    favoritePrompts: [],

    // ─── Init ───

    async init() {
      this.loadPresets();
      await this.loadTranslations();
      await this.loadConstants();
      this.assignColors();
      await this.loadPacks();
      document.addEventListener('pwa-installable', () => {
        this.pwaInstallable = true;
      });
      try {
        const r = await fetch('/api/version');
        if (r.ok) { const d = await r.json(); this.version = d.version; }
      } catch(e) {}
      await this.loadComfyConfig();
      this.loadGenerationHistory();
    },

    // ─── Packs ───

    async loadPacks() {
      try {
        const res = await fetch('/api/packs');
        if (!res.ok) { console.error('loadPacks status:', res.status); return; }
        const list = await res.json();
        for (const p of list) {
          try {
            p.categories_list = JSON.parse(p.categories || '[]');
          } catch(e) {
            p.categories_list = [];
          }
        }
        this.packs = list;
        if (this.packs.length > 0 && !this.selectedPackId) {
          this.selectedPackId = this.packs[0].id;
          await this.refreshFreshCategories(this.selectedPackId);
          this.loadAll();
        }
      } catch (e) {
        console.error('loadPacks:', e);
      }
    },

    async refreshFreshCategories(packId) {
      try {
        const res = await fetch(`/api/pack/info?id=${packId}`);
        if (!res.ok) return;
        const info = await res.json();
        const pack = this.packs.find(p => p.id === packId);
        if (pack && info.categories) {
          pack.categories_list = info.categories;
        }
      } catch (e) {
        console.error('refreshFreshCategories:', e);
      }
    },

    async syncTags() {
      if (this.syncing) return;
      this.syncing = true;
      try {
        const res = await fetch('/api/sync', { method: 'POST' });
        if (res.ok) {
          await this.loadPacks();
          this.loadAll();
        } else {
          const data = await res.json().catch(() => ({}));
          this.showToast(data.error || 'Sync failed: ' + res.status, 5000);
        }
      } catch (e) {
        console.error('sync:', e);
        this.showToast('Sync error: ' + e.message, 5000);
      } finally {
        this.syncing = false;
      }
    },

    // ─── Tree ───

    async loadTree() {
      if (!this.selectedPackId) return;
      try {
        const res = await fetch(`/api/tags/tree?pack_id=${this.selectedPackId}`);
        if (!res.ok) { console.error('loadTree status:', res.status); return; }
        this.tree = await res.json();
        this.treeOpen = {};
        this.assignColors();
      } catch (e) {
        console.error('loadTree:', e);
      }
    },

    async toggleCategory(cat) {
      if (!cat) return;
      const name = typeof cat === 'string' ? cat : cat.name;
      this.treeOpen[name] = !this.treeOpen[name];
      if (this.treeOpen[name]) {
        const treeCat = this.tree.find(c => c.name === name);
        if (treeCat && !treeCat._tags) {
          if (this._treeLoading) return;
          this._treeLoading = name;
          this.treeModal = true;
          this.treeModalProgress = 0;
          try {
            const res = await fetch(`/api/tags/tree?pack_id=${this.selectedPackId}&category=${encodeURIComponent(name)}&offset=0&limit=99999`);
            const page = await res.json();
            treeCat._tags = page.tags || [];
            for (const t of treeCat._tags) {
              this.tagToCategory[t.tag_name] = name;
              this.tagInfoMap[t.tag_name] = { category: name, subcategory: '' };
            }
            this.treeModalProgress = 100;
          } catch (e) {
            console.error('toggleCategory:', e);
            treeCat._tags = [];
          } finally {
            this._treeLoading = null;
            this.treeModal = false;
          }
        }
      }
    },

    // ─── Search ───

    async searchTags() {
      if (!this.selectedPackId || !this.searchQuery.trim()) {
        this.searchResults = [];
        return;
      }
      try {
        const res = await fetch(`/api/tags/search?pack_id=${this.selectedPackId}&q=${encodeURIComponent(this.searchQuery)}&limit=20`);
        if (!res.ok) { console.error('searchTags status:', res.status); return; }
        this.searchResults = await res.json();
      } catch (e) {
        console.error('search:', e);
      }
    },

    // ─── Chips ───

    resolveBlockId(category, subcategory) {
      if (category === 'const') {
        return BLOCK_IDS[subcategory] || 4;
      }
      return 4;
    },

    resolveBlockIdByName(tagName) {
      return this.tagBlockMap[tagName] || 4;
    },

    makeChip(tag) {
      const category = tag.category_name || '';
      const subcategory = tag.subcategory_name || '';
      let block_id = this.resolveBlockId(category, subcategory);
      if (block_id === 4) {
        const pack = this.packs.find(p => p.id === this.selectedPackId);
        if (pack && pack.categories_list) {
          const catCfg = pack.categories_list.find(c => c.name === category);
          if (catCfg && catCfg.block_id) block_id = catCfg.block_id;
        }
      }
      return { name: tag.tag_name, category, subcategory, block_id };
    },

    addTag(tag) {
      const ch = this.makeChip(tag);
      const posIdx = this.positiveChips.findIndex(c => c.name === ch.name);
      if (posIdx !== -1) {
        this.positiveChips.splice(posIdx, 1);
      } else {
        const negIdx = this.negativeChips.findIndex(c => c.name === ch.name);
        if (negIdx !== -1) this.negativeChips.splice(negIdx, 1);
        this.positiveChips.push(ch);
      }
      this.updateChipNames();
      this.autoSavePrompt();
    },

    addNegativeTag(tag) {
      const ch = this.makeChip(tag);
      const negIdx = this.negativeChips.findIndex(c => c.name === ch.name);
      if (negIdx !== -1) {
        this.negativeChips.splice(negIdx, 1);
      } else {
        const posIdx = this.positiveChips.findIndex(c => c.name === ch.name);
        if (posIdx !== -1) this.positiveChips.splice(posIdx, 1);
        this.negativeChips.push(ch);
      }
      this.updateChipNames();
      this.autoSavePrompt();
    },

    removeChip(type, name) {
      if (this._ignoreNextClick) { this._ignoreNextClick = false; return; }
      const arr = type === 'positive' ? this.positiveChips : this.negativeChips;
      const idx = arr.findIndex(c => c.name === name);
      if (idx !== -1) arr.splice(idx, 1);
      this.updateChipNames();
      this.autoSavePrompt();
    },

    // ─── Drag & drop ───

    onDragStart(ev, name) {
      this.dragState = { name };
      ev.dataTransfer.effectAllowed = 'move';
      ev.dataTransfer.setData('text/plain', name);
      ev.currentTarget.classList.add('chip-dragging');
    },

    _clearDropVisuals() {
      document.querySelectorAll('.drag-over, .drop-before, .drop-after')
        .forEach(el => el.classList.remove('drag-over', 'drop-before', 'drop-after'));
    },

    onDragEnd(ev) {
      ev.currentTarget.classList.remove('chip-dragging');
      this._clearDropVisuals();
      this.dragState = null;
      this.dropTarget = null;
    },

    onDragOver(ev) {
      ev.preventDefault();
      ev.dataTransfer.dropEffect = 'move';
      const name = this.dragState?.name;
      if (!name) return;
      const blockEl = ev.currentTarget.closest('[data-block-id]') || ev.currentTarget;
      const chipEls = [...blockEl.querySelectorAll('[data-chip-name]')]
        .filter(el => el.dataset.chipName !== name);
      // Find closest chip by Euclidean distance to center
      let closestEl = null, closestDist = Infinity;
      for (const el of chipEls) {
        const r = el.getBoundingClientRect();
        const dx = ev.clientX - (r.left + r.width / 2);
        const dy = ev.clientY - (r.top + r.height / 2);
        const d = dx * dx + dy * dy;
        if (d < closestDist) { closestDist = d; closestEl = el; }
      }
      // Compute new drop target
      let newInsertBeforeName = null, newInsertAfterName = null;
      if (closestEl) {
        const r = closestEl.getBoundingClientRect();
        if (ev.clientX < r.left + r.width / 2) {
          newInsertBeforeName = closestEl.dataset.chipName;
        } else {
          newInsertAfterName = closestEl.dataset.chipName;
        }
      }
      // Update visual if changed
      const prev = this.dropTarget;
      if (!prev || prev.before !== newInsertBeforeName || prev.after !== newInsertAfterName) {
        // Clear old visuals on all chips
        blockEl.querySelectorAll('.drop-before, .drop-after')
          .forEach(el => el.classList.remove('drop-before', 'drop-after'));
        // Apply new visual
        if (newInsertBeforeName) {
          const el = blockEl.querySelector(`[data-chip-name="${CSS.escape(newInsertBeforeName)}"]`);
          if (el) el.classList.add('drop-before');
        }
        if (newInsertAfterName) {
          const el = blockEl.querySelector(`[data-chip-name="${CSS.escape(newInsertAfterName)}"]`);
          if (el) el.classList.add('drop-after');
        }
        this.dropTarget = { before: newInsertBeforeName, after: newInsertAfterName };
      }
    },

    onDragEnter(ev) {
      ev.preventDefault();
      if (!ev.currentTarget.contains(ev.relatedTarget)) {
        ev.currentTarget.classList.add('drag-over');
      }
    },

    onDragLeave(ev) {
      if (!ev.currentTarget.contains(ev.relatedTarget)) {
        ev.currentTarget.classList.remove('drag-over');
      }
    },

    onDrop(ev, targetType) {
      ev.preventDefault();
      this._clearDropVisuals();
      this._ignoreNextClick = true;
      const name = this.dragState?.name;
      if (!name) return;
      const allChips = [...this.positiveChips, ...this.negativeChips];
      const chip = allChips.find(c => c.name === name);
      if (!chip) return;
      const srcArr = this.positiveChips.includes(chip) ? this.positiveChips : this.negativeChips;
      const tgtArr = targetType === 'positive' ? this.positiveChips : this.negativeChips;
      const raw = parseInt(ev.currentTarget.dataset.blockId);
      const targetBlockId = isNaN(raw) ? 4 : raw;
      const dt = this.dropTarget;

      // Remove from old position
      const oldIdx = srcArr.indexOf(chip);
      srcArr.splice(oldIdx, 1);

      // Update block_id for target
      chip.block_id = targetBlockId;

      if (dt && dt.before) {
        const idx = tgtArr.findIndex(c => c.name === dt.before);
        tgtArr.splice(idx < 0 ? 0 : idx, 0, chip);
      } else if (dt && dt.after) {
        const idx = tgtArr.findIndex(c => c.name === dt.after);
        tgtArr.splice(idx < 0 ? tgtArr.length : idx + 1, 0, chip);
      } else {
        // No target — append to end of block
        const blockChips = tgtArr.filter(c => (c.block_id || 4) === targetBlockId);
        if (blockChips.length > 0) {
          const lastInBlock = blockChips[blockChips.length - 1];
          tgtArr.splice(tgtArr.indexOf(lastInBlock) + 1, 0, chip);
        } else {
          let insertIdx = 0;
          for (let i = 0; i < tgtArr.length; i++) {
            if ((tgtArr[i].block_id || 4) < targetBlockId) insertIdx = i + 1;
          }
          tgtArr.splice(insertIdx, 0, chip);
        }
      }
      this.dragState = null;
      this.dropTarget = null;
      this.updateChipNames();
      this.autoSavePrompt();
    },

    clearPositiveChips() {
      this.positiveChips.splice(0);
      this.updateChipNames();
      this.autoSavePrompt();
    },

    clearNegativeChips() {
      this.negativeChips.splice(0);
      this.updateChipNames();
      this.autoSavePrompt();
    },

    // ─── Custom tag ───

    addCustomTag(negative) {
      const name = this.customTag.trim();
      if (!name) return;
      const ch = { name, category: 'custom', subcategory: '', block_id: this.resolveBlockIdByName(name) };
      // Remove from both arrays first (toggle behavior)
      const posIdx = this.positiveChips.findIndex(c => c.name === name);
      const negIdx = this.negativeChips.findIndex(c => c.name === name);
      if (posIdx !== -1) this.positiveChips.splice(posIdx, 1);
      if (negIdx !== -1) this.negativeChips.splice(negIdx, 1);
      // Add to target array
      if (negative) {
        this.negativeChips.push(ch);
      } else {
        this.positiveChips.push(ch);
      }
      this.customTag = '';
      this.updateChipNames();
      this.autoSavePrompt();
    },

    // ─── Favorites ───

    async loadFavorites() {
      if (!this.selectedPackId) return;
      try {
        const res = await fetch(`/api/favorites?pack_id=${this.selectedPackId}`);
        if (!res.ok) { console.error('loadFavorites status:', res.status); return; }
        this.favorites = await res.json();
      } catch (e) {
        console.error('loadFavorites:', e);
      }
    },

    async toggleFavorite(tag) {
      try {
        const isFav = this.favorites.some(f => f.tag_name === tag.tag_name);
        const res = await fetch('/api/favorites', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            pack_id: parseInt(this.selectedPackId),
            tag_name: tag.tag_name,
            add: !isFav
          })
        });
        if (!res.ok) { console.error('toggleFavorite status:', res.status); return; }
        this.loadFavorites();
      } catch (e) {
        console.error('toggleFavorite:', e);
      }
    },

    isFav(tagName) {
      return this.favorites.some(f => f.tag_name === tagName);
    },

    selectedClass(tagName) {
      if (this.posNames[tagName]) return 'selected-pos';
      if (this.negNames[tagName]) return 'selected-neg';
      return '';
    },

    selectedCountInCategory(cat) {
      let count = 0;
      if (cat.tags) {
        for (const t of cat.tags) {
          if (this.posNames[t] || this.negNames[t]) count++;
        }
      }
      if (cat.subcategories) {
        for (const sub of cat.subcategories) {
          if (sub.tags) {
            for (const t of sub.tags) {
              if (this.posNames[t] || this.negNames[t]) count++;
            }
          }
        }
      }
      return count;
    },

    selectedCountInSub(sub) {
      let count = 0;
      if (sub.tags) {
        for (const t of sub.tags) {
          if (this.posNames[t] || this.negNames[t]) count++;
        }
      }
      return count;
    },

    selectedCountInTree(cat) {
      let count = 0;
      for (const ch of this.positiveChips) {
        if (ch.category === cat.name || this.tagToCategory[ch.name] === cat.name) count++;
        else if (cat._tags && cat._tags.some(t => t.tag_name === ch.name)) count++;
      }
      for (const ch of this.negativeChips) {
        if (ch.category === cat.name || this.tagToCategory[ch.name] === cat.name) count++;
        else if (cat._tags && cat._tags.some(t => t.tag_name === ch.name)) count++;
      }
      return count;
    },

    counterColorClass(count) {
      if (count === 0) return 'text-gray-400 dark:text-dark-400';
      if (count <= 3) return 'text-blue-600 dark:text-blue-400 font-semibold';
      return 'text-green-600 dark:text-green-400 font-semibold';
    },

    updateChipNames() {
      const p = {}, n = {};
      for (const c of this.positiveChips) p[c.name] = true;
      for (const c of this.negativeChips) n[c.name] = true;
      this.posNames = p;
      this.negNames = n;
      const tc = {};
      for (const c of this.positiveChips) { if (c.category) tc[c.name] = c.category; }
      for (const c of this.negativeChips) { if (c.category) tc[c.name] = c.category; }
      this.tagToCategory = tc;
    },

    assignColors() {
      const map = {};
      let i = 1;
      for (const cat of this.tree) {
        const idx = i++ % 7 + 1;
        map[cat.name] = BLOCK_COLORS[idx];
      }
      for (const cat of this.constantTags) {
        const key = cat.tkey?.split('.').pop() || cat.name;
        if (!map[key]) {
          const idx = i++ % 7 + 1;
          map[key] = BLOCK_COLORS[idx];
        }
      }
      this.categoryColor = map;
    },

    getColorForChip(chip) {
      return BLOCK_COLORS[chip.block_id] || null;
    },

    getCategoryColor(cat) {
      const key = cat.tkey?.split('.').pop() || cat.name;
      if (this.categoryColor[key]) return this.categoryColor[key];
      const blockId = BLOCK_IDS[key];
      if (blockId) return BLOCK_COLORS[blockId];
      return null;
    },

    chipCategoryName(chip) {
      let catName = '';
      let isConst = false;
      if (chip.category === 'const') {
        catName = chip.subcategory;
        isConst = true;
      } else if (chip.category === 'meta' && this.tagInfoMap[chip.name]) {
        const info = this.tagInfoMap[chip.name];
        catName = info.subcategory;
        isConst = info.category === 'const';
      } else if (chip.category && chip.category !== 'meta') {
        catName = chip.category;
      }
      if (!catName) return '';
      if (isConst) {
        const group = this.constantTags.find(g => (g.tkey?.split('.').pop() || g.name) === catName);
        return group ? this.t(group.tkey || group.name) : catName;
      }
      return this.tCat(catName) || catName;
    },

    enrichChips() {
      let dirty = false;
      for (const ch of this.positiveChips) {
        if (ch.category === 'meta' && this.tagInfoMap[ch.name]) {
          ch.category = this.tagInfoMap[ch.name].category;
          ch.subcategory = this.tagInfoMap[ch.name].subcategory;
          dirty = true;
        }
      }
      for (const ch of this.negativeChips) {
        if (ch.category === 'meta' && this.tagInfoMap[ch.name]) {
          ch.category = this.tagInfoMap[ch.name].category;
          ch.subcategory = this.tagInfoMap[ch.name].subcategory;
          dirty = true;
        }
      }
      if (dirty) {
        this.positiveChips = this.positiveChips.slice();
        this.negativeChips = this.negativeChips.slice();
      }
    },

    // ─── Presets ───

    applyPreset(name) {
      const data = this.presetData[name];
      if (!data) return;
      this.positiveChips.splice(0);
      this.negativeChips.splice(0);

      const posParts = this.splitAtBreak(data.positive);
      const posSubs = ['artist', 'general'];
      for (let i = 0; i < posParts.length; i++) {
        const sub = posSubs[i] || 'general';
        for (const n of posParts[i]) {
          const ch = { name: n, category: 'meta', subcategory: sub, block_id: this.resolveBlockIdByName(n) };
          if (!this.positiveChips.some(c => c.name === ch.name)) {
            this.positiveChips.push(ch);
          }
        }
      }

      for (const n of data.negative) {
        const ch = { name: n, category: 'meta', subcategory: 'general', block_id: 4 };
        if (!this.negativeChips.some(c => c.name === ch.name)) {
          this.negativeChips.push(ch);
        }
      }
      this.enrichChips();
      this.updateChipNames();
      this.autoSavePrompt();
    },

    splitAtBreak(arr) {
      const parts = [[]];
      for (const item of arr) {
        if (item === 'BREAK') {
          parts.push([]);
        } else {
          parts[parts.length - 1].push(item);
        }
      }
      return parts;
    },

    // ─── Prompt actions ───



    async loadHistory() {
      try {
        const res = await fetch('/api/prompts?favorites=0');
        if (!res.ok) { console.error('loadHistory status:', res.status); return; }
        this.history = await res.json();
      } catch (e) {
        console.error('loadHistory:', e);
      }
    },

    async loadFavoritePrompts() {
      try {
        const res = await fetch('/api/prompts?favorites=1');
        if (!res.ok) { console.error('loadFavoritePrompts status:', res.status); return; }
        this.favoritePrompts = await res.json();
      } catch (e) {
        console.error('loadFavPrompts:', e);
      }
    },

    async copyTagName(name) {
      if (!name) return;
      try {
        await navigator.clipboard.writeText(name);
        this.showToast((this.t('toast.copied') || 'Скопировано') + ': ' + name);
      } catch (e) {
        console.error('copyTagName:', e);
      }
    },

    showToast(text, duration = 1500) {
      this.toastText = text;
      this.toastVisible = true;
      if (this.toastTimer) clearTimeout(this.toastTimer);
      this.toastTimer = setTimeout(() => { this.toastVisible = false; }, duration);
    },

    async copyPrompt(type) {
      if (type !== 'positive' && type !== 'negative') return;
      const text = type === 'positive' ? this.positivePrompt : this.negativePrompt;
      if (!text) return;
      try {
        await navigator.clipboard.writeText(text);
        this.showToast((this.t('toast.copied') || 'Скопировано: ') + text.substring(0, 30) + (text.length > 30 ? '…' : ''));
      } catch (e) {
        console.error('copy:', e);
      }
    },

    async loadPrompt(p) {
      this.positiveChips.splice(0);
      this.negativeChips.splice(0);
      const posParser = parsePromptData(p.positive_text);
      for (const n of posParser) {
        const ch = { name: n, category: 'meta', subcategory: 'loaded', block_id: this.resolveBlockIdByName(n) };
        if (!this.positiveChips.some(c => c.name === n)) {
          this.positiveChips.push(ch);
        }
      }
      const negParser = parsePromptData(p.negative_text);
      for (const n of negParser) {
        const ch = { name: n, category: 'meta', subcategory: 'loaded', block_id: this.resolveBlockIdByName(n) };
        if (!this.negativeChips.some(c => c.name === n)) {
          this.negativeChips.push(ch);
        }
      }
      // Restore generation settings if available
      if (p.gen_data) {
        try {
          const g = JSON.parse(p.gen_data);
          if (g.workflow) this.selectedWorkflow = g.workflow;
          if (g.checkpoint) this.selectedCheckpoint = g.checkpoint;
          if (g.resolution) this.selectedResolution = g.resolution;
          if (g.sampler) this.selectedSampler = g.sampler;
          if (g.scheduler) this.selectedScheduler = g.scheduler;
          if (g.steps) this.steps = g.steps;
          if (g.cfg) this.cfg = g.cfg;
          if (g.seed !== undefined) { this.seed = g.seed; this.seedFixed = true; }
        } catch(_) {}
      }
      this.activeTab = 'generation';
      this.enrichChips();
      this.updateChipNames();
      this.autoSavePrompt();
    },

    async deleteHistoryItem(id) {
      try {
        await fetch('/api/prompts?id=' + id, { method: 'DELETE' });
        this.loadHistory();
      } catch(e) { console.error('deleteHistoryItem:', e); }
    },

    // ─── Tag image preview ───

    tagImage: null,
    tagImagePos: {},

    showTagImage(event, tagName, isStatic = false) {
      if (!isStatic && !this.selectedPackId) return;
      _tagImgId++;
      const myId = _tagImgId;
      const el = event.currentTarget;
      const img = new Image();
      img.onload = () => {
        if (myId !== _tagImgId) return;
        if (img.naturalWidth <= 1) return;
        this.tagImage = img.src;
        const r = el.getBoundingClientRect();
        this.tagImagePos = { x: r.left, y: r.bottom + 4 };
      };
      img.onerror = () => {
        if (myId !== _tagImgId) return;
        this.tagImage = null;
      };
      if (isStatic) {
        img.src = `/api/static/image?tag=${encodeURIComponent(tagName)}`;
      } else if (this.selectedPackId) {
        img.src = `/api/tags/image?pack_id=${this.selectedPackId}&tag=${encodeURIComponent(tagName)}`;
      }
    },

    hideTagImage() {
      this.tagImage = null;
    },

    autoSavePrompt() {
      if (this._autoSaveTimer) clearTimeout(this._autoSaveTimer);
      this._autoSaveTimer = setTimeout(() => {
        const data = {
          positive_text: this.positivePrompt,
          negative_text: this.negativePrompt
        };
        try {
          localStorage.setItem('autosave_prompt', JSON.stringify(data));
          localStorage.setItem('autosave_chips', JSON.stringify({
            positiveChips: this.positiveChips,
            negativeChips: this.negativeChips
          }));
        } catch (_) {}
      }, 150);
    },

    loadAutoSave() {
      try {
        const chips = localStorage.getItem('autosave_chips');
        if (chips) {
          const data = JSON.parse(chips);
          if (data.positiveChips?.length || data.negativeChips?.length) {
            this.positiveChips = data.positiveChips || [];
            this.negativeChips = data.negativeChips || [];
            return;
          }
        }
      } catch(_) {}
      try {
        const raw = localStorage.getItem('autosave_prompt');
        if (raw) {
          const data = JSON.parse(raw);
          if (data.positive_text || data.negative_text) {
            const posTags = parsePromptData(data.positive_text || '');
            const negTags = parsePromptData(data.negative_text || '');
            for (const n of posTags) {
              const ch = { name: n, category: 'meta', subcategory: 'autosave', block_id: this.resolveBlockIdByName(n) };
              if (!this.positiveChips.some(c => c.name === n)) {
                this.positiveChips.push(ch);
              }
            }
            for (const n of negTags) {
              const ch = { name: n, category: 'meta', subcategory: 'autosave', block_id: this.resolveBlockIdByName(n) };
              if (!this.negativeChips.some(c => c.name === n)) {
                this.negativeChips.push(ch);
              }
            }
          }
        }
      } catch (_) {}
    },

    // ─── ComfyUI ───

    async loadComfyConfig() {
      try {
        const r = await fetch('/api/config');
        if (!r.ok) return;
        const c = await r.json();
        this.comfyEnabled = c.comfy_enabled;
        this.resolutions = (c.resolutions || 'Square 1:1#512x512').split('\n').map(s => s.trim()).filter(s => s.length > 0).map(s => {
          const parts = s.split('#');
          return parts.length === 2 ? { name: parts[0], dims: parts[1] } : { name: s, dims: s };
        });
        this.selectedResolution = this.resolutions[0]?.dims || '512x512';
      } catch(_) {}
    },

    async loadWorkflows() {
      try {
        const r = await fetch('/api/comfy/workflows');
        if (!r.ok) return;
        this.workflows = await r.json();
        if (this.workflows.length > 0) this.selectedWorkflow = this.workflows[0].name;
      } catch(e) { console.error('loadWorkflows:', e); }
    },

    async loadCheckpoints() {
      try {
        const r = await fetch('/api/comfy/object_info/CheckpointLoaderSimple');
        if (!r.ok) return;
        const data = await r.json();
        const ckpts = data?.CheckpointLoaderSimple?.input?.required?.ckpt_name?.[0];
        if (ckpts) {
          this.checkpoints = ckpts;
          if (ckpts.length > 0) this.selectedCheckpoint = ckpts[0];
        }
      } catch(e) { console.error('loadCheckpoints:', e); }
    },

    async loadSamplers() {
      try {
        const r = await fetch('/api/comfy/object_info/KSampler');
        if (!r.ok) return;
        const data = await r.json();
        const samplerList = data?.KSampler?.input?.required?.sampler_name?.[0];
        if (samplerList) { this.samplers = samplerList; this.selectedSampler = samplerList[0] || 'euler'; }
        const schedList = data?.KSampler?.input?.required?.scheduler?.[0];
        if (schedList) { this.schedulers = schedList; this.selectedScheduler = schedList[0] || 'normal'; }
      } catch(e) { console.error('loadSamplers:', e); }
    },

    async loadGenerationData() {
      if (this._genDataLoaded) return;
      this._genDataLoaded = true;
      await this.loadWorkflows();
      await this.loadNodeTitles();
      await this.loadCheckpoints();
      await this.loadSamplers();
      this.loadGenSettings();
    },

    async refreshGenerationData() {
      this._genDataLoaded = false;
      await this.loadGenerationData();
    },

    async loadNodeTitles() {
      if (!this.selectedWorkflow) return;
      try {
        const r = await fetch('/api/comfy/workflows?name=' + encodeURIComponent(this.selectedWorkflow));
        if (!r.ok) return;
        const data = await r.json();
        this.nodeTitles = {};
        for (const [id, node] of Object.entries(data)) {
          if (node._meta?.title) {
            this.nodeTitles[id] = node._meta.title;
          }
        }
      } catch(_) {}
    },

    async generate() {
      if (this.generating) return;
      if (!this.selectedWorkflow) { this.generationStatus = this.t('comfy.no_workflow'); return; }
      this.saveGenSettings();
      this.generating = true;
      this.generationProgress = 0;
      this.generationStatus = '';
      this.generationResult = null;

      if (!this.seedFixed) {
        this.seed = Math.floor(Math.random() * 2147483647);
      }

      // Auto-save prompt to history
      try {
        const now = new Date();
        const ts = now.toLocaleDateString() + ' ' + now.toLocaleTimeString();
        await fetch('/api/prompts', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: this.seed + ' - ' + ts,
            positive_text: this.positivePrompt,
            negative_text: this.negativePrompt,
            gen_data: JSON.stringify({
              workflow: this.selectedWorkflow,
              checkpoint: this.selectedCheckpoint,
              resolution: this.selectedResolution,
              sampler: this.selectedSampler,
              scheduler: this.selectedScheduler,
              steps: this.steps,
              cfg: this.cfg,
              seed: this.seed
            })
          })
        });
        this.loadHistory();
      } catch(_) {}

      const clientId = crypto.randomUUID();
      const parts = this.selectedResolution.split('x');
      const width = parts[0];
      const height = parts[1];

      const wsUrl = 'ws://' + window.location.host + '/api/comfy/ws?clientId=' + clientId;
      let ws;
      try { ws = new WebSocket(wsUrl); } catch (e) {
        this.generationStatus = this.t('comfy.error') + ': WebSocket ' + e.message;
        this.generating = false;
        return;
      }

      ws.onopen = () => {
        fetch('/api/comfy/generate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            client_id: clientId,
            workflow: this.selectedWorkflow,
            macros: {
              STEPS: String(this.steps),
              CFG: String(this.cfg),
              SAMPLER_NAME: this.selectedSampler,
              SCHEDULER: this.selectedScheduler,
              CKPT: this.selectedCheckpoint,
              SEED: String(this.seed),
              WIDTH: width,
              HEIGHT: height,
              PROMPT_POSITIVE: this.positivePrompt,
              PROMPT_NEGATIVE: this.negativePrompt
            }
          })
        }).then(r => {
          if (!r.ok) {
            r.json().then(d => {
              this.generationStatus = this.t('comfy.error') + ': ' + (d.error || r.status);
              ws.close();
            }).catch(() => {
              this.generationStatus = this.t('comfy.error') + ': ' + r.status;
              ws.close();
            });
          }
        }).catch(e => {
          this.generationStatus = this.t('comfy.error') + ': ' + e.message;
          ws.close();
        });
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === 'progress') {
            this.generationProgress = (msg.data.value / msg.data.max) * 100;
            this.generationStatus = msg.data.value + '/' + msg.data.max + ' (' + Math.round(this.generationProgress) + '%)';
          } else if (msg.type === 'executed' && msg.data?.output?.images?.length > 0) {
            const img = msg.data.output.images[0];
            const params = new URLSearchParams({
              filename: img.filename,
              subfolder: img.subfolder || '',
              type: img.type || 'output'
            });
            const url = '/api/comfy/image?' + params.toString();
            this.generationResult = url;
            this.generationHistory.unshift(url);
            if (this.generationHistory.length > 50) this.generationHistory.length = 50;
            this.saveGenerationHistory();
            this.generationProgress = 100;
            this.generationStatus = this.t('comfy.result') || 'Done';
            fetch('/api/comfy/save-image', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ filename: img.filename, subfolder: img.subfolder || '', type: img.type || 'output' })
            }).catch(() => {});
          } else if (msg.type === 'execution_error') {
            this.generationStatus = this.t('comfy.error') + ': ' + (msg.data?.exception_message || 'unknown');
            ws.close();
          } else if (msg.type === 'executing' && msg.data?.node === null) {
            ws.close();
          } else if (msg.type === 'executing' && msg.data?.node) {
            const name = this.nodeTitles[msg.data.node] || 'Node ' + msg.data.node;
            this.generationStatus = name + '...';
          }
        } catch(_) {}
      };

      ws.onerror = () => {
        this.generationStatus = this.t('comfy.error') + ': WebSocket';
        this.generating = false;
      };

      ws.onclose = () => {
        this.generating = false;
      };
    },

    // ─── Computed ───

    get positivePrompt() {
      const groups = [[], [], [], [], [], [], []];
      for (const ch of this.positiveChips) {
        const idx = Math.max(0, Math.min((ch.block_id || 4) - 1, 6));
        groups[idx].push(ch.name);
      }
      return groups.filter(g => g.length > 0).map(g => g.join(', ')).join(' BREAK ');
    },

    get selectedResolutionText() {
      const r = this.resolutions.find(x => x.dims === this.selectedResolution);
      return r ? r.name + ' - ' + r.dims : this.selectedResolution;
    },

    get negativePrompt() {
      return this.negativeChips.map(ch => ch.name).join(', ');
    },

    positiveByBlock(blockId) {
      return this.positiveChips.filter(c => (c.block_id || 4) === blockId);
    },

    // ─── Bulk ───

    get isDark() {
      if (this.theme === 'dark') return true;
      if (this.theme === 'light') return false;
      return window.matchMedia('(prefers-color-scheme: dark)').matches;
    },

    async loadAllTreeTags() {
      if (!this.selectedPackId || !this.tree.length) return;
      try {
        await Promise.all(this.tree.map(cat =>
          fetch(`/api/tags/tree?pack_id=${this.selectedPackId}&category=${encodeURIComponent(cat.name)}&offset=0&limit=99999`)
            .then(r => { if (!r.ok) throw new Error(r.status); return r.json(); })
            .then(page => {
              cat._tags = page.tags || [];
              for (const t of cat._tags) {
                this.tagInfoMap[t.tag_name] = { category: cat.name, subcategory: '' };
                this.tagToCategory[t.tag_name] = cat.name;
              }
            })
            .catch(e => console.error('loadTreeTags:', cat.name, e))
        ));
        this.enrichChips();
        this.updateChipNames();
      } catch(_) {}
    },

    async loadAll() {
      await this.loadTree();
      this.assignColors();
      this.loadFavorites();
      this.loadHistory();
      this.loadFavoritePrompts();
      this.loadAutoSave();
      this.enrichChips();
      this.updateChipNames();
      this.loadAllTreeTags();
    },

    get leftStyle() {
      return 'width:' + this.leftRatio + '%';
    },

    // ─── Resizer ───
    resizerStart(e, type) {
      this.resizing = type;
      const self = this;
      const body = document.body;
      body.classList.add(type === 'work' ? 'resizing-y' : 'resizing');
      const move = (e) => {
        if (!self.resizing) return;
        const rect = document.getElementById('layout-body').getBoundingClientRect();
        if (self.resizing === 'left') {
          self.leftRatio = Math.max(20, Math.min(80, Math.round((e.clientX - rect.left) / rect.width * 100)));
        } else if (self.resizing === 'right') {
          self.rightWidth = Math.max(300, Math.min(rect.width * 0.5, rect.right - e.clientX));
        } else if (self.resizing === 'work') {
          const pct = Math.round((e.clientY - rect.top) / rect.height * 100);
          const key = self.comfyEnabled ? 'workComfyRatio' : 'workNoComfyRatio';
          self[key] = Math.max(10, Math.min(90, pct));
        }
      };
      const up = () => {
        self.resizing = null;
        body.classList.remove('resizing', 'resizing-y');
        window.removeEventListener('mousemove', move);
        window.removeEventListener('mouseup', up);
        localStorage.setItem('layout_left', self.leftRatio);
        localStorage.setItem('layout_right', self.rightWidth);
        localStorage.setItem('layout_work_nc', self.workNoComfyRatio);
        localStorage.setItem('layout_work_c', self.workComfyRatio);
      };
      window.addEventListener('mousemove', move);
      window.addEventListener('mouseup', up);
      e.preventDefault();
    },

    // ─── Persistence ───
    loadGenSettings() {
      try {
        const d = JSON.parse(localStorage.getItem('gen_settings') || '{}');
        if (d.selectedWorkflow) this.selectedWorkflow = d.selectedWorkflow;
        if (d.selectedCheckpoint) this.selectedCheckpoint = d.selectedCheckpoint;
        if (d.selectedResolution) this.selectedResolution = d.selectedResolution;
        if (d.selectedSampler) this.selectedSampler = d.selectedSampler;
        if (d.selectedScheduler) this.selectedScheduler = d.selectedScheduler;
        if (d.steps) this.steps = d.steps;
        if (d.cfg) this.cfg = d.cfg;
        if (d.seed !== undefined) this.seed = d.seed;
        if (d.seedFixed !== undefined) this.seedFixed = d.seedFixed;
      } catch(_) {}
    },
    saveGenSettings() {
      try {
        localStorage.setItem('gen_settings', JSON.stringify({
          selectedWorkflow: this.selectedWorkflow,
          selectedCheckpoint: this.selectedCheckpoint,
          selectedResolution: this.selectedResolution,
          selectedSampler: this.selectedSampler,
          selectedScheduler: this.selectedScheduler,
          steps: this.steps, cfg: this.cfg,
          seed: this.seed, seedFixed: this.seedFixed
        }));
      } catch(_) {}
    },
    loadGenerationHistory() {
      try { this.generationHistory = JSON.parse(localStorage.getItem('gen_history') || '[]'); } catch(_) {}
    },
    saveGenerationHistory() {
      try { localStorage.setItem('gen_history', JSON.stringify(this.generationHistory)); } catch(_) {}
    },

    // ─── Viewer ───
    openViewer(index) {
      if (index < 0 || index >= this.generationHistory.length) return;
      this.viewerIndex = index;
      this.viewerImage = this.generationHistory[index];
    },
    closeViewer() {
      this.viewerImage = null;
      this.viewerIndex = -1;
    },
    prevImage() {
      if (!this.viewerImage || this.generationHistory.length < 2) return;
      this.viewerIndex = (this.viewerIndex - 1 + this.generationHistory.length) % this.generationHistory.length;
      this.viewerImage = this.generationHistory[this.viewerIndex];
    },
    nextImage() {
      if (!this.viewerImage || this.generationHistory.length < 2) return;
      this.viewerIndex = (this.viewerIndex + 1) % this.generationHistory.length;
      this.viewerImage = this.generationHistory[this.viewerIndex];
    },
  };
}

function promptPreview(text) {
  if (!text) return '';
  const parts = text.split('BREAK').map(s => s.trim()).filter(s => s.length > 0);
  const result = parts.join(' | ');
  return result.substring(0, 100) + (result.length > 100 ? '...' : '');
}

function parsePromptData(text) {
  if (!text) return [];
  return text.split(/BREAK|,\s*/).map(s => s.trim()).filter(s => s.length > 0);
}
