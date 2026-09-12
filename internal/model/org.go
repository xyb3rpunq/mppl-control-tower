package model

// Member adalah satu posisi pada struktur organisasi proyek (asal: Tugas 5
// dan Tugas 10). Person kosong berarti posisi belum diisi nama - pada dokumen
// asli hanya tiga peran yang punya nama mahasiswa.
type Member struct {
	Role         Role
	Person       string
	NIM          string
	Title        Text
	ReportsTo    Role // kosong untuk Project Manager
	Level        int  // 1 sponsor, 2 PM, 3 core lead, 4 tim pelaksana
	Optional     bool
	Responsibili []Text
}

// Team adalah struktur organisasi proyek, disusun sesuai bagan hirarki pada
// Tugas 10: lima level dari sponsor sampai pengguna akhir.
var Team = []Member{
	{
		Role: RolePM, Person: "Andri Zulfikar", NIM: "20200801426", Level: 2,
		Title: Text{ID: "Project Manager", EN: "Project Manager"},
		Responsibili: []Text{
			{ID: "Menyusun project plan, jadwal, dan Work Breakdown Structure", EN: "Build the project plan, schedule, and Work Breakdown Structure"},
			{ID: "Mengorganisir tim, membagi tugas, dan mengalokasikan sumber daya", EN: "Organise the team, assign work, and allocate resources"},
			{ID: "Memantau progres, anggaran Rp 16 juta, dan jadwal 17 minggu", EN: "Monitor progress, the IDR 16m budget, and the 17-week schedule"},
			{ID: "Mengelola komunikasi antara sponsor, tim, dan pemangku kepentingan", EN: "Manage communication between sponsor, team, and stakeholders"},
			{ID: "Menjalankan manajemen risiko dan mitigasinya", EN: "Run risk management and mitigation"},
		},
	},
	{
		Role: RoleBA, Level: 3, ReportsTo: RolePM,
		Title: Text{ID: "Business Analyst", EN: "Business Analyst"},
		Responsibili: []Text{
			{ID: "Menggali kebutuhan lewat wawancara bagian akademik dan alumni", EN: "Elicit requirements by interviewing the academic and alumni offices"},
			{ID: "Menyusun Business Requirements Document", EN: "Draft the Business Requirements Document"},
			{ID: "Menetapkan KPI proyek, termasuk target partisipasi 70%", EN: "Define project KPIs, including the 70% participation target"},
			{ID: "Memvalidasi solusi terhadap nilai bisnis yang disepakati", EN: "Validate the solution against agreed business value"},
		},
	},
	{
		Role: RoleSA, Level: 3, ReportsTo: RolePM,
		Title: Text{ID: "System Analyst", EN: "System Analyst"},
		Responsibili: []Text{
			{ID: "Menerjemahkan kebutuhan bisnis menjadi spesifikasi teknis", EN: "Translate business requirements into technical specifications"},
			{ID: "Merancang arsitektur sistem, skema basis data, dan ERD", EN: "Design the system architecture, database schema, and ERD"},
			{ID: "Memastikan integrasi dengan sistem akademik kampus berjalan", EN: "Ensure integration with the campus academic system works"},
			{ID: "Melakukan studi kelayakan teknis dan analisis biaya solusi", EN: "Run technical feasibility studies and solution cost analysis"},
		},
	},
	{
		Role: RoleTL, Level: 3, ReportsTo: RolePM,
		Title: Text{ID: "Technical Lead", EN: "Technical Lead"},
		Responsibili: []Text{
			{ID: "Memimpin tim pengembang backend, frontend, dan basis data", EN: "Lead the backend, frontend, and database developers"},
			{ID: "Menetapkan standar coding dan melakukan code review", EN: "Set coding standards and perform code review"},
			{ID: "Mengambil keputusan teknis soal tumpukan teknologi dan arsitektur", EN: "Decide on technology stack and architecture"},
			{ID: "Menjaga skalabilitas, keamanan, dan performa sistem", EN: "Safeguard system scalability, security, and performance"},
		},
	},
	{
		Role: RoleBE, Person: "Daniel Hutajulu", NIM: "20210801207", Level: 4, ReportsTo: RoleTL,
		Title: Text{ID: "Backend Developer", EN: "Backend Developer"},
		Responsibili: []Text{
			{ID: "Membangun RESTful API untuk autentikasi, kuesioner, dasbor, dan admin", EN: "Build RESTful APIs for authentication, questionnaire, dashboard, and admin"},
			{ID: "Mengimplementasikan business logic sistem tracer study", EN: "Implement the tracer study business logic"},
			{ID: "Menerapkan fitur keamanan: enkripsi dan login aman", EN: "Implement security features: encryption and secure login"},
			{ID: "Menulis unit test dan dokumentasi API", EN: "Write unit tests and API documentation"},
		},
	},
	{
		Role: RoleFE, Level: 4, ReportsTo: RoleTL,
		Title: Text{ID: "Frontend Developer", EN: "Frontend Developer"},
		Responsibili: []Text{
			{ID: "Mengimplementasikan desain UI/UX menjadi antarmuka responsif", EN: "Turn the UI/UX design into a responsive interface"},
			{ID: "Membangun landing page, form kuesioner, dasbor, dan panel admin", EN: "Build the landing page, questionnaire form, dashboard, and admin panel"},
			{ID: "Mengintegrasikan antarmuka dengan backend API", EN: "Integrate the interface with the backend API"},
			{ID: "Memastikan kompatibilitas lintas peramban dan perangkat", EN: "Ensure cross-browser and cross-device compatibility"},
		},
	},
	{
		Role: RoleDBA, Person: "Fendi Dzunnurain", NIM: "20240801367", Level: 4, ReportsTo: RoleTL,
		Title: Text{ID: "Database Administrator", EN: "Database Administrator"},
		Responsibili: []Text{
			{ID: "Merancang skema basis data yang ternormalisasi dan efisien", EN: "Design a normalised, efficient database schema"},
			{ID: "Menerapkan prosedur backup dan pemulihan data alumni", EN: "Implement backup and recovery procedures for alumni data"},
			{ID: "Menjaga keamanan dan integritas data", EN: "Safeguard data security and integrity"},
			{ID: "Melakukan migrasi dan impor data alumni eksisting", EN: "Migrate and import existing alumni data"},
		},
	},
	{
		Role: RoleUX, Level: 4, ReportsTo: RoleTL,
		Title: Text{ID: "UI/UX Designer", EN: "UI/UX Designer"},
		Responsibili: []Text{
			{ID: "Melakukan user research dan menyusun user persona", EN: "Run user research and build user personas"},
			{ID: "Membuat wireframe, mockup, dan prototipe interaktif", EN: "Produce wireframes, mockups, and interactive prototypes"},
			{ID: "Menyusun design system dan style guide", EN: "Build the design system and style guide"},
			{ID: "Menjalankan usability testing bersama alumni dan admin", EN: "Run usability testing with alumni and admins"},
		},
	},
	{
		Role: RoleQA, Level: 4, ReportsTo: RoleTL,
		Title: Text{ID: "QA Tester", EN: "QA Tester"},
		Responsibili: []Text{
			{ID: "Menyusun test plan, test strategy, dan test case", EN: "Write the test plan, test strategy, and test cases"},
			{ID: "Menjalankan functional, performance, security, dan usability testing", EN: "Run functional, performance, security, and usability testing"},
			{ID: "Mendokumentasikan defect dan melakukan regression testing", EN: "Document defects and run regression testing"},
			{ID: "Memastikan sistem memenuhi target uptime 95% dan respons < 3 detik", EN: "Verify the system meets 95% uptime and sub-3-second response targets"},
		},
	},
	{
		Role: RoleOPS, Level: 4, ReportsTo: RoleTL, Optional: true,
		Title: Text{ID: "DevOps Engineer (opsional)", EN: "DevOps Engineer (optional)"},
		Responsibili: []Text{
			{ID: "Menyiapkan server dan hosting web sesuai pagu Rp 3 juta", EN: "Provision server and web hosting within the IDR 3m allocation"},
			{ID: "Membangun pipeline CI/CD dan otomatisasi deployment", EN: "Build the CI/CD pipeline and deployment automation"},
			{ID: "Memantau performa sistem dan uptime (target minimal 95%)", EN: "Monitor system performance and uptime (95% minimum)"},
			{ID: "Merencanakan backup dan pemulihan bencana", EN: "Plan backup and disaster recovery"},
		},
	},
}

