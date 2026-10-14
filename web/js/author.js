// Materiallary taýýarlamak: sapaklar, soraglar, synaglar we toparlar.
// Mugallym öň taýýarlanan sapagy gaýtadan ulanyp, göçürip we üýtgedip bilýär.
import { mountDemo } from './demos.js';
import { TkSelect } from './select.js';

const { createApp, reactive, computed, toRefs, nextTick } = Vue;

const KINDS = {
  choice: 'Jogap saýlamak', short: 'Gysga jogap', numeric: 'Sanly jogap',
  match: 'Gabat getirmek', order: 'Yzygiderlilik düzmek',
  open: 'Açyk jogap', practical: 'Amaly iş',
};

const DEMOS = {
  function_plot: 'Funksiýanyň grafigi (matematika)',
  circuit: 'Elektrik zynjyry (fizika)',
  reaction: 'Reaksiýany deňlemek (himiýa)',
  cell: 'Öýjügiň gurluşy (biologiýa)',
  algorithm: 'Tertipleşdirme algoritmi (informatika)',
};

const LETTERS = 'abcdefgh';

createApp({
  setup() {
    const s = reactive({
      me: null, busy: false, error: null, flash: '',
      tab: 'lessons',
      subjects: [], lessons: [], tests: [], groups: [], bank: [],
      filterSubject: '',
      lesson: null, questions: [], question: null,
      demoConfigText: '{}',
      test: null,
      groupName: '',
      rosterGroup: null, roster: [], rosterNames: '', creds: [],
      journalGroup: null, journal: null,
    });

    function setError(msg, next) { s.error = msg ? { msg, next: next || '' } : null; }
    function ok(msg) { s.flash = msg; setTimeout(() => { s.flash = ''; }, 2500); }

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

    const subjectName = (code) => s.subjects.find((x) => x.code === code)?.name_tk || code;

    const subjectFilterOptions = computed(() => [
      { value: '', label: 'Ähli dersler' },
      ...s.subjects.map((x) => ({ value: x.code, label: x.name_tk })),
    ]);
    const subjectOptions = computed(() =>
      s.subjects.map((x) => ({ value: x.id, label: x.name_tk })));
    const kindOptions = computed(() =>
      Object.entries(KINDS).map(([value, label]) => ({ value, label })));
    const demoOptions = computed(() => [
      { value: '', label: '— görkezme ýok —' },
      ...Object.entries(DEMOS).map(([value, label]) => ({ value, label, hint: value })),
    ]);
    const policyOptions = [
      { value: 'after_close', label: 'Mugallym ýapanda', hint: 'iň köp ulanylýan' },
      { value: 'immediately', label: 'Tabşyran badyna' },
      { value: 'manual', label: 'Diňe el bilen' },
    ];
    const groupOptions = computed(() => [
      { value: null, label: '— topary saýlaň —' },
      ...s.groups.map((g) => ({
        value: g.id, label: g.name,
        badge: g.student_count ? `${g.student_count} okuwçy` : null,
      })),
    ]);

    const filteredLessons = computed(() =>
      s.filterSubject ? s.lessons.filter((l) => l.subject_code === s.filterSubject) : s.lessons);

    async function reload() {
      [s.subjects, s.lessons, s.tests, s.groups] = await Promise.all([
        api('/api/subjects'), api('/api/lessons'), api('/api/tests'), api('/api/groups'),
      ]);
    }

    // ---- Sapaklar ----

    function blankLesson() {
      return {
        id: null, subject_id: s.subjects[0]?.id || 1, title: '', topic: '', goal: '',
        body_md: '', demo_kind: '', practical_task: '',
      };
    }

    function newLesson() {
      s.lesson = blankLesson();
      s.questions = [];
      s.question = null;
      s.demoConfigText = '{}';
    }

    const editLesson = guard(async (l) => {
      s.lesson = { ...l };
      s.demoConfigText = JSON.stringify(l.demo_config || {});
      s.question = null;
      s.questions = await api(`/api/lessons/${l.id}/questions`);
      await nextTick();
      previewDemo();
    });

    const saveLesson = guard(async () => {
      let config;
      try { config = JSON.parse(s.demoConfigText || '{}'); }
      catch { throw Object.assign(new Error('x'), { body: { error: 'Görkezmäniň sazlamasy nädogry JSON', next: 'Mysal: {"a":1,"b":0,"c":-4}' } }); }

      const payload = { ...s.lesson, demo_config: config };
      const saved = s.lesson.id
        ? await api(`/api/lessons/${s.lesson.id}`, { method: 'PUT', body: JSON.stringify(payload) })
        : await api('/api/lessons', { method: 'POST', body: JSON.stringify(payload) });

      s.lesson = { ...saved };
      s.demoConfigText = JSON.stringify(saved.demo_config || {});
      await reload();
      ok('Sapak saklandy');
    });

    // Nusga alnandan soň ony derrew açýarys: mugallym adatça şobada üýtgedýär.
    const copyLesson = guard(async (l) => {
      const copy = await api(`/api/lessons/${l.id}/copy`, { method: 'POST' });
      await reload();
      await editLesson(copy);
      ok('Nusga alyndy — indi üýtgedip bilersiňiz');
    });

    const removeLesson = guard(async (l) => {
      if (!confirm(`«${l.title}» sapagy pozulsynmy?`)) return;
      await api(`/api/lessons/${l.id}`, { method: 'DELETE' });
      if (s.lesson?.id === l.id) s.lesson = null;
      await reload();
      ok('Sapak pozuldy');
    });

    function previewDemo() {
      const host = document.getElementById('demo-preview');
      if (!host || !s.lesson?.demo_kind) return;
      let cfg = {};
      try { cfg = JSON.parse(s.demoConfigText || '{}'); } catch { /* nädogry JSON */ }
      mountDemo(host, s.lesson.demo_kind, cfg);
    }

    // ---- Soraglar ----

    function blankQuestion() {
      return {
        id: null, lesson_id: s.lesson.id, subject_id: s.lesson.subject_id,
        kind: 'choice', prompt: '', points: 1, hint: '', explanation: '',
        payload: { options: [{ id: 'a', text: '' }, { id: 'b', text: '' }], multiple: false },
        answer_key: { correct: [] },
      };
    }

    function newQuestion() { s.question = blankQuestion(); }

    function editQuestion(q) {
      s.question = JSON.parse(JSON.stringify(q));
      s.question.payload = s.question.payload || {};
      s.question.answer_key = s.question.answer_key || {};
      if (s.question.kind === 'choice' && !s.question.answer_key.correct) {
        s.question.answer_key.correct = [];
      }
    }

    // resetKey меняет форму ключа под выбранный тип задания.
    function resetKey() {
      const q = s.question;
      switch (q.kind) {
        case 'choice':
          q.payload = { options: [{ id: 'a', text: '' }, { id: 'b', text: '' }], multiple: false };
          q.answer_key = { correct: [] };
          break;
        case 'short':   q.payload = {}; q.answer_key = { accept: [] }; break;
        case 'numeric': q.payload = { unit: '' }; q.answer_key = { value: 0, tolerance: 0, unit: '' }; break;
        case 'match':
          q.payload = { left: [{ id: 'l1', text: '' }], right: [{ id: 'r1', text: '' }] };
          q.answer_key = { pairs: { l1: 'r1' } };
          break;
        case 'order':
          q.payload = { items: [{ id: 'i1', text: '' }] };
          q.answer_key = { order: ['i1'] };
          break;
        default: q.payload = {}; q.answer_key = {};
      }
    }

    function addOption() {
      const opts = s.question.payload.options;
      opts.push({ id: LETTERS[opts.length] || `o${opts.length}`, text: '' });
    }

    function toggleCorrect(id) {
      const key = s.question.answer_key;
      const cur = key.correct || [];
      if (cur.includes(id)) {
        key.correct = cur.filter((x) => x !== id);
      } else {
        key.correct = s.question.payload.multiple ? [...cur, id] : [id];
      }
    }

    function addPair() {
      const p = s.question.payload;
      const n = p.left.length + 1;
      p.left.push({ id: `l${n}`, text: '' });
      p.right.push({ id: `r${n}`, text: '' });
      s.question.answer_key.pairs[`l${n}`] = `r${n}`;
    }

    function removePair(i) {
      const p = s.question.payload;
      const left = p.left[i].id;
      p.left.splice(i, 1);
      p.right.splice(i, 1);
      delete s.question.answer_key.pairs[left];
    }

    function addItem() {
      const items = s.question.payload.items;
      const id = `i${items.length + 1}`;
      items.push({ id, text: '' });
      s.question.answer_key.order = items.map((x) => x.id);
    }

    function removeItem(i) {
      const items = s.question.payload.items;
      items.splice(i, 1);
      s.question.answer_key.order = items.map((x) => x.id);
    }

    const saveQuestion = guard(async () => {
      const q = s.question;
      // Для «order» правильный порядок — это тот, в котором учитель набрал элементы.
      if (q.kind === 'order') q.answer_key.order = q.payload.items.map((x) => x.id);

      if (q.id) {
        await api(`/api/questions/${q.id}`, { method: 'PUT', body: JSON.stringify(q) });
      } else {
        await api('/api/questions', { method: 'POST', body: JSON.stringify(q) });
      }
      s.questions = await api(`/api/lessons/${s.lesson.id}/questions`);
      s.question = null;
      ok('Sorag saklandy');
    });

    const removeQuestion = guard(async (q) => {
      if (!confirm('Sorag pozulsynmy?')) return;
      await api(`/api/questions/${q.id}`, { method: 'DELETE' });
      s.questions = await api(`/api/lessons/${s.lesson.id}/questions`);
      ok('Sorag pozuldy');
    });

    // ---- Synaglar ----

    function newTest() {
      s.test = {
        id: null, subject_id: s.subjects[0]?.id || 1, title: '', minutes: 15,
        shuffle_questions: false, shuffle_options: false, allow_retake: false,
        results_policy: 'after_close', question_ids: [],
      };
      loadBank();
    }

    const editTest = guard(async (t) => {
      const d = await api(`/api/tests/${t.id}`);
      const subject = s.subjects.find((x) => x.code === d.subject_code);
      s.test = {
        id: d.id, subject_id: subject?.id || 1, title: d.title,
        minutes: Math.round(d.duration_sec / 60),
        shuffle_questions: d.shuffle_questions, shuffle_options: d.shuffle_options,
        allow_retake: d.allow_retake, results_policy: d.results_policy,
        question_ids: d.question_ids,
      };
      await loadBank();
    });

    // loadBank собирает банк вопросов выбранного предмета из всех его уроков.
    async function loadBank() {
      const code = s.subjects.find((x) => x.id === s.test.subject_id)?.code;
      const lessons = s.lessons.filter((l) => l.subject_code === code);
      const lists = await Promise.all(lessons.map((l) => api(`/api/lessons/${l.id}/questions`)));
      s.bank = lists.flat();
    }

    function toggleBank(id) {
      const cur = s.test.question_ids;
      s.test.question_ids = cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id];
    }

    const saveTest = guard(async () => {
      const body = JSON.stringify({
        subject_id: s.test.subject_id, title: s.test.title,
        duration_sec: Math.max(1, s.test.minutes) * 60,
        shuffle_questions: s.test.shuffle_questions,
        shuffle_options: s.test.shuffle_options,
        allow_retake: s.test.allow_retake,
        results_policy: s.test.results_policy,
        question_ids: s.test.question_ids,
      });
      if (s.test.id) await api(`/api/tests/${s.test.id}`, { method: 'PUT', body });
      else await api('/api/tests', { method: 'POST', body });
      s.test = null;
      await reload();
      ok('Synag saklandy');
    });

    const removeTest = guard(async (t) => {
      if (!confirm(`«${t.title}» synagy pozulsynmy?`)) return;
      await api(`/api/tests/${t.id}`, { method: 'DELETE' });
      await reload();
      ok('Synag pozuldy');
    });

    // ---- Toparlar ----

    // ---- Topar sanawy ----

    const openRoster = guard(async (g) => {
      s.rosterGroup = g;
      s.rosterNames = '';
      s.creds = [];
      s.roster = await api(`/api/groups/${g.id}/students`);
    });

    const withoutAccount = computed(() => s.roster.filter((x) => !x.has_login).length);

    // Hasap döretmek: açar söz diňe şu jogapda açyk gelýär, soň diňe hash saklanýar.
    const makeAccounts = guard(async () => {
      const out = await api(`/api/groups/${s.rosterGroup.id}/accounts`, { method: 'POST' });
      s.creds = out;
      s.roster = await api(`/api/groups/${s.rosterGroup.id}/students`);
      await reload();
      ok(`${out.length} hasap döredildi`);
    });

    const resetAccount = guard(async (st) => {
      const c = await api(`/api/students/${st.id}/reset`, { method: 'POST' });
      s.creds = [c];
      ok('Täze açar söz döredildi');
    });

    function printCreds() { window.print(); }

    const saveRoster = guard(async () => {
      if (!s.rosterNames.trim()) return;
      const r = await api(`/api/groups/${s.rosterGroup.id}/students`, {
        method: 'POST', body: JSON.stringify({ names: s.rosterNames }),
      });
      s.rosterNames = '';
      s.roster = await api(`/api/groups/${s.rosterGroup.id}/students`);
      ok(`${r.added} okuwçy goşuldy`);
    });

    const removeStudent = guard(async (st) => {
      await api(`/api/students/${st.id}`, { method: 'DELETE' });
      s.roster = await api(`/api/groups/${s.rosterGroup.id}/students`);
      ok('Pozuldy');
    });

    // ---- Žurnal ----

    function openJournal() {
      s.tab = 'journal';
      if (!s.journalGroup && s.groups.length) {
        // Boş topar däl-de, okuwçylary bar bolan topar açylýar.
        const filled = s.groups.find((g) => g.student_count > 0);
        s.journalGroup = (filled || s.groups[0]).id;
      }
      if (s.journalGroup) loadJournal();
    }

    const loadJournal = guard(async () => {
      if (!s.journalGroup) { s.journal = null; return; }
      s.journal = await api(`/api/groups/${s.journalGroup}/journal`);
    });

    // Серверные ключи ячеек приходят как числа, а в JSON становятся строками.
    function cell(row, work) {
      return row.cells[work.session_id] ?? row.cells[String(work.session_id)] ?? null;
    }

    const addGroup = guard(async () => {
      if (!s.groupName.trim()) return;
      await api('/api/groups', { method: 'POST', body: JSON.stringify({ name: s.groupName.trim() }) });
      s.groupName = '';
      await reload();
      ok('Topar goşuldy');
    });

    const removeGroup = guard(async (g) => {
      await api(`/api/groups/${g.id}`, { method: 'DELETE' });
      await reload();
      ok('Topar pozuldy');
    });

    api('/api/me')
      .then((me) => { s.me = me; return reload(); })
      .catch(() => { location.href = '/'; });

    return {
      ...toRefs(s), KINDS, DEMOS, filteredLessons, subjectName,
      subjectFilterOptions, subjectOptions, kindOptions, demoOptions,
      policyOptions, groupOptions,
      newLesson, editLesson, saveLesson, copyLesson, removeLesson, previewDemo,
      newQuestion, editQuestion, saveQuestion, removeQuestion, resetKey,
      addOption, toggleCorrect, addPair, removePair, addItem, removeItem,
      newTest, editTest, saveTest, removeTest, loadBank, toggleBank,
      addGroup, removeGroup,
      openRoster, saveRoster, removeStudent,
      withoutAccount, makeAccounts, resetAccount, printCreds,
      host: location.host,
      openJournal, loadJournal, cell,
      fmtDate: (iso) => {
        const d = new Date(iso);
        const p = (n) => String(n).padStart(2, '0');
        return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()}`;
      },
      kindLabel: (k) => KINDS[k] || k,
      demoLabel: (k) => DEMOS[k] || k,
    };
  },
})
  .component('TkSelect', TkSelect)
  .mount('#app');
