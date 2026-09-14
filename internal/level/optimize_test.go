package level_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// assertFeasible memeriksa sifat yang wajib dimiliki jadwal levelling mana
// pun: kapasitas dihormati, relasi FS dihormati, dan isi pekerjaan tetap.
func assertFeasible(t *testing.T, acts []model.Activity, r level.Result, label string) {
	t.Helper()
	for _, role := range r.Roles {
		var used, want float64
		for k := 0; k < r.Duration && k < len(r.Usage[role]); k++ {
			if r.Usage[role][k] > r.Cap[role][k]+1e-6 {
				t.Fatalf("%s: %s hari %d memakai %.3f dari kapasitas %.3f", label, role, k, r.Usage[role][k], r.Cap[role][k])
			}
			used += r.Usage[role][k]
		}
		for _, a := range acts {
			for _, s := range a.Team {
				if s.Role == role {
					want += s.Alloc * float64(a.Duration)
				}
			}
		}
		if math.Abs(used-want) > 1e-6 {
			t.Errorf("%s: %s memakai %.3f hari-orang, mau %.3f", label, role, used, want)
		}
	}
	for _, a := range acts {
		task := r.Tasks[a.ID]
		for _, p := range a.Pred {
			if task.Start < r.Tasks[p.ID].Finish {
				t.Errorf("%s: %s mulai %d sebelum %s selesai %d", label, a.ID, task.Start, p.ID, r.Tasks[p.ID].Finish)
			}
		}
	}
}

func TestEveryRuleGivesAFeasibleSchedule(t *testing.T) {
	c := cal(t)
	for _, rule := range level.Rules {
		r, err := level.Run(model.Activities, level.Options{Calendar: c, Capacity: model.Capacity, UseWindows: true, Rule: rule})
		if err != nil {
			t.Fatalf("%s: %v", rule, err)
		}
		assertFeasible(t, model.Activities, r, string(rule))
	}
}

// Dua aktivitas paralel yang berebut satu orang: daftar menentukan siapa lebih
// dulu, dan aktivitas di luar daftar menyusul di belakang.
func TestOrderDecidesWhoGoesFirst(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", Duration: 3, Team: be(1)},
		{ID: "B", Duration: 2, Team: be(1)},
	}
	opts := level.Options{Calendar: cal(t), Capacity: map[model.Role]float64{model.RoleBE: 1}, Order: []string{"B"}}
	r, err := level.Run(acts, opts)
	if err != nil {
		t.Fatal(err)
	}
	if r.Tasks["B"].Start != 0 || r.Tasks["A"].Start != 2 {
		t.Errorf("urutan [B]: B mulai %d, A mulai %d; mau 0 dan 2", r.Tasks["B"].Start, r.Tasks["A"].Start)
	}
	opts.Order = []string{"A", "B"}
	r, _ = level.Run(acts, opts)
	if r.Tasks["A"].Start != 0 || r.Tasks["B"].Start != 3 {
		t.Errorf("urutan [A B]: A mulai %d, B mulai %d; mau 0 dan 3", r.Tasks["A"].Start, r.Tasks["B"].Start)
	}
}

// Aturan SPT mendahulukan durasi terpendek; MTS mendahulukan yang punya
// penerus terbanyak - dua kunci yang bertolak belakang pada jaringan ini.
func TestSPTAndMTSRulesPickDifferently(t *testing.T) {
	acts := []model.Activity{
		{ID: "L", Duration: 4, Team: be(1)},
		{ID: "S", Duration: 1, Team: be(1)},
		{ID: "N1", Duration: 1, Pred: []model.Predecessor{model.FS("L")}},
		{ID: "N2", Duration: 1, Pred: []model.Predecessor{model.FS("N1")}},
	}
	caps := map[model.Role]float64{model.RoleBE: 1}
	spt, err := level.Run(acts, level.Options{Calendar: cal(t), Capacity: caps, Rule: level.RuleSPT})
	if err != nil {
		t.Fatal(err)
	}
	mts, err := level.Run(acts, level.Options{Calendar: cal(t), Capacity: caps, Rule: level.RuleMTS})
	if err != nil {
		t.Fatal(err)
	}
	if spt.Tasks["S"].Start != 0 {
		t.Errorf("SPT harus mendahulukan S (durasi 1), S mulai %d", spt.Tasks["S"].Start)
	}
	if mts.Tasks["L"].Start != 0 {
		t.Errorf("MTS harus mendahulukan L (dua penerus), L mulai %d", mts.Tasks["L"].Start)
	}
	for _, rule := range []level.Rule{level.RuleLFT, level.RuleMSLK, level.RuleGRPW} {
		if _, err := level.Run(acts, level.Options{Calendar: cal(t), Capacity: caps, Rule: rule}); err != nil {
			t.Errorf("%s: %v", rule, err)
		}
	}
}

