package simulate_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

func TestAccelerationGridsFollowTheRules(t *testing.T) {
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 400)
	if _, err := (&simulate.Acceleration{OvertimeHours: 4}).Grids(cal, model.Capacity, 300, 0); err == nil {
		t.Error("4 jam setiap hari = 20 jam seminggu; seharusnya ditolak")
	}
	if _, err := (&simulate.Acceleration{}).Grids(nil, model.Capacity, 300, 0); err == nil {
		t.Error("tanpa kalender seharusnya galat")
	}
	if _, err := (&simulate.Acceleration{Hire: map[model.Role]float64{"XX": 1}}).Grids(cal, model.Capacity, 300, 0); err == nil {
		t.Error("peran yang tidak dikenal seharusnya galat")
	}

	uas := cal.IndexOf("2026-01-20") // UAS ganjil
	from := uas - 10
	ac := &simulate.Acceleration{
		From: from, OvertimeRoles: []model.Role{model.RoleBE, model.RoleOPS}, OvertimeHours: 3.6,
		Hire: map[model.Role]float64{model.RoleBE: 1}, RampDays: 4, RampFactor: 0.5,
	}
	g, err := ac.Grids(cal, model.Capacity, 300, 0)
	if err != nil {
		t.Fatal(err)
	}
	win := level.CapacityGridWith(cal, model.Capacity, true, 300, 0)
	be := model.RoleBE
	if g.Normal[be][from-1] != win[be][from-1] || g.Max[be][from-1] != win[be][from-1] {
		t.Error("sebelum tanggal keputusan tidak boleh ada kapasitas tambahan")
	}
	if math.Abs(g.Normal[be][from]-(win[be][from]+0.5)) > 1e-9 || math.Abs(g.Normal[be][from+4]-(win[be][from+4]+1)) > 1e-9 {
		t.Errorf("orang baru: %v lalu %v, mau +0,5 selama adaptasi lalu +1", g.Normal[be][from]-win[be][from], g.Normal[be][from+4]-win[be][from+4])
	}
	if g.Persons[be] != 2 {
		t.Errorf("orang BE %v, mau 2", g.Persons[be])
	}
	if math.Abs(g.Max[be][from]-g.Normal[be][from]-2*0.45) > 1e-9 || math.Abs(g.Rate[be][from]-1.45) > 1e-9 {
		t.Errorf("lembur hari biasa: +%v kapasitas, laju %v; mau +0,9 dan 1,45", g.Max[be][from]-g.Normal[be][from], g.Rate[be][from])
	}
	if g.Max[be][uas] != g.Normal[be][uas] || g.Rate[be][uas] != 1 {
		t.Error("tidak boleh ada lembur pada hari ujian")
	}
	if _, ok := g.Rate[model.RoleOPS]; ok || g.Max[model.RoleOPS][from] != g.Normal[model.RoleOPS][from] {
		t.Error("DevOps paruh waktu tanpa orang baru tidak boleh lembur")
	}
}

