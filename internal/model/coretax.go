package model

// Berkas ini memuat studi kasus Coretax DJP - proyek perangkat lunak
// pemerintah terbesar di Indonesia dalam satu dekade terakhir, dan contoh
// paling mahal dari kegagalan yang metode MPPL dirancang untuk mencegah.
//
// ATURAN MAIN BERKAS INI: setiap angka pada Verified punya satu URL sumber
// publik. Tidak ada angka yang ditaksir sendiri. Segala sesuatu yang berupa
// hitungan turunan berada di Derived, dan segala sesuatu yang berupa andaian
// berada di Scenario dan diberi label sebagai skenario - bukan fakta.
//
// Pemisahan itu bukan formalitas. Inti pelajaran Coretax adalah soal angka
// yang diyakini padahal tidak pernah diperiksa; mengulangi kesalahan yang sama
// di dalam analisis tentang kesalahan itu akan konyol.

// Source adalah rujukan publik untuk sebuah fakta.
type Source struct {
	Label     Text
	Publisher string
	URL       string
	Date      string
}

// CoretaxFact adalah satu fakta terverifikasi tentang proyek Coretax.
type CoretaxFact struct {
	Key       string
	Label     Text
	Value     Text    // nilai siap tampil
	Numeric   float64 // nilai numerik bila relevan, 0 bila tidak
	Unit      string
	Detail    Text
	SourceIdx []int // indeks ke CoretaxSources
}

// CoretaxSources adalah daftar sumber yang dipakai studi kasus ini.
var CoretaxSources = []Source{
	{ // 0
		Publisher: "Kompas", Date: "2025-05-11",
		Label: Text{ID: "Ironi Coretax, Habiskan Rp 1,3 Triliun, tapi Sering Eror", EN: "The Coretax irony: IDR 1.3 trillion spent, yet frequent errors"},
		URL:   "https://money.kompas.com/read/2025/05/11/100729826/ironi-coretax-habiskan-rp-13-triliun-tapi-sering-eror",
	},
	{ // 1
		Publisher: "Hukumonline", Date: "2025-01-16",
		Label: Text{ID: "Mengenal 3 Perusahaan Asing dalam Proyek Coretax Senilai Rp1,3 Triliun", EN: "The three foreign firms in the IDR 1.3 trillion Coretax project"},
		URL:   "https://www.hukumonline.com/berita/a/mengenal-3-perusahaan-asing-dalam-proyek-coretax-senilai-rp1-3-triliun-lt6789b53a48f69",
	},
	{ // 2
		Publisher: "Kompas", Date: "2025-01-28",
		Label: Text{ID: "169 Pegawai Khusus PSIAP di Balik Pembuatan Coretax", EN: "The 169 dedicated PSIAP staff behind Coretax"},
		URL:   "https://money.kompas.com/read/2025/01/28/113800626/169-pegawai-khusus-psiap-di-balik-pembuatan-coretax-yang-masih-bermasalah",
	},
	{ // 3
		Publisher: "Tempo", Date: "2025-02-12",
		Label: Text{ID: "Sejarah Sistem Coretax: Didesain sejak 2018", EN: "History of Coretax: designed since 2018"},
		URL:   "https://www.tempo.co/ekonomi/sejarah-sistem-coretax-didesain-sejak-2018-telan-anggaran-rp-1-2-triliun-hingga-panen-keluhan--1205508",
	},
	{ // 4
		Publisher: "Kompas", Date: "2025-03-14",
		Label: Text{ID: "Pendapatan Negara Turun Drastis, Benarkah Coretax Biang Keladinya?", EN: "State revenue falls sharply - is Coretax to blame?"},
		URL:   "https://money.kompas.com/read/2025/03/14/120000726/pendapatan-negara-turun-drastis-benarkah-coretax-biang-keladinya-",
	},
	{ // 5
		Publisher: "Kompas", Date: "2025-03-13",
		Label: Text{ID: "Penerimaan Pajak Anjlok, Ekonom: Biang Keladinya Permasalahan Coretax", EN: "Tax revenue slumps; economists point to Coretax problems"},
		URL:   "https://money.kompas.com/read/2025/03/13/093900526/penerimaan-pajak-anjlok-ekonom--biang-keladinya-permasalahan-coretax",
	},
	{ // 6
		Publisher: "DDTC News", Date: "2025-02-10",
		Label: Text{ID: "Batas Upload Faktur Pajak Tak Mundur Meski Coretax Terkendala", EN: "Tax invoice upload deadline unchanged despite Coretax trouble"},
		URL:   "https://news.ddtc.co.id/berita/nasional/1808811/ingat-batas-upload-faktur-pajak-tak-mundur-meski-coretax-terkendala",
	},
	{ // 7
		Publisher: "DDTC News", Date: "2025-06-18",
		Label: Text{ID: "Kejar Target Penerimaan Pajak, Perbaikan Coretax Tak Boleh Molor", EN: "Chasing the revenue target: Coretax fixes must not slip"},
		URL:   "https://news.ddtc.co.id/berita/nasional/1810594/kejar-target-penerimaan-pajak-perbaikan-coretax-tak-boleh-molor",
	},
	{ // 8
		Publisher: "DDTC News", Date: "2025-12-22",
		Label: Text{ID: "Refleksi Satu Tahun Coretax", EN: "Reflecting on one year of Coretax"},
		URL:   "https://news.ddtc.co.id/review/opini/1815906/refleksi-satu-tahun-coretax-renungan-untuk-melakukan-perbaikan",
	},
	{ // 9
		Publisher: "Direktorat Jenderal Pajak", Date: "2025-01-01",
		Label: Text{ID: "Keterangan Tertulis Perkembangan Informasi Terkini Coretax DJP", EN: "DJP written statement on the latest Coretax developments"},
		URL:   "https://pajak.go.id/en/node/115511",
	},
	{ // 10
		Publisher: "Beritasatu", Date: "2026-01-20",
		Label: Text{ID: "Target Pajak 2026 Naik, DJP Diminta Evaluasi Coretax", EN: "2026 tax target rises; DJP urged to evaluate Coretax"},
		URL:   "https://www.beritasatu.com/ekonomi/2960243/target-pajak-2026-naik-djp-diminta-evaluasi-coretax",
	},
}

