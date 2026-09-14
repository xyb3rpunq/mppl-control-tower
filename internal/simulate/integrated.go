package simulate

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/cost"
	"github.com/xyb3rpunq/mppl-control-tower/internal/gert"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Simulasi terpadu menjawab pertanyaan yang tidak bisa dijawab halaman PERT:
// berapa peluang proyek ini selesai TEPAT WAKTU DAN TEPAT ANGGARAN sekaligus,
// setelah semua sumber ketidakpastian yang diketahui ikut dihitung?
//
// Ketidakpastian ditambahkan lapis demi lapis, dan setiap lapis adalah efek
// nyata yang terpisah:
//
//	L0 independen  - durasi aktivitas acak, saling bebas (sama dengan halaman PERT)
//	L1 korelasi    - aktivitas yang dikerjakan orang yang sama bergerak bersama
//	L2 risiko      - kejadian risk register, bergerombol lewat penggerak bersama
//	L3 rework      - pemeriksaan bisa gagal dan diulang (putaran GERT)
//	L4 kapasitas   - jadwal disusun ulang menghormati kapasitas dan periode ujian
//
// Biaya dihitung dari isi pekerjaan: tarif x alokasi x durasi sampel, ditambah
// biaya non-tenaga-kerja, ditambah dampak biaya risiko yang terjadi, ditambah
// biaya rework. Biaya sewa dan langganan (model.Extra.TimeBased) mengikuti
// panjang proyek: rentang dari aktivitas pembelinya sampai proyek selesai.
// Hari tambahan akibat risiko menambah durasi tetapi SENGAJA tidak menambah
// biaya tenaga kerja, karena nilai dampak rupiah pada register sudah
// mencakupnya.

// Layer adalah tingkat realisme simulasi.
type Layer int

// Lapisan simulasi terpadu, kumulatif.
const (
	LayerIndependent Layer = iota
	LayerCorrelated
	LayerRisks
	LayerRework
	LayerResources
)

// Layers adalah seluruh lapisan dalam urutan tangga realisme.
var Layers = []Layer{LayerIndependent, LayerCorrelated, LayerRisks, LayerRework, LayerResources}

// IntegratedConfig mengatur satu simulasi terpadu.
type IntegratedConfig struct {
	Iterations int
	Seed       uint32
	Rho        float64
	// RiskLoading adalah bobot penggerak bersama pada kopula risiko (lambda).
	// Nol berarti kejadian risiko saling bebas.
	RiskLoading float64
	Layer       Layer
	Calendar    *workcal.Calendar      // wajib untuk LayerResources
	Capacity    map[model.Role]float64 // wajib untuk LayerResources
	Budget      float64                // pagu untuk menghitung peluang tepat anggaran
	Deadline    float64                // target durasi dalam hari kerja
	// ExamFactor mengganti faktor kapasitas jendela ujian bila > 0.
	ExamFactor float64
	// LevelOrder adalah daftar aktivitas jadwal levelling terbaik. Bila diisi,
	// setiap iterasi lapisan kapasitas menjalankan SGS dengan daftar ini dan
	// dengan aturan LST, lalu memakai yang lebih pendek.
	LevelOrder []string
	// InFlight, bila tidak nil, menjalankan simulasi dari tanggal data:
	// realisasi dikunci, sisa pekerjaan diambil sampelnya.
	InFlight *InFlight
	// AuditEvery, bila > 0 pada lapisan kapasitas, menjalankan level.Optimize
	// dan level.LowerBound pada setiap iterasi ke-AuditEvery dengan durasi yang
	// sama persis, untuk mengukur seberapa jauh SGS per iterasi dari optimum.
	// Audit tidak mengubah hasil simulasi.
	AuditEvery   int
	AuditSamples int
}

// LevelAudit mengukur mutu levelling cepat yang dipakai di dalam simulasi.
type LevelAudit struct {
	Audited int
	// SGSGap adalah durasi SGS per iterasi dikurangi jadwal terbaik Optimize.
	SGSGapMean float64
	SGSGapMax  int
	// BoundGap adalah jadwal terbaik dikurangi batas bawah.
	BoundGapMean float64
	BoundGapMax  int
	// ProvenShare adalah porsi iterasi teraudit yang jadwal terbaiknya terbukti
	// optimal; SGSOptimalShare porsi yang SGS per iterasinya sudah optimal.
	ProvenShare     float64
	SGSOptimalShare float64
}

