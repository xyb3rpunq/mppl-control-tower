package model

// Rumus analisis lanjutan: levelling sumber daya, kompresi jadwal, dan
// simulasi terpadu. Dipisah dari formulas.go supaya penambahannya bisa
// ditelusuri, tetapi digabungkan ke Formulas dan FormulaGroups lewat init
// sehingga halaman referensi tetap satu daftar.

func init() {
	FormulaGroups = append(FormulaGroups,
		struct {
			Key  string
			Name Text
			Desc Text
		}{"kompresi", Text{ID: "Levelling & Kompresi Jadwal", EN: "Levelling & Schedule Compression"},
			Text{ID: "Menyusun jadwal yang menghormati kapasitas orang, lalu menghitung harga mempercepatnya kembali dengan uang atau dengan risiko.", EN: "Building a schedule that respects people's capacity, then pricing the speed-up - with money or with risk."}},
		struct {
			Key  string
			Name Text
			Desc Text
		}{"terpadu", Text{ID: "Simulasi Terpadu & JCL", EN: "Integrated Simulation & JCL"},
			Text{ID: "Menggabungkan ketidakpastian durasi, korelasi, kejadian risiko, dan kapasitas untuk menjawab peluang tepat waktu dan tepat anggaran sekaligus.", EN: "Combining duration uncertainty, correlation, risk events, and capacity to answer the odds of on time and on budget at once."}},
	)
	Formulas = append(Formulas, upgradeFormulas...)
	applyClosureFormulas()
}

