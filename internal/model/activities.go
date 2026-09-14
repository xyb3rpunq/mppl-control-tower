package model

// Activities adalah jaringan aktivitas level 3 (Activity-on-Node).
//
// Dekomposisi ini adalah perluasan dari 15 paket kerja pada WBS Tugas 6.
// Perluasan diperlukan karena CPM tidak bisa menemukan jalur kritis yang
// bermakna pada WBS yang setiap fasenya hanya satu rantai lurus - tanpa
// aktivitas paralel, semua pekerjaan tampak kritis dan float selalu nol.
//
// Total durasi jaringan ini dikunci 85 hari kerja = 17 minggu, sama persis
// dengan jadwal yang disetujui di Project Charter. Uji di schedule_test.go
// menjaga invarian tersebut.
var Activities = []Activity{
	// ------------------------------------------------------------ 1.0
	{
		ID: "A01", WBS: "1.1", Duration: 4, Optimistic: 3, Pessimistic: 7,
		Name:   Text{ID: "Wawancara stakeholder akademik & bagian alumni", EN: "Interview academic & alumni-office stakeholders"},
		Team:   []TeamSlot{{RoleBA, 1}},
		Actual: Actual{Started: true, Start: 0, Duration: 5, Cost: 240_000},
	},
	{
		ID: "A02", WBS: "1.1", Duration: 3, Optimistic: 2, Pessimistic: 5, Pred: []Predecessor{FS("A01")},
		Name:   Text{ID: "Survei & observasi proses pendataan manual eksisting", EN: "Survey & observe the existing manual data process"},
		Team:   []TeamSlot{{RoleBA, 1}},
		Actual: Actual{Started: true, Start: 5, Duration: 3, Cost: 165_000},
	},
	{
		ID: "A03", WBS: "1.2", Duration: 4, Optimistic: 3, Pessimistic: 6, Pred: []Predecessor{FS("A02")},
		Name:   Text{ID: "Analisis kebutuhan bisnis & penetapan KPI partisipasi", EN: "Business requirements analysis & participation KPI"},
		Team:   []TeamSlot{{RoleBA, 1}, {RolePM, 0.3}},
		Actual: Actual{Started: true, Start: 8, Duration: 4, Cost: 285_000},
	},
	{
		ID: "A04", WBS: "1.2", Duration: 4, Optimistic: 3, Pessimistic: 8, Pred: []Predecessor{FS("A02")},
		Name:   Text{ID: "Analisis teknis & studi kelayakan integrasi SIAKAD", EN: "Technical analysis & SIAKAD integration feasibility"},
		Team:   []TeamSlot{{RoleSA, 1}},
		Actual: Actual{Started: true, Start: 8, Duration: 4, Cost: 275_000},
	},
	{
		ID: "A05", WBS: "1.3", Duration: 4, Optimistic: 3, Pessimistic: 6, Pred: []Predecessor{FS("A03")},
		Name:   Text{ID: "Penyusunan Business Requirements Document (BRD)", EN: "Draft the Business Requirements Document (BRD)"},
		Team:   []TeamSlot{{RoleBA, 1}},
		Actual: Actual{Started: true, Start: 12, Duration: 4, Cost: 225_000},
	},
	{
		ID: "A06", WBS: "1.3", Duration: 4, Optimistic: 3, Pessimistic: 7, Pred: []Predecessor{FS("A03"), FS("A04")},
		Name:   Text{ID: "Penyusunan Software Requirements Specification (SRS)", EN: "Draft the Software Requirements Specification (SRS)"},
		Team:   []TeamSlot{{RoleSA, 1}, {RoleBA, 0.5}},
		Actual: Actual{Started: true, Start: 12, Duration: 4, Cost: 360_000},
	},
	{
		ID: "M1", WBS: "1.3", Milestone: true, Pred: []Predecessor{FS("A05"), FS("A06")},
		Name:   Text{ID: "MILESTONE - Approval BRD & SRS", EN: "MILESTONE - BRD & SRS approved"},
		Actual: Actual{Started: true, Start: 16},
	},

	// ------------------------------------------------------------ 2.0
	{
		ID: "A08", WBS: "2.1", Duration: 5, Optimistic: 4, Pessimistic: 9, Pred: []Predecessor{FS("M1")},
		Name:   Text{ID: "Desain arsitektur sistem & pola integrasi", EN: "System architecture & integration pattern design"},
		Team:   []TeamSlot{{RoleSA, 1}, {RoleTL, 0.4}},
		Actual: Actual{Started: true, Start: 16, Duration: 5, Cost: 445_000},
	},
	{
		ID: "A09", WBS: "2.1", Duration: 5, Optimistic: 4, Pessimistic: 8, Pred: []Predecessor{FS("M1")},
		Name:   Text{ID: "Desain skema basis data & Entity Relationship Diagram", EN: "Database schema & Entity Relationship Diagram design"},
		Team:   []TeamSlot{{RoleDBA, 1}, {RoleSA, 0.3}},
		Actual: Actual{Started: true, Start: 16, Duration: 6, Cost: 385_000},
	},
	{
		ID: "A10", WBS: "2.2", Duration: 4, Optimistic: 3, Pessimistic: 6, Pred: []Predecessor{FS("M1")},
		Name:   Text{ID: "Wireframe & pemetaan alur pengguna alumni/admin", EN: "Wireframes & alumni/admin user-flow mapping"},
		Team:   []TeamSlot{{RoleUX, 1}},
		Extras: []Extra{{Key: "tools", Amount: 300_000, TimeBased: true, Label: Text{ID: "Langganan tool desain", EN: "Design tool subscription"}}},
		Actual: Actual{Started: true, Start: 16, Duration: 4, Cost: 520_000},
	},
	{
		ID: "A11", WBS: "2.2", Duration: 4, Optimistic: 3, Pessimistic: 7, Pred: []Predecessor{FS("A10")},
		Name:   Text{ID: "Mockup hi-fi & penyusunan design system", EN: "Hi-fi mockups & design system"},
		Team:   []TeamSlot{{RoleUX, 1}},
		Actual: Actual{Started: true, Start: 20, Duration: 4, Cost: 200_000},
	},
	{
		ID: "A12", WBS: "2.2", Duration: 3, Optimistic: 2, Pessimistic: 6, Pred: []Predecessor{FS("A11")},
		Name:   Text{ID: "Prototipe interaktif & usability testing", EN: "Interactive prototype & usability testing"},
		Team:   []TeamSlot{{RoleUX, 1}, {RoleBA, 0.3}},
		Actual: Actual{Started: true, Start: 24, Duration: 4, Cost: 215_000},
	},
	{
		ID: "A13", WBS: "2.3", Duration: 4, Optimistic: 3, Pessimistic: 8, Pred: []Predecessor{FS("A08"), FS("A09"), FS("A12")},
		Name:   Text{ID: "Review desain & persetujuan pemangku kepentingan", EN: "Design review & stakeholder sign-off"},
		Team:   []TeamSlot{{RolePM, 1}, {RoleSA, 0.3}},
		Actual: Actual{Started: true, Start: 28, Duration: 4, Cost: 320_000},
	},
	{
		ID: "M2", WBS: "2.3", Milestone: true, Pred: []Predecessor{FS("A13")},
		Name:   Text{ID: "MILESTONE - Approval Desain & Prototype", EN: "MILESTONE - Design & prototype approved"},
		Actual: Actual{Started: true, Start: 32},
	},

	// ------------------------------------------------------------ 3.0
	{
		ID: "A14", WBS: "3.1", Duration: 3, Optimistic: 2, Pessimistic: 6, Pred: []Predecessor{FS("M2")},
		Name: Text{ID: "Setup environment, repositori & pipeline CI/CD", EN: "Environment, repository & CI/CD pipeline setup"},
		Team: []TeamSlot{{RoleOPS, 1}},
		Extras: []Extra{
			{Key: "server", Amount: 500_000, TimeBased: true, Label: Text{ID: "Sewa server pengembangan", EN: "Development server rental"}},
			{Key: "tools", Amount: 900_000, TimeBased: true, Label: Text{ID: "Langganan tools & framework pendukung", EN: "Supporting tool & framework subscriptions"}},
		},
		Actual: Actual{Started: true, Start: 32, Duration: 3, Cost: 1_630_000},
	},
	{
		ID: "A15", WBS: "3.1", Duration: 3, Optimistic: 2, Pessimistic: 5, Pred: []Predecessor{FS("A14")},
		Name:   Text{ID: "Implementasi skema basis data & data seeding", EN: "Database schema implementation & data seeding"},
		Team:   []TeamSlot{{RoleDBA, 1}},
		Actual: Actual{Started: true, Start: 35, Duration: 3, Cost: 160_000},
	},
	{
		ID: "A16", WBS: "3.2", Duration: 5, Optimistic: 4, Pessimistic: 9, Pred: []Predecessor{FS("A15")},
		Name:   Text{ID: "API autentikasi & manajemen akun alumni", EN: "Authentication API & alumni account management"},
		Team:   []TeamSlot{{RoleBE, 1}},
		Actual: Actual{Started: true, Start: 38, Duration: 6, Cost: 345_000},
	},
	{
		ID: "A17", WBS: "3.2", Duration: 6, Optimistic: 4, Pessimistic: 11, Pred: []Predecessor{FS("A16")},
		Name:   Text{ID: "API kuesioner tracer study dinamis", EN: "Dynamic tracer study questionnaire API"},
		Team:   []TeamSlot{{RoleBE, 1}},
		Actual: Actual{Started: true, Start: 44, Duration: 7, Cost: 415_000},
	},
	{
		ID: "A18", WBS: "3.2", Duration: 5, Optimistic: 4, Pessimistic: 9, Pred: []Predecessor{FS("A17")},
		Name: Text{ID: "API dasbor & agregasi laporan tracer study", EN: "Dashboard API & tracer study report aggregation"},
		Team: []TeamSlot{{RoleBE, 1}},
	},
	{
		ID: "A19", WBS: "3.2", Duration: 4, Optimistic: 3, Pessimistic: 7, Pred: []Predecessor{FS("A16")},
		Name:   Text{ID: "API panel admin & pengelolaan data master", EN: "Admin panel API & master data management"},
		Team:   []TeamSlot{{RoleBE, 1}},
		Actual: Actual{Started: true, Start: 44, Duration: 5, Cost: 285_000},
	},
	{
		ID: "A20", WBS: "3.3", Duration: 4, Optimistic: 3, Pessimistic: 7, Pred: []Predecessor{FS("A14")},
		Name:   Text{ID: "Frontend: landing page & registrasi alumni", EN: "Frontend: landing page & alumni registration"},
		Team:   []TeamSlot{{RoleFE, 1}},
		Actual: Actual{Started: true, Start: 35, Duration: 4, Cost: 235_000},
	},
	{
		ID: "A21", WBS: "3.3", Duration: 6, Optimistic: 4, Pessimistic: 10, Pred: []Predecessor{FS("A20")},
		Name:   Text{ID: "Frontend: formulir kuesioner responsif", EN: "Frontend: responsive questionnaire form"},
		Team:   []TeamSlot{{RoleFE, 1}},
		Actual: Actual{Started: true, Start: 39, Duration: 7, Cost: 400_000},
	},
	{
		ID: "A22", WBS: "3.3", Duration: 5, Optimistic: 4, Pessimistic: 9, Pred: []Predecessor{FS("A21")},
		Name: Text{ID: "Frontend: dasbor & visualisasi hasil tracer study", EN: "Frontend: dashboard & tracer study visualisation"},
		Team: []TeamSlot{{RoleFE, 1}},
	},
	{
		ID: "A23", WBS: "3.3", Duration: 4, Optimistic: 3, Pessimistic: 7, Pred: []Predecessor{FS("A21")},
		Name: Text{ID: "Frontend: panel administrasi kampus", EN: "Frontend: campus administration panel"},
		Team: []TeamSlot{{RoleFE, 1}},
	},
	{
		ID: "A24", WBS: "3.4", Duration: 4, Optimistic: 3, Pessimistic: 8,
		Pred: []Predecessor{FS("A18"), FS("A19"), FS("A22"), FS("A23")},
		Name: Text{ID: "Integrasi frontend-backend & uji kontrak API", EN: "Frontend-backend integration & API contract testing"},
		Team: []TeamSlot{{RoleTL, 1}, {RoleBE, 0.5}, {RoleFE, 0.5}},
	},
	{
		ID: "A25", WBS: "3.4", Duration: 4, Optimistic: 3, Pessimistic: 10, Pred: []Predecessor{FS("A24")},
		Name: Text{ID: "Integrasi SIAKAD & migrasi data alumni eksisting", EN: "SIAKAD integration & existing alumni data migration"},
		Team: []TeamSlot{{RoleSA, 1}, {RoleDBA, 0.6}},
	},
	{
		ID: "A26", WBS: "3.5", Duration: 3, Optimistic: 2, Pessimistic: 6, Pred: []Predecessor{FS("A25")},
		Name: Text{ID: "Implementasi enkripsi, kontrol akses & audit log", EN: "Encryption, access control & audit log implementation"},
		Team: []TeamSlot{{RoleBE, 1}, {RoleTL, 0.3}},
	},
	{
		ID: "A27", WBS: "3.5", Duration: 2, Optimistic: 1, Pessimistic: 5, Pred: []Predecessor{FS("A26")},
		Name: Text{ID: "Hardening & uji penetrasi internal", EN: "Hardening & internal penetration test"},
		Team: []TeamSlot{{RoleTL, 1}},
	},
	{
		ID: "M3", WBS: "3.5", Milestone: true, Pred: []Predecessor{FS("A27")},
		Name: Text{ID: "MILESTONE - Sistem Development Selesai", EN: "MILESTONE - Development complete"},
	},

	// ------------------------------------------------------------ 4.0
	{
		ID: "A28", WBS: "4.1", Duration: 3, Optimistic: 2, Pessimistic: 5, Pred: []Predecessor{FS("M3")},
		Name:   Text{ID: "Unit testing & integration testing otomatis", EN: "Automated unit & integration testing"},
		Team:   []TeamSlot{{RoleQA, 1}},
		Extras: []Extra{{Key: "tools", Amount: 300_000, TimeBased: true, Label: Text{ID: "Langganan tools pengujian & pemantauan", EN: "Testing & monitoring tool subscriptions"}}},
	},
	{
		ID: "A29", WBS: "4.1", Duration: 2, Optimistic: 1, Pessimistic: 4, Pred: []Predecessor{FS("A28")},
		Name: Text{ID: "System & performance testing (target respons < 3 detik)", EN: "System & performance testing (target response < 3 s)"},
		Team: []TeamSlot{{RoleQA, 1}, {RoleOPS, 0.4}},
	},
	{
		ID: "A30", WBS: "4.1", Duration: 3, Optimistic: 2, Pessimistic: 6, Pred: []Predecessor{FS("A29")},
		Name: Text{ID: "User Acceptance Testing bersama bagian akademik", EN: "User Acceptance Testing with the academic office"},
		Team: []TeamSlot{{RoleQA, 1}, {RoleBA, 0.5}},
	},
	{
		ID: "A31", WBS: "4.2", Duration: 2, Optimistic: 1, Pessimistic: 6, Pred: []Predecessor{FS("A30")},
		Name: Text{ID: "Bug fixing & regression testing", EN: "Bug fixing & regression testing"},
		Team: []TeamSlot{{RoleBE, 1}, {RoleFE, 0.6}, {RoleQA, 0.5}},
	},
	{
		ID: "A32", WBS: "4.2", Duration: 4, Optimistic: 3, Pessimistic: 6, Pred: []Predecessor{FS("M3")},
		Name:   Text{ID: "Penyusunan materi & pelatihan admin kampus", EN: "Training material preparation & campus admin training"},
		Team:   []TeamSlot{{RolePM, 1}, {RoleBA, 0.4}},
		Extras: []Extra{{Key: "training", Amount: 1_200_000, Label: Text{ID: "Pelatihan admin kampus", EN: "Campus admin training"}}},
	},
	{
		ID: "M4", WBS: "4.2", Milestone: true, Pred: []Predecessor{FS("A31"), FS("A32")},
		Name: Text{ID: "MILESTONE - UAT Pass & User Training", EN: "MILESTONE - UAT passed & users trained"},
	},

	// ------------------------------------------------------------ 5.0
	{
		ID: "A33", WBS: "5.1", Duration: 3, Optimistic: 2, Pessimistic: 6, Pred: []Predecessor{FS("M4")},
		Name:   Text{ID: "Deployment produksi, konfigurasi domain & sertifikat SSL", EN: "Production deployment, domain & SSL certificate setup"},
		Team:   []TeamSlot{{RoleOPS, 1}},
		Extras: []Extra{{Key: "server", Amount: 2_500_000, Label: Text{ID: "Hosting produksi & domain (1 tahun)", EN: "Production hosting & domain (1 year)"}}},
	},
	{
		ID: "A34", WBS: "5.1", Duration: 2, Optimistic: 1, Pessimistic: 4, Pred: []Predecessor{FS("A33")},
		Name:   Text{ID: "Go-live & sosialisasi sistem kepada alumni", EN: "Go-live & system socialisation to alumni"},
		Team:   []TeamSlot{{RolePM, 1}},
		Extras: []Extra{{Key: "training", Amount: 800_000, Label: Text{ID: "Sosialisasi & kampanye partisipasi alumni", EN: "Alumni participation campaign"}}},
	},
	{
		ID: "A35", WBS: "5.2", Duration: 3, Optimistic: 2, Pessimistic: 6, Pred: []Predecessor{FS("A34")},
		Name: Text{ID: "Monitoring performa & uptime pasca go-live", EN: "Post go-live performance & uptime monitoring"},
		Team: []TeamSlot{{RoleOPS, 1}},
	},
	{
		ID: "A36", WBS: "5.2", Duration: 2, Optimistic: 1, Pessimistic: 4, Pred: []Predecessor{FS("A35")},
		Name: Text{ID: "Evaluasi proyek, lessons learned & serah terima dokumentasi", EN: "Project evaluation, lessons learned & documentation handover"},
		Team: []TeamSlot{{RolePM, 1}, {RoleTL, 0.5}},
	},
	{
		ID: "M5", WBS: "5.2", Milestone: true, Pred: []Predecessor{FS("A36")},
		Name: Text{ID: "MILESTONE - Go-Live & Evaluasi Selesai", EN: "MILESTONE - Go-live & evaluation complete"},
	},
}

// ActivityByID membuat indeks aktivitas berdasarkan kodenya.
func ActivityByID() map[string]Activity {
	m := make(map[string]Activity, len(Activities))
	for _, a := range Activities {
		m[a.ID] = a
	}
	return m
}

// BAC (Budget at Completion) adalah jumlah anggaran seluruh aktivitas. Inilah
// basis perhitungan Earned Value - cadangan tidak ikut, karena pekerjaan yang
// belum terjadi tidak bisa "diperoleh".
func BAC() float64 {
	var total float64
	for _, a := range Activities {
		total += a.Budget(RateCard)
	}
	return total
}

// CostBaseline adalah BAC ditambah cadangan kontinjensi - garis dasar biaya
// yang dipantau Project Manager.
func CostBaseline() float64 { return BAC() + ContingencyReserve }

// ManagementReserve adalah sisa pagu setelah estimasi bottom-up dan cadangan
// kontinjensi diambil. Inilah cara Project Charter (top-down, Rp 16 juta) dan
// estimasi rinci (bottom-up) direkonsiliasi tanpa memalsukan salah satunya.
func ManagementReserve() float64 { return TotalAuthorised - CostBaseline() }
