package schedule

import (
	"math"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Expected menghitung durasi harapan PERT: te = (O + 4M + P) / 6.
// Lihat Modul 4 MPPL, bagian Program Evaluation and Review Technique.
func Expected(o, m, p float64) float64 { return (o + 4*m + p) / 6 }

// StdDev menghitung simpangan baku PERT: sd = (P - O) / 6.
func StdDev(o, p float64) float64 { return (p - o) / 6 }

// Variance menghitung varians PERT: sd kuadrat.
func Variance(o, p float64) float64 {
	sd := StdDev(o, p)
	return sd * sd
}

// ActivityExpected mengembalikan te untuk satu aktivitas, dengan jatuh balik
// ke durasi paling mungkin bila estimasi tiga titik belum diisi.
func ActivityExpected(a model.Activity) float64 {
	o, p := float64(a.Optimistic), float64(a.Pessimistic)
	m := float64(a.Duration)
	if a.Optimistic == 0 && a.Pessimistic == 0 {
		return m
	}
	return Expected(o, m, p)
}

// PERTStats adalah ringkasan analisis PERT untuk keseluruhan proyek.
type PERTStats struct {
	ExpectedDuration float64  // jumlah te sepanjang jalur kritis PERT
	CPMDuration      int      // durasi CPM memakai te yang dibulatkan
	Variance         float64  // jumlah varians aktivitas di jalur kritis
	StdDev           float64  // akar varians
	CriticalPath     []string // jalur kritis menurut te
}

// ProbabilityBy mengembalikan peluang proyek selesai dalam target hari kerja,
// dengan asumsi Teorema Limit Pusat pada jumlah durasi di jalur kritis.
// Inilah asumsi klasik PERT - dan juga keterbatasannya, karena jalur non-kritis
// yang berisiko tinggi tidak ikut diperhitungkan. Simulasi Monte Carlo di
// paket simulate ada justru untuk menutup celah tersebut.
func (s PERTStats) ProbabilityBy(target float64) float64 {
	if s.StdDev <= 0 {
		if target >= s.ExpectedDuration {
			return 1
		}
		return 0
	}
	z := (target - s.ExpectedDuration) / s.StdDev
	return normalCDF(z)
}

// ZScore mengembalikan nilai Z untuk sebuah target durasi.
func (s PERTStats) ZScore(target float64) float64 {
	if s.StdDev <= 0 {
		return 0
	}
	return (target - s.ExpectedDuration) / s.StdDev
}

// AnalysePERT menghitung statistik PERT untuk sebuah jaringan.
func AnalysePERT(activities []model.Activity, opts Options) (PERTStats, error) {
	opts.DurationOf = func(a model.Activity) int {
		return int(math.Round(ActivityExpected(a)))
	}
	res, err := Compute(activities, opts)
	if err != nil {
		return PERTStats{}, err
	}
	byID := make(map[string]model.Activity, len(activities))
	for _, a := range activities {
		byID[a.ID] = a
	}
	var variance, expected float64
	for _, id := range res.CriticalPath {
		a := byID[id]
		expected += ActivityExpected(a)
		variance += Variance(float64(a.Optimistic), float64(a.Pessimistic))
	}
	return PERTStats{
		ExpectedDuration: expected,
		CPMDuration:      res.Duration,
		Variance:         variance,
		StdDev:           math.Sqrt(variance),
		CriticalPath:     res.CriticalPath,
	}, nil
}

// normalCDF adalah fungsi distribusi kumulatif normal baku, memakai fungsi
// galat bawaan math untuk akurasi penuh.
func normalCDF(z float64) float64 {
	return 0.5 * (1 + math.Erf(z/math.Sqrt2))
}