// DefaultRho adalah korelasi baku antar-aktivitas yang dikerjakan peran sama.
// Nilainya ASUMSI: literatur analisis risiko jadwal lazim memakai 0,3 sampai
// 0,6 untuk pekerjaan oleh tim yang sama. Halaman simulasi menunjukkan seberapa
// peka kesimpulannya terhadap angka ini.
const DefaultRho = 0.5

// levelHorizon adalah batas hari kerja untuk levelling di dalam simulasi.
const levelHorizon = 260

// IntegratedResult adalah keluaran simulasi terpadu.
type IntegratedResult struct {
	Config IntegratedConfig

	Durations []float64 // terurut
	Costs     []float64 // terurut
	pairs     [][2]float64

	DurMean, DurP50, DurP80, DurP90, DurP95      float64
	CostMean, CostP50, CostP80, CostP90, CostP95 float64

	OnTime   float64 // P(durasi <= tenggat)
	OnBudget float64 // P(biaya <= pagu)
	JCL      float64 // P(keduanya)
	// JointAtP80 adalah peluang bersama pada titik P80 marginal kedua sumbu.
	// Nilainya hampir selalu di bawah 80% - itulah alasan P80 jadwal dan P80
	// biaya tidak boleh sekadar dilaporkan berdampingan.
	JointAtP80 float64

	RiskHits map[string]int // berapa kali tiap risiko terjadi
	// RiskCount[k] adalah porsi iterasi dengan tepat k risiko terjadi.
	RiskCount []float64
	// RiskPhi adalah rerata koefisien phi kejadian antar-pasangan risiko yang
	// berbagi penggerak - bukti bahwa kopula benar-benar membuat risiko
	// bergerombol di sampel.
	RiskPhi float64

	// ReworkCycles adalah rerata putaran tambahan per putaran GERT.
	ReworkCycles map[string]float64
	ReworkDays   float64 // rerata hari tambahan akibat rework
	ReworkCost   float64 // rerata biaya rework
	TimeCost     float64 // rerata biaya sewa & langganan
	TimeCostPlan float64 // biaya sewa & langganan pada jadwal rencana

	// RealisedSameRole adalah rerata korelasi peringkat durasi antar-pasangan
	// aktivitas yang berbagi peran dominan - bukti bahwa rho yang diminta
	// benar-benar muncul di sampel.
	RealisedSameRole float64

	Audit LevelAudit
}

// exposure memetakan risiko ke aktivitas yang menanggung hari tambahannya:
// aktivitas terpanjang pada paket kerja risiko itu; risiko lintas fase
// membebani aktivitas non-milestone terakhir.
func exposure(acts []model.Activity) map[string]int {
	exposed := map[string]int{}
	lastIdx := len(acts) - 1
	for lastIdx > 0 && acts[lastIdx].Milestone {
		lastIdx--
	}
	for _, r := range model.Risks {
		best := -1
		for i, a := range acts {
			if a.Milestone || a.WBS != r.WBS {
				continue
			}
			if best < 0 || a.Duration > acts[best].Duration {
				best = i
			}
		}
		if best < 0 {
			best = lastIdx
		}
		exposed[r.ID] = best
	}
	return exposed
}