// TestAccelerationPayByHand: satu pekerjaan BE 6 hari dengan lembur 3,6 jam
// setiap hari dan satu orang baru.
func TestAccelerationPayByHand(t *testing.T) {
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 400)
	caps := map[model.Role]float64{model.RoleBE: 1}
	acts := []model.Activity{{ID: "A", Duration: 6, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}}}
	ot := &simulate.Acceleration{OvertimeRoles: []model.Role{model.RoleBE}, OvertimeHours: 3.6}
	g, err := ot.Grids(cal, caps, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := level.Run(acts, level.Options{Calendar: cal, Capacity: caps, CapGrid: g.Max, RateCap: g.Rate, Horizon: 30})
	if err != nil {
		t.Fatal(err)
	}
	// 1,45 hari kerja per hari: empat hari penuh lembur (5,8) lalu 0,2 di hari kelima.
	p := ot.Pay(r, acts, g, model.RateCard)
	perDay := model.OvertimeUnits(3.6) * model.RateCard[model.RoleBE] / 8
	if r.Duration != 5 || math.Abs(p.OvertimeHours-4*3.6) > 1e-9 || math.Abs(p.Overtime-4*perDay) > 1e-6 || p.Hire != 0 {
		t.Errorf("durasi %d, %v jam, upah %v; mau 5, 14,4 jam, %v", r.Duration, p.OvertimeHours, p.Overtime, 4*perDay)
	}

	hire := &simulate.Acceleration{From: 2, Hire: map[model.Role]float64{model.RoleBE: 1}}
	g2, err := hire.Grids(cal, caps, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	two := append(acts, model.Activity{ID: "B", Duration: 6, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}})
	r2, err := level.Run(two, level.Options{Calendar: cal, Capacity: caps, CapGrid: g2.Max, RateCap: g2.Rate, Horizon: 30})
	if err != nil {
		t.Fatal(err)
	}
	p2 := hire.Pay(r2, two, g2, model.RateCard)
	last := 0
	for _, tk := range r2.Tasks {
		if tk.Finish > last {
			last = tk.Finish
		}
	}
	if want := float64(last-2) * model.RateCard[model.RoleBE]; math.Abs(p2.Hire-want) > 1e-6 || p2.HireDays != float64(last-2) || p2.Total() != p2.Hire {
		t.Errorf("upah orang baru %v untuk %v hari, mau %v (selesai hari %d)", p2.Hire, p2.HireDays, want, last)
	}
}

// TestAccelerationInTheForecast: opsi kosong tidak mengubah apa pun, dan
// orang baru tidak pernah membuat prakiraan lebih lambat.
func TestAccelerationInTheForecast(t *testing.T) {
	fl, cal, _ := inflight(t, 0)
	base := baseConfig(300)
	base.Layer, base.InFlight, base.ExamFactor, base.ExactLevel, base.Calendar = simulate.LayerResources, fl, fl.ExamFactor, true, cal
	plain, err := simulate.RunIntegrated(model.Activities, base)
	if err != nil {
		t.Fatal(err)
	}
	empty := base
	empty.Accel = &simulate.Acceleration{From: fl.Now}
	same, err := simulate.RunIntegrated(model.Activities, empty)
	if err != nil {
		t.Fatal(err)
	}
	for i := range plain.Durations {
		if plain.Durations[i] != same.Durations[i] || plain.Costs[i] != same.Costs[i] {
			t.Fatalf("opsi kosong mengubah iterasi terurut ke-%d", i)
		}
	}
	hire := base
	hire.Accel = &simulate.Acceleration{From: fl.Now, Hire: map[model.Role]float64{model.RoleBE: 1}}
	h, err := simulate.RunIntegrated(model.Activities, hire)
	if err != nil {
		t.Fatal(err)
	}
	if h.DurP80 > plain.DurP80 || h.AccelHire <= 0 || h.AccelOvertime != 0 || h.Proof.ProvenShare() != 1 {
		t.Errorf("orang baru: P80 %v (tanpa %v), upah %v, lembur %v, terbukti %v", h.DurP80, plain.DurP80, h.AccelHire, h.AccelOvertime, h.Proof.ProvenShare())
	}
	if h.CostMean < plain.CostMean {
		t.Errorf("rerata biaya dengan orang baru %v di bawah tanpa %v, padahal upahnya dibayar", h.CostMean, plain.CostMean)
	}
	bad := base
	bad.Accel = &simulate.Acceleration{OvertimeHours: 9}
	if _, err := simulate.RunIntegrated(model.Activities, bad); err == nil {
		t.Error("lembur yang melanggar batas harus ditolak simulasi")
	}
}

