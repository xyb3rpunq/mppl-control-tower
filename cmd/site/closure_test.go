package main

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
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
			greedyClaim(a.Exact.GreedyOptimal, a.Exact.MaxGreedyExcess, lang),
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
	for _, k := range []string{"levelling-terbukti-optimal", "crashing-hampir-impas", "risiko-bergerombol", "rework-berulang", "prakiraan-berjalan", "ujian-terkalibrasi", "percepatan-tanggal-data"} {
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
	pr := fin.Proof
	if pr.Iterations != fin.Config.Iterations || pr.ProvenShare() != 1 {
		t.Errorf("tidak setiap iterasi lapisan kapasitas terbukti optimal: %+v", pr)
	}
	if a.Forecast.Proof.ProvenShare() != 1 {
		t.Errorf("tidak setiap iterasi prakiraan berjalan terbukti optimal: %+v", a.Forecast.Proof)
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

// TestCalibratedParametersReachThePages memastikan premi lembur PP 35/2021 dan
// kredibilitas empiris benar-benar tampil, dan sisa teks asumsi lama hilang.
func TestCalibratedParametersReachThePages(t *testing.T) {
	a := analysisFor(t)
	pages := renderAll(t)
	lo, hi := 1.0, 0.0
	illegal := 0
	for _, act := range model.Activities {
		p := act.Crash(model.RateCard)
		if p.Allowed {
			lo, hi = math.Min(lo, p.Premium), math.Max(hi, p.Premium)
		} else if p.OvertimeHrs > model.OvertimeMaxDaily {
			illegal++
		}
	}
	fl := a.InFlight
	for _, lang := range i18n.Langs {
		opt := pages[lang+" /optimasi/"]
		for _, want := range []string{
			model.OvertimeRegulationURL,
			render.Pct(lo, 0, lang) + "–" + render.Pct(hi, 1, lang),
			strconv.Itoa(illegal) + map[string]string{"id": " aktivitas melebihi batas 4 jam", "en": " activities exceed the 4-hour limit"}[lang],
		} {
			if !strings.Contains(opt, want) {
				t.Errorf("/optimasi/ (%s) tidak memuat %q", lang, want)
			}
		}
		fc := pages[lang+" /prakiraan/"]
		for _, want := range []string{
			render.Pct(fl.Credibility, 0, lang) + " / " + render.Pct(fl.CostCredibility, 0, lang) + " / " + render.Pct(fl.ExamCredibility, 0, lang),
			render.Pct(math.Sqrt(fl.DurationVar), 1, lang),
			"tau² / (tau² + Var)",
		} {
			if !strings.Contains(fc, want) {
				t.Errorf("/prakiraan/ (%s) tidak memuat %q", lang, want)
			}
		}
		for route, bad := range map[string]string{
			"/prakiraan/": "k = ", "/metode/": "k = ", "/optimasi/": map[string]string{"id": "(asumsi)", "en": "(assumed)"}[lang],
		} {
			if strings.Contains(pages[lang+" "+route], bad) {
				t.Errorf("%s (%s) masih memuat teks asumsi lama %q", route, lang, bad)
			}
		}
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(metricsJSON(a)), &doc); err != nil {
		t.Fatal(err)
	}
	pb := doc["prakiraan_berjalan"].(map[string]any)
	if pb["kredibilitas_empiris"] != true || pb["varians_penaksir_durasi"] == nil || pb["kredibilitas_ujian_z"] == nil {
		t.Error("metrik.json harus melaporkan kredibilitas empiris beserta varians penaksirnya")
	}
}

// TestOvertimeReachesThePages: lembur sah pada jadwal nyata harus memakai
// jadwal levelling yang sama, tampil di kedua bahasa, dan menggantikan saran
// lama yang membeli hari dari jaringan CPM.
func TestOvertimeReachesThePages(t *testing.T) {
	a := analysisFor(t)
	ot := a.Overtime
	if ot.Levelled != a.Level.Duration || ot.Base.Duration != a.Level.Duration {
		t.Fatalf("lembur berangkat dari %d hari, jadwal halaman %d", ot.Levelled, a.Level.Duration)
	}
	minp, ok := ot.PointAt(ot.MinDuration)
	if !ok {
		t.Fatal("tidak ada rencana lembur pada durasi minimum")
	}
	pages := renderAll(t)
	for _, lang := range i18n.Langs {
		opt := pages[lang+" /optimasi/"]
		for _, want := range []string{
			render.Rp(minp.Cost, lang), render.Rp(minp.Net, lang), a.OvertimeRoleText(minp, lang),
			map[string]string{"id": "Lembur pada jadwal yang bisa dijalankan", "en": "Overtime on the executable schedule"}[lang],
		} {
			if !strings.Contains(opt, want) {
				t.Errorf("/optimasi/ (%s) tidak memuat %q", lang, want)
			}
		}
		home := pages[lang+" /"]
		for _, bad := range []string{"dibeli lewat crashing", "bought through crashing"} {
			if strings.Contains(home, bad) {
				t.Errorf("beranda (%s) masih menjanjikan hari dari crashing CPM: %q", lang, bad)
			}
		}
	}
	cheap, ok := a.Decision.CheapestOption()
	for _, f := range a.Findings {
		if f.Key == "jadwal-tak-terjalankan" && (!ok || !strings.Contains(f.Action.ID, fmtRpForTest(cheap.PricePerDay))) {
			t.Errorf("rekomendasi jadwal tak terjalankan tidak memakai harga percepatan dari tanggal data: %q", f.Action.ID)
		}
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(metricsJSON(a)), &doc); err != nil {
		t.Fatal(err)
	}
	lj := doc["lembur_jadwal_nyata"].(map[string]any)
	if lj["terbukti_minimum"] != true || int(lj["durasi_minimum"].(float64)) != ot.MinDuration || len(lj["titik"].([]any)) != len(ot.Points) {
		t.Errorf("metrik.json lembur tidak lengkap: %v", lj)
	}
}

func fmtRpForTest(v float64) string { return render.Rp(v, "id") }

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

// TestAssetVersionBustsStaleCaches: versi aset harus stabil untuk isi yang
// sama, berubah bila satu bait pun berubah, menempel pada URL aset di HTML,
// dan menggantikan penanda nama cache service worker.
func TestAssetVersionBustsStaleCaches(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/js", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/js/mppl.wasm", []byte("versi-1"), 0o644); err != nil {
		t.Fatal(err)
	}
	v1, err := assetVersion(dir)
	if err != nil || len(v1) != 10 {
		t.Fatalf("versi %q, galat %v", v1, err)
	}
	if again, _ := assetVersion(dir); again != v1 {
		t.Error("versi aset harus stabil untuk isi yang sama")
	}
	if err := os.WriteFile(dir+"/js/mppl.wasm", []byte("versi-2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v2, _ := assetVersion(dir); v2 == v1 {
		t.Error("versi aset tidak berubah padahal isi WebAssembly berubah")
	}
	if _, err := assetVersion(dir + "/tidak-ada"); err == nil {
		t.Error("direktori yang tidak ada seharusnya galat")
	}

	html := renderAll(t)["id /prakiraan/"]
	for _, want := range []string{`app.css?v=uji1234567`, `app.js?v=uji1234567`, `data-asset-version="uji1234567"`} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML tidak memuat %q", want)
		}
	}

	out := t.TempDir()
	if err := writeExtras(out, "https://example.test", analysisFor(t)); err != nil {
		t.Fatal(err)
	}
	sw, err := os.ReadFile(out + "/sw.js")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sw), "__ASSET_VERSION__") || !strings.Contains(string(sw), "ct-mppl-") {
		t.Error("service worker harus memuat nama cache berversi, bukan penanda mentah")
	}
}