// RunIntegrated menjalankan simulasi terpadu sampai lapisan tertentu.
func RunIntegrated(acts []model.Activity, cfg IntegratedConfig) (IntegratedResult, error) {
	if cfg.Iterations <= 0 {
		cfg.Iterations = 10_000
	}
	if cfg.Seed == 0 {
		cfg.Seed = Defaults().Seed
	}
	rho := cfg.Rho
	if cfg.Layer < LayerCorrelated {
		rho = 0
	}
	loading := cfg.RiskLoading
	if loading < 0 {
		loading = 0
	}
	if loading > 0.999 {
		loading = 0.999
	}
	fl := cfg.InFlight
	if fl != nil && len(fl.State) != len(acts) {
		return IntegratedResult{}, fmt.Errorf("simulate: status tanggal data disiapkan untuk jaringan lain")
	}

	sampler := NewSampler(acts)
	rng := NewPRNG(cfg.Seed)
	riskRNG := NewPRNG(cfg.Seed ^ 0x9e3779b9)
	reworkRNG := NewPRNG(cfg.Seed ^ 0x85ebca6b)

	exposed := exposure(acts)
	// Risiko yang sudah dimodelkan sebagai jendela ketersediaan tidak boleh
	// menambah hari lagi pada lapisan kapasitas.
	modelledByWindow := map[string]bool{}
	if cfg.Layer >= LayerResources {
		for _, w := range model.AvailabilityWindows {
			if w.RiskID != "" {
				modelledByWindow[w.RiskID] = true
			}
		}
	}

	tcs, fixed, err := cost.Rentals(acts)
	if err != nil {
		return IntegratedResult{}, err
	}
	dailyLabour := make([]float64, len(acts))
	for i, a := range acts {
		for _, s := range a.Team {
			dailyLabour[i] += model.RateCard[s.Role] * s.Alloc
		}
	}
	index := make(map[string]int, len(acts))
	for i, a := range acts {
		index[a.ID] = i
	}

	type loopState struct {
		loop  model.ReworkLoop
		check int
		parts []int
	}
	var loops []loopState
	for _, l := range model.ReworkLoops {
		c, ok := index[l.Check]
		if !ok {
			continue
		}
		ls := loopState{loop: l, check: c}
		for _, p := range l.Parts {
			ls.parts = append(ls.parts, index[p.Activity])
		}
		loops = append(loops, ls)
	}

	// Penggerak risiko yang tidak ditautkan ke peran punya faktor laten
	// sendiri. Faktor ditarik setiap iterasi walau lambda nol, supaya aliran
	// bilangan acak sama untuk semua nilai lambda (common random numbers).
	driverIdx := map[string]int{}
	var freeDrivers []string
	for _, d := range model.RiskDrivers {
		if d.Role == "" {
			driverIdx[d.Key] = len(freeDrivers)
			freeDrivers = append(freeDrivers, d.Key)
		}
	}
	driverVal := make([]float64, len(freeDrivers))

	netActs := acts
	var release []int
	if fl != nil {
		netActs = fl.Residual
		release = fl.Release
		exposed = fl.Exposed
	}

	var capGrid map[model.Role][]float64
	if cfg.Layer >= LayerResources {
		if cfg.Calendar == nil || cfg.Capacity == nil {
			return IntegratedResult{}, fmt.Errorf("simulate: lapisan kapasitas butuh kalender dan kapasitas")
		}
		capGrid = level.CapacityGridWith(cfg.Calendar, cfg.Capacity, true, levelHorizon, cfg.ExamFactor)
	}

	res := IntegratedResult{Config: cfg, RiskHits: map[string]int{}, ReworkCycles: map[string]float64{}}
	res.pairs = make([][2]float64, 0, cfg.Iterations)
	res.RiskCount = make([]float64, len(model.Risks)+1)
	for _, t := range tcs {
		res.TimeCostPlan += t.Amount
	}
	drawn := make([]int, len(acts))
	withRisk := make([]int, len(acts))

	logDur := make([][]float64, len(acts))
	for i := range logDur {
		logDur[i] = make([]float64, 0, cfg.Iterations)
	}
	fired := make([][]bool, len(model.Risks))
	for k := range fired {
		fired[k] = make([]bool, 0, cfg.Iterations)
	}

	durOf := func(a model.Activity) int { return withRisk[sampler.Index(a.ID)] }
	var releaseOf func(model.Activity) int
	if release != nil {
		releaseOf = func(a model.Activity) int { return release[sampler.Index(a.ID)] }
	}

	for it := 0; it < cfg.Iterations; it++ {
		sampler.Draw(rng, rho, "pert", drawn)
		total := fixed
		if fl != nil {
			total = fl.fixedCost
		}
		for i := range acts {
			if fl != nil {
				drawn[i] = fl.duration(sampler, i, drawn[i])
				total += dailyLabour[i] * float64(drawn[i]) * fl.laborFactor(i)
			} else {
				total += dailyLabour[i] * float64(drawn[i])
			}
			logDur[i] = append(logDur[i], float64(drawn[i]))
		}
		copy(withRisk, drawn)

		if cfg.Layer >= LayerRisks {
			for d := range driverVal {
				driverVal[d] = NormalSample(riskRNG)
			}
			idio := math.Sqrt(1 - loading*loading)
			hits := 0
			for k, r := range model.Risks {
				eps := NormalSample(riskRNG)
				y := eps
				if key := r.DriverOf(); key != "" {
					var g float64
					if d, ok := model.DriverByKey(key); ok && d.Role != "" {
						g = sampler.Factor(d.Role)
					} else {
						g = driverVal[driverIdx[key]]
					}
					y = loading*g + idio*eps
				}
				occurs := stdNormalCDF(y) > 1-r.ResidualProb
				if fl != nil && fl.ClosedRisk[r.ID] {
					occurs = false
				}
				fired[k] = append(fired[k], occurs)
				if !occurs {
					continue
				}
				hits++
				res.RiskHits[r.ID]++
				total += r.ResidualImpact
				if !modelledByWindow[r.ID] {
					withRisk[exposed[r.ID]] += r.ScheduleImpact
				}
			}
			res.RiskCount[hits]++
		}

		if cfg.Layer >= LayerRework {
			for _, ls := range loops {
				u := reworkRNG.Float64()
				if fl != nil && fl.ClosedLoop[ls.loop.ID] {
					continue
				}
				n := gert.SampleCycles(u, ls.loop.FailProb)
				if n == 0 {
					continue
				}
				res.ReworkCycles[ls.loop.ID] += float64(n)
				var days, rc float64
				for pi, part := range ls.parts {
					d := ls.loop.Parts[pi].Fraction * float64(drawn[part])
					days += d
					rc += d * dailyLabour[part]
				}
				extra := int(math.Round(float64(n) * days))
				withRisk[ls.check] += extra
				res.ReworkDays += float64(extra)
				res.ReworkCost += float64(n) * rc
				total += float64(n) * rc
			}
		}

		var dur int
		starts := func(i int) int { return 0 }
		if cfg.Layer >= LayerResources {
			base := level.Options{
				Calendar: cfg.Calendar, Capacity: cfg.Capacity, UseWindows: true, ExamFactor: cfg.ExamFactor,
				DurationOf: durOf, ReleaseOf: releaseOf, Horizon: levelHorizon, Lite: true, CapGrid: capGrid,
			}
			lv, err := level.Run(netActs, base)
			if err != nil {
				return IntegratedResult{}, err
			}
			if len(cfg.LevelOrder) > 0 {
				alt := base
				alt.Order = cfg.LevelOrder
				lo, err := level.Run(netActs, alt)
				if err != nil {
					return IntegratedResult{}, err
				}
				if lo.Duration < lv.Duration {
					lv = lo
				}
			}
			dur = lv.Duration
			starts = func(i int) int { return lv.Tasks[acts[i].ID].Start }
			if cfg.AuditEvery > 0 && it%cfg.AuditEvery == 0 {
				opt, err := level.Optimize(netActs, level.OptimizeOptions{Options: base, Samples: cfg.AuditSamples, Seed: cfg.Seed + uint32(it)})
				if err != nil {
					return IntegratedResult{}, err
				}
				a := &res.Audit
				a.Audited++
				sg := dur - opt.Best.Duration
				if sg < 0 {
					sg = 0
				}
				a.SGSGapMean += float64(sg)
				if sg > a.SGSGapMax {
					a.SGSGapMax = sg
				}
				a.BoundGapMean += float64(opt.Gap)
				if opt.Gap > a.BoundGapMax {
					a.BoundGapMax = opt.Gap
				}
				if opt.Proven {
					a.ProvenShare++
				}
				if dur == opt.Bound.Value {
					a.SGSOptimalShare++
				}
			}
		} else {
			cpm, err := schedule.Compute(netActs, schedule.Options{DurationOf: durOf, ReleaseOf: releaseOf})
			if err != nil {
				return IntegratedResult{}, err
			}
			dur = cpm.ProjectFinish
			starts = func(i int) int { return cpm.Tasks[acts[i].ID].StartX }
		}

		for _, t := range tcs {
			var c float64
			if fl != nil && fl.State[t.Index].Kind != NotStarted {
				// Nilai rencana sudah ada di biaya aktual; tambahkan selisih rentang.
				c = t.Cost(fl.State[t.Index].Start, dur) - t.Amount
			} else {
				c = t.Cost(starts(t.Index), dur)
			}
			total += c
			res.TimeCost += c
		}
		res.pairs = append(res.pairs, [2]float64{float64(dur), total})
	}

	if a := &res.Audit; a.Audited > 0 {
		k := float64(a.Audited)
		a.SGSGapMean /= k
		a.BoundGapMean /= k
		a.ProvenShare /= k
		a.SGSOptimalShare /= k
	}
	n := float64(cfg.Iterations)
	res.ReworkDays /= n
	res.ReworkCost /= n
	res.TimeCost /= n
	if fl != nil {
		res.TimeCost += fl.timeCostPaid
	}
	for id := range res.ReworkCycles {
		res.ReworkCycles[id] /= n
	}
	for k := range res.RiskCount {
		res.RiskCount[k] /= n
	}

	res.Durations = make([]float64, len(res.pairs))
	res.Costs = make([]float64, len(res.pairs))
	for i, p := range res.pairs {
		res.Durations[i] = p[0]
		res.Costs[i] = p[1]
	}
	sort.Float64s(res.Durations)
	sort.Float64s(res.Costs)

	res.DurMean, res.DurP50 = Mean(res.Durations), Quantile(res.Durations, 0.5)
	res.DurP80, res.DurP90 = Quantile(res.Durations, 0.8), Quantile(res.Durations, 0.9)
	res.DurP95 = Quantile(res.Durations, 0.95)
	res.CostMean, res.CostP50 = Mean(res.Costs), Quantile(res.Costs, 0.5)
	res.CostP80, res.CostP90 = Quantile(res.Costs, 0.8), Quantile(res.Costs, 0.9)
	res.CostP95 = Quantile(res.Costs, 0.95)

	res.OnTime = res.ProbDuration(cfg.Deadline)
	res.OnBudget = res.ProbCost(cfg.Budget)
	res.JCL = res.Joint(cfg.Deadline, cfg.Budget)
	res.JointAtP80 = res.Joint(res.DurP80, res.CostP80)
	res.RealisedSameRole = realisedSameRole(sampler, acts, logDur)
	if cfg.Layer >= LayerRisks {
		res.RiskPhi = realisedRiskPhi(fired)
	}
	return res, nil
}

