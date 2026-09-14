package level_test

import (
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Jaringan kecil dengan urutan buruk yang diketahui: A dan B berebut satu
// Backend Developer, C (Frontend) menunggu B. Mendahulukan A memberi 9 hari;
// mendahulukan B memberi 6 hari, dan 6 adalah batas bawahnya.
func searchInstance(t *testing.T) ([]model.Activity, level.Options) {
	t.Helper()
	acts := []model.Activity{
		{ID: "A", Duration: 3, Team: be(1)},
		{ID: "B", Duration: 3, Team: be(1)},
		{ID: "C", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleFE, Alloc: 1}}, Pred: []model.Predecessor{{ID: "B"}}},
	}
	opts := level.Options{Calendar: cal(t), Capacity: map[model.Role]float64{model.RoleBE: 1, model.RoleFE: 1}}
	return acts, opts
}

func TestSearchReachesTheBound(t *testing.T) {
	acts, opts := searchInstance(t)
	bad := opts
	bad.Order = []string{"A", "B", "C"}
	start, err := level.Run(acts, bad)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := level.LowerBound(acts, opts)
	if err != nil {
		t.Fatal(err)
	}
	if start.Duration != 9 || bound.Value != 6 {
		t.Fatalf("awal %d hari, batas bawah %d; mau 9 dan 6", start.Duration, bound.Value)
	}

	got, ok, err := level.Search(acts, opts, start, bound.Value, 50, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.Duration != 6 {
		t.Errorf("pencarian berhenti di %d hari (terbukti %v), mau 6", got.Duration, ok)
	}
	assertFeasible(t, acts, got, "search")

	again, _, _ := level.Search(acts, opts, start, bound.Value, 50, 7)
	if again.Duration != got.Duration || len(again.Order) != len(got.Order) {
		t.Error("benih yang sama harus memberi hasil yang sama")
	}
	for i := range got.Order {
		if again.Order[i] != got.Order[i] {
			t.Fatal("benih yang sama harus memberi urutan yang sama")
		}
	}
}

func TestSearchReportsAnUnreachableTarget(t *testing.T) {
	acts, opts := searchInstance(t)
	bad := opts
	bad.Order = []string{"A", "B", "C"}
	start, _ := level.Run(acts, bad)

	got, ok, err := level.Search(acts, opts, start, 5, 20, 1)
	if err != nil {
		t.Fatal(err)
	}
	if ok || got.Duration > start.Duration || got.Duration != 6 {
		t.Errorf("target mustahil: %d hari, terbukti %v; mau 6 hari dan tidak terbukti", got.Duration, ok)
	}

	// Jadwal awal yang sudah mencapai target dikembalikan tanpa pencarian.
	if r, ok, _ := level.Search(acts, opts, got, 6, 0, 1); !ok || r.Duration != 6 {
		t.Error("jadwal yang sudah di target harus langsung dinyatakan terbukti")
	}
	// Tanpa langkah lokal, aturan prioritas tetap dicoba.
	if r, ok, _ := level.Search(acts, opts, start, 6, 0, 1); !ok || r.Duration != 6 {
		t.Errorf("tanpa langkah lokal: %d hari, terbukti %v", r.Duration, ok)
	}
}

func TestSearchRejectsACycle(t *testing.T) {
	acts := []model.Activity{
		{ID: "X", Duration: 2, Team: be(1), Pred: []model.Predecessor{{ID: "Y"}}},
		{ID: "Y", Duration: 2, Team: be(1), Pred: []model.Predecessor{{ID: "X"}}},
	}
	_, opts := searchInstance(t)
	if _, ok, err := level.Search(acts, opts, level.Result{Duration: 10}, 4, 5, 1); err == nil || ok {
		t.Error("jaringan bersiklus harus menghasilkan galat")
	}
}
