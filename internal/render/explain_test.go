package render

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// checkSVG memeriksa syarat yang berlaku untuk semua grafik: dokumen utuh,
// beraksesibilitas, tanpa koordinat rusak, dan tanpa warna tertanam.
func checkSVG(t *testing.T, name, svg string) {
	t.Helper()
	if !strings.HasPrefix(svg, "<svg ") || !strings.HasSuffix(svg, "</svg>") {
		t.Errorf("%s: bukan dokumen SVG utuh", name)
	}
	if !strings.Contains(svg, `role="img"`) {
		t.Errorf("%s: tidak punya role=\"img\"", name)
	}
	if !strings.Contains(svg, "<title") || !strings.Contains(svg, "<desc>") {
		t.Errorf("%s: tidak punya judul atau deskripsi aksesibilitas", name)
	}
	if strings.Contains(svg, "NaN") || strings.Contains(svg, "Infinity") {
		t.Errorf("%s: memuat NaN atau Infinity", name)
	}
	if strings.Contains(svg, `fill="#`) || strings.Contains(svg, `stroke="#`) {
		t.Errorf("%s: memuat warna heksadesimal langsung", name)
	}
	if strings.Count(svg, "<g") != strings.Count(svg, "</g>") {
		t.Errorf("%s: grup <g> tidak berpasangan (%d buka, %d tutup)", name, strings.Count(svg, "<g"), strings.Count(svg, "</g>"))
	}
}

var numAttr = regexp.MustCompile(`\b(x|y|x1|x2|y1|y2|cx|cy|width|height)="(-?[0-9.]+)"`)

// attrsWithin memastikan setiap koordinat berada di dalam kanvas (dengan
// sedikit kelonggaran untuk garis tepi), supaya label tidak terpotong.
func attrsWithin(t *testing.T, name, svg string, w, h float64) {
	t.Helper()
	for _, m := range numAttr.FindAllStringSubmatch(svg, -1) {
		v, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			t.Errorf("%s: atribut %s bukan angka: %s", name, m[1], m[2])
			continue
		}
		limit := w
		if strings.HasPrefix(m[1], "y") || m[1] == "height" || m[1] == "cy" {
			limit = h
		}
		if v < -1 || v > limit+1 {
			t.Errorf("%s: atribut %s=%s di luar kanvas [0, %v]", name, m[1], m[2], limit)
		}
	}
}

func testCal() *workcal.Calendar { return workcal.MustNew(model.ProjectCharter.StartDate, 220) }

func TestSpreadYKeepsOrderGapAndBounds(t *testing.T) {
	in := []float64{50, 52, 51, 200}
	out := spreadY(in, 14, 40, 120)
	if len(out) != len(in) {
		t.Fatalf("panjang berubah: %v", out)
	}
	// Urutan relatif dan jarak minimum di antara tiga label yang berdempet.
	if !(out[0] < out[2] && out[2] < out[1]) {
		t.Errorf("urutan label berubah: %v", out)
	}
	for _, pair := range [][2]int{{0, 2}, {2, 1}} {
		if out[pair[1]]-out[pair[0]] < 14-1e-9 {
			t.Errorf("jarak label %d-%d kurang dari 14: %v", pair[0], pair[1], out)
		}
	}
	for _, v := range out {
		if v > 120+1e-9 || v < 40-1e-9 {
			t.Errorf("label keluar dari batas [40,120]: %v", out)
		}
	}
	if got := spreadY(nil, 14, 0, 10); len(got) != 0 {
		t.Errorf("masukan kosong harus kosong, dapat %v", got)
	}
}

func TestLabelPlacerAvoidsCollisions(t *testing.T) {
	lp := &labelPlacer{w: 400, h: 300}
	type box struct{ x1, y1, x2, y2 float64 }
	var placed []box
	// Delapan label pada titik yang hampir sama: setiap label harus mendapat
	// tempat sendiri selama masih ada kandidat yang muat.
	for i := 0; i < 8; i++ {
		lines := []string{"Opsi percepatan", "Rp 1,2 juta"}
		x, y, anchor := lp.place(200, 150+float64(i), lines, 11)
		w := textWidth("Opsi percepatan", 11)
		x1 := x
		if anchor == "end" {
			x1 = x - w
		}
		b := box{x1, y - 11, x1 + w, y - 11 + 2*14}
		for _, o := range placed {
			if b.x1 < o.x2 && b.x2 > o.x1 && b.y1 < o.y2 && b.y2 > o.y1 {
				t.Errorf("label %d bertabrakan dengan label sebelumnya: %+v vs %+v", i, b, o)
			}
		}
		if b.x1 < 0 || b.x2 > 400 || b.y1 < 0 || b.y2 > 300 {
			t.Errorf("label %d keluar kanvas: %+v", i, b)
		}
		placed = append(placed, b)
	}
}