// RACIValue adalah satu nilai pada matriks RACI.
type RACIValue string

// Nilai RACI menurut PMBOK.
const (
	R RACIValue = "R" // Responsible - mengerjakan
	A RACIValue = "A" // Accountable - bertanggung jawab akhir & menyetujui
	C RACIValue = "C" // Consulted   - dimintai masukan
	I RACIValue = "I" // Informed    - diberi tahu
	X RACIValue = ""  // tidak terlibat
)

// RACIRow adalah satu baris matriks RACI untuk satu fase WBS.
type RACIRow struct {
	Phase      string
	Name       Text
	Assignment map[Role]RACIValue
}

// RACIRoles adalah urutan kolom matriks RACI.
var RACIRoles = []Role{RolePM, RoleBA, RoleSA, RoleTL, RoleBE, RoleFE, RoleDBA, RoleUX, RoleQA, RoleOPS}

// RACI adalah matriks tanggung jawab per fase (asal: Tugas 6 bagian 2.5 dan
// Tugas 10 bagian 3). Aturan mainnya: tepat satu A per baris, minimal satu R.
// Aturan itu diperiksa oleh uji, bukan sekadar dipercaya.
var RACI = []RACIRow{
	{
		Phase: "1.0", Name: Text{ID: "Analisis Kebutuhan", EN: "Requirements Analysis"},
		Assignment: map[Role]RACIValue{
			RolePM: A, RoleBA: R, RoleSA: R, RoleTL: C, RoleBE: I,
			RoleFE: I, RoleDBA: I, RoleUX: C, RoleQA: I, RoleOPS: I,
		},
	},
	{
		Phase: "2.0", Name: Text{ID: "Desain Sistem", EN: "System Design"},
		Assignment: map[Role]RACIValue{
			RolePM: A, RoleBA: C, RoleSA: R, RoleTL: C, RoleBE: C,
			RoleFE: C, RoleDBA: R, RoleUX: R, RoleQA: I, RoleOPS: I,
		},
	},
	{
		Phase: "3.0", Name: Text{ID: "Pengembangan", EN: "Development"},
		Assignment: map[Role]RACIValue{
			RolePM: A, RoleBA: I, RoleSA: C, RoleTL: R, RoleBE: R,
			RoleFE: R, RoleDBA: R, RoleUX: C, RoleQA: C, RoleOPS: R,
		},
	},
	{
		Phase: "4.0", Name: Text{ID: "Uji Coba & Pelatihan", EN: "Testing & Training"},
		Assignment: map[Role]RACIValue{
			RolePM: A, RoleBA: C, RoleSA: C, RoleTL: C, RoleBE: R,
			RoleFE: R, RoleDBA: I, RoleUX: C, RoleQA: R, RoleOPS: C,
		},
	},
	{
		Phase: "5.0", Name: Text{ID: "Implementasi & Evaluasi", EN: "Implementation & Evaluation"},
		Assignment: map[Role]RACIValue{
			RolePM: A, RoleBA: I, RoleSA: C, RoleTL: R, RoleBE: C,
			RoleFE: C, RoleDBA: R, RoleUX: I, RoleQA: C, RoleOPS: R,
		},
	},
}

