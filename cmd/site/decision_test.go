package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/site"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// TestDecisionIsConsistent mengunci logika paket keputusan: satu komitmen,
// opsi yang benar-benar dinilai dari tanggal data, dan rekomendasi yang
// mengikuti angkanya.
func TestDecisionIsConsistent(t *testing.T) {
	a := analysisFor(t)
	d := a.Decision
	if d == nil || len(d.Options) < 2 {
		t.Fatal("paket keputusan tidak dibangun")
	}
	if d.From != a.InFlight.Now {
		t.Errorf("keputusan dari hari %d, tanggal data hari %d", d.From, a.InFlight.Now)
	}
	base := d.Options[0]
	if base.Accel != nil || base.JCL70 != a.ForecastJCL70 || base.Sim.DurP80 != a.Forecast.DurP80 {
		t.Error("opsi tanpa percepatan harus sama persis dengan prakiraan berjalan")
	}
	if math.Abs(d.BudgetRequest-(a.ForecastJCL70.Budget-model.TotalAuthorised)) > 1e-6 {
		t.Errorf("permintaan anggaran %v", d.BudgetRequest)
	}
	for i, o := range d.Options {
		if !o.FloorProven || o.Floor != o.FloorBound {
			t.Errorf("%s: lantai %d, batas bawah %d - belum terbukti", o.Key, o.Floor, o.FloorBound)
		}
		if i > 0 && o.Floor > base.Floor {
			t.Errorf("%s: lantai %d lebih lambat dari tanpa percepatan %d", o.Key, o.Floor, base.Floor)
		}
		// Hampir semua iterasi harus terbukti; sisanya dilaporkan halaman apa adanya.
		if o.Sim.Proof.ProvenShare() < 0.999 {
			t.Errorf("%s: hanya %v iterasi terbukti optimal", o.Key, o.Sim.Proof.ProvenShare())
		}
		if !o.JCL70.Feasible || o.Sim.Joint(o.JCL70.Duration, o.JCL70.Budget) < 0.7 {
			t.Errorf("%s: titik JCL 70%% tidak mencapai 70%%", o.Key)
		}
		if i > 0 {
			if o.DaysEarlier != base.JCL70.Duration-o.JCL70.Duration || math.Abs(o.ExtraBudget-(o.JCL70.Budget-base.JCL70.Budget)) > 1e-6 {
				t.Errorf("%s: selisih terhadap tanpa percepatan tidak konsisten", o.Key)
			}
			if o.DaysEarlier > 0 && math.Abs(o.PricePerDay-o.ExtraBudget/o.DaysEarlier) > 1e-6 {
				t.Errorf("%s: harga per hari %v", o.Key, o.PricePerDay)
			}
		}
	}
	c, ok := d.CheapestOption()
	if !ok {
		t.Fatal("tidak ada opsi termurah")
	}
	for _, o := range d.Options[1:] {
		if o.Assumption.ID == "" && o.DaysEarlier > 0 && o.PricePerDay < c.PricePerDay {
			t.Errorf("%s lebih murah per hari (%v) dari opsi termurah %s (%v)", o.Key, o.PricePerDay, c.Key, c.PricePerDay)
		}
	}
	f, _ := d.FastestOption()
	for _, o := range d.Options[1:] {
		if o.DaysEarlier > 0 && o.JCL70.Duration < f.JCL70.Duration {
			t.Errorf("%s lebih cepat dari opsi tercepat %s", o.Key, f.Key)
		}
	}
	// Peran lembur diturunkan dari rencana termurah durasi minimum.
	if ot, ok := d.Option("lembur"); ok {
		want := d.Overtime.Points[len(d.Overtime.Points)-1].Roles()
		if len(ot.Accel.OvertimeRoles) != len(want) {
			t.Errorf("peran lembur %v, rencana minimum %v", ot.Accel.OvertimeRoles, want)
		}
		if ot.Floor != d.Overtime.MinDuration {
			t.Errorf("lantai opsi lembur %d, durasi minimum kurva lembur %d", ot.Floor, d.Overtime.MinDuration)
		}
	}
	if d.PlanMissed <= 0 || d.PlanMissedFirst >= a.StatusDate {
		t.Errorf("rencana lembur perencanaan: %d hari-peran lewat, pertama %s", d.PlanMissed, d.PlanMissedFirst)
	}
}