func TestDateHelpers(t *testing.T) {
	cal := testCal()
	ticks := dateTicks(cal, -5, 400, 8, "id")
	if len(ticks) == 0 {
		t.Fatal("tidak ada penanda tanggal")
	}
	for _, tk := range ticks {
		if tk.Value < 1 || int(math.Round(tk.Value))-1 >= cal.Len() {
			t.Errorf("penanda di luar kalender: %v", tk)
		}
	}
	if got := dateTicks(nil, 1, 50, 5, "id"); len(got) != 0 {
		t.Errorf("tanpa kalender tidak boleh ada penanda tanggal: %v", got)
	}
	if got := finishLabel(cal, 1, "id"); got != workcal.FormatDate(cal.ISOAt(0), "id") {
		t.Errorf("hari ke-1 harus tanggal mulai, dapat %q", got)
	}
	if got := finishLabel(nil, 97, "id"); got != "97" {
		t.Errorf("tanpa kalender harus angka hari, dapat %q", got)
	}
	if got := finishLabel(cal, 10000, "en"); got != "10,000" {
		t.Errorf("di luar kalender harus angka hari, dapat %q", got)
	}
}

func TestDateDotsWritesDatesOnThePoints(t *testing.T) {
	cal := testCal()
	items := []DateItem{
		{Label: "Piagam", Status: "tidak berlaku", Day: 85, Note: "Rp 16 juta", Class: "invalid"},
		{Label: "Komitmen", Status: "berlaku", Day: 123, Class: "inforce"},
		{Label: "Di tepi kanan", Day: 219, Class: "neutral"},
	}
	refs := []DateRef{{Day: 40, Label: "tanggal data", Class: "data"}}
	for _, lang := range []string{"id", "en"} {
		svg := string(DateDots(items, refs, cal, "Judul", "Deskripsi", lang))
		checkSVG(t, "DateDots "+lang, svg)
		attrsWithin(t, "DateDots "+lang, svg, 880, 70+40*3)
		for _, it := range items {
			if !strings.Contains(svg, finishLabel(cal, it.Day, lang)) {
				t.Errorf("%s: tanggal selesai %s tidak tertulis", lang, it.Label)
			}
		}
		if strings.Count(svg, `class="dd-dot`) != len(items) {
			t.Errorf("%s: jumlah titik salah", lang)
		}
		if !strings.Contains(svg, "Rp 16 juta") || !strings.Contains(svg, "tidak berlaku") {
			t.Errorf("%s: catatan dan status tidak tertulis", lang)
		}
	}
	checkSVG(t, "DateDots kosong", string(DateDots(nil, nil, cal, "Judul", "Deskripsi", "id")))
}

func TestOptionMapLinksOptionsToBase(t *testing.T) {
	cal := testCal()
	pts := []OptionPoint{
		{Label: "Tanpa percepatan", Day: 123, Budget: 21e6, DayLo: 120, DayHi: 126, BudgetLo: 20.5e6, BudgetHi: 21.6e6, Class: "base"},
		{Label: "Lembur", Day: 118, Budget: 21.8e6, Note: "Rp 0,8 juta", Class: "cheapest"},
		{Label: "Tambah orang", Day: 112, Budget: 23e6, Class: "fastest"},
	}
	svg := string(OptionMap(pts, cal, "id"))
	checkSVG(t, "OptionMap", svg)
	if got := strings.Count(svg, `class="om-link"`); got != 2 {
		t.Errorf("harus ada 2 panah dari opsi dasar, dapat %d", got)
	}
	if got := strings.Count(svg, `class="om-ci"`); got != 2 {
		t.Errorf("hanya opsi dasar yang punya interval (2 garis), dapat %d", got)
	}
	for _, p := range pts {
		if !strings.Contains(svg, ">"+p.Label+"<") {
			t.Errorf("label %q tidak tertulis", p.Label)
		}
	}
	// Titik tunggal dan titik tanpa nilai positif tidak boleh merusak domain.
	checkSVG(t, "OptionMap tunggal", string(OptionMap(pts[:1], cal, "en")))
	zero := string(OptionMap([]OptionPoint{{Label: "nol"}}, cal, "id"))
	checkSVG(t, "OptionMap nol", zero)
	if strings.Contains(zero, "om-pt") {
		t.Error("opsi tanpa hari dan anggaran positif tidak boleh digambar")
	}
	checkSVG(t, "OptionMap kosong", string(OptionMap(nil, cal, "id")))
}

