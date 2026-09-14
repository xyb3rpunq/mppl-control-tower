package model_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// TestOvertimeUnitsFollowPP35 mengunci Pasal 31 PP 35/2021: jam lembur
// pertama dibayar 1,5 kali upah sejam, jam-jam berikutnya 2 kali.
func TestOvertimeUnitsFollowPP35(t *testing.T) {
	cases := map[float64]float64{-1: 0, 0: 0, 0.5: 0.75, 1: 1.5, 2: 3.5, 4: 7.5}
	for h, want := range cases {
		if got := model.OvertimeUnits(h); math.Abs(got-want) > 1e-12 {
			t.Errorf("OvertimeUnits(%v) = %v, mau %v", h, got, want)
		}
	}
}

// TestOvertimePremiumByHand menghitung ulang premi jam demi jam, lepas dari
// rumus tertutupnya: pekerjaan hari yang dipotong dibagi rata ke hari tersisa.
func TestOvertimePremiumByHand(t *testing.T) {
	cases := []struct {
		cut, days int
		premium   float64
		hours     float64
		ok        bool
	}{
		{1, 4, 0.75, 2, true},         // M = 5
		{1, 3, 0.8125, 8.0 / 3, true}, // M = 4
		{1, 2, 0.875, 4, true},        // M = 3: tepat di batas 4 jam
		{2, 4, 0.875, 4, true},        // M = 6
		{1, 1, 0, 8, false},           // M = 2: 8 jam lembur sehari
		{3, 6, 0, 4, false},           // 4 jam x 5 hari = 20 jam > 18 jam seminggu
		{0, 3, 0, 0, false},
		{1, 0, 0, 0, false},
	}
	for _, c := range cases {
		p, h, ok := model.OvertimePremium(c.cut, c.days)
		if ok != c.ok || math.Abs(h-c.hours) > 1e-9 || math.Abs(p-c.premium) > 1e-9 {
			t.Errorf("OvertimePremium(%d, %d) = %v, %v jam, %v; mau %v, %v jam, %v", c.cut, c.days, p, h, ok, c.premium, c.hours, c.ok)
		}
		if !ok {
			continue
		}
		// Upah sejam = upah harian / 8 (Pasal 32: 1/173 upah sebulan, 173 = 21,6 hari x 8 jam).
		// Setiap menit lembur dibayar menurut jam ke berapa menit itu jatuh.
		const hourly = 1.0 / 8
		minutes := int(math.Round(h * 60))
		if math.Abs(float64(minutes)-h*60) > 1e-9 {
			t.Fatalf("kasus uji %v jam tidak jatuh di menit bulat", h)
		}
		var pay float64
		for d := 0; d < c.days; d++ {
			for m := 0; m < minutes; m++ {
				rate := 2.0
				if m < 60 {
					rate = 1.5
				}
				pay += hourly * rate / 60
			}
		}
		extra := pay - float64(c.cut) // tambahan di atas upah normal hari yang dipotong
		if got := extra / float64(c.cut); math.Abs(got-c.premium) > 1e-9 {
			t.Errorf("hitung jam demi jam (%d, %d) = %v, rumus %v", c.cut, c.days, got, c.premium)
		}
	}
}

// TestCrashPlansAreLegal: setiap aktivitas yang boleh dipercepat harus
// lembur dalam batas PP 35/2021, dan setiap penolakan punya alasan dwibahasa.
func TestCrashPlansAreLegal(t *testing.T) {
	var allowed, illegal int
	for _, a := range model.Activities {
		if a.Milestone {
			continue
		}
		p := a.Crash(model.RateCard)
		if !p.Allowed {
			if p.Reason.ID == "" || p.Reason.EN == "" {
				t.Errorf("%s ditolak tanpa alasan dwibahasa", a.ID)
			}
			if p.OvertimeHrs > model.OvertimeMaxDaily {
				illegal++
			}
			continue
		}
		allowed++
		if p.OvertimeHrs <= 0 || p.OvertimeHrs > model.OvertimeMaxDaily+1e-9 {
			t.Errorf("%s: %v jam lembur sehari di luar batas", a.ID, p.OvertimeHrs)
		}
		// Premi minimum 50% (jam pertama 1,5x), maksimum 100% (semua jam 2x).
		if p.Premium < 0.5 || p.Premium > 1 {
			t.Errorf("%s: premi %v di luar [0,5; 1]", a.ID, p.Premium)
		}
		want, _, _ := model.OvertimePremium(a.Duration-p.CrashDur, p.CrashDur)
		daily := a.LabourCost(model.RateCard) / float64(a.Duration)
		if math.Abs(p.Premium-want) > 1e-12 || math.Abs(p.SlopePerDay-daily*want) > 1e-6 {
			t.Errorf("%s: premi %v slope %v, mau %v dan %v", a.ID, p.Premium, p.SlopePerDay, want, daily*want)
		}
		if p.MaxDaysSaved != a.Duration-p.CrashDur || p.CrashDur < a.Optimistic {
			t.Errorf("%s: hemat %d hari, durasi crash %d, O %d", a.ID, p.MaxDaysSaved, p.CrashDur, a.Optimistic)
		}
	}
	if allowed == 0 || illegal != 5 {
		t.Errorf("%d aktivitas boleh crash, %d melanggar batas lembur; mau >0 dan 5", allowed, illegal)
	}
	if model.OvertimeRegulationURL == "" {
		t.Error("aturan lembur harus bersumber")
	}
}

