package simulate_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

func plan(t *testing.T) schedule.Result {
	t.Helper()
	r, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	return r
}

// TestSimulationIsDeterministic adalah uji terpenting di paket ini. Tanpa
// determinisme, angka Monte Carlo di dasbor akan berubah setiap kali situs
// dibangun dan tidak ada pembaca yang bisa memverifikasinya.
func TestSimulationIsDeterministic(t *testing.T) {
	p := plan(t)
	cfg := simulate.Config{Iterations: 2000, Seed: 42, Distribution: "pert"}

	a, err := simulate.Run(model.Activities, p, cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	b, err := simulate.Run(model.Activities, p, cfg)
	if err != nil {
		t.Fatalf("Run ulang: %v", err)
	}

	if a.Mean != b.Mean || a.P80 != b.P80 || a.OnTimeProb != b.OnTimeProb {
		t.Errorf("benih sama menghasilkan hasil berbeda: mean %v/%v, P80 %v/%v, onTime %v/%v",
			a.Mean, b.Mean, a.P80, b.P80, a.OnTimeProb, b.OnTimeProb)
	}
	for i := range a.Durations {
		if a.Durations[i] != b.Durations[i] {
			t.Fatalf("iterasi ke-%d berbeda antara dua jalan dengan benih sama", i)
		}
	}
}

func TestDifferentSeedsGiveDifferentSamplesButSimilarConclusion(t *testing.T) {
	p := plan(t)
	a, _ := simulate.Run(model.Activities, p, simulate.Config{Iterations: 5000, Seed: 1, Distribution: "pert"})
	b, _ := simulate.Run(model.Activities, p, simulate.Config{Iterations: 5000, Seed: 999, Distribution: "pert"})

	if a.Mean == b.Mean {
		t.Error("dua benih berbeda menghasilkan rerata identik; generator acaknya mencurigakan")
	}
	// Kesimpulannya harus stabil walau sampelnya berbeda - inilah tanda
	// hasilnya tidak bergantung pada pilihan benih.
	if math.Abs(a.Mean-b.Mean) > 1.0 {
		t.Errorf("rerata bergeser terlalu jauh antar benih: %.2f vs %.2f", a.Mean, b.Mean)
	}
	if math.Abs(a.P80-b.P80) > 2 {
		t.Errorf("P80 bergeser terlalu jauh antar benih: %v vs %v", a.P80, b.P80)
	}
}

func TestQuantilesAreOrdered(t *testing.T) {
	r, err := simulate.Run(model.Activities, plan(t), simulate.Config{Iterations: 3000, Seed: 7})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !(r.Min <= r.P10 && r.P10 <= r.P50 && r.P50 <= r.P80 && r.P80 <= r.P90 && r.P90 <= r.P95 && r.P95 <= r.Max) {
		t.Errorf("kuantil tidak terurut: min %v P10 %v P50 %v P80 %v P90 %v P95 %v max %v",
			r.Min, r.P10, r.P50, r.P80, r.P90, r.P95, r.Max)
	}
}

// TestMergeBiasPushesMeanAboveDeterministic menjaga argumen utama halaman
// PERT: rerata simulasi HARUS lebih besar daripada jadwal deterministik pada
// jaringan yang punya titik pertemuan jalur. Kalau invarian ini pecah, seluruh
// penjelasan merge bias di situs itu ikut salah.
func TestMergeBiasPushesMeanAboveDeterministic(t *testing.T) {
	p := plan(t)
	r, err := simulate.Run(model.Activities, p, simulate.Config{Iterations: 5000, Seed: 3})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.Mean <= float64(p.Duration) {
		t.Errorf("rerata simulasi %.2f tidak melebihi durasi deterministik %d; "+
			"merge bias dan kecondongan estimasi seharusnya mendorongnya naik", r.Mean, p.Duration)
	}
}

func TestProbabilityByIsMonotonic(t *testing.T) {
	r, _ := simulate.Run(model.Activities, plan(t), simulate.Config{Iterations: 3000, Seed: 11})
	prev := -1.0
	for d := r.Min - 5; d <= r.Max+5; d++ {
		p := r.ProbabilityBy(d)
		if p < 0 || p > 1 {
			t.Fatalf("P(<= %v) = %v di luar [0,1]", d, p)
		}
		if p < prev {
			t.Fatalf("peluang turun saat target naik pada %v", d)
		}
		prev = p
	}
	if r.ProbabilityBy(r.Max) < 0.999 {
		t.Error("peluang pada durasi maksimum seharusnya mendekati 1")
	}
}

func TestHistogramCountsSumToIterations(t *testing.T) {
	r, _ := simulate.Run(model.Activities, plan(t), simulate.Config{Iterations: 4000, Seed: 5})
	total := 0
	for _, b := range r.Histogram {
		total += b.Count
	}
	if total != r.Iterations {
		t.Errorf("jumlah isi bin = %d, mau %d", total, r.Iterations)
	}
	if last := r.Histogram[len(r.Histogram)-1].Cumulative; math.Abs(last-1) > 1e-9 {
		t.Errorf("kumulatif bin terakhir = %v, mau 1", last)
	}
}

func TestSensitivityIsSortedAndBounded(t *testing.T) {
	r, _ := simulate.Run(model.Activities, plan(t), simulate.Config{Iterations: 3000, Seed: 13})
	if len(r.Sensitivity) == 0 {
		t.Fatal("tidak ada aktivitas yang dilacak sensitivitasnya")
	}
	for i, s := range r.Sensitivity {
		if s.Correlation < -1 || s.Correlation > 1 {
			t.Errorf("%s: korelasi %v di luar [-1,1]", s.ID, s.Correlation)
		}
		if s.CriticalRate < 0 || s.CriticalRate > 1 {
			t.Errorf("%s: porsi kritis %v di luar [0,1]", s.ID, s.CriticalRate)
		}
		if i > 0 && math.Abs(s.Correlation) > math.Abs(r.Sensitivity[i-1].Correlation)+1e-9 {
			t.Errorf("sensitivitas tidak terurut menurun di posisi %d", i)
		}
	}
	// Aktivitas yang selalu kritis harus punya korelasi positif yang jelas.
	for _, s := range r.Sensitivity {
		if s.CriticalRate > 0.99 && s.Correlation <= 0 {
			t.Errorf("%s selalu kritis tetapi korelasinya %v", s.ID, s.Correlation)
		}
	}
}

func TestBothDistributionsAgreeOnConclusion(t *testing.T) {
	p := plan(t)
	pert, _ := simulate.Run(model.Activities, p, simulate.Config{Iterations: 4000, Seed: 21, Distribution: "pert"})
	tri, _ := simulate.Run(model.Activities, p, simulate.Config{Iterations: 4000, Seed: 21, Distribution: "triangular"})

	// Keduanya harus menolak jadwal 17 minggu. Angkanya boleh berbeda;
	// kesimpulannya tidak boleh.
	if pert.OnTimeProb > 0.2 || tri.OnTimeProb > 0.2 {
		t.Errorf("kedua sebaran seharusnya memberi peluang tepat waktu yang rendah: pert %v, segitiga %v",
			pert.OnTimeProb, tri.OnTimeProb)
	}
}

func TestPRNGIsUniformEnough(t *testing.T) {
	rng := simulate.NewPRNG(2024)
	const n = 200000
	buckets := make([]int, 10)
	for i := 0; i < n; i++ {
		v := rng.Float64()
		if v < 0 || v >= 1 {
			t.Fatalf("nilai acak %v di luar [0,1)", v)
		}
		buckets[int(v*10)]++
	}
	expected := float64(n) / 10
	for i, c := range buckets {
		if math.Abs(float64(c)-expected)/expected > 0.05 {
			t.Errorf("bucket %d berisi %d, jauh dari harapan %.0f", i, c, expected)
		}
	}
}

func TestBetaPERTStaysWithinBounds(t *testing.T) {
	rng := simulate.NewPRNG(99)
	for i := 0; i < 20000; i++ {
		v := simulate.BetaPERTSample(rng, 3, 6, 11, 4)
		if v < 3 || v > 11 {
			t.Fatalf("sampel beta-PERT %v keluar dari rentang [3, 11]", v)
		}
	}
}

func TestTriangularStaysWithinBounds(t *testing.T) {
	rng := simulate.NewPRNG(77)
	for i := 0; i < 20000; i++ {
		v := simulate.TriangularSample(rng, 2, 5, 9)
		if v < 2 || v > 9 {
			t.Fatalf("sampel segitiga %v keluar dari rentang [2, 9]", v)
		}
	}
}

func TestQuantileInterpolation(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5}
	cases := map[float64]float64{0: 1, 0.25: 2, 0.5: 3, 0.75: 4, 1: 5}
	for q, want := range cases {
		if got := simulate.Quantile(sorted, q); math.Abs(got-want) > 1e-9 {
			t.Errorf("Quantile(%v) = %v, mau %v", q, got, want)
		}
	}
}