func TestValueBandsChartDrawsEnvelopeAndEdges(t *testing.T) {
	lines := []BenefitLine{
		{Label: "Tanpa percepatan", Days: 0, Extra: 0, Class: "opt-tanpa"},
		{Label: "Lembur", Days: 5, Extra: 600000, Class: "opt-lembur"},
		{Label: "Tambah orang", Days: 11, Extra: 2.4e6, Class: "opt-tambah"},
	}
	bands := []BandSpan{
		{From: 0, To: 120000, Label: "Tanpa percepatan", Class: "opt-tanpa"},
		{From: 120000, To: 300000, Label: "Lembur", Class: "opt-lembur", EdgeLo: 100000, EdgeHi: 140000},
		{From: 300000, To: math.Inf(1), Label: "Tambah orang", Class: "opt-tambah", EdgeLo: 260000, EdgeHi: 350000},
	}
	for _, lang := range []string{"id", "en"} {
		svg := string(ValueBandsChart(lines, bands, lang))
		checkSVG(t, "ValueBands "+lang, svg)
		if strings.Count(svg, `class="vb-band `) != len(bands) {
			t.Errorf("%s: jumlah pita salah", lang)
		}
		if strings.Count(svg, `class="vb-edge"`) != 2 {
			t.Errorf("%s: batas pertama di nol tidak boleh digambar; harus 2 batas", lang)
		}
		if strings.Count(svg, `class="vb-ci"`) != 6 {
			t.Errorf("%s: dua interval x (garis + dua ujung) = 6, dapat %d", lang, strings.Count(svg, `class="vb-ci"`))
		}
		if !strings.Contains(svg, "vb-envelope") {
			t.Errorf("%s: selubung atas tidak digambar", lang)
		}
		if !strings.Contains(svg, Rp(120000, lang)) {
			t.Errorf("%s: nilai batas tidak tertulis", lang)
		}
	}
	checkSVG(t, "ValueBands kosong", string(ValueBandsChart(nil, bands, "id")))
	checkSVG(t, "ValueBands datar", string(ValueBandsChart([]BenefitLine{{Label: "nol"}}, bands[:1], "id")))
}

func TestScenarioStripsMarksAgreement(t *testing.T) {
	rows := []StripRow{
		{Label: "Dasar", Base: true, Same: true, Bands: []BandSpan{{From: 0, To: 200000, Class: "opt-tanpa"}, {From: 200000, To: math.Inf(1), Class: "opt-lembur"}}},
		{Label: "Pesimistis", Same: false, Bands: []BandSpan{{From: 0, To: math.Inf(1), Class: "opt-tanpa"}}},
		{Label: "Di luar sumbu", Same: true, Bands: []BandSpan{{From: 900000, To: math.Inf(1), Class: "opt-tambah"}}},
	}
	legend := []LegendItem{{"Tanpa percepatan", "opt-tanpa"}, {"Lembur sah dengan nama yang sangat panjang sekali", "opt-lembur"}, {"Tambah orang", "opt-tambah"}, {"Tambah orang dan lembur", "opt-tambah-lembur"}}
	svg := string(ScenarioStrips(rows, 500000, legend, "id"))
	checkSVG(t, "ScenarioStrips", svg)
	if strings.Count(svg, "✓") != 2 || strings.Count(svg, "✗") != 1 {
		t.Errorf("tanda kesesuaian salah: %d ✓, %d ✗", strings.Count(svg, "✓"), strings.Count(svg, "✗"))
	}
	// Pita yang mulai di luar sumbu tidak digambar; legenda tetap 4.
	if got := strings.Count(svg, `class="strip-seg `); got != 3+len(legend) {
		t.Errorf("jumlah segmen = %d, mau %d", got, 3+len(legend))
	}
	if !strings.Contains(svg, `class="strip-row base"`) {
		t.Error("baris dasar harus ditandai")
	}
	checkSVG(t, "ScenarioStrips kosong", string(ScenarioStrips(rows, 0, legend, "en")))
}

