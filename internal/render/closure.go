package render

import (
	"fmt"
	"html/template"
	"math"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
)

// TradeOffChart menggambar analisis time-cost trade-off: biaya crashing eksak,
// biaya sewa & langganan, dan jumlah keduanya pada setiap durasi proyek.
// Titik serakah yang lebih mahal dari optimum eksak ditandai tersendiri.
func TradeOffChart(t compress.TradeOff, lang string) template.HTML {
	cv := NewCanvas(880, 360, Padding{Top: 30, Right: 30, Bottom: 52, Left: 96})
	if len(t.Points) == 0 {
		return template.HTML(cv.SVG("tradeoff"))
	}
	maxY := 0.0
	for _, p := range t.Points {
		maxY = math.Max(maxY, math.Max(p.Total, p.CrashCost))
	}
	maxY *= 1.1
	cv.SetDomain(float64(t.ExactMin)-1, float64(t.Normal)+1, 0, maxY).Describe(
		tr(lang, "Kurva time-cost trade-off", "Time-cost trade-off curve"),
		tr(lang,
			"Untuk setiap durasi proyek: biaya crashing minimum menurut pemrograman linear, biaya sewa dan langganan pada solusi biaya total terendah, dan jumlah keduanya. Titik terendah kurva total adalah durasi termurah secara keseluruhan.",
			"For each project duration: the minimum crashing cost from linear programming, the rental and subscription cost at the lowest-total-cost solution, and their sum. The lowest point of the total curve is the cheapest duration overall."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, maxY, 6) {
		yTicks = append(yTicks, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yTicks, true)
	var xTicks []Tick
	for d := t.ExactMin; d <= t.Normal; d += 2 {
		xTicks = append(xTicks, Tick{float64(d), fmt.Sprintf("%d", d)})
	}
	cv.AxisX(xTicks, false)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-12, tr(lang, "durasi proyek (hari kerja) - makin ke kiri makin dipercepat", "project duration (working days) - further left is faster"), "axis-title", "middle")

	var xs, crash, rental, total []float64
	for _, p := range t.Points {
		xs = append(xs, float64(p.Duration))
		crash = append(crash, p.CrashCost)
		rental = append(rental, p.Rental)
		total = append(total, p.Total)
	}
	cv.PolyLine(xs, rental, "series rental")
	cv.PolyLine(xs, crash, "series crash")
	cv.PolyLine(xs, total, "series total")
	last := len(xs) - 1
	cv.Text(cv.X(xs[last])+4, cv.Y(total[last])-6, tr(lang, "total", "total"), "series-label total", "start")
	cv.Text(cv.X(xs[last])+4, cv.Y(crash[last])+14, tr(lang, "crashing eksak", "exact crashing"), "series-label crash", "start")
	cv.Text(cv.X(xs[0])-4, cv.Y(rental[0])+14, tr(lang, "sewa & langganan", "rentals & subscriptions"), "series-label rental", "end")

	for _, p := range t.Points {
		if p.Greedy > p.CrashCost+0.5 {
			cv.Group("pt").
				Circle(cv.X(float64(p.Duration)), cv.Y(p.Greedy), 4.5, "point greedy-excess").
				Title(fmt.Sprintf("%d %s: %s %s, %s %s", p.Duration, tr(lang, "hari", "days"),
					tr(lang, "serakah", "greedy"), Rp(p.Greedy, lang), tr(lang, "eksak", "exact"), Rp(p.CrashCost, lang))).
				EndGroup()
		}
	}
	o := t.Optimum
	cv.Group("pt").
		Circle(cv.X(float64(o.Duration)), cv.Y(o.Total), 6, "point optimum").
		Title(fmt.Sprintf("%s: %d %s, %s", tr(lang, "biaya total terendah", "lowest total cost"), o.Duration, tr(lang, "hari", "days"), Rp(o.Total, lang))).
		EndGroup()
	return template.HTML(cv.SVG("tradeoff"))
}

// BoundChart menggambar bagaimana tiga lapis batas bawah naik mendekati jadwal
// terbaik, dibandingkan dengan SGS polos. Bila batang batas bawah tertinggi
// menyentuh garis jadwal terbaik, jadwal itu terbukti optimal.
func BoundChart(o level.Optimized, lang string) template.HTML {
	type bar struct {
		label string
		value int
		class string
	}
	bars := []bar{
		{tr(lang, "CPM (tanpa kapasitas)", "CPM (no capacity)"), o.Bound.CPM, "bound"},
		{tr(lang, "Batas solo", "Solo bound"), o.Bound.Solo, "bound"},
		{tr(lang, "Batas energetik", "Energetic bound"), o.Bound.Energetic, "bound"},
		{tr(lang, "Batas bawah akhir", "Final lower bound"), o.Bound.Value, "bound final"},
		{tr(lang, "Jadwal terbaik ditemukan", "Best schedule found"), o.Best.Duration, "best"},
		{tr(lang, "SGS aturan LST", "SGS with LST rule"), o.Baseline.Duration, "baseline"},
	}
	rowH := 34.0
	cv := NewCanvas(880, 40+rowH*float64(len(bars))+30, Padding{Top: 20, Right: 70, Bottom: 30, Left: 220})
	lo := float64(o.Bound.CPM) * 0.9
	hi := float64(o.Baseline.Duration) * 1.03
	cv.SetDomain(lo, hi, 0, 1).Describe(
		tr(lang, "Batas bawah terhadap jadwal terbaik", "Lower bounds against the best schedule"),
		tr(lang,
			"Setiap batang batas bawah adalah durasi yang tidak mungkin dikalahkan jadwal mana pun. Bila batas bawah akhir sama dengan jadwal terbaik, jadwal itu terbukti optimal.",
			"Every lower-bound bar is a duration no schedule can beat. When the final lower bound equals the best schedule, that schedule is proven optimal."))
	var xTicks []Tick
	for _, v := range NiceTicks(lo, hi, 6) {
		xTicks = append(xTicks, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisX(xTicks, true)
	for i, b := range bars {
		y := cv.Pad.Top + float64(i)*rowH
		cv.Text(cv.Pad.Left-10, y+rowH/2+4, b.label, "axis-label", "end")
		x0 := cv.X(lo)
		x1 := cv.X(float64(b.value))
		cv.Rect(x0, y+6, math.Max(x1-x0, 1), rowH-12, "bound-bar "+b.class)
		cv.Text(x1+6, y+rowH/2+4, fmt.Sprintf("%d", b.value), "bar-value", "start")
	}
	bx := cv.X(float64(o.Best.Duration))
	cv.Line(bx, cv.Pad.Top-4, bx, cv.H-cv.Pad.Bottom, "ref-line best")
	return template.HTML(cv.SVG("bound-chart"))
}

// CountBars menggambar dua sebaran jumlah kejadian berdampingan - dipakai
// untuk membandingkan risiko saling bebas dengan risiko bergerombol.
func CountBars(a, b []float64, labelA, labelB string, maxK int, lang string) template.HTML {
	cv := NewCanvas(880, 300, Padding{Top: 34, Right: 24, Bottom: 50, Left: 64})
	if maxK <= 0 || len(a) == 0 || len(b) == 0 {
		return template.HTML(cv.SVG("count-bars"))
	}
	top := 0.0
	for k := 0; k <= maxK && k < len(a) && k < len(b); k++ {
		top = math.Max(top, math.Max(a[k], b[k]))
	}
	top *= 1.15
	cv.SetDomain(-0.5, float64(maxK)+0.5, 0, top).Describe(
		tr(lang, "Sebaran jumlah risiko yang terjadi per proyek", "Distribution of risks fired per project"),
		tr(lang,
			"Porsi iterasi dengan tepat k risiko terjadi, untuk risiko saling bebas dan risiko dengan penggerak bersama. Rata-ratanya sama; risiko bergerombol memindahkan peluang ke kedua ujung.",
			"Share of iterations with exactly k risks fired, for independent risks and risks with shared drivers. The mean is the same; clustered risks move probability to both ends."))
	var yTicks []Tick
	for _, v := range NiceTicks(0, top, 5) {
		yTicks = append(yTicks, Tick{v, Pct(v, 0, lang)})
	}
	cv.AxisY(yTicks, true)
	var xTicks []Tick
	for k := 0; k <= maxK; k++ {
		label := fmt.Sprintf("%d", k)
		if k == maxK {
			label += "+"
		}
		xTicks = append(xTicks, Tick{float64(k), label})
	}
	cv.AxisX(xTicks, false)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-10, tr(lang, "jumlah risiko yang terjadi dalam satu proyek", "number of risks fired in one project"), "axis-title", "middle")
	tail := func(v []float64) float64 {
		s := 0.0
		for k := maxK; k < len(v); k++ {
			s += v[k]
		}
		return s
	}
	w := cv.plotW() / float64(maxK+1) * 0.36
	for k := 0; k <= maxK; k++ {
		va, vb := a[k], b[k]
		if k == maxK {
			va, vb = tail(a), tail(b)
		}
		x := cv.X(float64(k))
		cv.Group("cb").Rect(x-w, cv.Y(va), w, cv.Y(0)-cv.Y(va), "count-bar a").Title(fmt.Sprintf("%s, k=%d: %s", labelA, k, Pct(va, 1, lang))).EndGroup()
		cv.Group("cb").Rect(x, cv.Y(vb), w, cv.Y(0)-cv.Y(vb), "count-bar b").Title(fmt.Sprintf("%s, k=%d: %s", labelB, k, Pct(vb, 1, lang))).EndGroup()
	}
	cv.Rect(cv.Pad.Left+10, 10, 10, 10, "count-bar a")
	cv.Text(cv.Pad.Left+26, 19, labelA, "axis-label", "start")
	cv.Rect(cv.Pad.Left+220, 10, 10, 10, "count-bar b")
	cv.Text(cv.Pad.Left+236, 19, labelB, "axis-label", "start")
	return template.HTML(cv.SVG("count-bars"))
}
