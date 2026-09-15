package render

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	elemRe = regexp.MustCompile(`<(rect|circle|line|text|path)\b([^>]*)>`)
	attrRe = regexp.MustCompile(`([a-z0-9-]+)="([^"]*)"`)
)

// elems mengembalikan atribut setiap elemen bertag tag yang kelasnya memuat
// seluruh token pada class.
func elems(svg, tag, class string) []map[string]string {
	var out []map[string]string
	want := strings.Fields(class)
	for _, m := range elemRe.FindAllStringSubmatch(svg, -1) {
		if m[1] != tag {
			continue
		}
		attrs := map[string]string{}
		for _, a := range attrRe.FindAllStringSubmatch(m[2], -1) {
			attrs[a[1]] = a[2]
		}
		have := map[string]bool{}
		for _, c := range strings.Fields(attrs["class"]) {
			have[c] = true
		}
		ok := true
		for _, w := range want {
			ok = ok && have[w]
		}
		if ok {
			out = append(out, attrs)
		}
	}
	return out
}

func fl(t *testing.T, m map[string]string, key string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(m[key], 64)
	if err != nil {
		t.Fatalf("atribut %s=%q bukan angka", key, m[key])
	}
	return v
}

func TestSweepPanelsDoNotExaggerateTinyChanges(t *testing.T) {
	xs := []float64{0, 0.3, 0.6, 0.9}
	panels := []SweepPanel{
		{Title: "Rerata biaya", Values: []float64{19_183_000, 19_180_000, 19_182_000, 19_181_000}, Format: func(v float64) string { return RpShort(v, "id") }},
		{Title: "Simpangan baku", Values: []float64{4.03, 4.4, 4.96, 5.88}},
		{Title: "Tanpa format", Values: []float64{1, 2}},
	}
	svg := string(SweepPanels(xs, "lambda", 0.6, panels, "Judul", "Deskripsi", "id"))
	checkSVG(t, "SweepPanels", svg)
	attrsWithin(t, "SweepPanels", svg, 880, 8+176)
	if got := strings.Count(svg, `class="sweep-panel"`); got != len(panels) {
		t.Errorf("panel = %d, mau %d", got, len(panels))
	}
	dots := elems(svg, "circle", "sweep-dot")
	if len(dots) != 4+4+2 {
		t.Fatalf("titik = %d, mau 10 (deret terpendek memotong sumbu)", len(dots))
	}
	spread := func(ds []map[string]string) float64 {
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, d := range ds {
			lo, hi = math.Min(lo, fl(t, d, "cy")), math.Max(hi, fl(t, d, "cy"))
		}
		return hi - lo
	}
	// Rp 3 ribu pada Rp 19 juta harus tampak datar; 4,03 -> 5,88 harus curam.
	if s := spread(dots[:4]); s > 2 {
		t.Errorf("perubahan 0,02%% digambar setinggi %.1f px - skala otomatis membesar-besarkan", s)
	}
	if s := spread(dots[4:8]); s < 40 {
		t.Errorf("perubahan 46%% hanya setinggi %.1f px", s)
	}
	if got := len(elems(svg, "circle", "sweep-dot default")); got != 2 {
		t.Errorf("titik bawaan = %d, mau 2 (panel ketiga tidak punya x = 0,6)", got)
	}
	if !strings.Contains(svg, "4,03 → 5,88") || !strings.Contains(svg, RpShort(19_183_000, "id")+" → "+RpShort(19_181_000, "id")) {
		t.Error("label perubahan tidak tertulis")
	}
	if !strings.Contains(svg, "1,00 → 2,00") {
		t.Error("tanpa format harus memakai dua desimal")
	}
	checkSVG(t, "SweepPanels kosong", string(SweepPanels(nil, "rho", 0, panels, "Judul", "Deskripsi", "en")))
	checkSVG(t, "SweepPanels x sama", string(SweepPanels([]float64{1, 1}, "rho", 1, panels[1:2], "Judul", "Deskripsi", "en")))
	checkSVG(t, "SweepPanels nol", string(SweepPanels(xs, "rho", 0, []SweepPanel{{Title: "nol", Values: []float64{0, 0, 0, 0}}, {Title: "kosong"}}, "Judul", "Deskripsi", "id")))
}

