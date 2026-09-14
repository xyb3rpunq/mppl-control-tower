package model

// Berkas ini memodelkan dua hal yang CPM abaikan sama sekali: kapan orang
// benar-benar bisa bekerja, dan berapa ongkos mempercepat pekerjaan.

// AvailabilityWindow adalah rentang tanggal ketika kapasitas sebagian peran
// turun di bawah normal.
//
// Seluruh tim proyek ini adalah mahasiswa aktif (Project Charter, bagian
// batasan). Periode ujian akhir semester adalah contoh paling nyata: orangnya
// ada, tetapi waktunya tidak. Risiko R08 pada register menyebutnya, dan di sini
// risiko itu diubah dari kalimat menjadi kapasitas yang bisa dihitung.
type AvailabilityWindow struct {
	From   string // YYYY-MM-DD, inklusif
	To     string // YYYY-MM-DD, inklusif
	Roles  []Role
	Factor float64 // porsi kapasitas normal yang tersisa, 0..1
	Label  Text
	// Asumsi menandai tanggal yang tidak diambil dari kalender akademik resmi.
	Asumsi bool
	// RiskID menyambungkan jendela ini ke risiko yang dimodelkannya, supaya
	// simulasi terpadu tidak menghitung dampak jadwal risiko itu dua kali.
	RiskID string
}

// StudentRoles adalah seluruh peran yang diisi mahasiswa.
var StudentRoles = []Role{RolePM, RoleBA, RoleSA, RoleTL, RoleBE, RoleFE, RoleDBA, RoleUX, RoleQA, RoleOPS}

// AvailabilityWindows adalah kalender ketersediaan tim.
var AvailabilityWindows = []AvailabilityWindow{
	{
		From: "2026-01-12", To: "2026-01-23", Roles: StudentRoles, Factor: 0.4, Asumsi: true, RiskID: "R08",
		Label: Text{
			ID: "Ujian Akhir Semester ganjil - kapasitas tim tersisa 40%",
			EN: "Odd-semester final exams - team capacity drops to 40%",
		},
	},
}

// CapacityOnDate mengembalikan kapasitas sebuah peran pada tanggal ISO tertentu,
// setelah seluruh jendela ketersediaan diterapkan. Jendela yang tumpang tindih
// dikalikan, bukan dijumlahkan: dua gangguan 50% menyisakan 25%, bukan 0%.
func CapacityOnDate(role Role, iso string, base map[Role]float64) float64 {
	c := base[role]
	for _, w := range AvailabilityWindows {
		if iso < w.From || iso > w.To {
			continue
		}
		for _, r := range w.Roles {
			if r == role {
				c *= w.Factor
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
