// Okuwçynyň şahsy otagy: öz maglumatlary we ähli işleriniň taryhy.
const { createApp, reactive, computed, toRefs } = Vue;

createApp({
  setup() {
    const s = reactive({
      me: null, profile: null, busy: false, error: null,
      form: { login: '', password: '' },
    });

    function setError(msg, next) { s.error = msg ? { msg, next: next || '' } : null; }

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

    async function load() {
      s.profile = await api('/api/okuwcy/profile');
    }

    async function login() {
      s.busy = true; setError(null);
      try {
        s.me = await api('/api/okuwcy/login', {
          method: 'POST', body: JSON.stringify(s.form),
        });
        await load();
      } catch (e) {
        setError(e.body?.error || e.message, e.body?.next);
      } finally { s.busy = false; }
    }

    async function logout() {
      await api('/api/okuwcy/logout', { method: 'POST' }).catch(() => {});
      location.reload();
    }

    api('/api/okuwcy/me').then((me) => { s.me = me; return load(); }).catch(() => {});

    return {
      ...toRefs(s), initials, login, logout,
      statusLabel: (w) => {
        if (w.status === 'submitted') return 'tabşyryldy';
        if (w.status === 'late') return 'giç tabşyryldy';
        return 'dowam edýär';
      },
      fmtDate: (iso) => {
        const d = new Date(iso);
        const p = (n) => String(n).padStart(2, '0');
        return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()}`;
      },
    };
  },
}).mount('#app');