// Stakeholder adalah satu pemangku kepentingan pada grid kuasa-kepentingan.
// Power dan Interest bernilai 1..5.
type Stakeholder struct {
	ID       string
	Name     Text
	Power    int
	Interest int
	Strategy Text
	Need     Text
	Internal bool
}

// Stakeholders adalah hasil analisis pemangku kepentingan. Empat kuadran grid
// menentukan strategi: kelola erat, jaga kepuasan, beri informasi, pantau.
var Stakeholders = []Stakeholder{
	{
		ID: "S1", Power: 5, Interest: 5, Internal: true,
		Name:     Text{ID: "Pimpinan STIE Jayakusuma (Project Sponsor)", EN: "STIE Jayakusuma leadership (Project Sponsor)"},
		Need:     Text{ID: "Bukti bahwa dana Rp 16 juta menghasilkan data layak akreditasi", EN: "Evidence the IDR 16m spend yields accreditation-grade data"},
		Strategy: Text{ID: "Kelola erat - laporan mingguan berisi SPI, CPI, dan proyeksi biaya akhir", EN: "Manage closely - weekly report with SPI, CPI, and cost forecast"},
	},
	{
		ID: "S2", Power: 4, Interest: 5, Internal: true,
		Name:     Text{ID: "Bagian Akademik & Alumni", EN: "Academic & Alumni Office"},
		Need:     Text{ID: "Sistem yang mengurangi kerja manual dan siap dipakai saat borang disusun", EN: "A system that cuts manual work and is ready when accreditation forms are compiled"},
		Strategy: Text{ID: "Kelola erat - libatkan di setiap review desain dan UAT", EN: "Manage closely - involve in every design review and UAT"},
	},
	{
		ID: "S3", Power: 4, Interest: 3, Internal: true,
		Name:     Text{ID: "Unit IT Kampus", EN: "Campus IT Unit"},
		Need:     Text{ID: "Sistem yang bisa dirawat unit sendiri tanpa ketergantungan pada tim proyek", EN: "A system the unit can maintain without depending on the project team"},
		Strategy: Text{ID: "Jaga kepuasan - sertakan dalam keputusan arsitektur dan serah terima dokumentasi", EN: "Keep satisfied - include in architecture decisions and documentation handover"},
	},
	{
		ID: "S4", Power: 2, Interest: 5, Internal: false,
		Name:     Text{ID: "Alumni (pengguna akhir)", EN: "Alumni (end users)"},
		Need:     Text{ID: "Pengisian kuesioner yang cepat, ringan di kuota, dan bisa lewat ponsel", EN: "Fast, data-light questionnaire that works on a phone"},
		Strategy: Text{ID: "Beri informasi - kampanye sosialisasi, pengingat, dan umpan balik hasil", EN: "Keep informed - outreach campaign, reminders, and result feedback"},
	},
	{
		ID: "S5", Power: 5, Interest: 2, Internal: false,
		Name:     Text{ID: "BAN-PT / asesor akreditasi", EN: "BAN-PT / accreditation assessors"},
		Need:     Text{ID: "Data tracer study yang dapat ditelusuri dan sesuai instrumen resmi", EN: "Traceable tracer study data matching the official instrument"},
		Strategy: Text{ID: "Jaga kepuasan - pastikan format laporan mengikuti instrumen resmi sejak desain", EN: "Keep satisfied - align report format to the official instrument from design onward"},
	},
	{
		ID: "S6", Power: 3, Interest: 4, Internal: true,
		Name:     Text{ID: "Dosen Pembimbing MPPL", EN: "Course supervisor"},
		Need:     Text{ID: "Penerapan metode manajemen proyek yang benar, bukan sekadar aplikasi jadi", EN: "Correct application of project management method, not just a finished app"},
		Strategy: Text{ID: "Kelola erat - tunjukkan artefak CPM, EVM, dan risk register di setiap tugas", EN: "Manage closely - show CPM, EVM, and risk register artefacts in every assignment"},
	},
	{
		ID: "S7", Power: 3, Interest: 2, Internal: false,
		Name:     Text{ID: "Vendor SIAKAD", EN: "SIAKAD vendor"},
		Need:     Text{ID: "Kejelasan lingkup akses data dan beban permintaan ke sistem mereka", EN: "Clarity on data-access scope and load on their system"},
		Strategy: Text{ID: "Pantau - kesepakatan akses data tertulis sebelum fase desain berakhir", EN: "Monitor - written data-access agreement before design closes"},
	},
	{
		ID: "S8", Power: 2, Interest: 2, Internal: false,
		Name:     Text{ID: "Calon mahasiswa & orang tua", EN: "Prospective students & parents"},
		Need:     Text{ID: "Informasi serapan kerja lulusan yang terbuka", EN: "Open information on graduate employment outcomes"},
		Strategy: Text{ID: "Pantau - publikasikan ringkasan agregat setelah data matang", EN: "Monitor - publish aggregate summaries once data matures"},
	},
	{
		ID: "S9", Power: 4, Interest: 4, Internal: true,
		Name:     Text{ID: "Tim proyek (9 peran)", EN: "Project team (9 roles)"},
		Need:     Text{ID: "Beban kerja yang realistis di sela kuliah dan ujian", EN: "Realistic workload alongside classes and exams"},
		Strategy: Text{ID: "Kelola erat - histogram sumber daya mingguan dan penjadwalan ulang saat over-alokasi", EN: "Manage closely - weekly resource histogram and re-plan on over-allocation"},
	},
	{
		ID: "S10", Power: 1, Interest: 3, Internal: false,
		Name:     Text{ID: "Mitra industri penyerap lulusan", EN: "Industry partners hiring graduates"},
		Need:     Text{ID: "Jalur umpan balik terhadap kualitas lulusan", EN: "A feedback channel on graduate quality"},
		Strategy: Text{ID: "Pantau - pertimbangkan modul umpan balik pemberi kerja di fase berikutnya", EN: "Monitor - consider an employer feedback module in a later phase"},
	},
}

