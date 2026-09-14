package simulate_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

func TestRegIncBetaKnownValues(t *testing.T) {
	// Beta(1,1) adalah seragam: CDF = x.
	for _, x := range []float64{0.1, 0.37, 0.5, 0.92} {
		if got := simulate.RegIncBeta(1, 1, x); math.Abs(got-x) > 1e-10 {
			t.Errorf("I_%v(1,1) = %v, mau %v", x, got, x)
		}
	}
	// Beta(2,2): CDF = 3x^2 - 2x^3.
	for _, x := range []float64{0.2, 0.5, 0.8} {
		want := 3*x*x - 2*x*x*x
		if got := simulate.RegIncBeta(2, 2, x); math.Abs(got-want) > 1e-10 {
			t.Errorf("I_%v(2,2) = %v, mau %v", x, got, want)
		}
	}
	// Simetri: I_x(a,b) = 1 - I_(1-x)(b,a).
	got := simulate.RegIncBeta(2.5, 4.1, 0.3)
	want := 1 - simulate.RegIncBeta(4.1, 2.5, 0.7)
	if math.Abs(got-want) > 1e-10 {
		t.Errorf("simetri gagal: %v vs %v", got, want)
	}
	if simulate.RegIncBeta(3, 2, 0) != 0 || simulate.RegIncBeta(3, 2, 1) != 1 {
		t.Error("batas CDF harus 0 dan 1")
	}
}

func TestBetaPERTParamsGiveThePERTMean(t *testing.T) {
	o, m, p := 4.0, 6.0, 11.0
	a, b := simulate.BetaPERTParams(o, m, p)
	if a < 1 || b < 1 {
		t.Fatalf("parameter bentuk harus >= 1, dapat %v, %v", a, b)
	}
	mean := o + (p-o)*a/(a+b)
	if want := (o + 4*m + p) / 6; math.Abs(mean-want) > 1e-12 {
		t.Errorf("rerata beta = %v, mau rerata PERT %v", mean, want)
	}
	if a, b := simulate.BetaPERTParams(5, 5, 5); a != 1 || b != 1 {
		t.Errorf("rentang nol harus memberi (1,1), dapat (%v,%v)", a, b)
	}
}

func TestBetaPERTQuantileIsMonotonicAndBounded(t *testing.T) {
	prev := -1.0
	for i := 0; i <= 100; i++ {
		u := float64(i) / 100
		x := simulate.BetaPERTQuantile(4, 6, 11, u)
		if x < 4-1e-9 || x > 11+1e-9 {
			t.Fatalf("kuantil %v = %v keluar dari [4, 11]", u, x)
		}
		if x < prev-1e-9 {
			t.Fatalf("kuantil turun pada u=%v", u)
		}
		prev = x
	}
}

func TestSamplerMeanMatchesPERT(t *testing.T) {
	acts := []model.Activity{{ID: "X", Duration: 6, Optimistic: 4, Pessimistic: 11,
		Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}}}
	s := simulate.NewSampler(acts)
	rng := simulate.NewPRNG(5)
	out := make([]int, 1)
	var sum float64
	const n = 40000
	for i := 0; i < n; i++ {
		s.Draw(rng, 0, "pert", out)
		sum += float64(out[0])
	}
	// Pembulatan ke hari bulat menggeser rerata sedikit; toleransinya 0,1 hari.
	if got, want := sum/n, (4.0+4*6+11)/6; math.Abs(got-want) > 0.1 {
		t.Errorf("rerata sampel = %.3f, mau sekitar %.3f", got, want)
	}
	if s.DominantRole(0) != model.RoleBE {
		t.Errorf("peran dominan = %q, mau BE", s.DominantRole(0))
	}
}

func TestTriangularSamplesStayInRange(t *testing.T) {
	acts := []model.Activity{{ID: "X", Duration: 5, Optimistic: 2, Pessimistic: 9}}
	s := simulate.NewSampler(acts)
	rng := simulate.NewPRNG(77)
	out := make([]int, 1)
	for i := 0; i < 20000; i++ {
		s.Draw(rng, 0, "triangular", out)
		if out[0] < 2 || out[0] > 9 {
			t.Fatalf("sampel segitiga %d keluar dari [2, 9]", out[0])
		}
	}
}

func baseConfig(iter int) simulate.IntegratedConfig {
	return simulate.IntegratedConfig{
		Iterations: iter, Seed: 20210801, Rho: simulate.DefaultRho,
		Calendar: workcal.MustNew(model.ProjectCharter.StartDate, 400),
		Capacity: model.Capacity, Budget: model.TotalAuthorised, Deadline: 85,
	}
}

