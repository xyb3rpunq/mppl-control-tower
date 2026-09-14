package main

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// TestClosurePagesCarryLiveNumbers memastikan angka analisis penutup celah
// benar-benar sampai ke HTML dalam kedua bahasa, diformat dari struct.
func TestClosurePagesCarryLiveNumbers(t *testing.T) {
	a := analysisFor(t)
	pages := renderAll(t)
	for _, lang := range i18n.Langs {
		opt := pages[lang+" /optimasi/"]
		for _, want := range []string{
			render.Rp(a.TradeOffNet(5), lang),
			render.Rp(a.Exact.MaxGreedyExcess, lang),
			map[string]string{"id": "Terbukti optimal", "en": "Proven optimal"}[lang],
		} {
			if !strings.Contains(opt, want) {
				t.Errorf("/optimasi/ (%s) tidak memuat %q", lang, want)
			}
		}

		integ := pages[lang+" /simulasi-terpadu/"]
		for _, want := range []string{
			render.Num(a.GERT[0].Reduced.Mean(), 2, lang),
			render.Rp(a.RiskSweep[0].CostP95, lang),
			render.Rp(a.RentalDaily(), lang),
		} {
			if !strings.Contains(integ, want) {
				t.Errorf("/simulasi-terpadu/ (%s) tidak memuat %q", lang, want)
			}
		}

		fc := pages[lang+" /prakiraan/"]
		for _, want := range []string{
			render.Num(a.Forecast.DurP80, 0, lang),
			render.Num(a.IEACt, 1, lang),
			render.Rp(a.Forecast.CostP80, lang),
			render.Pct(a.InFlight.ExamFactor, 0, lang),
		} {
			if !strings.Contains(fc, want) {
				t.Errorf("/prakiraan/ (%s) tidak memuat %q", lang, want)
			}
		}
		if a.ForecastJCL70.Feasible && !strings.Contains(fc, render.Rp(a.ForecastJCL70.Budget, lang)) {
			t.Errorf("/prakiraan/ (%s) tidak memuat anggaran JCL 70%% berjalan", lang)
		}

		method := pages[lang+" /metode/"]
		if !strings.Contains(method, map[string]string{"id": "Batas yang tersisa", "en": "Remaining limits"}[lang]) {
			t.Errorf("/metode/ (%s) harus tetap menyebut batas yang tersisa", lang)
		}
	}
}

func TestClosureFindingsArePresent(t *testing.T) {
	a := analysisFor(t)
	keys := map[string]bool{}
	for _, f := range a.Findings {
		keys[f.Key] = true
	}
	for _, k := range []string{"levelling-terbukti-optimal", "crashing-hampir-impas", "risiko-bergerombol", "rework-berulang", "prakiraan-berjalan", "ujian-terkalibrasi"} {
		if !keys[k] {
			t.Errorf("temuan %q tidak diturunkan", k)
		}
	}
}

// TestLevelledScheduleIsTheProvenOne: seluruh halaman harus memakai jadwal
// yang sama - jadwal terbaik yang terbukti optimal, bukan SGS polos.
func TestLevelledScheduleIsTheProvenOne(t *testing.T) {
	a := analysisFor(t)
	if a.Level.Duration != a.LevelOpt.Best.Duration || a.Level.Duration != a.LevelWhy.WithWindows {
		t.Errorf("durasi levelling tidak konsisten: halaman %d, optimize %d, breakdown %d", a.Level.Duration, a.LevelOpt.Best.Duration, a.LevelWhy.WithWindows)
	}
	if !a.LevelOpt.Proven {
		t.Errorf("jadwal levelling tidak lagi terbukti optimal (celah %d); teks halaman Optimasi dan README harus diperbarui", a.LevelOpt.Gap)
	}
	if a.Level.Duration != 113 {
		t.Errorf("durasi levelling = %d, README dan memori menyebut 113", a.Level.Duration)
	}
	fin := a.Final()
	if fin.Audit.Audited == 0 {
		t.Error("audit levelling di dalam simulasi tidak berjalan")
	}
}

func TestGERTAnalyticMatchesSimulation(t *testing.T) {
	a := analysisFor(t)
	for _, g := range a.GERT {
		if math.Abs(g.MCCycles-g.Cycles) > 0.05 {
			t.Errorf("%s: rerata putaran simulasi %v jauh dari analitik %v", g.Loop.ID, g.MCCycles, g.Cycles)
		}
		if !g.CheckActive {
			t.Errorf("%s: pemeriksaannya belum lewat pada tanggal data, harus aktif", g.Loop.ID)
		}
	}
}