// CommsChannel adalah satu baris rencana komunikasi (asal: Tugas 10 bagian 6).
type CommsChannel struct {
	From      Text
	To        Text
	Frequency Text
	Medium    Text
	Purpose   Text
}

// CommsPlan adalah rencana komunikasi proyek.
var CommsPlan = []CommsChannel{
	{
		From: Text{ID: "Project Manager", EN: "Project Manager"}, To: Text{ID: "Project Sponsor", EN: "Project Sponsor"},
		Frequency: Text{ID: "Mingguan", EN: "Weekly"}, Medium: Text{ID: "Email + rapat daring", EN: "Email + online meeting"},
		Purpose: Text{ID: "Status update, keputusan besar, dan persetujuan perubahan", EN: "Status update, major decisions, and change approvals"},
	},
	{
		From: Text{ID: "Project Manager", EN: "Project Manager"}, To: Text{ID: "Core Team (BA, SA, Tech Lead)", EN: "Core Team (BA, SA, Tech Lead)"},
		Frequency: Text{ID: "Harian", EN: "Daily"}, Medium: Text{ID: "Slack / daily standup", EN: "Slack / daily standup"},
		Purpose: Text{ID: "Koordinasi harian dan perencanaan sprint", EN: "Daily coordination and sprint planning"},
	},
	{
		From: Text{ID: "Technical Lead", EN: "Technical Lead"}, To: Text{ID: "Tim pengembang", EN: "Development team"},
		Frequency: Text{ID: "Harian", EN: "Daily"}, Medium: Text{ID: "Slack + pull request", EN: "Slack + pull request"},
		Purpose: Text{ID: "Code review dan koordinasi teknis", EN: "Code review and technical coordination"},
	},
	{
		From: Text{ID: "Tim pengembang", EN: "Development team"}, To: Text{ID: "QA Tester", EN: "QA Tester"},
		Frequency: Text{ID: "Setiap rilis", EN: "Every release"}, Medium: Text{ID: "Issue tracker", EN: "Issue tracker"},
		Purpose: Text{ID: "Laporan bug dan hasil pengujian", EN: "Bug reports and test results"},
	},
	{
		From: Text{ID: "Business Analyst", EN: "Business Analyst"}, To: Text{ID: "Pemangku kepentingan kampus", EN: "Campus stakeholders"},
		Frequency: Text{ID: "Dua mingguan", EN: "Bi-weekly"}, Medium: Text{ID: "Rapat tatap muka", EN: "Face-to-face meeting"},
		Purpose: Text{ID: "Klarifikasi kebutuhan dan sesi UAT", EN: "Requirement clarification and UAT sessions"},
	},
	{
		From: Text{ID: "Project Manager", EN: "Project Manager"}, To: Text{ID: "Seluruh tim", EN: "Whole team"},
		Frequency: Text{ID: "Mingguan", EN: "Weekly"}, Medium: Text{ID: "Rapat tim", EN: "Team meeting"},
		Purpose: Text{ID: "Laporan progres dan penyelarasan tim", EN: "Progress report and team alignment"},
	},
}