func TestBridgeChartRunsTheTotal(t *testing.T) {
	steps := []BridgeStep{
		{Label: "Pagu piagam", Value: 16e6, Kind: "base"},
		{Label: "Kinerja biaya", Value: 3e6, Kind: "delta", Note: "CPI 0,92"},
		{Label: "Penghematan", Value: -1e6, Kind: "delta"},
		{Label: "Permintaan", Value: 18e6, Kind: "total"},
	}
	svg := string(BridgeChart(steps, "Jembatan", "Deskripsi", "id"))
	checkSVG(t, "Bridge", svg)
	if !strings.Contains(svg, "bridge-bar delta down") {
		t.Error("delta negatif harus berkelas down")
	}
	if !strings.Contains(svg, "+"+RpShort(3e6, "id")) {
		t.Error("delta positif harus bertanda +")
	}
	if strings.Count(svg, `class="bridge-link"`) != len(steps)-1 {
		t.Error("setiap langkah setelah yang pertama harus tersambung")
	}
	if !strings.Contains(svg, "CPI 0,92") {
		t.Error("catatan langkah tidak tertulis")
	}
	checkSVG(t, "Bridge kosong", string(BridgeChart(nil, "Jembatan", "Deskripsi", "id")))
}

func TestOvertimeCurveChart(t *testing.T) {
	cal := testCal()
	pts := []CurvePoint{{Day: 121, Cost: 200000, Net: 90000}, {Day: 119, Cost: 520000, Net: 300000, Note: "BE"}, {Day: 117, Cost: 1.1e6, Net: 800000}}
	compare := []CurvePoint{{Day: 121, Cost: 150000}, {Day: 117, Cost: 700000}}
	for _, lang := range []string{"id", "en"} {
		svg := string(OvertimeCurveChart(123, pts, compare, "crashing CPM", cal, lang))
		checkSVG(t, "OvertimeCurve "+lang, svg)
		if strings.Count(svg, `class="oc-dot"`) != len(pts)+1 {
			t.Errorf("%s: titik harus termasuk jadwal tanpa lembur", lang)
		}
		if !strings.Contains(svg, "oc-compare") || !strings.Contains(svg, "crashing CPM") {
			t.Errorf("%s: kurva pembanding tidak digambar", lang)
		}
		floor := tr(lang, "lantai: ", "floor: ") + Num(117, 0, lang)
		if !strings.Contains(svg, floor) {
			t.Errorf("%s: label lantai %q tidak ada", lang, floor)
		}
	}
	noCompare := string(OvertimeCurveChart(123, pts[:1], nil, "", cal, "id"))
	checkSVG(t, "OvertimeCurve tanpa pembanding", noCompare)
	if strings.Contains(noCompare, "oc-compare") {
		t.Error("tanpa pembanding tidak boleh ada kurva pembanding")
	}
	checkSVG(t, "OvertimeCurve kosong", string(OvertimeCurveChart(123, nil, nil, "", cal, "id")))
}

func TestCredibilityPanelsShowNoiseOnlyWhenKnown(t *testing.T) {
	rows := []CredRow{
		{Label: "Durasi", Prior: 1, Observed: 1.02, Noise: 0.08, Z: 0, Blended: 1, Meaning: "masih derau"},
		{Label: "Biaya harian", Prior: 1, Observed: 1.09, Noise: 0.02, Z: 0.85, Blended: 1.0765},
		{Label: "Tanpa derau", Prior: 0.5, Observed: 0.5, Noise: 0, Z: 1, Blended: 0.5},
	}
	svg := string(CredibilityPanels(rows, "id"))
	checkSVG(t, "Credibility", svg)
	if got := strings.Count(svg, `class="cred-noise"`); got != 2 {
		t.Errorf("pita derau hanya untuk baris ber-derau: %d", got)
	}
	// Panah hanya bila nilai yang dipakai bergeser dari rencana.
	if got := strings.Count(svg, `class="cred-arrow"`); got != 1 {
		t.Errorf("panah pergeseran = %d, mau 1", got)
	}
	if !strings.Contains(svg, "Z = 85%") || !strings.Contains(svg, "masih derau") {
		t.Error("bobot Z atau arti tidak tertulis")
	}
	checkSVG(t, "Credibility en", string(CredibilityPanels(rows, "en")))
	checkSVG(t, "Credibility kosong", string(CredibilityPanels(nil, "id")))
}

