function app() {
  return {
    // Theme: 'auto', 'dark', 'light'
    theme: localStorage.getItem('theme') || 'auto',

    // i18n
    lang: localStorage.getItem('lang') || (navigator.language || '').startsWith('ru') ? 'ru' : 'en',
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

    // Search
    searchQuery: '',
    searchResults: [],

    // Tree
    tree: [],
    treeOpen: {},
    subOpen: {},
    treeModal: false,
    treeModalProgress: 0,

    // Constant tags
    constOpen: {},
    constSubOpen: {},
    constantTags: [
      {
        name: 'Теги качества',
        tkey: 'const.quality',
        tags: ['score_9', 'score_8_up', 'score_7_up', 'score_6_up', 'score_5_up', 'score_4_up', 'score_6', 'score_5', 'score_4']
      },
      {
        name: 'Теги источники',
        tkey: 'const.sources',
        tags: ['source_anime', 'source_cartoon', 'source_furry', 'source_pony', 'source_rule34']
      },
      {
        name: 'Теги Рейтинга',
        tkey: 'const.rating',
        subcat: 'rating',
        tags: ['rating_safe', 'rating_questionable', 'rating_explicit']
      },
      {
        name: 'Позы и действия',
        tkey: 'const.pose',
        subcategories: [
          { name: 'Standing', tags: ['standing', 'leaning', 'leaning_against_wall', 'leaning_forward', 'leaning_back', 'contrapposto', 'hands_on_hips', 'hand_on_hip', 'arms_crossed', 'arms_up', 'arms_behind_back', 'arms_behind_head', 'hand_on_face', 'hand_on_chin', 'hand_on_chest', 'hand_in_hair', 'hand_in_pocket', 'peace_sign', 'finger_to_lips'] },
          { name: 'Sitting', tags: ['sitting', 'sitting_on_chair', 'sitting_on_floor', 'sitting_on_bed', 'sitting_on_edge', 'seiza', 'indian_style', 'crossed_legs', 'legs_together', 'one_knee_up', 'hugging_knees', 'straddling', 'lap_sitting'] },
          { name: 'Lying', tags: ['lying_down', 'lying_on_back', 'lying_on_side', 'lying_on_stomach', 'prone', 'supine', 'fetal_position', 'sprawled'] },
          { name: 'Other poses', tags: ['kneeling', 'crouching', 'squatting', 'bending_over', 'on_all_fours', 'crawling', 'stretching', 'arching_back', 'twisting', 'turning_around', 'spread_legs', 'legs_apart'] },
          { name: 'Actions', tags: ['walking', 'running', 'jumping', 'dancing', 'fighting', 'eating', 'drinking', 'reading', 'sleeping', 'bathing', 'swimming', 'holding_phone', 'holding_cup', 'typing', 'pointing', 'waving', 'hugging', 'kissing', 'posing', 'modeling', 'working_out', 'yoga', 'smoking'] },
          { name: 'Head & gaze', tags: ['head_tilt', 'head_turn', 'looking_at_viewer', 'looking_away', 'looking_back', 'looking_up', 'looking_down', 'looking_to_the_side', 'over_shoulder', 'profile', 'three-quarter_view', 'eye_contact', 'averting_eyes'] },
          { name: 'NSFW — Pose / Action', tags: ['sex', 'penetration', 'vaginal', 'anal', 'oral', 'blowjob', 'deepthroat', 'handjob', 'fingering', 'masturbation', 'riding', 'missionary', 'doggystyle', 'cowgirl', 'reverse_cowgirl', 'standing_sex', 'wall_sex', 'shower_sex', 'tribadism', 'sixty-nine', 'cum', 'cumshot', 'cum_in_pussy', 'cum_on_face', 'cum_on_body', 'cum_on_breasts', 'creampie', 'facial', 'spread_pussy', 'presenting', 'ass_up', 'face_down_ass_up', 'bent_over', 'grinding', 'dry_humping', 'lap_dance', 'striptease', 'groping', 'fondling', 'aroused', 'orgasm', 'wet', 'sweat', 'sweaty', 'afterglow', 'post-coital', 'submissive', 'dominant', 'lustful', 'willing', 'reluctant'] }
        ]
      },
      {
        name: 'Теги сцены и настройки',
        tkey: 'const.scene',
        subcategories: [
          { name: 'Location type', tags: ['indoors', 'outdoors', 'urban', 'rural', 'suburban', 'underwater', 'space'] },
          { name: 'Indoor', tags: ['bedroom', 'bathroom', 'kitchen', 'living_room', 'library', 'office', 'studio', 'classroom', 'gym', 'pool', 'locker_room', 'bar', 'restaurant', 'cafe', 'hotel_room', 'hospital', 'elevator', 'staircase', 'hallway', 'balcony', 'sauna', 'dressing_room'] },
          { name: 'Furniture / Props', tags: ['bed', 'couch', 'sofa', 'chair', 'desk', 'table', 'bathtub', 'shower', 'window', 'doorway', 'mirror', 'curtains', 'pillows', 'blanket', 'rug', 'bookshelf'] },
          { name: 'Outdoor', tags: ['beach', 'tropical_beach', 'forest', 'woods', 'park', 'garden', 'flower_garden', 'street', 'alley', 'rooftop', 'bridge', 'pier', 'mountain', 'cliff', 'lake', 'river', 'ocean', 'waterfall', 'field', 'meadow', 'desert', 'cave', 'island'] },
          { name: 'Urban', tags: ['cityscape', 'skyline', 'downtown', 'skyscraper', 'neon_city', 'market', 'shopping_mall', 'subway', 'train_station', 'parking_lot'] },
          { name: 'Fantasy', tags: ['fantasy', 'sci-fi', 'cyberpunk', 'steampunk', 'medieval', 'futuristic', 'post-apocalyptic', 'castle', 'palace', 'throne_room', 'dungeon', 'cathedral', 'temple', 'ruins', 'magical', 'spaceship'] },
          { name: 'Time / Weather', tags: ['day', 'night', 'sunset', 'sunrise', 'dawn', 'dusk', 'golden_hour', 'blue_hour', 'twilight', 'overcast', 'cloudy', 'clear_sky', 'starry_sky', 'moonlight', 'full_moon', 'rain', 'snow', 'fog', 'mist', 'storm', 'cherry_blossoms', 'autumn_leaves', 'spring', 'summer', 'winter'] }
        ]
      },
      {
        name: 'Теги стиля и освещения',
        tkey: 'const.style',
        subcategories: [
          { name: 'Lighting', tags: ['natural_lighting', 'soft_lighting', 'hard_lighting', 'dramatic_lighting', 'volumetric_lighting', 'backlighting', 'rim_lighting', 'studio_lighting', 'cinematic_lighting', 'sunlight', 'dappled_sunlight', 'moonlight', 'candlelight', 'firelight', 'neon_lights', 'ambient_lighting', 'harsh_lighting', 'silhouette', 'god_rays', 'spotlight', 'colored_lighting', 'warm_lighting', 'cool_lighting', 'chiaroscuro', 'rembrandt_lighting'] },
          { name: 'Render quality', tags: ['photorealistic', 'realistic', 'hyper_realistic', 'ultra_realistic', 'detailed', 'highly_detailed', 'intricate_details', 'sharp_focus', '8k', '4k', 'hdr', 'vibrant_colors', 'rich_colors', 'pastel_colors', 'muted_colors', 'monochrome', 'sepia', 'high_contrast', 'low_contrast', 'dark_theme'] },
          { name: 'Art style', tags: ['digital_art', 'digital_painting', 'concept_art', 'illustration', 'painting', 'oil_painting', 'watercolor', 'cel_shading', 'flat_color', 'lineart', 'sketch', 'anime_style', 'manga_style', 'comic_style', '3d_render', 'photo', 'photograph', 'fashion_photography', 'portrait_photography', 'editorial', 'magazine_cover', 'art_nouveau', 'impressionist', 'surreal', 'minimalist', 'baroque', 'renaissance'] },
          { name: 'Mood', tags: ['cinematic', 'epic', 'peaceful', 'serene', 'melancholic', 'mysterious', 'romantic', 'intimate', 'sensual', 'eerie', 'whimsical', 'nostalgic', 'vintage', 'retro', 'dreamy', 'ethereal', 'gritty', 'elegant', 'cozy', 'warm_atmosphere', 'cold_atmosphere', 'dark_atmosphere', 'moody', 'atmospheric'] },
          { name: 'Background', tags: ['simple_background', 'white_background', 'black_background', 'gradient_background', 'detailed_background', 'blurry_background', 'scenery', 'landscape', 'symmetry', 'rule_of_thirds', 'framing', 'negative_space', 'centered', 'dynamic_composition'] },
          { name: 'Detail quality', tags: ['detailed_skin', 'skin_texture', 'skin_pores', 'smooth_skin', 'glowing_skin', 'subsurface_scattering', 'detailed_hands', 'detailed_fingers', 'perfect_hands', 'detailed_hair', 'individual_hair_strands', 'detailed_clothing', 'fabric_texture'] }
        ]
      },
      {
        name: 'Негативные теги',
        tkey: 'const.negative',
        subcategories: [
          { name: 'Quality', tags: ['ugly', 'deformed', 'blurry', 'lowres', 'worst_quality', 'low_quality', 'normal_quality', 'jpeg_artifacts', 'pixelated', 'oversaturated', 'underexposed', 'overexposed', 'washed_out'] },
          { name: 'Anatomy', tags: ['bad_anatomy', 'bad_hands', 'bad_fingers', 'bad_feet', 'missing_fingers', 'extra_digit', 'fewer_digits', 'extra_limb', 'missing_limbs', 'fused_fingers', 'too_many_fingers', 'malformed_hands', 'unnatural_face', 'unnatural_body', 'imperfect_eyes', 'skewed_eyes', 'cross-eyed', 'extra_arms', 'extra_legs', 'extra_heads', 'mutated', 'disfigured', 'disproportionate', 'long_neck', 'bad_proportions'] },
          { name: 'Artifacts', tags: ['text', 'error', 'signature', 'watermark', 'username', 'cropped', 'border', 'frame', 'letterbox', 'artifacts', 'noise', 'chromatic_aberration'] },
          { name: 'Anti-style', tags: ['sketch', 'painting', 'drawing', '3d', 'cgi', 'render', 'cartoon', 'anime', 'comic', 'manga', 'low_detail', 'flat', 'simple', 'amateur', 'stock_photo'] }
        ]
      }
    ],

    // Favorites
    favorites: [],
    favOpen: false,

    // Chips as objects: { name, category, subcategory }
    positiveChips: [],
    negativeChips: [],

    // Custom input
    customTag: '',

    // Presets (hardcoded)
    presetData: {
      'Anime': {
        positive: ['score_9', 'score_8_up', 'score_7_up', 'score_6_up', 'score_5_up', 'BREAK', '1girl', 'solo', 'anime_style', 'manga', 'anime_coloring', 'japanese_art'],
        negative: ['western', 'cartoon', 'realistic', 'photorealistic']
      },
      'Cartoon': {
        positive: ['score_9', 'score_8_up', 'score_7_up', 'score_6_up', 'score_5_up', 'BREAK', 'cartoon', 'cartoon_style', 'disney', 'western', 'comic', 'toon'],
        negative: ['realistic', 'photorealistic', 'anime']
      },
      'Realistic': {
        positive: ['score_9', 'score_8_up', 'score_7_up', 'score_6_up', 'score_5_up', 'BREAK', 'photorealistic', 'realistic', 'photo', 'photography', 'detailed', 'sharp_focus', 'high_res'],
        negative: ['cartoon', 'anime', 'sketch', 'lineart']
      },
      'Portrait': {
        positive: ['score_9', 'score_8_up', 'score_7_up', 'score_6_up', 'score_5_up', 'BREAK', 'portrait', 'face', 'close-up', 'upper_body', 'looking_at_viewer', 'detailed_face', 'expression'],
        negative: ['full_body', 'wide_shot', 'landscape']
      },
      'Outdoor': {
        positive: ['score_9', 'score_8_up', 'score_7_up', 'score_6_up', 'score_5_up', 'BREAK', 'outdoor', 'nature', 'sky', 'clouds', 'sunlight', 'scenery', 'landscape', 'tree'],
        negative: ['indoor', 'wall', 'dark', 'night']
      },
      'Minimal': {
        positive: ['score_9', 'score_8_up', 'score_7_up', 'BREAK', 'minimalist', 'simple_background', 'white_background', 'plain_background', 'monochrome'],
        negative: ['detailed_background', 'complex', 'crowded']
      },
      'Quality Only': {
        positive: ['score_9', 'score_8_up', 'score_7_up', 'score_6_up', 'score_5_up', 'BREAK', 'best_quality', 'high_quality', 'high_res', 'masterpiece', 'detailed'],
        negative: ['low_quality', 'worst_quality', 'bad_art']
      }
    },

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
      this.updateChipNames();
      this.loadPacks();
    },

    // ─── Packs ───

    async loadPacks() {
      try {
        const res = await fetch('/api/packs');
        this.packs = await res.json();
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

    async toggleSubcategory(cat, sub) {
      const key = cat.name + '_' + sub.name;
      if (!this.subOpen[key]) {
        this.subOpen[key] = true;
        if (!sub._tags) {
          this.treeModal = true;
          this.treeModalProgress = 0;
          sub._tags = [];
          try {
            const res = await fetch(`/api/tags/tree?pack_id=${this.selectedPackId}&category=${encodeURIComponent(cat.name)}&subcategory=${encodeURIComponent(sub.name)}&offset=0&limit=99999`);
            const page = await res.json();
            this.treeModalProgress = 5;
            await this.renderTagProgressive(sub, page.tags);
          } catch (e) {
            console.error('toggleSubcategory:', e);
          } finally {
            this.treeModal = false;
          }
        }
      } else {
        this.subOpen[key] = false;
      }
    },

    renderTagProgressive(sub, allTags) {
      return new Promise(resolve => {
        const total = allTags.length;
        if (total === 0) { resolve(); return; }
        let i = 0;
        const batchSize = 500;
        const step = () => {
          if (i >= total) { resolve(); return; }
          const batch = allTags.slice(i, i + batchSize);
          sub._tags.push(...batch);
          i += batchSize;
          this.treeModalProgress = 5 + Math.round(i / total * 95);
          requestAnimationFrame(step);
        };
        requestAnimationFrame(step);
      });
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