// Konstanta terverifikasi. Dipisah sebagai konstanta bernama supaya setiap
// perhitungan turunan di seluruh aplikasi memakai angka yang sama persis.
const (
	// CoretaxContractValue adalah nilai kontrak konsorsium LG CNS-Qualysoft.
	CoretaxContractValue = 1_228_000_000_000.0
	// CoretaxConsultantValue adalah nilai kontrak Owner's Agent Deloitte
	// Consulting, sudah termasuk pajak.
	CoretaxConsultantValue = 110_300_000_000.0
	// CoretaxLostRevenueJan2025 adalah potensi penerimaan pajak yang hilang
	// pada Januari 2025 menurut catatan Kementerian Keuangan.
	CoretaxLostRevenueJan2025 = 64_000_000_000_000.0
	// CoretaxJanuaryRevenue adalah realisasi penerimaan perpajakan Januari 2025.
	CoretaxJanuaryRevenue = 115_180_000_000_000.0
	// CoretaxQ1Revenue adalah penerimaan pajak kuartal I 2025.
	CoretaxQ1Revenue = 322_600_000_000_000.0
	// CoretaxPSIAPHeadcount adalah jumlah anggota tim pelaksana PSIAP.
	CoretaxPSIAPHeadcount = 169
)

// CoretaxFacts adalah fakta terverifikasi, masing-masing dengan sumbernya.
var CoretaxFacts = []CoretaxFact{
	{
		Key: "dasar-hukum", Value: Text{ID: "Perpres 40/2018", EN: "Presidential Reg. 40/2018"}, SourceIdx: []int{3},
		Label:  Text{ID: "Dasar hukum program", EN: "Programme legal basis"},
		Detail: Text{ID: "Pembaruan Sistem Inti Administrasi Perpajakan (PSIAP) ditetapkan lewat Peraturan Presiden Nomor 40 Tahun 2018; sistemnya dirancang sejak 2018 dengan pendekatan commercial off-the-shelf.", EN: "The core tax administration system renewal (PSIAP) was set by Presidential Regulation 40/2018; the system was designed from 2018 using a commercial off-the-shelf approach."},
	},
	{
		Key: "tim-psiap", Value: Text{ID: "169 orang, mulai 1 Nov 2020", EN: "169 people, from 1 Nov 2020"}, Numeric: CoretaxPSIAPHeadcount, Unit: "orang", SourceIdx: []int{2},
		Label:  Text{ID: "Tim pelaksana PSIAP", EN: "PSIAP implementation team"},
		Detail: Text{ID: "Tim khusus beranggotakan 169 orang mulai bertugas 1 November 2020, sebulan sebelum pemenang tender ditetapkan.", EN: "A dedicated 169-person team started on 1 November 2020, a month before the tender winner was named."},
	},
	{
		Key: "pemenang-tender", Value: Text{ID: "Konsorsium LG CNS - Qualysoft", EN: "LG CNS - Qualysoft consortium"}, SourceIdx: []int{1, 0},
		Label:  Text{ID: "Pemenang tender", EN: "Tender winner"},
		Detail: Text{ID: "Ditetapkan lewat Keputusan Menteri Keuangan Nomor 549/KMK.03/2020 tanggal 1 Desember 2020. PT PricewaterhouseCoopers Consulting Indonesia bertindak sebagai agen pengadaan.", EN: "Named by Minister of Finance Decree 549/KMK.03/2020 dated 1 December 2020. PT PricewaterhouseCoopers Consulting Indonesia acted as procurement agent."},
	},
	{
		Key: "nilai-kontrak", Value: Text{ID: "Rp 1,228 triliun", EN: "IDR 1.228 trillion"}, Numeric: CoretaxContractValue, Unit: "Rp", SourceIdx: []int{0, 1},
		Label:  Text{ID: "Nilai kontrak utama", EN: "Primary contract value"},
		Detail: Text{ID: "Kontrak pengadaan sistem inti kepada konsorsium LG CNS-Qualysoft.", EN: "Core system procurement contract awarded to the LG CNS-Qualysoft consortium."},
	},
	{
		Key: "konsultan", Value: Text{ID: "Rp 110,3 miliar", EN: "IDR 110.3 billion"}, Numeric: CoretaxConsultantValue, Unit: "Rp", SourceIdx: []int{0},
		Label:  Text{ID: "Kontrak Owner's Agent", EN: "Owner's Agent contract"},
		Detail: Text{ID: "PT Deloitte Consulting sebagai Owner's Agent, nilai sudah termasuk pajak.", EN: "PT Deloitte Consulting as Owner's Agent; the figure includes tax."},
	},
	{
		Key: "total-apbn", Value: Text{ID: "lebih dari Rp 1,3 triliun", EN: "more than IDR 1.3 trillion"}, SourceIdx: []int{0, 1},
		Label:  Text{ID: "Total dana APBN terserap", EN: "Total state budget absorbed"},
		Detail: Text{ID: "Angka Rp 1,2 triliun pada kontrak utama belum termasuk gaji tim PSIAP, sehingga total belanja negara melampaui Rp 1,3 triliun.", EN: "The IDR 1.2 trillion primary contract excludes PSIAP team salaries, bringing total state spending past IDR 1.3 trillion."},
	},
	{
		Key: "go-live", Value: Text{ID: "1 Januari 2025", EN: "1 January 2025"}, SourceIdx: []int{0, 9},
		Label:  Text{ID: "Tanggal go-live", EN: "Go-live date"},
		Detail: Text{ID: "Coretax menggantikan sistem administrasi perpajakan yang dipakai DJP sejak 2002, serentak untuk seluruh wajib pajak.", EN: "Coretax replaced the tax administration system DJP had used since 2002, for all taxpayers at once."},
	},
	{
		Key: "gejala", Value: Text{ID: "gagal login, gagal input, faktur tertunda", EN: "login failures, input failures, stalled invoices"}, SourceIdx: []int{0},
		Label:  Text{ID: "Gejala kegagalan yang dilaporkan", EN: "Reported failure symptoms"},
		Detail: Text{ID: "Wajib pajak tidak bisa login, gagal memasukkan data, transaksi tertunda, halaman kosong, sistem lambat merespons, dan kesulitan membuat faktur pajak.", EN: "Taxpayers could not log in, data entry failed, transactions stalled, pages came back blank, the system responded slowly, and invoice creation was difficult."},
	},
	{
		Key: "penerimaan-januari", Value: Text{ID: "Rp 115,18 triliun (turun 34,5%)", EN: "IDR 115.18 trillion (down 34.5%)"}, Numeric: CoretaxJanuaryRevenue, Unit: "Rp", SourceIdx: []int{0},
		Label:  Text{ID: "Penerimaan perpajakan Januari 2025", EN: "January 2025 tax revenue"},
		Detail: Text{ID: "Realisasi pendapatan negara per 31 Januari 2025 turun 28,3%, dengan penerimaan perpajakan turun 34,5%.", EN: "State revenue realisation as of 31 January 2025 fell 28.3%, with tax revenue down 34.5%."},
	},
	{
		Key: "potensi-hilang", Value: Text{ID: "Rp 64 triliun", EN: "IDR 64 trillion"}, Numeric: CoretaxLostRevenueJan2025, Unit: "Rp", SourceIdx: []int{4},
		Label:  Text{ID: "Potensi penerimaan hilang, Januari 2025", EN: "Potential revenue lost, January 2025"},
		Detail: Text{ID: "Kementerian Keuangan mencatat potensi penerimaan pajak yang hilang akibat masalah Coretax pada Januari 2025 mencapai Rp 64 triliun, dengan penerimaan bulan itu turun 41,8%.", EN: "The Ministry of Finance recorded up to IDR 64 trillion in potential tax revenue lost to Coretax problems in January 2025, with that month's revenue down 41.8%."},
	},
	{
		Key: "kuartal-1", Value: Text{ID: "Rp 322,6 triliun (-19% yoy, 14,7% dari target)", EN: "IDR 322.6 trillion (-19% yoy, 14.7% of target)"}, Numeric: CoretaxQ1Revenue, Unit: "Rp", SourceIdx: []int{5},
		Label:  Text{ID: "Penerimaan pajak kuartal I 2025", EN: "Q1 2025 tax revenue"},
		Detail: Text{ID: "Terkontraksi 19% secara tahunan dan baru mencapai 14,7% dari target setahun.", EN: "Down 19% year on year and only 14.7% of the annual target."},
	},
	{
		Key: "tenggat-faktur", Value: Text{ID: "15 Februari 2025, tidak diundur", EN: "15 February 2025, not extended"}, SourceIdx: []int{6},
		Label:  Text{ID: "Tenggat unggah faktur pajak Januari 2025", EN: "January 2025 tax invoice upload deadline"},
		Detail: Text{ID: "Tenggat tetap 15 Februari 2025 meskipun sistemnya masih terkendala - beban kegagalan sistem dipindahkan ke wajib pajak.", EN: "The deadline stayed at 15 February 2025 even though the system was still failing - the cost of the failure was shifted onto taxpayers."},
	},
	{
		Key: "janji-perbaikan", Value: Text{ID: "Mei 2025, diundur ke 31 Juli 2025", EN: "May 2025, pushed to 31 July 2025"}, SourceIdx: []int{7},
		Label:  Text{ID: "Tenggat perbaikan yang dijanjikan DJP", EN: "DJP's promised remediation deadline"},
		Detail: Text{ID: "DJP semula menjanjikan perbaikan paling lambat Mei 2025, kemudian mengundurkannya ke 31 Juli 2025.", EN: "DJP first promised fixes by May 2025, then moved the date to 31 July 2025."},
	},
	{
		Key: "status-2026", Value: Text{ID: "masih dievaluasi pada musim SPT 2026", EN: "still under review in the 2026 filing season"}, SourceIdx: []int{8, 10},
		Label:  Text{ID: "Status setelah satu tahun", EN: "Status after one year"},
		Detail: Text{ID: "Memasuki 2026, wajib pajak orang pribadi masih mengalami kendala aktivasi akun dan wajib pajak badan harus beradaptasi dengan pelaporan SPT lewat sistem baru; DJP diminta mengevaluasi menyeluruh seiring target pajak 2026 yang naik.", EN: "Into 2026, individual taxpayers still hit account activation problems and corporate taxpayers must adapt to filing through the new system; DJP has been urged to run a full evaluation as the 2026 tax target rises."},
	},
}

