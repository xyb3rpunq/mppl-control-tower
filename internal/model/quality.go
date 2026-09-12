package model

// DefectRecord adalah temuan cacat pada satu modul, dikelompokkan menurut
// keparahan. Sumber angka: catatan pengujian internal tim sampai tanggal data.
type DefectRecord struct {
	Module   Text
	Critical int
	Major    int
	Minor    int
	// ReworkHours adalah jam kerja perbaikan yang sudah tercatat, dipakai
	// untuk menghitung biaya kegagalan internal.
	ReworkHours float64
}

// Total mengembalikan jumlah seluruh cacat pada modul.
func (d DefectRecord) Total() int { return d.Critical + d.Major + d.Minor }

// Weighted memberi bobot keparahan: kritis 5, mayor 3, minor 1. Diagram Pareto
// atas jumlah mentah bisa menyesatkan - sepuluh cacat kosmetik tidak setara
// dengan dua cacat yang membuat kuesioner gagal tersimpan.
func (d DefectRecord) Weighted() int { return d.Critical*5 + d.Major*3 + d.Minor }

// Defects adalah hasil pengujian per modul sampai tanggal data.
var Defects = []DefectRecord{
	{Module: Text{ID: "Modul kuesioner tracer study", EN: "Tracer study questionnaire module"}, Critical: 4, Major: 9, Minor: 12, ReworkHours: 31},
	{Module: Text{ID: "Integrasi SIAKAD & migrasi data", EN: "SIAKAD integration & data migration"}, Critical: 3, Major: 7, Minor: 5, ReworkHours: 24},
	{Module: Text{ID: "Autentikasi & manajemen akun", EN: "Authentication & account management"}, Critical: 2, Major: 5, Minor: 8, ReworkHours: 16},
	{Module: Text{ID: "Dasbor & agregasi laporan", EN: "Dashboard & report aggregation"}, Critical: 1, Major: 6, Minor: 11, ReworkHours: 14},
	{Module: Text{ID: "Panel administrasi", EN: "Administration panel"}, Critical: 0, Major: 4, Minor: 9, ReworkHours: 8},
	{Module: Text{ID: "Ekspor Excel/PDF", EN: "Excel/PDF export"}, Critical: 1, Major: 2, Minor: 6, ReworkHours: 6},
	{Module: Text{ID: "Landing page & registrasi", EN: "Landing page & registration"}, Critical: 0, Major: 1, Minor: 7, ReworkHours: 3},
	{Module: Text{ID: "Notifikasi surel", EN: "Email notification"}, Critical: 0, Major: 1, Minor: 3, ReworkHours: 2},
}

// ResponseSample adalah satu sampel pengukuran waktu respons sistem.
// Setiap sampel adalah rata-rata dari lima pengukuran pada minggu tersebut,
// sesuai praktik subgrup pada control chart.
type ResponseSample struct {
	Week  int
	Mean  float64 // detik
	Range float64 // selisih maksimum-minimum dalam subgrup
	Note  Text
}

// ResponseSamples adalah hasil pemantauan waktu respons halaman kuesioner.
// Standar mutu pada Project Charter: waktu respons di bawah 3 detik.
var ResponseSamples = []ResponseSample{
	{Week: 1, Mean: 1.32, Range: 0.42},
	{Week: 2, Mean: 1.28, Range: 0.35},
	{Week: 3, Mean: 1.41, Range: 0.51},
	{Week: 4, Mean: 1.35, Range: 0.38},
	{Week: 5, Mean: 1.44, Range: 0.47},
	{Week: 6, Mean: 1.39, Range: 0.44},
	{Week: 7, Mean: 1.62, Range: 0.66, Note: Text{ID: "Data uji diperbesar menjadi 5.000 baris alumni", EN: "Test data grown to 5,000 alumni rows"}},
	{Week: 8, Mean: 1.71, Range: 0.72},
	{Week: 9, Mean: 1.83, Range: 0.68},
	{Week: 10, Mean: 1.95, Range: 0.81, Note: Text{ID: "Agregasi dasbor belum memakai indeks", EN: "Dashboard aggregation still unindexed"}},
	{Week: 11, Mean: 2.08, Range: 0.77},
	{Week: 12, Mean: 2.21, Range: 0.85},
}