func TestFrontierLineTrimsFlatTail(t *testing.T) {
	cal := testCal()
	var days, budgets []float64
	for d := 100; d <= 180; d++ {
		days = append(days, float64(d))
		b := 20e6
		if d < 130 {
			b += float64(130-d) * 200000
		}
		budgets = append(budgets, b)
	}
	marks := []FrontierMark{{Day: 123, Budget: 21.4e6, Label: "Komitmen", Class: "commit"}, {Day: 100, Budget: 26e6, Label: "Tercepat", Class: "fast"}}
	svg := string(FrontierLine(days, budgets, marks, cal, "id"))
	checkSVG(t, "Frontier", svg)
	// Ekor datar setelah hari ke-130 dipotong: sumbu tidak boleh mencapai 180.
	if strings.Contains(svg, finishLabel(cal, 180, "id")) {
		t.Error("ekor datar tidak dipotong")
	}
	if strings.Count(svg, `class="fl-mark `) != len(marks) {
		t.Error("penanda tidak lengkap")
	}
	// Deret yang tidak sama panjang dan deret datar tidak boleh panik.
	checkSVG(t, "Frontier timpang", string(FrontierLine(days, budgets[:3], nil, cal, "en")))
	checkSVG(t, "Frontier datar", string(FrontierLine([]float64{1, 2, 3}, []float64{5, 5, 5}, nil, cal, "id")))
	checkSVG(t, "Frontier kosong", string(FrontierLine(nil, nil, nil, cal, "id")))
}

func TestCapacityTimelineShadesByCapacityLeft(t *testing.T) {
	cal := testCal()
	windows := []WindowSpan{
		{FromDay: 30, ToDay: 39, Label: "UAS", Factor: 0.4, Roles: "semua peran"},
		{FromDay: 60, ToDay: 64, Label: "Libur", Factor: 0, Roles: "BE", Assumed: true},
		{FromDay: 70, ToDay: 71, Label: "Aneh", Factor: -0.5},
		{FromDay: 80, ToDay: 80, Label: "Penuh", Factor: 1.2},
	}
	svg := string(CapacityTimeline(windows, 400, 150, 45, cal, "id"))
	checkSVG(t, "CapacityTimeline", svg)
	for _, want := range []string{"cap-bar cap-3", "cap-bar cap-5 assumed", "cap-bar cap-5", "cap-bar cap-1"} {
		if !strings.Contains(svg, want) {
			t.Errorf("kelas %q tidak ada", want)
		}
	}
	if strings.Contains(svg, "cap-6") || strings.Contains(svg, "cap-0") {
		t.Error("kelas kepekatan harus dibatasi 1..5")
	}
	attrsWithin(t, "CapacityTimeline", svg, 880, 96+38*4)
	if !strings.Contains(svg, "sisa 40%") {
		t.Error("sisa kapasitas tidak tertulis")
	}
	// Tanggal data di luar sumbu tidak digambar.
	if strings.Contains(string(CapacityTimeline(windows, 0, 150, 500, cal, "en")), "status-line") {
		t.Error("tanggal data di luar sumbu tidak boleh digambar")
	}
	checkSVG(t, "CapacityTimeline kosong", string(CapacityTimeline(nil, 0, 150, 45, cal, "id")))
}

func TestIndexDotsHandlesMissingIndices(t *testing.T) {
	rows := []IndexRow{{Label: "Analisis", SPI: 0.92, CPI: 0.95}, {Label: "Implementasi", SPI: 0}, {Label: "Hanya SPI", SPI: 1.3}}
	svg := string(IndexDots(rows, "id"))
	checkSVG(t, "IndexDots", svg)
	if !strings.Contains(svg, "belum dimulai") {
		t.Error("fase tanpa indeks harus ditandai belum dimulai")
	}
	if strings.Count(svg, `class="idx-spi"`) != 3 { // dua baris + legenda
		t.Errorf("lingkaran SPI = %d, mau 3", strings.Count(svg, `class="idx-spi"`))
	}
	if strings.Count(svg, `class="idx-cpi"`) != 2 { // satu baris + legenda
		t.Errorf("kotak CPI = %d, mau 2", strings.Count(svg, `class="idx-cpi"`))
	}
	checkSVG(t, "IndexDots en", string(IndexDots(rows, "en")))
}

