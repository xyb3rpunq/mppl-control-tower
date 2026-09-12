// Package simulate menjalankan simulasi Monte Carlo atas jaringan proyek.
//
// PERT klasik (lihat paket schedule) hanya menghitung varians di sepanjang
// jalur kritis. Kelemahannya nyata: jalur non-kritis yang float-nya tipis dan
// ketidakpastiannya besar bisa berubah menjadi kritis saat kenyataan meleset,
// dan PERT tidak akan pernah melihatnya. Simulasi ini mengambil sampel durasi
// setiap aktivitas lalu menjalankan CPM penuh ribuan kali, sehingga jalur
// kritis boleh berpindah dari satu iterasi ke iterasi berikutnya.
//
// Generator acaknya berbenih (seeded) supaya hasilnya sama persis di setiap
// muat halaman - kalau tidak, angka di dasbor akan bergoyang sendiri dan
// mustahil diverifikasi pembaca.
package simulate

import (
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// Config mengatur satu kali simulasi.
type Config struct {
	Iterations   int
	Seed         uint32
	Distribution string // "pert" (bawaan) atau "triangular"
}

// Defaults mengembalikan konfigurasi baku: 10.000 iterasi, benih tetap.
func Defaults() Config {
	return Config{Iterations: 10_000, Seed: 20210801, Distribution: "pert"}
}

// Sensitivity adalah pengaruh satu aktivitas terhadap durasi total proyek.
type Sensitivity struct {
	ID           string
	Name         model.Text
	Correlation  float64 // korelasi peringkat Spearman durasi aktivitas vs durasi proyek
	CriticalRate float64 // porsi iterasi di mana aktivitas ini berada di jalur kritis
}

// Result adalah keluaran simulasi.
type Result struct {
	Iterations    int
	Durations     []float64 // terurut menaik
	Mean          float64
	StdDev        float64
	Min           float64
	Max           float64
	P10           float64
	P50           float64
	P80           float64
	P90           float64
	P95           float64
	Deterministic int     // durasi CPM memakai durasi paling mungkin
	OnTimeProb    float64 // peluang selesai pada atau sebelum durasi rencana
	Histogram     []Bin
	Sensitivity   []Sensitivity
}

// Bin adalah satu batang histogram.
type Bin struct {
	From  float64
	To    float64
	Count int
	// Cumulative adalah porsi iterasi yang berakhir pada atau sebelum To -
	// dipakai menggambar kurva-S probabilitas di atas histogram.
	Cumulative float64
}

// Run menjalankan simulasi Monte Carlo.
func Run(activities []model.Activity, plan schedule.Result, cfg Config) (Result, error) {
	if cfg.Iterations <= 0 {
		cfg = Defaults()
	}
	rng := NewPRNG(cfg.Seed)

	// Hanya aktivitas dengan ketidakpastian sungguhan yang perlu dilacak
	// sensitivitasnya; milestone dan aktivitas berdurasi tetap tidak.
	var tracked []string
	for _, a := range activities {
		if !a.Milestone && a.Pessimistic > a.Optimistic {
			tracked = append(tracked, a.ID)
		}
	}
	sampleLog := make(map[string][]float64, len(tracked))
	for _, id := range tracked {
		sampleLog[id] = make([]float64, 0, cfg.Iterations)
	}
	criticalCount := make(map[string]int, len(tracked))

	durations := make([]float64, 0, cfg.Iterations)
	sample := make(map[string]int, len(activities))

	for i := 0; i < cfg.Iterations; i++ {
		for _, a := range activities {
			if a.Milestone {
				sample[a.ID] = 0
				continue
			}
			o, m, p := float64(a.Optimistic), float64(a.Duration), float64(a.Pessimistic)
			if p <= o {
				sample[a.ID] = a.Duration
				continue
			}
			var v float64
			if cfg.Distribution == "triangular" {
				v = TriangularSample(rng, o, m, p)
			} else {
				v = BetaPERTSample(rng, o, m, p, 4)
			}
			d := int(math.Round(v))
			if d < 1 {
				d = 1
			}
			sample[a.ID] = d
		}

		res, err := schedule.Compute(activities, schedule.Options{
			DurationOf: func(a model.Activity) int { return sample[a.ID] },
		})
		if err != nil {
			return Result{}, err
		}
		durations = append(durations, float64(res.Duration))
		for _, id := range tracked {
			sampleLog[id] = append(sampleLog[id], float64(sample[id]))
			if res.Tasks[id].Critical {
				criticalCount[id]++
			}
		}
	}

	sorted := append([]float64(nil), durations...)
	sort.Float64s(sorted)

	r := Result{
		Iterations:    cfg.Iterations,
		Durations:     sorted,
		Mean:          Mean(sorted),
		StdDev:        StdDev(sorted),
		Min:           sorted[0],
		Max:           sorted[len(sorted)-1],
		P10:           Quantile(sorted, 0.10),
		P50:           Quantile(sorted, 0.50),
		P80:           Quantile(sorted, 0.80),
		P90:           Quantile(sorted, 0.90),
		P95:           Quantile(sorted, 0.95),
		Deterministic: plan.Duration,
	}

	target := float64(plan.Duration)
	onTime := 0
	for _, d := range sorted {
		if d <= target {
			onTime++
		}
	}
	r.OnTimeProb = float64(onTime) / float64(len(sorted))
	r.Histogram = buildHistogram(sorted, 28)

	byID := model.ActivityByID()
	sens := make([]Sensitivity, 0, len(tracked))
	for _, id := range tracked {
		sens = append(sens, Sensitivity{
			ID:           id,
			Name:         byID[id].Name,
			Correlation:  Spearman(sampleLog[id], durations),
			CriticalRate: float64(criticalCount[id]) / float64(cfg.Iterations),
		})
	}
	sort.Slice(sens, func(i, j int) bool {
		return math.Abs(sens[i].Correlation) > math.Abs(sens[j].Correlation)
	})
	r.Sensitivity = sens
	return r, nil
}

func buildHistogram(sorted []float64, bins int) []Bin {
	if len(sorted) == 0 {
		return nil
	}
	min, max := sorted[0], sorted[len(sorted)-1]
	if max == min {
		return []Bin{{From: min, To: max, Count: len(sorted), Cumulative: 1}}
	}
	width := (max - min) / float64(bins)
	counts := make([]int, bins)
	for _, v := range sorted {
		i := int((v - min) / width)
		if i >= bins {
			i = bins - 1
		}
		counts[i]++
	}
	out := make([]Bin, bins)
	running := 0
	for i, c := range counts {
		running += c
		out[i] = Bin{
			From:       min + float64(i)*width,
			To:         min + float64(i+1)*width,
			Count:      c,
			Cumulative: float64(running) / float64(len(sorted)),
		}
	}
	return out
}

// ProbabilityBy mengembalikan porsi iterasi yang selesai dalam target hari.
func (r Result) ProbabilityBy(target float64) float64 {
	if len(r.Durations) == 0 {
		return 0
	}
	i := sort.SearchFloat64s(r.Durations, target+1e-9)
	return float64(i) / float64(len(r.Durations))
}

// DaysForConfidence mengembalikan durasi yang dibutuhkan untuk mencapai
// tingkat keyakinan tertentu, mis. 0,8 untuk komitmen P80.
func (r Result) DaysForConfidence(p float64) float64 {
	return Quantile(r.Durations, p)
}