// TestIndependentLayerMatchesPERTPage adalah jaminan konsistensi utama: lapisan
// pertama simulasi terpadu dan halaman PERT harus menghasilkan angka yang sama
// persis. Kalau berbeda, pembaca melihat dua "P80 independen" yang saling
// bertentangan.
func TestIndependentLayerMatchesPERTPage(t *testing.T) {
	plan := schedule.MustCompute(model.Activities, schedule.Options{})
	pert, err := simulate.Run(model.Activities, plan, simulate.Config{Iterations: 3000, Seed: 20210801, Distribution: "pert"})
	if err != nil {
		t.Fatal(err)
	}
	cfg := baseConfig(3000)
	cfg.Layer = simulate.LayerIndependent
	l0, err := simulate.RunIntegrated(model.Activities, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if pert.P80 != l0.DurP80 || pert.OnTimeProb != l0.OnTime || math.Abs(pert.Mean-l0.DurMean) > 1e-9 {
		t.Errorf("L0 berbeda dari halaman PERT: P80 %v/%v, tepat waktu %v/%v, rerata %v/%v",
			pert.P80, l0.DurP80, pert.OnTimeProb, l0.OnTime, pert.Mean, l0.DurMean)
	}
}

func TestCorrelationIsRealisedInSamples(t *testing.T) {
	low := baseConfig(3000)
	low.Layer, low.Rho = simulate.LayerCorrelated, 0
	high := low
	high.Rho = 0.8
	rl, err := simulate.RunIntegrated(model.Activities, low)
	if err != nil {
		t.Fatal(err)
	}
	rh, err := simulate.RunIntegrated(model.Activities, high)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(rl.RealisedSameRole) > 0.05 {
		t.Errorf("rho 0 menghasilkan korelasi terealisasi %.3f, seharusnya mendekati nol", rl.RealisedSameRole)
	}
	if rh.RealisedSameRole <= rl.RealisedSameRole+0.2 {
		t.Errorf("rho 0,8 tidak menaikkan korelasi terealisasi secara berarti: %.3f vs %.3f", rh.RealisedSameRole, rl.RealisedSameRole)
	}
	if simulate.StdDev(rh.Durations) <= simulate.StdDev(rl.Durations) {
		t.Error("korelasi positif seharusnya melebarkan sebaran durasi proyek")
	}
}

// TestLadderIsCumulative memeriksa bahwa setiap lapisan realisme memindahkan
// P80 ke arah yang benar - tidak ada efek nyata yang membuat proyek lebih cepat.
func TestLadderIsCumulative(t *testing.T) {
	ladder, err := simulate.Ladder(model.Activities, baseConfig(1500))
	if err != nil {
		t.Fatal(err)
	}
	if len(ladder) != 4 {
		t.Fatalf("tangga punya %d lapisan, mau 4", len(ladder))
	}
	if ladder[2].DurP80 <= ladder[1].DurP80 {
		t.Errorf("risiko tidak menaikkan P80 durasi: %v -> %v", ladder[1].DurP80, ladder[2].DurP80)
	}
	if ladder[2].CostP80 <= ladder[1].CostP80 {
		t.Errorf("risiko tidak menaikkan P80 biaya: %v -> %v", ladder[1].CostP80, ladder[2].CostP80)
	}
	if ladder[3].DurP80 <= ladder[2].DurP80 {
		t.Errorf("kapasitas tidak menaikkan P80 durasi: %v -> %v", ladder[2].DurP80, ladder[3].DurP80)
	}
	for i, r := range ladder {
		if r.JCL > r.OnTime+1e-12 || r.JCL > r.OnBudget+1e-12 {
			t.Errorf("lapisan %d: peluang bersama %v melebihi salah satu peluang marginal", i, r.JCL)
		}
		if r.JointAtP80 >= 0.8 {
			t.Errorf("lapisan %d: peluang bersama di P80 marginal = %v, seharusnya di bawah 0,8", i, r.JointAtP80)
		}
	}
}

func TestRiskHitRatesMatchProbabilities(t *testing.T) {
	cfg := baseConfig(8000)
	cfg.Layer = simulate.LayerRisks
	r, err := simulate.RunIntegrated(model.Activities, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, rk := range model.Risks {
		got := float64(r.RiskHits[rk.ID]) / float64(cfg.Iterations)
		if math.Abs(got-rk.ResidualProb) > 0.025 {
			t.Errorf("%s terjadi %.3f, mau sekitar %.2f", rk.ID, got, rk.ResidualProb)
		}
	}
}

func TestFrontierIsNonIncreasing(t *testing.T) {
	cfg := baseConfig(3000)
	cfg.Layer = simulate.LayerRisks
	r, err := simulate.RunIntegrated(model.Activities, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var ds []float64
	for d := r.DurP50; d <= r.Durations[len(r.Durations)-1]; d += 2 {
		ds = append(ds, d)
	}
	fr := r.Frontier(0.7, ds)
	prev := math.Inf(1)
	sawFeasible := false
	for _, p := range fr {
		if !p.Feasible {
			if sawFeasible {
				t.Errorf("tenggat %v tidak layak setelah tenggat lebih pendek sudah layak", p.Duration)
			}
			continue
		}
		sawFeasible = true
		if p.Budget > prev+1e-6 {
			t.Errorf("anggaran naik saat tenggat dilonggarkan: %v -> %v", prev, p.Budget)
		}
		if got := r.Joint(p.Duration, p.Budget); got < 0.7-1e-9 {
			t.Errorf("titik frontier (%v, %v) hanya mencapai JCL %v", p.Duration, p.Budget, got)
		}
		prev = p.Budget
	}
	if !sawFeasible {
		t.Error("tidak ada satu pun titik frontier yang layak")
	}
}

func TestDensityCountsEveryIteration(t *testing.T) {
	cfg := baseConfig(2000)
	cfg.Layer = simulate.LayerRisks
	r, err := simulate.RunIntegrated(model.Activities, cfg)
	if err != nil {
		t.Fatal(err)
	}
	g := r.Density(24, 18)
	total := 0
	for _, row := range g.Counts {
		for _, c := range row {
			total += c
		}
	}
	if total != cfg.Iterations {
		t.Errorf("isi grid %d, mau %d", total, cfg.Iterations)
	}
	if len(r.Pairs()) != cfg.Iterations {
		t.Errorf("pasangan = %d, mau %d", len(r.Pairs()), cfg.Iterations)
	}
}

func TestResourceLayerNeedsCalendar(t *testing.T) {
	cfg := baseConfig(10)
	cfg.Layer = simulate.LayerResources
	cfg.Calendar = nil
	if _, err := simulate.RunIntegrated(model.Activities, cfg); err == nil {
		t.Error("lapisan kapasitas tanpa kalender seharusnya galat")
	}
}