func TestPairBarsDescribesDirection(t *testing.T) {
	labels := []string{"Teknis", "Jadwal", "Baru"}
	svg := string(PairBars(labels, []float64{4e6, 1e6, 0}, []float64{1e6, 1.5e6, 2e5}, "inheren", "residual", "Judul", "Deskripsi", "id"))
	checkSVG(t, "PairBars", svg)
	if !strings.Contains(svg, "75% turun") {
		t.Error("penurunan tidak tertulis")
	}
	if !strings.Contains(svg, "50% naik") {
		t.Error("kenaikan harus ditulis naik, bukan turun negatif")
	}
	if strings.Contains(svg, "-") && strings.Contains(svg, "% turun") && strings.Contains(svg, "(-") {
		t.Error("persen turun negatif ditemukan")
	}
	// Deret yang lebih pendek dari label tidak boleh panik.
	checkSVG(t, "PairBars timpang", string(PairBars(labels, []float64{1}, []float64{1, 2}, "a", "b", "Judul", "Deskripsi", "en")))
	checkSVG(t, "PairBars nol", string(PairBars(labels[:1], []float64{0}, []float64{0}, "a", "b", "Judul", "Deskripsi", "id")))
}

// TestLabelPlacerFallbackStaysOnCanvas: bila tidak ada tempat bebas, label
// tetap di dalam kanvas dan memilih tumpang-tindih terkecil, bukan posisi
// pertama yang mungkin menimpa label lain sepenuhnya.
func TestLabelPlacerFallbackStaysOnCanvas(t *testing.T) {
	lp := &labelPlacer{w: 160, h: 70}
	for i := 0; i < 6; i++ {
		x, y, anchor := lp.place(80, 35, []string{"label"}, 11)
		w := textWidth("label", 11)
		x1 := x
		if anchor == "end" {
			x1 = x - w
		}
		if x1 < 0 || x1+w > 160 || y-11 < 0 || y+3 > 70 {
			t.Errorf("label %d keluar kanvas: x=%v y=%v anchor=%s", i, x, y, anchor)
		}
	}
	// Kanvas yang terlalu kecil untuk kandidat mana pun tetap mengembalikan
	// posisi default tanpa panik.
	tiny := &labelPlacer{w: 10, h: 10}
	if x, y, a := tiny.place(5, 5, []string{"panjang sekali"}, 11); x != 17 || y != -3 || a != "start" {
		t.Errorf("fallback kanvas mungil = (%v, %v, %s)", x, y, a)
	}
}

// TestLabelPlacerPrefersCandidatesOffLines: garis yang didaftarkan sebagai
// rintangan membuat label memilih kandidat lain yang bebas garis.
func TestLabelPlacerPrefersCandidatesOffLines(t *testing.T) {
	free := &labelPlacer{w: 400, h: 300}
	_, y0, a0 := free.place(200, 150, []string{"label"}, 11)
	if y0 != 142 || a0 != "start" {
		t.Fatalf("tanpa rintangan kandidat pertama (kanan atas) harus dipakai, dapat y=%v %s", y0, a0)
	}
	lined := &labelPlacer{w: 400, h: 300}
	// Garis mendatar tepat melewati kandidat kanan atas.
	lined.avoidSegment(150, 136, 300, 136)
	x, y, anchor := lined.place(200, 150, []string{"label"}, 11)
	w := textWidth("label", 11)
	x1 := x
	if anchor == "end" {
		x1 = x - w
	}
	if y-13 <= 138 && y-11+14 >= 134 && x1 < 300 && x1+w > 150 {
		t.Errorf("label tetap ditempatkan di atas garis: x=%v y=%v %s", x, y, anchor)
	}
	if len(lined.boxes) < 25 {
		t.Errorf("garis 150 px harus menjadi paling sedikit 25 rintangan, dapat %d", len(lined.boxes))
	}
	// Garis nol panjang tetap satu rintangan, tanpa pembagian nol.
	dot := &labelPlacer{w: 10, h: 10}
	dot.avoidSegment(5, 5, 5, 5)
	if len(dot.boxes) != 1 {
		t.Errorf("garis nol panjang = %d rintangan, mau 1", len(dot.boxes))
	}
}
