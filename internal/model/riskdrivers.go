package model

// Berkas ini mencabut asumsi bahwa kejadian risiko saling bebas.
//
// Risiko pada register jarang benar-benar independen: beberapa di antaranya
// lahir dari sebab yang sama. Kalau bagian akademik sedang sibuk, perubahan
// kebutuhan (R06) DAN sulitnya menjadwalkan UAT (R11) sama-sama lebih mungkin.
// Mengabaikannya membuat ekor kanan simulasi terlalu tipis - rata-ratanya
// tetap, tetapi skenario "semuanya salah bersamaan" tampak lebih langka dari
// kenyataan.
//
// Modelnya kopula Gauss satu faktor per penggerak:
//
//	Y_k = lambda * Z_penggerak + sqrt(1 - lambda^2) * eps_k
//	risiko k terjadi bila Phi(Y_k) > 1 - p_k
//
// Peluang marginal setiap risiko tetap persis p_k, sehingga EMV dan rerata
// biaya tidak berubah; yang berubah hanya seberapa sering risiko datang
// bergerombol. Penggerak yang ditautkan ke sebuah peran memakai faktor kinerja
// laten peran itu - faktor yang sama yang membuat durasi aktivitasnya panjang
// atau pendek - sehingga risiko juga berkorelasi dengan durasi.

// RiskDriver adalah sebab bersama bagi sekelompok risiko.
type RiskDriver struct {
	Key   string
	Label Text
	Why   Text
	// Role, bila diisi, menautkan penggerak ke faktor kinerja laten peran itu
	// pada sampler durasi.
	Role Role
}

// RiskLoading adalah bobot penggerak bersama (lambda) pada kopula risiko.
// Nilainya ASUMSI: 0,6 berarti korelasi laten 0,36 antar-risiko satu
// penggerak. Halaman Simulasi Terpadu menjalankan beberapa nilai lambda untuk
// menunjukkan kepekaannya.
const RiskLoading = 0.6

// RiskDrivers adalah penggerak bersama yang dibaca dari kolom Cause register.
var RiskDrivers = []RiskDriver{
	{
		Key: "kinerja-pengembang", Role: RoleBE,
		Label: Text{ID: "Kinerja tim pengembang", EN: "Development team performance"},
		Why: Text{
			ID: "R02 (modul inti telat) dan R08 (tim tak tersedia saat ujian) sama-sama bersumber dari tim mahasiswa dengan waktu terbatas. Ditautkan ke faktor kinerja Backend Developer, peran kritis hasil levelling.",
			EN: "R02 (core modules late) and R08 (team unavailable during exams) both stem from a student team with limited time. Tied to the performance factor of the Backend Developer, the critical role found by levelling.",
		},
	},
	{
		Key:   "pemangku-akademik",
		Label: Text{ID: "Beban bagian akademik kampus", EN: "Campus academic office workload"},
		Why: Text{
			ID: "R06 (perubahan kebutuhan) dan R11 (UAT sulit dijadwalkan) sama-sama bergantung pada bagian akademik yang sibuk dengan pendaftaran dan wisuda.",
			EN: "R06 (requirement changes) and R11 (UAT hard to schedule) both depend on an academic office busy with enrolment and graduation.",
		},
	},
	{
		Key:   "data-warisan",
		Label: Text{ID: "Sistem & data akademik warisan", EN: "Legacy academic systems & data"},
		Why: Text{
			ID: "R07 (API SIAKAD tak terdokumentasi) dan R09 (data alumni kotor) sama-sama berasal dari data akademik yang dikelola pihak lain bertahun-tahun tanpa standar.",
			EN: "R07 (undocumented SIAKAD API) and R09 (dirty alumni data) both come from academic data run by others for years without standards.",
		},
	},
	{
		Key:   "birokrasi-kampus",
		Label: Text{ID: "Birokrasi & anggaran kampus", EN: "Campus bureaucracy & budget"},
		Why: Text{
			ID: "R05 (pagu tidak cukup) dan R10 (pengadaan hosting telat) sama-sama melewati persetujuan berjenjang administrasi kampus.",
			EN: "R05 (budget cap runs out) and R10 (hosting procurement late) both pass through the campus's multi-level approval chain.",
		},
	},
	{
		Key:   "disiplin-rekayasa",
		Label: Text{ID: "Disiplin rekayasa tim", EN: "Team engineering discipline"},
		Why: Text{
			ID: "R03 (bug kritis lolos), R04 (kebocoran data), dan R12 (restore tak pernah diuji) sama-sama gejala praktik rekayasa yang belum matang - cakupan uji otomatis baru 41%.",
			EN: "R03 (critical bugs escape), R04 (data breach), and R12 (restore never tested) are all symptoms of immature engineering practice - automated test coverage is only 41%.",
		},
	},
}

// riskDriverOf memetakan risiko ke penggeraknya. R01 sengaja tanpa penggerak:
// keengganan alumni mengisi kuesioner adalah perilaku pihak luar yang tidak
// punya sebab bersama dengan risiko lain di register ini.
var riskDriverOf = map[string]string{
	"R02": "kinerja-pengembang", "R08": "kinerja-pengembang",
	"R06": "pemangku-akademik", "R11": "pemangku-akademik",
	"R07": "data-warisan", "R09": "data-warisan",
	"R05": "birokrasi-kampus", "R10": "birokrasi-kampus",
	"R03": "disiplin-rekayasa", "R04": "disiplin-rekayasa", "R12": "disiplin-rekayasa",
}

// DriverOf mengembalikan kunci penggerak sebuah risiko, atau string kosong.
func (r Risk) DriverOf() string { return riskDriverOf[r.ID] }

// DriverByKey mencari penggerak berdasarkan kuncinya.
func DriverByKey(key string) (RiskDriver, bool) {
	for _, d := range RiskDrivers {
		if d.Key == key {
			return d, true
		}
	}
	return RiskDriver{}, false
}
