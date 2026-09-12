package model

// Formula adalah satu rumus manajemen proyek beserta segala yang dibutuhkan
// untuk memakainya dengan benar: notasi, arti tiap simbol, cara membacanya,
// jebakan yang umum, dan contoh hitung dari data proyek ini.
//
// Bidang Example TIDAK diisi di berkas ini. Contoh hitungnya disuntikkan saat
// perakitan situs dari angka yang benar-benar dihitung engine, supaya contoh
// di halaman rumus tidak pernah bisa berbeda dari angka di halaman lain.
type Formula struct {
	Key      string
	Group    string
	Name     Text
	Notation Text // notasi matematis; sebagian memuat kata, jadi tetap dwibahasa
	Symbols  []Symbol
	Meaning  Text
	Reading  Text
	Pitfall  Text
	Source   Text // rujukan materi kuliah
	Route    string
}

// Symbol adalah keterangan satu simbol dalam rumus.
type Symbol struct {
	Sym  string
	Desc Text
}

// FormulaGroups adalah urutan kelompok rumus di halaman referensi.
var FormulaGroups = []struct {
	Key  string
	Name Text
	Desc Text
}{
	{"jadwal", Text{ID: "Penjadwalan & Jalur Kritis", EN: "Scheduling & Critical Path"},
		Text{ID: "Menentukan kapan setiap aktivitas paling cepat dan paling lambat boleh berjalan, serta mana yang tidak boleh molor sedetik pun.", EN: "Determining how early and how late each activity may run, and which ones cannot slip at all."}},
	{"pert", Text{ID: "Estimasi Tiga Titik & PERT", EN: "Three-Point Estimation & PERT"},
		Text{ID: "Mengubah ketidakpastian durasi dari tebakan tunggal menjadi sebaran yang bisa dihitung peluangnya.", EN: "Turning duration uncertainty from a single guess into a distribution whose probability can be computed."}},
	{"evm", Text{ID: "Earned Value Management", EN: "Earned Value Management"},
		Text{ID: "Menyatukan lingkup, jadwal, dan biaya dalam satu satuan - rupiah - sehingga ketiganya bisa dibandingkan langsung.", EN: "Uniting scope, schedule, and cost in one unit - money - so all three can be compared directly."}},
	{"risiko", Text{ID: "Risiko Kuantitatif", EN: "Quantitative Risk"},
		Text{ID: "Mengubah daftar kekhawatiran menjadi angka rupiah yang bisa dipakai menetapkan besar cadangan.", EN: "Turning a list of worries into monetary figures usable for sizing reserves."}},
	{"mutu", Text{ID: "Pengendalian Mutu", EN: "Quality Control"},
		Text{ID: "Membedakan variasi wajar dari sinyal bahwa sesuatu benar-benar berubah.", EN: "Separating ordinary variation from a signal that something has genuinely changed."}},
	{"sumberdaya", Text{ID: "Sumber Daya", EN: "Resources"},
		Text{ID: "Memeriksa apakah jadwal yang tampak rapi benar-benar bisa dikerjakan oleh orang yang ada.", EN: "Checking whether a tidy-looking schedule can actually be delivered by the people available."}},
}

