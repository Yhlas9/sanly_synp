// Mugallymyň otagy: taýýarlan materiallary, geçiren sapaklarynyň taryhy
// we öz açar sözüni çalyşmak.
const { createApp, reactive, computed, toRefs } = Vue;

const LABELS = {
  lessons: 'sapak', questions: 'sorag', tests: 'synag',
  groups: 'topar', students: 'okuwçy', sessions: 'geçiriliş', answers: 'jogap',
};

createApp({
  setup() {
    const s = reactive({
      me: null, profile: null, busy: false, error: null, flash: '',
      pw: { current: '', next: '' },
    });

    function setError(msg, next) { s.error = msg ? { msg, next: next || '' } : null; }
    function ok(msg) { s.flash = msg; setTimeout(() => { s.flash = ''; }, 3500); }

    async function api(path, opts = {}) {
      const res = await fetch(path, {
        ...opts, headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw Object.assign(new Error(body.error || 'Ýalňyşlyk'), { body });
      return body;
    }

    const initials = computed(() => {
      const parts = (s.me?.full_name || '').trim().split(/\s+/).slice(0, 2);
      return parts.map((p) => p[0] || '').join('').toUpperCase() || '?';
    });

    const statList = computed(() => (s.profile
      ? Object.entries(LABELS).map(([k, label]) => ({ k: label, v: s.profile.stats[k] ?? 0 }))
      : []));

    async function changePassword() {
      s.busy = true; setError(null);
      try {
        await api('/api/me/password', { method: 'POST', body: JSON.stringify(s.pw) });
        s.pw = { current: '', next: '' };
        ok('Açar söz çalşyldy');
      } catch (e) {
        setError(e.body?.error || e.message, e.body?.next);
      } finally { s.busy = false; }
    }

    async function logout() {
      await api('/api/logout', { method: 'POST' }).catch(() => {});
      location.href = '/';
    }

    api('/api/me')
      .then((me) => { s.me = me; return api('/api/me/profile'); })
      .then((p) => { s.profile = p; })
      .catch(() => { location.href = '/'; });

    return {
      ...toRefs(s), initials, statList, changePassword, logout,
      fmtDate: (iso) => {
        const d = new Date(iso);
        const p = (n) => String(n).padStart(2, '0');
        return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()}`;
      },
    };
  },
}).mount('#app');