// TestFindingsNameOneCommitment: tidak ada temuan yang lagi menyuruh sponsor
// memegang komitmen selain JCL 70% berjalan.
func TestFindingsNameOneCommitment(t *testing.T) {
	a := analysisFor(t)
	want := fmtRpForTest(a.ForecastJCL70.Budget)
	for _, f := range a.Findings {
		switch f.Key {
		case "jadwal-optimistis", "jcl-rendah", "prakiraan-berjalan":
			if !strings.Contains(f.Action.ID, want) {
				t.Errorf("%s tidak menyebut komitmen yang berlaku (%s): %q", f.Key, want, f.Action.ID)
			}
		case "eac-melewati-pagu":
			if !strings.Contains(f.Action.ID, fmtRpForTest(a.Decision.BudgetRequest)) {
				t.Errorf("%s tidak memakai permintaan anggaran tunggal: %q", f.Key, f.Action.ID)
			}
		}
		for _, bad := range []string{"Sampaikan komitmen P80", "Naikkan cadangan menjadi", "Ajukan percepatan lima hari ke sponsor"} {
			if strings.Contains(f.Action.ID, bad) {
				t.Errorf("%s masih merekomendasikan %q", f.Key, bad)
			}
		}
	}
	for _, f := range a.OtherActions() {
		if f.Key == "crashing-hampir-impas" || f.Key == "jcl-rendah" {
			t.Errorf("temuan komitmen %s tidak boleh muncul di tindakan lain", f.Key)
		}
	}
}

func TestDecisionReachesThePages(t *testing.T) {
	a := analysisFor(t)
	d := a.Decision
	c, _ := d.CheapestOption()
	share, gap := d.ProofSummary()
	pages := renderAll(t)
	for _, lang := range i18n.Langs {
		page := pages[lang+" /keputusan/"]
		for _, want := range []string{
			render.Rp(a.ForecastJCL70.Budget, lang), render.Rp(d.BudgetRequest, lang), render.Rp(c.PricePerDay, lang),
			c.Name.Get(lang), workcal.FormatDate(a.FinishISO(int(a.ForecastJCL70.Duration)), lang),
			map[string]string{"id": "BERLAKU", "en": "IN FORCE"}[lang],
		} {
			if !strings.Contains(page, want) {
				t.Errorf("/keputusan/ (%s) tidak memuat %q", lang, want)
			}
		}
		if share < 1 && !strings.Contains(page, render.Pct(share, 2, lang)) {
			t.Errorf("/keputusan/ (%s) tidak melaporkan porsi iterasi terbukti %v (celah %d)", lang, share, gap)
		}
		for _, o := range d.Options {
			if !strings.Contains(page, render.Rp(o.JCL70.Budget, lang)) {
				t.Errorf("/keputusan/ (%s) tidak memuat anggaran JCL 70%% opsi %s", lang, o.Key)
			}
		}
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(metricsJSON(a)), &doc); err != nil {
		t.Fatal(err)
	}
	ks := doc["keputusan_sponsor"].(map[string]any)
	if ks["opsi_termurah_per_hari"] != c.Key || len(ks["opsi"].([]any)) != len(d.Options) {
		t.Errorf("metrik.json keputusan tidak lengkap: %v", ks["opsi_termurah_per_hari"])
	}
}