func TestForecastIsConsistentWithActuals(t *testing.T) {
	a := analysisFor(t)
	fc := a.Forecast
	if fc.Durations[0] < a.Snapshot.AtDay {
		t.Errorf("prakiraan tercepat %v sebelum tanggal data %v", fc.Durations[0], a.Snapshot.AtDay)
	}
	if fc.Costs[0] < a.Snapshot.AC {
		t.Errorf("biaya prakiraan termurah %v di bawah AC %v", fc.Costs[0], a.Snapshot.AC)
	}
	if a.ForecastJCL70.Feasible {
		if got := fc.Joint(a.ForecastJCL70.Duration, a.ForecastJCL70.Budget); got < 0.7 {
			t.Errorf("titik JCL 70%% berjalan hanya mencapai %v", got)
		}
	}
	want := a.Snapshot.AtDay + (float64(a.Plan.Duration)-a.Snapshot.ES)/a.Snapshot.SPIt
	if math.Abs(a.IEACt-want) > 1e-9 {
		t.Errorf("IEAC(t) %v, mau %v", a.IEACt, want)
	}
	if len(a.EvidenceRows()) != a.InFlight.Evidence {
		t.Errorf("baris bukti %d, bukti kalibrasi %d", len(a.EvidenceRows()), a.InFlight.Evidence)
	}
	for _, r := range a.RunningRows() {
		if r.CondMean < r.Elapsed || r.Remaining <= 0 {
			t.Errorf("%s: durasi bersyarat %v tidak boleh di bawah hari berjalan %v", r.ID, r.CondMean, r.Elapsed)
		}
	}
}

// TestMetricsJSONIsValidAndComplete memastikan metrik.json bisa dibaca mesin
// dan memuat seluruh blok baru.
func TestMetricsJSONIsValidAndComplete(t *testing.T) {
	a := analysisFor(t)
	var doc map[string]any
	if err := json.Unmarshal([]byte(metricsJSON(a)), &doc); err != nil {
		t.Fatalf("metrik.json tidak sah: %v", err)
	}
	for _, k := range []string{"earned_value", "levelling", "kompresi", "simulasi_terpadu", "prakiraan_berjalan"} {
		if _, ok := doc[k]; !ok {
			t.Errorf("metrik.json tanpa blok %q", k)
		}
	}
	st := doc["simulasi_terpadu"].(map[string]any)
	if layers := st["lapisan"].([]any); len(layers) != len(simulate.Layers) {
		t.Errorf("metrik.json memuat %d lapisan, mau %d", len(layers), len(simulate.Layers))
	}
	lv := doc["levelling"].(map[string]any)
	if lv["terbukti_optimal"] != true {
		t.Error("metrik.json harus melaporkan levelling terbukti optimal")
	}
	if ev := doc["earned_value"].(map[string]any); ev["spi"] == nil || ev["ieac_t_hari"] == nil {
		t.Error("kunci earned_value harus huruf kecil dan memuat ieac_t_hari")
	}
}

func TestTemplateHelpers(t *testing.T) {
	a := analysisFor(t)
	if ruleName("LST", "id") == ruleName("LST", "en") || ruleName("XYZ", "id") != "XYZ" {
		t.Error("ruleName harus dwibahasa dan mengembalikan kode untuk aturan tak dikenal")
	}
	rows := thinFrontier(a.Frontier, 3)
	if len(rows) == 0 || rows[0] != a.JCL70 {
		t.Error("penipisan frontier harus dimulai dari titik JCL 70% pertama")
	}
	if len(thinFrontier(a.Frontier, 0)) == 0 {
		t.Error("langkah nol harus diperlakukan sebagai satu")
	}
	var open int
	for _, r := range model.Risks {
		if !a.InFlight.ClosedRisk[r.ID] {
			open++
		}
	}
	if open != len(model.Risks)-len(a.InFlight.ClosedRisk) {
		t.Error("hitungan risiko terbuka tidak konsisten")
	}
}

func TestAnalysisHelpersAndStaticCopy(t *testing.T) {
	a := analysisFor(t)
	crit, total := a.CriticalCount()
	if crit == 0 || crit > total || total != len(model.Activities) {
		t.Errorf("simpul kritis %d dari %d", crit, total)
	}
	if got := len(a.CrashStepsUpTo(5)); got != 5 {
		t.Errorf("CrashStepsUpTo(5) = %d langkah", got)
	}
	if got := len(a.CrashStepsUpTo(999)); got != len(a.Crash.Steps) {
		t.Errorf("CrashStepsUpTo harus dipotong di jumlah langkah, dapat %d", got)
	}
	d := PageData{Lang: "en"}
	if d.Tx(model.Text{ID: "halo", EN: "hello"}) != "hello" {
		t.Error("PageData.Tx harus memilih bahasa halaman")
	}

	dir := t.TempDir()
	if err := copyStatic("web/static", dir); err != nil {
		t.Fatalf("copyStatic: %v", err)
	}
	for _, name := range []string{"css/app.css", "js/app.js", "js/sw.js"} {
		if _, err := os.Stat(dir + "/" + name); err != nil {
			t.Errorf("%s tidak tersalin: %v", name, err)
		}
	}
	if err := copyStatic("tidak/ada", t.TempDir()); err == nil {
		t.Error("direktori sumber yang tidak ada seharusnya galat")
	}
}