// greedyClaim adalah teks KPI serakah vs eksak yang harus tampil sesuai hasilnya.
func greedyClaim(optimal bool, excess float64, lang string) string {
	if optimal {
		return map[string]string{"id": "serakah optimal di setiap titik", "en": "greedy is optimal at every point"}[lang]
	}
	return "+" + render.Rp(excess, lang)
}

// TestTablesNeverWidenThePage: halaman tidak boleh bergeser horizontal di
// layar sempit. Setiap tabel harus berkelas "data" (aturan CSS layar sempit
// menjadikannya wadah geser) atau berada di dalam .table-wrap, dan aturan
// CSS-nya harus tetap ada.
func TestTablesNeverWidenThePage(t *testing.T) {
	css, err := os.ReadFile("web/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	rule := regexp.MustCompile(`(?s)@media \(max-width: 720px\) \{\s*table\.data \{[^}]*display: block;[^}]*overflow-x: auto;`)
	if !rule.Match(css) {
		t.Fatal("aturan CSS tabel geser di layar sempit hilang")
	}
	if !strings.Contains(string(css), ".risk-list { display: grid; grid-template-columns: minmax(0, 1fr);") {
		t.Error("kolom grid kartu risiko harus boleh menyusut, atau tabelnya melebarkan halaman")
	}
	table := regexp.MustCompile(`<table(\s[^>]*)?>`)
	for key, html := range renderAll(t) {
		for _, m := range table.FindAllStringSubmatch(html, -1) {
			if !strings.Contains(m[1], `class="data`) {
				t.Errorf("%s: tabel tanpa kelas data tidak tertangani aturan layar sempit: %s", key, m[0])
			}
		}
	}
}
