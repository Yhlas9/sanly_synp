// Okuwçynyň sahypasy. Jogaplar ilki ýerli nobata düşýär, soň servere ugradylýar:
// tor kesilse hem öň ýazylan jogap ýitmeýär (ТЗ, 6-njy bölüm).
import { mountDemo } from './demos.js';
import { TkSelect } from './select.js';

const { createApp, reactive, computed, nextTick } = Vue;

const TOKEN_KEY = 'sanly_synp_token';
const QUEUE_KEY = 'sanly_synp_queue';

const KIND_LABELS = {
  choice: 'Jogap saýlamak',
  short: 'Gysga jogap',
  numeric: 'Sanly jogap',
  match: 'Gabat getirmek',
  order: 'Yzygiderlilik düzmek',
  open: 'Açyk jogap',
  practical: 'Amaly iş',
};

function loadQueue() {
  try { return JSON.parse(localStorage.getItem(QUEUE_KEY) || '{}'); } catch { return {}; }
}
function readToken() {
  try { return localStorage.getItem(TOKEN_KEY) || ''; } catch { return ''; }
}
function writeToken(v) {
  try { v ? localStorage.setItem(TOKEN_KEY, v) : localStorage.removeItem(TOKEN_KEY); }
  catch { /* hususy režim */ }
}
function saveQueue(q) {
  try { localStorage.setItem(QUEUE_KEY, JSON.stringify(q)); } catch { /* hususy režim */ }
}

