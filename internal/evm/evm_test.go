package evm_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/evm"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

func engineFor(t *testing.T, acts []model.Activity) *evm.Engine {
	t.Helper()
	plan, err := schedule.Compute(acts, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	return evm.New(acts, plan, model.RateCard)
}

// Jaringan sintetis dengan aritmetika yang bisa diperiksa tangan:
// satu aktivitas 10 hari, anggaran 1.000.000, realisasinya 20 hari dan
// menghabiskan 1.500.000. Pada hari ke-10:
//
//	PV = 1.000.000 (rencana sudah rampung)
//	EV =   500.000 (baru setengah jalan)
//	AC =   750.000 (setengah dari 1.500.000)
//	SPI = 0,5      CPI = 0,667
func synthetic() []model.Activity {
	return []model.Activity{
		{
			ID: "X", WBS: "1.1", Duration: 10,
			Team:   []model.TeamSlot{{Role: model.RolePM, Alloc: 1}}, // 10 x 60.000 = 600.000
			Extras: []model.Extra{{Key: "tools", Amount: 400_000}},   // total 1.000.000
			Actual: model.Actual{Started: true, Start: 0, Duration: 20, Cost: 1_500_000},
		},
	}
}

func TestSnapshotHandCheck(t *testing.T) {
	e := engineFor(t, synthetic())
	s := e.At(10)

	check := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1 {
			t.Errorf("%s = %.2f, mau %.2f", name, got, want)
		}
	}
	check("BAC", s.BAC, 1_000_000)
	check("PV", s.PV, 1_000_000)
	check("EV", s.EV, 500_000)
	check("AC", s.AC, 750_000)
	check("SV", s.SV, -500_000)
	check("CV", s.CV, -250_000)

	if math.Abs(s.SPI-0.5) > 1e-9 {
		t.Errorf("SPI = %v, mau 0,5", s.SPI)
	}
	if math.Abs(s.CPI-2.0/3.0) > 1e-9 {
		t.Errorf("CPI = %v, mau 0,6667", s.CPI)
	}
	// EAC tipikal = BAC / CPI = 1.000.000 / (2/3) = 1.500.000
	check("EAC tipikal", s.EACTypical, 1_500_000)
	// EAC optimistis = AC + (BAC - EV) = 750.000 + 500.000 = 1.250.000
	check("EAC optimistis", s.EACOptimistic, 1_250_000)
	// TCPI = (BAC - EV) / (BAC - AC) = 500.000 / 250.000 = 2
	if math.Abs(s.TCPI-2) > 1e-9 {
		t.Errorf("TCPI = %v, mau 2", s.TCPI)
	}
	check("VAC", s.VAC, -500_000)
}

func TestIdentitiesHoldOnRealProject(t *testing.T) {
	e := engineFor(t, model.Activities)
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 200)

	for _, iso := range []string{"2025-11-14", model.DefaultStatusDate, "2026-01-09"} {
		day := cal.FractionalIndexOf(iso)
		s := e.At(day)

		if math.Abs(s.SV-(s.EV-s.PV)) > 1e-6 {
			t.Errorf("%s: SV bukan EV-PV", iso)
		}
		if math.Abs(s.CV-(s.EV-s.AC)) > 1e-6 {
			t.Errorf("%s: CV bukan EV-AC", iso)
		}
		if s.PV > 0 && math.Abs(s.SPI-s.EV/s.PV) > 1e-9 {
			t.Errorf("%s: SPI bukan EV/PV", iso)
		}
		if s.AC > 0 && math.Abs(s.CPI-s.EV/s.AC) > 1e-9 {
			t.Errorf("%s: CPI bukan EV/AC", iso)
		}
		if math.Abs(s.VAC-(s.BAC-s.EACTypical)) > 1e-6 {
			t.Errorf("%s: VAC bukan BAC-EAC", iso)
		}
		if math.Abs(s.ETC-(s.EACTypical-s.AC)) > 1e-6 {
			t.Errorf("%s: ETC bukan EAC-AC", iso)
		}
		// EV tidak boleh melampaui PV kumulatif akhir maupun BAC.
		if s.EV > s.BAC+1e-6 {
			t.Errorf("%s: EV %.2f melebihi BAC %.2f", iso, s.EV, s.BAC)
		}
	}
}