func TestLoopProbIsADistribution(t *testing.T) {
	for _, p := range []float64{0, 0.25, 0.3, 0.9} {
		sum := 0.0
		for k := 0; k <= 4; k++ {
			sum += LoopProb(p, k, 4)
		}
		if math.Abs(sum-1) > 1e-12 {
			t.Errorf("p = %v: jumlah peluang %v, mau 1", p, sum)
		}
	}
	if got := LoopProb(0.3, 1, 4); math.Abs(got-0.21) > 1e-12 {
		t.Errorf("P(N=1) = %v, mau 0,21", got)
	}
	if got := LoopProb(0.5, 9, 4); math.Abs(got-0.0625) > 1e-12 {
		t.Errorf("ekor p^4 = %v, mau 0,0625", got)
	}
}

func TestLoopBars(t *testing.T) {
	loops := []LoopBar{{Label: "G1 Regresi", P: 0.3, Analytic: 0.429, Simulated: 0.445, Q90: 1}, {Label: "G2 Uji penetrasi", P: 0.25, Analytic: 0.333, Simulated: 0.33, Q90: 1}}
	svg := string(LoopBars(loops, 4, "id"))
	checkSVG(t, "LoopBars", svg)
	if got := len(elems(svg, "rect", "loop-bar")); got != 5*2+2 {
		t.Errorf("batang = %d, mau 12 (10 batang + 2 legenda)", got)
	}
	for _, want := range []string{"≥ 4", "p 30%", "analitik 0,429", "simulasi 0,445", "90%: ≤ 1", "70%"} {
		if !strings.Contains(svg, want) {
			t.Errorf("tidak memuat %q", want)
		}
	}
	checkSVG(t, "LoopBars en", string(LoopBars(loops, 4, "en")))
	checkSVG(t, "LoopBars kosong", string(LoopBars(nil, 4, "id")))
	checkSVG(t, "LoopBars maxK 0", string(LoopBars(loops, 0, "id")))
}

func TestColumnBarsDrawReferenceAsLegend(t *testing.T) {
	bars := []ColumnBar{{Label: "L0", Value: 2.2e6, Class: "step-0", Note: "Independen"}, {Label: "L4", Value: 3.5e6, Class: "step-4"}, {Label: "Negatif", Value: -1}}
	svg := string(ColumnBars(bars, 2e6, "pada jadwal rencana", true, "Judul", "Deskripsi", "id"))
	checkSVG(t, "ColumnBars", svg)
	rects := elems(svg, "rect", "col-bar")
	if len(rects) != 3 {
		t.Fatalf("kolom = %d, mau 3", len(rects))
	}
	if h := fl(t, rects[2], "height"); h != 0 {
		t.Errorf("nilai negatif harus setinggi nol, dapat %v", h)
	}
	if len(elems(svg, "line", "ref-line target")) != 2 || !strings.Contains(svg, "pada jadwal rencana "+RpShort(2e6, "id")) {
		t.Error("garis acuan dan legendanya harus digambar")
	}
	if !strings.Contains(svg, "Independen") || !strings.Contains(svg, RpShort(3.5e6, "id")) {
		t.Error("catatan atau nilai kolom tidak tertulis")
	}
	noRef := string(ColumnBars(bars[:1], 0, "", false, "Judul", "Deskripsi", "en"))
	if strings.Contains(noRef, "ref-line") || !strings.Contains(noRef, "2,200,000.0") {
		t.Error("tanpa acuan tidak boleh ada garis acuan, dan angka biasa memakai satu desimal")
	}
	checkSVG(t, "ColumnBars nol", string(ColumnBars([]ColumnBar{{Label: "nol"}}, 0, "", true, "Judul", "Deskripsi", "id")))
}

