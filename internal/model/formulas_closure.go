package model

// Rumus penutup celah: optimalitas levelling, crashing eksak dengan LP, biaya
// yang bergantung waktu, kopula risiko, GERT, dan prakiraan berjalan.
// Dipanggil dari init pada formulas_upgrade.go SETELAH rumus lanjutan
// ditambahkan, supaya urutannya tidak bergantung pada urutan nama berkas.

func applyClosureFormulas() {
	FormulaGroups = append(FormulaGroups,
		struct {
			Key  string
			Name Text
			Desc Text
		}{"prakiraan", Text{ID: "Prakiraan Berjalan", EN: "In-flight Forecasting"},
			Text{ID: "Memprakirakan sisa proyek dari tanggal data: mengunci realisasi, menarik durasi bersyarat, dan menggabungkan rencana dengan bukti lewat bobot kredibilitas.", EN: "Forecasting the rest of the project from the data date: locking actuals, drawing conditional durations, and blending plan with evidence through credibility weighting."}},
	)
	for i := range Formulas {
		switch Formulas[i].Key {
		case "sgs":
			Formulas[i].Notation = Text{ID: "ulangi: pilih j layak dengan kunci prioritas terkecil (LST, LFT, MSLK, GRPW, MTS, SPT, atau daftar acak berbias);  mulai_j = hari t >= siap_j pertama dengan laju(t) >= 0,2;  kerjakan sampai isi pekerjaan habis", EN: "repeat: pick eligible j with the smallest priority key (LST, LFT, MSLK, GRPW, MTS, SPT, or a biased random list);  start_j = first day t >= ready_j with rate(t) >= 0.2;  work until the work content is used up"}
			Formulas[i].Symbols = append(Formulas[i].Symbols, Symbol{"0,2", Text{ID: "laju mulai minimum: setara satu hari kerja per minggu", EN: "minimum start rate: one working day per week"}})
			Formulas[i].Pitfall = Text{ID: "Satu kali SGS adalah heuristik: aturan LST sendirian memberi jadwal satu hari lebih panjang dari optimum pada proyek ini. Jangan menyebut hasil SGS 'jadwal terpendek' tanpa batas bawah yang menyamainya - lihat rumus batas bawah.", EN: "A single SGS run is a heuristic: the LST rule alone gives a schedule one day longer than optimal on this project. Do not call an SGS result 'the shortest schedule' without a lower bound that matches it - see the lower-bound formula."}
		case "crash":
			Formulas[i].Pitfall = Text{ID: "Kurva serakah - potong satu hari termurah, jangan pernah batalkan potongan lama - tidak dijamin optimum: satu potongan bersama yang mahal bisa dipilih walau pasangan cabang paralel lebih murah. Pemecahan eksaknya pemrograman linear. Crashing juga hanya berlaku pada jaringan CPM; pada jadwal yang dibatasi orang, lembur menambah kapasitas, bukan memotong durasi.", EN: "The greedy curve - cut the cheapest single day, never undo an earlier cut - is not guaranteed optimal: an expensive shared cut can be chosen even when a pair of parallel branches is cheaper. The exact solution is linear programming. Crashing also only applies to the CPM network; on a people-constrained schedule, overtime adds capacity rather than cutting durations."}
		case "kejadianrisiko":
			Formulas[i].Notation = Text{ID: "untuk setiap risiko r: bila Phi(Y_r) > 1 - p_residual(r) maka biaya += dampak(r) dan d_terpapar(r) += hari(r)     E[biaya risiko] = jumlah EMV residual", EN: "for every risk r: if Phi(Y_r) > 1 - p_residual(r) then cost += impact(r) and d_exposed(r) += days(r)     E[risk cost] = sum of residual EMV"}
			Formulas[i].Symbols = []Symbol{
				{"d_terpapar(r)", Text{ID: "aktivitas terpanjang pada paket kerja risiko; aktivitas terakhir bila risikonya lintas fase", EN: "the longest activity in the risk's work package; the final activity for cross-phase risks"}},
				{"Y_r", Text{ID: "variabel laten risiko dari kopula risiko - lihat rumus risiko bergerombol", EN: "the risk's latent variable from the risk copula - see the clustered-risk formula"}},
			}
			Formulas[i].Pitfall = Text{ID: "Biaya tambahan akibat risiko harus dibandingkan dengan EMV setelah biaya sewa dikeluarkan dari selisihnya: risiko memperpanjang proyek, dan sewa yang ikut memanjang bukan bagian dari EMV register.", EN: "The added risk cost must be compared with EMV after rental cost is taken out of the difference: risks lengthen the project, and the rentals that lengthen with it are not part of the register's EMV."}
		}
	}
	Formulas = append(Formulas, closureFormulas...)
}

