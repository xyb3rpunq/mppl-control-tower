package main

import (
	"math"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/site"
)

// TestUpgradePagesCarryLiveNumbers memastikan angka analisis lanjutan sampai
// ke halaman dalam kedua bahasa. Angka yang dicari diformat dari struct
// analisis, bukan diketik - kalau model berubah, uji ini tetap benar.
func TestUpgradePagesCarryLiveNumbers(t *testing.T) {
	a := analysisFor(t)
	pages := renderAll(t)
	fin := a.Final()

	for _, lang := range i18n.Langs {
		opt := pages[lang+" /optimasi/"]
		for _, want := range []string{
			render.Num(float64(a.Level.Duration), 0, lang),
			render.Num(float64(a.LevelWhy.CapacityOnly), 0, lang),
			string(a.CriticalRole),
			render.Rp(a.Crash.Steps[len(a.Crash.Steps)-1].TotalCost, lang),
		} {
			if !strings.Contains(opt, want) {
				t.Errorf("/optimasi/ (%s) tidak memuat %q", lang, want)
			}
		}

		integ := pages[lang+" /simulasi-terpadu/"]
		for _, want := range []string{
			render.Num(fin.DurP80, 0, lang),
			render.Rp(fin.CostP80, lang),
			render.Pct(fin.JointAtP80, 1, lang),
			render.Rp(a.JCL70.Budget, lang),
		} {
			if !strings.Contains(integ, want) {
				t.Errorf("/simulasi-terpadu/ (%s) tidak memuat %q", lang, want)
			}
		}

		dash := pages[lang+" /"]
		if !strings.Contains(dash, render.Pct(fin.JCL, 1, lang)) {
			t.Errorf("dasbor (%s) tidak menampilkan JCL", lang)
		}
	}
}

// TestRiskLayerCostMatchesRegisterEMV adalah pemeriksaan silang antar-halaman:
// kenaikan rerata biaya dari lapisan korelasi ke lapisan risiko - setelah
// biaya sewa yang ikut memanjang dikeluarkan - harus mendekati jumlah EMV
// residual pada halaman Risiko. Kalau berbeda jauh, simulasi dan register
// sedang memakai angka yang berbeda.
func TestRiskLayerCostMatchesRegisterEMV(t *testing.T) {
	a := analysisFor(t)
	delta := site.RiskCostDelta(a)
	emv := a.Risk.TotalResidualEMV
	if math.Abs(delta-emv)/emv > 0.05 {
		t.Errorf("kenaikan rerata biaya %.0f menyimpang lebih dari 5%% dari EMV residual %.0f", delta, emv)
	}
}

// TestPERTPageAndLadderAgree memastikan halaman PERT dan lapisan L0 tidak
// pernah menampilkan dua P80 "independen" yang berbeda.
func TestPERTPageAndLadderAgree(t *testing.T) {
	a := analysisFor(t)
	if a.Sim.P80 != a.Ladder[0].DurP80 || a.Sim.OnTimeProb != a.Ladder[0].OnTime {
		t.Errorf("halaman PERT (P80 %v, tepat waktu %v) berbeda dari L0 (P80 %v, tepat waktu %v)",
			a.Sim.P80, a.Sim.OnTimeProb, a.Ladder[0].DurP80, a.Ladder[0].OnTime)
	}
	for _, p := range a.RhoSweep {
		if p.Rho == 0 && math.Abs(p.StdDev-stdDevOf(a.Ladder[0].Durations)) > 1e-9 {
			t.Error("baris rho 0 berbeda dari lapisan independen")
		}
	}
}

func TestJCL70PointIsConsistent(t *testing.T) {
	a := analysisFor(t)
	if !a.JCL70.Feasible {
		t.Skip("frontier JCL 70% tidak tercapai; temuan dan halaman menanganinya dengan teks lain")
	}
	fin := a.Final()
	if got := fin.Joint(a.JCL70.Duration, a.JCL70.Budget); got < 0.7 {
		t.Errorf("titik JCL70 (%v, %v) hanya mencapai %v", a.JCL70.Duration, a.JCL70.Budget, got)
	}
	for _, p := range a.Frontier {
		if p.Feasible && p.Duration < a.JCL70.Duration {
			t.Errorf("ada tenggat layak %v yang lebih pendek dari titik JCL70 %v", p.Duration, a.JCL70.Duration)
		}
	}
}

func TestUpgradeFindingsArePresent(t *testing.T) {
	a := analysisFor(t)
	keys := map[string]bool{}
	for _, f := range a.Findings {
		keys[f.Key] = true
	}
	for _, k := range []string{"jadwal-tak-terjalankan", "jcl-rendah", "korelasi-diabaikan"} {
		if !keys[k] {
			t.Errorf("temuan %q tidak diturunkan", k)
		}
	}
}

// TestNoIndonesianMonthInEnglishFindings menangkap kebocoran yang lolos dari
// uji kata fungsi: singkatan bulan Indonesia di dalam kalimat Inggris.
func TestNoIndonesianMonthInEnglishFindings(t *testing.T) {
	a := analysisFor(t)
	for _, f := range a.Findings {
		for _, m := range []string{" Okt ", " Des ", " Mei ", " Agu "} {
			if strings.Contains(f.Detail.EN, m) || strings.Contains(f.Action.EN, m) {
				t.Errorf("temuan %q memuat singkatan bulan Indonesia %q dalam teks Inggris", f.Key, strings.TrimSpace(m))
			}
		}
	}
}

func TestNavigationOrderPlacesNewPages(t *testing.T) {
	want := []string{"/", "/piagam/", "/jadwal/", "/optimasi/", "/pert/", "/simulasi-terpadu/", "/biaya/", "/prakiraan/", "/keputusan/", "/risiko/"}
	for i, route := range want {
		if site.Pages[i].Route != route {
			t.Errorf("urutan navigasi ke-%d = %s, mau %s", i, site.Pages[i].Route, route)
		}
	}
	if len(site.Pages) != 16 {
		t.Errorf("jumlah rute = %d, mau 16", len(site.Pages))
	}
}

func TestAvailabilityWindowsAreWellFormed(t *testing.T) {
	for _, w := range model.AvailabilityWindows {
		if w.From > w.To {
			t.Errorf("jendela %s-%s terbalik", w.From, w.To)
		}
		if w.Factor <= 0 || w.Factor > 1 {
			t.Errorf("faktor kapasitas %v di luar (0,1]", w.Factor)
		}
		if w.Label.ID == "" || w.Label.EN == "" {
			t.Error("jendela ketersediaan tanpa label dua bahasa")
		}
		if w.RiskID != "" {
			found := false
			for _, r := range model.Risks {
				if r.ID == w.RiskID {
					found = true
				}
			}
			if !found {
				t.Errorf("jendela menunjuk risiko tak dikenal %q", w.RiskID)
			}
		}
	}
}

func stdDevOf(v []float64) float64 {
	if len(v) < 2 {
		return 0
	}
	var m float64
	for _, x := range v {
		m += x
	}
	m /= float64(len(v))
	var ss float64
	for _, x := range v {
		ss += (x - m) * (x - m)
	}
	return math.Sqrt(ss / float64(len(v)-1))
}
