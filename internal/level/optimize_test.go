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
