/* Control Tower MPPL — lapisan JavaScript.
 *
 * Sengaja tipis. Seluruh isi halaman dan seluruh grafik sudah dirender di sisi
 * server sebagai HTML dan SVG, jadi halaman ini utuh tanpa satu baris pun
 * JavaScript berjalan. Berkas ini hanya menambahkan tiga hal:
 *
 *   1. pengalih tema yang diingat antar kunjungan,
 *   2. pemuat WebAssembly yang menyalakan kendali interaktif bila tersedia,
 *   3. pendaftaran service worker agar situs bisa dibuka luring.
 *
 * Kalau WebAssembly gagal dimuat - jaringan lambat, peramban lama, apa pun -
 * halaman tetap menampilkan angka yang benar untuk tanggal data bawaan.
 */

(function () {
  'use strict';

  // ------------------------------------------------------------ tema
  var root = document.documentElement;

  function applyTheme(mode) {
    if (mode) {
      root.dataset.theme = mode;
    } else {
      delete root.dataset.theme;
    }
    try { localStorage.setItem('ct-theme', mode || ''); } catch (e) { /* mode privat */ }
  }

  document.querySelectorAll('[data-theme-toggle]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var current = root.dataset.theme;
      if (!current) {
        // Belum pernah dipilih: lawan preferensi sistem saat ini.
        var systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        applyTheme(systemDark ? 'light' : 'dark');
      } else {
        applyTheme(current === 'dark' ? 'light' : 'dark');
      }
    });
  });

  // ------------------------------------------------------ format angka
  function fmt(v, decimals) {
    if (v === null || v === undefined || isNaN(v)) return '—';
    return new Intl.NumberFormat(document.documentElement.lang === 'en' ? 'en-US' : 'id-ID', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals
    }).format(v);
  }

  function rp(v) {
    var prefix = document.documentElement.lang === 'en' ? 'IDR ' : 'Rp ';
    return prefix + fmt(Math.round(v), 0);
  }

  function pct(v, decimals) { return fmt(v * 100, decimals === undefined ? 1 : decimals) + '%'; }

  // ------------------------------------------------- WebAssembly loader
  var base = (document.querySelector('link[rel="stylesheet"]') || {}).href || '';
  base = base.replace(/\/assets\/css\/app\.css.*$/, '');

  function loadWasm() {
    if (typeof WebAssembly !== 'object') return;
    if (!document.querySelector('[data-wasm-panel]')) return; // halaman ini tak butuh

    var s = document.createElement('script');
    s.src = base + '/assets/js/wasm_exec.js';
    s.onload = function () {
      if (typeof Go !== 'function') return;
      var go = new Go();
      var src = base + '/assets/js/mppl.wasm';
      var run = function (result) { go.run(result.instance); };
      if (WebAssembly.instantiateStreaming) {
        WebAssembly.instantiateStreaming(fetch(src), go.importObject).then(run).catch(fallback);
      } else {
        fallback();
      }
      function fallback() {
        fetch(src)
          .then(function (r) { return r.arrayBuffer(); })
          .then(function (b) { return WebAssembly.instantiate(b, go.importObject); })
          .then(run)
          .catch(function (err) { console.warn('WebAssembly tidak dapat dimuat:', err); });
      }
    };
    document.head.appendChild(s);
  }

  // Dipanggil dari Go setelah seluruh fungsi didaftarkan.
  window.mpplReady = function () {
    document.querySelectorAll('[data-wasm-panel]').forEach(function (panel) {
      panel.hidden = false;
    });
    wireDataDate();
    wireSimulation();
  };

  // -------------------------------------------------- kendali tanggal data
  function wireDataDate() {
    var input = document.querySelector('[data-status-date]');
    if (!input || typeof window.mpplRecompute !== 'function') return;

    var b = window.mpplBounds();
    if (b && b.ok) {
      input.min = b.start;
      // Dibatasi pada hari terakhir yang punya data realisasi, bukan akhir
      // proyek: di luar itu EV berhenti tumbuh sementara PV terus naik, dan
      // SPI yang anjlok akan terbaca sebagai temuan padahal hanya artefak
      // kehabisan data.
      input.max = b.actualsThrough || b.finish;
      if (!input.value) input.value = b.default;
      var note = document.querySelector('[data-bounds-note]');
      if (note) note.textContent = note.textContent.replace('__MAX__', b.actualsThrough || b.finish);
    }

    function update() {
      var res = window.mpplRecompute(input.value);
      if (!res || !res.ok) return;
      setText('[data-out="spi"]', fmt(res.spi, 4));
      setText('[data-out="cpi"]', fmt(res.cpi, 4));
      setText('[data-out="pv"]', rp(res.pv));
      setText('[data-out="ev"]', rp(res.ev));
      setText('[data-out="ac"]', rp(res.ac));
      setText('[data-out="sv"]', signed(res.sv));
      setText('[data-out="cv"]', signed(res.cv));
      setText('[data-out="eac"]', rp(res.eac));
      setText('[data-out="vac"]', signed(res.vac));
      setText('[data-out="tcpi"]', fmt(res.tcpi, 4));
      setText('[data-out="svt"]', fmt(res.svt, 2));
      setText('[data-out="complete"]', pct(res.complete));
      setText('[data-out="elapsed"]', pct(res.elapsed));
      toggleClass('[data-out="spi"]', 'neg', res.spi < 1);
      toggleClass('[data-out="cpi"]', 'neg', res.cpi < 1);
      toggleClass('[data-out="eac"]', 'neg', res.overCap);
    }

    input.addEventListener('input', update);
    update();
  }

  function signed(v) {
    var s = rp(Math.abs(v));
    if (v > 0) return '+' + s;
    if (v < 0) return '−' + s;
    return s;
  }

  // ----------------------------------------------------- kendali simulasi
  function wireSimulation() {
    var form = document.querySelector('[data-sim-form]');
    if (!form || typeof window.mpplSimulate !== 'function') return;

    form.addEventListener('submit', function (e) {
      e.preventDefault();
      var iterations = parseInt(form.querySelector('[name="iterations"]').value, 10) || 10000;
      var seed = parseInt(form.querySelector('[name="seed"]').value, 10) || 20210801;
      var dist = form.querySelector('[name="distribution"]').value;

      var btn = form.querySelector('button[type="submit"]');
      var original = btn.textContent;
      btn.disabled = true;
      btn.textContent = '…';

      // Beri peramban satu frame untuk menggambar keadaan sibuk sebelum
      // WebAssembly memblokir thread utama selama simulasi berjalan.
      requestAnimationFrame(function () {
        setTimeout(function () {
          var t0 = performance.now();
          var res = window.mpplSimulate(iterations, seed, dist);
          var ms = performance.now() - t0;
          btn.disabled = false;
          btn.textContent = original;
          if (!res || !res.ok) return;

          setText('[data-sim="onTime"]', pct(res.onTime, 2));
          setText('[data-sim="mean"]', fmt(res.mean, 1));
          setText('[data-sim="stdDev"]', fmt(res.stdDev, 2));
          setText('[data-sim="p50"]', fmt(res.p50, 0));
          setText('[data-sim="p80"]', fmt(res.p80, 0));
          setText('[data-sim="p90"]', fmt(res.p90, 0));
          setText('[data-sim="range"]', fmt(res.min, 0) + '–' + fmt(res.max, 0));
          setText('[data-sim="iterations"]', fmt(res.iterations, 0));
          setText('[data-sim="elapsed"]', fmt(ms, 0) + ' ms');
          drawHistogram(res);
        }, 0);
      });
    });
  }

  // Menggambar ulang histogram hasil simulasi baru. Sengaja memakai SVG
  // sederhana: yang digambar ulang hanyalah batangnya, bukan seluruh grafik
  // yang sudah dirender server.
  function drawHistogram(res) {
    var host = document.querySelector('[data-sim-chart]');
    if (!host || !res.histogram || !res.histogram.length) return;

    var W = 880, H = 260, padL = 50, padR = 20, padT = 16, padB = 34;
    var plotW = W - padL - padR, plotH = H - padT - padB;
    var maxCount = 0;
    res.histogram.forEach(function (b) { if (b.count > maxCount) maxCount = b.count; });
    var lo = res.histogram[0].from, hi = res.histogram[res.histogram.length - 1].to;
    var x = function (v) { return padL + (v - lo) / (hi - lo) * plotW; };
    var y = function (v) { return padT + plotH - v / (maxCount * 1.12) * plotH; };

    var parts = ['<svg viewBox="0 0 ' + W + ' ' + H + '" class="chart histogram" role="img">'];
    parts.push('<line x1="' + padL + '" y1="' + (padT + plotH) + '" x2="' + (padL + plotW) +
      '" y2="' + (padT + plotH) + '" class="axis-line"/>');

    res.histogram.forEach(function (b) {
      var x1 = x(b.from), x2 = x(b.to), yy = y(b.count);
      parts.push('<rect x="' + (x1 + 0.5).toFixed(2) + '" y="' + yy.toFixed(2) +
        '" width="' + Math.max(x2 - x1 - 1, 0.5).toFixed(2) + '" height="' +
        (padT + plotH - yy).toFixed(2) + '" class="hist-bar"><title>' +
        fmt(b.from, 0) + '–' + fmt(b.to, 0) + ': ' + fmt(b.count, 0) + '</title></rect>');
    });

    var cum = res.histogram.map(function (b) {
      return [x((b.from + b.to) / 2), y(b.cum * maxCount * 1.12)];
    });
    parts.push('<path d="M' + cum.map(function (p) { return p[0].toFixed(2) + ' ' + p[1].toFixed(2); })
      .join(' L') + '" class="series cumulative"/>');

    [['deterministic', 'plan'], ['p50', 'p50'], ['p80', 'p80'], ['p90', 'p90']].forEach(function (m) {
      var v = res[m[0]];
      if (v === undefined) return;
      var px = x(v);
      parts.push('<line x1="' + px.toFixed(2) + '" y1="' + padT + '" x2="' + px.toFixed(2) +
        '" y2="' + (padT + plotH) + '" class="marker ' + m[1] + '"/>');
      parts.push('<text x="' + px.toFixed(2) + '" y="' + (padT - 4) +
        '" class="marker ' + m[1] + '" text-anchor="middle">' +
        (m[1] === 'plan' ? (document.documentElement.lang === 'en' ? 'plan ' : 'rencana ') : m[1].toUpperCase() + ' ') +
        fmt(v, 0) + '</text>');
    });

    for (var t = Math.ceil(lo / 5) * 5; t <= hi; t += 5) {
      parts.push('<text x="' + x(t).toFixed(2) + '" y="' + (padT + plotH + 16) +
        '" class="axis-label" text-anchor="middle">' + t + '</text>');
    }
    parts.push('</svg>');
    host.innerHTML = parts.join('');
  }

  // ------------------------------------------------------------- utilitas
  function setText(sel, value) {
    document.querySelectorAll(sel).forEach(function (el) { el.textContent = value; });
  }
  function toggleClass(sel, cls, on) {
    document.querySelectorAll(sel).forEach(function (el) { el.classList.toggle(cls, !!on); });
  }

  // ------------------------------------------------------ service worker
  if ('serviceWorker' in navigator && location.protocol !== 'file:') {
    window.addEventListener('load', function () {
      navigator.serviceWorker.register(base + '/sw.js', { scope: base + '/' })
        .catch(function () { /* luring bersifat opsional; kegagalan tak perlu diumumkan */ });
    });
  }

  loadWasm();
})();
