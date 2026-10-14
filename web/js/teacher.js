// Mugallymyň paneli: sapagy dolandyrmak, ekrany görkezmek, netijeleri derňemek.
import { TkDateTime } from './datetime.js';
import { TkSelect } from './select.js';

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
const STATE_LABELS = { lobby: 'garaşylýar', lesson: 'sapak', test: 'synag', closed: 'ýapyk' };

// 5 kadr/sekunt — ТЗ-niň kabul ediş talaby (720p, gijikme 2 sekuntdan köp däl).
const SHARE_FPS = 5;
const SHARE_HEIGHT = 720;

createApp({
  setup() {
    const s = reactive({
      me: null, busy: false, error: null,
      form: { login: 'mugallym', password: '' },
      subjects: [], lessons: [], tests: [], groups: [],
      pick: { lesson: null, test: null, group: null, mode: 'live', opens: '', closes: '' },
      session: null,
      participants: [],
      results: null,
      detail: null,
      grade: {}, reason: {},
      sharing: false, preview: '', fps: 0,
      studentUrls: [],
      qr: '',
    });

    let ws = null, shareTimer = null, shareStream = null, video = null, canvas = null;

    // Adresi serwer aýdýar: mugallym localhost arkaly girse-de, okuwçylara
    // ýerli toruň salgysy görkezilmeli.
    const subjectName = (code) => s.subjects.find((x) => x.code === code)?.name_tk || code;

    const lessonOptions = computed(() => [
      { value: null, label: '— sapak saýlanmadyk —' },
      ...s.lessons.map((l) => ({
        value: l.id, label: l.title, hint: subjectName(l.subject_code),
      })),
    ]);

    const testOptions = computed(() => [
      { value: null, label: '— synag saýlanmadyk —' },
      ...s.tests.map((t) => ({
        value: t.id, label: t.title,
        hint: `${subjectName(t.subject_code)} · ${t.question_count} sorag`,
        badge: `${Math.round(t.duration_sec / 60)} min`,
      })),
    ]);

    const groupOptions = computed(() => [
      { value: null, label: 'Topar ýok', hint: 'islendik at bilen girip bolýar' },
      ...s.groups.map((g) => ({
        value: g.id, label: g.name,
        badge: g.student_count ? `${g.student_count} okuwçy` : null,
      })),
    ]);

    const studentURL = computed(() => s.studentUrls[0] || `${location.host}/okuwcy`);

    function setError(msg, next) { s.error = msg ? { msg, next: next || '' } : null; }

    async function api(path, opts = {}) {
      const res = await fetch(path, {
        ...opts,
        headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw Object.assign(new Error(body.error || 'Ýalňyşlyk'), { body });
      return body;
    }

    async function login() {
      s.busy = true; setError(null);
      try {
        s.me = await api('/api/login', { method: 'POST', body: JSON.stringify(s.form) });
        await Promise.all([loadCatalog(), loadNet()]);
      } catch (e) {
        setError(e.body?.error || e.message, e.body?.next);
      } finally { s.busy = false; }
    }

    async function logout() {
      await api('/api/logout', { method: 'POST' }).catch(() => {});
      location.reload();
    }

    async function loadNet() {
      try { s.studentUrls = (await api('/api/net')).student_urls || []; }
      catch { s.studentUrls = []; }
      makeQR();
    }

    // QR kod ýerli kitaphana bilen döredilýär — internet gerek däl.
    function makeQR() {
      const url = s.studentUrls[0];
      if (!url || typeof qrcode !== 'function') { s.qr = ''; return; }
      try {
        const q = qrcode(0, 'M');
        q.addData('http://' + url);
        q.make();
        s.qr = q.createDataURL(5, 8);
      } catch { s.qr = ''; }
    }

    async function loadCatalog() {
      [s.subjects, s.lessons, s.tests, s.groups] = await Promise.all([
        api('/api/subjects'), api('/api/lessons'), api('/api/tests'), api('/api/groups'),
      ]);
    }

    async function createSession() {
      s.busy = true; setError(null);
      try {
        const body = { lesson_id: s.pick.lesson, test_id: s.pick.test,
                       group_id: s.pick.group, mode: s.pick.mode };
        if (s.pick.mode === 'homework') {
          // datetime-local ýerli wagty berýär; servere ISO görnüşinde ugradýarys.
          body.opens_at = s.pick.opens ? new Date(s.pick.opens).toISOString() : null;
          body.closes_at = s.pick.closes ? new Date(s.pick.closes).toISOString() : null;
        }
        s.session = await api('/api/sessions', { method: 'POST', body: JSON.stringify(body) });
        openSocket();
        loadParticipants();
      } catch (e) {
        setError(e.body?.error || e.message, e.body?.next);
      } finally { s.busy = false; }
    }

    async function refreshSession() {
      const out = await api(`/api/sessions/${s.session.id}`);
      s.session = out.session;
    }

    async function loadParticipants() {
      s.participants = await api(`/api/sessions/${s.session.id}/participants`);
    }

    async function setState(state) {
      try {
        await api(`/api/sessions/${s.session.id}/state`, {
          method: 'POST', body: JSON.stringify({ state }),
        });
        await refreshSession();
      } catch (e) { setError(e.body?.error || e.message, e.body?.next); }
    }

    async function toggleJoin() {
      try {
        await api(`/api/sessions/${s.session.id}/join-open`, {
          method: 'POST', body: JSON.stringify({ open: !s.session.join_open }),
        });
        await refreshSession();
      } catch (e) { setError(e.body?.error || e.message, e.body?.next); }
    }

    async function startTest() {
      try {
        s.session = await api(`/api/sessions/${s.session.id}/start-test`, { method: 'POST' });
        setError(null);
      } catch (e) { setError(e.body?.error || e.message, e.body?.next); }
    }

    // Приём опоздавших и дополнительное время — отдельное действие учителя.
    async function addTime(minutes) {
      try {
        s.session = await api(`/api/sessions/${s.session.id}/extra-time`, {
          method: 'POST', body: JSON.stringify({ minutes }),
        });
      } catch (e) { setError(e.body?.error || e.message, e.body?.next); }
    }

    async function release() {
      try {
        await api(`/api/sessions/${s.session.id}/release`, { method: 'POST' });
        await refreshSession();
        await loadResults();
      } catch (e) { setError(e.body?.error || e.message, e.body?.next); }
    }

    async function loadResults() {
      try {
        s.results = await api(`/api/sessions/${s.session.id}/results`);
      } catch (e) { setError(e.body?.error || e.message, e.body?.next); }
    }

    async function openStudent(row) {
      const rows = await api(`/api/sessions/${s.session.id}/students/${row.participant_id}`);
      s.detail = { name: row.name, id: row.participant_id, rows };
      for (const d of rows) if (d.score != null) s.grade[d.answer_id] = d.score;
    }

    async function saveGrade(d) {
      try {
        await api(`/api/answers/${d.answer_id}/grade`, {
          method: 'POST',
          body: JSON.stringify({
            score: s.grade[d.answer_id] ?? 0,
            reason: s.reason[d.answer_id] || 'mugallymyň düzedişi',
            comment: s.reason[d.answer_id] || '',
          }),
        });
        await Promise.all([loadResults(), openStudent({ ...s.detail, participant_id: s.detail.id })]);
      } catch (e) { setError(e.body?.error || e.message, e.body?.next); }
    }

    // ---- Janly birikme ----

    function openSocket() {
      const proto = location.protocol === 'https:' ? 'wss' : 'ws';
      ws = new WebSocket(`${proto}://${location.host}/ws/teacher?session=${s.session.id}`);
      ws.onclose = () => setTimeout(() => { if (s.session) openSocket(); }, 2000);
      ws.onmessage = (ev) => {
        let msg; try { msg = JSON.parse(ev.data); } catch { return; }
        if (msg.type === 'participants') s.participants = msg.data;
      };
    }

    // ---- Ekrany görkezmek ----
    // WebRTC däl-de, kadrlary JPEG görnüşinde WebSocket arkaly ugradýarys:
    // ТЗ 5 kadr/sekunt talap edýär, bu usul ony ýerli torda ynamly ýerine ýetirýär.

    async function toggleShare() {
      if (s.sharing) { stopShare(); return; }
      try {
        shareStream = await navigator.mediaDevices.getDisplayMedia({
          video: { frameRate: SHARE_FPS }, audio: false,
        });
      } catch {
        setError('Ekrany görkezmek rugsat edilmedi', 'Brauzerde rugsat beriň we gaýtadan synanyşyň');
        return;
      }

      video = document.createElement('video');
      video.srcObject = shareStream;
      video.muted = true;
      await video.play();

      canvas = document.createElement('canvas');
      shareStream.getVideoTracks()[0].addEventListener('ended', stopShare);

      s.sharing = true;
      let sent = 0;
      const since = Date.now();

      shareTimer = setInterval(() => {
        if (!video.videoWidth) return;
        const scale = Math.min(1, SHARE_HEIGHT / video.videoHeight);
        canvas.width = Math.round(video.videoWidth * scale);
        canvas.height = Math.round(video.videoHeight * scale);
        canvas.getContext('2d').drawImage(video, 0, 0, canvas.width, canvas.height);

        const data = canvas.toDataURL('image/jpeg', 0.6);
        s.preview = data;
        if (ws?.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'frame', data }));
          sent++;
          s.fps = Math.round(sent / Math.max(1, (Date.now() - since) / 1000));
        }
      }, Math.round(1000 / SHARE_FPS));
    }

    function stopShare() {
      clearInterval(shareTimer);
      shareStream?.getTracks().forEach((t) => t.stop());
      shareStream = null; s.sharing = false; s.preview = ''; s.fps = 0;
      // Görkezmek bes edilende okuwçy sapak sahypasyna dolanýar.
      if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'frame_stop' }));
    }

    setInterval(() => { if (s.session) loadParticipants().catch(() => {}); }, 5000);

    api('/api/me')
      .then((me) => { s.me = me; return Promise.all([loadCatalog(), loadNet()]); })
      .catch(() => {});

    return {
      ...toRefs(s), studentURL, subjectName,
      lessonOptions, testOptions, groupOptions,
      login, logout, createSession, setState, toggleJoin, startTest,
      loadResults, openStudent, saveGrade, toggleShare, addTime, release,
      kindLabel: (k) => KIND_LABELS[k] || k,
      stateLabel: (k) => STATE_LABELS[k] || k,
      fmtTime: (iso) => {
        if (!iso) return '—';
        const d = new Date(iso);
        const p = (n) => String(n).padStart(2, '0');
        return `${p(d.getDate())}.${p(d.getMonth() + 1)} ${p(d.getHours())}:${p(d.getMinutes())}`;
      },
      statusLabel: (r) => {
        if (r.preliminary) return 'deslapky';
        if (r.status === 'submitted') return 'tabşyryldy';
        if (r.status === 'late') return 'giç tabşyryldy';
        return 'dowam edýär';
      },
      statusClass: (r) => (r.preliminary ? 'warn' : r.status === 'submitted' ? 'ok' : ''),
      showValue: (v, payload) => describe(v, payload),
    };
  },
})
  .component('TkDateTime', TkDateTime)
  .component('TkSelect', TkSelect)
  .mount('#app');