// TestActualsRoundTrip: ekspor realisasi lalu impor tanpa perubahan harus
// mengembalikan realisasi yang sama persis.
func TestActualsRoundTrip(t *testing.T) {
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 400)
	acts := append([]model.Activity(nil), model.Activities...)
	csv := actualsCSV(acts, cal)
	for i := range acts {
		acts[i].Actual = model.Actual{}
	}
	if err := applyActualsCSV(strings.NewReader(csv), acts, cal); err != nil {
		t.Fatal(err)
	}
	for i, a := range acts {
		if a.Actual != model.Activities[i].Actual {
			t.Errorf("%s: realisasi %+v setelah impor, semula %+v", a.ID, a.Actual, model.Activities[i].Actual)
		}
	}
	if !strings.HasPrefix(csv, actualsHeader+"\n") || strings.Count(csv, "\n") != len(acts)+1 {
		t.Error("ekspor harus berkepala kolom dan satu baris per aktivitas")
	}

	// Realisasi baru benar-benar mengubah kalibrasi prakiraan berjalan.
	changed := append([]model.Activity(nil), model.Activities...)
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	var buf bytes.Buffer
	buf.WriteString(lines[0] + "\n")
	bumped := ""
	for _, l := range lines[1:] {
		parts := strings.Split(l, ",")
		if bumped == "" && parts[1] == "true" && parts[3] != "0" {
			bumped = parts[0]
			parts[3] = "9"
			l = strings.Join(parts, ",")
		}
		buf.WriteString(l + "\n")
	}
	if err := applyActualsCSV(&buf, changed, cal); err != nil {
		t.Fatal(err)
	}
	day := cal.FractionalIndexOf(model.DefaultStatusDate)
	before, err := simulate.PrepareInFlight(model.Activities, cal, day, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := simulate.PrepareInFlight(changed, cal, day, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bumped == "" || after.ObservedRatio == before.ObservedRatio {
		t.Errorf("mengubah durasi %s tidak mengubah rasio durasi teramati (%v)", bumped, after.ObservedRatio)
	}
}

func TestActualsRejectBadFiles(t *testing.T) {
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 400)
	good := actualsCSV(model.Activities, cal)
	lines := strings.Split(strings.TrimSpace(good), "\n")
	first := strings.Split(lines[1], ",")
	started := ""
	for _, l := range lines[1:] {
		if strings.Contains(l, ",true,") {
			started = l
			break
		}
	}
	id := strings.Split(started, ",")[0]
	replace := func(old, new string) string { return strings.Replace(good, old, new, 1) }
	cases := map[string]string{
		"kepala salah":          strings.Replace(good, actualsHeader, "id,mulai", 1),
		"kolom kurang":          replace(lines[1], first[0]+",false"),
		"aktivitas asing":       replace(lines[1], "Z99,false,,,"),
		"dua kali":              good + lines[1] + "\n",
		"dimulai bukan boolean": replace(lines[1], first[0]+",mungkin,,,"),
		"hari libur":            replace(started, id+",true,2025-10-25,3,100"),
		"durasi negatif":        replace(started, id+",true,"+strings.Split(started, ",")[2]+",-1,100"),
		"biaya negatif":         replace(started, id+",true,"+strings.Split(started, ",")[2]+",3,-5"),
		"baris hilang":          strings.Replace(good, lines[len(lines)-1]+"\n", "", 1),
		"csv rusak":             good + "\"tanpa penutup\n",
	}
	for name, body := range cases {
		acts := append([]model.Activity(nil), model.Activities...)
		if err := applyActualsCSV(strings.NewReader(body), acts, cal); err == nil {
			t.Errorf("%s: seharusnya ditolak", name)
			continue
		}
		for i := range acts {
			if acts[i].Actual != model.Activities[i].Actual {
				t.Fatalf("%s: impor gagal tetapi realisasi %s sudah berubah", name, acts[i].ID)
			}
		}
	}
	if err := loadActuals("tidak-ada.csv"); err == nil {
		t.Error("berkas yang tidak ada seharusnya galat")
	}
}

