package simulate_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/evm"
	"github.com/xyb3rpunq/mppl-control-tower/internal/gert"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

func TestPhiKnownTables(t *testing.T) {
	x := []bool{true, true, false, false}
	if phi, ok := simulate.Phi(x, x); !ok || math.Abs(phi-1) > 1e-12 {
		t.Errorf("phi deret dengan dirinya = %v, mau 1", phi)
	}
	y := []bool{false, false, true, true}
	if phi, _ := simulate.Phi(x, y); math.Abs(phi+1) > 1e-12 {
		t.Errorf("phi deret berlawanan = %v, mau -1", phi)
	}
	if _, ok := simulate.Phi(x, []bool{true, true, true, true}); ok {
		t.Error("deret tanpa variasi harus dilaporkan tidak terdefinisi")
	}
}

// TestRiskCopulaPreservesMarginalsButClusters adalah jaminan utama kopula
// risiko: peluang tiap risiko dan rerata biaya tetap, ekor dan gerombolnya berubah.
func TestRiskCopulaPreservesMarginalsButClusters(t *testing.T) {
	run := func(lam float64) simulate.IntegratedResult {
		c := baseConfig(8000)
		c.Layer, c.RiskLoading = simulate.LayerRisks, lam
		r, err := simulate.RunIntegrated(model.Activities, c)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	ind, mid, high := run(0), run(0.3), run(0.9)
	for _, r := range []simulate.IntegratedResult{ind, high} {
		for _, rk := range model.Risks {
			got := float64(r.RiskHits[rk.ID]) / float64(r.Config.Iterations)
			if math.Abs(got-rk.ResidualProb) > 0.03 {
				t.Errorf("lambda %v: %s terjadi %.3f, mau sekitar %.2f", r.Config.RiskLoading, rk.ID, got, rk.ResidualProb)
			}
		}
	}
	if math.Abs(ind.RiskPhi) > 0.03 {
		t.Errorf("phi pada lambda 0 = %v, seharusnya mendekati nol", ind.RiskPhi)
	}
	if !(high.RiskPhi > mid.RiskPhi && mid.RiskPhi > ind.RiskPhi) {
		t.Errorf("phi harus naik bersama lambda: %v, %v, %v", ind.RiskPhi, mid.RiskPhi, high.RiskPhi)
	}
	if math.Abs(high.CostMean-ind.CostMean)/ind.CostMean > 0.01 {
		t.Errorf("rerata biaya bergeser lebih dari 1%%: %v vs %v", ind.CostMean, high.CostMean)
	}
	if high.CostP95 <= ind.CostP95 {
		t.Errorf("risiko bergerombol seharusnya menebalkan P95 biaya: %v vs %v", ind.CostP95, high.CostP95)
	}
	var total float64
	for _, v := range high.RiskCount {
		total += v
	}
	if math.Abs(total-1) > 1e-9 {
		t.Errorf("sebaran jumlah risiko menjumlah %v, mau 1", total)
	}
}

// TestSamplerFactorDrivesActivities memastikan faktor laten yang dipakai
// kopula risiko benar-benar faktor yang menggerakkan durasi aktivitas.
func TestSamplerFactorDrivesActivities(t *testing.T) {
	acts := []model.Activity{{ID: "X", Duration: 5, Optimistic: 3, Pessimistic: 9, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}}}
	s := simulate.NewSampler(acts)
	rng := simulate.NewPRNG(11)
	out := make([]int, 1)
	for i := 0; i < 50; i++ {
		s.Draw(rng, 0.999, "pert", out)
		want := 0.5 * (1 + math.Erf(0.999*s.Factor(model.RoleBE)/math.Sqrt2))
		if math.Abs(s.U(0)-want) > 0.05 {
			t.Fatalf("u aktivitas %v tidak mengikuti faktor peran (mau sekitar %v)", s.U(0), want)
		}
	}
	if s.Factor(model.RoleUX) != 0 {
		t.Error("faktor peran yang tidak dikenal harus nol")
	}
}

