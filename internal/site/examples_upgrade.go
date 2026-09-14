package site

import (
	"fmt"
	"math"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// upgradeExamples menambahkan contoh hitung untuk rumus analisis lanjutan.
// Seluruh angkanya diambil dari hasil yang sama dengan yang dirender di
// halaman Optimasi dan Simulasi Terpadu.
func upgradeExamples(a *Analysis, lang string, ex map[string]WorkedExample) {
	n := func(v float64, d int) string { return render.Num(v, d, lang) }
	rp := func(v float64) string { return render.Rp(v, lang) }
	byID := model.ActivityByID()

	a19 := a.Level.Tasks["A19"]
	ex["sgs"] = WorkedExample{
		Substitution: fmt.Sprintf("CPM = %d  →  %s = %d  →  + %s = %d",
			a.LevelWhy.CPM, tr2(lang, "kapasitas", "capacity"), a.LevelWhy.CapacityOnly,
			tr2(lang, "jendela ujian", "exam window"), a.LevelWhy.WithWindows),
		Result: fmt.Sprintf("%s = %d %s", tr2(lang, "durasi levelling", "levelled duration"), a.Level.Duration, tr2(lang, "hari kerja", "working days")),
		Comment: model.Text{
			ID: fmt.Sprintf("A19 (ES CPM hari ke-%d) baru bisa mulai hari ke-%d: terbawa %d hari dari pendahulu, lalu menunggu %d hari karena Backend Developer sedang mengerjakan jalur kritis yang prioritasnya lebih tinggi.", a19.ES, a19.Start, a19.CarriedDays, a19.WaitDays),
			EN: fmt.Sprintf("A19 (CPM ES day %d) can only start on day %d: %d days carried from predecessors, then %d days waiting because the Backend Developer is on the higher-priority critical path.", a19.ES, a19.Start, a19.CarriedDays, a19.WaitDays),
		},
	}

	a33 := a.Level.Tasks["A33"]
	ex["laju"] = WorkedExample{
		Substitution: fmt.Sprintf("A33: alok_OPS = 1, kap_OPS = %s  →  %s = min(1, %s / 1) = %s",
			n(model.Capacity[model.RoleOPS], 1), tr2(lang, "laju", "rate"), n(model.Capacity[model.RoleOPS], 1), n(model.Capacity[model.RoleOPS], 1)),
		Result: fmt.Sprintf("%d %s → %d %s", byID["A33"].Duration, tr2(lang, "hari isi pekerjaan", "days of work"), a33.Finish-a33.Start, tr2(lang, "hari kalender kerja", "working-calendar days")),
		Comment: model.Text{
			ID: "Deployment produksi butuh DevOps penuh waktu, tetapi DevOps pada Project Charter bersifat opsional dan dimodelkan paruh waktu. Isi pekerjaannya tetap tiga hari-orang; kalendernya memanjang dua kali lipat.",
			EN: "Production deployment needs a full-time DevOps engineer, but the charter marks DevOps optional and it is modelled half-time. The work is still three person-days; the calendar doubles.",
		},
	}

	if len(a.Crash.Steps) >= 5 {
		first := a.Crash.Steps[0]
		act := byID[first.Crashed[0]]
		plan := act.Crash(model.RateCard)
		five := a.Crash.Steps[4]
		last := a.Crash.Steps[len(a.Crash.Steps)-1]
		ex["crash"] = WorkedExample{
			Substitution: fmt.Sprintf("%s: M = %d, O = %d  →  d_crash = %d;  slope = (%s / %d) x %s = %s",
				act.ID, act.Duration, act.Optimistic, plan.CrashDur,
				rp(act.LabourCost(model.RateCard)), act.Duration, n(model.CrashPremium, 2), rp(plan.SlopePerDay)),
			Result: fmt.Sprintf("%s = %s, %s %d → %d = %s",
				tr2(lang, "5 hari pertama", "first 5 days"), rp(five.TotalCost),
				tr2(lang, "penuh", "full"), a.Crash.NormalDuration, a.Crash.MinDuration, rp(last.TotalCost)),
			Comment: model.Text{
				ID: fmt.Sprintf("Langkah pertama memotong %s karena slope-nya termurah di jalur kritis. Langkah terakhir harus memotong %s sekaligus - jalur kritis paralel membuat satu potongan saja tidak memendekkan proyek.", strings.Join(first.Crashed, " + "), strings.Join(last.Crashed, " + ")),
				EN: fmt.Sprintf("The first step cuts %s because its slope is the cheapest on the critical path. The last step must cut %s together - parallel critical paths mean a single cut would not shorten the project.", strings.Join(first.Crashed, " + "), strings.Join(last.Crashed, " + ")),
			},
		}
	}

	if len(a.FastViable) > 0 {
		f := a.FastViable[0]
		succ := byID[f.Succ]
		naive := 0
		for _, v := range a.FastViable {
			naive += v.DaysSaved
		}
		ex["fasttrack"] = WorkedExample{
			Substitution: fmt.Sprintf("%s → %s: %s %d;  rework = %s x %d x %s",
				f.Pred, f.Succ, tr2(lang, "tumpang tindih", "overlap"), f.OverlapDays,
				n(compress.ReworkProbability, 2), f.OverlapDays, rp(succ.LabourCost(model.RateCard)/float64(succ.Duration))),
			Result: fmt.Sprintf("%s %d %s, rework %s", tr2(lang, "hemat", "saves"), f.DaysSaved, tr2(lang, "hari", "days"), rp(f.ExpectedRework)),
			Comment: model.Text{
				ID: fmt.Sprintf("Jumlah penghematan %d kandidat layak adalah %d hari, tetapi menerapkan semuanya sekaligus hanya memendekkan proyek menjadi %d hari kerja - hemat %d hari.", len(a.FastViable), naive, a.FastAllDur, a.Plan.Duration-a.FastAllDur),
				EN: fmt.Sprintf("The %d viable candidates add up to %d days saved, yet applying them all at once only brings the project to %d working days - %d days saved.", len(a.FastViable), naive, a.FastAllDur, a.Plan.Duration-a.FastAllDur),
			},
		}
	}

	a17 := byID["A17"]
	o, m, pp := float64(a17.Optimistic), float64(a17.Duration), float64(a17.Pessimistic)
	al, be := simulate.BetaPERTParams(o, m, pp)
	med := simulate.BetaPERTQuantile(o, m, pp, 0.5)
	p80 := simulate.BetaPERTQuantile(o, m, pp, 0.8)
	ex["betapert"] = WorkedExample{
		Substitution: fmt.Sprintf("A17: O = %d, M = %d, P = %d  →  alpha = 1 + 4x%d/%d = %s,  beta = 1 + 4x%d/%d = %s",
			a17.Optimistic, a17.Duration, a17.Pessimistic,
			a17.Duration-a17.Optimistic, a17.Pessimistic-a17.Optimistic, n(al, 3),
			a17.Pessimistic-a17.Duration, a17.Pessimistic-a17.Optimistic, n(be, 3)),
		Result: fmt.Sprintf("F^-1(0,5) = %s   F^-1(0,8) = %s", n(med, 2), n(p80, 2)),
		Comment: model.Text{
			ID: fmt.Sprintf("Median durasi A17 adalah %s hari - sudah di atas M = %d. Delapan dari sepuluh sampel selesai dalam %s hari. Rerata sebarannya persis te PERT: %s.", n(med, 2), a17.Duration, n(p80, 2), n((o+4*m+pp)/6, 2)),
			EN: fmt.Sprintf("A17's median duration is %s days - already above M = %d. Eight in ten samples finish within %s days. The distribution mean is exactly the PERT te: %s.", n(med, 2), a17.Duration, n(p80, 2), n((o+4*m+pp)/6, 2)),
		},
	}

	if len(a.RhoSweep) >= 3 && len(a.Ladder) >= 2 {
		r0, rDef := a.RhoSweep[0], a.RhoSweep[0]
		for _, p := range a.RhoSweep {
			if math.Abs(p.Rho-simulate.DefaultRho) < 1e-9 {
				rDef = p
			}
		}
		ex["kopula"] = WorkedExample{
			Substitution: fmt.Sprintf("rho = %s  →  %s: %s x %s + %s x %s",
				n(simulate.DefaultRho, 2), tr2(lang, "laten", "latent"), n(simulate.DefaultRho, 2), "z_BE",
				n(math.Sqrt(1-simulate.DefaultRho*simulate.DefaultRho), 3), "epsilon"),
			Result: fmt.Sprintf("sd %s → %s, %s %s", n(r0.StdDev, 2), n(rDef.StdDev, 2), tr2(lang, "korelasi terealisasi", "realised correlation"), n(rDef.Realised, 3)),
			Comment: model.Text{
				ID: fmt.Sprintf("Dengan rho %s, simpangan baku durasi proyek melebar %s. Korelasi peringkat yang benar-benar muncul antar-aktivitas peran sama %s - dekat dengan rho kuadrat %s, sesuai teori kopula.", n(simulate.DefaultRho, 2), render.Pct(rDef.StdDev/r0.StdDev-1, 1, lang), n(rDef.Realised, 3), n(simulate.DefaultRho*simulate.DefaultRho, 2)),
				EN: fmt.Sprintf("At rho %s the standard deviation of project duration widens by %s. The rank correlation actually appearing between same-role activities is %s - close to rho squared, %s, as copula theory predicts.", n(simulate.DefaultRho, 2), render.Pct(rDef.StdDev/r0.StdDev-1, 1, lang), n(rDef.Realised, 3), n(simulate.DefaultRho*simulate.DefaultRho, 2)),
			},
		}
	}

	if len(a.Ladder) > 0 {
		fin := a.Final()
		ex["jcl"] = WorkedExample{
			Substitution: fmt.Sprintf("JCL(%d, %s) = %s;   JCL(P80 %s, P80 %s) = %s",
				a.Plan.Duration, render.RpShort(model.TotalAuthorised, lang), render.Pct(fin.JCL, 2, lang),
				n(fin.DurP80, 0), render.RpShort(fin.CostP80, lang), render.Pct(fin.JointAtP80, 1, lang)),
			Result: fmt.Sprintf("frontier_70(%s) = %s", n(a.JCL70.Duration, 0), rp(a.JCL70.Budget)),
			Comment: model.Text{
				ID: fmt.Sprintf("Dua P80 yang dilaporkan berdampingan hanya memberi keyakinan bersama %s, bukan 80%%. Tenggat terpendek yang masih bisa mencapai JCL 70%% adalah %s hari kerja, dengan anggaran minimum %s.", render.Pct(fin.JointAtP80, 1, lang), n(a.JCL70.Duration, 0), rp(a.JCL70.Budget)),
				EN: fmt.Sprintf("Two P80s reported side by side give only %s joint confidence, not 80%%. The shortest deadline that can still reach a 70%% JCL is %s working days, with a minimum budget of %s.", render.Pct(fin.JointAtP80, 1, lang), n(a.JCL70.Duration, 0), rp(a.JCL70.Budget)),
			},
		}
	}
	closureExamples(a, lang, ex)
}
