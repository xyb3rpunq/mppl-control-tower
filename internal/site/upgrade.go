package site

import (
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// buildUpgrade menjalankan analisis lanjutan: levelling sumber daya, kompresi
// jadwal, dan simulasi terpadu. Dipisah dari Build supaya urutan ketergantungan
// terlihat jelas - semuanya membutuhkan kalender dan jadwal CPM yang sudah jadi.
func buildUpgrade(a *Analysis) error {
	opts := level.Options{Calendar: a.Calendar, Capacity: model.Capacity, UseWindows: true}
	lv, err := level.Run(model.Activities, opts)
	if err != nil {
		return err
	}
	a.Level = lv
	a.LevelProfile = lv.Profile()
	if a.LevelWhy, err = level.Explain(model.Activities, opts); err != nil {
		return err
	}

	// Peran kritis: peran dominan dari aktivitas yang paling lama menunggu
	// sumber daya. Hari tunggu yang terbawa dari pendahulu tidak dihitung,
	// karena itu akibat, bukan sebab.
	byID := model.ActivityByID()
	a.RoleWait = map[model.Role]int{}
	for id, t := range lv.Tasks {
		if t.WaitDays <= 0 {
			continue
		}
		a.RoleWait[byID[id].DominantRole()] += t.WaitDays
	}
	best := 0
	for r, w := range a.RoleWait {
		if w > best || (w == best && r < a.CriticalRole) {
			best, a.CriticalRole = w, r
		}
	}

	if a.Crash, err = compress.Crash(model.Activities, model.RateCard); err != nil {
		return err
	}
	if a.FastTracks, err = compress.FastTrackCandidates(model.Activities, model.RateCard); err != nil {
		return err
	}
	a.FastViable = compress.ViableCandidates(a.FastTracks)
	if a.FastAllDur, err = compress.ApplyFastTracks(model.Activities, a.FastViable); err != nil {
		return err
	}

	base := simulate.IntegratedConfig{
		Iterations: simulate.Defaults().Iterations,
		Seed:       simulate.Defaults().Seed,
		Rho:        simulate.DefaultRho,
		Calendar:   a.Calendar,
		Capacity:   model.Capacity,
		Budget:     model.TotalAuthorised,
		Deadline:   float64(a.Plan.Duration),
	}
	if a.Ladder, err = simulate.Ladder(model.Activities, base); err != nil {
		return err
	}

	for _, rho := range []float64{0, 0.25, 0.5, 0.75} {
		c := base
		// Iterasi dan benih sama dengan tangga, sehingga baris rho baku identik
		// dengan lapisan korelasi dan tidak ada dua angka yang tampak bertentangan.
		c.Layer, c.Rho = simulate.LayerCorrelated, rho
		r, err := simulate.RunIntegrated(model.Activities, c)
		if err != nil {
			return err
		}
		a.RhoSweep = append(a.RhoSweep, RhoPoint{
			Rho: rho, P80: r.DurP80, StdDev: simulate.StdDev(r.Durations),
			Realised: r.RealisedSameRole, OnTime: r.OnTime,
		})
	}

	final := a.Final()
	var ds []float64
	lo := math.Floor(final.DurP50)
	hi := final.Durations[len(final.Durations)-1]
	for d := lo; d <= hi; d++ {
		ds = append(ds, d)
	}
	a.Frontier = final.Frontier(0.7, ds)
	for _, p := range a.Frontier {
		if p.Feasible {
			a.JCL70 = p
			break
		}
	}
	a.Density = final.Density(36, 24)
	return nil
}

// CrashStepsUpTo mengembalikan langkah crashing sampai n hari.
func (a *Analysis) CrashStepsUpTo(n int) []compress.Step {
	if n > len(a.Crash.Steps) {
		n = len(a.Crash.Steps)
	}
	return a.Crash.Steps[:n]
}

// MovedTasks mengembalikan aktivitas yang bergeser akibat levelling.
func (a *Analysis) MovedTasks() []level.Task { return a.Level.Moved() }

// SortedRoleWait mengembalikan hari tunggu per peran, terbesar lebih dulu.
func (a *Analysis) SortedRoleWait() []RoleWaitRow {
	var out []RoleWaitRow
	for r, w := range a.RoleWait {
		out = append(out, RoleWaitRow{Role: r, Days: w})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Days != out[j].Days {
			return out[i].Days > out[j].Days
		}
		return out[i].Role < out[j].Role
	})
	return out
}

