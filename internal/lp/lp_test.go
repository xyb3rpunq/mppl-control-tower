package lp_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/lp"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-7 }

// Contoh buku teks: maks 3x + 5y dengan x <= 4, 2y <= 12, 3x + 2y <= 18.
// Optimum x = 2, y = 6, nilai 36.
func TestTextbookMaximisation(t *testing.T) {
	s := lp.Minimize([]float64{-3, -5}, [][]float64{{1, 0}, {0, 2}, {3, 2}}, []float64{4, 12, 18})
	if s.Status != lp.Optimal {
		t.Fatalf("status %v", s.Status)
	}
	if !near(s.X[0], 2) || !near(s.X[1], 6) || !near(-s.Objective, 36) {
		t.Errorf("x %v, nilai %v; mau (2, 6) dan 36", s.X, -s.Objective)
	}
}

// Batasan >= (b negatif) butuh fase 1: min x + y dengan x + y >= 3, x >= 1.
func TestPhaseOneWithGreaterEqual(t *testing.T) {
	s := lp.Minimize([]float64{1, 1}, [][]float64{{-1, -1}, {-1, 0}}, []float64{-3, -1})
	if s.Status != lp.Optimal || !near(s.Objective, 3) || s.X[0] < 1-1e-9 {
		t.Errorf("status %v, nilai %v, x %v; mau nilai 3 dengan x >= 1", s.Status, s.Objective, s.X)
	}
}

func TestInfeasibleAndUnbounded(t *testing.T) {
	// x <= 1 dan x >= 2.
	if s := lp.Minimize([]float64{1}, [][]float64{{1}, {-1}}, []float64{1, -2}); s.Status != lp.Infeasible {
		t.Errorf("seharusnya tidak layak, dapat %v", s.Status)
	}
	// min -x tanpa batas atas.
	if s := lp.Minimize([]float64{-1}, [][]float64{{-1}}, []float64{0}); s.Status != lp.Unbounded {
		t.Errorf("seharusnya tak terbatas, dapat %v", s.Status)
	}
	for _, st := range []lp.Status{lp.Optimal, lp.Infeasible, lp.Unbounded} {
		if st.String() == "" {
			t.Error("status tanpa nama")
		}
	}
}

// Masalah degeneratif klasik Beale yang membuat aturan Dantzig berputar;
// aturan Bland harus tetap selesai dengan nilai optimum -1/20.
func TestBealeDegenerateDoesNotCycle(t *testing.T) {
	c := []float64{-0.75, 150, -0.02, 6}
	A := [][]float64{
		{0.25, -60, -0.04, 9},
		{0.5, -90, -0.02, 3},
		{0, 0, 1, 0},
	}
	s := lp.Minimize(c, A, []float64{0, 0, 1})
	if s.Status != lp.Optimal || !near(s.Objective, -0.05) {
		t.Errorf("status %v nilai %v, mau optimal -0,05", s.Status, s.Objective)
	}
}

// Baris artifisial yang redundan (dua batasan >= identik) tidak boleh
// menggagalkan fase 2.
func TestRedundantArtificialRow(t *testing.T) {
	s := lp.Minimize([]float64{2, 1}, [][]float64{{-1, -1}, {-1, -1}, {1, 0}}, []float64{-4, -4, 10})
	if s.Status != lp.Optimal || !near(s.Objective, 4) {
		t.Errorf("status %v nilai %v, mau 4", s.Status, s.Objective)
	}
}
