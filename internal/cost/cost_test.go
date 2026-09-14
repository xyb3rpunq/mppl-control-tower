package cost_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/cost"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

func TestRentalsReproduceTheBudgetOnPlan(t *testing.T) {
	rs, fixed, err := cost.Rentals(model.Activities)
	if err != nil {
		t.Fatal(err)
	}
	plan := schedule.MustCompute(model.Activities, schedule.Options{})
	var extras, rentals float64
	for _, a := range model.Activities {
		extras += a.ExtraCost()
	}
	for _, r := range rs {
		start := plan.Task(r.Activity).StartX
		got := r.Cost(start, plan.Duration)
		if math.Abs(got-r.Amount) > 1e-6 {
			t.Errorf("%s: biaya pada jadwal rencana %v, mau %v", r.Activity, got, r.Amount)
		}
		if r.PlanSpan != plan.Duration-start || r.Index < 0 || model.Activities[r.Index].ID != r.Activity {
			t.Errorf("%s: rentang atau indeks salah", r.Activity)
		}
		rentals += r.Amount
	}
	if math.Abs(fixed+rentals-extras) > 1e-6 {
		t.Errorf("biaya tetap %v + sewa %v harus sama dengan seluruh biaya non-tenaga-kerja %v", fixed, rentals, extras)
	}
	if len(rs) != 4 || rentals != 2_000_000 {
		t.Errorf("sewa & langganan: %d pos senilai %v, mau 4 pos senilai 2.000.000", len(rs), rentals)
	}
	var want float64
	for _, r := range rs {
		want += r.Rate
	}
	if cost.DailyRate(rs) != want {
		t.Error("DailyRate harus menjumlah seluruh tarif")
	}
}

func TestRentalCostScalesWithSpan(t *testing.T) {
	r := cost.Rental{Rate: 10_000}
	if r.Cost(10, 30) != 200_000 {
		t.Errorf("20 hari x 10.000 = %v", r.Cost(10, 30))
	}
	if r.Cost(30, 10) != 0 {
		t.Error("rentang negatif harus bernilai nol, bukan biaya negatif")
	}
}

func TestRentalsSkipDegenerateSpans(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", Duration: 0, Milestone: true, Extras: []model.Extra{{Amount: 100, TimeBased: true}}},
	}
	rs, fixed, err := cost.Rentals(acts)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 0 || fixed != 100 {
		t.Errorf("sewa tanpa rentang harus diperlakukan sebagai biaya tetap: %d sewa, tetap %v", len(rs), fixed)
	}
	if _, _, err := cost.Rentals([]model.Activity{{ID: "X", Pred: []model.Predecessor{model.FS("?")}}}); err == nil {
		t.Error("jaringan tidak sah seharusnya galat")
	}
}