var closureFormulas = []Formula{
	{
		Key: "hargaperhari", Group: "prakiraan", Route: "/keputusan/",
		Name:     Text{ID: "Harga per hari percepatan dari tanggal data", EN: "Price per day of acceleration from the data date"},
		Notation: Text{ID: "harga per hari = (anggaran JCL70 opsi - anggaran JCL70 tanpa percepatan) / (durasi JCL70 tanpa percepatan - durasi JCL70 opsi)     permintaan anggaran = anggaran JCL70 tanpa percepatan - pagu", EN: "price per day = (option 70% JCL budget - no-acceleration 70% JCL budget) / (no-acceleration 70% JCL duration - option 70% JCL duration)     budget request = no-acceleration 70% JCL budget - cap"},
		Symbols: []Symbol{
			{"JCL70", Text{ID: "pasangan durasi dan anggaran terkecil dengan peluang tepat waktu DAN tepat anggaran 70%, disimulasikan dari tanggal data", EN: "the smallest duration-budget pair with a 70% chance of on time AND on budget, simulated from the data date"}},
			{"opsi", Text{ID: "lembur sah, orang baru, atau keduanya - mulai tanggal data", EN: "legal overtime, a new hire, or both - from the data date"}},
		},
		Meaning: Text{ID: "Percepatan dinilai pada titik yang sama dengan komitmen, sehingga selisih anggarannya sudah memuat risiko, sewa yang ikut memendek, dan upah tambahan.", EN: "Acceleration is valued at the same point as the commitment, so the budget difference already carries risk, the rentals that shorten, and the extra pay."},
		Reading: Text{ID: "Harga per hari hanya berarti bila dibandingkan dengan nilai satu hari lebih cepat bagi sponsor. Opsi termurah per hari belum tentu tercepat.", EN: "The price per day only means something against what a day earlier is worth to the sponsor. The cheapest option per day is not necessarily the fastest."},
		Pitfall: Text{ID: "Menghitung percepatan dari hari pertama proyek setelah proyek berjalan: rencana lembur yang harinya sudah lewat tidak bisa dibeli.", EN: "Computing acceleration from the project's first day once it is running: an overtime plan whose days have passed cannot be bought."},
		Source:  Text{ID: "NASA Cost Estimating Handbook - JCL 70%; PP 35/2021 - upah lembur", EN: "NASA Cost Estimating Handbook - 70% JCL; Government Regulation 35/2021 - overtime pay"},
	},
	{
		Key: "lemburlevelling", Group: "kompresi", Route: "/optimasi/",
		Name:     Text{ID: "Lembur sah pada jadwal berbatas sumber daya", EN: "Legal overtime on a resource-constrained schedule"},
		Notation: Text{ID: "kapasitas_r(t) = normal_r(t) + n_r x h / 8  dan  laju maks = 1 + h / 8  bila t bukan hari ujian     h = min(4, 18 / 5) = 3,6 jam     j = max(pakai_r(t) - normal_r(t), jumlah alokasi x (laju - 1)) x 8 / n_r     upah_r(t) = n_r x (1,5 min(j,1) + 2 max(j-1,0)) x tarif_r / 8", EN: "capacity_r(t) = normal_r(t) + n_r x h / 8  and  max rate = 1 + h / 8  when t is not an exam day     h = min(4, 18 / 5) = 3.6 hours     j = max(use_r(t) - normal_r(t), sum of alloc x (rate - 1)) x 8 / n_r     pay_r(t) = n_r x (1.5 min(j,1) + 2 max(j-1,0)) x rate_r / 8"},
		Symbols: []Symbol{
			{"n_r", Text{ID: "jumlah orang penuh waktu pada peran r; peran paruh waktu tidak diberi lembur", EN: "full-time people in role r; part-time roles get no overtime"}},
			{"h", Text{ID: "jam lembur per hari yang tetap di bawah 18 jam seminggu bila dilakukan setiap hari kerja", EN: "overtime hours per day that stay under 18 hours a week when worked every working day"}},
			{"j", Text{ID: "jam lembur per orang pada hari t: jam di atas kapasitas normal, atau jam di atas alokasi rencana saat pekerjaan dipercepat - yang lebih besar", EN: "overtime hours per person on day t: hours above normal capacity, or hours above the planned allocation when a task is sped up - whichever is larger"}},
			{"laju maks", Text{ID: "lembur mempercepat pekerjaan orang yang lembur; orang baru tidak mempercepat satu pekerjaan tak terbagi", EN: "overtime speeds up the work of whoever works overtime; a new hire does not speed up one indivisible task"}},
		},
		Meaning: Text{ID: "Pada jadwal yang dibatasi orang, lembur tidak memendekkan isi pekerjaan: lembur menambah jam kerja, baik untuk pekerjaan lain yang menunggu maupun untuk pekerjaan orang itu sendiri. Durasi terpendek dengan lembur maksimum dibuktikan dengan batas bawah energetik pada kapasitas itu.", EN: "On a people-constrained schedule, overtime does not shorten the work content: it adds working hours, both for other waiting work and for that person's own task. The shortest duration with maximum overtime is proven with the energetic lower bound at that capacity."},
		Reading: Text{ID: "Setiap rencana lembur sah memakai kapasitas yang tidak lebih besar dari kapasitas maksimum, jadi batas bawah itu berlaku untuk semuanya. Biaya per durasi berasal dari memangkas lembur yang tidak diperlukan.", EN: "Every legal overtime plan uses no more capacity than the maximum, so that lower bound holds for all of them. The cost per duration comes from trimming overtime that is not needed."},
		Pitfall: Text{ID: "Biayanya adalah rencana termurah yang ditemukan, bukan minimum yang terbukti. Dan kurva crashing CPM tidak bisa dipakai untuk menjanjikan percepatan jadwal nyata: ia memotong jaringan yang tidak menghormati kapasitas orang.", EN: "The cost is the cheapest plan found, not a proven minimum. And the CPM crashing curve cannot be used to promise acceleration of the real schedule: it cuts a network that ignores people's capacity."},
		Source:  Text{ID: "PP 35/2021 Pasal 26, 31, 32; Kolisch & Hartmann (1999) - RCPSP", EN: "Government Regulation 35/2021 Art. 26, 31, 32; Kolisch & Hartmann (1999) - RCPSP"},
	},
	{
		Key: "batasbawah", Group: "kompresi", Route: "/optimasi/",
		Name:     Text{ID: "Batas bawah energetik dan celah optimalitas", EN: "Energetic lower bound and optimality gap"},
		Notation: Text{ID: "untuk himpunan S aktivitas peran r dengan mulai >= h dan jarak ke j >= g:  mulai_j >= suplai_r(h, jumlah alok x d) + g     celah = durasi terbaik - batas bawah", EN: "for a set S of role-r activities starting at >= h with gap to j >= g:  start_j >= supply_r(h, sum alloc x d) + g     gap = best duration - lower bound"},
		Symbols: []Symbol{
			{"suplai_r(h, W)", Text{ID: "hari pertama ketika kapasitas kumulatif peran r sejak hari h mencapai W hari-orang", EN: "the first day on which role r's cumulative capacity since day h reaches W person-days"}},
			{"g", Text{ID: "jalur terpanjang dari selesainya anggota S sampai mulainya j", EN: "the longest path from the finish of a member of S to the start of j"}},
			{"celah", Text{ID: "nol berarti jadwal terbaik terbukti optimal", EN: "zero means the best schedule is proven optimal"}},
		},
		Meaning: Text{ID: "Hari-orang yang dibutuhkan setiap aktivitas tetap, berapa pun lajunya. Bila jendela waktu yang mungkin bagi sekelompok aktivitas tidak menyediakan kapasitas sebanyak itu, pekerjaan sesudahnya pasti bergeser - untuk jadwal apa pun.", EN: "The person-days each activity needs are fixed, whatever the pace. If the time window open to a group of activities does not supply that much capacity, later work must shift - in any schedule."},
		Reading: Text{ID: "Batas bawah dirambatkan maju dan mundur sampai tidak berubah, lalu setiap tenggat di atasnya diuji: bila tenggat E memaksa sebuah aktivitas mulai sebelum batas mulainya, E mustahil. Jadwal yang menyentuh batas bawah tidak mungkin diperbaiki dengan mengubah urutan.", EN: "The bound propagates forward and backward until it stops changing, then each deadline above it is tested: if deadline E forces an activity to start before its earliest possible start, E is impossible. A schedule that touches the lower bound cannot be improved by reordering."},
		Pitfall: Text{ID: "Batas bawah yang lemah tidak membuktikan apa pun. Celah nol hanya berarti optimal untuk MODEL yang dipakai - di sini model isi pekerjaan dengan laju pecahan dan laju mulai minimum 20%.", EN: "A weak lower bound proves nothing. A zero gap means optimal only for the MODEL used - here the work-content model with fractional rates and a 20% minimum start rate."},
		Source:  Text{ID: "Baptiste, Le Pape & Nuijten (2001); Klein & Scholl (1999) - batas bawah destruktif", EN: "Baptiste, Le Pape & Nuijten (2001); Klein & Scholl (1999) - destructive lower bounds"},
	},
	{
		Key: "lpcrash", Group: "kompresi", Route: "/optimasi/",
		Name:     Text{ID: "Crashing eksak dan trade-off biaya total (LP)", EN: "Exact crashing and total-cost trade-off (LP)"},
		Notation: Text{ID: "min jumlah c_ik x_ik + jumlah tarif_k (E - t_pembeli_k)   dengan  x_i = jumlah_k x_ik,  t_j >= t_i + d_i - x_i,  t_i + d_i - x_i <= E = T,  0 <= x_ik <= 1", EN: "min sum c_ik x_ik + sum rate_k (E - t_buyer_k)   s.t.  x_i = sum_k x_ik,  t_j >= t_i + d_i - x_i,  t_i + d_i - x_i <= E = T,  0 <= x_ik <= 1"},
		Symbols: []Symbol{
			{"x_ik", Text{ID: "hari ke-k yang dipotong dari aktivitas i; x_i adalah jumlahnya", EN: "the k-th day cut from activity i; x_i is their sum"}},
			{"c_ik", Text{ID: "biaya marjinal hari ke-k: tidak menurun, sehingga LP mengisi hari murah lebih dulu", EN: "marginal cost of day k: non-decreasing, so the LP fills cheap days first"}},
			{"tarif_k", Text{ID: "biaya sewa atau langganan k per hari", EN: "rental or subscription k per day"}},
			{"E, T", Text{ID: "hari selesai proyek dan tenggat yang diuji", EN: "project finish and the deadline being tested"}},
		},
		Meaning: Text{ID: "Untuk setiap tenggat, LP memilih kombinasi potongan termurah - termasuk membatalkan potongan yang tidak lagi perlu. Dengan biaya sewa ikut dihitung, titik terendah kurva adalah durasi yang paling murah secara keseluruhan.", EN: "For each deadline, the LP picks the cheapest combination of cuts - including undoing cuts no longer needed. With rentals included, the lowest point of the curve is the cheapest duration overall."},
		Reading: Text{ID: "Matriks batasannya unimodular total, sehingga simpleks langsung memberi hari bulat. Selisih biaya total terhadap durasi normal dibagi jumlah hari adalah nilai minimum satu hari percepatan agar keputusan crashing tidak merugi.", EN: "The constraint matrix is totally unimodular, so the simplex returns whole days directly. The total-cost difference from the normal duration divided by the days saved is the minimum value of a day of acceleration for crashing to break even."},
		Pitfall: Text{ID: "Tanpa mengunci tanggal mulai proyek, LP bisa 'menghemat' sewa dengan menggeser seluruh proyek ke kanan - hasil yang benar secara aljabar tetapi mustahil dijalankan.", EN: "Without pinning the project start date, the LP can 'save' rentals by shifting the whole project right - algebraically correct but impossible to execute."},
		Source:  Text{ID: "Kelley (1961) - critical path planning and scheduling; Modul 4 MPPL - kompresi waktu", EN: "Kelley (1961) - critical path planning and scheduling; MPPL Module 4 - schedule compression"},
	},
	{
		Key: "biayawaktu", Group: "terpadu", Route: "/simulasi-terpadu/",
		Name:     Text{ID: "Biaya sewa yang bergantung waktu", EN: "Time-dependent rental cost"},
		Notation: Text{ID: "tarif_k = nilai_k / (D_rencana - mulai_pembeli_k)     biaya_k = tarif_k x (selesai_proyek - mulai_pembeli_k)", EN: "rate_k = value_k / (D_plan - buyer_start_k)     cost_k = rate_k x (project_finish - buyer_start_k)"},
		Symbols: []Symbol{
			{"nilai_k", Text{ID: "nilai sewa atau langganan pada anggaran, untuk rentang rencana", EN: "budgeted rental or subscription value, for the planned span"}},
			{"mulai_pembeli_k", Text{ID: "hari mulai aktivitas yang membeli sewa itu", EN: "the start day of the activity that buys the rental"}},
		},
		Meaning: Text{ID: "Pembelian sekali tidak peduli proyek selesai kapan; sewa dan langganan bertambah setiap hari proyek berjalan. Pada jadwal rencana biayanya persis sama dengan BAC.", EN: "One-off purchases do not care when the project finishes; rentals and subscriptions grow with every day it runs. On the planned schedule the cost equals BAC exactly."},
		Reading: Text{ID: "Jumlah tarif adalah harga satu hari keterlambatan yang tidak pernah muncul di BAC - dan juga penghematan satu hari percepatan.", EN: "The sum of rates is the price of a day of delay that BAC never shows - and also the saving from a day of acceleration."},
		Pitfall: Text{ID: "Vendor yang menagih per bulan penuh membuat biaya naik bertahap, bukan halus per hari. Model harian adalah rata-rata dari tangga itu.", EN: "A vendor billing whole months makes cost rise in steps, not smoothly per day. The daily model averages that staircase."},
		Source:  Text{ID: "PMBOK - biaya langsung, tidak langsung, dan bergantung waktu", EN: "PMBOK - direct, indirect, and time-dependent costs"},
	},
	{
		Key: "kopularisiko", Group: "terpadu", Route: "/simulasi-terpadu/",
		Name:     Text{ID: "Risiko bergerombol lewat kopula faktor", EN: "Clustered risks via a factor copula"},
		Notation: Text{ID: "Y_r = lambda x Z_penggerak(r) + sqrt(1 - lambda^2) x epsilon_r     risiko r terjadi bila Phi(Y_r) > 1 - p_r", EN: "Y_r = lambda x Z_driver(r) + sqrt(1 - lambda^2) x epsilon_r     risk r fires when Phi(Y_r) > 1 - p_r"},
		Symbols: []Symbol{
			{"Z_penggerak", Text{ID: "faktor laten sebab bersama; untuk penggerak kinerja pengembang, faktor laten peran BE yang sama dengan sampler durasi", EN: "the shared cause's latent factor; for the development-performance driver, the same BE latent factor used by the duration sampler"}},
			{"lambda", Text{ID: "bobot penggerak; 0,6 berarti korelasi laten 0,36 antar-risiko satu penggerak", EN: "driver loading; 0.6 means a latent correlation of 0.36 between same-driver risks"}},
		},
		Meaning: Text{ID: "Y_r tetap normal baku, jadi P(risiko r terjadi) tetap persis p_r - EMV dan rerata biaya tidak berubah. Yang berubah adalah seberapa sering beberapa risiko terjadi bersamaan.", EN: "Y_r stays standard normal, so P(risk r fires) stays exactly p_r - EMV and mean cost do not change. What changes is how often several risks fire together."},
		Reading: Text{ID: "Korelasi phi antar-kejadian biner selalu jauh di bawah korelasi laten, karena kejadian yang jarang punya ruang korelasi yang sempit. Bacalah P95, bukan rerata.", EN: "The phi correlation between binary events is always far below the latent correlation, because rare events have little room to correlate. Read the P95, not the mean."},
		Pitfall: Text{ID: "Mengalikan peluang dua risiko untuk mendapat peluang keduanya terjadi hanya sah bila keduanya saling bebas - justru asumsi yang dicabut rumus ini.", EN: "Multiplying two risks' probabilities to get the chance of both is valid only when they are independent - exactly the assumption this formula removes."},
		Source:  Text{ID: "Kopula Gauss satu faktor (Vasicek); PMBOK - Perform Quantitative Risk Analysis", EN: "One-factor Gaussian copula (Vasicek); PMBOK - Perform Quantitative Risk Analysis"},
	},
	{
		Key: "gert", Group: "terpadu", Route: "/simulasi-terpadu/",
		Name:     Text{ID: "GERT: putaran rework dengan aturan Mason", EN: "GERT: rework loops with Mason's rule"},
		Notation: Text{ID: "W(s) = p x e^(s t)     W_E = W_maju / (1 - W_putar)     E[T] = W_E'(0) / W_E(0)     putaran: E[tambahan] = p r / (1 - p),  sd = sqrt(p) r / (1 - p)", EN: "W(s) = p x e^(s t)     W_E = W_forward / (1 - W_loop)     E[T] = W_E'(0) / W_E(0)     loop: E[extra] = p r / (1 - p),  sd = sqrt(p) r / (1 - p)"},
		Symbols: []Symbol{
			{"p", Text{ID: "peluang satu pemeriksaan gagal", EN: "probability one check fails"}},
			{"r", Text{ID: "hari kerja satu putaran perbaikan dan pemeriksaan ulang", EN: "working days of one fix-and-recheck round"}},
			{"W_E(0)", Text{ID: "peluang simpul akhir tercapai; 1 untuk putaran yang pasti berakhir", EN: "probability the end node is reached; 1 for a loop that surely ends"}},
		},
		Meaning: Text{ID: "Jaringan CPM tidak boleh punya putaran, padahal tes yang gagal memang diulang. GERT memberi setiap cabang peluang dan mereduksi jaringan menjadi satu cabang setara yang rerata dan variansnya bisa dihitung.", EN: "A CPM network may not contain loops, yet failed tests really are repeated. GERT gives every branch a probability and reduces the network to one equivalent branch whose mean and variance can be computed."},
		Reading: Text{ID: "Jumlah putaran tambahan mengikuti sebaran geometrik: P(N >= k) = p^k. Rerata p/(1-p) tumbuh tidak linear - menurunkan p sedikit memotong rerata putaran lebih banyak dari yang dikira.", EN: "The number of extra loops is geometric: P(N >= k) = p^k. The mean p/(1-p) grows non-linearly - lowering p a little cuts expected loops more than intuition suggests."},
		Pitfall: Text{ID: "Menambah buffer tetap sebesar satu putaran menutupi rerata tetapi tidak ekornya: masih ada peluang p^2 butuh dua putaran tambahan atau lebih.", EN: "Adding a fixed one-loop buffer covers the mean but not the tail: there is still a p^2 chance of needing two or more extra loops."},
		Source:  Text{ID: "Pritsker (1966) - GERT; Modul 4 MPPL - conditional diagramming methods", EN: "Pritsker (1966) - GERT; MPPL Module 4 - conditional diagramming methods"},
	},
	{
		Key: "kredibilitas", Group: "prakiraan", Route: "/prakiraan/",
		Name:     Text{ID: "Kredibilitas Bühlmann dan durasi bersyarat", EN: "Bühlmann credibility and conditional duration"},
		Notation: Text{ID: "X = jumlah aktual / jumlah rerata PERT     Var = jumlah varians PERT / (jumlah rerata PERT)^2     tau2 = max(0, (X - 1)^2 - Var)     Z = tau2 / (tau2 + Var)     faktor = Z X + (1 - Z)     durasi berjalan: u' = F(e) + u (1 - F(e))", EN: "X = sum actual / sum PERT mean     Var = sum PERT variance / (sum PERT mean)^2     tau2 = max(0, (X - 1)^2 - Var)     Z = tau2 / (tau2 + Var)     factor = Z X + (1 - Z)     running duration: u' = F(e) + u (1 - F(e))"},
		Symbols: []Symbol{
			{"Var", Text{ID: "varians X bila tim bekerja persis sesuai sebaran beta-PERT - derau estimasi, dihitung dari (mu - O)(P - mu)/7", EN: "the variance of X if the team works exactly to its beta-PERT distributions - estimation noise, from (mu - O)(P - mu)/7"}},
			{"tau2", Text{ID: "varians penyimpangan sistematis tim, diestimasi dengan metode momen", EN: "the variance of the team's systematic deviation, estimated by the method of moments"}},
			{"e", Text{ID: "hari yang sudah dilalui aktivitas yang sedang berjalan", EN: "days already elapsed on a running activity"}},
		},
		Meaning: Text{ID: "Rencana dan realisasi digabung tanpa bobot yang dipilih: selisih yang masih bisa dijelaskan derau estimasi memberi Z nol, selisih yang jauh melampaui derau memberi Z mendekati satu. Aktivitas yang sedang berjalan tidak mungkin berdurasi kurang dari hari yang sudah dilaluinya.", EN: "Plan and actuals are blended without a chosen weight: a gap that estimation noise can explain gives Z of zero, a gap far beyond the noise gives Z close to one. A running activity cannot last less than the days it has already used."},
		Reading: Text{ID: "Rasio aktual terhadap rerata PERT di sekitar 1 berarti tim bekerja sesuai sebaran estimasinya - keterlambatan jadwal lalu datang dari rencana yang disusun memakai M, bukan dari kinerja buruk.", EN: "An actual-to-PERT-mean ratio near 1 means the team works to its estimated distributions - the schedule delay then comes from a plan built on M, not from poor performance."},
		Pitfall: Text{ID: "Membandingkan durasi aktual dengan M, bukan dengan rerata PERT, membuat tim yang bekerja persis sesuai estimasi tampak lambat. Estimator momen memakai satu kelompok data, jadi tau2 bisa berlebihan pada satu selisih besar yang kebetulan - karena itu prakiraan tanpa belajar tetap ditampilkan.", EN: "Comparing actual durations with M instead of the PERT mean makes a team working exactly to estimate look slow. The moment estimator uses one group of data, so tau2 can overstate a single large chance deviation - which is why the no-learning forecast is still shown."},
		Source:  Text{ID: "Bühlmann (1967) - teori kredibilitas; Modul 4 MPPL - pengawasan dan pengendalian jadwal", EN: "Bühlmann (1967) - credibility theory; MPPL Module 4 - schedule monitoring and control"},
	},
}