// CoretaxMilestone adalah satu tonggak pada linimasa yang direkonstruksi.
// Semua tanggal berasal dari sumber publik; tidak ada yang dikira-kira.
type CoretaxMilestone struct {
	Date      string
	Label     Text
	Phase     string
	Kind      string // "regulasi", "pengadaan", "pelaksanaan", "cutover", "dampak", "remediasi"
	SourceIdx []int
}

// CoretaxTimeline adalah linimasa proyek Coretax dari regulasi sampai
// stabilisasi. Tanggal yang hanya diketahui tahunnya dipetakan ke 1 Januari
// tahun tersebut dan ditandai pada labelnya.
var CoretaxTimeline = []CoretaxMilestone{
	{Date: "2018-01-01", Phase: "1", Kind: "regulasi", SourceIdx: []int{3},
		Label: Text{ID: "Perpres 40/2018 menetapkan program PSIAP (tahun saja)", EN: "Presidential Regulation 40/2018 establishes the PSIAP programme (year only)"}},
	{Date: "2020-11-01", Phase: "2", Kind: "pelaksanaan", SourceIdx: []int{2},
		Label: Text{ID: "Tim pelaksana PSIAP 169 orang mulai bertugas", EN: "The 169-person PSIAP implementation team starts work"}},
	{Date: "2020-12-01", Phase: "2", Kind: "pengadaan", SourceIdx: []int{1},
		Label: Text{ID: "KMK 549/KMK.03/2020 menetapkan konsorsium LG CNS-Qualysoft", EN: "Decree 549/KMK.03/2020 names the LG CNS-Qualysoft consortium"}},
	{Date: "2025-01-01", Phase: "4", Kind: "cutover", SourceIdx: []int{0, 9},
		Label: Text{ID: "Go-live serentak menggantikan sistem 2002", EN: "Big-bang go-live replacing the 2002 system"}},
	{Date: "2025-01-31", Phase: "5", Kind: "dampak", SourceIdx: []int{0, 4},
		Label: Text{ID: "Penerimaan perpajakan Januari turun 34,5%; potensi hilang Rp 64 triliun", EN: "January tax revenue down 34.5%; up to IDR 64 trillion potentially lost"}},
	{Date: "2025-02-15", Phase: "5", Kind: "dampak", SourceIdx: []int{6},
		Label: Text{ID: "Tenggat unggah faktur pajak tetap berlaku meski sistem bermasalah", EN: "Invoice upload deadline holds despite the system still failing"}},
	{Date: "2025-03-31", Phase: "5", Kind: "dampak", SourceIdx: []int{5},
		Label: Text{ID: "Penerimaan kuartal I hanya 14,7% dari target setahun", EN: "Q1 revenue reaches only 14.7% of the annual target"}},
	{Date: "2025-07-31", Phase: "5", Kind: "remediasi", SourceIdx: []int{7},
		Label: Text{ID: "Tenggat perbaikan diundur dari Mei ke 31 Juli", EN: "Remediation deadline pushed from May to 31 July"}},
	{Date: "2026-01-20", Phase: "5", Kind: "remediasi", SourceIdx: []int{10, 8},
		Label: Text{ID: "Satu tahun berjalan, DJP diminta evaluasi menyeluruh", EN: "One year on, DJP is urged to run a full evaluation"}},
}