// COQCategory adalah salah satu dari empat kategori biaya kualitas.
type COQCategory string

// Empat kategori biaya kualitas menurut Juran dan Crosby, seperti dibahas pada
// Pertemuan 9 MPPL.
const (
	COQPrevention      COQCategory = "prevention"
	COQAppraisal       COQCategory = "appraisal"
	COQInternalFailure COQCategory = "internal"
	COQExternalFailure COQCategory = "external"
)

// COQItem adalah satu pos biaya kualitas.
type COQItem struct {
	Category COQCategory
	Label    Text
	Amount   float64
	Actual   bool // true bila sudah terjadi, false bila masih proyeksi
}

// CostOfQuality adalah rincian biaya kualitas proyek.
//
// Angka biaya kegagalan eksternal sengaja diproyeksikan, bukan nol: kalau
// diisi nol sebelum go-live, grafiknya akan memberi kesan keliru bahwa
// pencegahan sudah cukup. Justru perbandingan pencegahan+penilaian terhadap
// kegagalan itulah yang jadi argumen menambah pengujian di muka.
var CostOfQuality = []COQItem{
	{Category: COQPrevention, Amount: 300_000, Actual: true, Label: Text{ID: "Penyusunan standar coding & templat code review", EN: "Coding standards & code review templates"}},
	{Category: COQPrevention, Amount: 420_000, Actual: true, Label: Text{ID: "Pelatihan tim & penyusunan design system", EN: "Team training & design system"}},
	{Category: COQPrevention, Amount: 250_000, Actual: false, Label: Text{ID: "Gerbang mutu otomatis di pipeline CI", EN: "Automated quality gate in the CI pipeline"}},
	{Category: COQAppraisal, Amount: 300_000, Actual: true, Label: Text{ID: "Lisensi tools pengujian & pemantauan", EN: "Testing & monitoring tool licences"}},
	{Category: COQAppraisal, Amount: 480_000, Actual: false, Label: Text{ID: "Unit, integration, dan system testing", EN: "Unit, integration, and system testing"}},
	{Category: COQAppraisal, Amount: 360_000, Actual: false, Label: Text{ID: "User Acceptance Testing bersama bagian akademik", EN: "User Acceptance Testing with the academic office"}},
	{Category: COQAppraisal, Amount: 240_000, Actual: false, Label: Text{ID: "Uji penetrasi internal sebelum go-live", EN: "Internal penetration test before go-live"}},
	{Category: COQInternalFailure, Amount: 640_000, Actual: true, Label: Text{ID: "Rework modul kuesioner & integrasi (104 jam)", EN: "Questionnaire & integration rework (104 hours)"}},
	{Category: COQInternalFailure, Amount: 380_000, Actual: false, Label: Text{ID: "Regression testing ulang pasca bug fixing", EN: "Repeat regression testing after bug fixing"}},
	{Category: COQExternalFailure, Amount: 1_500_000, Actual: false, Label: Text{ID: "Proyeksi penanganan insiden pasca go-live & kampanye ulang partisipasi", EN: "Projected post go-live incident handling & repeat participation campaign"}},
	{Category: COQExternalFailure, Amount: 900_000, Actual: false, Label: Text{ID: "Proyeksi perbaikan data tracer study yang terlanjur bias", EN: "Projected repair of already-biased tracer study data"}},
}

// FishboneCause adalah satu cabang pada diagram sebab-akibat.
type FishboneCause struct {
	Category Text
	Causes   []Text
}

