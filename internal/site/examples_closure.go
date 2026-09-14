package site

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// closureExamples menambahkan contoh hitung untuk rumus penutup celah, dan
// memperbarui contoh lama yang maknanya bergeser setelah model diperluas.
func closureExamples(a *Analysis, lang string, ex map[string]WorkedExample) {
	n := func(v float64, d int) string { return render.Num(v, d, lang) }
	rp := func(v float64) string { return render.Rp(v, lang) }
	opt := a.LevelOpt

	if sgs, ok := ex["sgs"]; ok {
		sgs.Comment = model.Text{
			ID: sgs.Comment.ID + fmt.Sprintf(" Aturan LST sendirian memberi %d hari; jadwal terbaik dari %d daftar acak berbias memberi %d hari.", opt.Baseline.Duration, opt.Samples, opt.Best.Duration),
			EN: sgs.Comment.EN + fmt.Sprintf(" The LST rule alone gives %d days; the best of %d biased random lists gives %d days.", opt.Baseline.Duration, opt.Samples, opt.Best.Duration),
		}
		ex["sgs"] = sgs
	}

	verdict := tr2(lang, "terbukti optimal", "proven optimal")
	if !opt.Proven {
		verdict = tr2(lang, "belum terbukti optimal", "not yet proven optimal")
	}
	ex["batasbawah"] = WorkedExample{
		Substitution: fmt.Sprintf("CPM %d  ≤  solo %d  ≤  %s %d  →  %s = %d",
			opt.Bound.CPM, opt.Bound.Solo, tr2(lang, "energetik", "energetic"), opt.Bound.Energetic,
			tr2(lang, "batas bawah", "lower bound"), opt.Bound.Value),
		Result: fmt.Sprintf("%s = %d - %d = %d (%s)", tr2(lang, "celah", "gap"), opt.Best.Duration, opt.Bound.Value, opt.Gap, verdict),
		Comment: model.Text{
			ID: fmt.Sprintf("Batas solo saja sudah %d hari karena DevOps paruh waktu dan periode ujian. Argumen energetik atas peran yang sama menaikkannya ke %d - tepat di durasi jadwal terbaik, sehingga tidak ada urutan kerja lain yang bisa selesai lebih cepat.", opt.Bound.Solo, opt.Bound.Energetic),
			EN: fmt.Sprintf("The solo bound alone is already %d days because of half-time DevOps and the exam periods. Energetic reasoning over shared roles raises it to %d - exactly the best schedule's duration, so no other work order can finish sooner.", opt.Bound.Solo, opt.Bound.Energetic),
		},
	}

	if len(a.Exact.Points) > 5 {
		p0, p5 := a.Exact.Points[0], a.Exact.Points[5]
		var cuts []string
		for id, k := range p5.Cuts {
			if k > 1 {
				cuts = append(cuts, fmt.Sprintf("%s×%d", id, k))
			} else {
				cuts = append(cuts, id)
			}
		}
		sort.Strings(cuts)
		ex["lpcrash"] = WorkedExample{
			Substitution: fmt.Sprintf("T = %d: x = {%s};  %s %s + %s %s",
				p5.Duration, strings.Join(cuts, ", "), tr2(lang, "crash", "crash"), rp(p5.TotalCrash), tr2(lang, "sewa", "rentals"), rp(p5.Rental)),
			Result: fmt.Sprintf("%s(%d) = %s;  %s(%d) = %s", tr2(lang, "total", "total"), p5.Duration, rp(p5.Total), tr2(lang, "total", "total"), p0.Duration, rp(p0.Total)),
			Comment: model.Text{
				ID: fmt.Sprintf("Mempercepat lima hari berbiaya bersih %s, bukan premi %s, karena sewa ikut memendek. %s", rp(a.TradeOffNet(5)), rp(p5.CrashCost), greedyNoteID(a)),
				EN: fmt.Sprintf("Accelerating five days costs a net %s, not the %s premium, because rentals shorten too. %s", rp(a.TradeOffNet(5)), rp(p5.CrashCost), greedyNoteEN(a)),
			},
		}
	}

	if ot := a.Overtime; len(ot.Points) > 0 {
		p := ot.Points[len(ot.Points)-1]
		var role model.Role
		for _, r := range p.Roles() {
			if p.Hours[r] > p.Hours[role] {
				role = r
			}
		}
		first := ot.Points[0]
		ex["lemburlevelling"] = WorkedExample{
			Substitution: fmt.Sprintf("h = min(4, 18/5) = %s;  %s %d → %d;  %s = %s %s (%d %s)",
				n(ot.HoursPerDay, 1), tr2(lang, "durasi", "duration"), ot.Levelled, ot.MinDuration,
				string(role), n(p.Hours[role], 1), tr2(lang, "jam", "h"), p.Days[role], tr2(lang, "hari", "days")),
			Result: fmt.Sprintf("%s(%d) = %s;  %s(%d) = %s;  %s = %d",
				tr2(lang, "upah", "pay"), p.Duration, rp(p.Cost), tr2(lang, "upah", "pay"), first.Duration, rp(first.Cost),
				tr2(lang, "batas bawah", "lower bound"), ot.Bound.Value),
			Comment: model.Text{
				ID: fmt.Sprintf("Crashing CPM menyebut %d hari seharga %s. Pada jadwal yang bisa dijalankan, hari yang sama butuh upah lembur %s, bersih %s setelah sewa yang dihemat.", ot.Levelled-ot.MinDuration, rp(cpmCost(a, ot.Levelled-ot.MinDuration)), rp(p.Cost), rp(p.Net)),
				EN: fmt.Sprintf("CPM crashing prices %d days at %s. On the executable schedule the same days need %s in overtime pay, net %s after the rentals saved.", ot.Levelled-ot.MinDuration, rp(cpmCost(a, ot.Levelled-ot.MinDuration)), rp(p.Cost), rp(p.Net)),
			},
		}
	}

	if len(a.Rentals) > 0 {
		r := a.Rentals[0]
		for _, x := range a.Rentals {
			if x.Amount > r.Amount {
				r = x
			}
		}
		fin := a.Final()
		ex["biayawaktu"] = WorkedExample{
			Substitution: fmt.Sprintf("%s (%s): %s / %d %s = %s", r.Label.Get(lang), r.Activity, rp(r.Amount), r.PlanSpan, tr2(lang, "hari", "days"), rp(r.Rate)),
			Result:       fmt.Sprintf("%s = %s", tr2(lang, "jumlah tarif harian", "sum of daily rates"), rp(a.RentalDaily())),
			Comment: model.Text{
				ID: fmt.Sprintf("Biaya sewa rencana %s. Pada lapisan kapasitas, rerata biaya sewa simulasi %s - selisih yang tidak pernah terlihat di BAC.", rp(fin.TimeCostPlan), rp(fin.TimeCost)),
				EN: fmt.Sprintf("Planned rental cost is %s. In the capacity layer the simulated mean rental cost is %s - a difference BAC never shows.", rp(fin.TimeCostPlan), rp(fin.TimeCost)),
			},
		}
	}

	if len(a.RiskSweep) >= 3 {
		ind, def := a.RiskSweep[0], a.RiskSweep[0]
		for _, p := range a.RiskSweep {
			if p.Loading == model.RiskLoading {
				def = p
			}
		}
		ex["kopularisiko"] = WorkedExample{
			Substitution: fmt.Sprintf("lambda = %s  →  %s = %s^2 = %s;  phi %s", n(model.RiskLoading, 1),
				tr2(lang, "korelasi laten", "latent correlation"), n(model.RiskLoading, 1), n(model.RiskLoading*model.RiskLoading, 2), n(def.Phi, 3)),
			Result: fmt.Sprintf("P95 %s %s → %s", tr2(lang, "biaya", "cost"), rp(ind.CostP95), rp(def.CostP95)),
			Comment: model.Text{
				ID: fmt.Sprintf("Rerata biaya %s pada risiko saling bebas dan %s pada risiko bergerombol - praktis sama, karena peluang marginal dijaga. Ekor kananlah yang menebal.", rp(ind.CostMean), rp(def.CostMean)),
				EN: fmt.Sprintf("Mean cost is %s with independent risks and %s with clustered risks - practically the same, because marginal probabilities are preserved. It is the right tail that thickens.", rp(ind.CostMean), rp(def.CostMean)),
			},
		}
	}

	if len(a.GERT) > 0 {
		g := a.GERT[0]
		ex["gert"] = WorkedExample{
			Substitution: fmt.Sprintf("%s: p = %s, r = %s  →  E = %s x %s / (1 - %s)",
				g.Loop.ID, n(g.Loop.FailProb, 2), n(g.ReworkDays, 2), n(g.Loop.FailProb, 2), n(g.ReworkDays, 2), n(g.Loop.FailProb, 2)),
			Result: fmt.Sprintf("E[%s] = %s %s, sd = %s;  E[N] %s %s, Monte Carlo %s",
				tr2(lang, "tambahan", "extra"), n(g.Reduced.Mean(), 3), tr2(lang, "hari", "days"), n(g.Reduced.SD(), 3),
				tr2(lang, "analitik", "analytic"), n(g.Cycles, 3), n(g.MCCycles, 3)),
			Comment: model.Text{
				ID: fmt.Sprintf("Peluang regresi butuh dua putaran tambahan atau lebih adalah %s. Rerata putaran dari simulasi %s hanya berbeda derau sampel dari bentuk tertutup %s.", render.Pct(g.AtLeastTwo, 1, lang), n(g.MCCycles, 3), n(g.Cycles, 3)),
				EN: fmt.Sprintf("The chance regression needs two or more extra rounds is %s. The simulated mean of %s loops differs from the closed form's %s only by sampling noise.", render.Pct(g.AtLeastTwo, 1, lang), n(g.MCCycles, 3), n(g.Cycles, 3)),
			},
		}
	}

	if fl := a.InFlight; fl != nil {
		var running string
		for _, r := range a.RunningRows() {
			running = fmt.Sprintf("%s: e = %s, E[d | d > e] = %s", r.ID, n(r.Elapsed, 1), n(r.CondMean, 2))
			break
		}
		ex["kredibilitas"] = WorkedExample{
			Substitution: fmt.Sprintf("Var = %s;  tau2 = max(0, (%s - 1)^2 - %s) = %s;  Z = %s;  %s: tau2 = %s, Z = %s",
				n(fl.DurationVar, 5), n(fl.ObservedRatio, 4), n(fl.DurationVar, 5), n(fl.DurationTau2, 5), n(fl.Credibility, 3),
				tr2(lang, "ujian", "exams"), n(fl.ExamTau2, 4), n(fl.ExamCredibility, 3)),
			Result: fmt.Sprintf("%s = %s;  %s", tr2(lang, "faktor durasi", "duration factor"), n(fl.DurationFactor, 3), running),
			Comment: model.Text{
				ID: fmt.Sprintf("Dengan realisasi dikunci, P80 dari tanggal data %s hari kerja; tanpa belajar dari realisasi %s hari. Faktor kapasitas ujian ikut diperbarui dari %s menjadi %s.", n(a.Forecast.DurP80, 0), n(a.ForecastPrior.DurP80, 0), render.Pct(model.ExamCapacityFactor, 0, lang), render.Pct(fl.ExamFactor, 1, lang)),
				EN: fmt.Sprintf("With actuals locked, P80 from the data date is %s working days; without learning from actuals, %s days. The exam capacity factor is updated too, from %s to %s.", n(a.Forecast.DurP80, 0), n(a.ForecastPrior.DurP80, 0), render.Pct(model.ExamCapacityFactor, 0, lang), render.Pct(fl.ExamFactor, 1, lang)),
			},
		}
	}

	if len(a.Ladder) > int(simulate.LayerRisks) {
		l1, l2 := a.Ladder[simulate.LayerCorrelated], a.Ladder[simulate.LayerRisks]
		r02 := model.Risks[1]
		hit := float64(l2.RiskHits[r02.ID]) / float64(l2.Config.Iterations)
		delta := RiskCostDelta(a)
		ex["kejadianrisiko"] = WorkedExample{
			Substitution: fmt.Sprintf("%s: p = %s → %s %s;  %s +%s, +%d %s",
				r02.ID, render.Pct(r02.ResidualProb, 0, lang), tr2(lang, "terjadi", "fired"), render.Pct(hit, 1, lang),
				tr2(lang, "bila terjadi", "if fired"), rp(r02.ResidualImpact), r02.ScheduleImpact, tr2(lang, "hari", "days")),
			Result: fmt.Sprintf("%s %s → %s", tr2(lang, "rerata biaya di luar sewa", "mean cost excluding rentals"), rp(l1.CostMean-l1.TimeCost), rp(l2.CostMean-l2.TimeCost)),
			Comment: model.Text{
				ID: fmt.Sprintf("Setelah biaya sewa yang ikut memanjang dikeluarkan, kenaikan rerata biaya %s hampir persis sama dengan jumlah EMV residual pada halaman Risiko, %s - pemeriksaan silang bahwa simulasi dan register memakai angka yang sama.", rp(delta), rp(a.Risk.TotalResidualEMV)),
				EN: fmt.Sprintf("Once the rentals that lengthen with the project are taken out, the rise in mean cost, %s, almost exactly matches the residual EMV total on the Risk page, %s - the cross-check that simulation and register use the same numbers.", rp(delta), rp(a.Risk.TotalResidualEMV)),
			},
		}
	}
}

// RiskCostDelta mengembalikan kenaikan rerata biaya dari lapisan korelasi ke
// lapisan risiko setelah biaya sewa dikeluarkan - besaran yang harus
// mendekati jumlah EMV residual register.
func RiskCostDelta(a *Analysis) float64 {
	if len(a.Ladder) <= int(simulate.LayerRisks) {
		return 0
	}
	l1, l2 := a.Ladder[simulate.LayerCorrelated], a.Ladder[simulate.LayerRisks]
	return (l2.CostMean - l2.TimeCost) - (l1.CostMean - l1.TimeCost)
}

// cpmCost adalah biaya crashing eksak pada jaringan CPM untuk memotong days hari.
func cpmCost(a *Analysis, days int) float64 {
	p, _ := a.Exact.PointAt(a.Exact.Normal - days)
	return p.CrashCost
}
