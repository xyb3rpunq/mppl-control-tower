package model

// Charter adalah Project Charter versi terstruktur (asal: Tugas 6).
type Charter struct {
	Name             Text
	ShortName        Text
	Code             string
	Version          string
	CharterDate      string
	StartDate        string
	TargetFinish     string
	DurationWeeks    int
	Sponsor          Text
	Manager          string
	ManagerNIM       string
	Advisor          string
	Course           Text
	Background       Text
	Objectives       []Text
	InScope          []Text
	OutScope         []Text
	Deliverables     []Text
	Assumptions      []Text
	Constraints      []Text
	SuccessCriteria  []SuccessCriterion
	QualityStandards []Text
}

// ProjectCharter adalah piagam proyek yang dipakai seluruh aplikasi.
var ProjectCharter = Charter{
	Name: Text{
		ID: "Pengembangan Sistem Informasi Alumni & Tracer Study STIE Jayakusuma",
		EN: "Alumni Information System & Tracer Study Development, STIE Jayakusuma",
	},
	ShortName:     Text{ID: "SIA Tracer Study", EN: "Alumni Tracer Study IS"},
	Code:          "SIATS-2025",
	Version:       "1.0",
	CharterDate:   "2025-10-21",
	StartDate:     "2025-10-20",
	TargetFinish:  "2026-02-13", // 17 minggu kalender polos, belum menghitung hari libur
	DurationWeeks: 17,
	Sponsor:       Text{ID: "Pihak STIE Jayakusuma", EN: "STIE Jayakusuma Management"},
	Manager:       "Andri Zulfikar",
	ManagerNIM:    "20200801426",
	Advisor:       "Dr., Ir. Gerry Firmansyah, S.T., M.Kom",
	Course: Text{
		ID: "Manajemen Proyek Perangkat Lunak - Universitas Esa Unggul",
		EN: "Software Project Management - Universitas Esa Unggul",
	},
	Background: Text{
		ID: "STIE Jayakusuma belum memiliki aplikasi khusus untuk mengelola data alumni dan tracer study. Pengumpulan data masih manual atau lewat formulir sederhana sehingga lambat, rawan duplikasi, dan sulit diaudit saat akreditasi. Proyek ini membangun sistem informasi berbasis web untuk mendata alumni secara terpusat, mengumpulkan informasi karier, serta menyajikan laporan tracer study yang dibutuhkan BAN-PT.",
		EN: "STIE Jayakusuma has no dedicated application for alumni data and tracer study. Data collection is still manual or via simple forms, making it slow, duplication-prone, and hard to audit during accreditation. This project builds a web-based information system to centralise alumni records, collect career information, and produce the tracer study reports required by BAN-PT.",
	},
	Objectives: []Text{
		{ID: "Digitalisasi pengelolaan data alumni yang sebelumnya manual", EN: "Digitalise alumni data management that was previously manual"},
		{ID: "Menyediakan platform kuesioner daring yang bisa diakses dari mana saja", EN: "Provide an online questionnaire platform accessible from anywhere"},
		{ID: "Menyajikan dasbor laporan untuk manajemen kampus dan akreditasi BAN-PT", EN: "Deliver a reporting dashboard for campus management and BAN-PT accreditation"},
		{ID: "Meningkatkan partisipasi alumni hingga minimal 70%", EN: "Raise alumni participation to at least 70%"},
		{ID: "Menyediakan data dan laporan sesuai standar BAN-PT", EN: "Supply data and reports meeting BAN-PT standards"},
	},
	InScope: []Text{
		{ID: "Modul pendaftaran & autentikasi alumni", EN: "Alumni registration & authentication module"},
		{ID: "Modul kuesioner tracer study daring yang dapat dikonfigurasi", EN: "Configurable online tracer study questionnaire module"},
		{ID: "Dasbor laporan (grafik & statistik)", EN: "Reporting dashboard (charts & statistics)"},
		{ID: "Panel admin untuk pengelolaan sistem oleh kampus", EN: "Admin panel for campus system management"},
		{ID: "Ekspor laporan ke Excel/PDF", EN: "Report export to Excel/PDF"},
		{ID: "Integrasi dengan basis data akademik (SIAKAD)", EN: "Integration with the academic database (SIAKAD)"},
		{ID: "Keamanan & enkripsi data alumni", EN: "Alumni data security & encryption"},
	},
	OutScope: []Text{
		{ID: "Aplikasi mobile native (iOS/Android)", EN: "Native mobile apps (iOS/Android)"},
		{ID: "Integrasi dengan sistem eksternal di luar kampus", EN: "Integration with systems outside the campus"},
		{ID: "Kustomisasi fitur di luar spesifikasi awal tanpa persetujuan", EN: "Feature customisation beyond the initial spec without approval"},
	},
	Deliverables: []Text{
		{ID: "Dokumen Analisis Kebutuhan (BRD & SRS)", EN: "Requirements documents (BRD & SRS)"},
		{ID: "Dokumen Desain Sistem (arsitektur, ERD, wireframe)", EN: "System design documents (architecture, ERD, wireframes)"},
		{ID: "Aplikasi web tracer study yang terintegrasi", EN: "Integrated tracer study web application"},
		{ID: "Dokumentasi teknis (user manual, dokumentasi API)", EN: "Technical documentation (user manual, API docs)"},
		{ID: "Laporan pengujian (unit, integrasi, UAT)", EN: "Test reports (unit, integration, UAT)"},
		{ID: "Materi pelatihan admin & pengguna", EN: "Training materials for admins & users"},
	},
	Assumptions: []Text{
		{ID: "Alumni memiliki akses internet untuk mengisi kuesioner", EN: "Alumni have internet access to complete the questionnaire"},
		{ID: "Kampus menyediakan infrastruktur server yang memadai", EN: "The campus provides adequate server infrastructure"},
		{ID: "Pemangku kepentingan tersedia untuk requirement gathering dan UAT", EN: "Stakeholders are available for requirement gathering and UAT"},
		{ID: "Data alumni eksisting dapat dimigrasikan ke sistem baru", EN: "Existing alumni data can be migrated to the new system"},
	},
	Constraints: []Text{
		{ID: "Anggaran terbatas Rp 16.000.000", EN: "Budget capped at IDR 16,000,000"},
		{ID: "Jadwal maksimal 17 minggu", EN: "Schedule capped at 17 weeks"},
		{ID: "Tim terdiri dari mahasiswa dengan keterbatasan waktu", EN: "The team consists of students with limited availability"},
		{ID: "Teknologi harus kompatibel dengan infrastruktur kampus eksisting", EN: "Technology must be compatible with existing campus infrastructure"},
	},
	SuccessCriteria: []SuccessCriterion{
		{MetricKey: "scope", Target: 100, Unit: "%", Label: Text{ID: "Sistem berfungsi 100% sesuai spesifikasi yang disetujui", EN: "System works 100% to approved specification"}},
		{MetricKey: "participation", Target: 70, Unit: "%", Label: Text{ID: "Minimal 70% alumni mengisi tracer study", EN: "At least 70% of alumni complete the tracer study"}},
		{MetricKey: "uptime", Target: 95, Unit: "%", Label: Text{ID: "Uptime sistem minimal 95%", EN: "System uptime at least 95%"}},
		{MetricKey: "satisfaction", Target: 80, Unit: "%", Label: Text{ID: "Skor kepuasan pengguna di atas 80%", EN: "User satisfaction score above 80%"}},
		{MetricKey: "schedule", Target: 17, Unit: "minggu", Label: Text{ID: "Selesai tepat waktu dalam 17 minggu", EN: "Finished on time within 17 weeks"}},
		{MetricKey: "budget", Target: 16000000, Unit: "Rp", Label: Text{ID: "Selesai sesuai anggaran Rp 16.000.000", EN: "Finished within the IDR 16,000,000 budget"}},
		{MetricKey: "uat", Target: 100, Unit: "%", Label: Text{ID: "Sistem lulus User Acceptance Testing", EN: "System passes User Acceptance Testing"}},
	},
	QualityStandards: []Text{
		{ID: "Usability - antarmuka ramah pengguna & responsif di perangkat mobile", EN: "Usability - friendly interface, responsive on mobile"},
		{ID: "Security - login aman dan data alumni terenkripsi", EN: "Security - secure login and encrypted alumni data"},
		{ID: "Performance - waktu respons di bawah 3 detik, uptime minimal 95%", EN: "Performance - response time under 3 s, uptime at least 95%"},
		{ID: "Functionality - data tracer study dapat diekspor ke Excel/PDF", EN: "Functionality - tracer study data exportable to Excel/PDF"},
		{ID: "Reliability - downtime sistem seminimal mungkin", EN: "Reliability - minimal system downtime"},
		{ID: "Compliance - memenuhi standar BAN-PT untuk tracer study", EN: "Compliance - meets BAN-PT tracer study standards"},
	},
}