// LifecyclePhase adalah satu tahap siklus hidup proyek (asal: Tugas 3).
type LifecyclePhase struct {
	Key        string
	Name       Text
	Goal       Text
	Activities []Text
	Outputs    Text
	Watch      Text
	WBSCodes   []string
}

// Lifecycle adalah empat tahap siklus hidup: konsepsi, perencanaan, eksekusi,
// operasi - kerangka yang dipakai pada Tugas 3.
var Lifecycle = []LifecyclePhase{
	{
		Key: "konsepsi", WBSCodes: []string{},
		Name: Text{ID: "Tahap Konsepsi", EN: "Conception"},
		Goal: Text{ID: "Menetapkan alasan bisnis dan memutuskan apakah proyek layak dilanjutkan", EN: "Establish the business case and decide whether the project should proceed"},
		Activities: []Text{
			{ID: "Pengumpulan informasi awal dari akademik, bagian alumni, dan unit IT", EN: "Gather initial information from academic, alumni office, and IT unit"},
			{ID: "Identifikasi masalah pada proses pendataan alumni yang berjalan", EN: "Identify problems in the current alumni data process"},
			{ID: "Analisis kelayakan teknis, biaya, dan operasional", EN: "Technical, cost, and operational feasibility analysis"},
			{ID: "Penunjukan sponsor dan pemangku kepentingan utama", EN: "Appoint the sponsor and key stakeholders"},
			{ID: "Penyusunan Piagam Proyek", EN: "Draft the Project Charter"},
		},
		Outputs: Text{ID: "Piagam Proyek, daftar pemangku kepentingan, ringkasan studi kelayakan, rekomendasi go / no-go", EN: "Project Charter, stakeholder register, feasibility summary, go / no-go recommendation"},
		Watch:   Text{ID: "Libatkan perwakilan akademik sejak awal agar kebutuhan akreditasi tidak terlewat", EN: "Involve academic representatives early so accreditation needs are not missed"},
	},
	{
		Key: "perencanaan", WBSCodes: []string{"1.0"},
		Name: Text{ID: "Tahap Perencanaan", EN: "Planning"},
		Goal: Text{ID: "Menyusun rencana terperinci sebagai pedoman pelaksanaan dan pengendalian", EN: "Produce a detailed plan to guide execution and control"},
		Activities: []Text{
			{ID: "Penyusunan ruang lingkup rinci dan WBS sampai level yang dapat diestimasi", EN: "Detail the scope and decompose the WBS to an estimable level"},
			{ID: "Penyusunan jadwal dan identifikasi jalur kritis", EN: "Build the schedule and identify the critical path"},
			{ID: "Estimasi anggaran termasuk cadangan risiko", EN: "Estimate the budget including risk reserves"},
			{ID: "Perancangan arsitektur awal, basis data, dan sketsa antarmuka", EN: "Draft the architecture, database, and interface sketches"},
			{ID: "Penyusunan rencana kualitas, komunikasi, risiko, dan pengadaan", EN: "Write the quality, communication, risk, and procurement plans"},
		},
		Outputs: Text{ID: "Rencana Manajemen Proyek lengkap, WBS, daftar persyaratan awal", EN: "Complete Project Management Plan, WBS, initial requirements list"},
		Watch:   Text{ID: "Libatkan orang yang mengerjakan saat mengestimasi durasi untuk mengurangi bias", EN: "Involve the people who do the work when estimating, to reduce bias"},
	},
	{
		Key: "eksekusi", WBSCodes: []string{"2.0", "3.0", "4.0"},
		Name: Text{ID: "Tahap Eksekusi", EN: "Execution"},
		Goal: Text{ID: "Menghasilkan perangkat lunak sesuai lingkup dan kriteria kualitas yang disepakati", EN: "Deliver software meeting the agreed scope and quality criteria"},
		Activities: []Text{
			{ID: "Desain teknis rinci termasuk arsitektur integrasi dengan sistem akademik", EN: "Detailed technical design including academic-system integration architecture"},
			{ID: "Pembuatan prototipe antarmuka dan validasi bersama pemangku kepentingan", EN: "Build interface prototypes and validate with stakeholders"},
			{ID: "Pengembangan modul inti: registrasi, kuesioner, dasbor, panel admin", EN: "Develop core modules: registration, questionnaire, dashboard, admin panel"},
			{ID: "Pengujian berjenjang: unit, integrasi, sistem, dan UAT", EN: "Layered testing: unit, integration, system, and UAT"},
			{ID: "Pelatihan administrator dan pengguna kunci", EN: "Train administrators and key users"},
		},
		Outputs: Text{ID: "Perangkat lunak teruji, laporan pengujian, dokumentasi lengkap, bukti pelatihan", EN: "Tested software, test reports, complete documentation, training evidence"},
		Watch:   Text{ID: "Prioritaskan keamanan data, mekanisme backup, dan audit log sebelum deploy", EN: "Prioritise data security, backup, and audit logging before deployment"},
	},
	{
		Key: "operasi", WBSCodes: []string{"5.0"},
		Name: Text{ID: "Tahap Operasi", EN: "Operation"},
		Goal: Text{ID: "Menjalankan sistem di produksi, memastikan adopsi, dan memelihara secara berkelanjutan", EN: "Run the system in production, drive adoption, and maintain it"},
		Activities: []Text{
			{ID: "Deploy ke lingkungan produksi dan verifikasi pasca deploy", EN: "Deploy to production and verify afterwards"},
			{ID: "Sosialisasi sistem dan kampanye partisipasi kepada alumni", EN: "Socialise the system and campaign for alumni participation"},
			{ID: "Pemantauan performa, uptime, dan keamanan", EN: "Monitor performance, uptime, and security"},
			{ID: "Evaluasi keberhasilan proyek dan penyusunan lessons learned", EN: "Evaluate project success and capture lessons learned"},
			{ID: "Serah terima dokumentasi kepada unit IT kampus", EN: "Hand over documentation to the campus IT unit"},
		},
		Outputs: Text{ID: "Sistem live, laporan evaluasi, dokumentasi operasional, lessons learned", EN: "Live system, evaluation report, operations documentation, lessons learned"},
		Watch:   Text{ID: "Partisipasi alumni hanya tumbuh bila kampanye berjalan; jangan anggap selesai di go-live", EN: "Alumni participation only grows with an active campaign; go-live is not the finish line"},
	},
}