// FishboneProblem adalah satu diagram sebab-akibat lengkap.
type FishboneProblem struct {
	Key      string
	Effect   Text
	Branches []FishboneCause
	// RootCause adalah kesimpulan setelah menelusuri lima lapis "mengapa".
	RootCause Text
}

// Fishbones adalah diagram sebab-akibat untuk dua masalah mutu utama.
// Kategori cabangnya memakai 6M yang lazim dipakai di sektor jasa: Manusia,
// Metode, Mesin, Material, Pengukuran, dan Lingkungan.
var Fishbones = []FishboneProblem{
	{
		Key:    "partisipasi",
		Effect: Text{ID: "Partisipasi alumni di bawah target 70%", EN: "Alumni participation below the 70% target"},
		Branches: []FishboneCause{
			{Category: Text{ID: "Manusia", EN: "People"}, Causes: []Text{
				{ID: "Alumni tidak merasakan manfaat langsung dari mengisi", EN: "Alumni see no direct benefit from responding"},
				{ID: "Tidak ada petugas khusus yang menindaklanjuti alumni pasif", EN: "No dedicated officer follows up passive alumni"},
			}},
			{Category: Text{ID: "Metode", EN: "Method"}, Causes: []Text{
				{ID: "Kampanye hanya sekali saat go-live, tanpa pengingat berkala", EN: "One-off campaign at go-live, no periodic reminders"},
				{ID: "Kuesioner terlalu panjang untuk diselesaikan sekali duduk", EN: "Questionnaire too long to finish in one sitting"},
			}},
			{Category: Text{ID: "Material (data)", EN: "Material (data)"}, Causes: []Text{
				{ID: "Nomor telepon dan surel alumni banyak yang kedaluwarsa", EN: "Many alumni phone numbers and emails are stale"},
				{ID: "Data angkatan lama tidak lengkap sehingga tak bisa dihubungi", EN: "Older cohort records are incomplete and unreachable"},
			}},
			{Category: Text{ID: "Mesin (sistem)", EN: "Machine (system)"}, Causes: []Text{
				{ID: "Formulir berat dibuka di jaringan seluler lambat", EN: "The form is heavy on slow mobile networks"},
				{ID: "Sesi terputus membuat jawaban yang sudah diisi hilang", EN: "Dropped sessions lose answers already entered"},
			}},
			{Category: Text{ID: "Pengukuran", EN: "Measurement"}, Causes: []Text{
				{ID: "Tidak ada pemantauan harian tingkat penyelesaian per langkah", EN: "No daily monitoring of per-step completion rate"},
			}},
			{Category: Text{ID: "Lingkungan", EN: "Environment"}, Causes: []Text{
				{ID: "Alumni sudah jenuh oleh survei kampus lain", EN: "Alumni fatigued by other campus surveys"},
			}},
		},
		RootCause: Text{ID: "Basis kontak alumni tidak pernah dipelihara, sehingga kampanye apa pun hanya menjangkau sebagian kecil populasi sasaran", EN: "The alumni contact base was never maintained, so any campaign reaches only a fraction of the target population"},
	},
	{
		Key:    "bug",
		Effect: Text{ID: "Cacat kritis lolos sampai pengujian sistem", EN: "Critical defects survive to system testing"},
		Branches: []FishboneCause{
			{Category: Text{ID: "Manusia", EN: "People"}, Causes: []Text{
				{ID: "Pengembang tunggal per modul, tidak ada pasangan peninjau", EN: "One developer per module, no review partner"},
				{ID: "Beban kuliah bersamaan dengan sprint pengembangan", EN: "Coursework load overlaps the development sprint"},
			}},
			{Category: Text{ID: "Metode", EN: "Method"}, Causes: []Text{
				{ID: "Pengujian dijadwalkan setelah seluruh modul jadi, bukan menyertai", EN: "Testing scheduled after all modules are built, not alongside"},
				{ID: "Definisi selesai tidak menyertakan uji otomatis", EN: "The definition of done omits automated tests"},
			}},
			{Category: Text{ID: "Mesin (perkakas)", EN: "Machine (tooling)"}, Causes: []Text{
				{ID: "Pipeline CI belum menjalankan uji pada setiap perubahan", EN: "The CI pipeline does not run tests on every change"},
			}},
			{Category: Text{ID: "Material (kode)", EN: "Material (code)"}, Causes: []Text{
				{ID: "Logika kuesioner dinamis tersebar di banyak berkas", EN: "Dynamic questionnaire logic is scattered across many files"},
			}},
			{Category: Text{ID: "Pengukuran", EN: "Measurement"}, Causes: []Text{
				{ID: "Cakupan uji tidak pernah diukur sehingga celahnya tak terlihat", EN: "Test coverage is never measured, so gaps stay invisible"},
			}},
			{Category: Text{ID: "Lingkungan", EN: "Environment"}, Causes: []Text{
				{ID: "Lingkungan staging berbeda konfigurasi dengan produksi", EN: "Staging is configured differently from production"},
			}},
		},
		RootCause: Text{ID: "Pengujian ditempatkan sebagai fase terpisah di akhir WBS, bukan sebagai gerbang mutu di dalam setiap aktivitas pengembangan", EN: "Testing sits as a separate phase at the end of the WBS instead of as a quality gate inside every development activity"},
	},
}