func TestCurvesAreMonotonic(t *testing.T) {
	e := engineFor(t, model.Activities)
	c := e.BuildCurves(44)

	mono := func(name string, xs []float64) {
		t.Helper()
		for i := 1; i < len(xs); i++ {
			if xs[i] < xs[i-1]-1e-6 {
				t.Errorf("%s turun di hari %d: %.2f setelah %.2f", name, i, xs[i], xs[i-1])
				return
			}
		}
	}
	mono("PV", c.PV)
	mono("EV", c.EV)
	mono("AC", c.AC)

	// Kurva PV harus berakhir tepat di BAC - kalau tidak, ada anggaran
	// aktivitas yang tidak pernah masuk kurva sama sekali.
	if last := c.PV[len(c.PV)-1]; math.Abs(last-e.BAC()) > 1 {
		t.Errorf("PV berakhir di %.2f, mau BAC %.2f", last, e.BAC())
	}
}

func TestEarnedScheduleMatchesPVCurve(t *testing.T) {
	e := engineFor(t, model.Activities)
	s := e.At(44)

	// Menurut definisinya, PV pada titik ES harus sama dengan EV sekarang.
	pvAtES := e.PVAt(s.ES)
	if math.Abs(pvAtES-s.EV) > s.BAC*0.005 {
		t.Errorf("PV(ES) = %.2f, mau EV = %.2f", pvAtES, s.EV)
	}
	if math.Abs(s.SVt-(s.ES-s.AtDay)) > 1e-9 {
		t.Errorf("SV(t) bukan ES - AT")
	}
	// Proyek tertinggal, jadi ES harus berada di belakang waktu berjalan.
	if s.ES >= s.AtDay {
		t.Errorf("ES %.2f tidak di belakang AT %.2f padahal SPI %.3f < 1", s.ES, s.AtDay, s.SPI)
	}
}

func TestPhaseRowsSumToWhole(t *testing.T) {
	e := engineFor(t, model.Activities)
	day := 44.0
	s := e.At(day)

	var pv, ev, ac, budget float64
	for _, r := range e.ByPhase(day) {
		pv += r.PV
		ev += r.EV
		ac += r.AC
		budget += r.Budget
	}
	tol := 1.0
	if math.Abs(pv-s.PV) > tol || math.Abs(ev-s.EV) > tol || math.Abs(ac-s.AC) > tol {
		t.Errorf("jumlah per fase tidak sama dengan total: PV %.2f/%.2f EV %.2f/%.2f AC %.2f/%.2f",
			pv, s.PV, ev, s.EV, ac, s.AC)
	}
	if math.Abs(budget-s.BAC) > tol {
		t.Errorf("jumlah anggaran fase %.2f tidak sama dengan BAC %.2f", budget, s.BAC)
	}
}

func TestActivityRowsSumToWhole(t *testing.T) {
	e := engineFor(t, model.Activities)
	day := 44.0
	s := e.At(day)

	var ev, ac float64
	for _, r := range e.ByActivity(day) {
		ev += r.EV
		ac += r.AC
	}
	if math.Abs(ev-s.EV) > 1 || math.Abs(ac-s.AC) > 1 {
		t.Errorf("jumlah per aktivitas tidak sama dengan total: EV %.2f/%.2f AC %.2f/%.2f", ev, s.EV, ac, s.AC)
	}
}

func TestZeroProgressAtProjectStart(t *testing.T) {
	e := engineFor(t, model.Activities)
	s := e.At(0)
	if s.PV != 0 || s.EV != 0 || s.AC != 0 {
		t.Errorf("pada hari 0 semuanya harus nol, dapat PV %.2f EV %.2f AC %.2f", s.PV, s.EV, s.AC)
	}
	// Pembagian nol tidak boleh menghasilkan NaN atau Inf yang bocor ke halaman.
	for name, v := range map[string]float64{"SPI": s.SPI, "CPI": s.CPI, "TCPI": s.TCPI} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Errorf("%s = %v pada hari 0; seharusnya nilai terhingga", name, v)
		}
	}
}

func TestEnginePlanDays(t *testing.T) {
	plan := schedule.MustCompute(model.Activities, schedule.Options{})
	if got := evm.New(model.Activities, plan, model.RateCard).PlanDays(); got != plan.Duration {
		t.Errorf("PlanDays = %d, mau %d", got, plan.Duration)
	}
}