func TestSamplerQuantileCDFAndMean(t *testing.T) {
	acts := []model.Activity{
		{ID: "V", Duration: 6, Optimistic: 4, Pessimistic: 11},
		{ID: "F", Duration: 3, Optimistic: 3, Pessimistic: 3},
		{ID: "M", Milestone: true},
	}
	s := simulate.NewSampler(acts)
	for _, u := range []float64{0.1, 0.5, 0.9} {
		x := s.Value(0, u, "pert")
		if got := s.CDF(0, x); math.Abs(got-u) > 2e-3 {
			t.Errorf("CDF(Q(%v)) = %v", u, got)
		}
	}
	if s.Value(1, 0.7, "pert") != 3 || s.Value(2, 0.7, "pert") != 0 {
		t.Error("aktivitas tetap dan milestone harus mengembalikan durasi tetapnya")
	}
	if s.CDF(1, 2.9) != 0 || s.CDF(1, 3) != 1 {
		t.Error("CDF aktivitas tetap harus berupa tangga di durasinya")
	}
	if tri := s.Value(0, 0.5, "triangular"); tri < 4 || tri > 11 {
		t.Errorf("kuantil segitiga %v keluar rentang", tri)
	}
	if got := s.PERTMean(0); math.Abs(got-(4.0+24+11)/6) > 1e-12 {
		t.Errorf("rerata PERT = %v", got)
	}
	if s.PERTMean(1) != 3 || s.PERTMean(2) != 0 {
		t.Error("rerata PERT aktivitas tetap/milestone salah")
	}
}

// TestReworkLayerMatchesGERT: rerata putaran yang disimulasikan harus sesuai
// bentuk tertutup p/(1-p).
func TestReworkLayerMatchesGERT(t *testing.T) {
	c := baseConfig(20000)
	c.Layer = simulate.LayerRework
	r, err := simulate.RunIntegrated(model.Activities, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range model.ReworkLoops {
		want := gert.ExpectedCycles(l.FailProb)
		if got := r.ReworkCycles[l.ID]; math.Abs(got-want) > 0.03 {
			t.Errorf("%s: rerata putaran %v, mau sekitar %v", l.ID, got, want)
		}
	}
	if r.ReworkCost <= 0 {
		t.Error("rework harus menambah biaya")
	}
}

// TestTimeCostMatchesPlanAtNominal: biaya sewa pada jadwal rencana harus
// persis sama dengan nilainya di anggaran.
func TestTimeCostMatchesPlanAtNominal(t *testing.T) {
	c := baseConfig(2000)
	c.Layer = simulate.LayerResources
	r, err := simulate.RunIntegrated(model.Activities, c)
	if err != nil {
		t.Fatal(err)
	}
	var want float64
	for _, a := range model.Activities {
		for _, e := range a.Extras {
			if e.TimeBased {
				want += e.Amount
			}
		}
	}
	if math.Abs(r.TimeCostPlan-want) > 1e-6 {
		t.Errorf("biaya sewa rencana %v, mau %v", r.TimeCostPlan, want)
	}
	if r.TimeCost <= want {
		t.Errorf("jadwal berkapasitas selalu lebih panjang, jadi rerata sewa %v harus di atas rencana %v", r.TimeCost, want)
	}
}

func inflight(t *testing.T, priorWeight float64) (*simulate.InFlight, *workcal.Calendar, *evm.Engine) {
	t.Helper()
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 400)
	plan := schedule.MustCompute(model.Activities, schedule.Options{})
	eng := evm.New(model.Activities, plan, model.RateCard)
	fl, err := simulate.PrepareInFlight(model.Activities, cal, cal.FractionalIndexOf(model.DefaultStatusDate), priorWeight, eng.ACAt)
	if err != nil {
		t.Fatal(err)
	}
	return fl, cal, eng
}

func TestInFlightStateAtTheDataDate(t *testing.T) {
	fl, cal, eng := inflight(t, 0)
	sd := cal.FractionalIndexOf(model.DefaultStatusDate)
	if math.Abs(fl.ACToDate-eng.ACAt(sd)) > 1e-6 {
		t.Errorf("AC prakiraan %v berbeda dari mesin EVM %v", fl.ACToDate, eng.ACAt(sd))
	}
	if len(fl.Completed)+len(fl.InProgress)+len(fl.NotStarted) != len(model.Activities) {
		t.Error("setiap aktivitas harus punya tepat satu status")
	}
	want := map[string]bool{"A17": true, "A19": true, "A21": true}
	for _, id := range fl.InProgress {
		if !want[id] {
			t.Errorf("%s tidak seharusnya sedang berjalan", id)
		}
	}
	if len(fl.InProgress) != 3 {
		t.Errorf("sedang berjalan %v, mau A17, A19, A21", fl.InProgress)
	}
	if fl.Evidence <= 0 || math.Abs(fl.Credibility-float64(fl.Evidence)/(float64(fl.Evidence)+simulate.DefaultPriorWeight)) > 1e-12 {
		t.Errorf("kredibilitas %v tidak sesuai n/(n+k) dengan n = %d", fl.Credibility, fl.Evidence)
	}
	if want := fl.Credibility*fl.ObservedRatio + 1 - fl.Credibility; math.Abs(fl.DurationFactor-want) > 1e-12 {
		t.Error("faktor durasi tidak sesuai rumus kredibilitas")
	}
	if fl.ExamEvidence == 0 {
		t.Error("UTS resmi sudah lewat pada tanggal data; harus ada bukti kalibrasi ujian")
	}
	lo, hi := model.ExamCapacityFactor, 1.0
	if fl.ExamFactor < lo-1e-9 || fl.ExamFactor > hi+1e-9 {
		t.Errorf("faktor ujian terkalibrasi %v di luar [%v, %v]", fl.ExamFactor, lo, hi)
	}
	for _, r := range model.Risks {
		closed := r.Status == "terjadi" || r.Status == "tertutup"
		if fl.ClosedRisk[r.ID] != closed {
			t.Errorf("%s berstatus %q, ditutup = %v", r.ID, r.Status, fl.ClosedRisk[r.ID])
		}
		if !closed && fl.State[fl.Exposed[r.ID]].Kind == simulate.Completed {
			t.Errorf("%s masih terbuka tetapi dampaknya dibebankan ke aktivitas yang sudah selesai", r.ID)
		}
	}
}