// TotalAuthorised adalah pagu anggaran proyek sesuai Project Charter.
const TotalAuthorised = 16_000_000.0

// ContingencyReserve adalah cadangan untuk risiko yang sudah teridentifikasi
// (known unknowns). Menurut PMBOK cadangan ini berada DI DALAM cost baseline.
const ContingencyReserve = 500_000.0

// CharterAllocation adalah rincian anggaran versi Project Charter. Dipakai
// untuk membandingkan alokasi top-down terhadap estimasi bottom-up.
var CharterAllocation = []BudgetLine{
	{Key: "server", Amount: 3_000_000, Label: Text{ID: "Server & hosting web (1 tahun)", EN: "Server & web hosting (1 year)"}},
	{Key: "labour", Amount: 8_000_000, Label: Text{ID: "Gaji/insentif tim pengembang", EN: "Development team wages/incentives"}},
	{Key: "training", Amount: 2_000_000, Label: Text{ID: "Pelatihan pengguna (admin & alumni)", EN: "User training (admin & alumni)"}},
	{Key: "tools", Amount: 1_500_000, Label: Text{ID: "Perangkat lunak & tools pendukung", EN: "Supporting software & tools"}},
	{Key: "reserve", Amount: 1_500_000, Label: Text{ID: "Maintenance & kontinjensi", EN: "Maintenance & contingency"}},
}