func TestDumbbellRows(t *testing.T) {
	rows := []DumbbellRow{{Label: "R01 Alumni", A: 0.3, B: 0.299}, {Label: "R02 Tertutup", A: 0.4, B: 0, Muted: true, Note: "ditutup: terjadi"}, {Label: "R99 Pasti", A: 0.95, B: 0.97}}
	svg := string(DumbbellRows(rows, "register", "simulasi", "Judul", "Deskripsi", "id"))
	checkSVG(t, "Dumbbell", svg)
	if got := len(elems(svg, "circle", "db-b")); got != 4 {
		t.Errorf("lingkaran penuh = %d, mau 4 (3 baris + legenda)", got)
	}
	if len(elems(svg, "circle", "db-b muted")) != 1 || !strings.Contains(svg, "ditutup: terjadi") || !strings.Contains(svg, `class="db-row muted"`) {
		t.Error("baris tertutup harus pudar dan memakai catatannya")
	}
	if !strings.Contains(svg, "29,9%") {
		t.Error("catatan bawaan harus nilai B")
	}
	if strings.Contains(svg, ">110%<") || !strings.Contains(svg, ">100%<") {
		t.Error("sumbu peluang tidak boleh melewati 100%")
	}
	checkSVG(t, "Dumbbell kosong", string(DumbbellRows(nil, "a", "b", "Judul", "Deskripsi", "en")))
	checkSVG(t, "Dumbbell nol", string(DumbbellRows([]DumbbellRow{{Label: "nol"}}, "a", "b", "Judul", "Deskripsi", "en")))
}

func TestRangeRowsOrderEstimates(t *testing.T) {
	rows := []RangeRow{{Label: "A17 API", Lo: 4, Mode: 6, Hi: 11, Mean: 6.5, Class: "critical"}, {Label: "A08 Desain", Lo: 4, Mode: 5, Hi: 9, Mean: 5.5}}
	svg := string(RangeRows(rows, "hari kerja", "Judul", "Deskripsi", "id"))
	checkSVG(t, "RangeRows", svg)
	ranges := elems(svg, "line", "rr-range")
	if len(ranges) != 2+2 {
		t.Fatalf("garis rentang = %d, mau 4 (2 baris + 2 legenda)", len(ranges))
	}
	modes := elems(svg, "line", "rr-mode")
	for i := range rows {
		r := ranges[2+i] // dua garis pertama adalah legenda
		x1, x2, xm := fl(t, r, "x1"), fl(t, r, "x2"), fl(t, modes[i+1], "x1")
		if !(x1 < xm && xm < x2) {
			t.Errorf("baris %d: M (%v) harus di antara O (%v) dan P (%v)", i, xm, x1, x2)
		}
	}
	if len(elems(svg, "line", "rr-range critical")) != 2 {
		t.Error("baris kritis dan legenda jalur kritis harus berkelas critical")
	}
	if !strings.Contains(svg, "4 · 6 · 11") || !strings.Contains(svg, "hari kerja") {
		t.Error("angka O · M · P atau satuan tidak tertulis")
	}
	checkSVG(t, "RangeRows kosong", string(RangeRows(nil, "hari", "Judul", "Deskripsi", "en")))
}

func TestStackRowsLabelsOnlyWhatFits(t *testing.T) {
	rows := []StackRow{
		{Label: "A19 API panel", Parts: []StackPart{{Value: 12, Class: "seg-carried", Label: "12"}, {Value: 15, Class: "seg-wait", Label: "15"}, {Value: 0, Class: "seg-stretch", Label: "0"}}, Note: "+27"},
		{Label: "A03 Analisis", Parts: []StackPart{{Value: 0.3, Class: "seg-stretch", Label: "label panjang sekali"}}},
	}
	legend := []LegendItem{{"terbawa dari pendahulu dengan nama legenda yang panjang", "seg-carried"}, {"menunggu orang dengan nama legenda yang juga panjang", "seg-wait"}, {"memanjang saat dikerjakan karena paruh waktu atau ujian", "seg-stretch"}}
	svg := string(StackRows(rows, legend, false, "hari kerja", "Judul", "Deskripsi", "id"))
	checkSVG(t, "StackRows", svg)
	if got := len(elems(svg, "rect", "stack-seg")) - len(legend); got != 3 {
		t.Errorf("segmen = %d, mau 3 (segmen nol dilewati)", got)
	}
	if strings.Contains(svg, "label panjang sekali<") {
		t.Error("label yang tidak muat di segmen tidak boleh ditulis")
	}
	if !strings.Contains(svg, ">12<") || !strings.Contains(svg, "+27") || !strings.Contains(svg, ">0<") {
		t.Error("label segmen, catatan, atau jumlah bawaan tidak tertulis")
	}
	// Legenda yang membungkus menambah tinggi kanvas.
	if lr := legendRows(legend, 12, 868); lr < 2 {
		t.Errorf("legenda panjang harus membungkus, dapat %d baris", lr)
	}
	one := string(StackRows(rows, legend[:1], true, "", "Judul", "Deskripsi", "en"))
	if !strings.Contains(one, RpShort(0.3, "en")) {
		t.Error("jumlah rupiah bawaan harus memakai RpShort")
	}
	checkSVG(t, "StackRows kosong", string(StackRows(nil, nil, false, "", "Judul", "Deskripsi", "id")))
}