func TestReleaseDateHoldsWorkBack(t *testing.T) {
	acts := []model.Activity{{ID: "A", Duration: 2, Team: be(1)}}
	r, err := level.Run(acts, level.Options{
		Calendar: cal(t), Capacity: map[model.Role]float64{model.RoleBE: 1},
		ReleaseOf: func(model.Activity) int { return 7 },
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Tasks["A"].Start != 7 || r.Duration != 9 {
		t.Errorf("rilis hari 7: mulai %d, durasi %d; mau 7 dan 9", r.Tasks["A"].Start, r.Duration)
	}
}

func TestCapacityGridWithOverridesExamFactor(t *testing.T) {
	c := cal(t)
	iso := "2026-01-21"
	day := c.IndexOf(iso)
	base := level.CapacityGrid(c, model.Capacity, true, 200)
	over := level.CapacityGridWith(c, model.Capacity, true, 200, 0.9)
	if math.Abs(base[model.RoleBE][day]-model.ExamCapacityFactor) > 1e-9 {
		t.Errorf("grid bawaan saat UAS = %v, mau %v", base[model.RoleBE][day], model.ExamCapacityFactor)
	}
	if math.Abs(over[model.RoleBE][day]-0.9) > 1e-9 {
		t.Errorf("grid dengan faktor pengganti = %v, mau 0,9", over[model.RoleBE][day])
	}
}

// Kasus terkecil yang bisa dibuktikan dengan tangan: dua pekerjaan 3 hari
// untuk satu orang tidak mungkin selesai sebelum hari ke-6.
func TestOptimizeProvesATinyInstance(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", Duration: 3, Team: be(1)},
		{ID: "B", Duration: 3, Team: be(1)},
	}
	o, err := level.Optimize(acts, level.OptimizeOptions{Options: level.Options{Calendar: cal(t), Capacity: map[model.Role]float64{model.RoleBE: 1}}, Samples: 10})
	if err != nil {
		t.Fatal(err)
	}
	if o.Best.Duration != 6 || o.Bound.Value != 6 || !o.Proven || o.Gap != 0 {
		t.Errorf("terbaik %d, batas bawah %d, terbukti %v; mau 6, 6, true", o.Best.Duration, o.Bound.Value, o.Proven)
	}
	if len(o.Rules) != len(level.Rules) || o.Samples != 10 {
		t.Errorf("laporan pencarian tidak lengkap: %d aturan, %d sampel", len(o.Rules), o.Samples)
	}
}

// TestRealProjectIsProvenOptimal mengunci klaim utama halaman Optimasi.
func TestRealProjectIsProvenOptimal(t *testing.T) {
	c := cal(t)
	for _, tc := range []struct {
		windows bool
		want    int
	}{{false, 100}, {true, 113}} {
		o, err := level.Optimize(model.Activities, level.OptimizeOptions{Options: level.Options{Calendar: c, Capacity: model.Capacity, UseWindows: tc.windows}})
		if err != nil {
			t.Fatal(err)
		}
		if o.Best.Duration != tc.want || !o.Proven {
			t.Errorf("jendela %v: terbaik %d (batas bawah %d, celah %d), mau %d terbukti optimal", tc.windows, o.Best.Duration, o.Bound.Value, o.Gap, tc.want)
		}
		b := o.Bound
		if !(b.CPM <= b.Solo && b.Solo <= b.Energetic && b.Energetic <= b.Value) {
			t.Errorf("lapis batas bawah tidak berurutan: %+v", b)
		}
		if o.Baseline.Duration < o.Best.Duration {
			t.Errorf("SGS LST %d lebih pendek dari jadwal terbaik %d", o.Baseline.Duration, o.Best.Duration)
		}
		assertFeasible(t, model.Activities, o.Best, "optimize")
		if len(o.Best.OnDay) == 0 {
			t.Error("jadwal terbaik harus dibangun dalam mode lengkap untuk Gantt")
		}
	}
}

// TestLowerBoundIsSoundAgainstBruteForce adalah uji kejujuran batas bawah:
// pada jaringan kecil, SEMUA urutan aktivitas dicoba, dan tidak satu pun
// jadwal boleh lebih pendek dari batas bawah. Batas bawah yang terlalu tinggi
// akan membuat klaim "terbukti optimal" palsu.
func TestLowerBoundIsSoundAgainstBruteForce(t *testing.T) {
	c := cal(t)
	caps := map[model.Role]float64{model.RoleBE: 1, model.RoleFE: 1, model.RoleOPS: 0.5}
	instances := [][]model.Activity{
		{
			{ID: "A", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
			{ID: "B", Duration: 2, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 0.5}, {Role: model.RoleFE, Alloc: 1}}},
			{ID: "C", Duration: 4, Team: []model.TeamSlot{{Role: model.RoleFE, Alloc: 1}}, Pred: []model.Predecessor{model.FS("A")}},
			{ID: "D", Duration: 2, Team: []model.TeamSlot{{Role: model.RoleOPS, Alloc: 1}}, Pred: []model.Predecessor{model.FS("B")}},
			{ID: "E", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}, Pred: []model.Predecessor{model.FS("B")}},
		},
		{
			{ID: "P", Duration: 4, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
			{ID: "Q", Duration: 4, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
			{ID: "R", Duration: 2, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 0.6}, {Role: model.RoleOPS, Alloc: 0.4}}, Pred: []model.Predecessor{model.FS("P")}},
			{ID: "S", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleOPS, Alloc: 1}}},
			{ID: "T", Duration: 1, Pred: []model.Predecessor{model.FS("Q"), model.FS("R"), model.FS("S")}},
		},
	}
	for n, acts := range instances {
		for _, windows := range []bool{false, true} {
			opts := level.Options{Calendar: c, Capacity: caps, UseWindows: windows}
			b, err := level.LowerBound(acts, opts)
			if err != nil {
				t.Fatal(err)
			}
			best := math.MaxInt32
			ids := make([]string, len(acts))
			for i, a := range acts {
				ids[i] = a.ID
			}
			permute(ids, 0, func(order []string) {
				o := opts
				o.Order = append([]string(nil), order...)
				r, err := level.Run(acts, o)
				if err != nil {
					t.Fatal(err)
				}
				if r.Duration < best {
					best = r.Duration
				}
			})
			if b.Value > best {
				t.Errorf("instans %d (jendela %v): batas bawah %d melebihi jadwal terpendek hasil brute force %d", n, windows, b.Value, best)
			}
			o, err := level.Optimize(acts, level.OptimizeOptions{Options: opts, Samples: 40})
			if err != nil {
				t.Fatal(err)
			}
			if o.Best.Duration != best {
				t.Errorf("instans %d (jendela %v): Optimize %d, brute force %d", n, windows, o.Best.Duration, best)
			}
		}
	}
}