// Formulas adalah seluruh rumus yang dipakai aplikasi ini. Tidak ada rumus di
// sini yang tidak benar-benar dijalankan enginenya - daftar ini adalah
// dokumentasi kode yang berjalan, bukan ringkasan buku teks.
var Formulas = []Formula{
	// ------------------------------------------------------------ jadwal
	{
		Key: "ef", Group: "jadwal", Route: "/jadwal/",
		Name:     Text{ID: "Early Finish (forward pass)", EN: "Early Finish (forward pass)"},
		Notation: Text{ID: "EF = ES + d - 1     ES = max(EF_pendahulu) + 1 + lag", EN: "EF = ES + d - 1     ES = max(EF_predecessor) + 1 + lag"},
		Symbols: []Symbol{
			{"ES", Text{ID: "hari kerja paling awal aktivitas boleh mulai", EN: "earliest working day the activity may start"}},
			{"EF", Text{ID: "hari kerja paling awal aktivitas selesai", EN: "earliest working day the activity finishes"}},
			{"d", Text{ID: "durasi aktivitas dalam hari kerja", EN: "activity duration in working days"}},
			{"lag", Text{ID: "jeda wajib antar aktivitas; negatif berarti lead", EN: "mandatory delay between activities; negative means lead"}},
		},
		Meaning: Text{ID: "Forward pass menyusuri jaringan dari awal ke akhir dan menjawab: sepagi apa setiap pekerjaan bisa dimulai kalau semuanya lancar.", EN: "The forward pass walks the network start to finish and answers: how early can each piece of work begin if everything goes smoothly."},
		Reading: Text{ID: "Ambil EF terbesar dari seluruh pendahulu - satu pendahulu yang telat sudah cukup menahan aktivitas penerusnya.", EN: "Take the largest EF across all predecessors - one late predecessor is enough to hold up the successor."},
		Pitfall: Text{ID: "Angka -1 dan +1 di rumus adalah akibat batas inklusif. Milestone berdurasi nol akan kacau kalau aritmetikanya tidak dipindah ke batas eksklusif; itulah sebabnya engine ini menyimpan dua bentuk.", EN: "The -1 and +1 come from inclusive bounds. Zero-duration milestones break unless the arithmetic moves to exclusive bounds, which is why this engine keeps both forms."},
		Source:  Text{ID: "Modul 4 MPPL - Perencanaan Waktu Proyek", EN: "MPPL Module 4 - Project Time Planning"},
	},
	{
		Key: "ls", Group: "jadwal", Route: "/jadwal/",
		Name:     Text{ID: "Late Start (backward pass)", EN: "Late Start (backward pass)"},
		Notation: Text{ID: "LF = min(LS_penerus) - 1 - lag     LS = LF - d + 1", EN: "LF = min(LS_successor) - 1 - lag     LS = LF - d + 1"},
		Symbols: []Symbol{
			{"LS", Text{ID: "hari kerja paling lambat aktivitas boleh mulai tanpa menunda proyek", EN: "latest start that does not delay the project"}},
			{"LF", Text{ID: "hari kerja paling lambat aktivitas boleh selesai", EN: "latest finish that does not delay the project"}},
		},
		Meaning: Text{ID: "Backward pass menyusuri jaringan dari akhir ke awal dan menjawab: selambat apa setiap pekerjaan boleh dimulai sebelum tanggal akhir proyek ikut bergeser.", EN: "The backward pass walks the network finish to start and answers: how late may each piece of work begin before the project end date moves."},
		Reading: Text{ID: "Ambil LS terkecil dari seluruh penerus - aktivitas harus selesai sebelum penerus paling mendesak butuh memulai.", EN: "Take the smallest LS across all successors - the activity must finish before the most urgent successor needs to start."},
		Pitfall: Text{ID: "Aktivitas tanpa penerus memakai akhir proyek sebagai LF. Lupa menangani kasus ini membuat float-nya tak hingga dan jalur kritisnya hilang.", EN: "Activities with no successor take the project finish as their LF. Missing this case makes their float infinite and the critical path disappears."},
		Source:  Text{ID: "Modul 4 MPPL - Critical Path Method", EN: "MPPL Module 4 - Critical Path Method"},
	},
	{
		Key: "float", Group: "jadwal", Route: "/jadwal/",
		Name:     Text{ID: "Total float & free float", EN: "Total float & free float"},
		Notation: Text{ID: "TF = LS - ES = LF - EF     FF = min(ES_penerus) - EF - 1", EN: "TF = LS - ES = LF - EF     FF = min(ES_successor) - EF - 1"},
		Symbols: []Symbol{
			{"TF", Text{ID: "berapa lama aktivitas boleh molor tanpa menunda PROYEK", EN: "how long the activity may slip without delaying the PROJECT"}},
			{"FF", Text{ID: "berapa lama aktivitas boleh molor tanpa menggeser PENERUSNYA", EN: "how long it may slip without moving its SUCCESSOR"}},
		},
		Meaning: Text{ID: "Float adalah ruang gerak. Aktivitas dengan TF = 0 berada di jalur kritis: molor sehari berarti proyek molor sehari.", EN: "Float is room to manoeuvre. An activity with TF = 0 sits on the critical path: slip a day and the project slips a day."},
		Reading: Text{ID: "Free float selalu lebih kecil atau sama dengan total float. Memakai free float aman bagi tetangga; memakai total float memindahkan tekanan ke aktivitas lain di jalur yang sama.", EN: "Free float is always less than or equal to total float. Spending free float is safe for neighbours; spending total float pushes the pressure onto others on the same path."},
		Pitfall: Text{ID: "Float milik JALUR, bukan milik aktivitas. Kalau dua aktivitas berbagi jalur ber-float 4 hari, keduanya tidak boleh memakai 4 hari masing-masing.", EN: "Float belongs to the PATH, not the activity. Two activities sharing a 4-day float path cannot each spend 4 days."},
		Source:  Text{ID: "Modul 4 MPPL", EN: "MPPL Module 4"},
	},

	// -------------------------------------------------------------- PERT
	{
		Key: "te", Group: "pert", Route: "/pert/",
		Name:     Text{ID: "Durasi harapan PERT", EN: "PERT expected duration"},
		Notation: Text{ID: "te = (O + 4M + P) / 6", EN: "te = (O + 4M + P) / 6"},
		Symbols: []Symbol{
			{"O", Text{ID: "estimasi optimistis - kalau semuanya berjalan lancar", EN: "optimistic estimate - if everything goes well"}},
			{"M", Text{ID: "estimasi paling mungkin - yang akan Anda sebut kalau ditanya sekali", EN: "most likely estimate - what you would say if asked once"}},
			{"P", Text{ID: "estimasi pesimistis - kalau hal-hal yang wajar meleset", EN: "pessimistic estimate - if reasonable things go wrong"}},
		},
		Meaning: Text{ID: "Rata-rata berbobot yang memberi bobot empat kali lipat pada nilai paling mungkin, mendekati rerata distribusi beta.", EN: "A weighted average giving four times the weight to the most likely value, approximating the mean of a beta distribution."},
		Reading: Text{ID: "Bila P jauh lebih jauh dari M ketimbang O, te akan LEBIH BESAR dari M. Itu bukan kesalahan hitung - itu cerminan bahwa pekerjaan perangkat lunak lebih sering meleset ke arah lama daripada ke arah cepat.", EN: "When P sits much further from M than O does, te comes out LARGER than M. That is not an arithmetic error - it reflects that software work misses towards slower far more often than towards faster."},
		Pitfall: Text{ID: "te menjumlahkan sepanjang jalur kritis, tetapi jalur kritis bisa berpindah ketika durasi berubah. PERT tidak melihat perpindahan itu; Monte Carlo melihatnya.", EN: "te sums along the critical path, but the critical path can move when durations change. PERT cannot see that shift; Monte Carlo can."},
		Source:  Text{ID: "Modul 4 MPPL - Program Evaluation and Review Technique", EN: "MPPL Module 4 - Program Evaluation and Review Technique"},
	},
	{
		Key: "sigma", Group: "pert", Route: "/pert/",
		Name:     Text{ID: "Simpangan baku & varians PERT", EN: "PERT standard deviation & variance"},
		Notation: Text{ID: "sigma = (P - O) / 6     var = sigma^2     sigma_proyek = akar(jumlah var di jalur kritis)", EN: "sigma = (P - O) / 6     var = sigma^2     sigma_project = sqrt(sum of var along the critical path)"},
		Symbols: []Symbol{
			{"sigma", Text{ID: "simpangan baku durasi satu aktivitas", EN: "standard deviation of one activity duration"}},
			{"var", Text{ID: "varians - kuadrat simpangan baku, satu-satunya bentuk yang boleh dijumlahkan", EN: "variance - the squared deviation, the only form that may be summed"}},
		},
		Meaning: Text{ID: "Rentang O sampai P dianggap mencakup enam simpangan baku, sehingga sigma adalah seperenam rentang itu.", EN: "The O-to-P range is taken to span six standard deviations, so sigma is one sixth of that range."},
		Reading: Text{ID: "Varians boleh dijumlahkan, simpangan baku tidak. Menjumlahkan sigma langsung akan melebih-lebihkan ketidakpastian proyek secara serius.", EN: "Variances add; standard deviations do not. Summing sigma directly badly overstates project uncertainty."},
		Pitfall: Text{ID: "Penjumlahan varians mengandaikan durasi antar-aktivitas saling bebas. Kalau satu pengembang mengerjakan tiga aktivitas berurutan, asumsi itu tidak berlaku.", EN: "Summing variance assumes durations are independent. If one developer handles three consecutive activities, that assumption fails."},
		Source:  Text{ID: "Modul 4 MPPL", EN: "MPPL Module 4"},
	},
	{
		Key: "zscore", Group: "pert", Route: "/pert/",
		Name:     Text{ID: "Peluang selesai pada tanggal target", EN: "Probability of finishing by a target date"},
		Notation: Text{ID: "Z = (target - te_proyek) / sigma_proyek     P = Phi(Z)", EN: "Z = (target - te_project) / sigma_project     P = Phi(Z)"},
		Symbols: []Symbol{
			{"Z", Text{ID: "jarak target dari durasi harapan, dalam satuan simpangan baku", EN: "distance of the target from expected duration, in standard deviations"}},
			{"Phi", Text{ID: "fungsi distribusi kumulatif normal baku", EN: "standard normal cumulative distribution function"}},
		},
		Meaning: Text{ID: "Teorema Limit Pusat mengizinkan jumlah banyak durasi acak didekati distribusi normal, sehingga peluangnya bisa dibaca dari tabel Z.", EN: "The Central Limit Theorem lets the sum of many random durations be approximated as normal, so probability can be read from a Z table."},
		Reading: Text{ID: "Z negatif berarti target berada di bawah durasi harapan, dan peluangnya di bawah lima puluh persen.", EN: "A negative Z means the target sits below expected duration and the odds are under fifty percent."},
		Pitfall: Text{ID: "Hanya jalur kritis yang dihitung. Jalur lain dengan float tipis dan ketidakpastian besar sama sekali tidak terlihat - inilah alasan angka PERT sering lebih optimistis daripada Monte Carlo.", EN: "Only the critical path is counted. Other paths with thin float and large uncertainty stay invisible - which is why PERT often reads more optimistic than Monte Carlo."},
		Source:  Text{ID: "Modul 4 MPPL", EN: "MPPL Module 4"},
	},
	{
		Key: "montecarlo", Group: "pert", Route: "/pert/",
		Name:     Text{ID: "Simulasi Monte Carlo jadwal", EN: "Monte Carlo schedule simulation"},
		Notation: Text{ID: "untuk i = 1..N:  d_j ~ BetaPERT(O_j, M_j, P_j)  lalu  T_i = CPM(d)     P(x) = |{T_i <= x}| / N", EN: "for i = 1..N:  d_j ~ BetaPERT(O_j, M_j, P_j)  then  T_i = CPM(d)     P(x) = |{T_i <= x}| / N"},
		Symbols: []Symbol{
			{"N", Text{ID: "jumlah iterasi; aplikasi ini memakai sepuluh ribu", EN: "number of iterations; this application uses ten thousand"}},
			{"d_j", Text{ID: "durasi aktivitas ke-j yang diambil acak dari sebarannya", EN: "duration of activity j sampled from its distribution"}},
			{"T_i", Text{ID: "durasi total proyek pada iterasi ke-i", EN: "total project duration in iteration i"}},
		},
		Meaning: Text{ID: "Alih-alih menjumlahkan varians di satu jalur, seluruh jaringan dijalankan ulang ribuan kali dengan durasi acak. Jalur kritis dibiarkan berpindah sendiri.", EN: "Instead of summing variance along one path, the whole network is re-run thousands of times with random durations. The critical path is allowed to move on its own."},
		Reading: Text{ID: "Bacalah P80, bukan rerata. P80 berarti delapan dari sepuluh kemungkinan selesai pada atau sebelum angka itu - inilah bentuk janji yang layak disampaikan ke sponsor.", EN: "Read P80, not the mean. P80 means eight of ten futures finish on or before that figure - the kind of promise worth making to a sponsor."},
		Pitfall: Text{ID: "Generator acak harus berbenih tetap. Tanpa itu, angka di dasbor berubah setiap kali halaman dimuat dan tidak ada yang bisa memverifikasinya.", EN: "The random generator must be seeded. Without that, dashboard figures shift on every page load and nobody can verify them."},
		Source:  Text{ID: "Modul 4 MPPL - bagian simulasi", EN: "MPPL Module 4 - simulation section"},
	},
	{
		Key: "spearman", Group: "pert", Route: "/pert/",
		Name:     Text{ID: "Sensitivitas peringkat (diagram tornado)", EN: "Rank sensitivity (tornado diagram)"},
		Notation: Text{ID: "rho = kovarians(peringkat d_j, peringkat T) / (sd_peringkat_d * sd_peringkat_T)", EN: "rho = cov(rank d_j, rank T) / (sd_rank_d * sd_rank_T)"},
		Symbols: []Symbol{
			{"rho", Text{ID: "korelasi peringkat Spearman, bernilai -1 sampai +1", EN: "Spearman rank correlation, ranging -1 to +1"}},
		},
		Meaning: Text{ID: "Mengukur seberapa kuat durasi satu aktivitas menggerakkan durasi total proyek di seluruh iterasi simulasi.", EN: "Measures how strongly one activity's duration moves total project duration across all simulation iterations."},
		Reading: Text{ID: "Aktivitas dengan rho tertinggi adalah tempat terbaik menaruh perhatian manajemen - bukan aktivitas yang paling mahal atau paling lama.", EN: "The activity with the highest rho is where management attention pays best - not the costliest or longest one."},
		Pitfall: Text{ID: "Korelasi Pearson keliru di sini karena hubungannya tidak linear: aktivitas non-kritis tidak berpengaruh sama sekali sampai float-nya habis, lalu berpengaruh penuh.", EN: "Pearson correlation is wrong here because the relationship is non-linear: a non-critical activity has no effect until its float runs out, then full effect."},
		Source:  Text{ID: "Praktik analisis risiko kuantitatif", EN: "Quantitative risk analysis practice"},
	},

	// --------------------------------------------------------------- EVM
	{
		Key: "pvevac", Group: "evm", Route: "/biaya/",
		Name:     Text{ID: "Tiga besaran pokok Earned Value", EN: "The three Earned Value quantities"},
		Notation: Text{ID: "PV = anggaran x kemajuan_rencana     EV = anggaran x kemajuan_aktual     AC = biaya yang benar-benar keluar", EN: "PV = budget x planned_progress     EV = budget x actual_progress     AC = cost actually incurred"},
		Symbols: []Symbol{
			{"PV", Text{ID: "Planned Value - nilai pekerjaan yang SEHARUSNYA selesai", EN: "Planned Value - work that SHOULD be done"}},
			{"EV", Text{ID: "Earned Value - nilai pekerjaan yang NYATANYA selesai", EN: "Earned Value - work that IS done"}},
			{"AC", Text{ID: "Actual Cost - uang yang NYATANYA keluar", EN: "Actual Cost - money actually spent"}},
		},
		Meaning: Text{ID: "Ketiganya diukur dalam rupiah pada satu tanggal data yang sama, sehingga kemajuan dan pengeluaran bisa dibandingkan langsung tanpa satuan yang berbeda.", EN: "All three are measured in money at the same data date, so progress and spending compare directly without mismatched units."},
		Reading: Text{ID: "EV adalah kuncinya. Tanpa EV, orang hanya bisa membandingkan uang keluar dengan uang dianggarkan - dan itu tidak memberi tahu apa pun tentang seberapa banyak pekerjaan yang sudah jadi.", EN: "EV is the key. Without it you can only compare money spent against money budgeted - which says nothing about how much work exists."},
		Pitfall: Text{ID: "Aturan pengukuran kemajuan (linear, 0/100, 50/50) harus disepakati di awal. Mengganti aturan di tengah proyek membuat seluruh deret EV tidak bisa dibandingkan.", EN: "The progress measurement rule (linear, 0/100, 50/50) must be agreed up front. Changing it mid-project makes the whole EV series incomparable."},
		Source:  Text{ID: "Pertemuan 9 MPPL & PMBOK", EN: "MPPL Session 9 & PMBOK"},
	},
	{
		Key: "varians", Group: "evm", Route: "/biaya/",
		Name:     Text{ID: "Varians jadwal & varians biaya", EN: "Schedule variance & cost variance"},
		Notation: Text{ID: "SV = EV - PV     CV = EV - AC", EN: "SV = EV - PV     CV = EV - AC"},
		Symbols: []Symbol{
			{"SV", Text{ID: "varians jadwal dalam rupiah; negatif berarti tertinggal", EN: "schedule variance in money; negative means behind"}},
			{"CV", Text{ID: "varians biaya dalam rupiah; negatif berarti boros", EN: "cost variance in money; negative means overspent"}},
		},
		Meaning: Text{ID: "Selisih absolut antara yang diperoleh dengan yang direncanakan dan dengan yang dibelanjakan.", EN: "The absolute gap between what was earned versus what was planned and what was spent."},
		Reading: Text{ID: "Keduanya negatif berarti proyek tertinggal sekaligus boros - kombinasi paling berbahaya, karena mengejar jadwal biasanya menambah biaya.", EN: "Both negative means behind and overspent at once - the most dangerous combination, because catching up usually costs more."},
		Pitfall: Text{ID: "SV dalam rupiah selalu kembali ke nol saat proyek selesai, betapapun telatnya. Untuk keterlambatan yang jujur, pakai Earned Schedule.", EN: "SV in money always returns to zero at completion no matter how late. For an honest delay figure, use Earned Schedule."},
		Source:  Text{ID: "Pertemuan 9 MPPL", EN: "MPPL Session 9"},
	},
	{
		Key: "indeks", Group: "evm", Route: "/biaya/",
		Name:     Text{ID: "Indeks kinerja jadwal & biaya", EN: "Schedule & cost performance indices"},
		Notation: Text{ID: "SPI = EV / PV     CPI = EV / AC", EN: "SPI = EV / PV     CPI = EV / AC"},
		Symbols: []Symbol{
			{"SPI", Text{ID: "di bawah 1 berarti lebih lambat dari rencana", EN: "below 1 means slower than planned"}},
			{"CPI", Text{ID: "di bawah 1 berarti tiap rupiah menghasilkan kurang dari satu rupiah pekerjaan", EN: "below 1 means each unit of money buys less than one unit of work"}},
		},
		Meaning: Text{ID: "Bentuk rasio dari kedua varians, sehingga bisa dibandingkan antar proyek dengan ukuran anggaran berbeda.", EN: "The ratio form of both variances, letting projects of different budget sizes be compared."},
		Reading: Text{ID: "CPI adalah indikator paling stabil dalam manajemen proyek: setelah sekitar dua puluh persen pekerjaan selesai, CPI jarang membaik dengan sendirinya.", EN: "CPI is the most stable indicator in project management: past roughly twenty percent complete, it rarely improves on its own."},
		Pitfall: Text{ID: "SPI mendekati 1 di akhir proyek karena PV dan EV sama-sama menuju BAC. Membaca SPI di akhir proyek nyaris tidak berguna.", EN: "SPI approaches 1 near completion because PV and EV both converge on BAC. Reading SPI late in a project is close to useless."},
		Source:  Text{ID: "Pertemuan 9 MPPL", EN: "MPPL Session 9"},
	},
	{
		Key: "eac", Group: "evm", Route: "/biaya/",
		Name:     Text{ID: "Tiga varian Estimate at Completion", EN: "Three Estimate at Completion variants"},
		Notation: Text{ID: "EAC_1 = AC + (BAC - EV)     EAC_2 = BAC / CPI     EAC_3 = AC + (BAC - EV) / (CPI x SPI)", EN: "EAC_1 = AC + (BAC - EV)     EAC_2 = BAC / CPI     EAC_3 = AC + (BAC - EV) / (CPI x SPI)"},
		Symbols: []Symbol{
			{"EAC_1", Text{ID: "varians dianggap kejadian sekali, sisa pekerjaan berjalan sesuai rencana", EN: "variance treated as one-off; remaining work runs to plan"}},
			{"EAC_2", Text{ID: "varians biaya dianggap terus berlanjut - varian paling sering dipakai", EN: "cost variance assumed to continue - the most commonly used variant"}},
			{"EAC_3", Text{ID: "telat DAN boros sama-sama berlanjut; skenario paling pesimistis", EN: "late AND overspent both continue; the most pessimistic case"}},
		},
		Meaning: Text{ID: "Perkiraan total biaya proyek saat selesai. Ketiganya sah; yang membedakan adalah asumsi tentang masa depan.", EN: "Forecast total cost at completion. All three are valid; what differs is the assumption about the future."},
		Reading: Text{ID: "Laporkan ketiganya sebagai rentang, bukan satu angka. Rentang memaksa pembaca menghadapi ketidakpastian ketimbang berpegang pada satu angka yang kebetulan enak dibaca.", EN: "Report all three as a range, not one number. A range forces the reader to confront uncertainty instead of clinging to whichever figure reads best."},
		Pitfall: Text{ID: "EAC_2 mengandaikan CPI stabil. Pada proyek yang baru berjalan di bawah dua puluh persen, CPI masih sangat berisik dan EAC-nya belum bisa dipercaya.", EN: "EAC_2 assumes a stable CPI. Below roughly twenty percent complete, CPI is still noisy and its EAC is not yet trustworthy."},
		Source:  Text{ID: "PMBOK, dibahas di Pertemuan 9 MPPL", EN: "PMBOK, covered in MPPL Session 9"},
	},
	{
		Key: "tcpi", Group: "evm", Route: "/biaya/",
		Name:     Text{ID: "To-Complete Performance Index", EN: "To-Complete Performance Index"},
		Notation: Text{ID: "TCPI = (BAC - EV) / (BAC - AC)", EN: "TCPI = (BAC - EV) / (BAC - AC)"},
		Symbols: []Symbol{
			{"TCPI", Text{ID: "efisiensi yang harus dicapai sisa pekerjaan agar proyek mendarat tepat di BAC", EN: "efficiency the remaining work must achieve to land exactly on BAC"}},
		},
		Meaning: Text{ID: "Menjawab pertanyaan yang selalu muncul setelah CPI buruk: seberapa jauh kami harus membaik supaya tetap sesuai anggaran?", EN: "Answers the question that always follows a bad CPI: how much better must we get to stay within budget?"},
		Reading: Text{ID: "TCPI jauh di atas CPI berjalan adalah tanda bahaya. Kalau tim baru mencapai CPI 0,92 selama ini, TCPI 1,06 berarti menuntut perbaikan lima belas persen tanpa alasan jelas mengapa itu mungkin.", EN: "A TCPI far above the running CPI is a warning. If the team has managed 0.92 so far, a TCPI of 1.06 demands a fifteen percent improvement with no stated reason it is achievable."},
		Pitfall: Text{ID: "TCPI bisa dihitung terhadap BAC atau terhadap EAC. Keduanya menjawab pertanyaan berbeda dan angkanya jauh berbeda - selalu sebutkan yang mana.", EN: "TCPI can be computed against BAC or against EAC. They answer different questions and differ widely - always state which."},
		Source:  Text{ID: "PMBOK", EN: "PMBOK"},
	},
	{
		Key: "es", Group: "evm", Route: "/biaya/",
		Name:     Text{ID: "Earned Schedule", EN: "Earned Schedule"},
		Notation: Text{ID: "ES = t dengan PV(t) = EV_sekarang     SV(t) = ES - AT     SPI(t) = ES / AT", EN: "ES = t where PV(t) = EV_now     SV(t) = ES - AT     SPI(t) = ES / AT"},
		Symbols: []Symbol{
			{"ES", Text{ID: "titik waktu pada kurva PV yang nilainya sama dengan EV sekarang", EN: "the point on the PV curve whose value equals today's EV"}},
			{"AT", Text{ID: "actual time - waktu yang sudah benar-benar berjalan", EN: "actual time - elapsed time so far"}},
		},
		Meaning: Text{ID: "Mengubah keterlambatan dari satuan rupiah menjadi satuan waktu: seberapa jauh di masa lalu posisi pekerjaan proyek ini sekarang.", EN: "Converts delay from money into time: how far back in the plan the project's actual progress sits today."},
		Reading: Text{ID: "SV(t) sebesar -2,9 hari berarti pekerjaan yang selesai hari ini semestinya selesai hampir tiga hari kerja yang lalu. Sponsor memahami kalimat itu; SV dalam rupiah tidak.", EN: "An SV(t) of -2.9 days means the work finished today should have been finished nearly three working days ago. Sponsors understand that sentence; SV in money they do not."},
		Pitfall: Text{ID: "ES membutuhkan kurva PV yang monoton naik. Kurva PV yang datar di suatu rentang membuat ES ambigu dan harus diinterpolasi.", EN: "ES needs a monotonically rising PV curve. A flat stretch makes ES ambiguous and it must be interpolated."},
		Source:  Text{ID: "Lipke (2003), pelengkap EVM di PMBOK", EN: "Lipke (2003), an EVM complement in PMBOK"},
	},

	// ------------------------------------------------------------ risiko
	{
		Key: "emv", Group: "risiko", Route: "/risiko/",
		Name:     Text{ID: "Expected Monetary Value", EN: "Expected Monetary Value"},
		Notation: Text{ID: "EMV = peluang x dampak     EMV_total = jumlah seluruh EMV risiko", EN: "EMV = probability x impact     EMV_total = sum of every risk EMV"},
		Symbols: []Symbol{
			{"EMV", Text{ID: "nilai harapan kerugian sebuah risiko, dalam rupiah", EN: "the expected monetary loss of a risk"}},
		},
		Meaning: Text{ID: "Menerjemahkan risiko menjadi angka tunggal yang bisa dijumlahkan, sehingga besar cadangan bisa dihitung, bukan ditebak.", EN: "Translates a risk into a single summable figure so reserves can be computed rather than guessed."},
		Reading: Text{ID: "Cadangan kontinjensi dihitung dari EMV RESIDUAL - setelah mitigasi - bukan dari EMV inheren. Memakai EMV inheren akan mengunci anggaran yang sebenarnya masih bisa dipakai.", EN: "Contingency is sized from RESIDUAL EMV - after mitigation - not inherent EMV. Using inherent EMV locks up budget that is still usable."},
		Pitfall: Text{ID: "EMV adalah rata-rata jangka panjang. Untuk risiko yang hanya terjadi sekali dengan dampak yang mematikan proyek, EMV kecil tetap tidak boleh jadi alasan mengabaikannya.", EN: "EMV is a long-run average. For a one-shot risk whose impact would end the project, a small EMV is still no reason to ignore it."},
		Source:  Text{ID: "PMBOK - Perform Quantitative Risk Analysis", EN: "PMBOK - Perform Quantitative Risk Analysis"},
	},
	{
		Key: "skor", Group: "risiko", Route: "/risiko/",
		Name:     Text{ID: "Skor risiko & matriks probabilitas-dampak", EN: "Risk score & probability-impact matrix"},
		Notation: Text{ID: "skor = tingkat_peluang x tingkat_dampak   (1..5 x 1..5 = 1..25)", EN: "score = probability_level x impact_level   (1..5 x 1..5 = 1..25)"},
		Symbols: []Symbol{
			{"tingkat", Text{ID: "peluang dan dampak dipetakan ke skala ordinal 1 sampai 5", EN: "probability and impact mapped onto an ordinal 1-to-5 scale"}},
		},
		Meaning: Text{ID: "Cara cepat mengurutkan risiko untuk dibahas di rapat, tanpa perlu menyepakati angka rupiah lebih dulu.", EN: "A quick way to rank risks for a meeting without first agreeing on monetary figures."},
		Reading: Text{ID: "Matriks untuk mengurutkan, EMV untuk menganggarkan. Keduanya melengkapi, bukan saling menggantikan.", EN: "The matrix ranks; EMV budgets. They complement rather than replace each other."},
		Pitfall: Text{ID: "Skala ordinal tidak boleh diperlakukan sebagai angka. Risiko berskor 20 tidak dua kali lebih buruk daripada berskor 10 - untuk perbandingan semacam itu, pakai EMV.", EN: "An ordinal scale is not a number. A score of 20 is not twice as bad as 10 - for that comparison, use EMV."},
		Source:  Text{ID: "PMBOK - Perform Qualitative Risk Analysis", EN: "PMBOK - Perform Qualitative Risk Analysis"},
	},
	{
		Key: "cadangan", Group: "risiko", Route: "/biaya/",
		Name:     Text{ID: "Struktur anggaran berlapis", EN: "Layered budget structure"},
		Notation: Text{ID: "cost baseline = BAC + cadangan kontinjensi     pagu total = cost baseline + cadangan manajemen", EN: "cost baseline = BAC + contingency reserve     total budget = cost baseline + management reserve"},
		Symbols: []Symbol{
			{"BAC", Text{ID: "Budget at Completion - jumlah anggaran seluruh aktivitas", EN: "Budget at Completion - the sum of all activity budgets"}},
			{"kontinjensi", Text{ID: "untuk risiko yang SUDAH teridentifikasi (known unknowns)", EN: "for risks already identified (known unknowns)"}},
			{"manajemen", Text{ID: "untuk hal yang belum terbayangkan (unknown unknowns)", EN: "for what has not been imagined yet (unknown unknowns)"}},
		},
		Meaning: Text{ID: "Tiga lapis dengan wewenang berbeda: manajer proyek boleh memakai kontinjensi, cadangan manajemen butuh persetujuan sponsor.", EN: "Three layers with different authority: the project manager may draw on contingency; management reserve needs sponsor approval."},
		Reading: Text{ID: "Earned Value dihitung terhadap BAC, bukan terhadap pagu total. Cadangan bukan pekerjaan, jadi tidak bisa diperoleh sebagai Earned Value.", EN: "Earned Value is computed against BAC, not the total cap. Reserves are not work, so they cannot be earned."},
		Pitfall: Text{ID: "Menyembunyikan cadangan di dalam estimasi aktivitas (padding) menghancurkan Earned Value: CPI jadi tampak sehat padahal cadangan sudah habis diam-diam.", EN: "Hiding reserves inside activity estimates (padding) destroys Earned Value: CPI looks healthy while the buffer quietly drains."},
		Source:  Text{ID: "PMBOK - Determine Budget", EN: "PMBOK - Determine Budget"},
	},

	// -------------------------------------------------------------- mutu
	{
		Key: "controlchart", Group: "mutu", Route: "/kualitas/",
		Name:     Text{ID: "Batas kendali peta X-bar", EN: "X-bar chart control limits"},
		Notation: Text{ID: "CL = rata-rata(x-bar)     UCL = CL + A2 x R-bar     LCL = CL - A2 x R-bar     sigma = R-bar / d2", EN: "CL = mean(x-bar)     UCL = CL + A2 x R-bar     LCL = CL - A2 x R-bar     sigma = R-bar / d2"},
		Symbols: []Symbol{
			{"R-bar", Text{ID: "rata-rata rentang dalam subgrup", EN: "average within-subgroup range"}},
			{"A2, d2", Text{ID: "konstanta yang bergantung ukuran subgrup; untuk n = 5, A2 = 0,577 dan d2 = 2,326", EN: "constants depending on subgroup size; for n = 5, A2 = 0.577 and d2 = 2.326"}},
		},
		Meaning: Text{ID: "Batas kendali dihitung DARI DATA proses itu sendiri, bukan dari keinginan pelanggan. Itulah yang membedakannya dari batas spesifikasi.", EN: "Control limits are computed FROM the process data itself, not from customer wishes. That is what separates them from specification limits."},
		Reading: Text{ID: "Titik di dalam batas kendali berarti variasinya wajar. Titik di luar berarti ada penyebab khusus yang layak ditelusuri - bukan sekadar hari sial.", EN: "A point inside the limits means ordinary variation. A point outside means a special cause worth chasing - not merely a bad day."},
		Pitfall: Text{ID: "Batas kendali dan batas spesifikasi sering tertukar. Proses bisa terkendali sempurna namun tetap gagal memenuhi spesifikasi, dan sebaliknya.", EN: "Control limits and specification limits are routinely confused. A process can be perfectly in control yet still fail specification, and the reverse."},
		Source:  Text{ID: "Pertemuan 9 MPPL - Seven Basic Tools of Quality", EN: "MPPL Session 9 - Seven Basic Tools of Quality"},
	},
	{
		Key: "nelson", Group: "mutu", Route: "/kualitas/",
		Name:     Text{ID: "Aturan Nelson untuk membaca peta kendali", EN: "Nelson rules for reading a control chart"},
		Notation: Text{ID: "Aturan 1: satu titik di luar 3 sigma  |  Aturan 2: tujuh titik berurutan di satu sisi  |  Aturan 3: enam titik menaik/menurun  |  Aturan 5: dua dari tiga di luar 2 sigma", EN: "Rule 1: one point beyond 3 sigma  |  Rule 2: seven consecutive points on one side  |  Rule 3: six points rising/falling  |  Rule 5: two of three beyond 2 sigma"},
		Symbols: []Symbol{
			{"sigma", Text{ID: "simpangan baku rata-rata subgrup, yaitu sepertiga jarak CL ke UCL", EN: "standard deviation of the subgroup mean, a third of the CL-to-UCL distance"}},
		},
		Meaning: Text{ID: "Aturan keputusan yang mengubah grafik naik-turun menjadi sinyal yang bisa ditindaklanjuti.", EN: "Decision rules that turn a wiggly line into an actionable signal."},
		Reading: Text{ID: "Aturan 2 dan 3 menangkap pergeseran bertahap yang tidak pernah melewati batas kendali - persis pola sistem yang melambat perlahan seiring data bertambah.", EN: "Rules 2 and 3 catch gradual drift that never breaches a control limit - exactly the pattern of a system slowing as data grows."},
		Pitfall: Text{ID: "Aturan harus diuji terhadap batas yang SAMA dengan yang digambar. Memakai sigma individual sementara grafiknya memakai batas subgrup akan meloloskan titik yang jelas-jelas di luar garis.", EN: "Rules must be tested against the SAME limits that are drawn. Using individual sigma while the chart uses subgroup limits lets obviously out-of-bounds points pass."},
		Source:  Text{ID: "Nelson (1984); aturan 1 dan 2 disebut di Pertemuan 9", EN: "Nelson (1984); rules 1 and 2 appear in Session 9"},
	},
	{
		Key: "cpk", Group: "mutu", Route: "/kualitas/",
		Name:     Text{ID: "Indeks kemampuan proses", EN: "Process capability index"},
		Notation: Text{ID: "Cpk (satu sisi atas) = (USL - CL) / (3 x sigma)", EN: "Cpk (upper one-sided) = (USL - CL) / (3 x sigma)"},
		Symbols: []Symbol{
			{"USL", Text{ID: "batas spesifikasi atas - di sini, respons 3 detik dari Project Charter", EN: "upper specification limit - here the 3-second response from the Project Charter"}},
		},
		Meaning: Text{ID: "Mengukur seberapa longgar proses berada di dalam batas yang diminta pelanggan.", EN: "Measures how comfortably the process sits inside the limit the customer asked for."},
		Reading: Text{ID: "Cpk di atas 1,33 lazim dianggap mampu. Tetapi Cpk dihitung dari data masa lalu; kalau prosesnya sedang bergeser, Cpk hari ini tidak berlaku bulan depan.", EN: "A Cpk above 1.33 is conventionally capable. But Cpk is computed from past data; if the process is drifting, today's Cpk will not hold next month."},
		Pitfall: Text{ID: "Cpk bagus pada proses yang tidak terkendali adalah angka kosong. Kendali dulu, kemampuan kemudian.", EN: "A good Cpk on an out-of-control process is meaningless. Control first, capability second."},
		Source:  Text{ID: "Pertemuan 9 MPPL - Six Sigma", EN: "MPPL Session 9 - Six Sigma"},
	},
	{
		Key: "coq", Group: "mutu", Route: "/kualitas/",
		Name:     Text{ID: "Biaya kualitas", EN: "Cost of quality"},
		Notation: Text{ID: "COQ = (pencegahan + penilaian) + (kegagalan internal + kegagalan eksternal)     rasio = kesesuaian / ketidaksesuaian", EN: "COQ = (prevention + appraisal) + (internal failure + external failure)     ratio = conformance / non-conformance"},
		Symbols: []Symbol{
			{"kesesuaian", Text{ID: "biaya yang dikeluarkan AGAR cacat tidak terjadi", EN: "cost spent SO THAT defects do not happen"}},
			{"ketidaksesuaian", Text{ID: "biaya yang muncul KARENA cacat terjadi", EN: "cost incurred BECAUSE defects happened"}},
		},
		Meaning: Text{ID: "Menempatkan pengujian bukan sebagai beban biaya melainkan sebagai investasi yang menggantikan biaya kegagalan di kemudian hari.", EN: "Frames testing not as an expense but as an investment that displaces later failure cost."},
		Reading: Text{ID: "Rasio di bawah 1 berarti proyek membayar akibat lebih banyak daripada mencegah sebab. Biaya kegagalan eksternal biasanya berlipat-lipat dibanding kegagalan internal.", EN: "A ratio below 1 means the project pays for consequences more than for prevention. External failure typically costs many times internal failure."},
		Pitfall: Text{ID: "Biaya kegagalan eksternal sering dicatat nol sebelum rilis, sehingga grafiknya memberi kesan keliru bahwa pencegahan sudah cukup.", EN: "External failure cost is often recorded as zero before release, making the chart falsely suggest prevention is already sufficient."},
		Source:  Text{ID: "Pertemuan 9 MPPL - Cost of Quality", EN: "MPPL Session 9 - Cost of Quality"},
	},
	{
		Key: "pareto", Group: "mutu", Route: "/kualitas/",
		Name:     Text{ID: "Analisis Pareto", EN: "Pareto analysis"},
		Notation: Text{ID: "urutkan menurun, lalu kumulatif_i = (jumlah nilai 1..i) / total; ambil kategori sampai kumulatif mencapai 80%", EN: "sort descending, then cumulative_i = (sum of values 1..i) / total; take categories until cumulative reaches 80%"},
		Symbols: []Symbol{
			{"bobot", Text{ID: "cacat dibobot menurut keparahan: kritis 5, mayor 3, minor 1", EN: "defects weighted by severity: critical 5, major 3, minor 1"}},
		},
		Meaning: Text{ID: "Menemukan sedikit penyebab yang menghasilkan sebagian besar masalah, sehingga perbaikan bisa difokuskan.", EN: "Finds the few causes behind most of the problems so remediation can be focused."},
		Reading: Text{ID: "Aturan 80/20 adalah pengamatan, bukan hukum. Yang penting bukan angka 80-nya, melainkan bahwa sebarannya memang timpang.", EN: "The 80/20 rule is an observation, not a law. What matters is not the 80 but that the distribution really is lopsided."},
		Pitfall: Text{ID: "Pareto atas jumlah cacat mentah menyesatkan. Sepuluh cacat kosmetik tidak setara dengan dua cacat yang membuat kuesioner gagal tersimpan - itulah sebabnya bobot keparahan dipakai.", EN: "A Pareto on raw defect counts misleads. Ten cosmetic defects do not equal two that lose a submitted questionnaire - hence the severity weighting."},
		Source:  Text{ID: "Pertemuan 9 MPPL - Seven Basic Tools of Quality", EN: "MPPL Session 9 - Seven Basic Tools of Quality"},
	},

	// -------------------------------------------------------- sumberdaya
	{
		Key: "beban", Group: "sumberdaya", Route: "/organisasi/",
		Name:     Text{ID: "Pembebanan & utilisasi sumber daya", EN: "Resource loading & utilisation"},
		Notation: Text{ID: "beban(peran, hari) = jumlah alokasi aktivitas aktif     utilisasi = total beban / total kapasitas", EN: "load(role, day) = sum of active activity allocations     utilisation = total load / total capacity"},
		Symbols: []Symbol{
			{"alokasi", Text{ID: "porsi hari-orang per hari kerja; 1 berarti penuh waktu", EN: "share of a person-day per working day; 1 means full time"}},
		},
		Meaning: Text{ID: "CPM menganggap sumber daya tak terbatas. Perhitungan ini memeriksa apakah jadwal yang dihasilkannya benar-benar bisa dikerjakan.", EN: "CPM assumes unlimited resources. This calculation checks whether the schedule it produces can actually be delivered."},
		Reading: Text{ID: "Beban melebihi kapasitas berarti satu orang dijadwalkan pada dua pekerjaan di hari yang sama - jadwal itu tidak salah hitung, tetapi tidak bisa dijalankan.", EN: "Load above capacity means one person is scheduled for two jobs on the same day - the schedule is not miscalculated, it is merely undeliverable."},
		Pitfall: Text{ID: "Resource levelling dapat memperpanjang proyek. Menggeser aktivitas ber-float aman; menggeser aktivitas kritis menambah durasi total.", EN: "Resource levelling can lengthen the project. Shifting activities with float is safe; shifting critical ones extends total duration."},
		Source:  Text{ID: "Modul 4 MPPL - mengelola tingkatan sumber daya", EN: "MPPL Module 4 - resource levelling"},
	},
	{
		Key: "kehalusan", Group: "sumberdaya", Route: "/organisasi/",
		Name:     Text{ID: "Kehalusan kurva kebutuhan tim", EN: "Team demand curve smoothness"},
		Notation: Text{ID: "kehalusan = rata-rata(|headcount_t - headcount_(t-1)|) / headcount_puncak", EN: "smoothness = mean(|headcount_t - headcount_(t-1)|) / peak_headcount"},
		Symbols: []Symbol{
			{"headcount", Text{ID: "jumlah peran yang aktif pada satu hari kerja", EN: "number of roles active on a given working day"}},
		},
		Meaning: Text{ID: "Mengukur seberapa bergerigi kebutuhan tim dari hari ke hari.", EN: "Measures how jagged the team's demand is from day to day."},
		Reading: Text{ID: "Nilai mendekati nol berarti tim bekerja stabil. Nilai besar berarti orang menganggur lalu kewalahan bergantian - mahal, dan buruk untuk mutu.", EN: "Near zero means steady work. A large value means people idle then scramble in turns - expensive, and bad for quality."},
		Pitfall: Text{ID: "Kurva yang halus bukan tujuan itu sendiri. Meratakan beban dengan mengorbankan jalur kritis memperpanjang proyek tanpa menghemat apa pun.", EN: "A smooth curve is not an end in itself. Levelling at the expense of the critical path lengthens the project without saving anything."},
		Source:  Text{ID: "Praktik resource smoothing", EN: "Resource smoothing practice"},
	},
}

// FormulasByGroup mengelompokkan rumus menurut kelompoknya.
func FormulasByGroup(group string) []Formula {
	var out []Formula
	for _, f := range Formulas {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}
