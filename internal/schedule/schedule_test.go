package schedule_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// mini membangun jaringan kecil yang jawabannya bisa dihitung dengan tangan,
// supaya kegagalan uji menunjuk ke rumusnya - bukan ke data proyek.
//
//	A(3) ─┐
//	      ├─> C(2) ─> D(4)
//	B(5) ─┘
//
// Forward: A ES0 EF2, B ES0 EF4, C ES5 EF6, D ES7 EF10. Durasi proyek 11.
// Backward: D LS7, C LS5, B LS0 (kritis), A LS2 (float 2).
func mini() []model.Activity {
	return []model.Activity{
		{ID: "A", Duration: 3},
		{ID: "B", Duration: 5},
		{ID: "C", Duration: 2, Pred: []model.Predecessor{model.FS("A"), model.FS("B")}},
		{ID: "D", Duration: 4, Pred: []model.Predecessor{model.FS("C")}},
	}
}

func TestForwardBackwardPassHandCheck(t *testing.T) {
	res, err := schedule.Compute(mini(), schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Duration != 11 {
		t.Errorf("durasi proyek = %d, mau 11", res.Duration)
	}
	want := map[string]struct{ es, ef, ls, lf, tf, ff int }{
		"A": {0, 2, 2, 4, 2, 2},
		"B": {0, 4, 0, 4, 0, 0},
		"C": {5, 6, 5, 6, 0, 0},
		"D": {7, 10, 7, 10, 0, 0},
	}
	for id, w := range want {
		got := res.Task(id)
		if got.ES != w.es || got.EF != w.ef || got.LS != w.ls || got.LF != w.lf {
			t.Errorf("%s: ES/EF/LS/LF = %d/%d/%d/%d, mau %d/%d/%d/%d",
				id, got.ES, got.EF, got.LS, got.LF, w.es, w.ef, w.ls, w.lf)
		}
		if got.TotalFloat != w.tf {
			t.Errorf("%s: total float = %d, mau %d", id, got.TotalFloat, w.tf)
		}
		if got.FreeFloat != w.ff {
			t.Errorf("%s: free float = %d, mau %d", id, got.FreeFloat, w.ff)
		}
	}
}

func TestCriticalPathIsConnected(t *testing.T) {
	res, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(res.CriticalPath) == 0 {
		t.Fatal("jalur kritis kosong")
	}
	// Setiap simpul di rantai harus benar-benar merupakan penerus langsung
	// dari simpul sebelumnya. Memfilter TotalFloat == 0 saja tidak cukup.
	for i := 1; i < len(res.CriticalPath); i++ {
		prev, cur := res.CriticalPath[i-1], res.CriticalPath[i]
		found := false
		for _, p := range res.Task(cur).Predecessors {
			if p.ID == prev {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rantai kritis terputus: %s bukan pendahulu langsung %s", prev, cur)
		}
	}
	// Seluruh simpul di jalur kritis harus ber-float nol.
	for _, id := range res.CriticalPath {
		if tf := res.Task(id).TotalFloat; tf != 0 {
			t.Errorf("%s ada di jalur kritis tetapi total float-nya %d", id, tf)
		}
	}
}

func TestFreeFloatNeverExceedsTotalFloat(t *testing.T) {
	res, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	for id, task := range res.Tasks {
		if task.FreeFloat > task.TotalFloat {
			t.Errorf("%s: free float %d melebihi total float %d", id, task.FreeFloat, task.TotalFloat)
		}
		if task.FreeFloat < 0 || task.TotalFloat < 0 {
			t.Errorf("%s: float negatif (FF %d, TF %d)", id, task.FreeFloat, task.TotalFloat)
		}
	}
}

// TestProjectDurationMatchesCharter menjaga invarian paling penting dari
// seluruh model: jaringan aktivitas harus menghasilkan 85 hari kerja, yaitu
// 17 minggu persis seperti yang disetujui di Project Charter. Kalau ada yang
// menambah aktivitas atau menggeser durasi tanpa menyesuaikan yang lain, uji
// inilah yang menangkapnya.
func TestProjectDurationMatchesCharter(t *testing.T) {
	res, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	want := model.ProjectCharter.DurationWeeks * 5
	if res.Duration != want {
		t.Errorf("durasi jaringan = %d hari kerja, Project Charter menjanjikan %d", res.Duration, want)
	}
}

func TestRelationTypes(t *testing.T) {
	cases := []struct {
		name  string
		rel   string
		lag   int
		wantB int // ES aktivitas B
	}{
		{"FS tanpa lag", "FS", 0, 4},
		{"FS lag 2", "FS", 2, 6},
		{"FS lead -1", "FS", -1, 3},
		{"SS tanpa lag", "SS", 0, 0},
		{"SS lag 2", "SS", 2, 2},
		{"FF tanpa lag", "FF", 0, 1}, // B selesai bersama A: ES = 4 - 3
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			acts := []model.Activity{
				{ID: "A", Duration: 4},
				{ID: "B", Duration: 3, Pred: []model.Predecessor{{ID: "A", Type: c.rel, Lag: c.lag}}},
			}
			res, err := schedule.Compute(acts, schedule.Options{})
			if err != nil {
				t.Fatalf("Compute: %v", err)
			}
			if got := res.Task("B").ES; got != c.wantB {
				t.Errorf("ES(B) = %d, mau %d", got, c.wantB)
			}
		})
	}
}