func TestScatterLabeledDrawsLabelledPointsOnTop(t *testing.T) {
	pts := []ScatterPoint{{X: 2, Y: 60000, Label: "A30→A31", Class: "ft-viable"}, {X: 0, Y: 50000, Class: "ft-same", Note: "A16→A17"}, {X: 3, Y: 100000, Label: "A18→A24", Class: "ft-viable"}}
	legend := []LegendItem{{"layak", "ft-viable"}, {"orang yang sama", "ft-same"}}
	svg := string(ScatterLabeled(pts, "hari dihemat", "rework", true, legend, "Judul", "Deskripsi", "id"))
	checkSVG(t, "Scatter", svg)
	attrsWithin(t, "Scatter", svg, 880, 390)
	circles := elems(svg, "circle", "sc-pt")
	if len(circles) != len(pts)+len(legend) {
		t.Fatalf("lingkaran = %d, mau %d", len(circles), len(pts)+len(legend))
	}
	// Titik tanpa label digambar pertama (setelah legenda).
	if !strings.Contains(circles[len(legend)]["class"], "ft-same") {
		t.Errorf("titik tanpa label harus digambar paling bawah, dapat %q", circles[len(legend)]["class"])
	}
	if len(elems(svg, "text", "sc-label")) != 2 {
		t.Error("hanya titik berlabel yang diberi teks")
	}
	if !strings.Contains(svg, "<title>A16→A17</title>") {
		t.Error("catatan titik harus menjadi tooltip")
	}
	checkSVG(t, "Scatter nol", string(ScatterLabeled([]ScatterPoint{{X: 1}}, "x", "y", false, nil, "Judul", "Deskripsi", "en")))
	checkSVG(t, "Scatter kosong", string(ScatterLabeled(nil, "x", "y", false, nil, "Judul", "Deskripsi", "en")))
}

func TestDivergingRowsFollowSigns(t *testing.T) {
	neg := []DivRow{{Label: "A16 API autentikasi", Value: -70000, Note: "CPI 0,80"}, {Label: "A13 Review", Value: -8000}}
	svg := string(DivergingRows(neg, true, "boros", "hemat", "Judul", "Deskripsi", "id"))
	checkSVG(t, "Diverging", svg)
	zero := elems(svg, "line", "ref-line divider")
	if len(zero) != 1 || fl(t, zero[0], "x1") < 700 {
		t.Errorf("tanpa nilai positif garis nol harus di kanan, dapat %v", zero)
	}
	if len(elems(svg, "rect", "div-bar neg")) != 2 || len(elems(svg, "rect", "div-bar pos")) != 0 {
		t.Error("kelas batang harus mengikuti tanda")
	}
	if len(elems(svg, "text", "div-label-in")) == 0 {
		t.Error("label batang terpanjang harus masuk ke dalam batang agar tidak menimpa nama baris")
	}
	for _, tx := range elems(svg, "text", "bar-value") {
		if tx["text-anchor"] == "end" && fl(t, tx, "x") < 270 {
			t.Errorf("label berakhir di area nama baris: x=%v", tx["x"])
		}
	}
	if !strings.Contains(svg, "-"+RpShort(70000, "id")+" · CPI 0,80") {
		t.Error("nilai bertanda dan catatan tidak tertulis")
	}
	mixed := string(DivergingRows([]DivRow{{Label: "hemat", Value: 5}, {Label: "boros", Value: -2}, {Label: "nol", Value: 0}}, false, "kurang", "lebih", "Judul", "Deskripsi", "en"))
	checkSVG(t, "Diverging campur", mixed)
	if !strings.Contains(mixed, "+5.0") || len(elems(mixed, "rect", "div-bar pos")) != 2 {
		t.Error("nilai positif dan nol digambar ke kanan dengan tanda")
	}
	pos := string(DivergingRows([]DivRow{{Label: "hemat", Value: 1000, Note: "catatan yang sangat panjang sekali sampai melewati tepi kanan kanvas"}}, false, "a", "b", "Judul", "Deskripsi", "en"))
	if len(elems(pos, "text", "div-label-in")) != 1 {
		t.Error("label positif yang mentok tepi kanan harus masuk ke dalam batang")
	}
	checkSVG(t, "Diverging kosong", string(DivergingRows(nil, true, "a", "b", "Judul", "Deskripsi", "id")))
}

