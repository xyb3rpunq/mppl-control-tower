package render_test

import (
	"math"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// TestNumberFormatFollowsLocale menjaga hal yang tampak sepele tetapi bisa
// salah seribu kali lipat: "Rp 1,338" dan "Rp 1.338" bermakna sangat berbeda.
func TestNumberFormatFollowsLocale(t *testing.T) {
	cases := []struct {
		v      float64
		dec    int
		id, en string
	}{
		{1234567, 0, "1.234.567", "1,234,567"},
		{1234.5, 1, "1.234,5", "1,234.5"},
		{0.915, 3, "0,915", "0.915"},
		{-2500, 0, "-2.500", "-2,500"},
		{0, 0, "0", "0"},
	}
	for _, c := range cases {
		if got := render.Num(c.v, c.dec, "id"); got != c.id {
			t.Errorf("Num(%v, %d, id) = %q, mau %q", c.v, c.dec, got, c.id)
		}
		if got := render.Num(c.v, c.dec, "en"); got != c.en {
			t.Errorf("Num(%v, %d, en) = %q, mau %q", c.v, c.dec, got, c.en)
		}
	}
}

func TestCurrencyFormat(t *testing.T) {
	if got := render.Rp(14832000, "id"); got != "Rp 14.832.000" {
		t.Errorf("Rp(id) = %q", got)
	}
	if got := render.Rp(14832000, "en"); got != "IDR 14,832,000" {
		t.Errorf("Rp(en) = %q", got)
	}
	if got := render.Rp(-500000, "id"); got != "-Rp 500.000" {
		t.Errorf("rupiah negatif = %q", got)
	}
}

func TestShortCurrencyPicksRightUnit(t *testing.T) {
	cases := map[float64]string{
		1_338_300_000_000: "Rp 1,34 triliun",
		27_301_947_889:    "Rp 27,30 miliar",
		14_832_000:        "Rp 14,83 juta",
		500_000:           "Rp 500,00 ribu",
		250:               "Rp 250",
	}
	for v, want := range cases {
		if got := render.RpShort(v, "id"); got != want {
			t.Errorf("RpShort(%v) = %q, mau %q", v, got, want)
		}
	}
}

func TestSignedAlwaysShowsDirection(t *testing.T) {
	if got := render.Signed(-534286, "id"); !strings.HasPrefix(got, "-") {
		t.Errorf("varians negatif harus diawali tanda minus, dapat %q", got)
	}
	if got := render.Signed(534286, "id"); !strings.HasPrefix(got, "+") {
		t.Errorf("varians positif harus diawali tanda plus, dapat %q", got)
	}
	if got := render.Signed(0, "id"); strings.ContainsAny(got, "+-") {
		t.Errorf("nol tidak boleh bertanda, dapat %q", got)
	}
}

func TestPercentAndIndex(t *testing.T) {
	if got := render.Pct(0.9153, 2, "id"); got != "91,53%" {
		t.Errorf("Pct = %q", got)
	}
	if got := render.Index(0.918034, "id"); got != "0,918" {
		t.Errorf("Index = %q", got)
	}
}

func TestNonFiniteValuesDegradeGracefully(t *testing.T) {
	inf := math.Inf(1)
	nan := math.NaN()
	for _, v := range []float64{inf, -inf, nan} {
		if got := render.Num(v, 2, "id"); got != "-" {
			t.Errorf("Num(%v) = %q, mau \"-\"", v, got)
		}
		if got := render.Pct(v, 1, "id"); got != "-" {
			t.Errorf("Pct(%v) = %q, mau \"-\"", v, got)
		}
	}
}

func TestNiceTicksProduceRoundNumbers(t *testing.T) {
	ticks := render.NiceTicks(0, 14832000, 6)
	if len(ticks) < 3 {
		t.Fatalf("terlalu sedikit penanda: %v", ticks)
	}
	for i := 1; i < len(ticks); i++ {
		if ticks[i] <= ticks[i-1] {
			t.Errorf("penanda tidak menaik: %v", ticks)
		}
	}
	if ticks[0] < 0 {
		t.Errorf("penanda pertama negatif pada domain non-negatif: %v", ticks[0])
	}
	// Domain degeneratif tidak boleh memicu pembagian nol.
	if got := render.NiceTicks(5, 5, 4); len(got) == 0 {
		t.Error("domain nol lebar harus tetap menghasilkan satu penanda")
	}
}

// TestChartsProduceValidSVG memastikan setiap grafik menghasilkan SVG yang
// utuh dan beraksesibilitas - judul dan deskripsi wajib, karena grafik tanpa
// keduanya adalah kotak kosong bagi pembaca layar.
func TestChartsProduceValidSVG(t *testing.T) {
	plan, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	cal := workcal.MustNew(model.ProjectCharter.StartDate, 200)

	charts := map[string]string{
		"gantt":    string(render.Gantt(model.Activities, plan, cal, 44, "id")),
		"network":  string(render.Network(model.Activities, plan, "id")),
		"org":      string(render.OrgChart(model.Team, model.ProjectCharter.Sponsor, "id")),
		"fishbone": string(render.Fishbone(model.Fishbones[0], "id")),
		"power":    string(render.PowerInterestGrid(model.Stakeholders, "id")),
		"waterfall": string(render.BudgetWaterfall(
			model.BAC(), model.ContingencyReserve, model.ManagementReserve(), model.TotalAuthorised, "id")),
	}
	for name, svg := range charts {
		if !strings.HasPrefix(svg, "<svg ") || !strings.HasSuffix(svg, "</svg>") {
			t.Errorf("%s: bukan dokumen SVG utuh", name)
		}
		if strings.Count(svg, "<svg") != strings.Count(svg, "</svg>") {
			t.Errorf("%s: tag svg tidak berpasangan", name)
		}
		if !strings.Contains(svg, `role="img"`) {
			t.Errorf("%s: tidak punya role=\"img\"", name)
		}
		if !strings.Contains(svg, "<title") || !strings.Contains(svg, "<desc>") {
			t.Errorf("%s: tidak punya judul atau deskripsi aksesibilitas", name)
		}
		if strings.Contains(svg, "NaN") || strings.Contains(svg, "Infinity") {
			t.Errorf("%s: memuat koordinat NaN atau Infinity", name)
		}
		// Warna tidak boleh ditulis langsung di markup; semuanya harus lewat
		// kelas CSS supaya tema gelap bekerja tanpa menggambar ulang.
		if strings.Contains(svg, `fill="#`) || strings.Contains(svg, `stroke="#`) {
			t.Errorf("%s: memuat warna heksadesimal langsung, seharusnya lewat kelas CSS", name)
		}
	}
}

func TestSVGEscapesUserText(t *testing.T) {
	// Nama aktivitas memuat karakter & yang wajib di-escape di dalam SVG.
	plan, _ := schedule.Compute(model.Activities, schedule.Options{})
	svg := string(render.Network(model.Activities, plan, "id"))
	if strings.Contains(svg, "& ") {
		t.Error("ampersand mentah ditemukan di dalam SVG; teks belum di-escape")
	}
	if !strings.Contains(svg, "&amp;") {
		t.Error("tidak ada ampersand ter-escape sama sekali; prasyarat uji mungkin salah")
	}
}

func TestSparklineHandlesShortInput(t *testing.T) {
	if got := render.Sparkline([]float64{1}, 60, 20, ""); got != "" {
		t.Error("sparkline dengan satu titik harus mengembalikan kosong, bukan SVG rusak")
	}
	if got := render.Sparkline([]float64{1, 2, 3}, 60, 20, ""); !strings.Contains(string(got), "<svg") {
		t.Error("sparkline dengan tiga titik seharusnya menghasilkan SVG")
	}
}

func TestWeeksConversion(t *testing.T) {
	if got := render.Weeks(85, "id"); got != "17,0" {
		t.Errorf("Weeks(85) = %q, mau \"17,0\"", got)
	}
}

func TestPercentPointsAndWorkdayLabel(t *testing.T) {
	if got := render.PctPoints(41, 0, "id"); got != "41%" {
		t.Errorf("PctPoints = %q", got)
	}
	if render.WorkdayLabel(12, "id") != "hari ke-12" || render.WorkdayLabel(12, "en") != "day 12" {
		t.Error("WorkdayLabel harus dwibahasa")
	}
}
