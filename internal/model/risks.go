package model

// Response adalah strategi penanganan risiko menurut PMBOK.
type Response string

// Strategi untuk ancaman (negative risk).
const (
	ResponseAvoid    Response = "avoid"    // hindari - ubah rencana agar risikonya hilang
	ResponseMitigate Response = "mitigate" // kurangi peluang atau dampak
	ResponseTransfer Response = "transfer" // alihkan ke pihak lain (asuransi, vendor)
	ResponseAccept   Response = "accept"   // terima, siapkan cadangan
)

// Risk adalah satu entri pada risk register.
//
// Probability dan Impact adalah nilai SEBELUM respons dijalankan (inherent
// risk); ResidualProb dan ResidualImpact adalah nilai SESUDAH respons
// (residual risk). Cadangan kontinjensi seharusnya dihitung dari risiko
// residual, bukan dari risiko inheren - kalau tidak, cadangannya akan
// membengkak dan anggaran terkunci sia-sia.
type Risk struct {
	ID             string
	Category       Text
	Title          Text
	Cause          Text
	Effect         Text
	Probability    float64 // 0..1
	Impact         float64 // rupiah
	ScheduleImpact int     // hari kerja tambahan bila risiko terjadi
	ResidualProb   float64
	ResidualImpact float64
	Response       Response
	Mitigation     Text
	Trigger        Text
	Owner          Role
	// WBS menandai paket kerja yang paling terpapar; kosong berarti lintas fase.
	WBS    string
	Status string // "terbuka", "terpantau", "terjadi", "tertutup"
}

// EMV mengembalikan Expected Monetary Value risiko inheren.
func (r Risk) EMV() float64 { return r.Probability * r.Impact }

// ResidualEMV mengembalikan EMV setelah respons dijalankan.
func (r Risk) ResidualEMV() float64 { return r.ResidualProb * r.ResidualImpact }