func TestInFlightForecastRespectsActuals(t *testing.T) {
	fl, cal, _ := inflight(t, 0)
	c := baseConfig(1500)
	c.Layer, c.InFlight, c.ExamFactor, c.Calendar = simulate.LayerResources, fl, fl.ExamFactor, cal
	r, err := simulate.RunIntegrated(model.Activities, c)
	if err != nil {
		t.Fatal(err)
	}
	maxDone := 0
	for i := range model.Activities {
		if fl.State[i].Kind == simulate.Completed && fl.State[i].Finish > maxDone {
			maxDone = fl.State[i].Finish
		}
	}
	if r.Durations[0] < float64(fl.Now) || r.Durations[0] < float64(maxDone) {
		t.Errorf("prakiraan tercepat %v sebelum tanggal data %d atau realisasi %d", r.Durations[0], fl.Now, maxDone)
	}
	if r.Costs[0] < fl.ACToDate {
		t.Errorf("biaya akhir %v lebih kecil dari biaya yang sudah keluar %v", r.Costs[0], fl.ACToDate)
	}
	for _, id := range []string{"R02", "R08"} {
		if r.RiskHits[id] != 0 {
			t.Errorf("%s sudah terjadi pada register, tidak boleh disampel lagi", id)
		}
	}

	prior, _, _ := inflight(t, 1e12)
	if math.Abs(prior.DurationFactor-1) > 1e-6 || math.Abs(prior.CostFactor-1) > 1e-6 {
		t.Errorf("bobot awal sangat besar harus mengembalikan faktor ke 1, dapat %v dan %v", prior.DurationFactor, prior.CostFactor)
	}

	bad := *fl
	bad.State = bad.State[:3]
	c.InFlight = &bad
	if _, err := simulate.RunIntegrated(model.Activities, c); err == nil {
		t.Error("status tanggal data untuk jaringan lain seharusnya galat")
	}
	if _, err := simulate.PrepareInFlight(model.Activities, nil, 10, 0, nil); err == nil {
		t.Error("tanpa kalender seharusnya galat")
	}
}

// TestAuditAndLevelOrder: urutan jadwal optimal tidak pernah membuat levelling
// per iterasi lebih panjang, dan audit mengukur jaraknya dari optimum.
func TestAuditAndLevelOrder(t *testing.T) {
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 400)
	opt, err := level.Optimize(model.Activities, level.OptimizeOptions{Options: level.Options{Calendar: cal, Capacity: model.Capacity, UseWindows: true}, Samples: 60})
	if err != nil {
		t.Fatal(err)
	}
	c := baseConfig(400)
	c.Layer = simulate.LayerResources
	plain, err := simulate.RunIntegrated(model.Activities, c)
	if err != nil {
		t.Fatal(err)
	}
	c.LevelOrder, c.AuditEvery, c.AuditSamples = opt.Best.Order, 100, 20
	ordered, err := simulate.RunIntegrated(model.Activities, c)
	if err != nil {
		t.Fatal(err)
	}
	if ordered.DurMean > plain.DurMean+1e-9 {
		t.Errorf("urutan optimal memperpanjang rerata durasi: %v -> %v", plain.DurMean, ordered.DurMean)
	}
	a := ordered.Audit
	if a.Audited != 4 {
		t.Errorf("teraudit %d iterasi, mau 4", a.Audited)
	}
	if a.SGSGapMean < 0 || a.BoundGapMean < 0 || a.ProvenShare < 0 || a.ProvenShare > 1 || a.SGSOptimalShare > 1 {
		t.Errorf("hasil audit tidak masuk akal: %+v", a)
	}
	if plain.Audit.Audited != 0 {
		t.Error("audit tidak boleh berjalan bila AuditEvery nol")
	}
}
