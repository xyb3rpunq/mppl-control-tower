// Package model berisi sumber tunggal kebenaran untuk data proyek
// Sistem Informasi Alumni & Tracer Study STIE Jayakusuma.
//
// Seluruh angka yang muncul di situs diturunkan dari paket ini. Tidak ada satu
// pun metrik yang ditulis manual di lapisan tampilan - kalau ada nilai yang
// perlu dikoreksi, koreksinya cukup di sini dan seluruh halaman ikut berubah.
//
// Asal data:
//
//	Tugas 2  : penetapan 9 area pengetahuan & deskripsi masalah
//	Tugas 3  : siklus hidup proyek (konsepsi/perencanaan/eksekusi/operasi)
//	Tugas 5  : struktur organisasi & uraian peran
//	Tugas 6  : Project Charter + Work Breakdown Structure
//	Tugas 10 : bagan organisasi, RACI, jalur komunikasi
package model

// Text adalah sepasang teks dwibahasa. Bahasa Indonesia adalah bahasa sumber;
// bahasa Inggris disediakan untuk pembaca luar (mis. penguji akreditasi).
type Text struct {
	ID string
	EN string
}

// Get mengembalikan teks untuk bahasa lang ("id"/"en"), jatuh balik ke ID.
func (t Text) Get(lang string) string {
	if lang == "en" && t.EN != "" {
		return t.EN
	}
	return t.ID
}

// Role adalah kode peran dalam tim proyek.
type Role string

// Daftar peran sesuai struktur organisasi pada Tugas 5 dan Tugas 10.
const (
	RolePM  Role = "PM"
	RoleBA  Role = "BA"
	RoleSA  Role = "SA"
	RoleTL  Role = "TL"
	RoleBE  Role = "BE"
	RoleFE  Role = "FE"
	RoleDBA Role = "DBA"
	RoleUX  Role = "UX"
	RoleQA  Role = "QA"
	RoleOPS Role = "OPS"
)

// Predecessor adalah satu relasi ketergantungan antar-aktivitas.
// Type memakai notasi PMBOK: FS, SS, FF, SF. Lag positif berarti jeda,
// lag negatif berarti lead (tumpang tindih).
type Predecessor struct {
	ID   string
	Type string
	Lag  int
}

// FS membuat relasi Finish-to-Start tanpa lag - relasi paling umum.
func FS(id string) Predecessor { return Predecessor{ID: id, Type: "FS"} }

// TeamSlot adalah penugasan satu peran pada satu aktivitas.
// Alloc adalah porsi hari-orang per hari kerja (1 = penuh waktu).
type TeamSlot struct {
	Role  Role
	Alloc float64
}

// Extra adalah biaya non-tenaga-kerja yang melekat pada satu aktivitas.
// Key menyambung ke kategori anggaran pada Project Charter.
type Extra struct {
	Key    string
	Amount float64
	Label  Text
}

// Actual adalah realisasi pelaksanaan satu aktivitas sampai tanggal data.
// Start adalah indeks hari kerja mulai sesungguhnya, Duration durasi
// sesungguhnya, dan Cost total biaya sesungguhnya bila aktivitas rampung.
// Aktivitas yang belum dimulai memakai Started = false.
type Actual struct {
	Started  bool
	Start    int
	Duration int
	Cost     float64
}

// Activity adalah satu simpul pada jaringan Activity-on-Node.
type Activity struct {
	ID          string
	WBS         string // kode paket kerja induk, mis. "3.2"
	Name        Text
	Duration    int // durasi paling mungkin (M pada PERT), hari kerja
	Optimistic  int
	Pessimistic int
	Pred        []Predecessor
	Team        []TeamSlot
	Extras      []Extra
	Milestone   bool
	Actual      Actual
}

// LabourCost menghitung biaya tenaga kerja rencana dari rate card.
func (a Activity) LabourCost(rates map[Role]float64) float64 {
	var total float64
	for _, slot := range a.Team {
		total += rates[slot.Role] * slot.Alloc * float64(a.Duration)
	}
	return total
}

// ExtraCost menjumlahkan biaya non-tenaga-kerja aktivitas.
func (a Activity) ExtraCost() float64 {
	var total float64
	for _, e := range a.Extras {
		total += e.Amount
	}
	return total
}

// Budget menghitung total anggaran aktivitas (tenaga kerja + non-tenaga-kerja).
// Inilah nilai yang bisa "diperoleh" sebagai Earned Value ketika aktivitas
// rampung 100%.
func (a Activity) Budget(rates map[Role]float64) float64 {
	return a.LabourCost(rates) + a.ExtraCost()
}

// PersonDays mengembalikan total hari-orang rencana untuk aktivitas ini.
func (a Activity) PersonDays() float64 {
	var total float64
	for _, slot := range a.Team {
		total += slot.Alloc * float64(a.Duration)
	}
	return total
}

// WBSPackage adalah paket kerja level 2 pada Work Breakdown Structure.
type WBSPackage struct {
	Code string
	Name Text
}

// WBSPhase adalah fase level 1 pada Work Breakdown Structure.
type WBSPhase struct {
	Code      string
	Name      Text
	Weeks     int
	Milestone Text
	Packages  []WBSPackage
}

// BudgetLine adalah satu baris alokasi anggaran pada Project Charter.
type BudgetLine struct {
	Key    string
	Amount float64
	Label  Text
}

// SuccessCriterion adalah kriteria keberhasilan yang terukur.
// MetricKey menyambungkan kriteria ke metrik yang dihitung engine.
type SuccessCriterion struct {
	MetricKey string
	Target    float64
	Unit      string
	Label     Text
}
