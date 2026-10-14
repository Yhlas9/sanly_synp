// Aýratyn okuwçynyň hasabaty — çap etmek üçin amatly görnüş (ТЗ, 7-nji bölüm).
const { createApp, reactive, computed, toRefs } = Vue;

// Belgileri okalýan tekste öwürýäris: hasabatda "b" däl-de, jogabyň özi görünmeli.
function labelMap(payload) {
  const map = {};
  for (const key of ['options', 'items', 'left', 'right']) {
    for (const x of payload?.[key] || []) map[x.id] = x.text;
  }
  return map;
}

function describe(v, payload) {
  if (v == null) return '—';
  if (typeof v !== 'object') return String(v);

  const m = labelMap(payload);
  const name = (id) => m[id] || id;

  if (v.selected) return v.selected.map(name).join(', ') || '—';
  if (v.correct) return v.correct.map(name).join(', ') || '—';
  if (v.order) return v.order.map(name).join(' → ') || '—';
  if (v.pairs) {
    const pairs = Object.entries(v.pairs).map(([a, b]) => `${name(a)} → ${name(b)}`);
    return pairs.join('; ') || '—';
  }
  if (v.text != null) return v.text || '—';
  if (v.value != null) return `${v.value} ${v.unit || ''}`.trim();
  if (v.accept) return v.accept.join(' / ');
  if (v.files) return v.files.map((f) => f.file_name).join(', ');
  return JSON.stringify(v);
}

const KIND_LABELS = {
  choice: 'Jogap saýlamak', short: 'Gysga jogap', numeric: 'Sanly jogap',
  match: 'Gabat getirmek', order: 'Yzygiderlilik düzmek',
  open: 'Açyk jogap', practical: 'Amaly iş',
};

createApp({
  setup() {
    const s = reactive({ error: null, session: {}, row: null, rows: [] });

    const params = new URLSearchParams(location.search);
    const sessionID = params.get('session');
    const participantID = params.get('student');

    async function api(path) {
      const res = await fetch(path, { headers: { 'Content-Type': 'application/json' } });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw Object.assign(new Error(body.error || 'Ýalňyşlyk'), { body });
      return body;
    }

    const autoCount = computed(() => s.rows.filter((d) => !d.needs_manual).length);
    const correctCount = computed(() => s.rows.filter((d) => !d.needs_manual && d.is_correct).length);

    (async () => {
      if (!sessionID || !participantID) {
        s.error = { msg: 'Salgy doly däl', next: 'Netijeler sahypasyndan açyň' };
        return;
      }
      try {
        const [info, results, detail] = await Promise.all([
          api(`/api/sessions/${sessionID}`),
          api(`/api/sessions/${sessionID}/results`),
          api(`/api/sessions/${sessionID}/students/${participantID}`),
        ]);
        s.session = info.session;
        s.rows = detail;
        s.row = results.rows.find((r) => String(r.participant_id) === String(participantID));
        if (!s.row) {
          s.error = { msg: 'Okuwçy tapylmady', next: 'Netijeler sahypasyna dolanyň' };
          return;
        }
        document.title = `${s.row.name} — hasabat`;
      } catch (e) {
        s.error = { msg: e.body?.error || e.message, next: e.body?.next || 'Gaýtadan giriň' };
      }
    })();

    const fmtDate = (iso) => {
      if (!iso) return '—';
      const d = new Date(iso);
      const p = (n) => String(n).padStart(2, '0');
      return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()} ${p(d.getHours())}:${p(d.getMinutes())}`;
    };

    return {
      ...toRefs(s), autoCount, correctCount,
      today: fmtDate(new Date().toISOString()).slice(0, 10),
      print: () => window.print(),
      fmtDate,
      kindLabel: (k) => KIND_LABELS[k] || k,
      showValue: (v, payload) => describe(v, payload),
    };
  },
}).mount('#app');