// CoretaxLesson memetakan satu temuan Coretax ke praktik MPPL yang relevan dan
// ke cerminannya pada proyek SIATS - proyek mahasiswa di situs ini.
type CoretaxLesson struct {
	Key      string
	Title    Text
	Finding  Text
	Practice Text // praktik MPPL yang seharusnya menangkap masalah ini
	Mirror   Text // gejala serupa pada proyek SIATS
	Route    string
	Severity string // "kritis", "tinggi", "sedang"
}

// CoretaxLessons adalah jembatan antara kasus nasional dan proyek kuliah.
// Inilah alasan studi kasus ini ada di sini: pola kegagalannya identik, hanya
// skalanya yang berbeda sekitar delapan puluh ribu kali lipat.
var CoretaxLessons = []CoretaxLesson{
	{
		Key: "cutover", Severity: "kritis", Route: "/risiko/",
		Title:    Text{ID: "Cutover serentak tanpa jalur mundur", EN: "Big-bang cutover with no way back"},
		Finding:  Text{ID: "Sistem yang dipakai sejak 2002 diganti sekaligus untuk seluruh wajib pajak pada satu tanggal. Ketika yang baru gagal, tidak ada sistem lama yang bisa dihidupkan kembali.", EN: "A system in use since 2002 was replaced for every taxpayer on a single date. When the new one failed, there was no old system left to fall back on."},
		Practice: Text{ID: "Strategi transisi adalah keputusan manajemen risiko, bukan keputusan teknis. Pilihannya - big bang, paralel, bertahap per segmen, atau pilot - harus dinilai dengan EMV, bukan dengan keyakinan.", EN: "Transition strategy is a risk management decision, not a technical one. Big bang, parallel run, phased, or pilot must be judged by expected monetary value, not by confidence."},
		Mirror:   Text{ID: "Aktivitas A34 pada proyek SIATS juga merencanakan go-live serentak untuk seluruh alumni tanpa periode paralel dengan formulir manual.", EN: "Activity A34 in the SIATS project likewise plans a simultaneous go-live for all alumni with no parallel period alongside the manual form."},
	},
	{
		Key: "paparan", Severity: "kritis", Route: "/biaya/",
		Title:    Text{ID: "Paparan risiko jauh lebih besar daripada nilai proyek", EN: "Risk exposure dwarfs the project value"},
		Finding:  Text{ID: "Potensi penerimaan yang hilang dalam satu bulan saja tercatat Rp 64 triliun, sementara seluruh biaya membangun sistemnya Rp 1,34 triliun. Uang dibelanjakan untuk membangun, nyaris tidak ada yang dibelanjakan untuk mengamankan peralihannya.", EN: "Potential revenue lost in a single month was recorded at IDR 64 trillion, while building the entire system cost IDR 1.34 trillion. The money went into building; almost none went into de-risking the transition."},
		Practice: Text{ID: "Anggaran proyek bukan ukuran kerugian bila proyek gagal. Yang harus dihitung adalah nilai proses bisnis yang bergantung padanya, lalu cadangan ditetapkan terhadap angka itu.", EN: "The project budget is not the measure of loss when a project fails. What must be counted is the value of the business process depending on it, and reserves sized against that."},
		Mirror:   Text{ID: "Proyek SIATS menanggung data akreditasi BAN-PT. Nilai yang dipertaruhkan bukan Rp 16 juta biaya proyek, melainkan peringkat akreditasi program studi.", EN: "SIATS carries BAN-PT accreditation data. What is at stake is not the IDR 16 million project cost but the study programme's accreditation rating."},
	},
	{
		Key: "tenggat", Severity: "tinggi", Route: "/jadwal/",
		Title:    Text{ID: "Tanggal ditetapkan lebih dulu, kesiapan menyusul", EN: "The date came first, readiness came later"},
		Finding:  Text{ID: "Tanggal 1 Januari dipilih karena batas tahun pajak, bukan karena sistemnya siap. Tenggat unggah faktur 15 Februari pun tidak diundur meskipun sistem masih gagal - beban kegagalan dipindahkan ke wajib pajak.", EN: "1 January was chosen because it is the tax-year boundary, not because the system was ready. The 15 February invoice deadline was not moved either, shifting the burden of failure onto taxpayers."},
		Practice: Text{ID: "Tanggal yang tidak bisa digeser harus dipasangkan dengan lingkup yang bisa digeser. Kalau tanggal dan lingkup sama-sama dikunci, yang tersisa untuk mengalah hanyalah mutu.", EN: "An immovable date must be paired with movable scope. If date and scope are both locked, the only thing left to give is quality."},
		Mirror:   Text{ID: "Proyek SIATS mengunci 17 minggu dan seluruh daftar fitur sekaligus. Simulasi Monte Carlo di situs ini memberi peluang hanya 1,4% untuk memenuhi keduanya.", EN: "SIATS locks 17 weeks and the entire feature list at once. The Monte Carlo simulation on this site puts the odds of meeting both at just 1.4%."},
	},
	{
		Key: "cots", Severity: "tinggi", Route: "/wbs/",
		Title:    Text{ID: "Produk jadi tetap butuh pekerjaan penyesuaian yang besar", EN: "An off-the-shelf product still needs enormous adaptation"},
		Finding:  Text{ID: "Coretax memakai pendekatan commercial off-the-shelf yang sudah dipakai banyak negara. Pendekatan itu memindahkan risiko dari pembuatan ke penyesuaian, dan penyesuaian dengan aturan perpajakan Indonesia bukan pekerjaan kecil.", EN: "Coretax used a commercial off-the-shelf approach already deployed in many countries. That shifts risk from building to adapting, and adapting to Indonesian tax rules is no small job."},
		Practice: Text{ID: "Pada proyek COTS, WBS harus memuat paket kerja konfigurasi, migrasi data, dan integrasi secara eksplisit dengan durasi sendiri - bukan diasumsikan ikut terbawa lisensi.", EN: "In a COTS project the WBS must carry configuration, data migration, and integration as explicit work packages with their own durations - not assume they come with the licence."},
		Mirror:   Text{ID: "Paket kerja 3.4 pada SIATS menyatukan integrasi SIAKAD dan migrasi data alumni dalam empat hari kerja. Simulasi menempatkan aktivitas itu di peringkat sensitivitas tertinggi kedua.", EN: "Work package 3.4 in SIATS folds SIAKAD integration and alumni data migration into four working days. The simulation ranks that activity second highest for sensitivity."},
	},
	{
		Key: "durasi", Severity: "sedang", Route: "/pert/",
		Title:    Text{ID: "Proyek berdurasi bertahun-tahun kehilangan umpan balik", EN: "Multi-year projects lose their feedback loop"},
		Finding:  Text{ID: "Dari penetapan pemenang tender sampai go-live berjarak sekitar empat tahun satu bulan. Dalam rentang selama itu, pengguna baru bertemu sistemnya pada hari pertama sistem itu wajib dipakai.", EN: "From tender award to go-live took roughly four years and one month. Over that span, users first met the system on the very day it became mandatory."},
		Practice: Text{ID: "Panjang jalur kritis berbanding lurus dengan ketidakpastian. Rilis bertahap memperpendek jarak antara keputusan dan koreksinya.", EN: "Critical path length scales with uncertainty. Incremental releases shorten the distance between a decision and its correction."},
		Mirror:   Text{ID: "Jalur kritis SIATS berisi 29 dari 41 simpul. Nyaris tidak ada float yang tersisa untuk menyerap kejutan.", EN: "The SIATS critical path holds 29 of 41 nodes. There is almost no float left to absorb surprises."},
	},
	{
		Key: "mutu", Severity: "tinggi", Route: "/kualitas/",
		Title:    Text{ID: "Gejala yang dilaporkan semuanya bisa ditangkap sebelum rilis", EN: "Every reported symptom was catchable before release"},
		Finding:  Text{ID: "Gagal login, halaman kosong, respons lambat, dan gagal membuat faktur adalah kelas cacat yang ditemukan uji beban dan uji penerimaan - bukan cacat yang hanya muncul di produksi.", EN: "Login failures, blank pages, slow responses, and failed invoice creation are defect classes that load testing and acceptance testing find - not defects that surface only in production."},
		Practice: Text{ID: "Biaya kualitas: setiap rupiah pencegahan dan penilaian sebelum rilis menggantikan berlipat-lipat rupiah kegagalan eksternal sesudahnya.", EN: "Cost of quality: every rupiah of prevention and appraisal before release replaces many rupiah of external failure after it."},
		Mirror:   Text{ID: "Cakupan uji otomatis SIATS baru 41% terhadap target 80%, dan 11 cacat kritis masih terbuka pada tanggal data.", EN: "SIATS automated test coverage sits at 41% against an 80% target, with 11 critical defects still open at the data date."},
	},
}
