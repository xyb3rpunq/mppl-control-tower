package model

import (
	"fmt"
	"math"
	"strings"
)

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
// TA 2025/2026 (SK Rektor No. 039/SK-R/UEU/III/2025, 24 Maret 2025). Seluruh
// tanggal ujian di bawah dibaca dari lampiran PDF halaman 1 (UTS) dan
// halaman 2 (UAS). UTS dan UAS susulan tidak dimodelkan: hanya diikuti
// mahasiswa yang berhalangan, bukan seluruh tim.
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
		Key: "uas-ganjil", From: "2026-01-19", To: "2026-01-31", Roles: StudentRoles, Factor: ExamCapacityFactor,
		Source: AcademicCalendarURL, RiskID: "R08",
		Label: Text{
			ID: "Ujian Akhir Semester ganjil (kalender akademik resmi)",
			EN: "Odd-semester final exams (official academic calendar)",
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
	{
		Key: "uas-genap", From: "2026-07-20", To: "2026-08-01", Roles: StudentRoles, Factor: ExamCapacityFactor,
		Source: AcademicCalendarURL, RiskID: "R08",
		Label: Text{
			ID: "Ujian Akhir Semester genap (kalender akademik resmi) - hanya tersentuh ekor simulasi",
			EN: "Even-semester final exams (official academic calendar) - reached only by the simulation tail",
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

// Crashing di proyek ini berarti lembur: pekerjaan hari yang dipotong
// dikerjakan sebagai jam lembur pada hari-hari yang tersisa. Ongkosnya tidak
// diasumsikan, melainkan dihitung dari aturan upah lembur PP 35/2021:
//
//	Pasal 26 ayat (1): lembur paling lama 4 jam sehari dan 18 jam seminggu
//	Pasal 31 ayat (1): jam lembur pertama 1,5 x upah sejam; jam berikutnya 2 x
//	Pasal 32:          upah sejam = 1/173 x upah sebulan
//
// 173 jam sebulan sama dengan 8 jam x 21,6 hari kerja, sehingga upah sejam
// setara upah harian dibagi 8. Efek koordinasi (hukum Brooks) tidak
// ditambahkan karena tidak ada data untuk mengukurnya; premi di sini adalah
// premi minimum menurut aturan.
const (
	OvertimeFirstHour  = 1.5
	OvertimeNextHour   = 2.0
	OvertimeMaxDaily   = 4.0
	OvertimeMaxWeekly  = 18.0
	RegularHoursPerDay = 8.0

	// OvertimeRegulationURL adalah salinan PP 35/2021 (JDIH/Hukumonline).
	OvertimeRegulationURL = "https://learning.hukumonline.com/wp-content/uploads/2021/03/Peraturan-Pemerintah-Nomor-35-tahun-2021-Perjanjian-Kerja-Waktu-Tertentu-Alih-Daya-Waktu-Kerja-dan-Waktu-Istirahat-dan-Pemutusan-Hubungan-Kerja.pdf"
)

// OvertimeUnits mengembalikan upah lembur satu hari dalam satuan upah sejam
// untuk h jam lembur: 1,5 untuk jam pertama dan 2 untuk setiap jam berikutnya.
func OvertimeUnits(h float64) float64 {
	if h <= 0 {
		return 0
	}
	first := math.Min(h, 1)
	return OvertimeFirstHour*first + OvertimeNextHour*math.Max(h-1, 0)
}

// OvertimePremium menghitung premi per hari yang dipotong bila daysCut hari
// kerja dipindah menjadi lembur yang dibagi rata ke crashDays hari tersisa -
// pembagian rata adalah yang termurah karena jam pertama paling murah.
//
//	jam lembur per hari h = 8 x daysCut / crashDays
//	tambahan biaya        = upah harian x (crashDays x unit(h) / 8 - daysCut)
//	premi per hari        = tambahan biaya / (upah harian x daysCut)
//
// ok bernilai false bila h melampaui 4 jam sehari atau 18 jam seminggu.
func OvertimePremium(daysCut, crashDays int) (premium, hoursPerDay float64, ok bool) {
	if daysCut <= 0 || crashDays <= 0 {
		return 0, 0, false
	}
	h := RegularHoursPerDay * float64(daysCut) / float64(crashDays)
	weekDays := math.Min(float64(crashDays), 5)
	if h > OvertimeMaxDaily+1e-9 || h*weekDays > OvertimeMaxWeekly+1e-9 {
		return 0, h, false
	}
	extra := float64(crashDays)*OvertimeUnits(h)/RegularHoursPerDay - float64(daysCut)
	return extra / float64(daysCut), h, true
}

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
	CrashDur     int     // durasi terpendek yang masih masuk akal dan sah
	SlopePerDay  float64 // rerata tambahan biaya per hari bila dipotong penuh
	MaxDaysSaved int
	// Marginal[k-1] adalah tambahan biaya hari ke-k yang dipotong. Nilainya
	// tidak pernah menurun: setiap hari tambahan menaikkan jam lembur harian,
	// dan jam di atas jam pertama dibayar 2x, bukan 1,5x.
	Marginal        []float64
	MarginalPremium []float64 // Marginal dibagi upah harian
	Premium         float64   // premi rerata per hari dipotong saat dipotong penuh
	OvertimeHrs     float64   // jam lembur per hari saat dipotong penuh
	Reason          Text      // diisi bila Allowed == false
}

// CostToCut mengembalikan tambahan biaya memotong k hari: jumlah biaya
// marjinal k hari pertama. k di luar rentang dipotong ke batasnya.
func (p CrashPlan) CostToCut(k int) float64 {
	var c float64
	for i := 0; i < k && i < len(p.Marginal); i++ {
		c += p.Marginal[i]
	}
	return c
}

// Crash menurunkan batas percepatan sebuah aktivitas dengan aturan yang sama
// untuk semuanya, supaya tidak ada angka yang dipilih-pilih:
//
//	durasi crash    = max(O, M - max(1, M/3))
//	biaya(k)        = upah harian x k x OvertimePremium(k, M - k)
//	marjinal hari k = biaya(k) - biaya(k-1)
//
// Potongan berhenti pada k terbesar yang masih sah menurut batas lembur; bila
// satu hari pun tidak sah, aktivitas itu tidak bisa dipercepat dengan lembur.
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
	plan := CrashPlan{}
	var prev float64
	for k := 1; k <= a.Duration-crash; k++ {
		premium, hrs, ok := OvertimePremium(k, a.Duration-k)
		if !ok {
			if k == 1 {
				return CrashPlan{OvertimeHrs: hrs, Reason: Text{
					ID: fmt.Sprintf("Memotong %d hari menjadi %d butuh %s jam lembur per hari - melampaui batas 4 jam sehari (PP 35/2021 Pasal 26)", a.Duration, a.Duration-1, strings.ReplaceAll(fmt.Sprintf("%.1f", hrs), ".", ",")),
					EN: fmt.Sprintf("Cutting %d days to %d needs %.1f overtime hours a day - beyond the 4-hour daily limit (Government Regulation 35/2021, Art. 26)", a.Duration, a.Duration-1, hrs),
				}}
			}
			break
		}
		total := daily * premium * float64(k)
		plan.Marginal = append(plan.Marginal, total-prev)
		plan.MarginalPremium = append(plan.MarginalPremium, (total-prev)/daily)
		prev = total
		plan.Premium, plan.OvertimeHrs, plan.MaxDaysSaved = premium, hrs, k
	}
	plan.Allowed = true
	plan.CrashDur = a.Duration - plan.MaxDaysSaved
	plan.SlopePerDay = prev / float64(plan.MaxDaysSaved)
	return plan
}