// RateCard adalah tarif harian per peran (Rp per hari-orang). Basis estimasi
// biaya bottom-up untuk seluruh aktivitas.
// Tarif sengaja di bawah harga pasar profesional: Project Charter menyatakan
// tim terdiri dari mahasiswa dengan keterbatasan waktu, sehingga yang dibayar
// adalah insentif, bukan gaji penuh.
var RateCard = map[Role]float64{
	RolePM: 60_000, RoleBA: 50_000, RoleSA: 60_000, RoleTL: 60_000, RoleBE: 55_000,
	RoleFE: 55_000, RoleDBA: 50_000, RoleUX: 45_000, RoleQA: 45_000, RoleOPS: 60_000,
}

// Capacity adalah kapasitas hari-orang yang tersedia per peran per hari kerja.
// DevOps hanya paruh waktu - sesuai Project Charter yang menandainya opsional.
var Capacity = map[Role]float64{
	RolePM: 1, RoleBA: 1, RoleSA: 1, RoleTL: 1, RoleBE: 1,
	RoleFE: 1, RoleDBA: 1, RoleUX: 1, RoleQA: 1, RoleOPS: 0.5,
}

// DefaultStatusDate adalah tanggal data (data date) awal untuk pelaporan
// Earned Value. Bisa digeser dari antarmuka.
const DefaultStatusDate = "2025-12-19"

