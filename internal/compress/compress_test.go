package compress_test

import (
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

func TestCrashCurveIsMonotonic(t *testing.T) {
	c, err := compress.Crash(model.Activities, model.RateCard)
	if err != nil {
		t.Fatalf("Crash: %v", err)
	}
	if c.NormalDuration != 85 {
		t.Errorf("durasi normal = %d, mau 85", c.NormalDuration)
	}
	if len(c.Steps) == 0 {
		t.Fatal("tidak ada satu langkah crashing pun")
	}
	prevDur, prevTotal := c.NormalDuration, 0.0
	for i, s := range c.Steps {
		if s.Duration != prevDur-1 {
			t.Errorf("langkah %d: durasi %d, mau %d (tepat satu hari per langkah)", i, s.Duration, prevDur-1)
		}
		if s.TotalCost <= prevTotal {
			t.Errorf("langkah %d: biaya kumulatif tidak naik", i)
		}
		if s.StepCost <= 0 {
			t.Errorf("langkah %d: biaya langkah tidak positif", i)
		}
		prevDur, prevTotal = s.Duration, s.TotalCost
	}
	if c.MinDuration != c.Steps[len(c.Steps)-1].Duration {
		t.Error("durasi minimum tidak sama dengan langkah terakhir")
	}
	if c.Exhausted.ID == "" || c.Exhausted.EN == "" {
		t.Error("alasan berhenti tidak dijelaskan dalam dua bahasa")
	}
}

// TestCrashStepsReallyShortenTheProject menerapkan ulang setiap langkah dan
// memeriksa durasinya dengan CPM. Kurva waktu-biaya yang "membeli" hari tanpa
// benar-benar memendekkan proyek adalah kesalahan paling mahal di teknik ini.
func TestCrashStepsReallyShortenTheProject(t *testing.T) {
	c, err := compress.Crash(model.Activities, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	dur := map[string]int{}
	for _, a := range model.Activities {
		dur[a.ID] = a.Duration
	}
	for i, s := range c.Steps {
		for _, id := range s.Crashed {
			dur[id]--
		}
		r, err := schedule.Compute(model.Activities, schedule.Options{DurationOf: func(a model.Activity) int { return dur[a.ID] }})
		if err != nil {
			t.Fatal(err)
		}
		if r.Duration != s.Duration {
			t.Errorf("langkah %d mengklaim %d hari, CPM menghitung %d", i, s.Duration, r.Duration)
		}
	}
	for _, a := range model.Activities {
		p := a.Crash(model.RateCard)
		if dur[a.ID] < a.Duration && !p.Allowed {
			t.Errorf("%s dipotong padahal tidak boleh di-crash", a.ID)
		}
		if p.Allowed && dur[a.ID] < p.CrashDur {
			t.Errorf("%s dipotong sampai %d, di bawah batas crash %d", a.ID, dur[a.ID], p.CrashDur)
		}
	}
}

func TestParallelCriticalPathsNeedPairs(t *testing.T) {
	// Dua jalur kritis paralel sepanjang 4 hari. Memotong satu jalur saja
	// tidak memendekkan proyek; crashing harus memotong keduanya sekaligus.
	team := []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}
	acts := []model.Activity{
		{ID: "X1", WBS: "3.2", Duration: 4, Optimistic: 2, Pessimistic: 6, Team: team},
		{ID: "X2", WBS: "3.2", Duration: 4, Optimistic: 2, Pessimistic: 6, Team: team},
	}
	c, err := compress.Crash(acts, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Steps) == 0 {
		t.Fatal("tidak ada langkah; pencarian pasangan tidak bekerja")
	}
	if len(c.Steps[0].Crashed) != 2 {
		t.Errorf("langkah pertama memotong %v, mau kedua jalur sekaligus", c.Steps[0].Crashed)
	}
}

func TestCostToSave(t *testing.T) {
	c, err := compress.Crash(model.Activities, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := c.CostToSave(0); v != 0 || !ok {
		t.Error("memotong nol hari harus gratis")
	}
	five, ok := c.CostToSave(5)
	if !ok || five != c.Steps[4].TotalCost {
		t.Errorf("biaya 5 hari = %v", five)
	}
	if _, ok := c.CostToSave(len(c.Steps) + 1); ok {
		t.Error("melebihi kemampuan kompresi seharusnya melapor tidak mungkin")
	}
}

func TestForbiddenActivitiesNeverCrash(t *testing.T) {
	for _, id := range []string{"A01", "A13", "A30", "A35"} {
		a := model.ActivityByID()[id]
		if p := a.Crash(model.RateCard); p.Allowed || p.Reason.ID == "" {
			t.Errorf("%s seharusnya tidak bisa di-crash dan punya alasan", id)
		}
	}
	for _, a := range model.Activities {
		p := a.Crash(model.RateCard)
		if !p.Allowed {
			continue
		}
		if p.CrashDur < a.Optimistic {
			t.Errorf("%s: durasi crash %d di bawah estimasi optimistis %d", a.ID, p.CrashDur, a.Optimistic)
		}
		if p.CrashDur >= a.Duration || p.SlopePerDay <= 0 {
			t.Errorf("%s: rencana crash tidak masuk akal (%d hari, Rp %.0f)", a.ID, p.CrashDur, p.SlopePerDay)
		}
	}
}

func TestFastTrackCandidates(t *testing.T) {
	all, err := compress.FastTrackCandidates(model.Activities, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("tidak ada kandidat fast-tracking")
	}
	viable := compress.ViableCandidates(all)
	if len(viable) == 0 {
		t.Fatal("tidak ada kandidat yang layak")
	}
	for _, f := range all {
		if f.SameResource && f.Viable() {
			t.Errorf("%s->%s dikerjakan orang yang sama tetapi dianggap layak", f.Pred, f.Succ)
		}
		if f.ExpectedRework < 0 || f.OverlapDays < 1 {
			t.Errorf("%s->%s: rework atau tumpang tindih tidak masuk akal", f.Pred, f.Succ)
		}
		// Tidak boleh ada kandidat yang melompati gerbang persetujuan.
		if model.ActivityByID()[f.Pred].Milestone {
			t.Errorf("%s->%s melompati milestone persetujuan", f.Pred, f.Succ)
		}
	}
	// Kandidat layak harus di depan.
	seenNonViable := false
	for _, f := range all {
		if !f.Viable() {
			seenNonViable = true
		} else if seenNonViable {
			t.Error("kandidat layak muncul setelah kandidat tidak layak")
			break
		}
	}
}

// TestFastTrackSavingsAreNotAdditive menjaga pesan halaman Optimasi:
// penghematan fast-tracking tidak bisa dijumlahkan, karena memendekkan satu
// jalur membuat jalur lain menjadi kritis.
func TestFastTrackSavingsAreNotAdditive(t *testing.T) {
	all, err := compress.FastTrackCandidates(model.Activities, model.RateCard)
	if err != nil {
		t.Fatal(err)
	}
	viable := compress.ViableCandidates(all)
	naive := 0
	for _, f := range viable {
		naive += f.DaysSaved
	}
	d, err := compress.ApplyFastTracks(model.Activities, viable)
	if err != nil {
		t.Fatal(err)
	}
	actual := 85 - d
	if actual <= 0 {
		t.Fatalf("menerapkan semua kandidat tidak menghemat apa pun (durasi %d)", d)
	}
	if actual > naive {
		t.Errorf("penghematan gabungan %d melebihi jumlah naif %d - mustahil", actual, naive)
	}
}
