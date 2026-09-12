package resource_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/resource"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

func TestOverAllocationIsDetected(t *testing.T) {
	// Dua aktivitas paralel yang sama-sama membutuhkan Backend Developer
	// penuh waktu. CPM akan menjadwalkannya bersamaan; kapasitasnya tidak.
	acts := []model.Activity{
		{ID: "A", WBS: "3.2", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
		{ID: "B", WBS: "3.2", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
	}
	plan, err := schedule.Compute(acts, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	p := resource.Analyse(acts, plan, map[model.Role]float64{model.RoleBE: 1})

	if len(p.Conflicts) != 3 {
		t.Errorf("bentrokan terdeteksi = %d hari, mau 3", len(p.Conflicts))
	}
	for _, c := range p.Conflicts {
		if c.Load != 2 || c.Capacity != 1 || c.Excess != 1 {
			t.Errorf("bentrokan tidak terhitung benar: beban %v kapasitas %v kelebihan %v",
				c.Load, c.Capacity, c.Excess)
		}
		if len(c.Activities) != 2 {
			t.Errorf("bentrokan harus menyebut dua aktivitas, dapat %v", c.Activities)
		}
	}
}

func TestNoConflictWhenSequential(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", WBS: "3.2", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
		{ID: "B", WBS: "3.2", Duration: 3, Pred: []model.Predecessor{model.FS("A")},
			Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
	}
	plan, _ := schedule.Compute(acts, schedule.Options{})
	p := resource.Analyse(acts, plan, map[model.Role]float64{model.RoleBE: 1})
	if len(p.Conflicts) != 0 {
		t.Errorf("aktivitas berurutan tidak boleh bentrok, dapat %d", len(p.Conflicts))
	}
}

func TestPersonDaysMatchActivityTotals(t *testing.T) {
	plan, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	p := resource.Analyse(model.Activities, plan, model.Capacity)

	var want float64
	for _, a := range model.Activities {
		want += a.PersonDays()
	}
	if math.Abs(p.TotalPersonDays-want) > 1e-6 {
		t.Errorf("total hari-orang = %.2f, mau %.2f", p.TotalPersonDays, want)
	}
}

func TestUtilisationIsBounded(t *testing.T) {
	plan, _ := schedule.Compute(model.Activities, schedule.Options{})
	p := resource.Analyse(model.Activities, plan, model.Capacity)
	for _, r := range p.Roles {
		if r.Utilisation < 0 {
			t.Errorf("%s: utilisasi negatif", r.Role)
		}
		if r.PeakLoad <= 0 {
			t.Errorf("%s: puncak beban nol padahal peran ini ditugaskan", r.Role)
		}
		if r.FirstDay < 0 || r.LastDay < r.FirstDay {
			t.Errorf("%s: rentang hari kerja tidak masuk akal (%d..%d)", r.Role, r.FirstDay, r.LastDay)
		}
	}
}

func TestConflictsAreSortedByExcess(t *testing.T) {
	plan, _ := schedule.Compute(model.Activities, schedule.Options{})
	p := resource.Analyse(model.Activities, plan, model.Capacity)
	for i := 1; i < len(p.Conflicts); i++ {
		if p.Conflicts[i].Excess > p.Conflicts[i-1].Excess {
			t.Errorf("bentrokan tidak terurut dari kelebihan terbesar di posisi %d", i)
		}
	}
}

// TestRealProjectHasResourceConflicts menjaga temuan halaman Organisasi.
// Jadwal CPM proyek ini sah secara matematis tetapi tidak bisa dijalankan;
// kalau seseorang meratakan sumber dayanya, temuan itu harus ikut hilang.
func TestRealProjectHasResourceConflicts(t *testing.T) {
	plan, _ := schedule.Compute(model.Activities, schedule.Options{})
	p := resource.Analyse(model.Activities, plan, model.Capacity)
	if len(p.Conflicts) == 0 {
		t.Error("tidak ada over-alokasi; temuan di halaman Organisasi tidak lagi berlaku " +
			"dan teksnya harus diperbarui")
	}
	// Bentrokan terbesar harus melibatkan peran tunggal yang dikerjakan
	// paralel - itulah kelas masalah yang ingin ditunjukkan halaman itu.
	worst := p.Conflicts[0]
	if len(worst.Activities) < 2 {
		t.Errorf("bentrokan terberat hanya melibatkan %d aktivitas", len(worst.Activities))
	}
}

func TestSmoothnessIsBoundedAndMeaningful(t *testing.T) {
	plan, _ := schedule.Compute(model.Activities, schedule.Options{})
	p := resource.Analyse(model.Activities, plan, model.Capacity)

	s := p.Smoothness()
	if s < 0 {
		t.Errorf("kehalusan negatif: %v", s)
	}
	peak, day := p.PeakHeadcount()
	if peak <= 0 {
		t.Error("puncak jumlah peran aktif harus positif")
	}
	if day < 0 || day >= p.Horizon {
		t.Errorf("hari puncak %d di luar horizon %d", day, p.Horizon)
	}
	// Kurva yang benar-benar datar harus memberi kehalusan nol.
	flat := resource.Profile{Horizon: 4, Headcount: []float64{3, 3, 3, 3}}
	if got := flat.Smoothness(); got != 0 {
		t.Errorf("kurva datar memberi kehalusan %v, mau 0", got)
	}
}

func TestMilestonesDoNotConsumeResources(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", WBS: "1.1", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}},
		{ID: "M", WBS: "1.1", Milestone: true, Pred: []model.Predecessor{model.FS("A")}},
	}
	plan, _ := schedule.Compute(acts, schedule.Options{})
	p := resource.Analyse(acts, plan, map[model.Role]float64{model.RoleBE: 1})
	if math.Abs(p.TotalPersonDays-3) > 1e-9 {
		t.Errorf("total hari-orang = %v, mau 3; milestone tidak boleh menambah beban", p.TotalPersonDays)
	}
}