// TestLoadActualsFromFile: jalur bendera -realisasi membaca berkas yang sama
// dengan ekspor situs. Berkas tanpa perubahan meninggalkan model persis sama.
func TestLoadActualsFromFile(t *testing.T) {
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 400)
	before := append([]model.Activity(nil), model.Activities...)
	path := t.TempDir() + "/realisasi.csv"
	if err := os.WriteFile(path, []byte(actualsCSV(model.Activities, cal)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := loadActuals(path); err != nil {
		t.Fatal(err)
	}
	for i := range before {
		if model.Activities[i].Actual != before[i].Actual {
			t.Fatalf("%s berubah setelah memuat ekspornya sendiri", before[i].ID)
		}
	}
}

// TestDecisionRuleAndRobustness mengunci aturan pita nilai, bootstrap, dan
// skenario asumsi pada analisis sungguhan.
func TestDecisionRuleAndRobustness(t *testing.T) {
	a := analysisFor(t)
	d := a.Decision
	if len(d.Bands) == 0 || d.Bands[0].Option != 0 || d.Bands[0].From != 0 || !d.Bands[len(d.Bands)-1].Open() {
		t.Fatalf("pita nilai tidak dimulai dari tanpa percepatan atau tidak terbuka di atas: %+v", d.Bands)
	}
	if len(d.Bands) > 1 {
		if c, _ := d.CheapestOption(); d.Options[d.Bands[1].Option].Key != c.Key || math.Abs(d.Bands[1].From-c.PricePerDay) > 1e-6 {
			t.Errorf("pita kedua harus opsi termurah dengan batas = harga per harinya")
		}
	}
	// Manfaat bersih opsi pita memang terbesar di tengah setiap pita.
	for _, b := range d.Bands {
		v := b.From + 1
		if !b.Open() {
			v = (b.From + b.To) / 2
		}
		win := d.Options[b.Option]
		for _, o := range d.Options {
			if o.Assumption.ID == "" && v*o.DaysEarlier-o.ExtraBudget > v*win.DaysEarlier-win.ExtraBudget+1e-6 {
				t.Errorf("pada nilai %v, %s lebih baik dari opsi pita %s", v, o.Key, win.Key)
			}
		}
	}
	r := d.Robust
	if r == nil || r.Reps != site.BootstrapReps {
		t.Fatal("bootstrap tidak dijalankan")
	}
	for i, o := range d.Options {
		if o.JCL70.Duration < r.DurLo[i] || o.JCL70.Duration > r.DurHi[i] || o.JCL70.Budget < r.BudgetLo[i]-1 || o.JCL70.Budget > r.BudgetHi[i]+1 {
			t.Errorf("%s: angka titik di luar interval bootstrap 90%%", o.Key)
		}
	}
	if r.CheapestSame <= 0 || r.CheapestSame > 1 || r.BandsSame <= 0 || r.BandsSame > 1 || len(r.EdgeLo) != len(d.Bands)-1 {
		t.Errorf("ketahanan tidak konsisten: %+v", r)
	}
	if len(d.Scenarios) != 7 || d.Scenarios[0].Key != "dasar" {
		t.Fatalf("%d skenario, mau pembanding + 6 asumsi", len(d.Scenarios))
	}
	for _, s := range d.Scenarios {
		if len(s.Sims) != len(d.ScenarioOptions) {
			t.Fatalf("%s: %d simulasi untuk %d opsi", s.Key, len(s.Sims), len(d.ScenarioOptions))
		}
		for _, sim := range s.Sims {
			if sim.Config.Iterations != site.ScenarioIterations || sim.Config.ExactLevel {
				t.Errorf("%s: skenario harus %d iterasi dengan levelling cepat", s.Key, site.ScenarioIterations)
			}
		}
	}
	if got := d.Scenarios[3].Sims[0].Config.RiskLoading; got != 0.3 {
		t.Errorf("skenario lambda rendah memakai lambda %v", got)
	}
	if got := d.Scenarios[6].Sims[0].Config.ReworkScale; got != 1.5 {
		t.Errorf("skenario GERT tinggi memakai skala %v", got)
	}
	pages := renderAll(t)
	for _, lang := range i18n.Langs {
		page := pages[lang+" /keputusan/"]
		for _, b := range d.Bands[1:] {
			if !strings.Contains(page, render.Rp(b.From, lang)) {
				t.Errorf("/keputusan/ (%s) tidak memuat batas pita %v", lang, b.From)
			}
		}
		if !strings.Contains(page, render.Pct(r.BandsSame, 1, lang)) {
			t.Errorf("/keputusan/ (%s) tidak melaporkan ketahanan urutan pita", lang)
		}
	}
	for _, f := range a.Findings {
		if f.Key == "percepatan-tanggal-data" && (!strings.Contains(f.Action.ID, render.Rp(d.Bands[len(d.Bands)-1].From, "id")) || !strings.Contains(f.Action.EN, "assumption scenarios")) {
			t.Errorf("temuan percepatan tidak memakai aturan pita: %q", f.Action.ID)
		}
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(metricsJSON(a)), &doc); err != nil {
		t.Fatal(err)
	}
	ks := doc["keputusan_sponsor"].(map[string]any)
	if len(ks["pita_nilai_satu_hari"].([]any)) != len(d.Bands) || len(ks["skenario_asumsi"].([]any)) != len(d.Scenarios) || ks["ketahanan_bootstrap"] == nil {
		t.Error("metrik.json tanpa pita, ketahanan, atau skenario")
	}
}
