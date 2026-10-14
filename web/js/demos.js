// Interaktiw görkezmeler. Okuwçynyň hereketi netijä täsir edýär:
// parametr üýtgände grafik täzelenýär ýa-da düşündiriş açylýar.
// Statik surat interaktiw görkezme hasaplanmaýar (ТЗ, 8-nji bölüm).

const css = (name) => getComputedStyle(document.body).getPropertyValue(name).trim();

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === 'class') node.className = v;
    else if (k === 'text') node.textContent = v;
    else if (k.startsWith('on')) node.addEventListener(k.slice(2), v);
    else node.setAttribute(k, v);
  }
  for (const c of [].concat(children)) node.appendChild(c);
  return node;
}

function slider(label, { min, max, step, value, unit = '' }, onInput) {
  const out = el('output', { text: value + unit });
  const input = el('input', {
    type: 'range', min, max, step, value,
    'aria-label': label,
    oninput: (e) => {
      const v = parseFloat(e.target.value);
      out.textContent = v + unit;
      onInput(v);
    },
  });
  return { row: el('div', { class: 'slider-row' }, [el('label', { text: label }), input, out]), input };
}

function hiDPI(canvas, w, h) {
  const ratio = window.devicePixelRatio || 1;
  canvas.width = w * ratio;
  canvas.height = h * ratio;
  canvas.style.aspectRatio = `${w} / ${h}`;
  const ctx = canvas.getContext('2d');
  ctx.scale(ratio, ratio);
  return ctx;
}

/* ---------- Matematika: parametrleri üýtgedilýän funksiýa grafigi ---------- */
function functionPlot(host, cfg) {
  const W = 560, H = 320;
  const canvas = el('canvas');
  const readout = el('div', { class: 'demo-readout' });
  const state = { a: cfg.a ?? 1, b: cfg.b ?? 0, c: cfg.c ?? 0 };
  const ctx = hiDPI(canvas, W, H);

  const sx = (x) => W / 2 + x * 28;
  const sy = (y) => H / 2 - y * 28;

  function draw() {
    ctx.clearRect(0, 0, W, H);
    ctx.strokeStyle = css('--border'); ctx.lineWidth = 1;
    for (let gx = -9; gx <= 9; gx++) {
      ctx.beginPath(); ctx.moveTo(sx(gx), 0); ctx.lineTo(sx(gx), H); ctx.stroke();
    }
    for (let gy = -5; gy <= 5; gy++) {
      ctx.beginPath(); ctx.moveTo(0, sy(gy)); ctx.lineTo(W, sy(gy)); ctx.stroke();
    }

    ctx.strokeStyle = css('--muted'); ctx.lineWidth = 2;
    ctx.beginPath(); ctx.moveTo(0, sy(0)); ctx.lineTo(W, sy(0));
    ctx.moveTo(sx(0), 0); ctx.lineTo(sx(0), H); ctx.stroke();

    const { a, b, c } = state;
    ctx.strokeStyle = css('--primary'); ctx.lineWidth = 3;
    ctx.beginPath();
    for (let px = 0; px <= W; px++) {
      const x = (px - W / 2) / 28;
      const y = a * x * x + b * x + c;
      const py = sy(y);
      px === 0 ? ctx.moveTo(px, py) : ctx.lineTo(px, py);
    }
    ctx.stroke();

    // Depe nokady we kökler — okuwçy parametri üýtgedende bular hem üýtgeýär.
    let text = `y = ${a}x² + ${b}x + ${c}`;
    if (a !== 0) {
      const vx = -b / (2 * a);
      const vy = a * vx * vx + b * vx + c;
      ctx.fillStyle = css('--danger');
      ctx.beginPath(); ctx.arc(sx(vx), sy(vy), 6, 0, Math.PI * 2); ctx.fill();

      const d = b * b - 4 * a * c;
      text += ` · depe (${vx.toFixed(2)}; ${vy.toFixed(2)}) · diskriminant D = ${d.toFixed(2)}`;
      if (d > 0) {
        const r1 = (-b - Math.sqrt(d)) / (2 * a), r2 = (-b + Math.sqrt(d)) / (2 * a);
        text += ` · kökler x₁ = ${r1.toFixed(2)}, x₂ = ${r2.toFixed(2)}`;
        ctx.fillStyle = css('--ok');
        for (const r of [r1, r2]) { ctx.beginPath(); ctx.arc(sx(r), sy(0), 5, 0, Math.PI * 2); ctx.fill(); }
      } else if (d === 0) text += ' · bir kök';
      else text += ' · hakyky kök ýok';
    }
    readout.textContent = text;
  }

  const sa = slider('a', { min: -3, max: 3, step: 0.1, value: state.a }, (v) => { state.a = v; draw(); });
  const sb = slider('b', { min: -6, max: 6, step: 0.5, value: state.b }, (v) => { state.b = v; draw(); });
  const sc = slider('c', { min: -6, max: 6, step: 0.5, value: state.c }, (v) => { state.c = v; draw(); });

  host.append(canvas, sa.row, sb.row, sc.row, readout);
  draw();
}

