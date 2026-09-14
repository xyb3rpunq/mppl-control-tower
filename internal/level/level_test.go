package level_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

func cal(t *testing.T) *workcal.Calendar {
	t.Helper()
	return workcal.MustNew(model.ProjectCharter.StartDate, 400)
}

func be(alloc float64) []model.TeamSlot {
	return []model.TeamSlot{{Role: model.RoleBE, Alloc: alloc}}
}

// Dua aktivitas 3 hari yang sama-sama butuh satu Backend Developer penuh
// waktu. CPM menjadwalkannya paralel (3 hari); kapasitas nyata memaksanya
// berurutan (6 hari).
func TestParallelSameRoleBecomesSequential(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", WBS: "3.2", Duration: 3, Team: be(1)},
		{ID: "B", WBS: "3.2", Duration: 3, Team: be(1)},
	}
	r, err := level.Run(acts, level.Options{Calendar: cal(t), Capacity: map[model.Role]float64{model.RoleBE: 1}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.CPMDuration != 3 {
		t.Errorf("durasi CPM = %d, mau 3", r.CPMDuration)
	}
	if r.Duration != 6 {
		t.Errorf("durasi levelling = %d, mau 6", r.Duration)
	}
}

// Aktivitas penuh waktu yang dikerjakan peran paruh waktu harus memanjang dua
// kali lipat di kalender, tanpa menambah hari-orang.
func TestHalfTimeRoleStretchesCalendarNotWork(t *testing.T) {
	acts := []model.Activity{{ID: "A", WBS: "3.1", Duration: 3, Team: []model.TeamSlot{{Role: model.RoleOPS, Alloc: 1}}}}
	r, err := level.Run(acts, level.Options{Calendar: cal(t), Capacity: map[model.Role]float64{model.RoleOPS: 0.5}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.Duration != 6 {
		t.Errorf("durasi = %d, mau 6 (3 hari kerja pada kapasitas 50%%)", r.Duration)
	}
	task := r.Tasks["A"]
	if !task.HalfTime {
		t.Error("penyebab paruh waktu tidak ditandai")
	}
	var used float64
	for _, u := range r.Usage[model.RoleOPS] {
		used += u
	}
	// Isi pekerjaannya tetap 1 x 3 = 3 hari-orang; yang berubah hanya
	// kecepatannya (0,5 per hari) sehingga kalendernya menjadi 6 hari.
	if math.Abs(used-3) > 1e-9 {
		t.Errorf("hari-orang terpakai = %v, mau 3 (isi pekerjaan tidak boleh berubah)", used)
	}
}

func TestSequentialActivitiesAreUnchanged(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", WBS: "3.2", Duration: 3, Team: be(1)},
		{ID: "B", WBS: "3.2", Duration: 2, Team: be(1), Pred: []model.Predecessor{model.FS("A")}},
	}
	r, err := level.Run(acts, level.Options{Calendar: cal(t), Capacity: map[model.Role]float64{model.RoleBE: 1}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.Duration != r.CPMDuration {
		t.Errorf("jaringan tanpa bentrokan seharusnya tidak berubah: CPM %d, levelling %d", r.CPMDuration, r.Duration)
	}
	if len(r.Moved()) != 0 {
		t.Errorf("tidak ada aktivitas yang seharusnya bergeser, dapat %d", len(r.Moved()))
	}
}

// TestRealProjectLevellingInvariants menjaga sifat yang harus berlaku pada
// jadwal levelling mana pun, lalu mengunci angka yang ditampilkan di situs.
func TestRealProjectLevellingInvariants(t *testing.T) {
	c := cal(t)
	r, err := level.Run(model.Activities, level.Options{Calendar: c, Capacity: model.Capacity, UseWindows: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 1. Tidak ada satu hari-peran pun yang melebihi kapasitas.
	for _, role := range r.Roles {
		for k := 0; k < r.Duration; k++ {
			if r.Usage[role][k] > r.Cap[role][k]+1e-6 {
				t.Fatalf("%s hari %d: pemakaian %.3f melebihi kapasitas %.3f", role, k, r.Usage[role][k], r.Cap[role][k])
			}
		}
	}

	// 2. Hari-orang dilestarikan: levelling menggeser pekerjaan, tidak
	//    menambah atau menghilangkannya.
	for _, role := range r.Roles {
		var want, got float64
		for _, a := range model.Activities {
			for _, s := range a.Team {
				if s.Role == role {
					want += s.Alloc * float64(a.Duration)
				}
			}
		}
		for _, u := range r.Usage[role] {
			got += u
		}
		if math.Abs(want-got) > 1e-6 {
			t.Errorf("%s: hari-orang %.3f setelah levelling, mau %.3f", role, got, want)
		}
	}

	// 3. Setiap relasi FS tetap dihormati, dan tidak ada yang mulai lebih
	//    awal dari CPM.
	for _, a := range model.Activities {
		task := r.Tasks[a.ID]
		if task.Start < task.ES {
			t.Errorf("%s mulai hari %d, lebih awal dari ES CPM %d", a.ID, task.Start, task.ES)
		}
		for _, p := range a.Pred {
			if pt := r.Tasks[p.ID]; task.Start < pt.Finish {
				t.Errorf("%s mulai hari %d sebelum pendahulu %s selesai hari %d", a.ID, task.Start, p.ID, pt.Finish)
			}
		}
		if task.CarriedDays+task.WaitDays != task.Start-task.ES {
			t.Errorf("%s: pemecahan keterlambatan tidak menjumlah", a.ID)
		}
	}

	// 4. Durasi levelling tidak mungkin lebih pendek dari CPM.
	if r.Duration < r.CPMDuration {
		t.Errorf("levelling %d lebih pendek dari CPM %d", r.Duration, r.CPMDuration)
	}
	// SGS polos dengan aturan LST: satu hari di atas optimum (lihat
	// TestRealProjectIsProvenOptimal).
	if r.Duration != 114 {
		t.Errorf("durasi levelling LST = %d hari kerja, mau 114 - perbarui teks situs bila model berubah", r.Duration)
	}
}

func TestBreakdownIsMonotonic(t *testing.T) {
	b, err := level.Explain(model.Activities, level.OptimizeOptions{Options: level.Options{Calendar: cal(t), Capacity: model.Capacity}, Samples: 60})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if !(b.CPM <= b.CapacityOnly && b.CapacityOnly <= b.WithWindows) {
		t.Errorf("tahapan tidak monoton: CPM %d, kapasitas %d, + jendela %d", b.CPM, b.CapacityOnly, b.WithWindows)
	}
	if b.WithWindows == b.CapacityOnly {
		t.Error("jendela ujian tidak berpengaruh sama sekali; periksa tanggal jendela terhadap jadwal")
	}
	if b.Stage[0].Best.Duration != b.CapacityOnly || b.Stage[1].Best.Duration != b.WithWindows {
		t.Error("Breakdown harus memakai jadwal terbaik dari Optimize pada kedua tahap")
	}
}

// TestLiteModeMatchesFullMode memastikan jalan pintas simulasi tidak mengubah
// hasil. Mode ringkas hanya boleh melewatkan pencatatan, bukan menghitung beda.
func TestLiteModeMatchesFullMode(t *testing.T) {
	c := cal(t)
	full, err := level.Run(model.Activities, level.Options{Calendar: c, Capacity: model.Capacity, UseWindows: true})
	if err != nil {
		t.Fatal(err)
	}
	grid := level.CapacityGrid(c, model.Capacity, true, 300)
	lite, err := level.Run(model.Activities, level.Options{Calendar: c, Capacity: model.Capacity, UseWindows: true, Lite: true, CapGrid: grid})
	if err != nil {
		t.Fatal(err)
	}
	if full.Duration != lite.Duration {
		t.Fatalf("durasi berbeda: lengkap %d, ringkas %d", full.Duration, lite.Duration)
	}
	for id, ft := range full.Tasks {
		lt := lite.Tasks[id]
		if ft.Start != lt.Start || ft.Finish != lt.Finish {
			t.Errorf("%s: lengkap %d-%d, ringkas %d-%d", id, ft.Start, ft.Finish, lt.Start, lt.Finish)
		}
	}
}

func TestProfileHasNoOverAllocation(t *testing.T) {
	r, err := level.Run(model.Activities, level.Options{Calendar: cal(t), Capacity: model.Capacity, UseWindows: true})
	if err != nil {
		t.Fatal(err)
	}
	p := r.Profile()
	for _, role := range p.Roles {
		if role.OverDays != 0 {
			t.Errorf("%s masih punya %d hari over-alokasi setelah levelling", role.Role, role.OverDays)
		}
	}
	if len(p.Conflicts) != 0 {
		t.Errorf("profil levelling memuat %d konflik", len(p.Conflicts))
	}
}

func TestInvalidOptionsAreRejected(t *testing.T) {
	if _, err := level.Run(model.Activities, level.Options{Capacity: model.Capacity}); err == nil {
		t.Error("tanpa kalender seharusnya galat")
	}
	if _, err := level.Run(model.Activities, level.Options{Calendar: cal(t)}); err == nil {
		t.Error("tanpa kapasitas seharusnya galat")
	}
	// Horizon terlalu pendek harus melapor, bukan berputar tanpa akhir.
	if _, err := level.Run(model.Activities, level.Options{Calendar: cal(t), Capacity: model.Capacity, Horizon: 20}); err == nil {
		t.Error("horizon 20 hari tidak mungkin cukup, seharusnya galat")
	}
}

func TestCapacityOnDateAppliesExamWindow(t *testing.T) {
	inside := model.CapacityOnDate(model.RoleBE, "2026-01-15", model.Capacity)
	outside := model.CapacityOnDate(model.RoleBE, "2026-02-02", model.Capacity)
	if math.Abs(inside-0.4) > 1e-9 {
		t.Errorf("kapasitas saat ujian = %v, mau 0,4", inside)
	}
	if outside != 1 {
		t.Errorf("kapasitas di luar ujian = %v, mau 1", outside)
	}
	if got := model.CapacityOnDate(model.RoleOPS, "2026-01-15", model.Capacity); math.Abs(got-0.2) > 1e-9 {
		t.Errorf("DevOps paruh waktu saat ujian = %v, mau 0,2 (0,5 x 0,4)", got)
	}
}
