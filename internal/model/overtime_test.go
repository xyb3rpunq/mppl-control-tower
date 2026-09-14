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