// realisedRiskPhi menghitung rerata koefisien phi antar-pasangan risiko yang
// berbagi penggerak.
func realisedRiskPhi(fired [][]bool) float64 {
	var sum float64
	n := 0
	for a := 0; a < len(model.Risks); a++ {
		for b := a + 1; b < len(model.Risks); b++ {
			da, db := model.Risks[a].DriverOf(), model.Risks[b].DriverOf()
			if da == "" || da != db {
				continue
			}
			if phi, ok := Phi(fired[a], fired[b]); ok {
				sum += phi
				n++
			}
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// Phi mengembalikan koefisien korelasi phi dua deret kejadian biner.
// Mengembalikan false bila salah satu deret tidak bervariasi.
func Phi(x, y []bool) (float64, bool) {
	var n11, n10, n01, n00 float64
	for i := range x {
		switch {
		case x[i] && y[i]:
			n11++
		case x[i]:
			n10++
		case y[i]:
			n01++
		default:
			n00++
		}
	}
	den := (n11 + n10) * (n01 + n00) * (n11 + n01) * (n10 + n00)
	if den == 0 {
		return 0, false
	}
	return (n11*n00 - n10*n01) / math.Sqrt(den), true
}

// ProbDuration mengembalikan P(durasi <= d).
func (r IntegratedResult) ProbDuration(d float64) float64 {
	if len(r.Durations) == 0 {
		return 0
	}
	return float64(sort.SearchFloat64s(r.Durations, d+1e-9)) / float64(len(r.Durations))
}

// ProbCost mengembalikan P(biaya <= c).
func (r IntegratedResult) ProbCost(c float64) float64 {
	if len(r.Costs) == 0 {
		return 0
	}
	return float64(sort.SearchFloat64s(r.Costs, c+1e-9)) / float64(len(r.Costs))
}

// Joint mengembalikan Joint Confidence Level: P(durasi <= d DAN biaya <= c).
func (r IntegratedResult) Joint(d, c float64) float64 {
	if len(r.pairs) == 0 {
		return 0
	}
	n := 0
	for _, p := range r.pairs {
		if p[0] <= d+1e-9 && p[1] <= c+1e-9 {
			n++
		}
	}
	return float64(n) / float64(len(r.pairs))
}

// Pairs mengembalikan salinan pasangan (durasi, biaya) per iterasi.
func (r IntegratedResult) Pairs() [][2]float64 {
	return append([][2]float64(nil), r.pairs...)
}

// FrontierPoint adalah satu titik pada kurva JCL: untuk tenggat Duration,
// anggaran minimum yang dibutuhkan agar peluang bersama mencapai target.
type FrontierPoint struct {
	Duration float64
	Budget   float64
	Feasible bool
}

// Frontier menghitung kurva iso-JCL untuk tingkat keyakinan target (mis. 0,7
// seperti yang dipakai NASA). Setiap titik menjawab: "kalau tenggatnya D,
// berapa anggaran minimum agar peluang tepat waktu DAN tepat anggaran >= target?"
func (r IntegratedResult) Frontier(target float64, durations []float64) []FrontierPoint {
	out := make([]FrontierPoint, 0, len(durations))
	need := int(math.Ceil(target * float64(len(r.pairs))))
	for _, d := range durations {
		var costs []float64
		for _, p := range r.pairs {
			if p[0] <= d+1e-9 {
				costs = append(costs, p[1])
			}
		}
		if len(costs) < need || need == 0 {
			out = append(out, FrontierPoint{Duration: d})
			continue
		}
		sort.Float64s(costs)
		out = append(out, FrontierPoint{Duration: d, Budget: costs[need-1], Feasible: true})
	}
	return out
}

// Grid adalah histogram dua dimensi untuk peta kepadatan JCL.
type Grid struct {
	DurMin, DurMax   float64
	CostMin, CostMax float64
	Cols, Rows       int
	Counts           [][]int // [baris][kolom]
	Max              int
}

// Density membangun histogram dua dimensi pasangan durasi-biaya.
func (r IntegratedResult) Density(cols, rows int) Grid {
	g := Grid{Cols: cols, Rows: rows, Counts: make([][]int, rows)}
	for i := range g.Counts {
		g.Counts[i] = make([]int, cols)
	}
	if len(r.pairs) == 0 {
		return g
	}
	g.DurMin, g.DurMax = r.Durations[0], r.Durations[len(r.Durations)-1]
	g.CostMin, g.CostMax = r.Costs[0], r.Costs[len(r.Costs)-1]
	if g.DurMax == g.DurMin {
		g.DurMax++
	}
	if g.CostMax == g.CostMin {
		g.CostMax++
	}
	for _, p := range r.pairs {
		c := int((p[0] - g.DurMin) / (g.DurMax - g.DurMin) * float64(cols))
		ro := int((p[1] - g.CostMin) / (g.CostMax - g.CostMin) * float64(rows))
		if c >= cols {
			c = cols - 1
		}
		if ro >= rows {
			ro = rows - 1
		}
		g.Counts[ro][c]++
		if g.Counts[ro][c] > g.Max {
			g.Max = g.Counts[ro][c]
		}
	}
	return g
}

// realisedSameRole menghitung rerata korelasi Spearman antar-pasangan
// aktivitas yang punya peran dominan sama.
func realisedSameRole(s *Sampler, acts []model.Activity, logDur [][]float64) float64 {
	var sum float64
	n := 0
	for i := range acts {
		if acts[i].Milestone || acts[i].Pessimistic <= acts[i].Optimistic {
			continue
		}
		for j := i + 1; j < len(acts); j++ {
			if acts[j].Milestone || acts[j].Pessimistic <= acts[j].Optimistic {
				continue
			}
			ri, rj := s.DominantRole(i), s.DominantRole(j)
			if ri == "" || ri != rj {
				continue
			}
			sum += Spearman(logDur[i], logDur[j])
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// Ladder menjalankan kelima lapisan dengan benih yang sama, sehingga selisih
// antar-lapisan murni akibat efek yang ditambahkan, bukan akibat derau sampel.
func Ladder(acts []model.Activity, base IntegratedConfig) ([]IntegratedResult, error) {
	out := make([]IntegratedResult, 0, len(Layers))
	for _, l := range Layers {
		c := base
		c.Layer = l
		r, err := RunIntegrated(acts, c)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
