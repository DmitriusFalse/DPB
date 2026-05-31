function app() {
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
      try {
        const res = await fetch('/static/constants.json');
        this.constantTags = await res.json();
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

    // Constant tags
    constOpen: {},
    constSubOpen: {},
    constantTags: [],

    // Favorites
    favorites: [],
    favOpen: false,

    // Chips as objects: { name, category, subcategory }
    positiveChips: [],
    negativeChips: [],

    // Custom input
    customTag: '',

    // Presets
    presetData: {},

    // Drawer
    drawerOpen: false,
    drawerTab: 'history',
    history: [],
    favoritePrompts: [],

    // Save modal
    saveModalOpen: false,
    savePromptName: '',

    // ─── Init ───

    init() {
      this.loadTranslations();
      this.loadPresets();
      this.loadConstants();
      this.updateChipNames();
      this.loadPacks();
      document.addEventListener('pwa-installable', () => {
        this.pwaInstallable = true;
      });
    },

    // ─── Packs ───

    async loadPacks() {
      try {
        const res = await fetch('/api/packs');
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
          this.loadAll();
        }
      } catch (e) {
        console.error('loadPacks:', e);
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
        this.tree = await res.json();
        this.treeOpen = {};
      } catch (e) {
        console.error('loadTree:', e);
      }
    },

    async toggleCategory(cat) {
      const name = typeof cat === 'string' ? cat : cat.name;
      this.treeOpen[name] = !this.treeOpen[name];
      if (this.treeOpen[name]) {
        const treeCat = this.tree.find(c => c.name === name);
        if (treeCat && !treeCat._tags) {
          this.treeModal = true;
          this.treeModalProgress = 0;
          try {
            const res = await fetch(`/api/tags/tree?pack_id=${this.selectedPackId}&category=${encodeURIComponent(name)}&offset=0&limit=99999`);
            const page = await res.json();
            treeCat._tags = page.tags || [];
            this.treeModalProgress = 100;
          } catch (e) {
            console.error('toggleCategory:', e);
            treeCat._tags = [];
          } finally {
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
        this.searchResults = await res.json();
      } catch (e) {
        console.error('search:', e);
      }
    },

    // ─── Chips ───

    makeChip(tag) {
      return {
        name: tag.tag_name,
        category: tag.category_name || '',
        subcategory: tag.subcategory_name || '',
      };
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

    removeChip(type, idx) {
      if (type === 'positive') {
        this.positiveChips.splice(idx, 1);
      } else {
        this.negativeChips.splice(idx, 1);
      }
      this.updateChipNames();
      this.autoSavePrompt();
    },

    clearAll() {
      this.positiveChips.splice(0);
      this.negativeChips.splice(0);
      this.updateChipNames();
      this.autoSavePrompt();
    },

    // ─── Custom tag ───

    addCustomTag(negative) {
      const name = this.customTag.trim();
      if (!name) return;
      const ch = { name, category: 'custom', subcategory: '' };
      if (negative) {
        const posIdx = this.positiveChips.findIndex(c => c.name === name);
        if (posIdx !== -1) this.positiveChips.splice(posIdx, 1);
        if (!this.negativeChips.some(c => c.name === name)) {
          this.negativeChips.push(ch);
        }
      } else {
        if (!this.positiveChips.some(c => c.name === name)) {
          this.positiveChips.push(ch);
        }
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

    updateChipNames() {
      const p = {}, n = {};
      for (const c of this.positiveChips) p[c.name] = true;
      for (const c of this.negativeChips) n[c.name] = true;
      this.posNames = p;
      this.negNames = n;
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
          const ch = { name: n, category: 'meta', subcategory: sub };
          if (!this.positiveChips.some(c => c.name === ch.name)) {
            this.positiveChips.push(ch);
          }
        }
      }

      for (const n of data.negative) {
        const ch = { name: n, category: 'meta', subcategory: 'general' };
        if (!this.negativeChips.some(c => c.name === ch.name)) {
          this.negativeChips.push(ch);
        }
      }
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

    async savePrompt() {
      this.savePromptName = '';
      this.saveModalOpen = true;
    },

    async doSavePrompt() {
      if (!this.savePromptName.trim()) return;
      try {
        await fetch('/api/prompts', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: this.savePromptName.trim(),
            positive_text: this.positivePrompt,
            negative_text: this.negativePrompt
          })
        });
        this.saveModalOpen = false;
        this.loadHistory();
      } catch (e) {
        console.error('savePrompt:', e);
      }
    },

    async loadHistory() {
      try {
        const res = await fetch('/api/prompts?favorites=0');
        this.history = await res.json();
      } catch (e) {
        console.error('loadHistory:', e);
      }
    },

    async loadFavoritePrompts() {
      try {
        const res = await fetch('/api/prompts?favorites=1');
        this.favoritePrompts = await res.json();
      } catch (e) {
        console.error('loadFavPrompts:', e);
      }
    },

    async copyTagName(name) {
      if (!name) return;
      try {
        await navigator.clipboard.writeText(name);
        this.showToast((this.t('toast.copied') || 'Скопировано: ') + name);
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
      this.clearAll();
      const posParser = parsePromptData(p.positive_text);
      for (const n of posParser) {
        const ch = { name: n, category: 'meta', subcategory: 'loaded' };
        if (!this.positiveChips.some(c => c.name === n)) {
          this.positiveChips.push(ch);
        }
      }
      const negParser = parsePromptData(p.negative_text);
      for (const n of negParser) {
        const ch = { name: n, category: 'meta', subcategory: 'loaded' };
        if (!this.negativeChips.some(c => c.name === n)) {
          this.negativeChips.push(ch);
        }
      }
      this.drawerOpen = false;
      this.updateChipNames();
      this.autoSavePrompt();
    },

    // ─── Tag image preview ───

    tagImage: null,
    tagImagePos: {},
    _tagImgId: 0,

    showTagImage(event, tagName) {
      if (!this.selectedPackId) return;
      this._tagImgId++;
      const myId = this._tagImgId;
      const img = new Image();
      img.onload = () => {
        if (myId !== this._tagImgId) return;
        if (img.naturalWidth <= 1) return;
        this.tagImage = img.src;
        this.tagImagePos = { x: event.clientX + 15, y: event.clientY + 15 };
      };
      img.onerror = () => {
        if (myId !== this._tagImgId) return;
        this.tagImage = null;
      };
      img.src = `/api/tags/image?pack_id=${this.selectedPackId}&tag=${encodeURIComponent(tagName)}`;
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
        } catch (_) {}
      }, 150);
    },

    loadAutoSave() {
      try {
        const raw = localStorage.getItem('autosave_prompt');
        if (raw) {
          const data = JSON.parse(raw);
          if (data.positive_text || data.negative_text) {
            const posTags = parsePromptData(data.positive_text || '');
            const negTags = parsePromptData(data.negative_text || '');
            for (const n of posTags) {
              const ch = { name: n, category: 'meta', subcategory: 'autosave' };
              if (!this.positiveChips.some(c => c.name === n)) {
                this.positiveChips.push(ch);
              }
            }
            for (const n of negTags) {
              const ch = { name: n, category: 'meta', subcategory: 'autosave' };
              if (!this.negativeChips.some(c => c.name === n)) {
                this.negativeChips.push(ch);
              }
            }
          }
        }
      } catch (_) {}
    },

    // ─── Computed ───

    get positivePrompt() {
      const groups = [[], [], [], [], [], [], []];
      for (const ch of this.positiveChips) {
        const idx = this.blockIndex(ch.category);
        if (idx >= 0 && idx < 7) groups[idx].push(ch.name);
      }
      return groups.filter(g => g.length > 0).map(g => g.join(', ')).join(' BREAK ');
    },

    get negativePrompt() {
      const groups = [[], [], []];
      for (const ch of this.negativeChips) {
        const idx = this.negBlockIndex(ch.category);
        if (idx >= 0 && idx < 3) groups[idx].push(ch.name);
      }
      return groups.filter(g => g.length > 0).map(g => g.join(', ')).join(' BREAK ');
    },

    blockIndex(category) {
      const map = {
        'artist': 0, 'artist_groups': 0, 'individual_artists': 0,
        'copyright': 1, 'game': 1, 'series': 1, 'games': 1, 'anime': 1,
        'character': 2, 'characters': 2,
        'general': 3,
        'species': 4,
        'rating': 5,
        'quality_resolution': 6, 'meta': 6
      };
      return map[category] ?? 3;
    },

    negBlockIndex(category) {
      const map = {
        'general': 0, 'species': 0, 'character': 0, 'characters': 0,
        'quality_resolution': 1, 'meta': 1,
        'rating': 2
      };
      return map[category] ?? 0;
    },

    // ─── Bulk ───

    get isDark() {
      if (this.theme === 'dark') return true;
      if (this.theme === 'light') return false;
      return window.matchMedia('(prefers-color-scheme: dark)').matches;
    },

    loadAll() {
      this.loadTree();
      this.loadFavorites();
      this.loadHistory();
      this.loadFavoritePrompts();
      this.loadAutoSave();
    }
  };
}

function promptPreview(text) {
  if (!text) return '';
  const parts = text.split('BREAK').map(s => s.trim()).filter(s => s.length > 0);
  return parts.join(' | ').substring(0, 100) + (text.length > 100 ? '...' : '');
}

function parsePromptData(text) {
  if (!text) return [];
  return text.split(/BREAK|,/).map(s => s.trim()).filter(s => s.length > 0);
}