// RoleWaitRow adalah satu baris tabel hari tunggu per peran.
type RoleWaitRow struct {
	Role model.Role
	Days int
}

// upgradeFindings menurunkan temuan dari analisis lanjutan.
func upgradeFindings(a *Analysis) []Finding {
	var out []Finding
	lv := a.Level

	if lv.Duration > a.Plan.Duration {
		extra := lv.Duration - a.Plan.Duration
		save5, _ := a.Crash.CostToSave(5)
		out = append(out, Finding{
			Key: "jadwal-tak-terjalankan", Severity: "kritis", Route: "/optimasi/",
			Title: model.Text{
				ID: "Jadwal 85 hari hanya sah di atas kertas",
				EN: "The 85-day schedule is only valid on paper",
			},
			Detail: model.Text{
				ID: "Begitu kapasitas orang dihormati, jadwal yang bisa dijalankan menjadi " + fmtInt(float64(lv.Duration)) + " hari kerja dan selesai " + fmtDate(a.FinishISO(lv.Duration)) + " - tambahan " + fmtInt(float64(extra)) + " hari. Sebanyak " + fmtInt(float64(a.LevelWhy.CapacityOnly-a.LevelWhy.CPM)) + " hari berasal dari bentrokan kapasitas dan " + fmtInt(float64(a.LevelWhy.WithWindows-a.LevelWhy.CapacityOnly)) + " hari dari periode ujian. Peran yang paling banyak membuat pekerjaan menunggu: " + string(a.CriticalRole) + ".",
				EN: "Once people's capacity is respected, the deliverable schedule becomes " + fmtInt(float64(lv.Duration)) + " working days, finishing " + fmtDateEN(a.FinishISO(lv.Duration)) + " - " + fmtInt(float64(extra)) + " days more. " + fmtInt(float64(a.LevelWhy.CapacityOnly-a.LevelWhy.CPM)) + " days come from capacity clashes and " + fmtInt(float64(a.LevelWhy.WithWindows-a.LevelWhy.CapacityOnly)) + " from the exam period. The role that makes work wait the most: " + string(a.CriticalRole) + ".",
			},
			Metric: model.Text{ID: "levelling SGS vs CPM", EN: "SGS levelling vs CPM"},
			Action: model.Text{
				ID: "Tambah kapasitas pada peran " + string(a.CriticalRole) + " selama fase pengembangan, dan geser pekerjaan ber-float agar tidak beririsan dengan ujian. Kalau tanggal tetap dikunci, lima hari pertama bisa dibeli lewat crashing seharga " + fmtRp(save5) + ".",
				EN: "Add capacity to the " + string(a.CriticalRole) + " role during development and move float-bearing work out of the exam window. If the date stays locked, the first five days can be bought through crashing for " + fmtRp(save5) + ".",
			},
		})
	}

	final := a.Final()
	if final.JCL < 0.5 {
		act := model.Text{
			ID: "Tidak ada kombinasi tenggat dan anggaran dalam rentang simulasi yang mencapai JCL 70%. Proyek ini perlu dirancang ulang lingkupnya, bukan sekadar diberi cadangan.",
			EN: "No deadline-budget combination in the simulated range reaches a 70% JCL. The scope needs redesigning, not just a bigger reserve.",
		}
		if a.JCL70.Feasible {
			act = model.Text{
				ID: "Untuk keyakinan 70% selesai tepat waktu DAN tepat anggaran, komitmennya adalah " + fmtInt(a.JCL70.Duration) + " hari kerja (" + fmtDate(a.FinishISO(int(a.JCL70.Duration))) + ") dengan anggaran " + fmtRp(a.JCL70.Budget) + ". Sampaikan pasangan angka itu, bukan dua P80 yang dilaporkan terpisah.",
				EN: "For 70% confidence of finishing on time AND on budget, the commitment is " + fmtInt(a.JCL70.Duration) + " working days (" + fmtDateEN(a.FinishISO(int(a.JCL70.Duration))) + ") with a budget of " + fmtRp(a.JCL70.Budget) + ". Report that pair, not two P80s reported separately.",
			}
		}
		out = append(out, Finding{
			Key: "jcl-rendah", Severity: "kritis", Route: "/simulasi-terpadu/",
			Title: model.Text{
				ID: "Peluang tepat waktu dan tepat anggaran sekaligus nyaris nol",
				EN: "The odds of hitting both the deadline and the budget are close to zero",
			},
			Detail: model.Text{
				ID: "Dengan korelasi peran, kejadian risiko, dan kapasitas nyata dihitung bersama, peluang selesai dalam " + fmtInt(float64(a.Plan.Duration)) + " hari kerja DAN dalam pagu " + fmtRp(model.TotalAuthorised) + " adalah " + fmtPct(final.JCL) + ". P80 durasi " + fmtInt(final.DurP80) + " hari kerja, P80 biaya " + fmtRp(final.CostP80) + ".",
				EN: "With role correlation, risk events, and real capacity counted together, the chance of finishing within " + fmtInt(float64(a.Plan.Duration)) + " working days AND within the " + fmtRp(model.TotalAuthorised) + " cap is " + fmtPct(final.JCL) + ". P80 duration " + fmtInt(final.DurP80) + " working days, P80 cost " + fmtRp(final.CostP80) + ".",
			},
			Metric: model.Text{ID: "Joint Confidence Level = P(durasi <= T dan biaya <= C)", EN: "Joint Confidence Level = P(duration <= T and cost <= C)"},
			Action: act,
		})
	}

	if len(a.Ladder) > 1 {
		l0, l1 := a.Ladder[0], a.Ladder[1]
		sd0, sd1 := simulate.StdDev(l0.Durations), simulate.StdDev(l1.Durations)
		if sd0 > 0 && sd1/sd0-1 >= 0.1 {
			out = append(out, Finding{
				Key: "korelasi-diabaikan", Severity: "sedang", Route: "/simulasi-terpadu/",
				Title: model.Text{
					ID: "Mengabaikan korelasi membuat ketidakpastian tampak lebih kecil",
					EN: "Ignoring correlation makes uncertainty look smaller than it is",
				},
				Detail: model.Text{
					ID: "Ketika aktivitas yang dikerjakan orang yang sama dibiarkan bergerak bersama (rho " + render.Num(simulate.DefaultRho, 2, "id") + "), simpangan baku durasi proyek naik dari " + fmtNum(sd0) + " menjadi " + fmtNum(sd1) + " hari kerja - " + fmtPct(sd1/sd0-1) + " lebih lebar. Simulasi independen menyembunyikan ekor itu.",
					EN: "When activities done by the same person move together (rho " + render.Num(simulate.DefaultRho, 2, "en") + "), the standard deviation of project duration rises from " + fmtNum(sd0) + " to " + fmtNum(sd1) + " working days - " + fmtPct(sd1/sd0-1) + " wider. An independent simulation hides that tail.",
				},
				Metric: model.Text{ID: "simpangan baku durasi, rho 0 vs rho baku", EN: "duration standard deviation, rho 0 vs default rho"},
				Action: model.Text{
					ID: "Laporkan sebaran dari simulasi berkorelasi. Untuk menurunkan korelasinya sendiri, jangan menaruh seluruh rantai kritis pada satu orang.",
					EN: "Report the spread from the correlated simulation. To reduce the correlation itself, do not put the whole critical chain on one person.",
				},
			})
		}
	}
	return out
}