createApp({
  setup() {
    const s = reactive({
      token: readToken(),
      joined: false,
      busy: false,
      error: null,
      connected: false,
      me: null,
      session: {},
      lesson: null,
      questions: [],
      values: {},     // question_id -> jogabyň mazmuny
      status: {},     // question_id -> 'pending' | 'saved'
      submitted: false,
      myResult: null,
      closedReason: '',
      uploading: {},
      remaining: 0,
      frame: '',
      zoom: false,
      form: { code: '', name: '' },
      rosterNames: [], rosterGroup: '',
    });

    // Nobat: serwere entek ýetmedik jogaplar. Sahypa ýapylsa hem saklanýar.
    // reactive bolmaly: nobatdaky jogaplaryň sany ekranda janly görünýär.
    const queue = reactive(loadQueue());
    let clockSkew = 0;     // serwer bilen enjamyň sagat tapawudy
    let deadlineMs = 0;
    let ws = null;

    const state = computed(() => s.session.state || 'lobby');
    const answeredCount = computed(() => Object.keys(s.values).length);
    const pendingCount = computed(() => Object.keys(queue).length);

    const clock = computed(() => {
      const t = Math.max(0, Math.floor(s.remaining));
      const m = String(Math.floor(t / 60)).padStart(2, '0');
      const sec = String(t % 60).padStart(2, '0');
      return `${m}:${sec}`;
    });

    function setError(msg, next) { s.error = msg ? { msg, next: next || '' } : null; }

    async function api(path, opts = {}) {
      const res = await fetch(path, {
        ...opts,
        headers: {
          'Content-Type': 'application/json',
          'X-Student-Token': s.token,
          ...(opts.headers || {}),
        },
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw Object.assign(new Error(body.error || 'Ýalňyşlyk'), { body });
      return body;
    }

    // Kod doly ýazylanda sapagyň topar sanawyny soraýarys: şonda okuwçy
    // adyny ýazman, sanawdan saýlaýar.
    let peekTimer = null;
    function peekRoster() {
      clearTimeout(peekTimer);
      const code = (s.form.code || '').trim();
      if (code.length < 5) { s.rosterNames = []; s.rosterGroup = ''; return; }

      peekTimer = setTimeout(async () => {
        try {
          const r = await fetch(`/api/roster?code=${encodeURIComponent(code)}`);
          if (!r.ok) { s.rosterNames = []; s.rosterGroup = ''; return; }
          const d = await r.json();
          s.rosterNames = d.names || [];
          s.rosterGroup = d.group_name || '';
          if (!s.rosterNames.includes(s.form.name)) s.form.name = '';
        } catch { s.rosterNames = []; s.rosterGroup = ''; }
      }, 300);
    }

    async function join() {
      s.busy = true;
      setError(null);
      try {
        const out = await api('/api/join', {
          method: 'POST',
          body: JSON.stringify({ code: s.form.code, name: s.form.name }),
        });
        s.token = out.token;
        writeToken(out.token);
        await refresh();
        openSocket();
      } catch (e) {
        setError(e.body?.error || e.message, e.body?.next);
      } finally {
        s.busy = false;
      }
    }

    async function refresh() {
      if (!s.token) return;
      try {
        const out = await api('/api/student/state');
        s.joined = true;
        s.me = out.participant;
        s.session = out.session;
        s.lesson = out.lesson || null;
        s.questions = out.questions || [];

        // Serweriň sagady esas: enjamdaky sagady üýtgetmek möhlete täsir etmeýär.
        clockSkew = new Date(out.server_time).getTime() - Date.now();
        s.closedReason = out.closed_reason || '';
        // Öý işinde möhlet her okuwçyda öz: serwer ony taýýar görnüşde berýär.
        const dl = out.deadline || out.session.test_deadline;
        deadlineMs = dl ? new Date(dl).getTime() : 0;

        // Öň saklanan jogaplary dikeldýäris.
        for (const [qid, raw] of Object.entries(out.saved_answers || {})) {
          if (!queue[qid]) {
            s.values[qid] = raw;
            s.status[qid] = 'saved';
          }
        }
        for (const qid of Object.keys(queue)) s.status[qid] = 'pending';

        setError(null);
        await nextTick();
        mountLessonDemo();
      } catch (e) {
        if (e.body?.error === 'Sapaga goşulmadyk') {
          writeToken('');
          s.token = ''; s.joined = false;
        }
        setError(e.body?.error || e.message, e.body?.next);
      }
    }

    function mountLessonDemo() {
      const host = document.querySelector('[ref="demoHost"], .demo-box');
      if (host && s.lesson?.demo_kind) {
        mountDemo(host, s.lesson.demo_kind, s.lesson.demo_config || {});
      }
    }

    // ---- Jogaplar ----

    const val = (q) => s.values[q.id] || {};

    function setValue(q, patch) {
      const next = { ...val(q), ...patch };
      s.values[q.id] = next;
      s.status[q.id] = 'pending';
      // client_key jogabyň her täze görnüşi üçin täze: gaýtadan ugradylan
      // şol bir jogap iki gezek hasaba alynmaýar.
      queue[q.id] = { question_id: q.id, client_key: newKey(), value: next };
      saveQueue(queue);
      flush();
    }

    function newKey() {
      return (crypto.randomUUID?.() || String(Date.now() + Math.random()));
    }

    function isSelected(q, optId) {
      return (val(q).selected || []).includes(optId);
    }

    function toggleChoice(q, optId) {
      const cur = val(q).selected || [];
      let next;
      if (q.payload.multiple) {
        next = cur.includes(optId) ? cur.filter((x) => x !== optId) : [...cur, optId];
      } else {
        next = [optId];
      }
      setValue(q, { selected: next });
    }

    function setPair(q, leftId, rightId) {
      const pairs = { ...(val(q).pairs || {}) };
      if (rightId) pairs[leftId] = rightId; else delete pairs[leftId];
      setValue(q, { pairs });
    }

    function orderOf(q) {
      const cur = val(q).order;
      if (cur?.length) return cur;
      return (q.payload.items || []).map((i) => i.id);
    }

    function itemText(q, id) {
      return (q.payload.items || []).find((i) => i.id === id)?.text || id;
    }

    function move(q, pos, dir) {
      const arr = [...orderOf(q)];
      const to = pos + dir;
      if (to < 0 || to >= arr.length) return;
      [arr[pos], arr[to]] = [arr[to], arr[pos]];
      setValue(q, { order: arr });
    }

    // ---- Amaly iş üçin faýl ----

    async function uploadFile(q, ev) {
      const file = ev.target.files?.[0];
      if (!file) return;

      s.uploading[q.id] = true;
      setError(null);
      try {
        const fd = new FormData();
        fd.append('file', file);
        const res = await fetch('/api/student/upload', {
          method: 'POST',
          headers: { 'X-Student-Token': s.token },
          body: fd,
        });
        const body = await res.json().catch(() => ({}));
        if (!res.ok) throw Object.assign(new Error(body.error), { body });

        const files = [...(val(q).files || []), body];
        setValue(q, { files });
      } catch (e) {
        setError(e.body?.error || 'Faýl ugradylmady', e.body?.next || 'Gaýtadan synanyşyň');
      } finally {
        s.uploading[q.id] = false;
        ev.target.value = '';
      }
    }

    function removeFile(q, id) {
      setValue(q, { files: (val(q).files || []).filter((f) => f.id !== id) });
    }

    // ---- Nobaty boşatmak ----
    let flushing = false;

    async function flush() {
      if (flushing || !s.token) return;
      flushing = true;

      for (const qid of Object.keys(queue)) {
        const item = queue[qid];
        try {
          await api('/api/student/answer', { method: 'POST', body: JSON.stringify(item) });
          delete queue[qid];
          saveQueue(queue);
          s.status[qid] = 'saved';
        } catch (e) {
          // Tor ýok ýa-da serwer jogap bermeýär: jogap nobatda galýar.
          if (e.body?.error) setError(e.body.error, e.body.next);
          break;
        }
      }
      flushing = false;
    }

    // Свой результат открывается только после того, как учитель его выдал.
    async function fetchMyResult() {
      try { s.myResult = await api('/api/student/result'); }
      catch { s.myResult = null; }
    }

    async function submit() {
      s.busy = true;
      await flush();
      try {
        await api('/api/student/submit', { method: 'POST' });
        s.submitted = true;
        await fetchMyResult();
        s.session = { ...s.session, state: 'closed' };
        setError(null);
      } catch (e) {
        setError(e.body?.error || e.message, e.body?.next);
      } finally {
        s.busy = false;
      }
    }

    // ---- Janly birikme ----

    function openSocket() {
      if (!s.token) return;
      const proto = location.protocol === 'https:' ? 'wss' : 'ws';
      ws = new WebSocket(`${proto}://${location.host}/ws/student?token=${encodeURIComponent(s.token)}`);

      ws.onopen = () => { s.connected = true; flush(); };
      ws.onclose = () => {
        s.connected = false;
        setTimeout(openSocket, 2000); // 60 sekuntlyk kesilme hem dikeldilýär
      };
      ws.onmessage = (ev) => {
        let msg;
        try { msg = JSON.parse(ev.data); } catch { return; }

        if (msg.type === 'frame') { s.frame = msg.data; return; }
        if (msg.type === 'frame_stop') { s.frame = ''; return; }
        if (msg.type === 'test_started' || msg.type === 'state') {
          refresh();
          fetchMyResult();
        }
      };
    }

    // ---- Sagat we gaýtadan synanyşyk ----

    setInterval(() => {
      if (deadlineMs) s.remaining = (deadlineMs - (Date.now() + clockSkew)) / 1000;
      if (pendingCount.value) flush();
    }, 1000);

    window.addEventListener('online', flush);

    if (s.token) { refresh().then(openSocket); }

    return {
      ...Vue.toRefs(s),
      state, answeredCount, pendingCount, clock,
      rosterOptions: computed(() => [
        { value: '', label: '— adyňyzy saýlaň —' },
        ...s.rosterNames.map((n) => ({ value: n, label: n })),
      ]),
      matchOptions: (q) => [
        { value: '', label: '— saýlaň —' },
        ...(q.payload.right || []).map((r) => ({ value: r.id, label: r.text })),
      ],
      join, peekRoster, submit, fetchMyResult, uploadFile, removeFile, val, setValue, isSelected, toggleChoice, setPair,
      orderOf, itemText, move,
      kindLabel: (k) => KIND_LABELS[k] || k,
      kb: (n) => (n < 1024 ? n + ' B' : n < 1048576 ? Math.round(n / 1024) + ' KB' : (n / 1048576).toFixed(1) + ' MB'),
      fmtTime: (iso) => {
        if (!iso) return '—';
        const d = new Date(iso);
        const p = (n) => String(n).padStart(2, '0');
        return `${p(d.getDate())}.${p(d.getMonth() + 1)} ${p(d.getHours())}:${p(d.getMinutes())}`;
      },
      renderBody: (md) => (md || '')
        .replace(/&/g, '&amp;').replace(/</g, '&lt;')
        .replace(/\*\*(.+?)\*\*/g, '<b>$1</b>')
        .replace(/\n\n/g, '</p><p>')
        .replace(/^/, '<p>').concat('</p>'),
    };
  },
})
  .component('TkSelect', TkSelect)
  .mount('#app');
