// Ulgam dolandyryjysynyň paneli: hasaplar we ätiýaçlyk nusgalar.
import { TkSelect } from './select.js';

const { createApp, reactive, computed, toRefs } = Vue;

createApp({
  setup() {
    const SECTIONS = [
      { id: 'overview', title: 'Umumy syn',
        lede: 'Ulgamyň häzirki ýagdaýy we okuwçylar üçin salgylar.' },
      { id: 'accounts', title: 'Mugallymlar', badge: true,
        lede: 'Mugallymlaryň hasaplaryny döretmek, açar sözi çalyşmak we pozmak.' },
      { id: 'students', title: 'Okuwçylar', badge: true,
        lede: 'Ähli toparlar we okuwçylar; olara login we açar söz bermek.' },
      { id: 'backups', title: 'Ätiýaçlyk nusgalar', badge: true,
        lede: 'Maglumatlaryň nusgasyny döretmek, ýüklemek we ondan dikeltmek.' },
      { id: 'system', title: 'Ulgam',
        lede: 'Tehniki maglumatlar we howpsuzlyk barlagy.' },
    ];

    const s = reactive({
      me: null, busy: false, error: null, flash: '',
      tab: 'overview',
      stats: null, system: {}, teachers: [], backups: [],
      students: [], groups: [], creds: [], studentFilter: null,
      form: { login: '', password: '', full_name: '', role: 'teacher' },
      pwFor: null, pwValue: '',
      restoreFor: null, restoreConfirm: '',
    });

    function setError(msg, next) { s.error = msg ? { msg, next: next || '' } : null; }
    function ok(msg) { s.flash = msg; setTimeout(() => { s.flash = ''; }, 3000); }

    async function api(path, opts = {}) {
      const res = await fetch(path, {
        ...opts, headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw Object.assign(new Error(body.error || 'Ýalňyşlyk'), { body });
      return body;
    }

    const guard = (fn) => async (...args) => {
      s.busy = true; setError(null);
      try { return await fn(...args); }
      catch (e) { setError(e.body?.error || e.message, e.body?.next); }
      finally { s.busy = false; }
    };

    const LABELS = {
      teachers: 'hasap', groups: 'topar', students: 'okuwçy', lessons: 'sapak',
      questions: 'sorag', tests: 'synag', sessions: 'geçiriliş',
      answers: 'jogap', uploads: 'faýl',
    };
    const statList = computed(() =>
      s.stats ? Object.entries(LABELS).map(([k, label]) => ({ k: label, v: s.stats[k] ?? 0 })) : []);

    async function reload() {
      [s.stats, s.system, s.teachers, s.backups, s.students, s.groups] = await Promise.all([
        api('/api/admin/stats'), api('/api/admin/system'),
        api('/api/admin/teachers'), api('/api/admin/backups'),
        api('/api/admin/students'), api('/api/admin/groups'),
      ]);
    }

    const groupFilterOptions = computed(() => [
      { value: null, label: 'Ähli toparlar' },
      ...s.groups.map((g) => ({
        value: g.id, label: g.name, hint: g.teacher_name,
        badge: `${g.students}`,
      })),
    ]);

    const filteredStudents = computed(() => (s.studentFilter
      ? s.students.filter((x) => x.group_id === s.studentFilter)
      : s.students));

    // Okuwçylara hasap bermek: açar sözler jogapda bir gezek açyk gelýär.
    const makeGroupAccounts = guard(async (g) => {
      const out = await api(`/api/admin/groups/${g.id}/accounts`, { method: 'POST' });
      s.creds = out.map((c) => ({ ...c, path: '/profil' }));
      await reload();
      ok(`${out.length} okuwçy hasaby döredildi`);
    });

    const resetStudent = guard(async (st) => {
      const c = await api(`/api/admin/students/${st.id}/reset`, { method: 'POST' });
      s.creds = [{ ...c, path: '/profil' }];
      ok('Täze açar söz döredildi');
    });

    function printCreds() { window.print(); }

    const current = computed(() => SECTIONS.find((x) => x.id === s.tab) || SECTIONS[0]);

    function badge(id) {
      if (id === 'accounts') return s.teachers.length;
      if (id === 'students') return s.students.length;
      if (id === 'backups') return s.backups.length;
      return 0;
    }

    const uptime = computed(() => {
      const total = s.system.uptime_sec || 0;
      const h = Math.floor(total / 3600);
      const m = Math.floor((total % 3600) / 60);
      if (h) return `${h} sag ${m} min`;
      if (m) return `${m} min`;
      return `${total} sek`;
    });

    async function logout() {
      await api('/api/logout', { method: 'POST' }).catch(() => {});
      location.href = '/';
    }

    // ---- Hasaplar ----

    const createTeacher = guard(async () => {
      const out = await api('/api/admin/teachers', {
        method: 'POST', body: JSON.stringify(s.form),
      });
      // Mugallyma bermeli maglumatlary şol bir çap edilýän sanawda görkezýäris.
      s.creds = [{
        full_name: out.teacher.full_name, login: out.teacher.login,
        password: out.password, path: '/',
      }];
      s.form = { login: '', password: '', full_name: '', role: 'teacher' };
      await reload();
      ok('Hasap döredildi');
    });

    function askPassword(t) { s.pwFor = t; s.pwValue = ''; }

    const savePassword = guard(async () => {
      await api(`/api/admin/teachers/${s.pwFor.id}/password`, {
        method: 'POST', body: JSON.stringify({ password: s.pwValue }),
      });
      s.pwFor = null;
      ok('Açar söz üýtgedildi');
    });

    const removeTeacher = guard(async (t) => {
      if (!confirm(`«${t.full_name}» hasaby pozulsynmy?`)) return;
      await api(`/api/admin/teachers/${t.id}`, { method: 'DELETE' });
      await reload();
      ok('Hasap pozuldy');
    });

    // ---- Ätiýaçlyk nusgalar ----

    const makeBackup = guard(async () => {
      const b = await api('/api/admin/backups', { method: 'POST' });
      s.backups = await api('/api/admin/backups');
      ok(`Nusga döredildi: ${b.name}`);
    });

    function askRestore(b) { s.restoreFor = b; s.restoreConfirm = ''; }

    const doRestore = guard(async () => {
      await api('/api/admin/restore', {
        method: 'POST',
        body: JSON.stringify({ name: s.restoreFor.name, confirm: s.restoreConfirm }),
      });
      s.restoreFor = null;
      await reload();
      ok('Maglumatlar dikeldildi');
    });

    api('/api/me')
      .then((me) => {
        s.me = me;
        if (me.role !== 'admin') { location.href = '/'; return null; }
        return reload();
      })
      .catch(() => { location.href = '/'; });

    const roleOptions = [
      { value: 'teacher', label: 'Mugallym', hint: 'sapak geçirýär we material taýýarlaýar' },
      { value: 'admin', label: 'Ulgam dolandyryjysy', hint: 'hasaplar we ätiýaçlyk nusgalar' },
    ];

    return {
      ...toRefs(s), SECTIONS, roleOptions,
      groupFilterOptions, filteredStudents,
      makeGroupAccounts, resetStudent, printCreds,
      host: location.host, statList, current, badge, uptime, logout,
      createTeacher, askPassword, savePassword, removeTeacher,
      makeBackup, askRestore, doRestore,
      kb: (n) => (n < 1024 ? n + ' B' : n < 1048576 ? Math.round(n / 1024) + ' KB' : (n / 1048576).toFixed(1) + ' MB'),
      fmtDate: (iso) => {
        const d = new Date(iso);
        const p = (n) => String(n).padStart(2, '0');
        return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()} ${p(d.getHours())}:${p(d.getMinutes())}`;
      },
    };
  },
})
  .component('TkSelect', TkSelect)
  .mount('#app');