func permute(a []string, k int, visit func([]string)) {
	if k == len(a) {
		visit(a)
		return
	}
	for i := k; i < len(a); i++ {
		a[k], a[i] = a[i], a[k]
		permute(a, k+1, visit)
		a[k], a[i] = a[i], a[k]
	}
}

func TestJustifyNeverWorsens(t *testing.T) {
	c := cal(t)
	opts := level.Options{Calendar: c, Capacity: model.Capacity, UseWindows: true, Lite: true}
	for _, rule := range level.Rules {
		o := opts
		o.Rule = rule
		r, err := level.Run(model.Activities, o)
		if err != nil {
			t.Fatal(err)
		}
		if j := level.Justify(model.Activities, opts, r, 5); j.Duration > r.Duration {
			t.Errorf("%s: justifikasi memperburuk %d -> %d", rule, r.Duration, j.Duration)
		}
	}
	// Dengan tanggal rilis, justifikasi tidak berlaku dan jadwal dikembalikan apa adanya.
	o := opts
	o.ReleaseOf = func(model.Activity) int { return 0 }
	r, _ := level.Run(model.Activities, o)
	if j := level.Justify(model.Activities, o, r, 5); j.Duration != r.Duration {
		t.Error("justifikasi dengan tanggal rilis harus mengembalikan jadwal semula")
	}
}

