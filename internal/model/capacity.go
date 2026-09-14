package model

// Berkas ini memodelkan dua hal yang CPM abaikan sama sekali: kapan orang
// benar-benar bisa bekerja, dan berapa ongkos mempercepat pekerjaan.

// AvailabilityWindow adalah rentang tanggal ketika kapasitas sebagian peran
// turun di bawah normal.
//
// Seluruh tim proyek ini adalah mahasiswa aktif Universitas Esa Unggul
// (Project Charter, bagian batasan). Periode ujian adalah contoh paling nyata:
// orangnya ada, tetapi waktunya tidak. Risiko R08 pada register menyebutnya,
// dan di sini risiko itu diubah dari kalimat menjadi kapasitas yang bisa
// dihitung.
type AvailabilityWindow struct {
	Key    string
	From   string // YYYY-MM-DD, inklusif
	To     string // YYYY-MM-DD, inklusif
	Roles  []Role
	Factor float64 // porsi kapasitas normal yang tersisa, 0..1
	Label  Text
	// Asumsi menandai TANGGAL yang tidak diambil dari kalender akademik resmi.
	// Faktor kapasitasnya selalu asumsi (ExamCapacityFactor) dan dinyatakan
	// terpisah di halaman Metode.
	Asumsi bool
	Source string // dokumen resmi asal tanggal, kosong bila Asumsi
	// RiskID menyambungkan jendela ini ke risiko yang dimodelkannya, supaya
	// simulasi terpadu tidak menghitung dampak jadwal risiko itu dua kali.
	RiskID string
}

// StudentRoles adalah seluruh peran yang diisi mahasiswa.
var StudentRoles = []Role{RolePM, RoleBA, RoleSA, RoleTL, RoleBE, RoleFE, RoleDBA, RoleUX, RoleQA, RoleOPS}

// ExamCapacityFactor adalah porsi kapasitas yang tersisa selama periode ujian.
// Nilainya ASUMSI perencanaan. Halaman Prakiraan Berjalan membandingkannya
// dengan laju kerja nyata tim selama UTS yang sudah lewat, lalu memperbaruinya.
const ExamCapacityFactor = 0.4

// AcademicCalendarURL adalah kalender akademik resmi Universitas Esa Unggul
// TA 2025/2026 (SK Rektor No. 039/SK-R/UEU/III/2025, 24 Maret 2025).
const AcademicCalendarURL = "https://www.esaunggul.ac.id/en/kalender-akademik-tahun-akademik-2025-2026/"

// AvailabilityWindows adalah kalender ketersediaan tim, terurut menurut tanggal.
var AvailabilityWindows = []AvailabilityWindow{
	{
		Key: "uts-ganjil", From: "2025-11-03", To: "2025-11-15", Roles: StudentRoles, Factor: ExamCapacityFactor,
		Source: AcademicCalendarURL, RiskID: "R08",
		Label: Text{
			ID: "Ujian Tengah Semester ganjil (kalender akademik resmi)",
			EN: "Odd-semester midterm exams (official academic calendar)",
		},
	},
	{
		Key: "uas-ganjil", From: "2026-01-12", To: "2026-01-23", Roles: StudentRoles, Factor: ExamCapacityFactor,
		Asumsi: true, RiskID: "R08",
		Label: Text{
			ID: "Ujian Akhir Semester ganjil - tanggal belum terbaca dari halaman 2 kalender resmi",
			EN: "Odd-semester final exams - dates not yet read from page 2 of the official calendar",
		},
	},
	{
		Key: "uts-genap", From: "2026-05-18", To: "2026-05-30", Roles: StudentRoles, Factor: ExamCapacityFactor,
		Source: AcademicCalendarURL, RiskID: "R08",
		Label: Text{
			ID: "Ujian Tengah Semester genap (kalender akademik resmi) - hanya tersentuh ekor simulasi",
			EN: "Even-semester midterm exams (official academic calendar) - reached only by the simulation tail",
		},
	},
}

// CapacityOnDate mengembalikan kapasitas sebuah peran pada tanggal ISO tertentu,
// setelah seluruh jendela ketersediaan diterapkan. Jendela yang tumpang tindih
// dikalikan, bukan dijumlahkan: dua gangguan 50% menyisakan 25%, bukan 0%.
func CapacityOnDate(role Role, iso string, base map[Role]float64) float64 {
	return CapacityOnDateWith(role, iso, base, 0)
}

