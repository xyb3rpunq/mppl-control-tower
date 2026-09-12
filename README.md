# Control Tower MPPL

Ruang kendali kuantitatif untuk proyek perangkat lunak, ditulis dalam Go.

**Situs live:** https://xyb3rpunq.github.io/mppl-control-tower/
**Bahasa Inggris:** https://xyb3rpunq.github.io/mppl-control-tower/en/

---

## Masalah yang dipecahkan

Proyek perangkat lunak di Indonesia rutin gagal pada jadwal dan biaya, dan
nyaris tidak ada yang menjalankan hitungan yang sebenarnya sudah bisa
memperingatkan sejak awal. Perkakas yang mampu melakukannya — MS Project,
Primavera — berbayar dan tertutup, dan tidak satu pun berbahasa Indonesia.

Aplikasi ini menjalankan hitungan itu sebagai kode Go yang terbuka dan teruji:
CPM dengan empat relasi PDM, PERT, Monte Carlo sepuluh ribu iterasi, Earned
Value lengkap sampai Earned Schedule, Expected Monetary Value untuk menetapkan
cadangan, dan Seven Basic Tools of Quality beserta aturan keputusannya.

Lalu membuktikannya pada satu kasus nyata berskala nasional:
[**bedah kasus Coretax DJP**](https://xyb3rpunq.github.io/mppl-control-tower/coretax/) —
proyek Rp 1,34 triliun yang diluncurkan serentak untuk seluruh wajib pajak
Indonesia pada 1 Januari 2025, dan dalam bulan pertama menurunkan penerimaan
perpajakan sebesar 34,5%.

Pola kegagalannya identik dengan proyek kuliah Rp 14,8 juta yang dianalisis di
situs ini. Hanya skalanya yang berbeda sekitar sembilan puluh ribu kali.

---

## Satu basis kode, dua tempat berjalan

Paket hitung di `internal/` dipakai dua kali:

**Sebagai generator situs statis.** `cmd/site` merender seluruh halaman dan
seluruh grafik SVG di sisi server. Konsekuensinya baik: situs tetap utuh tanpa
JavaScript, bisa dicetak jadi PDF apa adanya, terbaca mesin pengindeks, dan
tidak ada kedipan saat muat.

**Sebagai WebAssembly.** `cmd/wasm` mengompilasi paket yang sama untuk peramban,
sehingga menggeser tanggal data akan menghitung ulang CPM, Earned Value, dan
Monte Carlo secara langsung — tanpa server.

Tidak ada rumus yang ditulis dua kali, jadi tidak mungkin ada versi JavaScript
yang diam-diam berbeda dari versi Go. Panel interaktif di halaman Biaya
menghasilkan SPI 0,9153 dan CPI 0,9180 — angka yang sama persis dengan yang
dirender server.

---

## Isi situs

| Halaman | Isi |
| --- | --- |
| Ruang Kendali | Lima angka penentu, kurva-S Earned Value, tujuh temuan yang diturunkan dari ambang |
| Piagam & Lingkup | Project Charter, WBS 5 fase / 15 paket / 35 aktivitas, pemeriksaan aturan 100% |
| Jadwal & Jalur Kritis | Gantt dengan float dan realisasi, diagram jaringan AON, tabel CPM lengkap |
| PERT & Monte Carlo | Estimasi tiga titik, Z-score, 10.000 iterasi, diagram tornado sensitivitas |
| Biaya & Earned Value | PV/EV/AC, SPI, CPI, tiga varian EAC, TCPI, Earned Schedule, panel interaktif |
| Manajemen Risiko | 12 risiko dengan EMV, peta panas 5×5 sebelum/sesudah mitigasi, uji kecukupan cadangan |
| Organisasi & Sumber Daya | Bagan organisasi, RACI tervalidasi, histogram beban, grid kuasa-kepentingan |
| Manajemen Mutu | Peta kendali beraturan Nelson, Pareto berbobot keparahan, fishbone, biaya kualitas |
| Bedah Kasus: Coretax | Fakta bersumber, metrik turunan, perbandingan strategi transisi dengan EMV |
| Referensi Rumus | 24 rumus dengan simbol, cara membaca, jebakan, dan contoh hitung dari data hidup |
| Materi & Area Pengetahuan | 9 area pengetahuan, 4 tahap siklus hidup, peta materi kuliah ke fitur |
| Metode & Sumber | Cara menghitungnya, asumsi yang dipakai, dan apa yang belum tertutup |

Dua belas rute, dua bahasa, 24 halaman.

---

## Temuan utama

Semuanya diturunkan dari angka, bukan ditulis tetap. Kalau datanya diperbaiki,
temuannya ikut hilang dengan sendirinya.

- **Proyeksi biaya akhir Rp 16.157.314 melewati pagu Rp 16.000.000.** Dengan CPI
  0,918, seluruh cadangan kontinjensi dan manajemen habis terpakai dan masih
  kurang Rp 157.314.
- **Komitmen 17 minggu punya peluang 1,4%.** Dari 10.000 iterasi Monte Carlo,
  hanya 1,4% yang selesai dalam 85 hari kerja. P80 berada di 98 hari kerja.
- **Cadangan kontinjensi hanya menutup 17% paparan.** EMV residual seluruh
  risiko Rp 2.880.000; cadangan yang tersedia Rp 500.000.
- **Tanggal selesai piagam salah lima hari kerja.** Tujuh belas minggu kalender
  polos berakhir 13 Februari 2026; 85 hari kerja sesungguhnya berakhir
  20 Februari 2026 setelah akhir pekan dan lima hari libur dikeluarkan.
- **Jadwalnya sah secara matematis tetapi tidak bisa dijalankan.** Ada 26
  hari-peran dengan beban melebihi kapasitas — satu Backend Developer
  dijadwalkan mengerjakan dua API pada hari yang sama.
- **Waktu respons bergeser sistematis, bukan berfluktuasi.** Tidak satu pun
  nilai melewati batas spesifikasi 3 detik, tetapi dua aturan Nelson terpicu.

---

## Menjalankan secara lokal

```bash
go test ./...                                   # 10 paket, termasuk uji render HTML sungguhan
go run ./cmd/site -out dist -base ""            # bangun situs statis
GOOS=js GOARCH=wasm go build -o web/static/js/mppl.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/static/js/
cd dist && python -m http.server 8231           # buka http://127.0.0.1:8231
```

Bendera `-status` menggeser tanggal data pelaporan Earned Value:

```bash
go run ./cmd/site -out dist -base "" -status 2026-01-09
```

---

## Susunan kode

```
internal/
  workcal    kalender hari kerja, hari libur, pemetaan indeks ke tanggal
  schedule   CPM empat relasi PDM (FS/SS/FF/SF) dengan lag, float, jalur kritis, PERT
  evm        PV, EV, AC, indeks kinerja, tiga EAC, TCPI, Earned Schedule
  simulate   Monte Carlo, PRNG mulberry32 berbenih, beta-PERT, korelasi Spearman
  risk       EMV, matriks probabilitas-dampak 5×5, kecukupan cadangan
  resource   histogram pembebanan, deteksi over-alokasi, kehalusan kurva tim
  quality    peta kendali X-bar, aturan Nelson, Pareto berbobot, Cpk, biaya kualitas
  coretax    metrik turunan studi kasus, perbandingan strategi transisi
  render     pembangun SVG dan seluruh jenis grafik
  model      sumber tunggal kebenaran seluruh data proyek
  site       perakit analisis dan penurun temuan
  i18n       kamus teks antarmuka dwibahasa
cmd/
  site       generator situs statis
  wasm       mesin hitung untuk peramban
web/
  templates  12 templat halaman + kerangka
  static     CSS, JavaScript, service worker, manifest
```

**Nol dependensi pihak ketiga.** Seluruh grafik, statistik, dan format angka
ditulis di sini — yang berarti setiap angka bisa ditelusuri sampai ke barisnya.

---

## Yang dijaga oleh uji

Uji di repositori ini bukan sekadar memeriksa fungsi berjalan; sebagian besar
menjaga klaim yang ditampilkan di situs tetap benar.

- **Durasi jaringan harus 85 hari kerja.** Kalau ada yang menambah aktivitas
  tanpa menyesuaikan yang lain, uji inilah yang menangkapnya sebelum pembaca.
- **Jalur kritis harus benar-benar tersambung.** Memfilter `TotalFloat == 0`
  saja bisa memunculkan rantai yang terputus.
- **Anggaran berlapis harus rekonsiliasi.** BAC + kontinjensi + cadangan
  manajemen harus persis sama dengan pagu Rp 16.000.000.
- **Matriks RACI harus punya tepat satu Accountable per baris.** Dua A berarti
  tidak ada yang benar-benar bertanggung jawab.
- **Simulasi harus deterministik.** Benih yang sama wajib menghasilkan angka
  yang sama, atau dasbor akan bergoyang sendiri dan mustahil diverifikasi.
- **Rerata Monte Carlo harus melebihi durasi deterministik.** Kalau tidak,
  penjelasan merge bias di halaman PERT ikut salah.
- **Halaman Inggris tidak boleh memuat kata fungsi Indonesia.** Memeriksa
  kelengkapan kamus tidak cukup — teks bisa lengkap tetapi templatnya lupa
  memilih cabang bahasa. Uji ini merender HTML sungguhan lalu membacanya, dan
  memang menemukan empat kebocoran nyata saat pertama dijalankan.
- **Setiap fakta Coretax harus punya URL sumber.** Analisis tentang angka yang
  tidak pernah diperiksa tidak boleh memuat angka yang tidak pernah diperiksa.

---

## Data terbuka

| Berkas | Isi |
| --- | --- |
| [`data/metrik.json`](https://xyb3rpunq.github.io/mppl-control-tower/data/metrik.json) | Seluruh metrik Earned Value, simulasi, risiko, dan mutu |
| [`data/aktivitas.csv`](https://xyb3rpunq.github.io/mppl-control-tower/data/aktivitas.csv) | 40 simpul dengan ES, EF, LS, LF, float, dan Earned Value |
| [`data/risiko.csv`](https://xyb3rpunq.github.io/mppl-control-tower/data/risiko.csv) | 12 risiko dengan EMV inheren dan residual |

---

## Sumber

Data proyek berasal dari tugas mata kuliah Manajemen Proyek Perangkat Lunak,
Universitas Esa Unggul (Tugas 2, 3, 5, 6, dan 10), serta materi Modul 2, Modul 4,
Pertemuan 3, Pertemuan 7, dan Pertemuan 9.

Studi kasus Coretax bersumber dari pemberitaan publik Kompas, Tempo,
Hukumonline, DDTC News, Beritasatu, dan keterangan resmi Direktorat Jenderal
Pajak; daftar lengkapnya ada di
[halaman studi kasus](https://xyb3rpunq.github.io/mppl-control-tower/coretax/).

Situs ini tidak berafiliasi dengan Direktorat Jenderal Pajak maupun pihak mana
pun yang disebut. Analisisnya adalah kerja akademik.

## Lisensi

MIT.
