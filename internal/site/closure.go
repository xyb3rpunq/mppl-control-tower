package site

import (
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/xyb3rpunq/mppl-control-tower/internal/gert"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// RiskPoint adalah hasil lapisan risiko untuk satu nilai lambda kopula risiko.
type RiskPoint struct {
	Loading  float64
	DurP80   float64
	DurP95   float64
	CostMean float64
	CostP95  float64
	Phi      float64
	// ManyRisks adalah peluang empat risiko atau lebih terjadi dalam satu proyek.
	ManyRisks float64
	// Count[k] adalah porsi iterasi dengan tepat k risiko terjadi.
	Count []float64
}

// GERTRow adalah analisis satu putaran rework: bentuk tertutup GERT
// dibandingkan dengan hasil Monte Carlo.
type GERTRow struct {
	Loop        model.ReworkLoop
	ReworkDays  float64 // hari per putaran pada durasi paling mungkin
	ReworkCost  float64 // biaya tenaga kerja per putaran
	Reduced     gert.W  // transmitansi hasil reduksi Mason
	Cycles      float64 // E[N] analitik
	AtLeastTwo  float64 // P(N >= 2)
	Q90         int     // jumlah putaran pada keyakinan 90%
	MCCycles    float64 // rerata putaran di simulasi
	CheckActive bool    // pemeriksaan belum lewat pada tanggal data
}

// ManyRiskThreshold adalah ambang "banyak risiko sekaligus" pada tabel kepekaan.
const ManyRiskThreshold = 4

// runSimulations menjalankan seluruh simulasi Monte Carlo situs secara
// paralel: kelima lapisan tangga, sapuan rho, sapuan lambda risiko, dan dua
// prakiraan berjalan. Setiap simulasi punya generator acak berbenih sendiri
// dan hanya MEMBACA data model, sehingga hasilnya identik dengan eksekusi
// berurutan - paralelisme hanya memangkas waktu build.
func runSimulations(a *Analysis) error {
	base := a.baseSimConfig()

	fl, err := simulate.PrepareInFlight(model.Activities, a.Calendar, a.StatusDay, 0, a.EVM.ACAt)
	if err != nil {
		return err
	}
	a.InFlight = fl
	// Pembanding tanpa belajar: bobot keyakinan awal sangat besar sehingga
	// Z mendekati nol, dan faktor ujian kembali ke asumsi perencanaan.
	prior, err := simulate.PrepareInFlight(model.Activities, a.Calendar, a.StatusDay, 1e12, a.EVM.ACAt)
	if err != nil {
		return err
	}

	rhos := []float64{0, 0.25, 0.5, 0.75}
	lambdas := []float64{0, 0.3, model.RiskLoading, 0.9}
	ladder := make([]simulate.IntegratedResult, len(simulate.Layers))
	rhoRes := make([]simulate.IntegratedResult, len(rhos))
	lamRes := make([]simulate.IntegratedResult, len(lambdas))

	var jobs []func() error
	for i, l := range simulate.Layers {
		i, c := i, base
		c.Layer = l
		jobs = append(jobs, func() (err error) { ladder[i], err = simulate.RunIntegrated(model.Activities, c); return })
	}
	for i, rho := range rhos {
		// Iterasi dan benih sama dengan tangga, sehingga baris rho baku identik
		// dengan lapisan korelasi dan tidak ada dua angka yang tampak bertentangan.
		i, c := i, base
		c.Layer, c.Rho = simulate.LayerCorrelated, rho
		jobs = append(jobs, func() (err error) { rhoRes[i], err = simulate.RunIntegrated(model.Activities, c); return })
	}
	for i, lam := range lambdas {
		i, c := i, base
		c.Layer, c.RiskLoading = simulate.LayerRisks, lam
		jobs = append(jobs, func() (err error) { lamRes[i], err = simulate.RunIntegrated(model.Activities, c); return })
	}
	fc := base
	fc.Layer, fc.InFlight, fc.ExamFactor = simulate.LayerResources, fl, fl.ExamFactor
	jobs = append(jobs, func() (err error) { a.Forecast, err = simulate.RunIntegrated(model.Activities, fc); return })
	if a.Decision, err = prepareDecision(a, fl); err != nil {
		return err
	}
	jobs = append(jobs, decisionJobs(a, fl, fc)...)
	pc := base
	pc.Layer, pc.InFlight, pc.ExamFactor = simulate.LayerResources, prior, 0
	jobs = append(jobs, func() (err error) { a.ForecastPrior, err = simulate.RunIntegrated(model.Activities, pc); return })

	if err := parallel(jobs); err != nil {
		return err
	}

	a.Ladder = ladder
	for i, r := range rhoRes {
		a.RhoSweep = append(a.RhoSweep, RhoPoint{
			Rho: rhos[i], P80: r.DurP80, StdDev: simulate.StdDev(r.Durations),
			Realised: r.RealisedSameRole, OnTime: r.OnTime,
		})
	}
	for i, r := range lamRes {
		many := 0.0
		for k := ManyRiskThreshold; k < len(r.RiskCount); k++ {
			many += r.RiskCount[k]
		}
		a.RiskSweep = append(a.RiskSweep, RiskPoint{
			Loading: lambdas[i], DurP80: r.DurP80, DurP95: r.DurP95,
			CostMean: r.CostMean, CostP95: r.CostP95, Phi: r.RiskPhi, ManyRisks: many,
			Count: r.RiskCount,
		})
	}
	return nil
}

// parallel menjalankan pekerjaan sekaligus dan mengembalikan galat pertama.
func parallel(jobs []func() error) error {
	errs := make([]error, len(jobs))
	var wg sync.WaitGroup
	for i, job := range jobs {
		wg.Add(1)
		go func(i int, job func() error) {
			defer wg.Done()
			errs[i] = job()
		}(i, job)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// finishClosure menurunkan analisis penutup celah dari hasil simulasi.
func finishClosure(a *Analysis) error {
	byID := model.ActivityByID()
	rework := a.Ladder[simulate.LayerRework]
	for _, l := range model.ReworkLoops {
		var days, rc float64
		for _, p := range l.Parts {
			act := byID[p.Activity]
			d := p.Fraction * float64(act.Duration)
			days += d
			rc += d * act.LabourCost(model.RateCard) / math.Max(1, float64(act.Duration))
		}
		a.GERT = append(a.GERT, GERTRow{
			Loop: l, ReworkDays: days, ReworkCost: rc,
			Reduced:     gert.SelfLoop(l.FailProb, days),
			Cycles:      gert.ExpectedCycles(l.FailProb),
			AtLeastTwo:  gert.AtLeast(l.FailProb, 2),
			Q90:         gert.CycleQuantile(l.FailProb, 0.9),
			MCCycles:    rework.ReworkCycles[l.ID],
			CheckActive: a.InFlight == nil || !a.InFlight.ClosedLoop[l.ID],
		})
	}

	if len(a.Forecast.Durations) > 0 {
		var ds []float64
		for d := math.Floor(a.Forecast.DurP50); d <= a.Forecast.Durations[len(a.Forecast.Durations)-1]; d++ {
			ds = append(ds, d)
		}
		a.ForecastFrontier = a.Forecast.Frontier(0.7, ds)
		for _, p := range a.ForecastFrontier {
			if p.Feasible {
				a.ForecastJCL70 = p
				break
			}
		}
	}

	s := a.Snapshot
	if s.SPIt > 0 {
		a.IEACt = s.AtDay + (float64(a.Plan.Duration)-s.ES)/s.SPIt
	}
	finishDecision(a)
	return nil
}

// EvidenceRow adalah satu aktivitas selesai yang menjadi bukti kalibrasi.
type EvidenceRow struct {
	ID       string
	M        int
	PERTMean float64
	Actual   int
	Ratio    float64 // aktual / rerata PERT
	InExam   bool    // beririsan dengan jendela ujian yang sudah lewat
}

// EvidenceRows mengembalikan aktivitas selesai pada tanggal data, urut kode.
func (a *Analysis) EvidenceRows() []EvidenceRow {
	if a.InFlight == nil {
		return nil
	}
	var wins [][2]string
	for _, w := range model.AvailabilityWindows {
		wins = append(wins, [2]string{w.From, w.To})
	}
	var out []EvidenceRow
	for i, act := range model.Activities {
		st := a.InFlight.State[i]
		if st.Kind != simulate.Completed || act.Milestone {
			continue
		}
		mean := (float64(act.Optimistic) + 4*float64(act.Duration) + float64(act.Pessimistic)) / 6
		row := EvidenceRow{ID: act.ID, M: act.Duration, PERTMean: mean, Actual: act.Actual.Duration}
		if mean > 0 {
			row.Ratio = float64(act.Actual.Duration) / mean
		}
		from, to := a.Calendar.ISOAt(st.Start), a.Calendar.ISOAt(st.Finish-1)
		for _, w := range wins {
			if from <= w[1] && to >= w[0] {
				row.InExam = true
			}
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// RunningRow adalah satu aktivitas yang sedang berjalan pada tanggal data.
type RunningRow struct {
	ID         string
	Start      int
	Elapsed    float64
	M          int
	CondMean   float64 // E[durasi | durasi > hari berjalan]
	Remaining  float64 // sisa harapan setelah faktor kalibrasi
	PriorMeans float64 // rerata PERT tanpa syarat, pembanding
}

// RunningRows menghitung durasi bersyarat aktivitas yang sedang berjalan
// dengan integrasi numerik atas kuantil bersyarat.
func (a *Analysis) RunningRows() []RunningRow {
	if a.InFlight == nil {
		return nil
	}
	s := simulate.NewSampler(model.Activities)
	var out []RunningRow
	for i, act := range model.Activities {
		st := a.InFlight.State[i]
		if st.Kind != simulate.InProgress {
			continue
		}
		const k = 400
		fe := s.CDF(i, st.Elapsed)
		var sum float64
		for j := 0; j < k; j++ {
			u := fe + (float64(j)+0.5)/k*(1-fe)
			sum += s.Value(i, u, "pert")
		}
		cond := sum / k
		out = append(out, RunningRow{
			ID: act.ID, Start: st.Start, Elapsed: st.Elapsed, M: act.Duration,
			CondMean:   cond,
			Remaining:  (cond - st.Elapsed) * a.InFlight.DurationFactor,
			PriorMeans: s.PERTMean(i),
		})
	}
	return out
}

// RentalDaily mengembalikan jumlah tarif harian seluruh sewa & langganan.
func (a *Analysis) RentalDaily() float64 {
	var t float64
	for _, r := range a.Rentals {
		t += r.Rate
	}
	return t
}

// TradeOffNet mengembalikan biaya total bersih mempercepat proyek d hari
// dari durasi normal, menurut LP biaya total.
func (a *Analysis) TradeOffNet(days int) float64 {
	if len(a.Exact.Points) == 0 {
		return 0
	}
	p, ok := a.Exact.PointAt(a.Exact.Normal - days)
	if !ok {
		return 0
	}
	return p.Total - a.Exact.Points[0].Total
}

// closureFindings menurunkan temuan dari analisis penutup celah.
func closureFindings(a *Analysis) []Finding {
	var out []Finding
	opt := a.LevelOpt

	if opt.Proven {
		out = append(out, Finding{
			Key: "levelling-terbukti-optimal", Severity: "baik", Route: "/optimasi/",
			Title: model.Text{
				ID: "Jadwal levelling terbukti tidak bisa diperpendek dengan mengubah urutan kerja",
				EN: "The levelled schedule is proven impossible to shorten by reordering work",
			},
			Detail: model.Text{
				ID: "Enam aturan prioritas, " + fmtInt(float64(opt.Samples)) + " daftar acak berbias, dan justifikasi maju-mundur menemukan jadwal " + fmtInt(float64(opt.Best.Duration)) + " hari kerja. Batas bawah energetik membuktikan tidak ada jadwal di bawah " + fmtInt(float64(opt.Bound.Value)) + " hari - celahnya nol. Aturan LST yang dipakai sebelumnya memberi " + fmtInt(float64(opt.Baseline.Duration)) + " hari.",
				EN: "Six priority rules, " + fmtInt(float64(opt.Samples)) + " biased random lists, and forward-backward justification find a " + fmtInt(float64(opt.Best.Duration)) + "-working-day schedule. The energetic lower bound proves no schedule can go below " + fmtInt(float64(opt.Bound.Value)) + " days - the gap is zero. The LST rule used previously gave " + fmtInt(float64(opt.Baseline.Duration)) + " days.",
			},
			Metric: model.Text{ID: "celah optimalitas = durasi terbaik - batas bawah", EN: "optimality gap = best duration - lower bound"},
			Action: model.Text{
				ID: "Berhenti mencari urutan yang lebih pintar. Jadwal ini hanya bisa diperpendek dengan menambah kapasitas - lembur atau orang baru pada peran " + string(a.CriticalRole) + ", dengan harga per hari di halaman Keputusan Sponsor - atau dengan mengubah jaringan aktivitasnya.",
				EN: "Stop looking for a smarter ordering. This schedule can only be shortened by adding capacity - overtime or a new hire in the " + string(a.CriticalRole) + " role, with prices per day on the Sponsor Decisions page - or by changing the activity network.",
			},
		})
	}

	// "Hampir impas" hanya sah bila sewa yang dihemat menutup lebih dari
	// separuh premi lima hari pertama.
	if len(a.Exact.Points) > 5 && a.TradeOffNet(5) < a.Exact.Points[5].CrashCost/2 {
		net5 := a.TradeOffNet(5)
		crash5 := a.Exact.Points[5].CrashCost
		out = append(out, Finding{
			Key: "crashing-hampir-impas", Severity: "baik", Route: "/optimasi/",
			Title: model.Text{
				ID: "Pada jaringan CPM, lima hari percepatan pertama hampir dibayar sendiri oleh sewa yang dihemat",
				EN: "On the CPM network, the first five days of acceleration nearly pay for themselves in saved rentals",
			},
			Detail: model.Text{
				ID: "Pada jaringan CPM, crashing lima hari menelan premi " + fmtRp(crash5) + ", tetapi setiap hari proyek lebih pendek juga menghemat sewa server dan langganan sampai " + fmtRp(a.RentalDaily()) + ". Menurut pemrograman linear biaya total, biaya bersihnya hanya " + fmtRp(net5) + ". " + greedyNoteID(a),
				EN: "On the CPM network, crashing five days costs " + fmtRp(crash5) + " in premiums, but every day shorter also saves up to " + fmtRp(a.RentalDaily()) + " in server rental and subscriptions. Per the total-cost linear program, the net cost is only " + fmtRp(net5) + ". " + greedyNoteEN(a),
			},
			Metric: model.Text{ID: "LP biaya total = premi lembur PP 35/2021 + tarif sewa x rentang sewa", EN: "total-cost LP = overtime premium under Government Regulation 35/2021 + rental rate x rental span"},
			Action: model.Text{
				ID: "Pakai pelajaran metodenya, bukan angkanya: nilai percepatan selalu sebagai biaya bersih setelah sewa (" + fmtRp(net5) + ", bukan " + fmtRp(crash5) + "), karena angka premi saja membuat keputusan murah tampak mahal. Jangan menjanjikan lima hari ini ke sponsor - jaringan CPM tidak bisa dijalankan; harga percepatan yang berlaku ada di halaman Keputusan Sponsor.",
				EN: "Use the method's lesson, not its figure: always value acceleration as a net cost after rentals (" + fmtRp(net5) + ", not " + fmtRp(crash5) + "), because the premium alone makes a cheap decision look expensive. Do not promise these five days to the sponsor - the CPM network cannot be executed; the acceleration prices in force are on the Sponsor Decisions page.",
			},
		})
	}

	if len(a.RiskSweep) >= 3 {
		ind, def := a.RiskSweep[0], a.RiskSweep[0]
		for _, p := range a.RiskSweep {
			if math.Abs(p.Loading-model.RiskLoading) < 1e-9 {
				def = p
			}
		}
		if def.CostP95 > ind.CostP95 {
			out = append(out, Finding{
				Key: "risiko-bergerombol", Severity: "sedang", Route: "/simulasi-terpadu/",
				Title: model.Text{
					ID: "Risiko yang berbagi sebab menebalkan ekor biaya tanpa mengubah rata-ratanya",
					EN: "Risks with shared causes thicken the cost tail without moving the average",
				},
				Detail: model.Text{
					ID: "Dengan penggerak bersama (lambda " + fmtNum(model.RiskLoading) + "), rerata biaya tetap " + fmtRp(def.CostMean) + " - sama dengan risiko saling bebas - tetapi P95 biaya naik dari " + fmtRp(ind.CostP95) + " menjadi " + fmtRp(def.CostP95) + ". Korelasi phi antar-risiko satu penggerak yang terealisasi " + fmtNum(def.Phi) + ".",
					EN: "With shared drivers (lambda " + fmtNum(model.RiskLoading) + "), mean cost stays at " + fmtRp(def.CostMean) + " - the same as independent risks - but P95 cost rises from " + fmtRp(ind.CostP95) + " to " + fmtRp(def.CostP95) + ". The realised phi correlation between same-driver risks is " + fmtNum(def.Phi) + ".",
				},
				Metric: model.Text{ID: "P95 biaya, lambda 0 vs lambda baku", EN: "P95 cost, lambda 0 vs default lambda"},
				Action: model.Text{
					ID: "Mitigasi penggeraknya, bukan risikonya satu per satu. Menaikkan cakupan uji otomatis menekan R03, R04, dan R12 sekaligus karena ketiganya lahir dari disiplin rekayasa yang sama.",
					EN: "Mitigate the driver, not each risk separately. Raising automated test coverage suppresses R03, R04, and R12 together because all three stem from the same engineering discipline.",
				},
			})
		}
	}

	if len(a.GERT) > 0 {
		var days, cost float64
		for _, g := range a.GERT {
			days += g.Reduced.Mean()
			cost += g.Cycles * g.ReworkCost
		}
		g := a.GERT[0]
		out = append(out, Finding{
			Key: "rework-berulang", Severity: "sedang", Route: "/simulasi-terpadu/",
			Title: model.Text{
				ID: "Pemeriksaan yang bisa gagal berulang kali belum ada di jadwal mana pun",
				EN: "Checks that can fail repeatedly are missing from every schedule",
			},
			Detail: model.Text{
				ID: "Jaringan CPM hanya memuat satu kali perbaikan. Reduksi GERT atas " + fmtInt(float64(len(a.GERT))) + " putaran rework memberi tambahan harapan " + fmtNum(days) + " hari kerja dan " + fmtRp(cost) + ". Untuk regresi pasca perbaikan bug, peluang butuh dua putaran tambahan atau lebih adalah " + fmtPct(g.AtLeastTwo) + ".",
				EN: "The CPM network holds only one round of fixing. GERT reduction of " + fmtInt(float64(len(a.GERT))) + " rework loops gives an expected " + fmtNum(days) + " extra working days and " + fmtRp(cost) + ". For regression after bug fixing, the chance of needing two or more extra rounds is " + fmtPct(g.AtLeastTwo) + ".",
			},
			Metric: model.Text{ID: "E[waktu tambahan] = p x r / (1 - p) per putaran", EN: "E[extra time] = p x r / (1 - p) per loop"},
			Action: model.Text{
				ID: "Tekan peluang gagalnya, bukan menambah buffer. Menaikkan cakupan uji otomatis dari 41% menurunkan p, dan setiap penurunan p memotong rerata putaran secara tidak linear: p/(1-p).",
				EN: "Push the failure chance down rather than adding buffer. Raising automated test coverage from 41% lowers p, and every drop in p cuts expected loops non-linearly: p/(1-p).",
			},
		})
	}

	if d := a.Decision; d != nil && len(d.Options) > 1 && a.ForecastJCL70.Feasible {
		base := d.Options[0]
		var lines []string
		var linesEN []string
		for _, o := range d.Options[1:] {
			if o.Assumption.ID != "" || !o.JCL70.Feasible {
				continue
			}
			lines = append(lines, lowerFirst(o.Name.ID)+" memajukan "+fmtInt(o.DaysEarlier)+" hari (+"+fmtRp(o.ExtraBudget)+")")
			linesEN = append(linesEN, lowerFirst(o.Name.EN)+" gains "+fmtInt(o.DaysEarlier)+" days (+"+fmtRp(o.ExtraBudget)+")")
		}
		missed := model.Text{}
		if d.PlanMissed > 0 {
			missed = model.Text{
				ID: " Rencana lembur dari hari pertama proyek (" + fmtInt(float64(a.Overtime.Levelled)) + " → " + fmtInt(float64(a.Overtime.MinDuration)) + " hari) tidak bisa dibeli lagi: " + fmtInt(float64(d.PlanMissed)) + " hari-peran lemburnya jatuh sebelum tanggal data.",
				EN: " The overtime plan from the project's first day (" + fmtInt(float64(a.Overtime.Levelled)) + " → " + fmtInt(float64(a.Overtime.MinDuration)) + " days) can no longer be bought: " + fmtInt(float64(d.PlanMissed)) + " of its overtime role-days fall before the data date.",
			}
		}
		title := model.Text{ID: "Dari tanggal data, tidak ada opsi yang memajukan komitmen JCL 70%", EN: "From the data date, no option advances the 70% JCL commitment"}
		act := model.Text{
			ID: "Pertahankan komitmen tanpa percepatan: " + fmtInt(base.JCL70.Duration) + " hari kerja dengan " + fmtRp(base.JCL70.Budget) + ".",
			EN: "Keep the commitment without acceleration: " + fmtInt(base.JCL70.Duration) + " working days with " + fmtRp(base.JCL70.Budget) + ".",
		}
		if c, ok := d.CheapestOption(); ok {
			title = model.Text{
				ID: "Dari tanggal data, percepatan termurah memajukan " + fmtInt(c.DaysEarlier) + " hari seharga " + fmtRp(c.PricePerDay) + " per hari",
				EN: "From the data date, the cheapest acceleration gains " + fmtInt(c.DaysEarlier) + " days at " + fmtRp(c.PricePerDay) + " per day",
			}
			act = model.Text{
				ID: "Beri sponsor dua harga, bukan satu janji: tanpa percepatan " + fmtInt(base.JCL70.Duration) + " hari kerja dengan " + fmtRp(base.JCL70.Budget) + ", atau " + lowerFirst(c.Name.ID) + " untuk " + fmtInt(c.JCL70.Duration) + " hari kerja dengan " + fmtRp(c.JCL70.Budget) + ". Putuskan dengan membandingkan " + fmtRp(c.PricePerDay) + " per hari dengan nilai satu hari lebih cepat bagi sponsor.",
				EN: "Give the sponsor two prices, not one promise: no acceleration at " + fmtInt(base.JCL70.Duration) + " working days with " + fmtRp(base.JCL70.Budget) + ", or " + lowerFirst(c.Name.EN) + " for " + fmtInt(c.JCL70.Duration) + " working days with " + fmtRp(c.JCL70.Budget) + ". Decide by comparing " + fmtRp(c.PricePerDay) + " per day with what a day earlier is worth to the sponsor.",
			}
		}
		out = append(out, Finding{
			Key: "percepatan-tanggal-data", Severity: "tinggi", Route: "/keputusan/",
			Title: title,
			Detail: model.Text{
				ID: "Keputusan percepatan diambil pada tanggal data, jadi dihitung dari sana." + missed.ID + " Tanpa percepatan, lantai jadwalnya " + fmtInt(float64(base.Floor)) + " hari kerja dan komitmen JCL 70% " + fmtInt(base.JCL70.Duration) + " hari. Pada titik JCL 70%: " + strings.Join(lines, "; ") + ".",
				EN: "The acceleration decision is taken at the data date, so it is computed from there." + missed.EN + " Without acceleration the schedule floor is " + fmtInt(float64(base.Floor)) + " working days and the 70% JCL commitment " + fmtInt(base.JCL70.Duration) + " days. At the 70% JCL: " + strings.Join(linesEN, "; ") + ".",
			},
			Metric: model.Text{ID: "harga per hari = selisih anggaran JCL 70% / hari yang dimajukan", EN: "price per day = 70% JCL budget difference / days gained"},
			Action: act,
		})
	}

	if fl := a.InFlight; fl != nil && len(a.Forecast.Durations) > 0 {
		fc := a.Forecast
		act := model.Text{
			ID: "Tidak ada kombinasi tenggat dan anggaran dari tanggal data yang mencapai JCL 70% dalam rentang simulasi.",
			EN: "No deadline-budget combination from the data date reaches a 70% JCL within the simulated range.",
		}
		if a.ForecastJCL70.Feasible {
			act = model.Text{
				ID: "Perbarui komitmen ke sponsor memakai prakiraan berjalan: " + fmtInt(a.ForecastJCL70.Duration) + " hari kerja (" + fmtDate(a.FinishISO(int(a.ForecastJCL70.Duration))) + ") dengan anggaran " + fmtRp(a.ForecastJCL70.Budget) + " untuk keyakinan bersama 70%. IEAC(t) Earned Schedule " + fmtNum(a.IEACt) + " hari hanya memperpanjang tren SPI dan buta terhadap kapasitas, ujian, dan risiko yang belum terjadi.",
				EN: "Update the sponsor commitment from the in-flight forecast: " + fmtInt(a.ForecastJCL70.Duration) + " working days (" + fmtDateEN(a.FinishISO(int(a.ForecastJCL70.Duration))) + ") with a budget of " + fmtRp(a.ForecastJCL70.Budget) + " for 70% joint confidence. The Earned Schedule IEAC(t) of " + fmtNum(a.IEACt) + " days only extends the SPI trend and is blind to capacity, exams, and risks yet to happen.",
			}
		}
		out = append(out, Finding{
			Key: "prakiraan-berjalan", Severity: "kritis", Route: "/prakiraan/",
			Title: model.Text{
				ID: "Dari tanggal data, P80 penyelesaian jauh melampaui prakiraan Earned Value",
				EN: "From the data date, the P80 finish lies far beyond the Earned Value forecast",
			},
			Detail: model.Text{
				ID: "Dengan " + fmtInt(float64(len(fl.Completed))) + " simpul selesai dikunci pada realisasinya, prakiraan berjalan memberi P50 " + fmtInt(fc.DurP50) + " dan P80 " + fmtInt(fc.DurP80) + " hari kerja (" + fmtDate(a.FinishISO(int(fc.DurP80))) + "), biaya P80 " + fmtRp(fc.CostP80) + ". Earned Schedule saja memprakirakan " + fmtNum(a.IEACt) + " hari, dan EAC tipikal " + fmtRp(a.Snapshot.EACTypical) + ".",
				EN: "With " + fmtInt(float64(len(fl.Completed))) + " completed nodes locked to their actuals, the in-flight forecast gives P50 " + fmtInt(fc.DurP50) + " and P80 " + fmtInt(fc.DurP80) + " working days (" + fmtDateEN(a.FinishISO(int(fc.DurP80))) + "), P80 cost " + fmtRp(fc.CostP80) + ". Earned Schedule alone forecasts " + fmtNum(a.IEACt) + " days, and the typical EAC is " + fmtRp(a.Snapshot.EACTypical) + ".",
			},
			Metric: model.Text{ID: "simulasi terpadu dari tanggal data vs IEAC(t) = AT + (PD - ES) / SPI(t)", EN: "integrated simulation from the data date vs IEAC(t) = AT + (PD - ES) / SPI(t)"},
			Action: act,
		})

		if fl.ExamEvidence > 0 {
			sev := "sedang"
			out = append(out, Finding{
				Key: "ujian-terkalibrasi", Severity: sev, Route: "/prakiraan/",
				Title: model.Text{
					ID: "Realisasi selama UTS membantah asumsi kapasitas ujian 40%",
					EN: "Actuals during the midterm exams contradict the 40% exam-capacity assumption",
				},
				Detail: model.Text{
					ID: fmtInt(float64(fl.ExamEvidence)) + " aktivitas beririsan dengan UTS resmi 3-15 November 2025. Laju kerjanya " + fmtPct(fl.ExamObserved) + " dari laju di luar ujian. Dengan bobot kredibilitas " + fmtPct(fl.ExamCredibility) + ", faktor kapasitas ujian diperbarui dari " + fmtPct(model.ExamCapacityFactor) + " menjadi " + fmtPct(fl.ExamFactor) + " untuk UAS mendatang.",
					EN: fmtInt(float64(fl.ExamEvidence)) + " activities overlapped the official midterms of 3-15 November 2025. Their work rate was " + fmtPct(fl.ExamObserved) + " of the rate outside exams. At a credibility weight of " + fmtPct(fl.ExamCredibility) + ", the exam capacity factor is updated from " + fmtPct(model.ExamCapacityFactor) + " to " + fmtPct(fl.ExamFactor) + " for the upcoming finals.",
				},
				Metric: model.Text{ID: "faktor = Z x teramati + (1 - Z) x asumsi, Z = tau2 / (tau2 + Var)", EN: "factor = Z x observed + (1 - Z) x assumed, Z = tau2 / (tau2 + Var)"},
				Action: model.Text{
					ID: "Jangan menambah buffer ujian berdasarkan asumsi lama. Kumpulkan data kehadiran nyata saat UAS; bila tim tetap produktif, jadwal perencanaan terlalu pesimistis di periode ujian.",
					EN: "Do not add exam buffer from the old assumption. Collect real attendance during the finals; if the team stays productive, the planning schedule is too pessimistic around exams.",
				},
			})
		}
	}
	return out
}

// greedyNoteID dan greedyNoteEN menjelaskan hasil perbandingan serakah dengan
// LP sesuai angkanya: dengan biaya lembur per hari, serakah bisa saja optimal.
func greedyNoteID(a *Analysis) string {
	if a.Exact.GreedyOptimal {
		return "Dengan biaya lembur per hari, kurva serakah sama dengan optimum eksak di setiap durasi - hanya LP yang bisa membuktikannya."
	}
	return "Kurva crashing serakah melebihi optimum eksak sampai " + fmtRp(a.Exact.MaxGreedyExcess) + " pada satu titik."
}

func greedyNoteEN(a *Analysis) string {
	if a.Exact.GreedyOptimal {
		return "With per-day overtime costs, the greedy curve equals the exact optimum at every duration - only the LP can prove it."
	}
	return "The greedy crashing curve exceeds the exact optimum by up to " + fmtRp(a.Exact.MaxGreedyExcess) + " at one point."
}
