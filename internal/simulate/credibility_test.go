package simulate_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// TestPERTVarIsBetaVariance membandingkan rumus tertutup dengan varians yang
// diintegrasikan secara numerik dari kuantil sampler itu sendiri.
func TestPERTVarIsBetaVariance(t *testing.T) {
	acts := []model.Activity{
		{ID: "S", Duration: 6, Optimistic: 4, Pessimistic: 11},
		{ID: "K", Duration: 2, Optimistic: 1, Pessimistic: 6},
		{ID: "F", Duration: 3, Optimistic: 3, Pessimistic: 3},
		{ID: "M", Milestone: true},
	}
	s := simulate.NewSampler(acts)
	for i := 0; i < 2; i++ {
		const n = 20000
		var m1, m2 float64
		for k := 0; k < n; k++ {
			x := s.Value(i, (float64(k)+0.5)/n, "pert")
			m1 += x / n
			m2 += x * x / n
		}
		numeric := m2 - m1*m1
		if got := s.PERTVar(i); math.Abs(got-numeric)/numeric > 0.01 {
			t.Errorf("%s: PERTVar %v, integrasi numerik %v", acts[i].ID, got, numeric)
		}
	}
	if s.PERTVar(2) != 0 || s.PERTVar(3) != 0 {
		t.Error("aktivitas tetap dan milestone tidak punya varians")
	}
}

// TestCredibilityMomentEstimator mengunci perilaku estimator Buhlmann pada
// titik-titik yang bisa dihitung dengan tangan.
func TestCredibilityMomentEstimator(t *testing.T) {
	cases := []struct {
		obs, prior, v float64
		z, tau2       float64
	}{
		{1, 1, 0.01, 0, 0},        // tanpa selisih: rencana dipertahankan
		{1.05, 1, 0.01, 0, 0},     // selisih 0,05^2 = 0,0025 < derau 0,01
		{1.2, 1, 0.02, 0.5, 0.02}, // selisih^2 = 0,04 = dua kali derau
		{1.3, 1, 0.01, 0.8889, 0.08},
		{1.1, 1, 0, 1, 0.01},  // tanpa derau, selisih apa pun dipercaya
		{1, 1, 0, 0, 0},       // tanpa derau dan tanpa selisih
		{0.9, 1, -5, 1, 0.01}, // varians negatif diperlakukan nol
	}
	for _, c := range cases {
		z, tau2 := simulate.Credibility(c.obs, c.prior, c.v)
		if math.Abs(z-c.z) > 1e-4 || math.Abs(tau2-c.tau2) > 1e-9 {
			t.Errorf("Credibility(%v, %v, %v) = %v, %v; mau %v, %v", c.obs, c.prior, c.v, z, tau2, c.z, c.tau2)
		}
	}
	// Z naik monoton terhadap besarnya selisih.
	prev := -1.0
	for d := 0.0; d <= 1; d += 0.05 {
		z, _ := simulate.Credibility(1+d, 1, 0.01)
		if z < prev-1e-12 {
			t.Fatalf("Z turun di selisih %v", d)
		}
		prev = z
	}
}

// TestInFlightCredibilityInputs memeriksa varians penaksir yang dipakai
// prakiraan berjalan terhadap definisinya.
func TestInFlightCredibilityInputs(t *testing.T) {
	fl, _, _ := inflight(t, 0)
	s := simulate.NewSampler(model.Activities)
	var sumMean, sumVar float64
	for i, a := range model.Activities {
		if fl.State[i].Kind == simulate.Completed && !a.Milestone {
			sumMean += s.PERTMean(i)
			sumVar += s.PERTVar(i)
		}
	}
	if want := sumVar / (sumMean * sumMean); math.Abs(fl.DurationVar-want) > 1e-15 {
		t.Errorf("varians penaksir durasi %v, mau %v", fl.DurationVar, want)
	}
	if fl.CostVar <= 0 || fl.ExamVar <= 0 {
		t.Errorf("varians biaya %v dan ujian %v harus positif", fl.CostVar, fl.ExamVar)
	}
	z, tau2 := simulate.Credibility(fl.ObservedCostRatio, 1, fl.CostVar)
	if math.Abs(fl.CostCredibility-z) > 1e-12 || math.Abs(fl.CostTau2-tau2) > 1e-15 {
		t.Errorf("kredibilitas biaya %v, mau %v", fl.CostCredibility, z)
	}
	if want := z*fl.ObservedCostRatio + 1 - z; math.Abs(fl.CostFactor-want) > 1e-12 {
		t.Errorf("faktor biaya %v, mau %v", fl.CostFactor, want)
	}
	// Laju saat ujian dibatasi 1: ujian tidak pernah menaikkan kapasitas.
	ze, _ := simulate.Credibility(math.Min(1, fl.ExamObserved), model.ExamCapacityFactor, fl.ExamVar)
	if math.Abs(fl.ExamCredibility-ze) > 1e-12 {
		t.Errorf("kredibilitas ujian %v, mau %v", fl.ExamCredibility, ze)
	}

	// Bobot yang dipaksakan mengembalikan rumus lama n / (n + k) untuk semua.
	forced, _, _ := inflight(t, 10)
	want := float64(forced.Evidence) / (float64(forced.Evidence) + 10)
	if forced.Empirical || math.Abs(forced.Credibility-want) > 1e-12 || math.Abs(forced.CostCredibility-want) > 1e-12 {
		t.Errorf("bobot paksa: empiris %v, Z %v dan %v, mau %v", forced.Empirical, forced.Credibility, forced.CostCredibility, want)
	}
}