var upgradeFormulas = []Formula{
	{
		Key: "sgs", Group: "kompresi", Route: "/optimasi/",
		Name:     Text{ID: "Serial Schedule Generation Scheme", EN: "Serial Schedule Generation Scheme"},
		Notation: Text{ID: "ulangi: pilih j layak dengan LS_j terkecil;  mulai_j = hari t >= siap_j pertama dengan laju(t) >= 0,25;  kerjakan sampai isi pekerjaan habis", EN: "repeat: pick eligible j with the smallest LS_j;  start_j = first day t >= ready_j with rate(t) >= 0.25;  work until the work content is used up"},
		Symbols: []Symbol{
			{"layak", Text{ID: "aktivitas yang seluruh pendahulunya sudah terjadwal", EN: "an activity whose predecessors are all scheduled"}},
			{"LS_j", Text{ID: "late start CPM, dipakai sebagai aturan prioritas", EN: "the CPM late start, used as the priority rule"}},
			{"siap_j", Text{ID: "hari paling awal yang diizinkan pendahulu pada jadwal levelling", EN: "the earliest day allowed by predecessors on the levelled schedule"}},
		},
		Meaning: Text{ID: "Heuristik baku untuk penjadwalan berbatas sumber daya: jadwalkan satu aktivitas pada satu waktu, dahulukan yang paling mendesak, dan jangan pernah melanggar kapasitas.", EN: "The standard heuristic for resource-constrained scheduling: schedule one activity at a time, most urgent first, and never exceed capacity."},
		Reading: Text{ID: "Selisih antara durasi hasil SGS dan durasi CPM adalah harga asumsi 'sumber daya tak terbatas' - angka persisnya untuk proyek ini ada pada contoh hitung di bawah.", EN: "The gap between the SGS duration and the CPM duration is the price of the 'unlimited resources' assumption - the exact figure for this project is in the worked example below."},
		Pitfall: Text{ID: "SGS adalah heuristik, bukan optimum - penjadwalan berbatas sumber daya adalah masalah NP-hard. Durasinya batas atas yang bisa dicapai, bukan bukti tidak ada jadwal yang lebih pendek.", EN: "SGS is a heuristic, not an optimum - resource-constrained scheduling is NP-hard. Its duration is an achievable upper bound, not proof that no shorter schedule exists."},
		Source:  Text{ID: "Kolisch (1996); Modul 4 MPPL - mengelola tingkatan sumber daya", EN: "Kolisch (1996); MPPL Module 4 - resource levelling"},
	},
	{
		Key: "laju", Group: "kompresi", Route: "/optimasi/",
		Name:     Text{ID: "Laju kerja berbatas kapasitas", EN: "Capacity-limited work rate"},
		Notation: Text{ID: "laju(t) = min(1, min_i (kap_i(t) - pakai_i(t)) / alok_i)     progres += laju(t) setiap hari sampai progres = d", EN: "rate(t) = min(1, min_i (cap_i(t) - used_i(t)) / alloc_i)     progress += rate(t) each day until progress = d"},
		Symbols: []Symbol{
			{"kap_i(t)", Text{ID: "kapasitas peran i pada hari t, setelah jendela ketersediaan diterapkan", EN: "capacity of role i on day t, after availability windows"}},
			{"pakai_i(t)", Text{ID: "kapasitas peran i yang sudah dipakai aktivitas lain pada hari t", EN: "capacity of role i already used by other activities on day t"}},
			{"alok_i", Text{ID: "alokasi peran i yang dibutuhkan aktivitas ini", EN: "allocation of role i this activity needs"}},
		},
		Meaning: Text{ID: "Isi pekerjaan tetap (alokasi x durasi hari-orang); yang berubah hanya kecepatannya. Pada hari ketika orangnya hanya tersedia separuh, pekerjaan maju separuh hari.", EN: "Work content stays fixed (allocation x duration person-days); only the pace changes. On a day when the person is half available, work advances half a day."},
		Reading: Text{ID: "Peran yang paling sempit menentukan laju seluruh aktivitas - pekerjaan integrasi tidak bisa jalan lebih cepat dari orang tersibuk yang terlibat.", EN: "The tightest role sets the pace of the whole activity - integration work cannot move faster than the busiest person involved."},
		Pitfall: Text{ID: "Model ini tidak menambah hari-orang saat pekerjaan melambat. Pada kenyataannya pekerjaan yang terputus-putus butuh waktu 'memanaskan ulang' - jadi angka ini cenderung optimistis.", EN: "This model adds no person-days when work slows. In reality interrupted work needs 'warm-up' time again - so the figures lean optimistic."},
		Source:  Text{ID: "Model isi pekerjaan (work-content scheduling)", EN: "Work-content scheduling model"},
	},
	{
		Key: "crash", Group: "kompresi", Route: "/optimasi/",
		Name:     Text{ID: "Crashing dan slope biaya", EN: "Crashing and cost slope"},
		Notation: Text{ID: "d_crash = max(O, M - max(1, floor(M/3)))     h = 8x / d_crash     premi = (d_crash x (1,5 min(h,1) + 2 max(h-1,0)) / 8 - x) / x     slope = (biaya tenaga kerja / M) x premi", EN: "d_crash = max(O, M - max(1, floor(M/3)))     h = 8x / d_crash     premium = (d_crash x (1.5 min(h,1) + 2 max(h-1,0)) / 8 - x) / x     slope = (labour cost / M) x premium"},
		Symbols: []Symbol{
			{"slope", Text{ID: "tambahan biaya per hari yang dipotong dari satu aktivitas", EN: "extra cost per day cut from one activity"}},
			{"x, h", Text{ID: "hari yang dipotong, dan jam lembur per hari saat pekerjaannya dibagi rata ke hari tersisa; h di atas 4 dilarang PP 35/2021", EN: "days cut, and overtime hours per day when the work is spread over the remaining days; h above 4 is prohibited by Government Regulation 35/2021"}},
			{"premi", Text{ID: "upah lembur PP 35/2021: jam pertama 1,5x, jam berikutnya 2x upah sejam (1/173 upah sebulan)", EN: "overtime pay under Government Regulation 35/2021: first hour 1.5x, later hours 2x the hourly wage (1/173 of monthly pay)"}},
		},
		Meaning: Text{ID: "Membeli waktu dengan uang: aktivitas kritis termurah dipercepat lebih dulu, satu hari demi satu hari, sampai tidak ada lagi yang bisa dipotong.", EN: "Buying time with money: the cheapest critical activity is accelerated first, one day at a time, until nothing more can be cut."},
		Reading: Text{ID: "Kurva waktu-biaya selalu melengkung ke atas. Hari-hari pertama murah; hari-hari terakhir mahal dan sering menuntut memotong dua jalur kritis paralel sekaligus.", EN: "The time-cost curve always bends upward. The first days are cheap; the last are expensive and often require cutting two parallel critical paths at once."},
		Pitfall: Text{ID: "Memotong aktivitas kritis di satu jalur tidak memendekkan proyek bila ada jalur kritis paralel. Kurva yang tidak memeriksa ulang durasi proyek setelah setiap potongan akan 'membeli' hari yang tidak pernah ada.", EN: "Cutting a critical activity on one path does not shorten the project if a parallel critical path exists. A curve that does not recheck project duration after each cut will 'buy' days that never materialise."},
		Source:  Text{ID: "Modul 4 MPPL - kompresi waktu", EN: "MPPL Module 4 - schedule compression"},
	},
	{
		Key: "fasttrack", Group: "kompresi", Route: "/optimasi/",
		Name:     Text{ID: "Fast-tracking dan rework harapan", EN: "Fast-tracking and expected rework"},
		Notation: Text{ID: "FS(pendahulu) diganti SS dengan lag = d_pendahulu - ceil(d_pendahulu/2)     rework = p_rework x hari tumpang tindih x biaya harian penerus", EN: "FS(predecessor) becomes SS with lag = d_pred - ceil(d_pred/2)     rework = p_rework x overlap days x successor daily cost"},
		Symbols: []Symbol{
			{"p_rework", Text{ID: "peluang pekerjaan tumpang tindih harus diulang sebagian; 30% (asumsi)", EN: "probability overlapping work must be partly redone; 30% (assumed)"}},
			{"tumpang tindih", Text{ID: "separuh durasi pendahulu, dibulatkan ke atas", EN: "half of the predecessor's duration, rounded up"}},
		},
		Meaning: Text{ID: "Membeli waktu dengan risiko: penerus dimulai saat pendahulunya baru setengah jalan. Tidak ada uang keluar di muka, tetapi sebagian pekerjaan dibangun di atas masukan yang belum final.", EN: "Buying time with risk: the successor starts while its predecessor is only half done. No money goes out up front, but part of the work rests on inputs that are not yet final."},
		Reading: Text{ID: "Kandidat yang pendahulu dan penerusnya dikerjakan orang yang sama ditolak: satu orang tidak bisa mengerjakan dua pekerjaan tumpang tindih.", EN: "Candidates whose predecessor and successor share a person are rejected: one person cannot do two overlapping jobs."},
		Pitfall: Text{ID: "Penghematan fast-tracking tidak bisa dijumlahkan. Memendekkan satu jalur membuat jalur lain menjadi kritis, sehingga menerapkan semua kandidat sekaligus menghemat jauh lebih sedikit daripada jumlah per kandidat.", EN: "Fast-tracking savings do not add up. Shortening one path makes another critical, so applying every candidate at once saves far less than the per-candidate total."},
		Source:  Text{ID: "Modul 4 MPPL - kompresi waktu; PMBOK - Develop Schedule", EN: "MPPL Module 4 - schedule compression; PMBOK - Develop Schedule"},
	},
	{
		Key: "betapert", Group: "pert", Route: "/pert/",
		Name:     Text{ID: "Sebaran beta-PERT dan transformasi invers", EN: "Beta-PERT distribution and inverse transform"},
		Notation: Text{ID: "alpha = 1 + 4(M-O)/(P-O)     beta = 1 + 4(P-M)/(P-O)     F(x) = I_((x-O)/(P-O))(alpha, beta)     sampel = F^-1(u)", EN: "alpha = 1 + 4(M-O)/(P-O)     beta = 1 + 4(P-M)/(P-O)     F(x) = I_((x-O)/(P-O))(alpha, beta)     sample = F^-1(u)"},
		Symbols: []Symbol{
			{"I_x(a, b)", Text{ID: "fungsi beta tak lengkap teregularisasi - CDF sebaran Beta", EN: "regularised incomplete beta function - the Beta CDF"}},
			{"u", Text{ID: "bilangan seragam di (0,1)", EN: "a uniform number in (0,1)"}},
		},
		Meaning: Text{ID: "Parameterisasi ini menjamin rerata (O + 4M + P)/6 dan kedua parameter selalu >= 1, sehingga densitasnya terbatas. Sampel diambil lewat invers CDF yang ditabulasi pada 2.049 titik.", EN: "This parameterisation guarantees the mean (O + 4M + P)/6 and both parameters >= 1, so the density stays bounded. Samples come from an inverse CDF tabulated at 2,049 points."},
		Reading: Text{ID: "Transformasi invers dipilih karena satu alasan: korelasi bisa disuntikkan pada tingkat u, sementara sebaran marginal setiap aktivitas tetap persis beta-PERT.", EN: "The inverse transform is chosen for one reason: correlation can be injected at the level of u while each activity's marginal distribution stays exactly beta-PERT."},
		Pitfall: Text{ID: "Ada lebih dari satu rumus alpha-beta untuk PERT di literatur. Rumus yang berbeda memberi rerata sama tetapi varians berbeda - selalu sebutkan yang mana.", EN: "The literature has more than one alpha-beta formula for PERT. Different formulas share the mean but not the variance - always state which one."},
		Source:  Text{ID: "Vose (2008); Numerical Recipes 6.4 untuk I_x", EN: "Vose (2008); Numerical Recipes 6.4 for I_x"},
	},
	{
		Key: "kopula", Group: "terpadu", Route: "/simulasi-terpadu/",
		Name:     Text{ID: "Korelasi lewat kopula Gauss", EN: "Correlation via a Gaussian copula"},
		Notation: Text{ID: "z_peran ~ N(0,1) per iterasi     u_j = Phi(rho x z_peran(j) + sqrt(1 - rho^2) x epsilon_j)     d_j = F_j^-1(u_j)", EN: "z_role ~ N(0,1) per iteration     u_j = Phi(rho x z_role(j) + sqrt(1 - rho^2) x epsilon_j)     d_j = F_j^-1(u_j)"},
		Symbols: []Symbol{
			{"z_peran", Text{ID: "faktor kinerja laten satu peran pada satu iterasi", EN: "the latent performance factor of one role in one iteration"}},
			{"rho", Text{ID: "kekuatan ikatan aktivitas ke faktor perannya; 0,5 sebagai asumsi baku", EN: "how strongly an activity follows its role factor; 0.5 as the default assumption"}},
			{"peran(j)", Text{ID: "peran dominan aktivitas j - alokasi terbesar", EN: "the dominant role of activity j - the largest allocation"}},
		},
		Meaning: Text{ID: "Kalau Backend Developer sedang lambat pada satu iterasi, ia lambat pada SEMUA aktivitasnya. Dengan rho = 0 model ini kembali menjadi sampel independen, persis seperti halaman PERT.", EN: "If the Backend Developer is slow in one iteration, they are slow on ALL their activities. With rho = 0 the model reduces to independent samples, exactly as on the PERT page."},
		Reading: Text{ID: "Korelasi peringkat yang benar-benar muncul antar-aktivitas peran sama sekitar rho kuadrat. Korelasi melebarkan sebaran ke dua arah - ekor kanan dan kiri sama-sama memanjang.", EN: "The rank correlation that actually appears between same-role activities is roughly rho squared. Correlation widens the spread both ways - both tails lengthen."},
		Pitfall: Text{ID: "Mengabaikan korelasi tidak netral: simulasi independen secara sistematis melaporkan ketidakpastian yang terlalu kecil, karena keterlambatan saling menutupi secara acak - padahal di dunia nyata keterlambatan cenderung bergerombol.", EN: "Ignoring correlation is not neutral: an independent simulation systematically understates uncertainty because delays cancel out at random - whereas in reality delays cluster."},
		Source:  Text{ID: "Kopula Gauss dalam analisis risiko jadwal", EN: "Gaussian copula in schedule risk analysis"},
	},
	{
		Key: "kejadianrisiko", Group: "terpadu", Route: "/simulasi-terpadu/",
		Name:     Text{ID: "Kejadian risiko dalam simulasi", EN: "Risk events in simulation"},
		Notation: Text{ID: "untuk setiap risiko r: bila U < p_residual(r) maka biaya += dampak(r) dan d_terpapar(r) += hari(r)     E[biaya risiko] = jumlah EMV residual", EN: "for every risk r: if U < p_residual(r) then cost += impact(r) and d_exposed(r) += days(r)     E[risk cost] = sum of residual EMV"},
		Symbols: []Symbol{
			{"d_terpapar(r)", Text{ID: "aktivitas terpanjang pada paket kerja risiko; aktivitas terakhir bila risikonya lintas fase", EN: "the longest activity in the risk's work package; the final activity for cross-phase risks"}},
			{"U", Text{ID: "bilangan seragam, satu per risiko per iterasi", EN: "a uniform number, one per risk per iteration"}},
		},
		Meaning: Text{ID: "Risk register berhenti menjadi tabel dan menjadi bagian dari model: setiap iterasi adalah satu masa depan di mana sebagian risiko terjadi dan sebagian tidak.", EN: "The risk register stops being a table and becomes part of the model: every iteration is one future where some risks fire and some do not."},
		Reading: Text{ID: "Rerata tambahan biaya di simulasi harus mendekati jumlah EMV residual pada halaman Risiko - itulah pemeriksaan silang bahwa keduanya memakai angka yang sama.", EN: "The mean added cost in simulation should approach the residual EMV total on the Risk page - that is the cross-check that both use the same numbers."},
		Pitfall: Text{ID: "Kejadian risiko diandaikan saling bebas dan tidak berkorelasi dengan durasi aktivitas. Dalam kenyataan, risiko cenderung datang bersamaan - proyek yang sedang telat lebih mungkin juga mengalami perubahan lingkup.", EN: "Risk events are assumed independent of each other and of activity durations. In reality risks tend to arrive together - a project already running late is more likely to see scope change too."},
		Source:  Text{ID: "PMBOK - Perform Quantitative Risk Analysis", EN: "PMBOK - Perform Quantitative Risk Analysis"},
	},
	{
		Key: "jcl", Group: "terpadu", Route: "/simulasi-terpadu/",
		Name:     Text{ID: "Joint Confidence Level", EN: "Joint Confidence Level"},
		Notation: Text{ID: "JCL(T, C) = (1/N) x jumlah 1[durasi_i <= T dan biaya_i <= C]     frontier_70(T) = anggaran terkecil C dengan JCL(T, C) >= 0,70", EN: "JCL(T, C) = (1/N) x sum 1[duration_i <= T and cost_i <= C]     frontier_70(T) = the smallest budget C with JCL(T, C) >= 0.70"},
		Symbols: []Symbol{
			{"T, C", Text{ID: "tenggat (hari kerja) dan anggaran (rupiah)", EN: "deadline (working days) and budget (money)"}},
			{"1[...]", Text{ID: "bernilai 1 bila kedua syarat terpenuhi pada iterasi itu", EN: "equals 1 when both conditions hold in that iteration"}},
		},
		Meaning: Text{ID: "Peluang selesai dalam tenggat DAN dalam anggaran sekaligus. NASA mewajibkan anggaran proyek besarnya ditetapkan pada JCL 70%.", EN: "The probability of finishing within the deadline AND within budget at once. NASA requires budgets for its major projects to be set at a 70% JCL."},
		Reading: Text{ID: "Peluang bersama di titik P80 durasi dan P80 biaya hampir selalu di bawah 80%. Dua P80 yang dilaporkan berdampingan memberi rasa aman palsu.", EN: "The joint probability at the P80 duration and the P80 cost almost always falls below 80%. Two P80s reported side by side give false comfort."},
		Pitfall: Text{ID: "Frontier bukan satu angka tetapi pilihan: tenggat lebih longgar menurunkan anggaran yang dibutuhkan. Menetapkan salah satunya tanpa melihat yang lain membuang informasi terpenting.", EN: "The frontier is not one number but a menu: a looser deadline lowers the budget needed. Fixing one without looking at the other throws away the key information."},
		Source:  Text{ID: "NASA Cost Estimating Handbook (JCL 70%)", EN: "NASA Cost Estimating Handbook (70% JCL)"},
	},
}