// Risks adalah risk register proyek. Enam risiko pertama berasal langsung dari
// Project Charter (Tugas 6); sisanya adalah risiko yang muncul saat jaringan
// aktivitas didekomposisi ke level 3 dan ketergantungan nyata terlihat.
var Risks = []Risk{
	{
		ID: "R01", Owner: RolePM, Status: "terbuka", WBS: "5.1",
		Category:    Text{ID: "Eksternal - adopsi pengguna", EN: "External - user adoption"},
		Title:       Text{ID: "Alumni enggan mengisi kuesioner tracer study", EN: "Alumni reluctant to complete the tracer study"},
		Cause:       Text{ID: "Tidak ada insentif langsung dan kontak alumni banyak yang kedaluwarsa", EN: "No direct incentive and many alumni contacts are stale"},
		Effect:      Text{ID: "Target partisipasi 70% tidak tercapai sehingga data tidak layak untuk akreditasi", EN: "The 70% participation target is missed, leaving data unusable for accreditation"},
		Probability: 0.50, Impact: 2_500_000, ScheduleImpact: 0,
		ResidualProb: 0.30, ResidualImpact: 1_500_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Sosialisasi berjenjang lewat grup alumni, pengingat berkala, dan gamifikasi sederhana (sertifikat digital)", EN: "Tiered outreach through alumni groups, periodic reminders, and light gamification (digital certificate)"},
		Trigger:    Text{ID: "Partisipasi di bawah 25% pada dua minggu pertama pasca go-live", EN: "Participation below 25% in the first two weeks after go-live"},
	},
	{
		ID: "R02", Owner: RoleTL, Status: "terjadi", WBS: "3.2",
		Category:    Text{ID: "Jadwal", EN: "Schedule"},
		Title:       Text{ID: "Keterlambatan pengembangan modul inti", EN: "Core module development runs late"},
		Cause:       Text{ID: "Estimasi durasi dibuat tanpa melibatkan pengembang yang mengerjakan", EN: "Durations were estimated without the developers who do the work"},
		Effect:      Text{ID: "Fase pengujian terjepit dan tanggal go-live mundur", EN: "The testing phase is squeezed and go-live slips"},
		Probability: 0.55, Impact: 1_800_000, ScheduleImpact: 8,
		ResidualProb: 0.40, ResidualImpact: 1_200_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Sprint dua mingguan dengan demo wajib, buffer di jalur kritis, dan eskalasi bila SPI < 0,9", EN: "Two-week sprints with mandatory demos, critical-path buffer, escalate when SPI < 0.9"},
		Trigger:    Text{ID: "SPI turun di bawah 0,90 pada laporan mingguan", EN: "SPI drops below 0.90 in the weekly report"},
	},
	{
		ID: "R03", Owner: RoleQA, Status: "terpantau", WBS: "4.1",
		Category:    Text{ID: "Teknis - kualitas", EN: "Technical - quality"},
		Title:       Text{ID: "Bug kritis lolos ke lingkungan produksi", EN: "Critical bugs reach production"},
		Cause:       Text{ID: "Cakupan pengujian otomatis rendah dan regression testing manual", EN: "Low automated test coverage and manual regression testing"},
		Effect:      Text{ID: "Kuesioner gagal tersimpan, kepercayaan alumni turun, rework mahal", EN: "Questionnaire submissions fail, alumni trust drops, rework is costly"},
		Probability: 0.40, Impact: 1_200_000, ScheduleImpact: 4,
		ResidualProb: 0.20, ResidualImpact: 800_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Gerbang mutu di CI: unit test wajib untuk modul kuesioner, code review dua mata, smoke test pasca deploy", EN: "CI quality gate: mandatory unit tests for the questionnaire module, two-eyes code review, post-deploy smoke test"},
		Trigger:    Text{ID: "Defect density melewati 2 temuan per modul pada uji sistem", EN: "Defect density exceeds 2 findings per module in system testing"},
	},
	{
		ID: "R04", Owner: RoleTL, Status: "terbuka", WBS: "3.5",
		Category:    Text{ID: "Keamanan & kepatuhan", EN: "Security & compliance"},
		Title:       Text{ID: "Kebocoran data pribadi alumni", EN: "Alumni personal data breach"},
		Cause:       Text{ID: "Data alumni memuat NIK, nomor telepon, dan riwayat gaji tanpa enkripsi at-rest", EN: "Alumni records hold national ID, phone numbers, and salary history without encryption at rest"},
		Effect:      Text{ID: "Sanksi UU PDP, kerugian reputasi kampus, proyek dihentikan", EN: "Personal-data-law sanctions, reputational damage, project halted"},
		Probability: 0.15, Impact: 6_000_000, ScheduleImpact: 10,
		ResidualProb: 0.07, ResidualImpact: 3_500_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Enkripsi at-rest untuk kolom sensitif, audit log akses, uji penetrasi internal sebelum go-live, minimisasi data yang dikumpulkan", EN: "Encrypt sensitive columns at rest, audit access logs, internal penetration test before go-live, minimise collected data"},
		Trigger:    Text{ID: "Temuan uji penetrasi berkategori tinggi atau akses tak lazim pada audit log", EN: "High-severity penetration-test finding or unusual access in the audit log"},
	},
	{
		ID: "R05", Owner: RolePM, Status: "terpantau", WBS: "",
		Category:    Text{ID: "Anggaran", EN: "Budget"},
		Title:       Text{ID: "Pagu anggaran kampus tidak mencukupi sampai akhir", EN: "The campus budget cap runs out before completion"},
		Cause:       Text{ID: "Estimasi bottom-up sudah menyerap hampir seluruh pagu sejak awal", EN: "The bottom-up estimate already absorbs nearly the entire cap from the outset"},
		Effect:      Text{ID: "Fitur dipangkas atau proyek berhenti sebelum go-live", EN: "Features are cut or the project stops before go-live"},
		Probability: 0.35, Impact: 2_000_000, ScheduleImpact: 5,
		ResidualProb: 0.25, ResidualImpact: 1_400_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Prioritaskan fitur MoSCoW, implementasi bertahap, laporan CPI mingguan ke sponsor sejak minggu pertama", EN: "MoSCoW feature prioritisation, phased implementation, weekly CPI reporting to the sponsor from week one"},
		Trigger:    Text{ID: "CPI di bawah 0,95 atau EAC melewati Rp 15,5 juta", EN: "CPI below 0.95 or EAC above IDR 15.5 million"},
	},
	{
		ID: "R06", Owner: RoleBA, Status: "terbuka", WBS: "1.3",
		Category:    Text{ID: "Ruang lingkup", EN: "Scope"},
		Title:       Text{ID: "Perubahan kebutuhan di tengah pengembangan (scope creep)", EN: "Mid-development requirement changes (scope creep)"},
		Cause:       Text{ID: "Standar instrumen tracer study BAN-PT ditafsirkan berbeda antar bagian", EN: "BAN-PT tracer study instrument standards are read differently by each unit"},
		Effect:      Text{ID: "Pekerjaan ulang pada modul kuesioner dan laporan, jadwal molor", EN: "Rework on the questionnaire and reporting modules, schedule slips"},
		Probability: 0.45, Impact: 1_500_000, ScheduleImpact: 6,
		ResidualProb: 0.30, ResidualImpact: 900_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Baseline SRS ditandatangani, semua perubahan lewat change request dengan dampak biaya dan jadwal yang dihitung", EN: "Sign off the SRS baseline; every change goes through a change request with computed cost and schedule impact"},
		Trigger:    Text{ID: "Lebih dari dua change request dalam satu sprint", EN: "More than two change requests in a single sprint"},
	},
	{
		ID: "R07", Owner: RoleSA, Status: "terbuka", WBS: "3.4",
		Category:    Text{ID: "Teknis - integrasi", EN: "Technical - integration"},
		Title:       Text{ID: "Integrasi SIAKAD gagal karena API tidak terdokumentasi", EN: "SIAKAD integration fails due to undocumented APIs"},
		Cause:       Text{ID: "Sistem akademik dikelola vendor lain dan belum ada kesepakatan akses data", EN: "The academic system is run by another vendor with no data-access agreement yet"},
		Effect:      Text{ID: "Data alumni harus diunggah manual, nilai jual utama sistem hilang", EN: "Alumni data must be uploaded manually, removing the system's core value"},
		Probability: 0.35, Impact: 2_200_000, ScheduleImpact: 7,
		ResidualProb: 0.20, ResidualImpact: 1_200_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Kesepakatan akses data ditandatangani sebelum fase desain berakhir; siapkan jalur cadangan impor CSV terjadwal", EN: "Sign the data-access agreement before design closes; prepare a scheduled CSV import fallback"},
		Trigger:    Text{ID: "Belum ada dokumen API dari vendor pada akhir minggu ke-6", EN: "No vendor API document by the end of week 6"},
	},
	{
		ID: "R08", Owner: RolePM, Status: "terjadi", WBS: "",
		Category:    Text{ID: "Sumber daya manusia", EN: "Human resources"},
		Title:       Text{ID: "Anggota tim tidak tersedia saat periode ujian kampus", EN: "Team members unavailable during campus exam period"},
		Cause:       Text{ID: "Seluruh tim adalah mahasiswa aktif dengan jadwal UAS di tengah proyek", EN: "The whole team are active students with final exams mid-project"},
		Effect:      Text{ID: "Kapasitas turun drastis selama dua minggu, jalur kritis tertahan", EN: "Capacity drops sharply for two weeks and the critical path stalls"},
		Probability: 0.50, Impact: 1_400_000, ScheduleImpact: 6,
		ResidualProb: 0.35, ResidualImpact: 900_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Jadwalkan pekerjaan jalur kritis di luar periode ujian; siapkan pasangan kerja untuk setiap peran tunggal", EN: "Schedule critical-path work outside the exam window; pair up every single-person role"},
		Trigger:    Text{ID: "Kalender akademik menunjukkan UAS beririsan dengan fase pengembangan", EN: "The academic calendar shows exams overlapping the development phase"},
	},
	{
		ID: "R09", Owner: RoleDBA, Status: "terbuka", WBS: "3.4",
		Category:    Text{ID: "Data", EN: "Data"},
		Title:       Text{ID: "Data alumni eksisting kotor, duplikat, atau tidak lengkap", EN: "Existing alumni data is dirty, duplicated, or incomplete"},
		Cause:       Text{ID: "Data terkumpul bertahun-tahun dalam berkas spreadsheet tanpa kunci unik", EN: "Data accumulated for years in spreadsheets with no unique key"},
		Effect:      Text{ID: "Migrasi menghasilkan akun ganda dan statistik tracer study bias", EN: "Migration produces duplicate accounts and biased tracer study statistics"},
		Probability: 0.40, Impact: 900_000, ScheduleImpact: 3,
		ResidualProb: 0.25, ResidualImpact: 600_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Profiling data sebelum migrasi, aturan deduplikasi berbasis NIM, dan migrasi percobaan di lingkungan staging", EN: "Profile data before migration, NIM-based deduplication rules, and a trial migration on staging"},
		Trigger:    Text{ID: "Lebih dari 5% baris gagal validasi pada migrasi percobaan", EN: "More than 5% of rows fail validation in the trial migration"},
	},
	{
		ID: "R10", Owner: RoleOPS, Status: "terpantau", WBS: "5.1",
		Category:    Text{ID: "Pengadaan", EN: "Procurement"},
		Title:       Text{ID: "Pengadaan hosting dan domain terlambat", EN: "Hosting and domain procurement runs late"},
		Cause:       Text{ID: "Proses pengadaan kampus butuh persetujuan berjenjang", EN: "Campus procurement requires multi-level approval"},
		Effect:      Text{ID: "Deployment produksi tertunda padahal sistem sudah siap", EN: "Production deployment waits even though the system is ready"},
		Probability: 0.20, Impact: 700_000, ScheduleImpact: 5,
		ResidualProb: 0.10, ResidualImpact: 400_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Ajukan pengadaan pada minggu ke-8, bukan menjelang deployment; siapkan hosting sementara bila persetujuan molor", EN: "Raise procurement in week 8 rather than just before deployment; keep interim hosting ready if approval slips"},
		Trigger:    Text{ID: "Belum ada nomor pesanan pada akhir minggu ke-12", EN: "No purchase order by the end of week 12"},
	},
	{
		ID: "R11", Owner: RoleBA, Status: "terbuka", WBS: "4.1",
		Category:    Text{ID: "Pemangku kepentingan", EN: "Stakeholder"},
		Title:       Text{ID: "Pemangku kepentingan sulit dijadwalkan untuk UAT", EN: "Stakeholders hard to schedule for UAT"},
		Cause:       Text{ID: "Bagian akademik sedang sibuk periode pendaftaran dan wisuda", EN: "The academic office is busy with enrolment and graduation"},
		Effect:      Text{ID: "UAT mundur dan menahan seluruh fase implementasi", EN: "UAT slips and holds up the entire implementation phase"},
		Probability: 0.35, Impact: 800_000, ScheduleImpact: 4,
		ResidualProb: 0.20, ResidualImpact: 500_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Kunci slot UAT di kalender sejak minggu pertama dan tunjuk pengganti resmi untuk setiap peran penguji", EN: "Lock UAT slots in the calendar from week one and name an official deputy for each tester role"},
		Trigger:    Text{ID: "Undangan UAT belum dikonfirmasi tujuh hari sebelum jadwal", EN: "UAT invitation unconfirmed seven days before the slot"},
	},
	{
		ID: "R12", Owner: RoleDBA, Status: "terbuka", WBS: "5.2",
		Category:    Text{ID: "Kelangsungan layanan", EN: "Service continuity"},
		Title:       Text{ID: "Kehilangan data karena prosedur pemulihan belum pernah diuji", EN: "Data loss because the recovery procedure has never been tested"},
		Cause:       Text{ID: "Backup dijadwalkan tetapi restore tidak pernah dicoba", EN: "Backups are scheduled but restores are never rehearsed"},
		Effect:      Text{ID: "Data tracer study hilang permanen menjelang akreditasi", EN: "Tracer study data is permanently lost right before accreditation"},
		Probability: 0.10, Impact: 4_000_000, ScheduleImpact: 12,
		ResidualProb: 0.04, ResidualImpact: 2_000_000,
		Response:   ResponseMitigate,
		Mitigation: Text{ID: "Latihan restore terjadwal setiap bulan dengan bukti tertulis; simpan salinan di lokasi terpisah", EN: "Monthly rehearsed restores with written evidence; keep an off-site copy"},
		Trigger:    Text{ID: "Tidak ada bukti restore berhasil dalam 30 hari terakhir", EN: "No evidence of a successful restore in the last 30 days"},
	},
}
