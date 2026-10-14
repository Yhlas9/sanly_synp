// Türkmençe sene we wagt saýlaýjy.
// Brauzeriň öz <input type="datetime-local"> penjiresi türkmen diline
// terjime edilmeýär — ol ulanyjynyň ulgam dilinde açylýar. Şonuň üçin
// öz saýlaýjymyzy ulanýarys.

const MONTHS = ['Ýanwar', 'Fewral', 'Mart', 'Aprel', 'Maý', 'Iýun',
  'Iýul', 'Awgust', 'Sentýabr', 'Oktýabr', 'Noýabr', 'Dekabr'];

import { TkSelect } from './select.js';

// Hepde duşenbeden başlaýar.
const WEEKDAYS = ['Du', 'Si', 'Ça', 'Pe', 'An', 'Şe', 'Ýe'];

const pad = (n) => String(n).padStart(2, '0');

// Giriş we çykyş görnüşi: "YYYY-MM-DDTHH:mm" — ýerli wagt, sagat guşaklygy ýok.
function toValue(d) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
    + `T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function parseValue(v) {
  if (!v) return null;
  const m = /^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/.exec(v);
  if (!m) return null;
  return new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5]);
}

export const TkDateTime = {
  name: 'TkDateTime',
  components: { TkSelect },
  props: {
    modelValue: { type: String, default: '' },
    id: { type: String, default: '' },
  },
  emits: ['update:modelValue'],

  data() {
    const now = new Date();
    const cur = parseValue(this.modelValue);
    return {
      open: false,
      viewYear: (cur || now).getFullYear(),
      viewMonth: (cur || now).getMonth(),
      hour: cur ? cur.getHours() : 18,
      minute: cur ? cur.getMinutes() : 0,
      MONTHS, WEEKDAYS,
    };
  },

  computed: {
    picked() { return parseValue(this.modelValue); },

    label() {
      const d = this.picked;
      if (!d) return 'saýlanmadyk';
      return `${d.getDate()} ${MONTHS[d.getMonth()]} ${d.getFullYear()}, `
        + `${pad(d.getHours())}:${pad(d.getMinutes())}`;
    },

    hourOptions() {
      return Array.from({ length: 24 }, (_, h) => ({ value: h, label: String(h).padStart(2, '0') }));
    },
    minuteOptions() {
      return [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55]
        .map((m) => ({ value: m, label: String(m).padStart(2, '0') }));
    },

    // Aýyň tory: duşenbeden başlaýan 6 hepde.
    weeks() {
      const first = new Date(this.viewYear, this.viewMonth, 1);
      const shift = (first.getDay() + 6) % 7; // ýekşenbe = 0 → 6
      const start = new Date(this.viewYear, this.viewMonth, 1 - shift);

      const out = [];
      for (let w = 0; w < 6; w++) {
        const row = [];
        for (let i = 0; i < 7; i++) {
          const d = new Date(start);
          d.setDate(start.getDate() + w * 7 + i);
          row.push(d);
        }
        out.push(row);
      }
      return out;
    },
  },

  methods: {
    isOther(d) { return d.getMonth() !== this.viewMonth; },

    isToday(d) {
      const n = new Date();
      return d.toDateString() === n.toDateString();
    },

    isPicked(d) {
      return this.picked && d.toDateString() === this.picked.toDateString();
    },

    shiftMonth(step) {
      const d = new Date(this.viewYear, this.viewMonth + step, 1);
      this.viewYear = d.getFullYear();
      this.viewMonth = d.getMonth();
    },

    choose(d) {
      const out = new Date(d);
      out.setHours(this.hour, this.minute, 0, 0);
      this.$emit('update:modelValue', toValue(out));
      this.viewYear = out.getFullYear();
      this.viewMonth = out.getMonth();
    },

    setTime() {
      this.hour = Number(this.hour);
      this.minute = Number(this.minute);
      const base = this.picked || new Date();
      const out = new Date(base);
      out.setHours(this.hour, this.minute, 0, 0);
      this.$emit('update:modelValue', toValue(out));
    },

    // Iň köp ulanylýan möhletler — bir basyşda.
    preset(kind) {
      const d = new Date();
      if (kind === 'today') d.setHours(23, 59, 0, 0);
      if (kind === 'tomorrow') { d.setDate(d.getDate() + 1); d.setHours(18, 0, 0, 0); }
      if (kind === 'week') { d.setDate(d.getDate() + 7); d.setHours(18, 0, 0, 0); }

      this.hour = d.getHours();
      this.minute = d.getMinutes();
      this.viewYear = d.getFullYear();
      this.viewMonth = d.getMonth();
      this.$emit('update:modelValue', toValue(d));
      this.open = false;
    },

    clear() {
      this.$emit('update:modelValue', '');
      this.open = false;
    },

    toggle() {
      this.open = !this.open;
      if (this.open) {
        const base = this.picked || new Date();
        this.viewYear = base.getFullYear();
        this.viewMonth = base.getMonth();
      }
    },

    onOutside(e) {
      if (this.open && !this.$el.contains(e.target)) this.open = false;
    },
  },

  mounted() {
    document.addEventListener('click', this.onOutside, true);
    document.addEventListener('keydown', this.onEsc);
  },
  unmounted() {
    document.removeEventListener('click', this.onOutside, true);
    document.removeEventListener('keydown', this.onEsc);
  },
  created() {
    this.onEsc = (e) => { if (e.key === 'Escape') this.open = false; };
  },

  template: `
<div class="dt">
  <button type="button" class="dt-field" :id="id" @click="toggle"
          :aria-expanded="open" aria-haspopup="dialog">
    <span :class="{ muted: !picked }">{{ label }}</span>
    <span class="dt-caret" aria-hidden="true"></span>
  </button>

  <div v-if="open" class="dt-pop" role="dialog" aria-label="Sene we wagt saýlamak">
    <div class="dt-head">
      <button type="button" @click="shiftMonth(-1)" aria-label="Öňki aý">‹</button>
      <b>{{ MONTHS[viewMonth] }} {{ viewYear }}</b>
      <button type="button" @click="shiftMonth(1)" aria-label="Indiki aý">›</button>
    </div>

    <div class="dt-grid dt-wd">
      <span v-for="w in WEEKDAYS" :key="w">{{ w }}</span>
    </div>

    <div class="dt-grid">
      <template v-for="(row, ri) in weeks" :key="ri">
        <button type="button" v-for="d in row" :key="d.toISOString()"
                class="dt-day"
                :class="{ other: isOther(d), today: isToday(d), sel: isPicked(d) }"
                @click="choose(d)">{{ d.getDate() }}</button>
      </template>
    </div>

    <div class="dt-time">
      <label :for="id + '-h'">Wagty</label>
      <tk-select class="compact" :id="id + '-h'" v-model="hour"
                 :options="hourOptions" @change="setTime"></tk-select>
      <span>:</span>
      <tk-select class="compact" :id="id + '-m'" v-model="minute"
                 :options="minuteOptions" @change="setTime"></tk-select>
    </div>

    <div class="dt-presets">
      <button type="button" @click="preset('today')">Şu gün 23:59</button>
      <button type="button" @click="preset('tomorrow')">Ertir 18:00</button>
      <button type="button" @click="preset('week')">Bir hepdeden</button>
    </div>

    <div class="dt-foot">
      <button type="button" @click="clear">Arassalamak</button>
      <button type="button" class="btn-primary" @click="open = false">Taýýar</button>
    </div>
  </div>
</div>`,
};