func TestSpearmanKnownValues(t *testing.T) {
	asc := []float64{1, 2, 3, 4, 5}
	desc := []float64{5, 4, 3, 2, 1}
	if got := simulate.Spearman(asc, asc); math.Abs(got-1) > 1e-9 {
		t.Errorf("korelasi sempurna positif = %v, mau 1", got)
	}
	if got := simulate.Spearman(asc, desc); math.Abs(got+1) > 1e-9 {
		t.Errorf("korelasi sempurna negatif = %v, mau -1", got)
	}
	flat := []float64{2, 2, 2, 2, 2}
	if got := simulate.Spearman(asc, flat); got != 0 {
		t.Errorf("deret konstan seharusnya memberi korelasi 0, dapat %v", got)
	}
}

func TestMeanAndStdDev(t *testing.T) {
	v := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	if got := simulate.Mean(v); math.Abs(got-5) > 1e-9 {
		t.Errorf("Mean = %v, mau 5", got)
	}
	// Simpangan baku sampel (pembagi n-1) untuk deret ini adalah sqrt(32/7).
	want := math.Sqrt(32.0 / 7.0)
	if got := simulate.StdDev(v); math.Abs(got-want) > 1e-9 {
		t.Errorf("StdDev = %v, mau %v", got, want)
	}
}
