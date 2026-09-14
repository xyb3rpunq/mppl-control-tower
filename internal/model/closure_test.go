package model_test

import (
	"math"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

func TestRiskDriversAreGroundedAndShared(t *testing.T) {
	count := map[string]int{}
	for _, r := range model.Risks {
		key := r.DriverOf()
		if key == "" {
			continue
		}
		if _, ok := model.DriverByKey(key); !ok {
			t.Errorf("%s menunjuk penggerak tak dikenal %q", r.ID, key)
		}
		count[key]++
	}
	for _, d := range model.RiskDrivers {
		if count[d.Key] < 2 {
			t.Errorf("penggerak %q hanya punya %d risiko; sebab bersama butuh paling sedikit dua", d.Key, count[d.Key])
		}
		if d.Label.ID == "" || d.Label.EN == "" || d.Why.ID == "" || d.Why.EN == "" {
			t.Errorf("penggerak %q tanpa label atau alasan dua bahasa", d.Key)
		}
	}
	if _, ok := model.DriverByKey("tidak-ada"); ok {
		t.Error("kunci penggerak tak dikenal harus mengembalikan false")
	}
	if model.RiskLoading <= 0 || model.RiskLoading >= 1 {
		t.Errorf("lambda %v di luar (0,1)", model.RiskLoading)
	}
}

func TestReworkLoopsPointAtRealActivities(t *testing.T) {
	byID := model.ActivityByID()
	for _, l := range model.ReworkLoops {
		if _, ok := byID[l.Check]; !ok {
			t.Errorf("%s: pemeriksaan %q tidak ada di jaringan", l.ID, l.Check)
		}
		if l.FailProb <= 0 || l.FailProb >= 1 {
			t.Errorf("%s: peluang gagal %v di luar (0,1)", l.ID, l.FailProb)
		}
		for _, p := range l.Parts {
			if _, ok := byID[p.Activity]; !ok || p.Fraction <= 0 || p.Fraction > 1 {
				t.Errorf("%s: bagian rework %+v tidak sah", l.ID, p)
			}
		}
		if got, ok := model.LoopByCheck(l.Check); !ok || got.ID != l.ID {
			t.Errorf("LoopByCheck(%s) tidak menemukan %s", l.Check, l.ID)
		}
		if l.Why.ID == "" || l.Why.EN == "" {
			t.Errorf("%s tanpa alasan dua bahasa", l.ID)
		}
	}
	if _, ok := model.LoopByCheck("A01"); ok {
		t.Error("A01 bukan pemeriksaan")
	}
}

func TestAvailabilityWindowsAreOfficialOrFlagged(t *testing.T) {
	prev := ""
	for _, w := range model.AvailabilityWindows {
		if w.From <= prev {
			t.Errorf("jendela %s tidak terurut", w.Key)
		}
		prev = w.From
		if !w.Asumsi && !strings.HasPrefix(w.Source, "https://") {
			t.Errorf("jendela %s berstatus resmi tetapi tanpa sumber", w.Key)
		}
		if w.Factor != model.ExamCapacityFactor {
			t.Errorf("jendela %s memakai faktor %v, mau faktor ujian tunggal %v", w.Key, w.Factor, model.ExamCapacityFactor)
		}
	}
	// UTS ganjil resmi: 3-15 November 2025.
	if got := model.CapacityOnDate(model.RoleBA, "2025-11-05", model.Capacity); math.Abs(got-model.ExamCapacityFactor) > 1e-9 {
		t.Errorf("kapasitas BA saat UTS = %v", got)
	}
	if got := model.CapacityOnDateWith(model.RoleBA, "2025-11-05", model.Capacity, 0.7); math.Abs(got-0.7) > 1e-9 {
		t.Errorf("faktor pengganti tidak dipakai: %v", got)
	}
	if got := model.CapacityOnDateWith(model.RoleBA, "2025-12-05", model.Capacity, 0.7); got != 1 {
		t.Errorf("faktor pengganti tidak boleh berlaku di luar jendela: %v", got)
	}
	// Lampiran kalender akademik resmi: keempat ujian semester, tanpa satu pun tanggal asumsi.
	want := map[string][2]string{
		"uts-ganjil": {"2025-11-03", "2025-11-15"},
		"uas-ganjil": {"2026-01-19", "2026-01-31"},
		"uts-genap":  {"2026-05-18", "2026-05-30"},
		"uas-genap":  {"2026-07-20", "2026-08-01"},
	}
	for _, w := range model.AvailabilityWindows {
		if w.Asumsi {
			t.Errorf("jendela %s masih berstatus asumsi", w.Key)
		}
		if d, ok := want[w.Key]; !ok || d[0] != w.From || d[1] != w.To {
			t.Errorf("jendela %s = %s..%s tidak sesuai kalender resmi", w.Key, w.From, w.To)
		}
	}
	if len(model.AvailabilityWindows) != len(want) {
		t.Errorf("jendela ujian %d, mau %d", len(model.AvailabilityWindows), len(want))
	}
}

// TestTimeBasedExtrasKeepBAC: menandai biaya sebagai sewa mengubah perilakunya
// saat proyek molor, tetapi tidak boleh mengubah BAC.
func TestTimeBasedExtrasKeepBAC(t *testing.T) {
	var rentals float64
	for _, a := range model.Activities {
		for _, e := range a.Extras {
			if e.TimeBased {
				rentals += e.Amount
			}
		}
	}
	if rentals != 2_000_000 {
		t.Errorf("sewa & langganan = %v, mau 2.000.000", rentals)
	}
	if model.BAC() != 14_832_000 {
		t.Errorf("BAC = %v, mau 14.832.000", model.BAC())
	}
}

func TestClosureFormulasAreRegistered(t *testing.T) {
	keys := map[string]model.Formula{}
	for _, f := range model.Formulas {
		if _, dup := keys[f.Key]; dup {
			t.Errorf("rumus %q terdaftar dua kali", f.Key)
		}
		keys[f.Key] = f
	}
	for _, k := range []string{"batasbawah", "lpcrash", "lemburlevelling", "hargaperhari", "biayawaktu", "kopularisiko", "gert", "kredibilitas"} {
		f, ok := keys[k]
		if !ok {
			t.Errorf("rumus %q tidak terdaftar", k)
			continue
		}
		if len(model.FormulasByGroup(f.Group)) == 0 {
			t.Errorf("rumus %q berada di kelompok tanpa anggota %q", k, f.Group)
		}
	}
	if !strings.Contains(keys["sgs"].Notation.ID, "0,2") {
		t.Error("notasi SGS belum diperbarui ke laju mulai minimum 0,2")
	}
}