// WBSPhases adalah Work Breakdown Structure sampai level paket kerja,
// persis seperti yang disetujui pada Tugas 6.
var WBSPhases = []WBSPhase{
	{
		Code: "1.0", Weeks: 3,
		Name:      Text{ID: "Analisis Kebutuhan", EN: "Requirements Analysis"},
		Milestone: Text{ID: "Approval BRD & SRS", EN: "BRD & SRS approved"},
		Packages: []WBSPackage{
			{Code: "1.1", Name: Text{ID: "Requirement Gathering & Wawancara Stakeholder", EN: "Requirement Gathering & Stakeholder Interviews"}},
			{Code: "1.2", Name: Text{ID: "Analisis Kebutuhan Bisnis & Teknis", EN: "Business & Technical Requirements Analysis"}},
			{Code: "1.3", Name: Text{ID: "Dokumentasi BRD & SRS", EN: "BRD & SRS Documentation"}},
		},
	},
	{
		Code: "2.0", Weeks: 3,
		Name:      Text{ID: "Desain Sistem", EN: "System Design"},
		Milestone: Text{ID: "Approval Desain & Prototype", EN: "Design & prototype approved"},
		Packages: []WBSPackage{
			{Code: "2.1", Name: Text{ID: "Desain Arsitektur Sistem & Database (ERD)", EN: "System Architecture & Database Design (ERD)"}},
			{Code: "2.2", Name: Text{ID: "Desain UI/UX & Wireframe", EN: "UI/UX & Wireframe Design"}},
			{Code: "2.3", Name: Text{ID: "Review Desain & Persetujuan", EN: "Design Review & Sign-off"}},
		},
	},
	{
		Code: "3.0", Weeks: 7,
		Name:      Text{ID: "Pengembangan", EN: "Development"},
		Milestone: Text{ID: "Sistem Development Selesai", EN: "Development complete"},
		Packages: []WBSPackage{
			{Code: "3.1", Name: Text{ID: "Setup Environment & Database", EN: "Environment & Database Setup"}},
			{Code: "3.2", Name: Text{ID: "Pengembangan Backend (API, Logic)", EN: "Backend Development (API, Logic)"}},
			{Code: "3.3", Name: Text{ID: "Pengembangan Frontend (UI)", EN: "Frontend Development (UI)"}},
			{Code: "3.4", Name: Text{ID: "Integrasi Backend-Frontend", EN: "Backend-Frontend Integration"}},
			{Code: "3.5", Name: Text{ID: "Implementasi Security & Enkripsi", EN: "Security & Encryption Implementation"}},
		},
	},
	{
		Code: "4.0", Weeks: 2,
		Name:      Text{ID: "Uji Coba & Pelatihan", EN: "Testing & Training"},
		Milestone: Text{ID: "UAT Pass & User Training", EN: "UAT passed & users trained"},
		Packages: []WBSPackage{
			{Code: "4.1", Name: Text{ID: "Testing (Unit, Integration, UAT)", EN: "Testing (Unit, Integration, UAT)"}},
			{Code: "4.2", Name: Text{ID: "Bug Fixing & Pelatihan User", EN: "Bug Fixing & User Training"}},
		},
	},
	{
		Code: "5.0", Weeks: 2,
		Name:      Text{ID: "Implementasi & Evaluasi", EN: "Implementation & Evaluation"},
		Milestone: Text{ID: "Go-Live & Evaluasi", EN: "Go-live & evaluation"},
		Packages: []WBSPackage{
			{Code: "5.1", Name: Text{ID: "Deployment ke Server Production", EN: "Deployment to Production Server"}},
			{Code: "5.2", Name: Text{ID: "Monitoring, Evaluasi & Dokumentasi", EN: "Monitoring, Evaluation & Documentation"}},
		},
	},
}

// PackageName mencari nama paket kerja berdasarkan kodenya.
func PackageName(code string) Text {
	for _, ph := range WBSPhases {
		for _, pk := range ph.Packages {
			if pk.Code == code {
				return pk.Name
			}
		}
	}
	return Text{ID: code, EN: code}
}

// PhaseOf mengembalikan fase induk untuk sebuah kode paket kerja.
func PhaseOf(packageCode string) WBSPhase {
	for _, ph := range WBSPhases {
		for _, pk := range ph.Packages {
			if pk.Code == packageCode {
				return ph
			}
		}
	}
	return WBSPhase{}
}
