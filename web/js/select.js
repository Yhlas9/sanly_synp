// Öz saýlaw meýdançamyz. Brauzeriň <select> penjiresi her ulgamda başgaça
// görünýär we bezegi üýtgedip bolmaýar — şonuň üçin ony çalyşýarys.
// Klawiatura bilen doly işleýär (9-njy bölümiň talaby).

export const TkSelect = {
  name: 'TkSelect',
  props: {
    modelValue: { default: null },
    // options: [{ value, label, hint?, badge? }]
    options: { type: Array, default: () => [] },
    placeholder: { type: String, default: '— saýlaň —' },
    id: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
  },
  emits: ['update:modelValue', 'change'],

  data() {
    return { open: false, cursor: -1 };
  },

  computed: {
    selected() {
      return this.options.find((o) => this.same(o.value, this.modelValue)) || null;
    },
    label() { return this.selected ? this.selected.label : this.placeholder; },
  },

  methods: {
    // null we undefined deň hasaplanýar: Vue-de boş saýlaw iki hili gelip bilýär.
    same(a, b) {
      if (a === null || a === undefined) return b === null || b === undefined;
      return String(a) === String(b);
    },

    toggle() {
      if (this.disabled) return;
      this.open = !this.open;
      if (this.open) {
        this.cursor = this.options.findIndex((o) => this.same(o.value, this.modelValue));
        this.$nextTick(() => this.scrollToCursor());
      }
    },

    pick(opt) {
      this.$emit('update:modelValue', opt.value);
      this.$emit('change', opt.value);
      this.open = false;
      this.$refs.trigger?.focus();
    },

    move(step) {
      if (!this.open) { this.toggle(); return; }
      const n = this.options.length;
      if (!n) return;
      this.cursor = (this.cursor + step + n) % n;
      this.scrollToCursor();
    },

    scrollToCursor() {
      const el = this.$refs.list?.children[this.cursor];
      el?.scrollIntoView({ block: 'nearest' });
    },

    onKey(e) {
      switch (e.key) {
        case 'ArrowDown': e.preventDefault(); this.move(1); break;
        case 'ArrowUp': e.preventDefault(); this.move(-1); break;
        case 'Home': if (this.open) { e.preventDefault(); this.cursor = 0; this.scrollToCursor(); } break;
        case 'End': if (this.open) { e.preventDefault(); this.cursor = this.options.length - 1; this.scrollToCursor(); } break;
        case 'Enter':
        case ' ':
          e.preventDefault();
          if (this.open && this.cursor >= 0) this.pick(this.options[this.cursor]);
          else this.toggle();
          break;
        case 'Escape': if (this.open) { e.preventDefault(); this.open = false; } break;
        case 'Tab': this.open = false; break;
        default: break;
      }
    },

    onOutside(e) {
      if (this.open && !this.$el.contains(e.target)) this.open = false;
    },
  },

  mounted() { document.addEventListener('click', this.onOutside, true); },
  unmounted() { document.removeEventListener('click', this.onOutside, true); },

  template: `
<div class="sel" :class="{ open, disabled }">
  <button type="button" class="sel-field" ref="trigger" :id="id"
          :disabled="disabled" @click="toggle" @keydown="onKey"
          role="combobox" :aria-expanded="open" aria-haspopup="listbox">
    <span class="sel-text" :class="{ muted: !selected }">{{ label }}</span>
    <span v-if="selected && selected.badge" class="badge">{{ selected.badge }}</span>
    <span class="sel-caret" aria-hidden="true"></span>
  </button>

  <ul v-if="open" class="sel-list" ref="list" role="listbox" :aria-activedescendant="id + '-o' + cursor">
    <li v-for="(o, i) in options" :key="i" :id="id + '-o' + i"
        class="sel-opt"
        :class="{ cur: i === cursor, sel: same(o.value, modelValue) }"
        role="option" :aria-selected="same(o.value, modelValue)"
        @click="pick(o)" @mouseenter="cursor = i">
      <span class="sel-mark" aria-hidden="true"></span>
      <span class="sel-body">
        <span class="sel-label">{{ o.label }}</span>
        <small v-if="o.hint">{{ o.hint }}</small>
      </span>
      <span v-if="o.badge" class="badge">{{ o.badge }}</span>
    </li>
    <li v-if="!options.length" class="sel-empty">Saýlar zat ýok</li>
  </ul>
</div>`,
};