func TestEventTimelineKeepsTimeScaleAndLanes(t *testing.T) {
	events := []TimelineEvent{
		{Date: "2025-01-31", Label: "Dampak pertama", Class: "kind-dampak"},
		{Date: "2018-01-01", Label: "Perpres 40/2018 menetapkan program PSIAP dengan label yang panjang", Class: "kind-regulasi"},
		{Date: "2025-01-01", Label: "Go-live serentak", Class: "kind-cutover"},
		{Date: "2025-02-15", Label: "Tenggat faktur", Class: "kind-dampak"},
		{Date: "bukan-tanggal", Label: "diabaikan"},
	}
	svg := string(EventTimeline(events, "Linimasa", "Deskripsi", "id"))
	checkSVG(t, "EventTimeline", svg)
	dots := elems(svg, "circle", "tl-dot")
	if len(dots) != 4 {
		t.Fatalf("titik = %d, mau 4 (tanggal rusak dilewati)", len(dots))
	}
	for i := 1; i < len(dots); i++ {
		if fl(t, dots[i], "cx") < fl(t, dots[i-1], "cx") {
			t.Error("kejadian harus terurut menurut tanggal")
		}
	}
	// Skala waktu: jeda 2018 -> 2025 jauh lebih lebar dari jeda Januari -> Februari 2025.
	if gapYears, gapWeeks := fl(t, dots[1], "cx")-fl(t, dots[0], "cx"), fl(t, dots[3], "cx")-fl(t, dots[2], "cx"); gapYears < 20*gapWeeks {
		t.Errorf("jeda tujuh tahun (%v px) harus jauh lebih lebar dari jeda enam minggu (%v px)", gapYears, gapWeeks)
	}
	for _, y := range []string{">2018<", ">2025<", ">2026<"} {
		if !strings.Contains(svg, y) {
			t.Errorf("penanda tahun %s hilang", y)
		}
	}
	if strings.Contains(svg, "diabaikan") {
		t.Error("kejadian bertanggal rusak tidak boleh digambar")
	}
	// Tiga kejadian berdekatan tidak boleh berbagi lajur: posisi tegak label tanggal berbeda.
	ys := map[string]bool{}
	for _, tx := range elems(svg, "text", "tlc-date") {
		x := fl(t, tx, "x")
		if x > 600 {
			ys[tx["y"]] = true
		}
	}
	if len(ys) < 3 {
		t.Errorf("label kejadian 2025 berbagi lajur: %v", ys)
	}
	if !strings.Contains(svg, "Perpres 40/2018 menetapkan program PSIAP dengan…") && !strings.Contains(svg, "Perpres 40/2018 menetapkan program PSIAP de") {
		t.Error("label panjang harus dipotong")
	}
	checkSVG(t, "EventTimeline en", string(EventTimeline(events, "Timeline", "Description", "en")))
	checkSVG(t, "EventTimeline kosong", string(EventTimeline([]TimelineEvent{{Date: "x"}}, "Linimasa", "Deskripsi", "id")))
}
