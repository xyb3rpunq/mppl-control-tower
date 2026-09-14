package simulate

import (
	"fmt"
	"math"
	"sort"

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
//	L2 risiko      - kejadian pada risk register terjadi dengan peluang residualnya
//	L3 kapasitas   - jadwal disusun ulang menghormati kapasitas dan periode ujian
//
// Biaya dihitung dari isi pekerjaan: tarif x alokasi x durasi sampel, ditambah
// biaya non-tenaga-kerja, ditambah dampak biaya risiko yang terjadi. Hari
// tambahan akibat risiko menambah durasi tetapi SENGAJA tidak menambah biaya
// tenaga kerja, karena nilai dampak rupiah pada register sudah mencakupnya.

// Layer adalah tingkat realisme simulasi.
type Layer int

// Lapisan simulasi terpadu, kumulatif.
const (
	LayerIndependent Layer = iota
	LayerCorrelated
	LayerRisks
	LayerResources
)

// IntegratedConfig mengatur satu simulasi terpadu.
type IntegratedConfig struct {
	Iterations int
	Seed       uint32
	Rho        float64
	Layer      Layer
	Calendar   *workcal.Calendar      // wajib untuk LayerResources
	Capacity   map[model.Role]float64 // wajib untuk LayerResources
	Budget     float64                // pagu untuk menghitung peluang tepat anggaran
	Deadline   float64                // target durasi dalam hari kerja
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

	DurMean, DurP50, DurP80, DurP90     float64
	CostMean, CostP50, CostP80, CostP90 float64

	OnTime   float64 // P(durasi <= tenggat)
	OnBudget float64 // P(biaya <= pagu)
	JCL      float64 // P(keduanya)
	// JointAtP80 adalah peluang bersama pada titik P80 marginal kedua sumbu.
	// Nilainya hampir selalu di bawah 80% - itulah alasan P80 jadwal dan P80
	// biaya tidak boleh sekadar dilaporkan berdampingan.
	JointAtP80 float64

	RiskHits map[string]int // berapa kali tiap risiko terjadi

	// RealisedSameRole adalah rerata korelasi peringkat durasi antar-pasangan
	// aktivitas yang berbagi peran dominan - bukti bahwa rho yang diminta
	// benar-benar muncul di sampel.
	RealisedSameRole float64
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

	sampler := NewSampler(acts)
	rng := NewPRNG(cfg.Seed)
	riskRNG := NewPRNG(cfg.Seed ^ 0x9e3779b9)

	// Aktivitas yang terpapar tiap risiko: aktivitas terpanjang pada paket
	// kerja risiko itu; risiko lintas fase membebani aktivitas terakhir.
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

	var fixed float64
	for _, a := range acts {
		fixed += a.ExtraCost()
	}
	dailyLabour := make([]float64, len(acts))
	for i, a := range acts {
		for _, s := range a.Team {
			dailyLabour[i] += model.RateCard[s.Role] * s.Alloc
		}
	}

	var capGrid map[model.Role][]float64
	if cfg.Layer >= LayerResources {
		if cfg.Calendar == nil || cfg.Capacity == nil {
			return IntegratedResult{}, fmt.Errorf("simulate: lapisan kapasitas butuh kalender dan kapasitas")
		}
		capGrid = level.CapacityGrid(cfg.Calendar, cfg.Capacity, true, levelHorizon)
	}

	res := IntegratedResult{Config: cfg, RiskHits: map[string]int{}}
	res.pairs = make([][2]float64, 0, cfg.Iterations)
	drawn := make([]int, len(acts))
	withRisk := make([]int, len(acts))

	// Pencatat korelasi: simpan durasi sampel untuk pasangan peran sama.
	logDur := make([][]float64, len(acts))
	for i := range logDur {
		logDur[i] = make([]float64, 0, cfg.Iterations)
	}

	for it := 0; it < cfg.Iterations; it++ {
		sampler.Draw(rng, rho, "pert", drawn)
		cost := fixed
		for i := range acts {
			cost += dailyLabour[i] * float64(drawn[i])
			logDur[i] = append(logDur[i], float64(drawn[i]))
		}
		copy(withRisk, drawn)

		if cfg.Layer >= LayerRisks {
			for _, r := range model.Risks {
				if riskRNG.Float64() >= r.ResidualProb {
					continue
				}
				res.RiskHits[r.ID]++
				cost += r.ResidualImpact
				if !modelledByWindow[r.ID] {
					withRisk[exposed[r.ID]] += r.ScheduleImpact
				}
			}
		}

		durOf := func(a model.Activity) int { return withRisk[sampler.Index(a.ID)] }
		var dur int
		if cfg.Layer >= LayerResources {
			lv, err := level.Run(acts, level.Options{
				Calendar: cfg.Calendar, Capacity: cfg.Capacity, UseWindows: true,
				DurationOf: durOf, Horizon: levelHorizon, Lite: true, CapGrid: capGrid,
			})
			if err != nil {
				return IntegratedResult{}, err
			}
			dur = lv.Duration
		} else {
			cpm, err := schedule.Compute(acts, schedule.Options{DurationOf: durOf})
			if err != nil {
				return IntegratedResult{}, err
			}
			dur = cpm.Duration
		}
		res.pairs = append(res.pairs, [2]float64{float64(dur), cost})
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
	res.CostMean, res.CostP50 = Mean(res.Costs), Quantile(res.Costs, 0.5)
	res.CostP80, res.CostP90 = Quantile(res.Costs, 0.8), Quantile(res.Costs, 0.9)

	res.OnTime = res.ProbDuration(cfg.Deadline)
	res.OnBudget = res.ProbCost(cfg.Budget)
	res.JCL = res.Joint(cfg.Deadline, cfg.Budget)
	res.JointAtP80 = res.Joint(res.DurP80, res.CostP80)
	res.RealisedSameRole = realisedSameRole(sampler, acts, logDur)
	return res, nil
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

// Ladder menjalankan keempat lapisan dengan benih yang sama, sehingga selisih
// antar-lapisan murni akibat efek yang ditambahkan, bukan akibat derau sampel.
func Ladder(acts []model.Activity, base IntegratedConfig) ([]IntegratedResult, error) {
	out := make([]IntegratedResult, 0, 4)
	for _, l := range []Layer{LayerIndependent, LayerCorrelated, LayerRisks, LayerResources} {
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