// TestFirstFeasibleMatchesFrontier: titik layak pertama yang dihitung langsung
// harus sama dengan pemindaian Frontier.
func TestFirstFeasibleMatchesFrontier(t *testing.T) {
	c := baseConfig(2000)
	c.Layer = simulate.LayerRework
	r, err := simulate.RunIntegrated(model.Activities, c)
	if err != nil {
		t.Fatal(err)
	}
	var ds []float64
	for d := math.Floor(r.DurP50); d <= r.Durations[len(r.Durations)-1]; d++ {
		ds = append(ds, d)
	}
	var want simulate.FrontierPoint
	for _, p := range r.Frontier(0.7, ds) {
		if p.Feasible {
			want = p
			break
		}
	}
	if got := simulate.FirstFeasible(r.Pairs(), 0.7); got != want {
		t.Errorf("FirstFeasible %+v, Frontier %+v", got, want)
	}
	if got := simulate.FirstFeasible(nil, 0.7); got.Feasible {
		t.Error("tanpa iterasi tidak ada titik layak")
	}
	if got := simulate.FirstFeasible([][2]float64{{3, 10}}, 1.5); got.Feasible {
		t.Error("target di atas 100% tidak mungkin layak")
	}
}

// TestPairedBootstrap: berbenih deterministik, menarik ulang indeks yang sama
// untuk setiap hasil, dan menolak jumlah iterasi yang berbeda.
func TestPairedBootstrap(t *testing.T) {
	c := baseConfig(1000)
	c.Layer = simulate.LayerRework
	a, _ := simulate.RunIntegrated(model.Activities, c)
	b1, err := simulate.PairedBootstrap([]simulate.IntegratedResult{a, a}, 0.7, 50, 7)
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := simulate.PairedBootstrap([]simulate.IntegratedResult{a, a}, 0.7, 50, 7)
	if len(b1) != 50 {
		t.Fatalf("%d ulangan, mau 50", len(b1))
	}
	point := simulate.FirstFeasible(a.Pairs(), 0.7)
	var lower, higher int
	for i := range b1 {
		// Dua salinan hasil yang sama pada indeks yang sama harus identik.
		if b1[i][0] != b1[i][1] || b1[i][0] != b2[i][0] {
			t.Fatal("bootstrap berpasangan tidak menarik indeks yang sama atau tidak berbenih")
		}
		if b1[i][0].Budget < point.Budget {
			lower++
		} else if b1[i][0].Budget > point.Budget {
			higher++
		}
	}
	if lower == 0 || higher == 0 {
		t.Errorf("sebaran bootstrap tidak mengapit angka titik: %d di bawah, %d di atas", lower, higher)
	}
	c.Iterations = 500
	short, _ := simulate.RunIntegrated(model.Activities, c)
	if _, err := simulate.PairedBootstrap([]simulate.IntegratedResult{a, short}, 0.7, 5, 1); err == nil {
		t.Error("jumlah iterasi berbeda seharusnya galat")
	}
	if out, err := simulate.PairedBootstrap(nil, 0.7, 5, 1); out != nil || err != nil {
		t.Error("tanpa hasil, bootstrap kosong tanpa galat")
	}
}

// TestReworkScaleMovesLoops: skala peluang gagal menaikkan dan menurunkan
// rerata putaran sesuai p/(1-p), dengan bilangan acak yang sama.
func TestReworkScaleMovesLoops(t *testing.T) {
	c := baseConfig(4000)
	c.Layer = simulate.LayerRework
	base, _ := simulate.RunIntegrated(model.Activities, c)
	c.ReworkScale = 1.5
	high, _ := simulate.RunIntegrated(model.Activities, c)
	c.ReworkScale = 0.5
	low, _ := simulate.RunIntegrated(model.Activities, c)
	if !(low.ReworkDays < base.ReworkDays && base.ReworkDays < high.ReworkDays) {
		t.Errorf("hari rework %v / %v / %v tidak naik bersama skala", low.ReworkDays, base.ReworkDays, high.ReworkDays)
	}
	for _, l := range model.ReworkLoops {
		p := math.Min(0.95, l.FailProb*1.5)
		if want := p / (1 - p); math.Abs(high.ReworkCycles[l.ID]-want) > 0.08 {
			t.Errorf("%s: rerata putaran %v pada skala 1,5, mau %v", l.ID, high.ReworkCycles[l.ID], want)
		}
	}
}