// TestCrashCostIsConvex: biaya marjinal tiap hari yang dipotong tidak pernah
// menurun, jumlahnya sama dengan biaya potongan penuh, dan contoh M = 6 cocok
// dengan hitungan tangan.
func TestCrashCostIsConvex(t *testing.T) {
	for _, a := range model.Activities {
		p := a.Crash(model.RateCard)
		if !p.Allowed {
			continue
		}
		daily := a.LabourCost(model.RateCard) / float64(a.Duration)
		if len(p.Marginal) != p.MaxDaysSaved || len(p.MarginalPremium) != p.MaxDaysSaved {
			t.Errorf("%s: %d biaya marjinal untuk %d hari", a.ID, len(p.Marginal), p.MaxDaysSaved)
		}
		for i := 1; i < len(p.Marginal); i++ {
			if p.Marginal[i] < p.Marginal[i-1]-1e-9 {
				t.Errorf("%s: biaya hari ke-%d %v di bawah hari sebelumnya %v", a.ID, i+1, p.Marginal[i], p.Marginal[i-1])
			}
		}
		full := daily * p.Premium * float64(p.MaxDaysSaved)
		if math.Abs(p.CostToCut(p.MaxDaysSaved)-full) > 1e-6 || math.Abs(p.SlopePerDay*float64(p.MaxDaysSaved)-full) > 1e-6 {
			t.Errorf("%s: jumlah marjinal %v, potongan penuh %v", a.ID, p.CostToCut(p.MaxDaysSaved), full)
		}
		if p.CostToCut(0) != 0 || p.CostToCut(p.MaxDaysSaved+5) != p.CostToCut(p.MaxDaysSaved) {
			t.Errorf("%s: CostToCut di luar rentang tidak dipotong ke batas", a.ID)
		}
	}
	// M = 6, upah harian 55.000: hari pertama 1,6 jam/hari selama 5 hari,
	// hari kedua menaikkan ke 4 jam/hari selama 4 hari.
	a17 := model.ActivityByID()["A17"].Crash(model.RateCard)
	if len(a17.Marginal) != 2 || math.Abs(a17.Marginal[0]-37812.5) > 1e-6 || math.Abs(a17.Marginal[1]-58437.5) > 1e-6 {
		t.Errorf("A17 marjinal %v, mau [37812,5 58437,5]", a17.Marginal)
	}
}

// TestCrashStopsAtTheLastLegalDay: bila potongan penuh melanggar batas
// mingguan, hari-hari yang masih sah tetap boleh dipotong.
func TestCrashStopsAtTheLastLegalDay(t *testing.T) {
	a := model.Activity{ID: "X", Duration: 9, Optimistic: 1, Pessimistic: 12, Team: []model.TeamSlot{{Role: model.RoleBE, Alloc: 1}}}
	p := a.Crash(model.RateCard)
	// k = 3: 4 jam x 5 hari = 20 jam > 18; k = 2: 16/7 jam x 5 hari = 11,4 jam.
	if !p.Allowed || p.MaxDaysSaved != 2 || p.CrashDur != 7 || math.Abs(p.OvertimeHrs-16.0/7) > 1e-9 {
		t.Errorf("rencana %+v, mau 2 hari sah sampai durasi 7", p)
	}
}