func TestCycleIsRejected(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", Duration: 1, Pred: []model.Predecessor{model.FS("C")}},
		{ID: "B", Duration: 1, Pred: []model.Predecessor{model.FS("A")}},
		{ID: "C", Duration: 1, Pred: []model.Predecessor{model.FS("B")}},
	}
	if _, err := schedule.Compute(acts, schedule.Options{}); err == nil {
		t.Fatal("jaringan bersiklus seharusnya ditolak, tetapi diterima")
	}
}

func TestUnknownPredecessorIsRejected(t *testing.T) {
	acts := []model.Activity{
		{ID: "A", Duration: 1, Pred: []model.Predecessor{model.FS("TIDAK-ADA")}},
	}
	if _, err := schedule.Compute(acts, schedule.Options{}); err == nil {
		t.Fatal("predecessor tak dikenal seharusnya ditolak")
	}
}

func TestDuplicateIDIsRejected(t *testing.T) {
	acts := []model.Activity{{ID: "A", Duration: 1}, {ID: "A", Duration: 2}}
	if _, err := schedule.Compute(acts, schedule.Options{}); err == nil {
		t.Fatal("kode aktivitas ganda seharusnya ditolak")
	}
}

func TestMilestoneHasZeroDuration(t *testing.T) {
	res, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	for _, a := range model.Activities {
		if !a.Milestone {
			continue
		}
		task := res.Task(a.ID)
		if task.Duration != 0 {
			t.Errorf("milestone %s berdurasi %d, seharusnya 0", a.ID, task.Duration)
		}
		// Konvensi inklusif membuat EF = ES - 1 untuk durasi nol.
		if task.EF != task.ES-1 {
			t.Errorf("milestone %s: EF=%d bukan ES-1=%d", a.ID, task.EF, task.ES-1)
		}
	}
}

func TestPERTExpectedFormula(t *testing.T) {
	// (2 + 4*5 + 14) / 6 = 36/6 = 6
	if got := schedule.Expected(2, 5, 14); math.Abs(got-6) > 1e-9 {
		t.Errorf("Expected(2,5,14) = %v, mau 6", got)
	}
	// Simetris: te harus sama dengan M ketika O dan P berjarak sama.
	if got := schedule.Expected(3, 5, 7); math.Abs(got-5) > 1e-9 {
		t.Errorf("Expected(3,5,7) = %v, mau 5", got)
	}
	// Condong ke kanan: te harus LEBIH BESAR dari M.
	if got := schedule.Expected(4, 6, 11); got <= 6 {
		t.Errorf("Expected(4,6,11) = %v, seharusnya di atas 6 karena ekor pesimistis lebih panjang", got)
	}
}

func TestPERTStdDevAndVariance(t *testing.T) {
	if got := schedule.StdDev(2, 14); math.Abs(got-2) > 1e-9 {
		t.Errorf("StdDev(2,14) = %v, mau 2", got)
	}
	if got := schedule.Variance(2, 14); math.Abs(got-4) > 1e-9 {
		t.Errorf("Variance(2,14) = %v, mau 4", got)
	}
}

func TestPERTProbabilityIsBounded(t *testing.T) {
	stats, err := schedule.AnalysePERT(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("AnalysePERT: %v", err)
	}
	if stats.StdDev <= 0 {
		t.Fatal("simpangan baku proyek harus positif")
	}
	// Peluang harus monoton naik terhadap target dan berada di [0,1].
	prev := -1.0
	for target := 60.0; target <= 130; target += 5 {
		p := stats.ProbabilityBy(target)
		if p < 0 || p > 1 {
			t.Errorf("P(<= %v) = %v, di luar [0,1]", target, p)
		}
		if p < prev {
			t.Errorf("peluang turun saat target naik: P(%v)=%v setelah %v", target, p, prev)
		}
		prev = p
	}
	// Peluang pada durasi harapan harus persis di sekitar 50%.
	if p := stats.ProbabilityBy(stats.ExpectedDuration); math.Abs(p-0.5) > 0.02 {
		t.Errorf("P(<= te) = %v, mau sekitar 0,5", p)
	}
}
