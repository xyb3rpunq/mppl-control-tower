package site

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/quality"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// WorkedExample adalah contoh hitung satu rumus memakai angka yang benar-benar
// dihitung engine pada tanggal data yang sedang aktif.
//
// Contoh ini TIDAK ditulis tetap. Kalau data proyek diubah, contohnya ikut
// berubah - sehingga halaman rumus tidak akan pernah memuat angka yang berbeda
// dari halaman analisisnya.
type WorkedExample struct {
	Substitution string // rumus dengan angka disubstitusikan
	Result       string
	Comment      model.Text
}

// Examples membangun contoh hitung untuk setiap rumus.
func (a *Analysis) Examples(lang string) map[string]WorkedExample {
	s := a.Snapshot
	n := func(v float64, d int) string { return render.Num(v, d, lang) }
	rp := func(v float64) string { return render.Rp(v, lang) }
	ex := map[string]WorkedExample{}

	// --- jadwal -----------------------------------------------------------
	a17 := a.Plan.Task("A17")
	ex["ef"] = WorkedExample{
		Substitution: fmt.Sprintf("A17: ES = %d, d = %d  →  EF = %d + %d - 1", a17.ES, a17.Duration, a17.ES, a17.Duration),
		Result:       fmt.Sprintf("EF = %d", a17.EF),
		Comment: model.Text{
			ID: fmt.Sprintf("Aktivitas A17 (API kuesioner) paling cepat mulai pada hari kerja ke-%d dan selesai hari ke-%d, karena pendahulunya A16 baru rampung sebelum itu.", a17.ES, a17.EF),
			EN: fmt.Sprintf("Activity A17 (questionnaire API) can start no earlier than working day %d and finishes on day %d, because predecessor A16 only completes just before.", a17.ES, a17.EF),
		},
	}
	a08 := a.Plan.Task("A08")
	ex["ls"] = WorkedExample{
		Substitution: fmt.Sprintf("A08: LF = %d, d = %d  →  LS = %d - %d + 1", a08.LF, a08.Duration, a08.LF, a08.Duration),
		Result:       fmt.Sprintf("LS = %d", a08.LS),
		Comment: model.Text{
			ID: fmt.Sprintf("A08 (desain arsitektur) boleh mulai selambat-lambatnya hari ke-%d, padahal paling cepat bisa mulai hari ke-%d.", a08.LS, a08.ES),
			EN: fmt.Sprintf("A08 (architecture design) may start as late as day %d even though it could start as early as day %d.", a08.LS, a08.ES),
		},
	}
	ex["float"] = WorkedExample{
		Substitution: fmt.Sprintf("A08: TF = %d - %d      FF = %d", a08.LS, a08.ES, a08.FreeFloat),
		Result:       fmt.Sprintf("TF = %d, FF = %d", a08.TotalFloat, a08.FreeFloat),
		Comment: model.Text{
			ID: fmt.Sprintf("A08 punya %d hari kerja ruang gerak. Bandingkan dengan A17 yang total float-nya %d - A17 tidak boleh molor sedetik pun.", a08.TotalFloat, a17.TotalFloat),
			EN: fmt.Sprintf("A08 holds %d working days of slack. Compare A17 whose total float is %d - A17 cannot slip at all.", a08.TotalFloat, a17.TotalFloat),
		},
	}

	// --- PERT -------------------------------------------------------------
	byID := model.ActivityByID()
	act17 := byID["A17"]
	te17 := schedule.Expected(float64(act17.Optimistic), float64(act17.Duration), float64(act17.Pessimistic))
	sd17 := schedule.StdDev(float64(act17.Optimistic), float64(act17.Pessimistic))
	ex["te"] = WorkedExample{
		Substitution: fmt.Sprintf("A17: O = %d, M = %d, P = %d  →  te = (%d + 4x%d + %d) / 6",
			act17.Optimistic, act17.Duration, act17.Pessimistic,
			act17.Optimistic, act17.Duration, act17.Pessimistic),
		Result: fmt.Sprintf("te = %s %s", n(te17, 2), tr2(lang, "hari kerja", "working days")),
		Comment: model.Text{
			ID: fmt.Sprintf("te = %s lebih besar daripada M = %d. Ekor pesimistis yang panjang menarik durasi harapan ke atas - itulah bentuk kejujuran yang hilang saat orang hanya menyebut satu angka.", n(te17, 2), act17.Duration),
			EN: fmt.Sprintf("te = %s exceeds M = %d. The long pessimistic tail pulls expected duration upward - the honesty that disappears when someone states a single number.", n(te17, 2), act17.Duration),
		},
	}
	ex["sigma"] = WorkedExample{
		Substitution: fmt.Sprintf("A17: sigma = (%d - %d) / 6 = %s      %s: sigma = akar(%s)",
			act17.Pessimistic, act17.Optimistic, n(sd17, 3),
			tr2(lang, "seluruh jalur kritis", "whole critical path"), n(a.PERT.Variance, 3)),
		Result: fmt.Sprintf("sigma_%s = %s %s", tr2(lang, "proyek", "project"), n(a.PERT.StdDev, 3), tr2(lang, "hari kerja", "working days")),
		Comment: model.Text{
			ID: fmt.Sprintf("Varians %s aktivitas kritis dijumlahkan lebih dulu, baru diakarkan. Menjumlahkan sigma langsung akan memberi angka jauh lebih besar dan keliru.", n(float64(len(a.PERT.CriticalPath)), 0)),
			EN: fmt.Sprintf("The variances of %s critical activities are summed first, then square-rooted. Summing sigma directly would give a much larger, wrong figure.", n(float64(len(a.PERT.CriticalPath)), 0)),
		},
	}
	target := float64(a.Plan.Duration)
	z := a.PERT.ZScore(target)
	ex["zscore"] = WorkedExample{
		Substitution: fmt.Sprintf("Z = (%s - %s) / %s", n(target, 0), n(a.PERT.ExpectedDuration, 2), n(a.PERT.StdDev, 3)),
		Result:       fmt.Sprintf("Z = %s  →  P = %s", n(z, 3), render.Pct(a.PERT.ProbabilityBy(target), 2, lang)),
		Comment: model.Text{
			ID: fmt.Sprintf("Menurut PERT, peluang selesai dalam %s hari kerja hanya %s. Simulasi Monte Carlo memberi %s - keduanya sama-sama menolak jadwal 17 minggu, lewat jalan yang berbeda.", n(target, 0), render.Pct(a.PERT.ProbabilityBy(target), 2, lang), render.Pct(a.Sim.OnTimeProb, 2, lang)),
			EN: fmt.Sprintf("PERT puts the odds of finishing within %s working days at just %s. Monte Carlo gives %s - both reject the 17-week schedule, by different routes.", n(target, 0), render.Pct(a.PERT.ProbabilityBy(target), 2, lang), render.Pct(a.Sim.OnTimeProb, 2, lang)),
		},
	}
	ex["montecarlo"] = WorkedExample{
		Substitution: fmt.Sprintf("N = %s, %s  →  P50 = %s, P80 = %s, P90 = %s",
			n(float64(a.Sim.Iterations), 0), tr2(lang, "sebaran beta-PERT", "beta-PERT distribution"),
			n(a.Sim.P50, 0), n(a.Sim.P80, 0), n(a.Sim.P90, 0)),
		Result: fmt.Sprintf("P(T <= %s) = %s", n(target, 0), render.Pct(a.Sim.OnTimeProb, 2, lang)),
		Comment: model.Text{
			ID: fmt.Sprintf("Rerata simulasi %s hari kerja, sembilan hari lebih lama daripada jadwal deterministik %s hari. Selisih itu adalah merge bias: setiap titik pertemuan jalur mengambil yang terlambat.", n(a.Sim.Mean, 1), n(target, 0)),
			EN: fmt.Sprintf("The simulated mean is %s working days, nine longer than the deterministic %s. That gap is merge bias: every path convergence takes the later branch.", n(a.Sim.Mean, 1), n(target, 0)),
		},
	}
	if len(a.Sim.Sensitivity) > 0 {
		top := a.Sim.Sensitivity[0]
		ex["spearman"] = WorkedExample{
			Substitution: fmt.Sprintf("%s: rho = %s, %s = %s", top.ID, n(top.Correlation, 3),
				tr2(lang, "porsi iterasi kritis", "share of iterations critical"), render.Pct(top.CriticalRate, 0, lang)),
			Result: fmt.Sprintf("%s %s", top.ID, top.Name.Get(lang)),
			Comment: model.Text{
				ID: "Aktivitas inilah yang paling menentukan durasi akhir. Menambah cadangan waktu di sini jauh lebih berguna daripada menyebarnya merata ke semua aktivitas.",
				EN: "This activity drives final duration most. Adding time buffer here pays far better than spreading it evenly across everything.",
			},
		}
	}

	// --- EVM --------------------------------------------------------------
	ex["pvevac"] = WorkedExample{
		Substitution: fmt.Sprintf("%s %s: PV = %s | EV = %s | AC = %s",
			tr2(lang, "pada", "at"), a.StatusDate, rp(s.PV), rp(s.EV), rp(s.AC)),
		Result: fmt.Sprintf("%s %s %s %s", tr2(lang, "dari BAC", "of BAC"), rp(s.BAC),
			tr2(lang, "- selesai", "- complete"), render.Pct(s.PercentComplete, 1, lang)),
		Comment: model.Text{
			ID: fmt.Sprintf("Waktu sudah berjalan %s, tetapi pekerjaan yang jadi baru %s dan uang yang keluar sudah %s dari anggaran.", render.Pct(s.PercentElapsed, 1, lang), render.Pct(s.PercentComplete, 1, lang), render.Pct(s.PercentSpent, 1, lang)),
			EN: fmt.Sprintf("Time elapsed is %s, yet only %s of the work exists and %s of the budget is gone.", render.Pct(s.PercentElapsed, 1, lang), render.Pct(s.PercentComplete, 1, lang), render.Pct(s.PercentSpent, 1, lang)),
		},
	}
	ex["varians"] = WorkedExample{
		Substitution: fmt.Sprintf("SV = %s - %s      CV = %s - %s", rp(s.EV), rp(s.PV), rp(s.EV), rp(s.AC)),
		Result:       fmt.Sprintf("SV = %s, CV = %s", render.Signed(s.SV, lang), render.Signed(s.CV, lang)),
		Comment: model.Text{
			ID: "Keduanya negatif: proyek tertinggal sekaligus boros. Ini kombinasi paling sulit dipulihkan, karena mengejar jadwal biasanya justru menambah biaya.",
			EN: "Both negative: behind and overspent at once. This is the hardest combination to recover from, because catching up usually costs more.",
		},
	}
	ex["indeks"] = WorkedExample{
		Substitution: fmt.Sprintf("SPI = %s / %s      CPI = %s / %s", rp(s.EV), rp(s.PV), rp(s.EV), rp(s.AC)),
		Result:       fmt.Sprintf("SPI = %s, CPI = %s", n(s.SPI, 4), n(s.CPI, 4)),
		Comment: model.Text{
			ID: fmt.Sprintf("CPI %s berarti setiap Rp 1.000 yang dibelanjakan hanya menghasilkan pekerjaan senilai Rp %s.", n(s.CPI, 3), n(s.CPI*1000, 0)),
			EN: fmt.Sprintf("A CPI of %s means every IDR 1,000 spent buys only IDR %s of work.", n(s.CPI, 3), n(s.CPI*1000, 0)),
		},
	}
	ex["eac"] = WorkedExample{
		Substitution: fmt.Sprintf("EAC_1 = %s | EAC_2 = %s / %s | EAC_3 = %s + (%s - %s) / (%s x %s)",
			rp(s.EACOptimistic), rp(s.BAC), n(s.CPI, 4), rp(s.AC), rp(s.BAC), rp(s.EV), n(s.CPI, 3), n(s.SPI, 3)),
		Result: fmt.Sprintf("%s - %s", rp(s.EACOptimistic), rp(s.EACPessimistic)),
		Comment: model.Text{
			ID: fmt.Sprintf("Rentangnya %s sampai %s, sementara pagu yang disetujui %s. Bahkan skenario paling optimistis pun sudah melampaui BAC.", rp(s.EACOptimistic), rp(s.EACPessimistic), rp(model.TotalAuthorised)),
			EN: fmt.Sprintf("The range runs %s to %s against an authorised %s. Even the most optimistic case already exceeds BAC.", rp(s.EACOptimistic), rp(s.EACPessimistic), rp(model.TotalAuthorised)),
		},
	}
	ex["tcpi"] = WorkedExample{
		Substitution: fmt.Sprintf("TCPI = (%s - %s) / (%s - %s)", rp(s.BAC), rp(s.EV), rp(s.BAC), rp(s.AC)),
		Result:       fmt.Sprintf("TCPI = %s", n(s.TCPI, 4)),
		Comment: model.Text{
			ID: fmt.Sprintf("Sisa pekerjaan harus dikerjakan dengan efisiensi %s, sementara sejauh ini tim baru mencapai %s. Selisih %s persen itu perlu penjelasan, bukan harapan.", n(s.TCPI, 3), n(s.CPI, 3), n((s.TCPI/s.CPI-1)*100, 1)),
			EN: fmt.Sprintf("The remaining work must run at %s efficiency while the team has managed %s so far. That %s percent gap needs an explanation, not hope.", n(s.TCPI, 3), n(s.CPI, 3), n((s.TCPI/s.CPI-1)*100, 1)),
		},
	}
	ex["es"] = WorkedExample{
		Substitution: fmt.Sprintf("PV(t) = EV = %s  →  ES = %s      SV(t) = %s - %s", rp(s.EV), n(s.ES, 2), n(s.ES, 2), n(s.AtDay, 0)),
		Result:       fmt.Sprintf("SV(t) = %s %s, SPI(t) = %s", n(s.SVt, 2), tr2(lang, "hari kerja", "working days"), n(s.SPIt, 4)),
		Comment: model.Text{
			ID: fmt.Sprintf("Pekerjaan yang selesai pada %s semestinya selesai %s hari kerja lebih awal. Kalimat itu bisa dibawa ke rapat sponsor; \"SV minus lima ratus tiga puluh empat ribu rupiah\" tidak.", a.StatusDate, n(-s.SVt, 1)),
			EN: fmt.Sprintf("The work completed on %s should have been finished %s working days earlier. That sentence survives a sponsor meeting; \"SV is minus five hundred thousand rupiah\" does not.", a.StatusDate, n(-s.SVt, 1)),
		},
	}

	// --- risiko -----------------------------------------------------------
	if len(a.Risk.TopByEMV) > 0 {
		top := a.Risk.TopByEMV[0]
		ex["emv"] = WorkedExample{
			Substitution: fmt.Sprintf("%s: %s x %s = %s  →  %s: %s x %s = %s",
				top.ID, render.Pct(top.Probability, 0, lang), rp(top.Impact), rp(top.EMV),
				tr2(lang, "residual", "residual"), render.Pct(top.ResidualProb, 0, lang), rp(top.ResidualImpact), rp(top.ResidualEMV)),
			Result: fmt.Sprintf("EMV %s = %s", tr2(lang, "residual total", "total residual"), rp(a.Risk.TotalResidualEMV)),
			Comment: model.Text{
				ID: fmt.Sprintf("Cadangan kontinjensi %s hanya menutup %s dari paparan itu. Kekurangannya %s.", rp(model.ContingencyReserve), render.Pct(a.Risk.ReserveCoverage, 1, lang), rp(a.Risk.ReserveGap)),
				EN: fmt.Sprintf("A contingency of %s covers only %s of that exposure, leaving a gap of %s.", rp(model.ContingencyReserve), render.Pct(a.Risk.ReserveCoverage, 1, lang), rp(a.Risk.ReserveGap)),
			},
		}
		ex["skor"] = WorkedExample{
			Substitution: fmt.Sprintf("%s: %s %d x %s %d", top.ID, tr2(lang, "peluang", "probability"), top.ProbLevel, tr2(lang, "dampak", "impact"), top.ImpactLev),
			Result:       fmt.Sprintf("%s = %d (%s)", tr2(lang, "skor", "score"), top.Score, top.Severity),
			Comment: model.Text{
				ID: fmt.Sprintf("Setelah mitigasi, skornya turun ke %d (%s) - tetapi skor ordinal tidak bisa dipakai menganggarkan; untuk itu EMV-lah yang dipakai.", top.ResidualScore, top.ResidualSeverity),
				EN: fmt.Sprintf("After mitigation the score drops to %d (%s) - but an ordinal score cannot size a budget; EMV does that.", top.ResidualScore, top.ResidualSeverity),
			},
		}
	}
	ex["cadangan"] = WorkedExample{
		Substitution: fmt.Sprintf("%s + %s = %s      + %s = %s", rp(a.BAC), rp(model.ContingencyReserve), rp(a.Baseline), rp(a.MgmtReserve), rp(model.TotalAuthorised)),
		Result:       fmt.Sprintf("BAC %s, %s %s, %s %s", rp(a.BAC), tr2(lang, "cost baseline", "cost baseline"), rp(a.Baseline), tr2(lang, "pagu", "cap"), rp(model.TotalAuthorised)),
		Comment: model.Text{
			ID: fmt.Sprintf("Cadangan manajemen %s adalah sisa setelah estimasi bottom-up dan kontinjensi diambil dari pagu Project Charter. Inilah cara angka top-down dan bottom-up direkonsiliasi tanpa memalsukan salah satunya.", rp(a.MgmtReserve)),
			EN: fmt.Sprintf("The %s management reserve is what remains of the charter cap after the bottom-up estimate and contingency. This is how top-down and bottom-up figures reconcile without faking either.", rp(a.MgmtReserve)),
		},
	}

	// --- mutu -------------------------------------------------------------
	cc := a.Control
	ex["controlchart"] = WorkedExample{
		Substitution: fmt.Sprintf("CL = %s, R-bar x A2 = %s  →  UCL = %s, LCL = %s",
			n(cc.CenterLine, 3), n(cc.UCL-cc.CenterLine, 3), n(cc.UCL, 3), n(cc.LCL, 3)),
		Result: fmt.Sprintf("sigma = %s %s", n(cc.Sigma, 4), tr2(lang, "detik", "seconds")),
		Comment: model.Text{
			ID: fmt.Sprintf("Batas kendali atas %s detik jauh di bawah batas spesifikasi 3 detik. Prosesnya masih memenuhi spesifikasi, tetapi sudah tidak terkendali - dua hal yang berbeda.", n(cc.UCL, 2)),
			EN: fmt.Sprintf("The upper control limit of %s seconds sits well below the 3-second specification. The process still meets spec yet is already out of control - two different things.", n(cc.UCL, 2)),
		},
	}
	ex["nelson"] = WorkedExample{
		Substitution: fmt.Sprintf("%d %s", len(cc.Violations), tr2(lang, "pelanggaran terdeteksi", "violations detected")),
		Result:       violationSummary(cc, lang),
		Comment: model.Text{
			ID: "Tidak satu pun nilai melewati batas spesifikasi, tetapi polanya sudah bicara: prosesnya bergeser, bukan berfluktuasi. Ini peringatan dini yang tidak akan terlihat dari grafik biasa.",
			EN: "Not one value breaches the specification limit, yet the pattern already speaks: the process is drifting, not fluctuating. This is an early warning an ordinary chart would not show.",
		},
	}
	if cc.HasCpk {
		ex["cpk"] = WorkedExample{
			Substitution: fmt.Sprintf("Cpk = (%s - %s) / (3 x %s)", n(cc.Spec, 1), n(cc.CenterLine, 3), n(cc.Sigma, 4)),
			Result:       fmt.Sprintf("Cpk = %s", n(cc.Cpk, 3)),
			Comment: model.Text{
				ID: fmt.Sprintf("Cpk %s tampak nyaman, tetapi angka ini dihitung dari data masa lalu. Karena prosesnya sedang bergeser naik, Cpk hari ini tidak berlaku bulan depan.", n(cc.Cpk, 2)),
				EN: fmt.Sprintf("A Cpk of %s looks comfortable, but it is computed from past data. With the process drifting upward, today's Cpk will not hold next month.", n(cc.Cpk, 2)),
			},
		}
	}
	ex["coq"] = WorkedExample{
		Substitution: fmt.Sprintf("(%s + %s) / (%s + %s)",
			rp(a.COQ.Prevention), rp(a.COQ.Appraisal), rp(a.COQ.InternalFailure), rp(a.COQ.ExternalFailure)),
		Result: fmt.Sprintf("%s = %s / %s = %s", tr2(lang, "rasio", "ratio"), rp(a.COQ.Conformance), rp(a.COQ.Nonconformance), n(a.COQ.Ratio, 3)),
		Comment: model.Text{
			ID: fmt.Sprintf("Rasio %s di bawah satu: proyek membayar akibat lebih banyak daripada mencegah sebab. Total biaya kualitas %s dari BAC.", n(a.COQ.Ratio, 2), render.Pct(a.COQ.ShareOfBudget, 1, lang)),
			EN: fmt.Sprintf("A ratio of %s, below one: the project pays for consequences more than prevention. Total cost of quality is %s of BAC.", n(a.COQ.Ratio, 2), render.Pct(a.COQ.ShareOfBudget, 1, lang)),
		},
	}
	if len(a.Pareto.Items) > 0 {
		ex["pareto"] = WorkedExample{
			Substitution: fmt.Sprintf("%s: %s (%s)", a.Pareto.Items[0].Label.Get(lang),
				n(a.Pareto.Items[0].Value, 0), render.Pct(a.Pareto.Items[0].Share, 1, lang)),
			Result: fmt.Sprintf("%d %s %d %s 80%%", a.Pareto.VitalFewCount,
				tr2(lang, "dari", "of"), len(a.Pareto.Items), tr2(lang, "modul menyumbang", "modules account for")),
			Comment: model.Text{
				ID: fmt.Sprintf("Sebarannya timpang tetapi tidak setajam 80/20 klasik: butuh %s kategori untuk mencapai 80 persen. Memfokuskan perbaikan pada modul kuesioner dan integrasi tetap langkah paling menguntungkan.", render.Pct(a.Pareto.VitalFewShare, 0, lang)),
				EN: fmt.Sprintf("The distribution is lopsided but not a textbook 80/20: it takes %s of the categories to reach 80 percent. Focusing remediation on the questionnaire and integration modules is still the best move.", render.Pct(a.Pareto.VitalFewShare, 0, lang)),
			},
		}
	}

	// --- sumber daya ------------------------------------------------------
	if len(a.Resources.Conflicts) > 0 {
		c := a.Resources.Conflicts[0]
		ex["beban"] = WorkedExample{
			Substitution: fmt.Sprintf("%s, %s %d: %s  →  %s = %s", string(c.Role),
				tr2(lang, "hari", "day"), c.Day, fmt.Sprint(c.Activities), tr2(lang, "beban", "load"), n(c.Load, 1)),
			Result: fmt.Sprintf("%s %s %s, %s %d %s", tr2(lang, "kapasitas", "capacity"), n(c.Capacity, 1),
				tr2(lang, "terlampaui", "exceeded"), tr2(lang, "total", "total"), len(a.Resources.Conflicts), tr2(lang, "hari-peran", "role-days")),
			Comment: model.Text{
				ID: "Jadwal CPM-nya sah secara matematis tetapi tidak bisa dijalankan: satu backend developer dijadwalkan mengerjakan dua API sekaligus.",
				EN: "The CPM schedule is mathematically valid but undeliverable: a single backend developer is scheduled on two APIs at once.",
			},
		}
	}
	peak, peakDay := a.Resources.PeakHeadcount()
	ex["kehalusan"] = WorkedExample{
		Substitution: fmt.Sprintf("%s = %s, %s %s %d", tr2(lang, "puncak", "peak"), n(peak, 0),
			tr2(lang, "pada hari", "on day"), tr2(lang, "kerja", "working"), peakDay),
		Result: fmt.Sprintf("%s = %s", tr2(lang, "kehalusan", "smoothness"), n(a.Resources.Smoothness(), 4)),
		Comment: model.Text{
			ID: fmt.Sprintf("Total %s hari-orang tersebar di %d hari kerja. Nilai kehalusan kecil berarti kurvanya cukup rata; masalahnya bukan di kurva keseluruhan, melainkan pada bentrokan per peran.", n(a.Resources.TotalPersonDays, 1), a.Resources.Horizon),
			EN: fmt.Sprintf("A total of %s person-days spread over %d working days. The low smoothness value means the overall curve is fairly even; the problem is not the aggregate curve but the per-role clashes.", n(a.Resources.TotalPersonDays, 1), a.Resources.Horizon),
		},
	}
	upgradeExamples(a, lang, ex)
	return ex
}

// violationSummary merangkum aturan mana saja yang dilanggar, sebagai satu
// baris ringkas untuk contoh hitung.
func violationSummary(cc quality.ControlChart, lang string) string {
	if len(cc.Violations) == 0 {
		return tr2(lang, "proses terkendali", "process in control")
	}
	seen := map[int]bool{}
	var rules []int
	for _, v := range cc.Violations {
		if !seen[v.Rule] {
			seen[v.Rule] = true
			rules = append(rules, v.Rule)
		}
	}
	sort.Ints(rules)
	parts := make([]string, 0, len(rules))
	for _, r := range rules {
		parts = append(parts, fmt.Sprintf("%s %d", tr2(lang, "aturan", "rule"), r))
	}
	return strings.Join(parts, ", ") + " - " + tr2(lang, "proses TIDAK terkendali", "process OUT of control")
}

func tr2(lang, id, en string) string {
	if lang == "en" {
		return en
	}
	return id
}