// QualityMetric adalah satu metrik mutu beserta target dan capaiannya.
type QualityMetric struct {
	Name           Text
	Target         float64
	Actual         float64
	Unit           string
	HigherIsBetter bool
	Source         Text
}

// QualityMetrics memetakan standar mutu Project Charter menjadi angka yang
// bisa diperiksa. Standar yang tidak bisa diukur bukan standar - itu harapan.
var QualityMetrics = []QualityMetric{
	{Name: Text{ID: "Waktu respons halaman kuesioner", EN: "Questionnaire page response time"}, Target: 3, Actual: 2.21, Unit: "detik", HigherIsBetter: false, Source: Text{ID: "Rata-rata subgrup minggu ke-12", EN: "Week 12 subgroup mean"}},
	{Name: Text{ID: "Uptime sistem", EN: "System uptime"}, Target: 95, Actual: 99.1, Unit: "%", HigherIsBetter: true, Source: Text{ID: "Pemantauan lingkungan staging", EN: "Staging environment monitoring"}},
	{Name: Text{ID: "Cakupan uji otomatis modul inti", EN: "Core module automated test coverage"}, Target: 80, Actual: 41, Unit: "%", HigherIsBetter: true, Source: Text{ID: "Laporan cakupan pipeline CI", EN: "CI pipeline coverage report"}},
	{Name: Text{ID: "Cacat kritis terbuka", EN: "Open critical defects"}, Target: 0, Actual: 11, Unit: "temuan", HigherIsBetter: false, Source: Text{ID: "Papan defect sampai tanggal data", EN: "Defect board as of the data date"}},
	{Name: Text{ID: "Skor usability (SUS)", EN: "Usability score (SUS)"}, Target: 80, Actual: 76, Unit: "poin", HigherIsBetter: true, Source: Text{ID: "Usability testing prototipe, 8 responden", EN: "Prototype usability testing, 8 respondents"}},
	{Name: Text{ID: "Kolom sensitif terenkripsi at-rest", EN: "Sensitive columns encrypted at rest"}, Target: 100, Actual: 0, Unit: "%", HigherIsBetter: true, Source: Text{ID: "Aktivitas A26 belum dimulai", EN: "Activity A26 not started"}},
	{Name: Text{ID: "Responsif di lebar layar 360 px", EN: "Responsive at 360 px width"}, Target: 100, Actual: 88, Unit: "%", HigherIsBetter: true, Source: Text{ID: "Audit halaman yang sudah jadi", EN: "Audit of completed pages"}},
}