func TestOptimizeAndBoundRejectMissingInputs(t *testing.T) {
	if _, err := level.Optimize(model.Activities, level.OptimizeOptions{}); err == nil {
		t.Error("Optimize tanpa kalender seharusnya galat")
	}
	if _, err := level.LowerBound(model.Activities, level.Options{Capacity: model.Capacity}); err == nil {
		t.Error("LowerBound tanpa kalender seharusnya galat")
	}
	if _, err := level.LowerBound(model.Activities, level.Options{Calendar: workcal.MustNew(model.ProjectCharter.StartDate, 50)}); err == nil {
		t.Error("LowerBound tanpa kapasitas seharusnya galat")
	}
}

// TestLowerBoundIsSoundWhenCapacityExceedsBase: lembur dan orang baru membuat
// grid kapasitas harian MELEBIHI kapasitas dasar. Penalaran yang tidak
// bertanggal (durasi solo dan energi keturunan) harus memakai kapasitas harian
// tertinggi, bukan kapasitas dasar - kalau tidak, batas bawah melampaui jadwal
// yang benar-benar bisa dibuat.
func TestLowerBoundIsSoundWhenCapacityExceedsBase(t *testing.T) {
	c := cal(t)
	caps := map[model.Role]float64{model.RoleBE: 1, model.RoleFE: 1, model.RoleOPS: 0.5}
	fe := []model.TeamSlot{{Role: model.RoleFE, Alloc: 1}}
	beam := []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}
	instances := [][]model.Activity{
		{
			{ID: "X", Duration: 1, Team: fe},
			{ID: "Y", Duration: 4, Team: beam, Pred: []model.Predecessor{model.FS("X")}},
			{ID: "Z", Duration: 4, Team: beam, Pred: []model.Predecessor{model.FS("X")}},
		},
		{
			{ID: "K1", Duration: 4, Team: beam},
			{ID: "K2", Duration: 5, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}, {Role: model.RoleFE, Alloc: 0.5}}, Pred: []model.Predecessor{model.FS("K1")}},
			{ID: "K3", Duration: 3, Team: fe},
			{ID: "K4", Duration: 3, Team: beam, Pred: []model.Predecessor{model.FS("K2"), model.FS("K3")}},
		},
		{
			{ID: "D", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleOPS, Alloc: 1}}},
			{ID: "E", Duration: 2, Team: beam, Pred: []model.Predecessor{model.FS("D")}},
			{ID: "F", Duration: 3, Team: beam},
			{ID: "G", Duration: 1, Team: fe, Pred: []model.Predecessor{model.FS("E"), model.FS("F")}},
		},
	}
	for n, acts := range instances {
		grid := level.CapacityGrid(c, caps, false, 60)
		rateCap := map[model.Role][]float64{model.RoleBE: make([]float64, 60)}
		for d := range grid[model.RoleBE] {
			grid[model.RoleBE][d] += 0.45
			grid[model.RoleOPS][d] += 0.5
			if d%3 != 2 { // lembur dua dari tiga hari, supaya laju berubah per hari
				rateCap[model.RoleBE][d] = 1.45
			}
		}
		for _, rc := range []map[model.Role][]float64{nil, rateCap} {
			opts := level.Options{Calendar: c, Capacity: caps, CapGrid: grid, Horizon: 60, RateCap: rc}
			b, err := level.LowerBound(acts, opts)
			if err != nil {
				t.Fatal(err)
			}
			best := math.MaxInt32
			ids := make([]string, len(acts))
			for i, a := range acts {
				ids[i] = a.ID
			}
			permute(ids, 0, func(order []string) {
				o := opts
				o.Order = append([]string(nil), order...)
				r, err := level.Run(acts, o)
				if err != nil {
					t.Fatal(err)
				}
				if r.Duration < best {
					best = r.Duration
				}
			})
			if b.Value > best {
				t.Errorf("instans %d (laju lembur %v): batas bawah %d (%+v) melebihi jadwal terpendek %d pada grid di atas kapasitas dasar", n, rc != nil, b.Value, b, best)
			}
		}
	}
}

