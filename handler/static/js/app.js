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
      this.tagBlockMap = {};
      try {
        const res = await fetch('/static/constants.json');
        this.constantTags = await res.json();
        const blockIds = { 'quality': 1, 'sources': 2, 'rating': 3, 'pose': 5, 'scene': 6, 'style': 7 };
        for (const group of this.constantTags) {
          const tkey = group.tkey || '';
          const parts = tkey.split('.');
          const cat = parts.length >= 2 ? parts[1] : '';
          const blockId = blockIds[cat];
          if (!blockId) continue;
          if (group.tags) {
            for (const tag of group.tags) this.tagBlockMap[tag] = blockId;
          }
          if (group.subcategories) {
            for (const sub of group.subcategories) {
              if (sub.tags) {
                for (const tag of sub.tags) this.tagBlockMap[tag] = blockId;
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

    // Constant tags
    constOpen: {},
    constSubOpen: {},
    constantTags: [],
    tagBlockMap: {},

    // Favorites
    favorites: [],
    favOpen: false,

    // Chips as objects: { name, category, subcategory, block_id }
    positiveChips: [],
    negativeChips: [],
    dragState: null,
    dropTarget: null,

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

    resolveBlockId(category, subcategory) {
      if (category === 'const') {
        const map = { 'quality': 1, 'sources': 2, 'rating': 3, 'pose': 5, 'scene': 6, 'style': 7 };
        return map[subcategory] || 4;
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
      const arr = type === 'positive' ? this.positiveChips : this.negativeChips;
      const idx = arr.findIndex(c => c.name === name);
      if (idx !== -1) arr.splice(idx, 1);
      this.updateChipNames();
      this.autoSavePrompt();
    },

    getChipIndex(type, name) {
      const arr = type === 'positive' ? this.positiveChips : this.negativeChips;
      return arr.findIndex(c => c.name === name);
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
      const name = this.dragState?.name;
      if (!name) return;
      const srcArr = this.positiveChips.find(c => c.name === name) ? this.positiveChips : this.negativeChips;
      const tgtArr = targetType === 'positive' ? this.positiveChips : this.negativeChips;
      const chip = srcArr.find(c => c.name === name);
      if (!chip) return;
      const targetBlockId = parseInt(ev.currentTarget.dataset.blockId) || 4;
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
      const ch = { name, category: 'custom', subcategory: '', block_id: 4 };
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
      this.drawerOpen = false;
      this.updateChipNames();
      this.autoSavePrompt();
    },

    // ─── Tag image preview ───

    tagImage: null,
    tagImagePos: {},
    _tagImgId: 0,

    showTagImage(event, tagName, isStatic = false) {
      if (!isStatic && !this.selectedPackId) return;
      this._tagImgId++;
      const myId = this._tagImgId;
      const el = event.currentTarget;
      const rect = el.getBoundingClientRect();
      const img = new Image();
      img.onload = () => {
        if (myId !== this._tagImgId) return;
        if (img.naturalWidth <= 1) return;
        this.tagImage = img.src;
        this.tagImagePos = { x: rect.left, y: rect.bottom + 4 };
      };
      img.onerror = () => {
        if (myId !== this._tagImgId) return;
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

    // ─── Computed ───

    get positivePrompt() {
      const groups = [[], [], [], [], [], [], []];
      for (const ch of this.positiveChips) {
        const idx = (ch.block_id || 4) - 1;
        if (idx >= 0 && idx < 7) groups[idx].push(ch.name);
      }
      return groups.filter(g => g.length > 0).map(g => g.join(', ')).join(' BREAK ');
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
