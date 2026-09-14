package compress_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/cost"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

func calendar() *workcal.Calendar { return workcal.MustNew(model.ProjectCharter.StartDate, 400) }

// overtimeCost menghitung ulang upah lembur dari pemakaian kapasitas, lepas
// dari kode yang diuji: satu orang per peran, jam pertama 1,5x, berikutnya 2x.
func overtimeCost(r level.Result, base map[model.Role][]float64, roles []model.Role) float64 {
	var c float64
	for _, role := range roles {
		for d := 0; d < r.Duration; d++ {
			over := math.Max(r.Usage[role][d]-base[role][d], r.Excess[role][d])
			if over <= 1e-9 {
				continue
			}
			h := over * 8
			units := 1.5 * math.Min(h, 1)
			if h > 1 {
				units += 2 * (h - 1)
			}
			c += units * model.RateCard[role] / 8
		}
	}
	return c
}

// TestLevelledOvertimeMatchesBruteForce: pada jaringan kecil setiap himpunan
// hari lembur dan setiap urutan aktivitas dicoba. Durasi minimum harus sama,
// dan biaya yang dilaporkan tidak boleh di bawah minimum brute force (itu
// berarti biaya yang mustahil) - pada kasus sekecil ini harus sama persis.
func TestLevelledOvertimeMatchesBruteForce(t *testing.T) {
	be := []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}
	fe := []model.TeamSlot{{Role: model.RoleFE, Alloc: 1}}
	acts := []model.Activity{
		{ID: "A", Duration: 3, Optimistic: 2, Pessimistic: 4, Team: be},
		{ID: "B", Duration: 3, Optimistic: 2, Pessimistic: 4, Team: be},
		{ID: "C", Duration: 2, Optimistic: 1, Pessimistic: 3, Team: fe, Pred: []model.Predecessor{model.FS("A")}},
	}
	caps := map[model.Role]float64{model.RoleBE: 1, model.RoleFE: 1, model.RoleOPS: 0.5}
	opts := level.Options{Calendar: calendar(), Capacity: caps, Horizon: 12}
	oc, err := compress.LevelledOvertime(acts, level.OptimizeOptions{Options: opts}, model.RateCard, 0)
	if err != nil {
		t.Fatal(err)
	}
	if oc.Levelled != 6 || oc.MinDuration != 5 || !oc.MinProven {
		t.Fatalf("tanpa lembur %d, minimum %d (terbukti %v); mau 6 dan 5 terbukti", oc.Levelled, oc.MinDuration, oc.MinProven)
	}
	if len(oc.Excluded) != 1 || oc.Excluded[0] != model.RoleOPS {
		t.Errorf("peran paruh waktu yang dikecualikan %v, mau [OPS]", oc.Excluded)
	}

	base := level.CapacityGrid(opts.Calendar, caps, false, 12)
	roles := []model.Role{model.RoleBE, model.RoleFE}
	extra := compress.SustainedOvertimeHours() / 8
	orders := [][]string{{"A", "B", "C"}, {"A", "C", "B"}, {"B", "A", "C"}, {"B", "C", "A"}, {"C", "A", "B"}, {"C", "B", "A"}}
	best := map[int]float64{}
	const days = 6
	for mask := 0; mask < 1<<(2*days); mask++ {
		grid := map[model.Role][]float64{}
		for r, row := range base {
			grid[r] = append([]float64(nil), row...)
		}
		rc := map[model.Role][]float64{}
		for i, role := range roles {
			rc[role] = make([]float64, len(grid[role]))
			for d := 0; d < days; d++ {
				if mask&(1<<(i*days+d)) != 0 {
					grid[role][d] += extra
					rc[role][d] = 1 + extra
				}
			}
		}
		for _, order := range orders {
			o := opts
			o.Lite, o.CapGrid, o.RateCap, o.Order = true, grid, rc, order
			r, err := level.Run(acts, o)
			if err != nil {
				t.Fatal(err)
			}
			c := overtimeCost(r, base, roles)
			for T := r.Duration; T <= oc.Levelled; T++ {
				if v, ok := best[T]; !ok || c < v {
					best[T] = c
				}
			}
		}
	}
	minT := oc.Levelled
	for T := range best {
		if T < minT {
			minT = T
		}
	}
	if minT != oc.MinDuration {
		t.Errorf("durasi minimum brute force %d, dilaporkan %d", minT, oc.MinDuration)
	}
	p, ok := oc.PointAt(5)
	if !ok {
		t.Fatal("tidak ada rencana lembur untuk 5 hari")
	}
	if p.Cost < best[5]-0.5 || math.Abs(p.Cost-best[5]) > 0.5 {
		t.Errorf("biaya 5 hari %v, brute force %v", p.Cost, best[5])
	}
	if math.Abs(p.Cost-overtimeCost(p.Schedule, base, roles)) > 1e-6 {
		t.Errorf("biaya %v tidak cocok dengan pemakaian jadwalnya %v", p.Cost, overtimeCost(p.Schedule, base, roles))
	}
	if _, ok := oc.PointAt(6); ok {
		t.Error("durasi tanpa lembur tidak boleh punya rencana lembur")
	}
}

