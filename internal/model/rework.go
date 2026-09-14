package model

// ReworkLoop adalah satu putaran pengulangan kerja dalam notasi GERT.
//
// Modul 4 MPPL menyebut GERT (Graphical Evaluation and Review Technique) untuk
// kegiatan yang tidak berurutan lurus, "seperti loop (tes yang harus diulang
// lebih dari sekali)". CPM tidak bisa menggambarnya: jaringan CPM wajib
// asiklis. GERT memberi setiap cabang peluang, sehingga sebuah pemeriksaan
// bisa lulus (keluar dari putaran) atau gagal (kembali memperbaiki lalu
// memeriksa ulang).
//
// Jaringan aktivitas proyek ini sudah memuat SATU kali perbaikan (A31 setelah
// UAT A30, A26 sebelum uji penetrasi A27). ReworkLoop memodelkan putaran
// TAMBAHAN: pemeriksaan ulang setelah perbaikan itu masih gagal.
type ReworkLoop struct {
	ID string
	// Check adalah aktivitas yang diakhiri pemeriksaan lulus/gagal. Hari
	// tambahan setiap putaran dibebankan setelah aktivitas ini, sebelum
	// penerusnya boleh mulai.
	Check string
	// FailProb adalah peluang satu pemeriksaan gagal. Jumlah putaran tambahan
	// N mengikuti sebaran geometrik: P(N = n) = (1 - p) p^n.
	FailProb float64
	// Parts adalah pekerjaan yang diulang pada setiap putaran, sebagai porsi
	// durasi aktivitas asalnya.
	Parts  []ReworkPart
	Label  Text
	Why    Text
	Asumsi bool
}

// ReworkPart adalah porsi sebuah aktivitas yang dikerjakan ulang per putaran.
type ReworkPart struct {
	Activity string
	Fraction float64
}

// ReworkLoops adalah dua pemeriksaan pada proyek ini yang realistis gagal
// lebih dari sekali. Peluangnya ASUMSI yang diturunkan dari data mutu, dan
// halaman Simulasi Terpadu menunjukkan kepekaannya.
var ReworkLoops = []ReworkLoop{
	{
		ID: "G1", Check: "A31", FailProb: 0.30, Asumsi: true,
		Parts: []ReworkPart{{Activity: "A31", Fraction: 0.5}, {Activity: "A30", Fraction: 1.0 / 3}},
		Label: Text{ID: "Regresi setelah perbaikan bug masih gagal", EN: "Regression after bug fixing still fails"},
		Why: Text{
			ID: "Papan defect mencatat 11 cacat kritis terbuka dan cakupan uji otomatis baru 41%. Dengan cakupan serendah itu, satu putaran perbaikan jarang membersihkan semuanya; peluang gagal 30% disamakan dengan asumsi rework pada fast-tracking. Setiap putaran mengulang separuh perbaikan dan sepertiga UAT.",
			EN: "The defect board shows 11 open critical defects and automated test coverage of only 41%. At that coverage one fix round rarely clears everything; the 30% failure chance matches the rework assumption used for fast-tracking. Each round repeats half the fixing and a third of UAT.",
		},
	},
	{
		ID: "G2", Check: "A27", FailProb: 0.25, Asumsi: true,
		Parts: []ReworkPart{{Activity: "A26", Fraction: 1.0 / 3}, {Activity: "A27", Fraction: 0.5}},
		Label: Text{ID: "Uji penetrasi menemukan celah berkategori tinggi", EN: "Penetration test finds a high-severity issue"},
		Why: Text{
			ID: "Metrik mutu menunjukkan 0% kolom sensitif sudah terenkripsi dan A26 belum dimulai. Uji penetrasi pertama atas pengamanan yang baru dibangun lazim menemukan temuan tinggi; peluang 25% dipakai. Setiap putaran mengulang sepertiga implementasi keamanan dan separuh uji penetrasi.",
			EN: "Quality metrics show 0% of sensitive columns encrypted and A26 not yet started. A first penetration test of newly built safeguards commonly finds high-severity issues; a 25% chance is used. Each round repeats a third of the security work and half the penetration test.",
		},
	},
}

// LoopByCheck mengembalikan putaran rework yang pemeriksaannya aktivitas id.
func LoopByCheck(id string) (ReworkLoop, bool) {
	for _, l := range ReworkLoops {
		if l.Check == id {
			return l, true
		}
	}
	return ReworkLoop{}, false
}
