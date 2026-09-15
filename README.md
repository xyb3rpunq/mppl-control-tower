# Control Tower MPPL

**Ruang kendali kuantitatif untuk proyek perangkat lunak, ditulis dalam Go.**
Satu basis kode yang sama merender situs statis di server *dan* berjalan di peramban lewat WebAssembly.

[![uji](https://github.com/xyb3rpunq/mppl-control-tower/actions/workflows/ci.yml/badge.svg)](https://github.com/xyb3rpunq/mppl-control-tower/actions/workflows/ci.yml)
[![terbitkan](https://github.com/xyb3rpunq/mppl-control-tower/actions/workflows/pages.yml/badge.svg)](https://github.com/xyb3rpunq/mppl-control-tower/actions/workflows/pages.yml)

| | |
| --- | --- |
| **Situs live** | https://xyb3rpunq.github.io/mppl-control-tower/ |
| **Bahasa Inggris** | https://xyb3rpunq.github.io/mppl-control-tower/en/ |
| **Mata kuliah** | Manajemen Proyek Perangkat Lunak — Universitas Esa Unggul |
| **Proyek yang dianalisis** | Sistem Informasi Alumni & Tracer Study STIE Jayakusuma |
| **Studi kasus nasional** | Coretax DJP (Rp 1,34 triliun) |

---

## Daftar isi

1. [Masalah yang dipecahkan](#1-masalah-yang-dipecahkan)
2. [Temuan utama](#2-temuan-utama)
3. [Peta situs: 16 halaman × 2 bahasa](#3-peta-situs-16-halaman--2-bahasa)
4. [Mesin hitung](#4-mesin-hitung)
5. [Referensi 41 rumus](#5-referensi-41-rumus)
6. [Bedah kasus Coretax](#6-bedah-kasus-coretax)
7. [Interaktivitas lewat WebAssembly](#7-interaktivitas-lewat-webassembly)
8. [Data terbuka](#8-data-terbuka)
9. [Arsitektur](#9-arsitektur)
10. [Menjalankan secara lokal](#10-menjalankan-secara-lokal)
11. [Pengujian](#11-pengujian)
12. [CI/CD](#12-cicd)
13. [Asumsi, celah yang ditutup, dan batas yang tersisa](#13-asumsi-celah-yang-ditutup-dan-batas-yang-tersisa)
14. [Sumber data](#14-sumber-data)

---

## 1. Masalah yang dipecahkan

Proyek perangkat lunak di Indonesia rutin gagal pada jadwal dan biaya, dan nyaris tidak ada yang menjalankan hitungan yang sebenarnya sudah bisa memperingatkan sejak awal. Perkakas yang mampu melakukannya — MS Project, Primavera — berbayar, tertutup, dan tidak berbahasa Indonesia.

Aplikasi ini menjalankan seluruh hitungan itu sebagai kode Go yang terbuka dan teruji:

- **Penjadwalan** — CPM empat relasi PDM, PERT, GERT, penjadwalan berbatas sumber daya yang **terbukti optimal lewat batas bawah**, crashing **eksak dengan pemrograman linear**, fast-tracking
- **Ketidakpastian** — Monte Carlo 10.000 iterasi lima lapis: korelasi antar-aktivitas, risiko yang bergerombol, putaran rework, dan kapasitas nyata
- **Biaya** — Earned Value lengkap sampai Earned Schedule, biaya sewa yang bergantung waktu, struktur anggaran berlapis, Joint Confidence Level
- **Pengendalian** — prakiraan berjalan dari tanggal data yang belajar dari realisasi lewat kredibilitas Bühlmann
- **Risiko** — Expected Monetary Value inheren dan residual, uji kecukupan cadangan
- **Mutu** — Seven Basic Tools of Quality beserta aturan keputusannya

Lalu membuktikannya pada satu kasus nyata berskala nasional: **proyek Coretax DJP** senilai Rp 1,34 triliun, yang diluncurkan serentak untuk seluruh wajib pajak Indonesia pada 1 Januari 2025 dan dalam bulan pertama menurunkan penerimaan perpajakan sebesar 34,5%. Pola kegagalannya identik dengan proyek kuliah Rp 14,8 juta yang dianalisis situs ini — hanya skalanya yang berbeda sekitar sembilan puluh ribu kali.

---

## 2. Temuan utama

Semua temuan **diturunkan dari angka, bukan ditulis tetap**. Setiap temuan menyebut metrik pemicunya, dan kalau datanya diperbaiki, temuannya ikut hilang dengan sendirinya.

| # | Keparahan | Temuan | Angka |
| --- | --- | --- | --- |
| 1 | kritis | Proyeksi biaya akhir melewati pagu | EAC Rp 16.157.314 vs pagu Rp 16.000.000 (CPI 0,918) |
| 2 | kritis | Cadangan kontinjensi jauh di bawah paparan risiko | Cadangan Rp 500.000 menutup 17,4% dari EMV residual Rp 2.880.000 |
| 3 | kritis | Komitmen 17 minggu nyaris mustahil | Peluang selesai ≤ 85 hari kerja: 1,2%; P80 = 98 hari kerja |
| 4 | kritis | Jadwal 85 hari hanya sah di atas kertas | Jadwal levelling optimal: **113 hari kerja**, selesai 10 April 2026 (+15 dari kapasitas, +13 dari UTS & UAS) |
| 5 | kritis | Peluang tepat waktu *dan* tepat anggaran nyaris nol | JCL pada target piagam: 0,0%; komitmen JCL 70% = **147 hari kerja (5 Juni 2026) & Rp 23.681.409** |
| 6 | kritis | Dari tanggal data, P80 penyelesaian jauh melampaui prakiraan Earned Value | Prakiraan berjalan P80 **125 hari kerja** (28 April 2026) vs IEAC(t) 91,0 hari; komitmen JCL 70% berjalan 121 hari & Rp 21.819.542 |
| 7 | tinggi | Satu orang dijadwalkan pada dua pekerjaan sekaligus | 26 hari-peran over-alokasi; peran kritis: Backend Developer |
| 8 | tinggi | Waktu respons bergeser sistematis | 10 pelanggaran aturan Nelson walau semua nilai di bawah spesifikasi 3 detik |
| 9 | tinggi | Proyek tertinggal dalam satuan waktu | Earned Schedule: SV(t) = −2,90 hari kerja |
| 10 | tinggi | Dari tanggal data, percepatan termurah memajukan 8 hari seharga Rp 141.607 per hari | Aturan menurut nilai satu hari lebih cepat: **< Rp 141.607** tanpa percepatan (121 hari); **Rp 141.607–836.801** lembur sah BE, DBA, FE, SA, TL (113 hari, Rp 22.952.401); **≥ Rp 836.801** tambah satu BE + lembur (111 hari, Rp 24.626.003). Tambah BE saja tidak pernah terbaik. Urutan bertahan pada 96,5% ulangan bootstrap dan 6 dari 6 skenario asumsi |
| 11 | sedang | Biaya kegagalan melebihi biaya pencegahan | Rasio kesesuaian/ketidaksesuaian 0,69 |
| 12 | sedang | Mengabaikan korelasi menyembunyikan ketidakpastian | Simpangan baku durasi melebar 23,1% dengan ρ = 0,5 |
| 13 | sedang | Risiko yang berbagi sebab menebalkan ekor biaya | Rerata tetap Rp 19,18 jt; P95 biaya naik dari Rp 22,79 jt ke Rp 23,23 jt |
| 14 | sedang | Pemeriksaan bisa gagal berulang (GERT) | Tambahan harapan 1,52 hari kerja dan Rp 121.690; P(regresi butuh ≥ 2 putaran tambahan) = 9% |
| 15 | sedang | Realisasi selama UTS membantah asumsi kapasitas ujian 40% | Laju saat UTS 103,5% dari normal; kredibilitas empiris 98,1%, faktor ujian diperbarui menjadi 98,8% |
| 16 | baik | Jadwal levelling terbukti tidak bisa diperpendek dengan mengubah urutan | Jadwal terbaik 113 = batas bawah 113; aturan LST lama memberi 114 |
| 17 | baik | Pada jaringan CPM, lima hari percepatan pertama hampir dibayar sendiri | Premi lembur PP 35/2021 Rp 191.563, biaya bersih setelah sewa hanya **Rp 38.851** (sewa menutup 80%) |

Temuan tambahan dari halaman Piagam dan pencocokan kalender resmi: **tanggal selesai di Project Charter salah enam hari kerja.** Tujuh belas minggu kalender polos berakhir 13 Februari 2026; 85 hari kerja sesungguhnya berakhir **23 Februari 2026** setelah akhir pekan, empat libur nasional, dan dua cuti bersama dikeluarkan. Pencocokan dengan SKB 3 Menteri menemukan cuti bersama Imlek 16 Februari 2026 yang sebelumnya tidak ada di model.

Temuan yang tidak terlihat dari SPI: rasio durasi aktual terhadap rerata PERT pada 16 aktivitas yang sudah selesai adalah **0,983** — penyimpangan itu masih di dalam derau estimasi beta-PERT (sd 4,2%), sehingga kredibilitas empirisnya nol: tidak ada bukti tim lebih lambat dari sebaran estimasinya. SPI 0,915 lahir dari jadwal yang disusun memakai M (paling mungkin), bukan dari kinerja buruk.

---

## 3. Peta situs: 16 halaman × 2 bahasa

Setiap halaman tersedia dalam bahasa Indonesia (akar situs) dan bahasa Inggris (`/en/…`), dengan tautan `hreflang` yang saling menunjuk. Total 32 halaman.

**Setiap grafik punya panduan baca.** Di bawah setiap gambar ada dua kolom: *Cara membaca* (apa arti sumbu, warna, garis, dan penanda pada grafik itu) dan *Artinya* (kesimpulan satu-dua kalimat yang angkanya diambil dari analisis, bukan diketik — mis. "5 dari 7 metrik belum memenuhi target" atau "15 hari tambahan datang dari bentrokan kapasitas dan 13 hari dari periode ujian"). Grafik yang berpasangan (dua peta risiko, dua tangga realisme, diagram tulang ikan) berbagi satu panduan di bagiannya. Selain 25 grafik lama, sebelas **grafik penjelas** menggantikan atau mendampingi tabel yang sulit dibaca:

| Grafik | Halaman | Pertanyaan yang dijawab |
|---|---|---|
| Linimasa komitmen | Keputusan, Prakiraan | Seberapa jauh setiap janji tanggal dari tanggal data, dan mana yang masih berlaku? |
| Peta opsi | Keputusan | Opsi mana yang lebih cepat (kiri) dan lebih mahal (atas), dengan interval bootstrap 90%? |
| Pita nilai | Keputusan | Pada nilai satu hari lebih cepat berapa setiap opsi menjadi yang terbaik? |
| Strip skenario | Keputusan | Apakah rekomendasi bertahan bila asumsi diubah? |
| Jembatan anggaran | Keputusan | Dari pagu piagam ke permintaan anggaran: dari mana setiap rupiah tambahan? |
| Kurva lembur | Keputusan, Optimasi | Berapa harga setiap hari yang dibeli dengan lembur, dibanding crashing CPM? |
| Panel kredibilitas | Prakiraan | Apakah realisasi cukup kuat untuk menggeser rencana, atau masih derau? |
| Garis frontier JCL | Prakiraan | Anggaran minimum untuk peluang 70% pada setiap tanggal selesai. |
| Linimasa kapasitas | Optimasi | Kapan tim kehilangan kapasitas, berapa banyak, dan untuk peran apa? |
| Titik SPI/CPI | Biaya | Fase mana yang menarik indeks proyek ke bawah? |
| Batang berpasangan | Risiko | Seberapa besar mitigasi menurunkan EMV setiap kategori? |

Label ditulis langsung di dalam gambar (bukan legenda terpisah) oleh penempat label yang mencoba posisi makin jauh dari titik setinggi labelnya, menghindari garis panah dan interval, lalu memilih tumpang-tindih terkecil bila kanvas penuh.

Bagian yang dulu hanya tabel panjang juga mendapat **grafik rincian** — sembilan bentuk generik yang dipakai di empat belas tempat, masing-masing dengan panduan baca:

| Grafik | Halaman | Pertanyaan yang dijawab |
|---|---|---|
| Panel sapuan rho dan lambda | Simulasi Terpadu | Besaran mana yang peka terhadap asumsi korelasi dan penggerak bersama? Skala tegak minimum 8% dari nilainya, jadi perubahan kecil tetap tampak datar. |
| Sebaran putaran GERT | Simulasi Terpadu | Berapa peluang pemeriksaan harus diulang 0, 1, 2, … kali, dan apakah rerata analitik cocok dengan simulasi? |
| Biaya sewa per lapisan | Simulasi Terpadu | Berapa biaya sewa yang lahir hanya karena proyek lebih lama dari rencana BAC? |
| Register dibanding simulasi | Simulasi Terpadu | Apakah kopula menjaga peluang setiap risiko? |
| Risiko sebelum dan sesudah berjalan | Prakiraan | Risiko mana yang sudah ditutup dan tidak lagi disampel? |
| Rentang tiga titik | PERT | Seberapa lebar dan condong estimasi setiap aktivitas? |
| Float bebas dan bersama | Jadwal | Aktivitas mana yang punya ruang gerak, dan apakah memakainya mengganggu penerus? |
| Pecahan pergeseran | Optimasi | Dari mana setiap hari keterlambatan levelling datang: terbawa, menunggu orang, atau memanjang? |
| Scatter fast-tracking | Optimasi | Kandidat mana yang menghemat hari tanpa rework mahal, dan mana yang mustahil karena orangnya sama? |
| Varians biaya per aktivitas | Biaya | Aktivitas mana yang membuat CPI proyek turun? |
| Anggaran per fase dan paket | Piagam | Ke mana anggaran aktivitas mengalir? |
| Linimasa Coretax berskala waktu | Coretax | Berapa lama membangun dibanding secepat apa dampaknya datang? |
| Biaya membangun vs kerugian sebulan | Coretax | Seberapa besar paparan dibanding biaya proyek, pada skala yang sama? |

### 3.1 Ruang Kendali — `/`

- **Delapan KPI** dengan warna status: SPI, CPI, EAC, peluang tepat waktu, cakupan cadangan risiko, durasi yang bisa dijalankan (dengan status terbukti optimal), JCL, dan P80 prakiraan berjalan (dengan SV(t) dan IEAC(t)).
- **Kurva-S Earned Value**: PV, EV, AC, proyeksi biaya sampai akhir (mengikuti bentuk sisa kurva PV, bukan garis lurus), garis BAC, dan garis tanggal data.
- **Enam belas temuan** diturunkan dari ambang, masing-masing dengan metrik pemicu, rekomendasi, dan tautan ke halaman perhitungannya.
- Ringkasan jadwal (termasuk selisih akibat hari libur) dan ringkasan anggaran berlapis.

### 3.2 Piagam & Lingkup — `/piagam/`

- Project Charter terstruktur: informasi umum, tujuan, lingkup masuk dan keluar, deliverable, asumsi, batasan.
- **Catatan pemeriksaan** yang membuktikan tanggal selesai piagam tidak konsisten dengan kalender kerja, lengkap dengan tabel libur nasional dan cuti bersama beserta dasar hukumnya (SKB 3 Menteri 2025 dan 2026).
- Kriteria keberhasilan dengan status menurut data (anggaran dan jadwal terancam; sisanya belum terukur sebelum go-live).
- **Work Breakdown Structure** 5 fase → 15 paket kerja → 35 aktivitas + 5 milestone, dengan estimasi tiga titik, anggaran, float, dan pendahulu per aktivitas.
- **Pemeriksaan aturan 100%** secara aritmetis (jumlah anggaran aktivitas = BAC).
- **Rekonsiliasi anggaran**: grafik air terjun BAC → cadangan kontinjensi → cadangan manajemen → pagu Rp 16 juta, plus tabel tarif harian per peran.

### 3.3 Jadwal & Jalur Kritis — `/jadwal/`

- **Gantt chart** dengan batang jalur kritis, batang float (bayangan), batang realisasi (hijau/jingga bila melampaui rencana), penanda milestone, panah ketergantungan jalur kritis, dan garis tanggal data.
- **Diagram jaringan Activity-on-Node** dengan tata letak berlapis menurut early start; setiap simpul memuat ES, durasi, EF, LS, total float, LF.
- **Tabel CPM lengkap** 40 simpul: d, ES, EF, LS, LF, TF, FF, tanggal mulai/selesai, pendahulu.
- Rantai jalur kritis tersambung (29 simpul) dan tabel aktivitas ber-float terbesar beserta maknanya.

### 3.4 Levelling & Kompresi — `/optimasi/`

- **Levelling sumber daya yang terbukti optimal**: CPM 85 → kapasitas nyata 100 → UTS dan UAS 113 hari kerja, selesai 10 April 2026.
- **Bukti optimalitas**: enam aturan prioritas (LST, LFT, MSLK, GRPW, MTS, SPT), 300 daftar acak berbias, dan justifikasi maju-mundur mencari batas atas; batas bawah tiga lapis (CPM 85, solo 108, energetik 113) plus pembuktian destruktif. Batas atas = batas bawah = 113, jadi tidak ada urutan kerja yang bisa selesai lebih cepat.
- **Levelling eksak di dalam simulasi**: **10.000 dari 10.000** iterasi lapisan kapasitas terbukti optimal — 8.077 langsung oleh SGS cepat (sama dengan batas bawahnya), 1.923 setelah `level.Search`, yang rata-rata menghemat 0,38 hari per iterasi dan paling banyak 10 hari. Hal yang sama berlaku untuk seluruh iterasi prakiraan berjalan.
- **Kalender ketersediaan**: keempat periode ujian dari lampiran kalender akademik resmi Esa Unggul (SK Rektor No. 039/SK-R/UEU/III/2025) — UTS ganjil 3–15 Nov 2025, UAS ganjil 19–31 Jan 2026, UTS genap 18–30 Mei 2026, UAS genap 20 Jul–1 Agu 2026; DevOps paruh waktu (dari piagam).
- **Gantt pembanding** CPM vs levelling, batang diwarnai menurut penyebab, jendela ujian diarsir.
- **Histogram pembebanan setelah levelling** dengan garis kapasitas bertangga per hari — nol over-alokasi.
- Hari menunggu per peran dan **peran kritis** (Backend Developer).
- **Premi lembur dari PP 35/2021**, bukan asumsi: pekerjaan hari yang dipotong dibagi rata sebagai lembur ke hari tersisa; jam pertama 1,5×, jam berikutnya 2× upah sejam (1/173 upah sebulan), paling lama 4 jam sehari dan 18 jam seminggu. Premi per aktivitas 75%–87,5%; **lima aktivitas dua hari (A27, A29, A31, A34, A36) tidak boleh dipotong** karena butuh 8 jam lembur dalam sehari.
- **Biaya crash per hari, bukan rata-rata**: hari kedua yang dipotong dari aktivitas yang sama lebih mahal karena jam lembur di atas jam pertama dibayar 2×. Contoh A17 (M = 6): hari pertama Rp 37.813 (1,6 jam/hari), hari kedua Rp 58.438 (naik ke 4 jam/hari). Potongan juga berhenti di hari terakhir yang masih sah, tidak langsung ditolak seluruhnya.
- **Crashing serakah vs eksak**: kurva 85 → 68 hari. Dengan biaya lembur per hari, serakah ternyata **sama dengan LP di setiap durasi** — kelebihan Rp 4.063 versi sebelumnya lahir dari slope rata-rata yang membuat hari kedua A17 tampak murah. Hanya LP yang membuktikannya; kelemahan serakah (satu potongan bersama yang mahal dipilih walau pasangan cabang paralel lebih murah) tetap dibuktikan pada jaringan uji.
- **Time-cost trade-off**: LP biaya total (premi lembur + sewa server & langganan Rp 44.740/hari). Lima hari pertama berpremi Rp 191.563 tetapi biaya bersihnya hanya Rp 38.851; dengan premi sesuai aturan, tidak ada durasi yang lebih murah dari 85 hari. Grafik tiga kurva (crash eksak, sewa, total) dengan titik biaya terendah.
- **Lembur pada jadwal yang bisa dijalankan**: kurva crashing memotong jaringan CPM yang tidak menghormati kapasitas orang. Pada jadwal levelling, lembur menambah jam kerja — untuk pekerjaan lain yang menunggu, dan untuk mempercepat pekerjaan orang itu sendiri sampai 1,45 hari kerja per hari — paling banyak 3,6 jam/hari (agar ≤ 18 jam/minggu), hanya peran penuh waktu, tidak saat ujian. Jam di atas alokasi rencana dibayar sebagai lembur walau pemakaian peran masih di bawah kapasitas normal.

  | Durasi | Selesai | Lembur | Upah lembur | Bersih |
  | --- | --- | --- | --- | --- |
  | 113 | 10 Apr 2026 | — | Rp 0 | Rp 0 |
  | 111 | 8 Apr 2026 | BA 8 jam | Rp 91.250 | Rp 91.250 |
  | 108 | 2 Apr 2026 | BA 16, PM 2,4, UX 16 jam | Rp 374.350 | Rp 365.779 |
  | 104 | 27 Mar 2026 | + BE 32 jam | Rp 783.412 | Rp 655.880 |
  | **101** | 17 Mar 2026 | BA, BE, FE, PM, SA, TL, UX | **Rp 1.380.475** | Rp 1.189.176 |

  **101 hari terbukti minimum** (batas bawah 101). Durasi yang terlewati (112, 109, …) berarti rencana termurahnya selesai sehari lebih awal. **Rencana ini dihitung dari hari pertama proyek dan tidak bisa dibeli lagi** — 23 hari-peran lemburnya jatuh sebelum tanggal data; opsi yang masih tersedia ada di Keputusan Sponsor.
- **Fast-tracking**: 23 kandidat diuji dengan tumpang tindih 50%; kandidat orang-sama **ditolak**; 11 kandidat layak; penerapan serentak memberi 66 hari.

### 3.5 PERT & Monte Carlo — `/pert/`

- Perbandingan PERT (Z-score jalur kritis) dan Monte Carlo 10.000 iterasi berdampingan.
- Penjelasan **merge bias**: mengapa rerata simulasi 94,3 hari, bukan 85.
- Histogram durasi dengan kurva kumulatif dan penanda rencana, P50, P80, P90.
- Tabel tingkat keyakinan P50–P95 beserta tanggal selesai dan kelayakan untuk dijanjikan.
- **Diagram tornado** sensitivitas (korelasi peringkat Spearman) dan porsi iterasi kritis per aktivitas.
- Tabel estimasi tiga titik: O, M, P, te, σ, varians, dan kecondongan.
- **Panel WebAssembly**: jalankan ulang Monte Carlo dengan iterasi, benih, dan sebaran (beta-PERT/segitiga) pilihan.

### 3.6 Simulasi Terpadu & JCL — `/simulasi-terpadu/`

- **Lima lapisan realisme** dengan benih yang sama sehingga selisihnya murni efek yang ditambahkan:

  | Lapisan | Efek | P80 durasi | P80 biaya | JCL |
  | --- | --- | --- | --- | --- |
  | L0 | Independen (identik dengan halaman PERT) | 98 | Rp 16,23 jt | 1,17% |
  | L1 | + Korelasi peran (kopula Gauss, ρ 0,5) | 99 | Rp 16,34 jt | 3,45% |
  | L2 | + Risiko bergerombol (kopula faktor, λ 0,6) | 117 | Rp 20,94 jt | 0,45% |
  | L3 | + Putaran rework GERT | 119 | Rp 21,18 jt | 0,31% |
  | L4 | + Kapasitas, UTS & UAS (levelling eksak per iterasi) | 151 | Rp 22,02 jt | 0,00% |

  Biaya di setiap lapisan sudah memuat sewa server dan langganan yang ikut memanjang bersama jadwal.
- **Tangga realisme** durasi dan biaya (P50–P90 dengan penanda P80 dan garis target piagam).
- **Peta kepadatan JCL** dengan **frontier JCL 70%**, silang target piagam, titik P80 × P80; tabel frontier dan histogram biaya akhir.
- **Uji kepekaan ρ** (0; 0,25; 0,5; 0,75) dengan korelasi terealisasi.
- **Risiko bergerombol**: lima penggerak bersama yang dibaca dari kolom penyebab register; penggerak kinerja pengembang memakai faktor laten Backend Developer yang sama dengan durasi aktivitasnya. Uji kepekaan λ (0; 0,3; 0,6; 0,9): rerata biaya tetap, phi terealisasi naik, P95 biaya menebal; grafik sebaran jumlah risiko per proyek.
- **Putaran rework GERT**: dua pemeriksaan yang bisa gagal berulang (regresi pasca perbaikan bug p = 30%, uji penetrasi p = 25%) direduksi dengan aturan Mason; rerata putaran analitik 0,429 dan 0,333 cocok dengan Monte Carlo 0,445 dan 0,330.
- **Biaya yang bergantung waktu**: empat pos sewa dan langganan dengan tarif harian dari rentang rencana.
- Frekuensi kejadian setiap risiko beserta penggeraknya.
- **Panel WebAssembly**: geser ρ, pilih lapisan L0–L4, jalankan simulasi terpadu di peramban.

### 3.7 Biaya & Earned Value — `/biaya/`

- KPI PV, EV, AC, BAC.
- **Tabel seluruh metrik turunan** dengan rumus dan kalimat cara membaca: SV, CV, SPI, CPI, ES, SV(t), SPI(t), tiga varian EAC, ETC, VAC, TCPI.
- **Panel WebAssembly** untuk menggeser tanggal data (dibatasi pada hari terakhir yang punya realisasi, agar SPI yang anjlok karena kehabisan data tidak terbaca sebagai temuan).
- Kurva-S, grafik anggaran berlapis, dan catatan bahwa proyeksi menembus seluruh cadangan.
- Earned Value per fase dan per aktivitas (% rencana, % aktual, EV, AC, CV, status).

### 3.8 Prakiraan Berjalan — `/prakiraan/` *(baru)*

- **Simulasi terpadu dari tanggal data** (19 Des 2025): 18 simpul selesai dikunci pada realisasinya, 3 aktivitas yang sedang berjalan (A17, A19, A21) memakai durasi bersyarat F(x | x > e), 19 sisanya dirilis pada tanggal data.
- **Lima prakiraan berdampingan**: rencana CPM 85; Earned Value IEAC(t) 91,0 & EAC Rp 16,16 jt; simulasi perencanaan P80 151; prakiraan berjalan tanpa belajar P80 129; **prakiraan berjalan terkalibrasi P80 125 & Rp 20,83 jt** — dengan kolom "buta terhadap" untuk tiap metode.
- **Kredibilitas Bühlmann empiris** — bobot tidak dipilih, tetapi diestimasi dengan metode momen: Var = derau estimasi, tau² = max(0, selisih² − Var), Z = tau² / (tau² + Var).
  - Durasi: rasio aktual/rerata PERT 0,983, derau sd 4,2% → tau² = 0, **Z = 0**, faktor 1.
  - Biaya tenaga kerja harian: rasio 1,057, derau sd 2,2% (estimator sandwich) → **Z = 85,1%**, faktor 1,049.
  - Kapasitas ujian: teramati 103,5% (dibatasi 100%), derau dari metode delta → **Z = 98,1%**.
- **Kalibrasi kapasitas ujian** dari realisasi selama UTS resmi: laju 103,5% dari normal, faktor untuk UAS diperbarui 40% → 98,8%.
- Tabel bukti per aktivitas selesai, durasi bersyarat aktivitas yang sedang berjalan, status risiko pada tanggal data (risiko berstatus "terjadi" ditutup; risiko terbuka dipindah ke pekerjaan yang belum selesai), dan frontier JCL 70% dari tanggal data (121 hari & Rp 21.819.542).
- **Panel WebAssembly**: pilih tanggal data lain dan prakirakan ulang — kalibrasi dihitung dari bukti yang tersedia saat itu.

### 3.9 Keputusan Sponsor — `/keputusan/` *(baru)*

- **Satu komitmen, bukan enam**: CPM 85, P80 PERT 98, levelling 113, JCL 70% perencanaan 147, IEAC(t) 91 — masing-masing dengan alasan tidak berlaku — dan **JCL 70% berjalan 121 hari kerja (22 April 2026) & Rp 21.819.542** sebagai satu-satunya yang berlaku.
- **Opsi percepatan dari tanggal data** (10.000 iterasi per opsi, levelling dibuktikan per iterasi; satu iterasi opsi masa adaptasi belum terbukti dan dilaporkan halaman):

  | Opsi | Lantai (terbukti) | P80 | JCL 70% | Anggaran | Lebih cepat | Harga/hari |
  | --- | --- | --- | --- | --- | --- | --- |
  | Tanpa percepatan | 101 | 125 | 121 (22 Apr) | Rp 21.819.542 | — | — |
  | **Lembur sah BE, DBA, FE, SA, TL** | 95 | 116 | 113 (10 Apr) | Rp 22.952.401 | 8 | **Rp 141.607** |
  | Tambah satu BE | 97 | 120 | 117 (16 Apr) | Rp 24.055.382 | 4 | Rp 558.960 |
  | Tambah satu BE, masa adaptasi 10 hari (asumsi) | 97 | 120 | 117 | Rp 24.055.382 | 4 | Rp 558.960 |
  | Tambah satu BE + lembur sah | 93 | 114 | **111** (8 Apr) | Rp 24.626.003 | 10 | Rp 280.646 |

  Peran lembur diturunkan dari rencana termurah durasi minimum (lantai 95 terbukti, upah Rp 773.813); orang baru dibayar dari tanggal data sampai pekerjaan terakhir perannya selesai; masa adaptasi tidak mengubah hasil karena pekerjaan BE yang menunggu baru menumpuk setelahnya.
- **Aturan keputusan menurut nilai satu hari lebih cepat** (selubung atas garis manfaat bersih): di bawah Rp 141.607 tanpa percepatan; Rp 141.607–836.801 lembur sah; Rp 836.801 ke atas tambah satu BE + lembur sah. Tambah satu BE saja tidak pernah terbaik pada nilai berapa pun.
- **Ketahanan terhadap derau**: bootstrap berpasangan 200 ulangan (iterasi yang sama ditarik ulang untuk semua opsi). Opsi termurah tetap sama pada 98,5% ulangan, urutan pita pada 96,5%; interval 90% batas pertama Rp 80.136–260.542, batas kedua Rp 615.852–1.530.740; JCL 70% setiap opsi bergeser paling banyak satu hari.
- **Kepekaan terhadap asumsi tanpa data**: ρ 0,25/0,75, λ risiko 0,3/0,9, peluang gagal GERT ×0,5/×1,5 — masing-masing 3.000 iterasi dengan levelling cepat, dibandingkan dengan asumsi dasar yang dihitung dengan cara yang sama. Rekomendasi sama pada **6 dari 6** skenario; batas pertama bergeser antara Rp 106.433 dan Rp 174.769.
- **Satu permintaan anggaran**: Rp 5.819.542 di atas pagu, dari JCL 70% berjalan — bukan dari EAC (Rp 157.314, buta risiko) dan tidak ditambah kenaikan cadangan (risiko yang sama sudah di dalamnya).
- **Tindakan lain dari temuan** dan **yang tidak bisa diputuskan dengan kode**: nilai satu hari lebih cepat bagi sponsor, tarif dan ketersediaan BE nyata, realisasi baru lewat `realisasi.csv`.

### 3.10 Manajemen Risiko — `/risiko/`

- KPI EMV inheren, EMV residual, cadangan tersedia, kekurangan cadangan dan paparan jadwal.
- **Dua peta panas 5×5** — sebelum dan sesudah mitigasi.
- **Risk register 12 entri**, masing-masing dengan kategori, pemilik, WBS terpapar, sebab, akibat, peluang/dampak/EMV/skor inheren dan residual, persentase penurunan EMV, strategi respons, mitigasi, dan pemicu.
- Paparan per kategori dan rekomendasi soal kecukupan cadangan.

### 3.11 Organisasi & Sumber Daya — `/organisasi/`

- **Bagan organisasi** lima tingkat (sponsor → PM → core lead → tim pelaksana) dan kartu tanggung jawab tiap peran.
- **Matriks RACI** per fase WBS yang divalidasi uji (tepat satu A per baris).
- **Histogram pembebanan** 10 peran sepanjang 85 hari kerja dengan batang over-alokasi merah, tabel utilisasi, dan daftar bentrokan terberat.
- **Grid kuasa-kepentingan** 10 pemangku kepentingan dan **rencana komunikasi** enam jalur.

### 3.12 Manajemen Mutu — `/kualitas/`

- **Tujuh metrik mutu** terukur terhadap target standar Project Charter.
- **Peta kendali X-bar** waktu respons: CL, UCL, LCL (metode A2·R̄), batas spesifikasi, σ proses, Cpk, dan penanda pelanggaran **empat aturan Nelson**.
- **Diagram Pareto** cacat berbobot keparahan (kritis 5, mayor 3, minor 1) per modul.
- **Dua diagram fishbone** (6M) dengan akar penyebab.
- **Biaya kualitas** empat kategori, rincian pos (terjadi vs proyeksi), dan rasio kesesuaian/ketidaksesuaian.

### 3.13 Bedah Kasus: Coretax — `/coretax/`

Lihat [bagian 6](#6-bedah-kasus-coretax).

### 3.14 Referensi Rumus — `/rumus/`

38 rumus dalam sembilan kelompok. Setiap rumus memuat notasi (dwibahasa), arti tiap simbol, makna, cara membaca, **jebakan umum**, rujukan materi, dan **contoh hitung yang disuntik dari angka hidup** — jadi halaman rumus tidak pernah bisa berbeda dari halaman analisisnya. Lihat [bagian 5](#5-referensi-38-rumus).

### 3.15 Materi & Area Pengetahuan — `/materi/`

- **Sembilan area pengetahuan** (Modul 2) dengan tautan ke artefak yang membuktikan area itu benar-benar dikerjakan, plus catatan area kesepuluh PMBOK 5.
- **Empat tahap siklus hidup** (Tugas 3) dipetakan ke fase WBS.
- **Peta materi kuliah → paket kode** yang mengimplementasikannya, termasuk GERT ("loop tes yang harus diulang") dan pengawasan jadwal dari Modul 4.
- Daftar dokumen sumber di folder mata kuliah, termasuk satu berkas yang tidak terkait (dokumen Oracle OLVM).

### 3.16 Metode & Sumber — `/metode/`

Arsitektur paket, keputusan teknis, **tabel asumsi** (alasan dan akibatnya bila keliru), **celah yang sudah ditutup** beserta buktinya, **batas yang tersisa**, dan tautan data terbuka.

---

## 4. Mesin hitung

Seluruh perhitungan berada di `internal/` sebagai paket Go murni **tanpa satu pun dependensi pihak ketiga**.

### `workcal` — kalender kerja
Pemetaan dua arah indeks hari kerja ↔ tanggal, seluruh libur nasional dan cuti bersama sampai akhir 2026 dengan dasar hukum SKB 3 Menteri, indeks pecahan untuk tanggal data yang jatuh di akhir pekan, dan daftar libur di dalam rentang.

### `schedule` — CPM dan PERT
- Urutan topologis dengan deteksi siklus, predecessor tak dikenal, dan kode ganda.
- Forward pass dan backward pass untuk **FS, SS, FF, SF** dengan lag dan lead, serta tanggal rilis (*start no earlier than*) untuk prakiraan berjalan.
- Total float, free float, dan **rantai kritis tersambung** (bukan sekadar filter float nol).
- PERT: te, σ, varians, varians jalur kritis, Z-score, dan peluang selesai.
- Konvensi batas inklusif untuk tampilan dan eksklusif untuk aritmetika, sehingga milestone berdurasi nol tidak butuh kasus khusus.

### `level` — penjadwalan berbatas sumber daya
- **Serial Schedule Generation Scheme** dengan enam aturan prioritas atau daftar aktivitas, tanggal rilis, dan laju mulai minimum 20% (satu hari kerja per minggu).
- **Model isi pekerjaan**: laju harian = min(batas laju, sisa kapasitas / alokasi) atas semua peran; hari-orang dilestarikan, kalender yang memanjang. Batas laju 1, kecuali `RateCap` (lembur: 1 + h/8); jam di atas alokasi tercatat di `Result.Excess`.
- **`Optimize`**: aturan prioritas + sampel acak berbias (*regret-based biased random sampling*) + **justifikasi maju-mundur** di atas jaringan dan kalender terbalik.
- **`LowerBound`**: batas solo (kapasitas nyata tanpa berbagi; penalaran tak bertanggal memakai kapasitas dan laju harian **tertinggi**, sehingga tetap sah pada grid lembur atau orang baru; CPM tidak dipakai bila laju bisa melebihi 1), penalaran **energetik** leluhur/keturunan/global yang dirambatkan sampai titik tetap, lalu **pembuktian destruktif** yang menguji tenggat bertanggal. Celah optimalitas = jadwal terbaik − batas bawah.
- Kapasitas per peran per hari dari `model.AvailabilityWindows`, dengan faktor ujian yang bisa diganti hasil kalibrasi.
- Pemecahan keterlambatan per aktivitas: **terbawa**, **menunggu**, **memanjang**, beserta penyebab.
- `Explain`: dekomposisi bertahap CPM → kapasitas → jendela, setiap tahap memakai `Optimize`.
- **`Search`**: pencarian berbenih menuju batas bawah untuk levelling per iterasi — langkah lokal memindah satu aktivitas di daftar (CPM dihitung sekali), enam aturan dengan justifikasi, lalu sampel berbias; berhenti begitu batas bawah tercapai.

### `compress` — kompresi jadwal
- **Crashing serakah** per hari dengan pencarian pasangan dan tiga aktivitas untuk jalur kritis paralel.
- **Crashing eksak** (`Exact`): LP per tenggat dengan satu kolom per hari yang boleh dipotong (biaya marjinal cembung terwakili persis); kolom duplikat mempertahankan unimodularitas total sehingga solusi simpleks berupa hari bulat, diverifikasi ulang dengan CPM; perbandingan titik demi titik dengan serakah.
- **`LevelledOvertime`**: grid kapasitas + `RateCap` lembur sah pada hari non-ujian untuk peran penuh waktu mulai hari `from` (0 untuk perencanaan, tanggal data untuk keputusan), `level.Optimize` + batas bawah pada grid maksimum untuk durasi minimum terbukti, lalu pemangkasan per peran, per hari dua arah, dan pencarian lokal tukar hari; upah = max(jam di atas kapasitas normal, jam di atas alokasi).
- **Time-cost trade-off**: LP biaya total dengan sewa `cost.Rental`, tanggal mulai proyek dikunci, titik biaya terendah, dan nilai impas per hari.
- **Fast-tracking** dengan rework harapan dan penolakan kandidat orang-sama.
- Biaya marjinal setiap hari yang dipotong dari `model.OvertimePremium` (PP 35/2021 Pasal 26, 31, 32), lewat `CrashPlan.Marginal` dan `CostToCut`.

### `lp` — pemrograman linear
Simpleks dua fase dengan tableau padat dan **aturan Bland** (tidak pernah berputar pada masalah degeneratif — diuji dengan contoh klasik Beale).

### `gert` — reduksi jaringan GERT
Transmitansi dibawa sebagai koefisien Taylor orde dua sehingga aljabarnya eksak: seri, paralel, putaran **aturan Mason**; rerata dan varians waktu; jumlah putaran geometrik, kuantil, dan sampel transformasi invers.

### `cost` — biaya yang bergantung waktu
Memisahkan sewa dan langganan dari biaya sekali beli; tarif harian dari rentang rencana sehingga biaya pada jadwal rencana persis sama dengan BAC.

### `evm` — Earned Value
PV/EV/AC dengan kemajuan linear dalam aktivitas; SV, CV, SPI, CPI; **Earned Schedule** (pencarian biner pada kurva PV) dengan SV(t) dan SPI(t); tiga varian EAC; ETC; VAC; TCPI; kurva-S dan proyeksi biaya; rincian per fase dan per aktivitas.

### `simulate` — Monte Carlo
- PRNG **mulberry32 berbenih** — hasil identik di server dan WebAssembly.
- **Sampler bersama** lewat transformasi invers: beta-PERT baku (α = 1 + 4(M−O)/(P−O)), CDF dari **fungsi beta tak lengkap teregularisasi** (pecahan berlanjut Lentz) ditabulasi pada 2.049 titik; sebaran segitiga dengan invers tertutup.
- **Kopula Gauss** per peran dominan: u = Φ(ρ·z_peran + √(1−ρ²)·ε); faktor laten peran bisa dibaca untuk kopula risiko.
- Simulasi PERT: histogram, kuantil, peluang, sensitivitas Spearman, porsi kritis.
- **Simulasi terpadu** lima lapisan: biaya per iterasi termasuk sewa bergantung waktu, **kopula faktor risiko** dengan phi terealisasi, **putaran rework GERT**, levelling per iterasi yang **dibuktikan optimal** (SGS cepat → batas bawah → `level.Search`, anggaran dinaikkan 10× bila perlu) dan dijalankan paralel tanpa mengubah hasil, Joint Confidence Level, frontier iso-JCL, histogram 2D.
- **`FirstFeasible`** (titik JCL layak pertama langsung dari pasangan iterasi) dan **`PairedBootstrap`** (menarik ulang indeks iterasi yang sama untuk semua opsi); `ReworkScale` untuk skenario peluang gagal GERT.
- **Opsi percepatan** (`Acceleration`): lembur sah (kapasitas + `RateCap`), orang baru (kapasitas, masa adaptasi), grid per opsi, dan upah per iterasi (`Pay`) yang masuk ke biaya dan JCL.
- **Prakiraan berjalan** (`PrepareInFlight`, `Deterministic` untuk jaringan sisa pada durasi paling mungkin): status per aktivitas pada tanggal data, jaringan sisa dengan tanggal rilis, durasi bersyarat, **kredibilitas Bühlmann empiris** (estimator momen; derau dari varians beta-PERT, estimator sandwich, dan metode delta) untuk durasi, biaya, dan kapasitas ujian.

### `risk` — risiko kuantitatif
Tingkat peluang dan dampak (relatif terhadap BAC), skor dan keparahan, matriks inheren dan residual, EMV, penurunan EMV per risiko, agregasi kategori, cakupan dan kekurangan cadangan, paparan jadwal harapan.

### `resource` — pembebanan
Beban harian per peran, deteksi over-alokasi dengan daftar aktivitas penyebab, utilisasi, jumlah peran aktif per hari, dan kehalusan kurva tim.

### `quality` — pengendalian mutu
Pareto berbobot dengan kelompok *vital few*; peta kendali X-bar dengan konstanta A2/d2; empat aturan Nelson yang **diuji terhadap batas yang sama dengan yang digambar**; Cpk satu sisi; biaya kualitas; evaluasi metrik dua arah.

### `coretax` — studi kasus
Metrik turunan dari fakta bersumber, empat skenario transisi dengan EMV, titik impas peluang kegagalan, dan tabel cermin Coretax ↔ proyek kuliah.

### `render` — grafik SVG
Pembangun kanvas SVG dan 45 jenis grafik (termasuk batas bawah levelling, kurva time-cost trade-off, sebaran jumlah risiko, sebelas grafik penjelas di `explain.go`, dan sembilan grafik rincian generik di `detail.go`), semuanya dengan `<title>` dan `<desc>` untuk pembaca layar, warna lewat kelas CSS (tema gelap tanpa gambar ulang), dan escape teks. `labelPlacer` menempatkan label tanpa saling menimpa; `spreadY` menjaga jarak label di ujung garis tanpa keluar dari bidang gambar.

### `site`, `model`, `i18n`
Perakit analisis dan penurun temuan; **paket keputusan** (`Decision`: opsi dari tanggal data, lantai terbukti, harga per hari, satu komitmen, satu permintaan anggaran, pita nilai waktu, bootstrap berpasangan, dan skenario asumsi); sumber tunggal kebenaran seluruh data proyek; kamus antarmuka dwibahasa. `cmd/site` juga mengekspor dan mengimpor `realisasi.csv`.

---

## 5. Referensi 41 rumus

| Kelompok | Rumus |
| --- | --- |
| **Penjadwalan & Jalur Kritis** | Early Finish (forward pass) · Late Start (backward pass) · Total & free float |
| **Estimasi Tiga Titik & PERT** | Durasi harapan te · Simpangan baku & varians · Peluang Z · Simulasi Monte Carlo · Sensitivitas Spearman · Sebaran beta-PERT & transformasi invers |
| **Earned Value Management** | PV/EV/AC · SV & CV · SPI & CPI · Tiga varian EAC · TCPI · Earned Schedule |
| **Risiko Kuantitatif** | EMV · Skor & matriks probabilitas-dampak · Struktur anggaran berlapis |
| **Pengendalian Mutu** | Batas kendali X-bar · Aturan Nelson · Cpk · Biaya kualitas · Analisis Pareto |
| **Sumber Daya** | Pembebanan & utilisasi · Kehalusan kurva tim |
| **Levelling & Kompresi Jadwal** | Serial Schedule Generation Scheme · Laju kerja berbatas kapasitas · Crashing & slope biaya · Fast-tracking & rework harapan · **Batas bawah energetik & celah optimalitas · Crashing eksak & trade-off biaya total (LP) · Lembur sah pada jadwal berbatas sumber daya** |
| **Simulasi Terpadu & JCL** | Korelasi lewat kopula Gauss · Kejadian risiko dalam simulasi · Joint Confidence Level · **Biaya sewa yang bergantung waktu · Risiko bergerombol lewat kopula faktor · GERT: putaran rework dengan aturan Mason** |
| **Prakiraan Berjalan** | **Kredibilitas Bühlmann & durasi bersyarat · Harga per hari percepatan dari tanggal data · Pita nilai waktu dan ketahanan keputusan** |

Rumus bercetak tebal ditambahkan pada upgrade terakhir. Setiap rumus punya contoh hitung dari data hidup — uji `TestEveryFormulaHasAWorkedExample` gagal bila ada yang tidak. Rumus lama yang maknanya bergeser (SGS, crashing, kejadian risiko) ikut diperbarui notasi dan jebakannya.

---

## 6. Bedah kasus Coretax

Halaman ini memakai **tiga lapis kejujuran** yang dijaga uji:

| Lapis | Aturan | Contoh |
| --- | --- | --- |
| **Fakta** | Setiap angka punya URL sumber publik | Kontrak LG CNS–Qualysoft Rp 1,228 T; Owner's Agent Rp 110,3 M; go-live 1 Jan 2025; penerimaan Januari turun 34,5%; potensi hilang Rp 64 T |
| **Turunan** | Aritmetika murni atas fakta, rumus ditampilkan | Rasio paparan **47,8×** biaya proyek; seluruh biaya proyek setara 0,65 hari kerugian penerimaan |
| **Skenario** | Andaian, diberi label bukan fakta | Empat strategi transisi dibandingkan dengan EMV; jalan paralel impas pada peluang gagal cutover 0,279% |

Isi halaman: 14 kartu fakta bersumber, linimasa 2018–2026, tabel metrik turunan, perbandingan empat strategi transisi (serentak, paralel, bertahap, pilot) dengan grafik biaya harapan, titik impas, **enam pelajaran yang dipetakan ke praktik MPPL dan ke gejala yang sama pada proyek kuliah**, dan daftar 11 sumber (Kompas, Tempo, Hukumonline, DDTC News, Beritasatu, Direktorat Jenderal Pajak).

---

## 7. Interaktivitas lewat WebAssembly

`cmd/wasm` mengompilasi paket `internal/` yang sama ke WebAssembly. Tidak ada rumus yang ditulis dua kali, jadi tidak mungkin ada versi JavaScript yang diam-diam berbeda dari versi Go. Terverifikasi di peramban: simulasi terpadu L4 10.000 iterasi dengan levelling eksak memberi P80 151 dan biaya P80 Rp 22.024.480,52, dan prakiraan berjalan 10.000 iterasi memberi P80 125 dan biaya P80 Rp 20.830.528,00 — keduanya identik sampai digit terakhir dengan hasil server.

| Halaman | Fungsi Go | Kendali |
| --- | --- | --- |
| Biaya | `mpplRecompute(tanggal)` | Geser tanggal data → SPI, CPI, SV(t), EV, AC, EAC, VAC, TCPI, % selesai |
| PERT | `mpplSimulate(iterasi, benih, sebaran)` | Monte Carlo ulang + histogram |
| Simulasi Terpadu | `mpplIntegrated(iterasi, benih, ρ, lapisan, eksak)` | Slider ρ, pilihan lapisan L0–L4, levelling eksak → JCL, peluang, P80, korelasi terealisasi, phi risiko, hari rework, biaya sewa, porsi iterasi terbukti optimal + histogram |
| Prakiraan Berjalan | `mpplForecast(tanggal, iterasi, eksak)` | Pilih tanggal data → status aktivitas, Z durasi / biaya / ujian, faktor durasi, P50, P80, biaya P80 + histogram |

Bila WebAssembly gagal dimuat, panel tetap tersembunyi dan halaman menampilkan angka yang benar untuk tanggal data bawaan — seluruh isi dan grafik sudah dirender server.

---

## 8. Data terbuka

| Berkas | Isi |
| --- | --- |
| [`data/metrik.json`](https://xyb3rpunq.github.io/mppl-control-tower/data/metrik.json) | Earned Value (termasuk IEAC(t)), anggaran berlapis, simulasi PERT, risiko, mutu, **levelling beserta batas bawah dan audit, kurva crashing eksak & biaya total, simulasi terpadu lima lapisan, putaran GERT, prakiraan berjalan, lembur jadwal nyata, dan keputusan sponsor** |
| [`data/aktivitas.csv`](https://xyb3rpunq.github.io/mppl-control-tower/data/aktivitas.csv) | 40 simpul: WBS, durasi, ES, EF, LS, LF, float, kritis, **mulai/selesai/geser levelling**, anggaran, PV, EV, AC |
| [`data/risiko.csv`](https://xyb3rpunq.github.io/mppl-control-tower/data/risiko.csv) | 12 risiko: peluang, dampak, EMV inheren dan residual, skor, keparahan, respons, pemilik, status |
| [`data/realisasi.csv`](https://xyb3rpunq.github.io/mppl-control-tower/data/realisasi.csv) | **Realisasi 40 simpul** (`id,dimulai,mulai,durasi_aktual,biaya_aktual`) — isi, lalu bangun ulang dengan `-realisasi` |
| [`sitemap.xml`](https://xyb3rpunq.github.io/mppl-control-tower/sitemap.xml) | 32 URL dengan pasangan `hreflang` |

---

## 9. Arsitektur

```mermaid
flowchart LR
    subgraph model["internal/model — sumber tunggal kebenaran"]
        D1[Charter, WBS, 40 simpul]
        D2[Risiko, organisasi, mutu]
        D3[Fakta Coretax bersumber]
        D4[Kalender ketersediaan & libur resmi]
        D5[Penggerak risiko, putaran rework, sewa]
    end

    subgraph mesin["Mesin hitung internal/"]
        S[schedule] --> L[level]
        S --> C[compress]
        LP[lp] --> C
        CO[cost] --> C
        CO --> M
        S --> E[evm]
        S --> M[simulate]
        L --> M
        G[gert] --> M
        R[risk]
        Q[quality]
        CT[coretax]
    end

    model --> mesin
    mesin --> SITE[internal/site<br/>analisis + temuan + contoh rumus]
    SITE --> RENDER[internal/render<br/>SVG]
    RENDER --> GEN[cmd/site<br/>html/template]
    GEN --> DIST[(dist/<br/>32 halaman + CSV + JSON + sitemap)]
    mesin --> WASM[cmd/wasm<br/>WebAssembly]
    WASM --> BROWSER[Peramban: 4 panel interaktif]
    DIST --> PAGES[GitHub Pages]
```

Keputusan teknis penting:

- **Grafik dirender di server** sebagai SVG inline. Halaman utuh tanpa JavaScript, bisa dicetak jadi PDF, terbaca mesin pengindeks, tanpa kedipan saat muat.
- **Semua perhitungan dijalankan sekali** per build dan dibagikan ke semua halaman, sehingga halaman biaya dan halaman risiko tidak mungkin melihat BAC yang berbeda. Lima belas simulasi Monte Carlo yang saling bebas dijalankan **paralel** dengan generator berbenih masing-masing — hasilnya identik dengan eksekusi berurutan, waktu analisis turun dari sekitar 25 detik ke 7 detik.
- **Benih acak tetap.** Tanpa itu angka dasbor bergoyang setiap build dan mustahil diverifikasi.
- **Nol dependensi.** Setiap angka bisa ditelusuri sampai ke barisnya.
- **PWA luring**: service worker di akar situs (cache-first untuk aset, network-first untuk halaman) dan manifest.

Ukuran kode: sekitar 14.650 baris Go aplikasi, 4.630 baris Go uji, 2.460 baris templat, 770 baris CSS.

---

## 10. Menjalankan secara lokal

Butuh Go 1.24 atau lebih baru.

```bash
go test ./...
```

```bash
GOOS=js GOARCH=wasm go build -o web/static/js/mppl.wasm ./cmd/wasm
```

```bash
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/static/js/
```

```bash
go run ./cmd/site -out dist -base ""
```

```bash
cd dist && python -m http.server 8231
```

Buka http://127.0.0.1:8231. Bendera generator:

| Bendera | Bawaan | Fungsi |
| --- | --- | --- |
| `-out` | `dist` | Direktori keluaran |
| `-base` | kosong | URL dasar untuk tautan kanonis, `hreflang`, sitemap |
| `-status` | `2025-12-19` | Tanggal data pelaporan Earned Value |
| `-realisasi` | kosong | `realisasi.csv` yang menggantikan realisasi di model; berkas harus memuat setiap aktivitas tepat sekali, baris tidak sah menghentikan build |

---

## 11. Pengujian

**280 fungsi uji di 18 paket, cakupan pernyataan 95,3%.** Satu-satunya fungsi yang tidak tersentuh uji adalah `main` pada generator situs, yang dijalankan langkah build di CI. Sebagian besar uji tidak sekadar memeriksa fungsi berjalan, tetapi **menjaga klaim yang ditampilkan situs tetap benar**:

**Penjadwalan dan kalender**
- Durasi jaringan harus 85 hari kerja = 17 minggu piagam; hari kerja ke-85 jatuh 23 Februari 2026.
- Setiap libur punya dasar hukum, jatuh pada hari kerja, terurut, dan tidak berstatus asumsi.
- Uji tangan forward/backward pass pada jaringan kecil, keempat relasi PDM, lag dan lead.
- Siklus, predecessor tak dikenal, dan kode ganda harus ditolak.

**Levelling dan kompresi**
- Tidak ada satu hari-peran pun melebihi kapasitas; **hari-orang dilestarikan**; setiap relasi dihormati — untuk setiap aturan prioritas.
- **Batas bawah diuji jujur terhadap brute force**: pada jaringan kecil, *semua* urutan aktivitas dicoba, dan tidak satu pun jadwal boleh lebih pendek dari batas bawah; `Optimize` harus menemukan optimum brute force.
- Jadwal proyek terbukti optimal: 100 hari tanpa ujian dan 113 hari dengan ujian, batas bawah sama.
- Justifikasi tidak pernah memperburuk jadwal; mode `Lite` identik dengan mode lengkap.
- **Crashing eksak diuji terhadap brute force**: setiap kombinasi potongan dicoba dengan CPM; biaya termurah per tenggat harus sama dengan LP.
- LP tidak pernah lebih mahal dari serakah; serakah terbukti tidak optimal pada jaringan proyek.
- Simpleks: contoh buku teks, fase 1, tidak layak, tak terbatas, baris artifisial redundan, dan contoh degeneratif Beale.
- Kandidat fast-tracking orang-sama tidak boleh dianggap layak.
- **Premi lembur dihitung ulang menit demi menit** terhadap rumus tertutup; batas 4 jam sehari dan 18 jam seminggu ditolak; setiap aktivitas yang boleh di-crash lembur dalam batas, dan tepat lima melanggarnya.
- **Biaya crash cembung**: biaya marjinal tidak pernah menurun, jumlahnya sama dengan potongan penuh, A17 cocok dengan hitung tangan (Rp 37.812,5 lalu Rp 58.437,5); potongan berhenti di hari terakhir yang sah bila potongan penuh melanggar batas mingguan; LP diuji brute force dengan biaya per hari.
- **Serakah bisa gagal** diuji pada jaringan dengan satu aktivitas bersama yang mahal dan dua cabang murah; laporan optimalitas serakah harus konsisten dengan titik-titiknya, bukan dikunci ke satu hasil.
- **Lembur pada jadwal nyata diuji brute force**: pada jaringan kecil setiap himpunan hari lembur × setiap urutan aktivitas dicoba; durasi minimum dan biaya harus sama. Pada proyek: 113 → 108 terbukti, upah naik saat durasi turun, jam per hari ≤ 3,6 dan per minggu ≤ 18, tidak ada lembur pada hari ujian atau peran paruh waktu, upah dihitung ulang dari pemakaian, sewa dan bersih konsisten.
- **Batas bawah sah pada kapasitas di atas dasar**: brute force pada grid yang dinaikkan, dengan dan tanpa `RateCap`; uji ini **gagal pada kode lama** (batas 9 melampaui jadwal 8) sebelum perbaikan. `RateCap` mempercepat satu pekerjaan (6 hari → 5), orang kedua tidak; `Excess` mencatat jam di atas alokasi.
- **Opsi percepatan**: aturan PP 35/2021 ditolak bila dilanggar, tidak ada lembur pada hari ujian atau untuk peran paruh waktu, upah dihitung tangan, opsi kosong identik dengan prakiraan, orang baru tidak pernah memperlambat.
- **Keputusan sponsor**: opsi tanpa percepatan sama persis dengan prakiraan berjalan; setiap titik JCL 70% benar-benar mencapai 70%; harga per hari, opsi termurah, dan opsi tercepat konsisten dengan angkanya; peran lembur sama dengan rencana termurah durasi minimum; tidak ada temuan yang lagi menyuruh memegang komitmen lain, mengajukan kenaikan cadangan terpisah, atau menjanjikan hari dari crashing CPM.
- **Pita nilai dihitung tangan**: selubung atas empat garis (termasuk opsi yang tidak pernah terbaik), seri, opsi yang lebih cepat dan lebih murah sekaligus; opsi berasumsi tidak boleh menjadi termurah. Pada analisis sungguhan manfaat bersih opsi pita memang terbesar di tengah pitanya, angka titik berada di dalam interval bootstrap 90%, dan setiap skenario benar-benar memakai asumsi yang dinamainya.
- **`FirstFeasible` sama dengan pemindaian `Frontier`**; bootstrap berpasangan berbenih dan menarik indeks yang sama; `ReworkScale` menggeser rerata putaran sesuai p/(1−p).
- **Realisasi**: ekspor → impor mengembalikan realisasi persis; mengubah satu durasi mengubah kalibrasi prakiraan; sepuluh jenis berkas rusak ditolak tanpa mengubah apa pun.
- `Search` mencapai batas bawah pada jaringan kecil dengan urutan buruk, berbenih deterministik, jujur melaporkan target yang mustahil, dan menolak jaringan bersiklus.

**Simulasi**
- Benih sama → hasil identik; benih berbeda → kesimpulan stabil.
- **Lapisan L0 harus identik persis dengan halaman PERT.**
- ρ = 0 memberi korelasi terealisasi mendekati nol; ρ = 0,8 menaikkannya dan melebarkan sebaran.
- Setiap lapisan realisme menggeser P80 ke arah yang benar; JCL tidak pernah melebihi peluang marginal; peluang bersama di P80×P80 selalu di bawah 80%.
- Frekuensi kejadian risiko cocok dengan peluang residual; frontier JCL tidak naik saat tenggat dilonggarkan.
- **Kenaikan rerata biaya dari L1 ke L2, di luar sewa yang ikut memanjang, harus mendekati EMV residual register** (pemeriksaan silang antar-halaman).
- **Kopula risiko menjaga peluang marginal** setiap risiko dan rerata biaya, sementara phi terealisasi naik bersama λ dan P95 biaya menebal.
- Faktor laten peran yang dibaca kopula risiko benar-benar faktor yang menggerakkan durasi aktivitas.
- **Rerata putaran rework di simulasi cocok dengan bentuk tertutup GERT** p/(1−p); aljabar Mason cocok dengan rumus geometrik.
- Biaya sewa pada jadwal rencana persis sama dengan anggaran; BAC tetap Rp 14.832.000.
- **Prakiraan berjalan**: status per aktivitas pada tanggal data, AC sama dengan mesin EVM, rumus kredibilitas, risiko berstatus "terjadi" tidak disampel lagi, tidak ada prakiraan yang selesai sebelum realisasi atau berbiaya di bawah AC.
- **Setiap iterasi lapisan kapasitas terbukti optimal** — 10.000 dari 10.000 di simulasi perencanaan dan prakiraan berjalan; hasilnya identik berapa pun jumlah pekerja paralel.
- **Kredibilitas empiris** diuji pada titik hitung tangan (selisih di dalam derau → Z = 0, selisih² = 2 × derau → Z = 0,5, tanpa derau → Z = 1) dan monoton; varians beta-PERT cocok dengan integrasi numerik kuantil sampler; bobot paksa masih mengembalikan n/(n+k).
- Nilai-nilai I_x(a,b) terhadap solusi tertutup; rerata beta-PERT sama dengan te; CDF(Q(u)) = u.

**Anggaran, risiko, mutu**
- BAC + kontinjensi + cadangan manajemen = Rp 16.000.000 persis; aturan 100% WBS dua arah.
- Tepat satu Accountable per baris RACI.
- Titik di luar UCL harus terdeteksi aturan 1 (batas yang digambar = batas yang diuji).

**Render dan konten**
- Seluruh 32 halaman dirender tanpa galat, tanpa sisa sintaks templat, tanpa kunci terjemahan hilang.
- **Halaman Inggris tidak boleh memuat kata fungsi Indonesia** — uji ini merender HTML sungguhan lalu memindainya, dan menemukan bocoran nyata (label status, notasi rumus, nilai fakta, metrik temuan) yang lolos dari pemeriksaan kelengkapan kamus.
- Tidak ada singkatan bulan Indonesia di dalam kalimat Inggris.
- Angka kunci — termasuk angka halaman Optimasi, Simulasi Terpadu, dan Prakiraan Berjalan — harus benar-benar sampai ke HTML dalam kedua bahasa, diformat dari struct analisis, bukan diketik.
- Setiap tabel harus berkelas `data` agar aturan CSS layar sempit menjadikannya wadah geser, dan aturan itu harus tetap ada.
- Lembur jadwal nyata harus berangkat dari jadwal levelling halaman, tampil di kedua bahasa, dan beranda tidak boleh lagi menjanjikan hari "dibeli lewat crashing".
- Rentang premi lembur, tautan PP 35/2021, dan ketiga Z empiris harus tampil di kedua bahasa, dan teks asumsi lama ("k = ", "premi (asumsi)") tidak boleh tersisa.
- `metrik.json` harus JSON sah dengan seluruh blok baru; seluruh temuan penutup celah harus diturunkan.
- Setiap fakta Coretax punya URL sumber yang tertaut di HTML dengan `rel="noopener"`.
- Setiap SVG utuh, beraksesibilitas, bebas NaN, tanpa warna heksadesimal langsung.
- **Setiap grafik di 32 halaman punya panduan baca** (di dalam figurnya atau panduan bersama di bagiannya); judul panduan sesuai bahasa, minimal dua butir cara membaca, dan arti berupa kalimat tanpa sisa templat.
- **Arti grafik mengutip angka analisis**: jumlah metrik gagal, hari tambahan dari kapasitas dan ujian, peluang tepat waktu PERT, konflik kapasitas, P80 L4, penurunan EMV, dan SPI dibandingkan langsung dengan struct.
- **Grafik penjelas diuji isinya, bukan hanya bentuknya**: tanggal selesai tertulis pada titiknya, panah hanya dari opsi dasar, batas pita di nol tidak digambar, tanda ✓/✗ skenario, delta negatif berkelas turun, pita derau hanya bila derau diketahui, ekor frontier datar dipotong, kelas kepekatan kapasitas dibatasi 1–5, fase tanpa indeks ditandai "belum dimulai", dan kenaikan ditulis "naik" bukan "turun" negatif. Masukan kosong, timpang, atau datar tidak boleh panik.
- **Grafik rincian diuji isinya**: perubahan Rp 3 ribu pada Rp 19 juta di panel sapuan harus setinggi paling banyak 2 px (uji ini gagal dengan skala otomatis: 55 px), peluang GERT menjumlah 1, kolom negatif setinggi nol, sumbu peluang tidak melewati 100%, garis M di antara O dan P, segmen nol dan label yang tidak muat dilewati, titik tanpa label digambar di bawah, batang divergen mengikuti tanda yang ada dan labelnya tidak menimpa nama baris, dan linimasa menjaga skala waktu (jeda tujuh tahun lebih dari 20 kali jeda enam minggu) serta memisahkan lajur kejadian yang berdekatan.
- **Klaim kalimat arti dicek secara independen**: frekuensi setiap risiko dalam 4 simpangan baku binomial peluang register; risiko tertutup berfrekuensi nol di prakiraan; condong kanan beta-PERT sama dengan hitung tangan O + P > 2M; pecahan pergeseran menjumlah ke total pergeseran; anggaran fase menjumlah ke BAC; kandidat fast-tracking layak + orang sama + tanpa hari = semua; Q90 GERT konsisten dengan sebaran geometriknya; sewa L4 di atas rencana. Fungsi grafik mengembalikan kosong, bukan panik, tanpa paket keputusan atau prakiraan berjalan.
- **Penempat label dan penyebar label diuji sifatnya**: delapan label dua baris pada titik yang hampir sama tidak boleh bertabrakan atau keluar kanvas, dan label tetap berurutan berjarak di dalam batas. Kedua uji ini **gagal pada kode lama** (kandidat terlalu rapat untuk label dua baris; satu label di luar batas menumpuk semua label) sebelum diperbaiki. Garis yang didaftarkan sebagai rintangan membuat label pindah ke kandidat yang bebas garis.
- Setiap rumus punya contoh hitung dalam kedua bahasa.

---

## 12. CI/CD

Dua alur kerja GitHub Actions:

**`uji`** — setiap push dan pull request: `gofmt`, `go vet`, `go test -race`, laporan cakupan, kompilasi WebAssembly, build situs, dan pemeriksaan keluaran (32 halaman termasuk halaman prakiraan dan keputusan dua bahasa, `realisasi.csv`, sitemap, service worker, metrik).

**`terbitkan`** — setiap push ke `main`: uji ulang, kompilasi WebAssembly, `configure-pages` (dijalankan **sebelum** build agar URL dasar benar), build situs, lalu penerbitan ke GitHub Pages.

---

## 13. Asumsi, celah yang ditutup, dan batas yang tersisa

**Asumsi** (seluruhnya dinyatakan juga di halaman Metode):

| Asumsi | Akibat bila keliru |
| --- | --- |
| Kemajuan linear dalam aktivitas | Aturan 0/100 atau 50/50 memberi EV berbeda |
| Data realisasi SIATS adalah skenario pelaksanaan yang wajar (proyek kuliah tidak dieksekusi) | Angka EV dan prakiraan berjalan berubah; rumus dan cara membaca tidak |
| Korelasi peran ρ = 0,5 | P80 hanya bergeser satu hari; lebar sebaran yang berubah |
| Kapasitas 40% selama empat periode ujian resmi | Tanpa jendela ujian levelling masih 100 hari kerja; realisasi UTS memperbarui faktornya menjadi 98,8% |
| Laju mulai minimum 20% (satu hari kerja per minggu) | Ambang 25% memberi jadwal terbaik satu hari lebih panjang |
| Lima penggerak risiko bersama, λ = 0,6 | Rerata biaya tidak berubah pada λ berapa pun; hanya ekor |
| Peluang gagal GERT 30% (regresi) dan 25% (uji penetrasi) | Rerata putaran p/(1−p) tidak linear |
| Kredibilitas dengan estimator momen satu kelompok | Satu selisih besar yang kebetulan bisa terbaca sistematis; prakiraan tanpa belajar ditampilkan sebagai pembanding |
| Sewa & langganan sebanding dengan rentang pemakaian | Vendor bulanan membuat biaya naik bertahap, bukan halus |
| Crashing = lembur PP 35/2021 tanpa kehilangan efisiensi koordinasi; peluang rework fast-tracking 30% | Premi aturan adalah batas bawah — biaya crash sesungguhnya hanya bisa lebih tinggi |
| Lembur jadwal nyata: tidak pada periode ujian, tidak untuk peran paruh waktu (DevOps), paling banyak 3,6 jam/hari | Lembur saat ujian atau untuk DevOps bisa memotong lebih banyak; lantai yang terbukti hanya berlaku di dalam aturan ini |
| Alokasi rencana tetap (jam di atas alokasi = lembur); orang baru dibayar dari tanggal data sampai pekerjaan terakhir perannya; masa adaptasi 10 hari pada 50% | Bila orangnya sebenarnya menganggur, lembur lebih murah; tarif kontraktor nyata di atas kartu proyek membuat opsi orang baru lebih mahal |
| Peluang gagal cutover Coretax 35% (skenario) | Titik impasnya 0,279% — kesimpulan bertahan |

**Celah yang sudah ditutup** (lima celah versi sebelumnya):

1. **Levelling adalah heuristik** → jadwal 113 hari **terbukti optimal**: batas atas dari enam aturan, 300 sampel berbias, dan justifikasi; batas bawah energetik dan destruktif 113; diuji jujur terhadap brute force.
2. **Kejadian risiko saling bebas** → **kopula faktor** dengan lima penggerak bersama; penggerak kinerja pengembang berkorelasi dengan durasi aktivitas Backend Developer; phi terealisasi 0,18.
3. **Biaya tidak punya komponen bergantung waktu** → **sewa dan langganan** mengikuti rentang pemakaian di setiap iterasi, dan crashing dihitung ulang sebagai **trade-off biaya total dengan LP**.
4. **Simulasi berpandangan perencanaan** → halaman **Prakiraan Berjalan**: realisasi dikunci, durasi bersyarat, kredibilitas Bühlmann, kalibrasi kapasitas ujian.
5. **Tidak ada pengulangan kerja (GERT)** → dua putaran rework direduksi dengan **aturan Mason** dan disimulasikan sebagai lapisan L3; analitik dan Monte Carlo saling cocok.

**Celah putaran kedua** (sebelumnya tercantum sebagai batas yang tersisa):

6. **Levelling di dalam simulasi hanya cepat** → setiap iterasi kini **dibuktikan optimal** terhadap batas bawahnya sendiri: 10.000 dari 10.000, dengan waktu build tetap wajar karena CPM di-cache dan iterasi dijalankan paralel.
7. **Bobot kredibilitas k = 10 dipilih** → **estimator momen Bühlmann**: Z dihitung dari selisih dibanding derau estimasi. Hasilnya berbeda nyata dari k = 10: durasi Z 0 (bukan 61,5%), biaya Z 85%, ujian Z 98%.
8. **Premi crash 75% diasumsikan** → **premi lembur PP 35/2021** per aktivitas (75%–87,5%), dan lima aktivitas dua hari ternyata tidak boleh dipotong secara hukum; durasi crash minimum naik dari 63 ke 68 hari.

**Celah putaran ketiga**:

9. **Slope crash rata-rata** → **biaya lembur per hari** yang cembung, di model, serakah, dan LP. Akibatnya klaim lama "serakah tidak optimal" ternyata artefak slope rata-rata; kini semua teks mengikuti hasil perbandingan, bukan ditulis tetap.
10. **Saran membeli hari dari kurva crashing CPM** untuk jadwal yang tidak bisa dijalankan → **lembur sah pada jadwal levelling** (angkanya dikoreksi lagi pada putaran keempat).
11. **12 dari 15 halaman bergeser horizontal di layar HP** → tabel menjadi wadah geser di layar sempit; 30 halaman tanpa geser pada 400 px, dikunci uji struktur.

**Celah putaran keempat** (eksekusi rekomendasi):

12. **Rekomendasi "pakai 108 hari sebagai lantai" tidak bisa dijalankan** — rencananya butuh lembur mulai 20 Okt 2025, sebelum tanggal data → seluruh opsi percepatan dihitung ulang **dari tanggal data** pada jaringan sisa, dengan lantai terbukti dan harga per hari pada titik JCL 70%.
13. **Batas bawah tidak sah pada kapasitas di atas dasar** (ditemukan saat opsi lembur memberi batas 98 untuk jadwal 97) → penalaran tak bertanggal memakai kapasitas dan laju tertinggi; uji brute force yang gagal pada kode lama.
14. **Lembur tidak bisa mempercepat satu pekerjaan** di model levelling → `RateCap` 1 + h/8; lantai perencanaan dengan lembur turun dari 108 ke **101**; jam di atas alokasi rencana kini dibayar (`Excess`), sehingga kurva lembur dan opsi lembur konsisten (95 = 95).
15. **Enam komitmen di enam halaman** → halaman **Keputusan Sponsor**: satu komitmen, satu permintaan anggaran; temuan P80, JCL perencanaan, EAC, cadangan, dan crashing CPM kini merujuk ke sana.
16. **"Kumpulkan realisasi" tanpa jalur masuk** → `realisasi.csv` diekspor dan diimpor lewat `-realisasi` dengan validasi ketat.

**Celah putaran kelima**:

17. **Harga per hari tanpa aturan keputusan** → pita nilai satu hari lebih cepat: sponsor cukup menyatakan rentang nilainya.
18. **Rekomendasi yang mungkin hanya derau** → bootstrap berpasangan 200 ulangan (98,5% dan 96,5%) dan enam skenario asumsi tanpa data (6 dari 6 sama).

Tambahan yang ditemukan selama penutupan: crashing serakah ternyata tidak optimal (kini LP eksak); kalender libur kini resmi dan menambahkan cuti bersama 16 Februari 2026; keempat periode ujian diambil dari lampiran kalender akademik resmi, dan UAS ganjil ternyata 19–31 Januari 2026 — seminggu lebih lambat dari asumsi lama 12–23 Januari.

**Batas yang tersisa** (bukan pekerjaan yang lupa, melainkan batas yang harus diketahui):

1. **Optimalitas berlaku di dalam model isi pekerjaan** — laju pecahan tanpa biaya berpindah konteks; tim sungguhan bisa sedikit lebih lambat.
2. **Parameter tanpa data tetap asumsi** — kapasitas ujian dan bobot kredibilitas kini diestimasi dari realisasi, premi lembur dari PP 35/2021; peluang gagal GERT, λ risiko, dan peluang rework fast-tracking masih asumsi dengan uji kepekaan karena belum ada data untuk mengukurnya.
3. **Premi lembur adalah batas bawah, biaya lembur jadwal nyata adalah batas atas** — aturan tidak memuat kehilangan efisiensi koordinasi; biaya lembur per durasi pada jadwal levelling adalah rencana termurah yang ditemukan, bukan minimum terbukti.
4. **Batas lembur crashing CPM diperiksa per aktivitas, bukan per orang** — pada jaringan CPM satu orang sudah terjadwal di dua pekerjaan sekaligus, jadi pemeriksaan per orang baru bermakna di jadwal levelling (dan di sana sudah dilakukan).
5. **Estimator kredibilitas memakai satu kelompok data** — satu selisih besar yang kebetulan bisa terbaca sebagai penyimpangan sistematis; data lintas proyek akan menstabilkannya.
6. **Data realisasi adalah skenario** — prakiraan berjalan memperagakan metodenya; `realisasi.csv` adalah jalur untuk data nyata.
7. **Nilai satu hari lebih cepat tetap harus datang dari sponsor** — aturan pita memberi opsi terbaik per rentang, tetapi rentang yang benar bergantung pada denda keterlambatan atau tenggat eksternal yang tidak tercatat.
9. **Skenario asumsi memakai cara yang lebih ringan** — 3.000 iterasi dengan levelling cepat, dibandingkan dengan asumsi dasar dengan cara yang sama; yang diuji urutan opsi, bukan nilai batasnya. Satu iterasi dari 10.000 pada opsi masa adaptasi belum terbukti optimal (celah satu hari).
8. **Opsi orang baru memakai tarif kartu proyek dan ketersediaan pada tanggal data** — tarif dan tanggal mulai kontraktor nyata belum ada.

---

## 14. Sumber data

**Proyek SIATS** — tugas mata kuliah Manajemen Proyek Perangkat Lunak, Universitas Esa Unggul: Tugas 2 (9 area pengetahuan), Tugas 3 (siklus hidup), Tugas 5 (uraian peran), Tugas 6 (Project Charter & WBS), Tugas 10 (bagan organisasi & RACI); serta materi Modul 2, Modul 4, Pertemuan 3, Pertemuan 7 (MS Project), dan Pertemuan 9 (manajemen mutu, Schwalbe bab 8).

**Kalender** — [Kalender Akademik Universitas Esa Unggul TA 2025/2026](https://www.esaunggul.ac.id/en/kalender-akademik-tahun-akademik-2025-2026/) (SK Rektor No. 039/SK-R/UEU/III/2025, lampiran halaman 1–2) untuk tanggal UTS dan UAS; [SKB 3 Menteri libur nasional dan cuti bersama 2026](https://setneg.go.id/baca/index/inilah_skb_3_menteri_libur_nasional_dan_cuti_bersama_2026) dan [SKB perubahan 2025](https://www.kompas.com/jawa-tengah/read/2025/12/09/104500088/apakah-tanggal-26-desember-2025-cuti-bersama-ini-jawabannya-sesuai) untuk hari libur.

**Upah lembur** — [PP No. 35 Tahun 2021](https://learning.hukumonline.com/wp-content/uploads/2021/03/Peraturan-Pemerintah-Nomor-35-tahun-2021-Perjanjian-Kerja-Waktu-Tertentu-Alih-Daya-Waktu-Kerja-dan-Waktu-Istirahat-dan-Pemutusan-Hubungan-Kerja.pdf) Pasal 26 (lembur paling lama 4 jam sehari dan 18 jam seminggu), Pasal 31 (jam pertama 1,5×, jam berikutnya 2× upah sejam), dan Pasal 32 (upah sejam = 1/173 upah sebulan).

**Studi kasus Coretax** — pemberitaan publik Kompas, Tempo, Hukumonline, DDTC News, Beritasatu, dan keterangan resmi Direktorat Jenderal Pajak; daftar lengkap dengan tanggal ada di [halaman studi kasus](https://xyb3rpunq.github.io/mppl-control-tower/coretax/).

**Metode** — PMBOK; Kolisch (1996) dan Kolisch & Hartmann (1999) untuk SGS dan aturan prioritas; Valls, Ballestín & Quintanilla (2005) untuk justifikasi; Baptiste, Le Pape & Nuijten (2001) untuk penalaran energetik; Klein & Scholl (1999) untuk batas bawah destruktif; Kelley (1961) untuk crashing dengan LP; Pritsker (1966) untuk GERT; Bühlmann (1967) untuk kredibilitas dan Klugman, Panjer & Willmot (*Loss Models*) untuk estimasi parameter kredibilitas secara empiris; Lipke (2003) untuk Earned Schedule; Nelson (1984) untuk aturan peta kendali; Vose (2008) untuk beta-PERT; Numerical Recipes untuk fungsi beta tak lengkap; NASA Cost Estimating Handbook untuk JCL 70%.

Situs ini tidak berafiliasi dengan Direktorat Jenderal Pajak maupun pihak mana pun yang disebut. Analisisnya adalah kerja akademik.

## Lisensi

MIT.