// TestRateCapLetsOvertimeSpeedOneTask: lembur mempercepat satu pekerjaan tak
// terbagi, sedangkan kapasitas tambahan tanpa RateCap (orang baru) tidak.
func TestRateCapLetsOvertimeSpeedOneTask(t *testing.T) {
	c := cal(t)
	acts := []model.Activity{{ID: "A", Duration: 6, Team: be(1)}}
	caps := map[model.Role]float64{model.RoleBE: 1}
	grid := level.CapacityGrid(c, caps, false, 20)
	rc := map[model.Role][]float64{model.RoleBE: make([]float64, 20)}
	for d := range grid[model.RoleBE] {
		grid[model.RoleBE][d] = 2 // orang kedua
		rc[model.RoleBE][d] = 1.45
	}
	hire, err := level.Run(acts, level.Options{Calendar: c, Capacity: caps, CapGrid: grid, Horizon: 20})
	if err != nil {
		t.Fatal(err)
	}
	ot, err := level.Run(acts, level.Options{Calendar: c, Capacity: caps, CapGrid: grid, Horizon: 20, RateCap: rc})
	if err != nil {
		t.Fatal(err)
	}
	// 6 hari kerja pada 1,45 hari per hari selesai pada hari kelima (6 / 1,45 = 4,14).
	if hire.Duration != 6 || ot.Duration != 5 {
		t.Errorf("orang kedua %d hari, lembur %d hari; mau 6 dan 5", hire.Duration, ot.Duration)
	}
	if math.Abs(ot.Usage[model.RoleBE][0]-1.45) > 1e-9 {
		t.Errorf("pemakaian hari pertama %v, mau 1,45", ot.Usage[model.RoleBE][0])
	}
	b, err := level.LowerBound(acts, level.Options{Calendar: c, Capacity: caps, CapGrid: grid, Horizon: 20, RateCap: rc})
	if err != nil {
		t.Fatal(err)
	}
	if b.Value != 5 {
		t.Errorf("batas bawah dengan lembur %d, mau 5", b.Value)
	}
	mixed := model.Activity{ID: "M", Duration: 2, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}, {Role: model.RoleFE, Alloc: 0.5}}}
	if v := level.RateLimit(mixed, 0, rc); v != 1 {
		t.Errorf("tim campuran dibatasi peran tanpa lembur: %v, mau 1", v)
	}
	if v := level.RateLimit(acts[0], 0, rc); v != 1.45 {
		t.Errorf("laju BE saat lembur %v, mau 1,45", v)
	}
	if v := level.RateLimit(acts[0], 99, rc); v != 1 || level.RateLimit(acts[0], 0, nil) != 1 {
		t.Error("di luar grid atau tanpa RateCap lajunya 1")
	}
}

// TestExcessCountsSpeedBeyondAllocation: pekerjaan berkapasitas setengah orang
// yang dipercepat lembur memakai jam di atas alokasinya walau pemakaian peran
// masih di bawah kapasitas normal - dan jam itu harus tercatat.
func TestExcessCountsSpeedBeyondAllocation(t *testing.T) {
	c := cal(t)
	caps := map[model.Role]float64{model.RoleBE: 1}
	acts := []model.Activity{{ID: "H", Duration: 4, Team: be(0.5)}}
	grid := level.CapacityGrid(c, caps, false, 20)
	rc := map[model.Role][]float64{model.RoleBE: make([]float64, 20)}
	for d := range grid[model.RoleBE] {
		grid[model.RoleBE][d] = 1.45
		rc[model.RoleBE][d] = 1.45
	}
	r, err := level.Run(acts, level.Options{Calendar: c, Capacity: caps, CapGrid: grid, RateCap: rc, Horizon: 20})
	if err != nil {
		t.Fatal(err)
	}
	if r.Duration != 3 || math.Abs(r.Usage[model.RoleBE][0]-0.725) > 1e-9 || math.Abs(r.Excess[model.RoleBE][0]-0.225) > 1e-9 {
		t.Errorf("durasi %d, pemakaian %v, kelebihan %v; mau 3, 0,725, 0,225", r.Duration, r.Usage[model.RoleBE][0], r.Excess[model.RoleBE][0])
	}
	plain, _ := level.Run(acts, level.Options{Calendar: c, Capacity: caps, Horizon: 20})
	for _, v := range plain.Excess[model.RoleBE] {
		if v != 0 {
			t.Fatal("tanpa RateCap tidak boleh ada kelebihan di atas alokasi")
		}
	}
}
