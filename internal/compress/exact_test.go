package compress_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

func TestExactIsNeverWorseThanGreedy(t *testing.T) {
	g, err := compress.Crash(model.Activities, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	x, err := compress.Exact(model.Activities, model.RateCard, g)
	if err != nil {
		t.Fatal(err)
	}
	if x.Normal != 85 || x.ExactMin > g.MinDuration {
		t.Errorf("normal %d, minimum eksak %d (serakah %d)", x.Normal, x.ExactMin, g.MinDuration)
	}
	for _, p := range x.Points {
		if p.Greedy >= 0 && p.CrashCost > p.Greedy+0.5 {
			t.Errorf("T=%d: LP %v lebih mahal dari serakah %v - LP tidak optimal", p.Duration, p.CrashCost, p.Greedy)
		}
		if p.Rental < 0 || math.Abs(p.Total-(p.TotalCrash+p.Rental)) > 1e-3 {
			t.Errorf("T=%d: biaya total %v tidak sama dengan crash %v + sewa %v", p.Duration, p.Total, p.TotalCrash, p.Rental)
		}
		if p.Total < x.Optimum.Total-0.5 {
			t.Errorf("titik %d lebih murah dari optimum yang dilaporkan", p.Duration)
		}
	}
	// Laporan optimalitas serakah harus konsisten dengan titik-titiknya.
	var worst float64
	for _, p := range x.Points {
		if p.Greedy >= 0 && p.Greedy-p.CrashCost > worst {
			worst = p.Greedy - p.CrashCost
		}
	}
	if (worst > 0.5) == x.GreedyOptimal || math.Abs(worst-x.MaxGreedyExcess) > 0.5 && !x.GreedyOptimal {
		t.Errorf("GreedyOptimal %v, kelebihan terbesar %v, kelebihan dari titik %v", x.GreedyOptimal, x.MaxGreedyExcess, worst)
	}
	if be := x.BreakEvenPerDay(x.Normal - 5); be <= 0 {
		t.Errorf("nilai impas per hari = %v, seharusnya positif", be)
	}
	if x.BreakEvenPerDay(x.Normal) != 0 {
		t.Error("nilai impas pada durasi normal harus nol")
	}
	if _, ok := x.PointAt(x.ExactMin - 1); ok {
		t.Error("titik di bawah durasi minimum tidak boleh ada")
	}
}

// TestExactMatchesBruteForce: pada jaringan kecil, setiap kombinasi potongan
// dicoba dengan CPM; biaya termurah per tenggat harus sama dengan LP.
func TestExactMatchesBruteForce(t *testing.T) {
	be := []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}
	fe := []model.TeamSlot{{Role: model.RoleFE, Alloc: 1}}
	qa := []model.TeamSlot{{Role: model.RoleQA, Alloc: 1}}
	acts := []model.Activity{
		{ID: "S", Duration: 6, Optimistic: 3, Pessimistic: 9, Team: be},
		{ID: "P", Duration: 6, Optimistic: 3, Pessimistic: 9, Team: fe, Pred: []model.Predecessor{model.FS("S")}},
		{ID: "Q", Duration: 5, Optimistic: 3, Pessimistic: 9, Team: qa, Pred: []model.Predecessor{model.FS("S")}},
		{ID: "R", Duration: 3, Optimistic: 2, Pessimistic: 5, Team: be, Pred: []model.Predecessor{model.FS("Q")}},
		{ID: "E", Duration: 6, Optimistic: 3, Pessimistic: 9, Team: qa, Pred: []model.Predecessor{model.FS("P"), model.FS("R")}},
	}
	g, err := compress.Crash(acts, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	x, err := compress.Exact(acts, model.RateCard, g)
	if err != nil {
		t.Fatal(err)
	}
	plans := make([]model.CrashPlan, len(acts))
	for i, a := range acts {
		plans[i] = a.Crash(model.RateCard)
	}
	best := map[int]float64{}
	cut := make([]int, len(acts))
	var walk func(i int)
	walk = func(i int) {
		if i == len(acts) {
			dur := map[string]int{}
			var cost float64
			for k, a := range acts {
				dur[a.ID] = a.Duration - cut[k]
				cost += plans[k].CostToCut(cut[k])
			}
			r := schedule.MustCompute(acts, schedule.Options{DurationOf: func(a model.Activity) int { return dur[a.ID] }})
			for T := r.Duration; T <= x.Normal; T++ {
				if c, ok := best[T]; !ok || cost < c {
					best[T] = cost
				}
			}
			return
		}
		max := 0
		if plans[i].Allowed {
			max = plans[i].MaxDaysSaved
		}
		for k := 0; k <= max; k++ {
			cut[i] = k
			walk(i + 1)
		}
		cut[i] = 0
	}
	walk(0)
	for _, p := range x.Points {
		if want := best[p.Duration]; math.Abs(p.CrashCost-want) > 0.5 {
			t.Errorf("T=%d: LP %v, brute force %v", p.Duration, p.CrashCost, want)
		}
	}
	minT := x.Normal
	for T := range best {
		if T < minT {
			minT = T
		}
	}
	if x.ExactMin != minT {
		t.Errorf("durasi minimum LP %d, brute force %d", x.ExactMin, minT)
	}
}

// TestGreedyCanMissTheExactOptimum: serakah mendahulukan satu potongan yang
// memendekkan proyek sebelum mencoba pasangan. Pada jaringan dengan satu
// aktivitas bersama yang mahal dan dua cabang paralel yang murah, satu
// potongan bersama itu dipilih walaupun pasangan cabangnya lebih murah.
func TestGreedyCanMissTheExactOptimum(t *testing.T) {
	shared := []model.TeamSlot{{Role: model.RoleTL, Alloc: 1}, {Role: model.RoleBE, Alloc: 0.5}, {Role: model.RoleFE, Alloc: 0.5}}
	ux := []model.TeamSlot{{Role: model.RoleUX, Alloc: 1}}
	acts := []model.Activity{
		{ID: "S", Duration: 5, Optimistic: 4, Pessimistic: 7, Team: shared},
		{ID: "P", Duration: 5, Optimistic: 4, Pessimistic: 7, Team: ux, Pred: []model.Predecessor{model.FS("S")}},
		{ID: "Q", Duration: 5, Optimistic: 4, Pessimistic: 7, Team: ux, Pred: []model.Predecessor{model.FS("S")}},
	}
	g, err := compress.Crash(acts, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	x, err := compress.Exact(acts, model.RateCard, g)
	if err != nil {
		t.Fatal(err)
	}
	s, p := acts[0].Crash(model.RateCard), acts[1].Crash(model.RateCard)
	if len(g.Steps) == 0 || len(g.Steps[0].Crashed) != 1 || g.Steps[0].Crashed[0] != "S" {
		t.Fatalf("langkah serakah pertama %v, mau S", g.Steps)
	}
	pt, ok := x.PointAt(9)
	if !ok || math.Abs(pt.CrashCost-2*p.Marginal[0]) > 0.5 || pt.Cuts["S"] != 0 {
		t.Errorf("LP pada 9 hari: %+v, mau P + Q seharga %v", pt, 2*p.Marginal[0])
	}
	if x.GreedyOptimal || math.Abs(x.MaxGreedyExcess-(s.Marginal[0]-2*p.Marginal[0])) > 0.5 {
		t.Errorf("kelebihan serakah %v, mau %v", x.MaxGreedyExcess, s.Marginal[0]-2*p.Marginal[0])
	}
}

func TestExactRejectsUnsupportedRelations(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", Duration: 3, Optimistic: 2, Pessimistic: 5},
		{ID: "B", Duration: 3, Optimistic: 2, Pessimistic: 5, Pred: []model.Predecessor{{ID: "A", Type: "FF"}}},
	}
	if _, err := compress.Exact(acts, model.RateCard, compress.Curve{}); err == nil {
		t.Error("relasi FF belum didukung LP; seharusnya galat, bukan hasil diam-diam salah")
	}
}
