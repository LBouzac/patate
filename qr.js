// QR code (mode octet, correction M, versions 1 à 10) : de quoi encoder l'adresse de la manette, sans bibliothèque.
// qrMatrix(texte) renvoie un tableau de lignes de booléens (true : module noir), sans la marge.
(function (root) {
  // GF(256), polynôme 0x11d
  const EXP = new Uint8Array(512), LOG = new Uint8Array(256);
  for (let i = 0, x = 1; i < 255; i++) { EXP[i] = x; LOG[x] = i; x <<= 1; if (x & 256) x ^= 0x11d; }
  for (let i = 255; i < 512; i++) EXP[i] = EXP[i - 255];
  const mul = (a, b) => (a && b) ? EXP[LOG[a] + LOG[b]] : 0;

  function rsGen(n) { // polynôme générateur de degré n
    let g = [1];
    for (let i = 0; i < n; i++) {
      const r = new Array(g.length + 1).fill(0);
      for (let j = 0; j < g.length; j++) { r[j] ^= g[j]; r[j + 1] ^= mul(g[j], EXP[i]); }
      g = r;
    }
    return g;
  }
  function rsEC(data, n) {
    const g = rsGen(n), r = [...data, ...new Array(n).fill(0)];
    for (let i = 0; i < data.length; i++) {
      const c = r[i];
      if (c) for (let j = 0; j < g.length; j++) r[i + j] ^= mul(g[j], c);
    }
    return r.slice(data.length);
  }

  // niveau M : [codes de correction par bloc, [nombre de blocs, octets de données par bloc]…]
  const BLOCKS = [null,
    [10, [1, 16]], [16, [1, 28]], [26, [1, 44]], [18, [2, 32]], [24, [2, 43]],
    [16, [4, 27]], [18, [4, 31]], [22, [2, 38], [2, 39]], [22, [3, 36], [2, 37]], [26, [4, 43], [1, 44]]];
  const ALIGN = [null, [], [6, 18], [6, 22], [6, 26], [6, 30], [6, 34], [6, 22, 38], [6, 24, 42], [6, 26, 46], [6, 28, 50]];
  const dataLen = v => BLOCKS[v].slice(1).reduce((s, [n, k]) => s + n * k, 0);

  function bch(value, poly, bits) { // reste de la division, pour les infos de format et de version
    let v = value << bits;
    const top = 31 - Math.clz32(poly);
    while (31 - Math.clz32(v) >= top) v ^= poly << (31 - Math.clz32(v) - top);
    return (value << bits) | v;
  }

  const MASKS = [
    (x, y) => (x + y) % 2 === 0, (x, y) => y % 2 === 0, (x, y) => x % 3 === 0, (x, y) => (x + y) % 3 === 0,
    (x, y) => (Math.floor(y / 2) + Math.floor(x / 3)) % 2 === 0, (x, y) => (x * y) % 2 + (x * y) % 3 === 0,
    (x, y) => ((x * y) % 2 + (x * y) % 3) % 2 === 0, (x, y) => ((x + y) % 2 + (x * y) % 3) % 2 === 0];

  function qrMatrix(text, forceMask) {
    const bytes = new TextEncoder().encode(text);
    let v = 1;
    while (v <= 10 && dataLen(v) < bytes.length + (v < 10 ? 2 : 3)) v++;
    if (v > 10) throw new Error('Texte trop long pour le QR code');
    // flux de bits : mode octet, longueur, données, fin, octets de remplissage
    const bits = [];
    const put = (val, n) => { for (let i = n - 1; i >= 0; i--) bits.push((val >>> i) & 1); };
    put(4, 4); put(bytes.length, v < 10 ? 8 : 16);
    for (const b of bytes) put(b, 8);
    const cap = dataLen(v) * 8;
    put(0, Math.min(4, cap - bits.length));
    while (bits.length % 8) bits.push(0);
    const data = [];
    for (let i = 0; i < bits.length; i += 8) data.push(parseInt(bits.slice(i, i + 8).join(''), 2));
    for (let p = 0; data.length < dataLen(v); p++) data.push(p % 2 ? 0x11 : 0xec);
    // blocs, correction d'erreurs, entrelacement
    const [ecn, ...groups] = BLOCKS[v], blocks = [];
    let at = 0;
    for (const [n, k] of groups) for (let i = 0; i < n; i++) { const d = data.slice(at, at += k); blocks.push([d, rsEC(d, ecn)]); }
    const words = [];
    for (let i = 0; i < Math.max(...blocks.map(b => b[0].length)); i++) for (const b of blocks) if (i < b[0].length) words.push(b[0][i]);
    for (let i = 0; i < ecn; i++) for (const b of blocks) words.push(b[1][i]);

    const size = 17 + 4 * v;
    const m = Array.from({ length: size }, () => new Array(size).fill(false));
    const fixed = Array.from({ length: size }, () => new Array(size).fill(false));
    const set = (x, y, on) => { if (x >= 0 && y >= 0 && x < size && y < size) { m[y][x] = on; fixed[y][x] = true; } };
    for (const [ox, oy] of [[0, 0], [size - 7, 0], [0, size - 7]]) // repères des coins, avec leur bordure blanche
      for (let y = -1; y <= 7; y++) for (let x = -1; x <= 7; x++) {
        const d = Math.max(Math.abs(x - 3), Math.abs(y - 3));
        set(ox + x, oy + y, d !== 2 && d !== 4);
      }
    for (let i = 8; i < size - 8; i++) { set(i, 6, i % 2 === 0); set(6, i, i % 2 === 0); } // lignes de synchronisation
    const al = ALIGN[v];
    for (const ay of al) for (const ax of al) {
      const last = al[al.length - 1];
      if ((ax === 6 && ay === 6) || (ax === 6 && ay === last) || (ax === last && ay === 6)) continue; // sous un repère de coin
      for (let y = -2; y <= 2; y++) for (let x = -2; x <= 2; x++) set(ax + x, ay + y, Math.max(Math.abs(x), Math.abs(y)) !== 1);
    }
    set(8, size - 8, true); // module noir obligatoire
    for (let i = 0; i < 9; i++) { // zones réservées au format
      if (!fixed[8][i]) set(i, 8, false);
      if (!fixed[i][8]) set(8, i, false);
    }
    for (let i = 0; i < 8; i++) { set(size - 1 - i, 8, false); set(8, size - 1 - i, false); }
    set(8, size - 8, true);
    if (v >= 7) { // version
      const vi = bch(v, 0x1f25, 12);
      for (let i = 0; i < 18; i++) {
        const on = ((vi >>> i) & 1) === 1, a = Math.floor(i / 3), b = size - 11 + (i % 3);
        set(a, b, on); set(b, a, on);
      }
    }
    // données en zigzag, deux colonnes à la fois, de bas en haut puis de haut en bas
    const stream = [];
    for (const w of words) for (let i = 7; i >= 0; i--) stream.push((w >>> i) & 1);
    let k = 0;
    for (let right = size - 1, up = true; right >= 1; right -= 2, up = !up) {
      if (right === 6) right = 5; // on saute la colonne de synchronisation
      for (let n = 0; n < size; n++) {
        const y = up ? size - 1 - n : n;
        for (const x of [right, right - 1]) if (!fixed[y][x]) m[y][x] = k < stream.length ? stream[k++] === 1 : false;
      }
    }

    function format(mask) { // niveau M (00) + masque, avec sa correction, aux deux endroits prévus
      const f = bch(mask, 0x537, 10) ^ 0x5412;
      const bit = i => ((f >>> i) & 1) === 1;
      for (let i = 0; i < 15; i++) {
        // colonne 8 : en haut (en sautant la ligne de synchronisation), puis en bas
        if (i < 6) m[i][8] = bit(i); else if (i < 8) m[i + 1][8] = bit(i); else m[size - 15 + i][8] = bit(i);
        // ligne 8 : à droite, puis à gauche (en sautant la colonne de synchronisation)
        if (i < 8) m[8][size - 1 - i] = bit(i); else if (i < 9) m[8][7] = bit(i); else m[8][14 - i] = bit(i);
      }
      m[size - 8][8] = true;
    }
    function apply(mask) {
      const f = MASKS[mask];
      for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) if (!fixed[y][x] && f(x, y)) m[y][x] = !m[y][x];
    }
    function penalty() {
      let p = 0;
      const line = get => { // règles 1 et 3 sur une ligne
        let run = 1;
        for (let i = 1; i <= size; i++) {
          if (i < size && get(i) === get(i - 1)) run++;
          else { if (run >= 5) p += run - 2; run = 1; }
        }
        for (let i = 0; i + 11 <= size; i++) {
          const s = []; for (let j = 0; j < 11; j++) s.push(get(i + j) ? 1 : 0);
          const t = s.join('');
          if (t === '10111010000' || t === '00001011101') p += 40;
        }
      };
      for (let y = 0; y < size; y++) line(x => m[y][x]);
      for (let x = 0; x < size; x++) line(y => m[y][x]);
      let dark = 0;
      for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) {
        if (m[y][x]) dark++;
        if (x < size - 1 && y < size - 1 && m[y][x] === m[y][x + 1] && m[y][x] === m[y + 1][x] && m[y][x] === m[y + 1][x + 1]) p += 3;
      }
      p += Math.floor(Math.abs(dark * 20 - size * size * 10) / (size * size)) * 10;
      return p;
    }
    let best = forceMask ?? 0;
    if (forceMask === undefined) {
      let low = Infinity;
      for (let mask = 0; mask < 8; mask++) {
        apply(mask); format(mask);
        const p = penalty();
        if (p < low) { low = p; best = mask; }
        apply(mask); // on enlève le masque (le XOR s'annule)
      }
    }
    apply(best); format(best);
    return m;
  }

  // qrSVG : un SVG carré (avec la marge blanche réglementaire de 4 modules)
  function qrSVG(text) {
    const m = qrMatrix(text), n = m.length + 8;
    let d = '';
    m.forEach((row, y) => row.forEach((on, x) => { if (on) d += `M${x + 4} ${y + 4}h1v1h-1z`; }));
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${n} ${n}" shape-rendering="crispEdges"><rect width="${n}" height="${n}" fill="#fff"/><path d="${d}" fill="#000"/></svg>`;
  }

  root.qrMatrix = qrMatrix;
  root.qrSVG = qrSVG;
})(typeof window !== 'undefined' ? window : globalThis);