/* ---------- Fizika: elektrik zynjyry, Om kanuny ---------- */
function circuit(host, cfg) {
  const W = 560, H = 260;
  const canvas = el('canvas');
  const readout = el('div', { class: 'demo-readout' });
  const state = { u: cfg.u ?? 12, r: cfg.r ?? 6 };
  const ctx = hiDPI(canvas, W, H);

  function draw() {
    const i = state.u / state.r;        // Om kanuny: I = U / R
    const p = state.u * i;              // Kuwwat: P = U · I

    ctx.clearRect(0, 0, W, H);
    ctx.strokeStyle = css('--text'); ctx.lineWidth = 3;
    ctx.strokeRect(80, 50, 400, 160);

    // Çeşme
    ctx.fillStyle = css('--surface'); ctx.fillRect(50, 110, 60, 40);
    ctx.strokeRect(50, 110, 60, 40);
    ctx.fillStyle = css('--text'); ctx.font = '600 14px system-ui'; ctx.textAlign = 'center';
    ctx.fillText(`${state.u} V`, 80, 135);

    // Garşylyk
    ctx.fillStyle = css('--surface'); ctx.fillRect(250, 30, 70, 40);
    ctx.strokeRect(250, 30, 70, 40);
    ctx.fillStyle = css('--text');
    ctx.fillText(`${state.r} Ω`, 285, 55);

    // Lampa: ýagtylygy kuwwata bagly — hereket netijä täsir edýär
    const glow = Math.min(1, p / 60);
    ctx.beginPath(); ctx.arc(450, 130, 26, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(255, 196, 60, ${0.12 + glow * 0.88})`;
    ctx.fill();
    ctx.strokeStyle = css('--text'); ctx.lineWidth = 2; ctx.stroke();

    readout.textContent =
      `I = U / R = ${state.u} / ${state.r} = ${i.toFixed(2)} A · P = U · I = ${p.toFixed(1)} Wt`;
  }

  const su = slider('Naprýaženiýe', { min: 1, max: 24, step: 1, value: state.u, unit: ' V' },
    (v) => { state.u = v; draw(); });
  const sr = slider('Garşylyk', { min: 1, max: 24, step: 1, value: state.r, unit: ' Ω' },
    (v) => { state.r = v; draw(); });

  host.append(canvas, su.row, sr.row, readout);
  draw();
}

/* ---------- Himiýa: reaksiýany deňlemek ---------- */
function reaction(host, cfg) {
  // cfg: { left:[{f:'H2', atoms:{H:2}}], right:[...] }
  const left = cfg.left || [{ f: 'H₂', atoms: { H: 2 } }, { f: 'O₂', atoms: { O: 2 } }];
  const right = cfg.right || [{ f: 'H₂O', atoms: { H: 2, O: 1 } }];
  const coef = [...left, ...right].map(() => 1);
  const readout = el('div', { class: 'demo-readout' });
  const eqLine = el('div', { class: 'row', style: 'font-size:1.2rem;font-weight:700;flex-wrap:wrap' });

  function stepper(idx) {
    const num = el('span', { text: String(coef[idx]), style: 'min-width:22px;text-align:center' });
    const dec = el('button', { text: '−', 'aria-label': 'azalt', onclick: () => { if (coef[idx] > 1) { coef[idx]--; num.textContent = coef[idx]; check(); } } });
    const inc = el('button', { text: '+', 'aria-label': 'köpelt', onclick: () => { if (coef[idx] < 9) { coef[idx]++; num.textContent = coef[idx]; check(); } } });
    for (const b of [dec, inc]) b.style.minHeight = '36px', b.style.padding = '2px 10px';
    return el('span', { class: 'row', style: 'gap:4px' }, [dec, num, inc]);
  }

  function tally(list, offset) {
    const sum = {};
    list.forEach((item, i) => {
      for (const [atom, n] of Object.entries(item.atoms)) {
        sum[atom] = (sum[atom] || 0) + n * coef[offset + i];
      }
    });
    return sum;
  }

  function check() {
    const l = tally(left, 0), r = tally(right, left.length);
    const atoms = new Set([...Object.keys(l), ...Object.keys(r)]);
    const parts = [];
    let balanced = true;

    for (const a of atoms) {
      const lv = l[a] || 0, rv = r[a] || 0;
      if (lv !== rv) balanced = false;
      parts.push(`${a}: ${lv} ${lv === rv ? '=' : '≠'} ${rv}`);
    }
    readout.textContent = (balanced ? '✓ Reaksiýa deňlenen. ' : '✗ Entek deňlenmedi. ') + parts.join(' · ');
    readout.style.color = balanced ? css('--ok') : css('--danger');
  }

  left.forEach((item, i) => {
    if (i) eqLine.append(el('span', { text: '+' }));
    eqLine.append(stepper(i), el('span', { text: item.f }));
  });
  eqLine.append(el('span', { text: '→', style: 'margin:0 8px' }));
  right.forEach((item, i) => {
    if (i) eqLine.append(el('span', { text: '+' }));
    eqLine.append(stepper(left.length + i), el('span', { text: item.f }));
  });

  host.append(eqLine, readout);
  check();
}

/* ---------- Biologiýa: öýjügiň bölekleri ---------- */
function cell(host, cfg) {
  const W = 560, H = 320;
  const canvas = el('canvas');
  const readout = el('div', { class: 'demo-readout', text: 'Öýjügiň bölegine basyň.' });
  const ctx = hiDPI(canvas, W, H);

  const parts = cfg.parts || [
    { name: 'Ýadro', x: 280, y: 160, r: 46, info: 'Ýadro — nesil maglumatyny saklaýar we öýjügi dolandyrýar.' },
    { name: 'Mitohondriýa', x: 160, y: 100, r: 30, info: 'Mitohondriýa — energiýa (ATF) öndürýär.' },
    { name: 'Ribosoma', x: 420, y: 110, r: 22, info: 'Ribosoma — belok sintezini amala aşyrýar.' },
    { name: 'Wakuola', x: 170, y: 230, r: 34, info: 'Wakuola — suw we iýmit maddalaryny saklaýar.' },
    { name: 'Membrana', x: 430, y: 235, r: 26, info: 'Membrana — öýjügi gorap, madda alyş-çalşyny sazlaýar.' },
  ];
  let active = -1;

  function draw() {
    ctx.clearRect(0, 0, W, H);
    ctx.fillStyle = css('--surface-2');
    ctx.beginPath(); ctx.ellipse(W / 2, H / 2, 258, 142, 0, 0, Math.PI * 2); ctx.fill();
    ctx.strokeStyle = css('--text'); ctx.lineWidth = 3; ctx.stroke();

    parts.forEach((p, i) => {
      ctx.beginPath(); ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
      ctx.fillStyle = i === active ? css('--primary') : css('--surface');
      ctx.fill();
      ctx.lineWidth = i === active ? 4 : 2;
      ctx.strokeStyle = i === active ? css('--primary') : css('--muted');
      ctx.stroke();

      ctx.fillStyle = i === active ? css('--primary-ink') : css('--text');
      ctx.font = '600 12px system-ui'; ctx.textAlign = 'center';
      ctx.fillText(p.name, p.x, p.y + 4);
    });
  }

  function pick(mx, my) {
    active = parts.findIndex((p) => Math.hypot(mx - p.x, my - p.y) <= p.r);
    readout.textContent = active >= 0 ? parts[active].info : 'Öýjügiň bölegine basyň.';
    draw();
  }

  canvas.addEventListener('click', (e) => {
    const rect = canvas.getBoundingClientRect();
    pick((e.clientX - rect.left) * (W / rect.width), (e.clientY - rect.top) * (H / rect.height));
  });
  // Klawiatura bilen hem işlemeli (9-njy bölüm).
  canvas.tabIndex = 0;
  canvas.setAttribute('role', 'application');
  canvas.setAttribute('aria-label', 'Öýjügiň gurluşy');
  canvas.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') { active = (active + 1) % parts.length; }
    else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') { active = (active - 1 + parts.length) % parts.length; }
    else return;
    e.preventDefault();
    readout.textContent = parts[active].info;
    draw();
  });

  host.append(canvas, readout);
  draw();
}

/* ---------- Informatika: algoritmiň ädimleri ---------- */
function algorithm(host, cfg) {
  const W = 560, H = 200;
  const canvas = el('canvas');
  const readout = el('div', { class: 'demo-readout' });
  const ctx = hiDPI(canvas, W, H);

  const initial = cfg.array || [5, 2, 9, 1, 7, 3];
  let arr = [...initial], i = 0, j = 0, done = false, swaps = 0, compares = 0;

  function reset() {
    arr = [...initial]; i = j = swaps = compares = 0; done = false;
    readout.textContent = 'Ädim düwmesine basyp, «köpürjik» tertipleşdirmesini yzarlaň.';
    draw();
  }

  function step() {
    if (done) return;
    if (i >= arr.length - 1) {
      done = true;
      readout.textContent = `Tamamlandy. Deňeşdirme: ${compares}, çalyşma: ${swaps}.`;
      draw();
      return;
    }
    compares++;
    if (arr[j] > arr[j + 1]) {
      [arr[j], arr[j + 1]] = [arr[j + 1], arr[j]];
      swaps++;
      readout.textContent = `${arr[j + 1]} > ${arr[j]} — çalyşdyk. Deňeşdirme: ${compares}, çalyşma: ${swaps}.`;
    } else {
      readout.textContent = `${arr[j]} ≤ ${arr[j + 1]} — çalyşmak gerek däl. Deňeşdirme: ${compares}.`;
    }
    j++;
    if (j >= arr.length - 1 - i) { i++; j = 0; }
    draw();
  }

  function draw() {
    ctx.clearRect(0, 0, W, H);
    const bw = Math.min(70, (W - 40) / arr.length);
    arr.forEach((v, idx) => {
      const h = v * 16;
      const x = 20 + idx * bw, y = H - 30 - h;
      const hot = !done && (idx === j || idx === j + 1);
      ctx.fillStyle = hot ? css('--primary') : css('--surface-2');
      ctx.strokeStyle = hot ? css('--primary') : css('--border');
      ctx.lineWidth = 2;
      ctx.fillRect(x, y, bw - 8, h);
      ctx.strokeRect(x, y, bw - 8, h);
      ctx.fillStyle = css('--text');
      ctx.font = '700 14px system-ui'; ctx.textAlign = 'center';
      ctx.fillText(String(v), x + (bw - 8) / 2, H - 12);
    });
  }

  host.append(
    canvas,
    el('div', { class: 'btn-row' }, [
      el('button', { class: 'btn-primary', text: 'Ädim', onclick: step }),
      el('button', { text: 'Başyndan', onclick: reset }),
    ]),
    readout,
  );
  reset();
}

const REGISTRY = {
  function_plot: functionPlot,
  circuit,
  reaction,
  cell,
  algorithm,
};

export function mountDemo(hostEl, kind, config) {
  hostEl.innerHTML = '';
  const build = REGISTRY[kind];
  if (!build) {
    hostEl.append(el('p', { class: 'muted', text: 'Bu sapak üçin interaktiw görkezme ýok.' }));
    return;
  }
  build(hostEl, config || {});
}

export const DEMO_KINDS = Object.keys(REGISTRY);