// KnowledgeArea adalah satu dari sembilan area pengetahuan pada Modul 2 MPPL.
type KnowledgeArea struct {
	No       int
	Name     Text
	Content  Text
	Evidence Text   // artefak di situs ini yang membuktikan area tersebut dikerjakan
	Route    string // tautan ke halaman bukti
}

// KnowledgeAreas adalah sembilan area pengetahuan sesuai Modul 2 dan Tugas 2.
//
// Catatan: PMBOK edisi ke-5 memekarkan daftar ini menjadi sepuluh dengan
// menambahkan Project Stakeholder Management. Materi kuliah memakai sembilan,
// jadi sembilan yang dipakai di sini - area kesepuluh tetap dikerjakan dan
// buktinya ada di halaman Pemangku Kepentingan.
var KnowledgeAreas = []KnowledgeArea{
	{
		No: 1, Route: "/wbs/",
		Name:     Text{ID: "Project Scope Management", EN: "Project Scope Management"},
		Content:  Text{ID: "Perancangan, pengembangan, pengujian, dan implementasi aplikasi web tracer study; dipecah menjadi 5 fase, 15 paket kerja, dan 35 aktivitas.", EN: "Design, development, testing, and implementation of the tracer study web app; decomposed into 5 phases, 15 work packages, and 35 activities."},
		Evidence: Text{ID: "WBS lengkap dengan pemeriksaan aturan 100% dan kamus paket kerja", EN: "Full WBS with 100%-rule check and work package dictionary"},
	},
	{
		No: 2, Route: "/jadwal/",
		Name:     Text{ID: "Project Time Management", EN: "Project Time Management"},
		Content:  Text{ID: "Durasi 17 minggu, dibagi analisis 3, desain 3, pengembangan 7, uji coba 2, dan implementasi 2 minggu.", EN: "17 weeks total: 3 analysis, 3 design, 7 development, 2 testing, 2 implementation."},
		Evidence: Text{ID: "Gantt chart dengan jalur kritis, tabel ES/EF/LS/LF, dan analisis float", EN: "Gantt chart with critical path, ES/EF/LS/LF table, and float analysis"},
	},
	{
		No: 3, Route: "/biaya/",
		Name:     Text{ID: "Project Cost Management", EN: "Project Cost Management"},
		Content:  Text{ID: "Pagu Rp 16 juta untuk server & hosting, insentif tim, pelatihan, tools, serta cadangan.", EN: "IDR 16m covering server & hosting, team incentives, training, tools, and reserves."},
		Evidence: Text{ID: "Earned Value lengkap: PV/EV/AC, SPI, CPI, EAC tiga varian, dan TCPI", EN: "Full Earned Value: PV/EV/AC, SPI, CPI, three EAC variants, and TCPI"},
	},
	{
		No: 4, Route: "/organisasi/",
		Name:     Text{ID: "Project Human Resource Management", EN: "Project Human Resource Management"},
		Content:  Text{ID: "Sembilan peran inti dari Project Manager sampai DevOps, ditambah sponsor dan pemangku kepentingan.", EN: "Nine core roles from Project Manager to DevOps, plus sponsor and stakeholders."},
		Evidence: Text{ID: "Bagan organisasi, matriks RACI tervalidasi, dan histogram beban sumber daya", EN: "Org chart, validated RACI matrix, and resource loading histogram"},
	},
	{
		No: 5, Route: "/risiko/",
		Name:     Text{ID: "Project Risk Management", EN: "Project Risk Management"},
		Content:  Text{ID: "Risiko adopsi alumni, keterlambatan, bug, keamanan data, dan keterbatasan anggaran.", EN: "Risks of alumni adoption, delay, bugs, data security, and budget limits."},
		Evidence: Text{ID: "Risk register 12 entri dengan EMV, peta panas 5x5, dan perhitungan kecukupan cadangan", EN: "12-entry risk register with EMV, 5x5 heat map, and reserve adequacy calculation"},
	},
	{
		No: 6, Route: "/pemangku-kepentingan/",
		Name:     Text{ID: "Project Communication Management", EN: "Project Communication Management"},
		Content:  Text{ID: "Komunikasi rutin via WhatsApp/Slack, laporan progres mingguan, rapat evaluasi bulanan.", EN: "Routine WhatsApp/Slack communication, weekly progress reports, monthly evaluation meetings."},
		Evidence: Text{ID: "Rencana komunikasi enam jalur dan grid kuasa-kepentingan sepuluh pemangku kepentingan", EN: "Six-channel communication plan and ten-stakeholder power-interest grid"},
	},
	{
		No: 7, Route: "/kualitas/",
		Name:     Text{ID: "Project Quality Management", EN: "Project Quality Management"},
		Content:  Text{ID: "Ramah pengguna, responsif, data terenkripsi, ekspor Excel/PDF, downtime minimal.", EN: "User-friendly, responsive, encrypted data, Excel/PDF export, minimal downtime."},
		Evidence: Text{ID: "Control chart dengan aturan Nelson, diagram Pareto, dan biaya kualitas", EN: "Control chart with Nelson rules, Pareto diagram, and cost of quality"},
	},
	{
		No: 8, Route: "/biaya/",
		Name:     Text{ID: "Project Procurement Management", EN: "Project Procurement Management"},
		Content:  Text{ID: "Pengadaan server hosting & domain, lisensi tools, dan jasa pelatihan admin.", EN: "Procurement of hosting & domain, tool licences, and admin training services."},
		Evidence: Text{ID: "Rincian biaya non-tenaga-kerja per aktivitas dengan kategori anggaran", EN: "Non-labour cost breakdown per activity with budget category"},
	},
	{
		No: 9, Route: "/",
		Name:     Text{ID: "Project Integration Management", EN: "Project Integration Management"},
		Content:  Text{ID: "Integrasi sistem tracer study dengan SIAKAD dan penyelarasan laporan untuk BAN-PT.", EN: "Integrating the tracer study system with SIAKAD and aligning reports for BAN-PT."},
		Evidence: Text{ID: "Ruang kendali yang menyatukan jadwal, biaya, risiko, dan mutu dalam satu model data", EN: "A control tower unifying schedule, cost, risk, and quality in one data model"},
	},
}
