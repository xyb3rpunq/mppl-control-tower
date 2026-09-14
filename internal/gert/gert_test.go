package gert_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/gert"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestArcMoments(t *testing.T) {
	w := gert.Arc(1, 4)
	if w.Prob() != 1 || w.Mean() != 4 || w.Var() != 0 {
		t.Errorf("cabang pasti 4 hari: p %v, rerata %v, varians %v", w.Prob(), w.Mean(), w.Var())
	}
	if gert.Arc(0, 3).Mean() != 0 || gert.Arc(0, 3).Var() != 0 {
		t.Error("cabang berpeluang nol harus memberi momen nol, bukan NaN")
	}
}

// Seri: waktu dijumlahkan, peluang dikalikan.
func TestSeriesAddsTimeMultipliesProbability(t *testing.T) {
	w := gert.Arc(0.9, 3).Then(gert.Arc(0.5, 2))
	if !near(w.Prob(), 0.45, 1e-12) || !near(w.Mean(), 5, 1e-12) || !near(w.Var(), 0, 1e-12) {
		t.Errorf("seri: p %v rerata %v varians %v", w.Prob(), w.Mean(), w.Var())
	}
}

// Paralel: dua cabang eksklusif 40% x 2 hari dan 60% x 7 hari.
func TestParallelIsAMixture(t *testing.T) {
	w := gert.Or(gert.Arc(0.4, 2), gert.Arc(0.6, 7))
	mean := 0.4*2 + 0.6*7
	second := 0.4*4 + 0.6*49
	if !near(w.Prob(), 1, 1e-12) || !near(w.Mean(), mean, 1e-12) || !near(w.Var(), second-mean*mean, 1e-12) {
		t.Errorf("paralel: p %v rerata %v varians %v", w.Prob(), w.Mean(), w.Var())
	}
}

// TestSelfLoopMatchesClosedForm menjaga aljabar Mason terhadap bentuk
// tertutup sebaran geometrik.
func TestSelfLoopMatchesClosedForm(t *testing.T) {
	for _, c := range []struct{ p, r float64 }{{0.3, 2}, {0.25, 1.5}, {0.6, 4}} {
		w := gert.SelfLoop(c.p, c.r)
		mean := c.p * c.r / (1 - c.p)
		v := c.p * c.r * c.r / ((1 - c.p) * (1 - c.p))
		if !near(w.Prob(), 1, 1e-12) {
			t.Errorf("p=%v: putaran harus pasti berakhir, dapat %v", c.p, w.Prob())
		}
		if !near(w.Mean(), mean, 1e-12) || !near(w.Var(), v, 1e-9) {
			t.Errorf("p=%v r=%v: rerata %v (mau %v), varians %v (mau %v)", c.p, c.r, w.Mean(), mean, w.Var(), v)
		}
		if !near(w.SD(), math.Sqrt(v), 1e-9) {
			t.Errorf("simpangan baku %v, mau %v", w.SD(), math.Sqrt(v))
		}
		if !near(gert.ExpectedCycles(c.p)*c.r, mean, 1e-12) {
			t.Error("ExpectedCycles x rework harus sama dengan rerata tambahan")
		}
	}
}

func TestLoopThatNeverExitsIsInfinite(t *testing.T) {
	if w := gert.Loop(gert.Arc(0, 0), gert.Arc(1, 1)); !math.IsInf(w.Prob(), 1) {
		t.Errorf("putaran dengan peluang kembali 1 tidak pernah keluar, dapat p %v", w.Prob())
	}
	if !math.IsInf(gert.ExpectedCycles(1), 1) {
		t.Error("ExpectedCycles(1) harus tak hingga")
	}
}

// TestSampleCyclesMatchesGeometric: sampel transformasi invers harus memberi
// P(N >= k) = p^k dan rerata p/(1-p).
func TestSampleCyclesMatchesGeometric(t *testing.T) {
	const p = 0.3
	const n = 200000
	var sum float64
	atLeast2 := 0
	for i := 0; i < n; i++ {
		u := (float64(i) + 0.5) / n // grid seragam: tanpa derau sampel
		c := gert.SampleCycles(u, p)
		sum += float64(c)
		if c >= 2 {
			atLeast2++
		}
	}
	if got := sum / n; !near(got, gert.ExpectedCycles(p), 1e-3) {
		t.Errorf("rerata putaran %v, mau %v", got, gert.ExpectedCycles(p))
	}
	if got := float64(atLeast2) / n; !near(got, gert.AtLeast(p, 2), 1e-4) {
		t.Errorf("P(N>=2) %v, mau %v", got, gert.AtLeast(p, 2))
	}
	if gert.SampleCycles(0.5, 0) != 0 || gert.SampleCycles(0, 0.3) != gert.MaxCycles || gert.SampleCycles(1, 0.3) != 0 {
		t.Error("kasus batas SampleCycles salah")
	}
	if gert.SampleCycles(1e-300, 0.3) != gert.MaxCycles || gert.SampleCycles(0.5, 1) != gert.MaxCycles {
		t.Error("sampel harus dipotong di MaxCycles")
	}
}

func TestCycleQuantile(t *testing.T) {
	// p = 0,3: P(N<=0) = 0,7; P(N<=1) = 0,91; P(N<=2) = 0,973.
	for _, c := range []struct {
		q    float64
		want int
	}{{0.5, 0}, {0.7, 0}, {0.8, 1}, {0.91, 1}, {0.95, 2}} {
		if got := gert.CycleQuantile(0.3, c.q); got != c.want {
			t.Errorf("kuantil %v = %d, mau %d", c.q, got, c.want)
		}
	}
	if gert.CycleQuantile(0, 0.9) != 0 || gert.CycleQuantile(1, 0.9) != math.MaxInt32 {
		t.Error("kasus batas CycleQuantile salah")
	}
}