// TestLevelledOvertimeOnTheProject mengunci klaim halaman Optimasi dan
// memeriksa setiap rencana terhadap aturan lembur dan kalender ujian.
func TestLevelledOvertimeOnTheProject(t *testing.T) {
	c := calendar()
	opts := level.Options{Calendar: c, Capacity: model.Capacity, UseWindows: true}
	oc, err := compress.LevelledOvertime(model.Activities, level.OptimizeOptions{Options: opts}, model.RateCard, 0)
	if err != nil {
		t.Fatal(err)
	}
	if oc.Levelled != 113 || oc.MinDuration != 101 || !oc.MinProven || oc.Bound.Value != 101 {
		t.Fatalf("tanpa lembur %d, minimum %d, batas bawah %d, terbukti %v; mau 113, 101, 101, true", oc.Levelled, oc.MinDuration, oc.Bound.Value, oc.MinProven)
	}
	// Satu durasi bisa terlewati bila rencana termurahnya selesai sehari lebih
	// awal; yang dijamin: durasi turun tegas dan berakhir di durasi minimum.
	if len(oc.Points) == 0 || len(oc.Points) > oc.Levelled-oc.MinDuration || oc.Points[len(oc.Points)-1].Duration != oc.MinDuration {
		t.Fatalf("%d rencana untuk %d hari yang bisa dibeli", len(oc.Points), oc.Levelled-oc.MinDuration)
	}
	H := 300
	win := level.CapacityGridWith(c, model.Capacity, true, H, 0)
	plain := level.CapacityGridWith(c, model.Capacity, false, H, 0)
	rentals, _, err := cost.Rentals(model.Activities)
	if err != nil {
		t.Fatal(err)
	}
	byIdx := model.Activities
	prevCost := 0.0
	for i, p := range oc.Points {
		if (i > 0 && p.Duration >= oc.Points[i-1].Duration) || p.Duration >= oc.Levelled || p.Schedule.Duration != p.Duration {
			t.Errorf("rencana ke-%d berdurasi %d (jadwal %d)", i, p.Duration, p.Schedule.Duration)
		}
		if p.Cost <= prevCost {
			t.Errorf("%d hari: upah %v tidak naik dari %v", p.Duration, p.Cost, prevCost)
		}
		prevCost = p.Cost
		if p.MaxDay > compress.SustainedOvertimeHours()+1e-9 || p.MaxDay > model.OvertimeMaxDaily || p.MaxWeek > model.OvertimeMaxWeekly+1e-9 {
			t.Errorf("%d hari: %v jam sehari, %v jam seminggu melanggar PP 35/2021", p.Duration, p.MaxDay, p.MaxWeek)
		}
		var rental float64
		for _, rt := range rentals {
			rental += rt.Rate * float64(p.Duration-p.Schedule.Tasks[byIdx[rt.Index].ID].Start)
		}
		if math.Abs(rental-p.Rental) > 1e-6 || math.Abs(p.RentalSaved-(oc.BaseRental-p.Rental)) > 1e-6 || math.Abs(p.Net-(p.Cost-p.RentalSaved)) > 1e-6 {
			t.Errorf("%d hari: sewa %v/%v, dihemat %v, bersih %v tidak konsisten", p.Duration, p.Rental, rental, p.RentalSaved, p.Net)
		}
		if got := overtimeCost(p.Schedule, win, oc.Eligible); math.Abs(got-p.Cost) > 1e-6 {
			t.Errorf("%d hari: upah %v, dihitung ulang dari pemakaian %v", p.Duration, p.Cost, got)
		}
		for _, role := range []model.Role{model.RoleOPS} {
			if p.Hours[role] > 0 {
				t.Errorf("%d hari: peran paruh waktu %s diberi lembur", p.Duration, role)
			}
		}
		for role, u := range p.Schedule.Usage {
			for d := 0; d < p.Duration; d++ {
				if win[role][d] < plain[role][d]-1e-9 && u[d] > win[role][d]+1e-9 {
					t.Fatalf("%d hari: %s lembur pada hari ujian %s", p.Duration, role, c.ISOAt(d))
				}
				if u[d] > win[role][d]+compress.SustainedOvertimeHours()/8+1e-9 {
					t.Fatalf("%d hari: %s memakai %v di atas kapasitas + lembur", p.Duration, role, u[d])
				}
			}
		}
	}
	// Jadwal tanpa lembur harus jadwal terbukti optimal yang sama dengan halaman.
	o, err := level.Optimize(model.Activities, level.OptimizeOptions{Options: opts})
	if err != nil {
		t.Fatal(err)
	}
	if o.Best.Duration != oc.Base.Duration {
		t.Errorf("jadwal tanpa lembur %d, Optimize %d", oc.Base.Duration, o.Best.Duration)
	}
}

func TestLevelledOvertimeEdgeCases(t *testing.T) {
	if _, err := compress.LevelledOvertime(model.Activities, level.OptimizeOptions{}, model.RateCard, 0); err == nil {
		t.Error("tanpa kalender dan kapasitas seharusnya galat")
	}
	// Tanpa perebutan sumber daya, lembur tidak memotong apa pun.
	acts := []model.Activity{{ID: "A", Duration: 3, Optimistic: 2, Pessimistic: 4, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}}}
	oc, err := compress.LevelledOvertime(acts, level.OptimizeOptions{Options: level.Options{Calendar: calendar(), Capacity: map[model.Role]float64{model.RoleBE: 1}, Horizon: 10}}, model.RateCard, 0)
	if err != nil {
		t.Fatal(err)
	}
	if oc.Levelled != 3 || oc.MinDuration != 3 || len(oc.Points) != 0 {
		t.Errorf("tanpa perebutan: %d, %d, %d rencana; mau 3, 3, 0", oc.Levelled, oc.MinDuration, len(oc.Points))
	}
	if _, ok := oc.PointAt(3); ok {
		t.Error("tidak boleh ada rencana saat lembur tidak memotong apa pun")
	}
	p := compress.OvertimePoint{Hours: map[model.Role]float64{model.RoleFE: 2, model.RoleBE: 1, model.RoleQA: 0}}
	if r := p.Roles(); len(r) != 2 || r[0] != model.RoleBE || r[1] != model.RoleFE {
		t.Errorf("peran yang lembur %v, mau [BE FE]", r)
	}
}