// CapacityOnDateWith sama dengan CapacityOnDate, tetapi faktor setiap jendela
// diganti examFactor bila examFactor > 0. Prakiraan berjalan memakainya untuk
// mengganti asumsi perencanaan dengan faktor hasil kalibrasi dari realisasi.
func CapacityOnDateWith(role Role, iso string, base map[Role]float64, examFactor float64) float64 {
	c := base[role]
	for _, w := range AvailabilityWindows {
		if iso < w.From || iso > w.To {
			continue
		}
		for _, r := range w.Roles {
			if r == role {
				f := w.Factor
				if examFactor > 0 {
					f = examFactor
				}
				c *= f
				break
			}
		}
	}
	return c
}

// CrashPremium adalah tambahan biaya per hari yang dipercepat, sebagai porsi
// dari biaya tenaga kerja harian aktivitas. 0,75 = premi lembur 50% ditambah
// 25% kehilangan efisiensi koordinasi saat orang dipaksa bekerja lebih rapat
// (hukum Brooks dalam bentuk paling ringan). Ini ASUMSI, dan dinyatakan begitu
// di halaman Metode.
const CrashPremium = 0.75

// crashForbidden adalah aktivitas yang durasinya tidak bisa dibeli dengan uang,
// beserta alasannya. Menambah lembur tidak membuat pemangku kepentingan lebih
// cepat menyetujui desain.
var crashForbidden = map[string]Text{
	"A01": {ID: "Bergantung pada jadwal pemangku kepentingan yang diwawancarai", EN: "Depends on the interviewees' calendars"},
	"A13": {ID: "Persetujuan desain bergantung pada ketersediaan pemangku kepentingan", EN: "Design sign-off depends on stakeholder availability"},
	"A30": {ID: "UAT dijalankan pengguna kampus, bukan tim proyek", EN: "UAT is run by campus users, not the project team"},
	"A35": {ID: "Periode pemantauan pasca go-live adalah jendela observasi tetap", EN: "Post go-live monitoring is a fixed observation window"},
}

// CrashPlan adalah batas percepatan satu aktivitas.
type CrashPlan struct {
	Allowed      bool
	CrashDur     int     // durasi terpendek yang masih masuk akal
	SlopePerDay  float64 // tambahan biaya per hari yang dipotong
	MaxDaysSaved int
	Reason       Text // diisi bila Allowed == false
}

// Crash menurunkan batas percepatan sebuah aktivitas dengan aturan yang sama
// untuk semuanya, supaya tidak ada angka yang dipilih-pilih:
//
//	durasi crash = max(O, M - max(1, M/3))
//	slope        = biaya tenaga kerja harian x CrashPremium
//
// Estimasi optimistis O dipakai sebagai lantai: kalau tim sendiri menilai
// pekerjaan itu tidak mungkin selesai lebih cepat dari O dalam kondisi terbaik,
// uang tidak akan mengubahnya.
func (a Activity) Crash(rates map[Role]float64) CrashPlan {
	if a.Milestone || a.Duration <= 1 {
		return CrashPlan{Reason: Text{ID: "Tidak punya durasi yang bisa dipotong", EN: "Has no duration left to cut"}}
	}
	if why, bad := crashForbidden[a.ID]; bad {
		return CrashPlan{Reason: why}
	}
	cut := a.Duration / 3
	if cut < 1 {
		cut = 1
	}
	crash := a.Duration - cut
	if crash < a.Optimistic {
		crash = a.Optimistic
	}
	if crash < 1 {
		crash = 1
	}
	if crash >= a.Duration {
		return CrashPlan{Reason: Text{ID: "Estimasi optimistis sudah sama dengan durasi rencana", EN: "The optimistic estimate already equals the planned duration"}}
	}
	daily := a.LabourCost(rates) / float64(a.Duration)
	return CrashPlan{
		Allowed:      true,
		CrashDur:     crash,
		SlopePerDay:  daily * CrashPremium,
		MaxDaysSaved: a.Duration - crash,
	}
}
