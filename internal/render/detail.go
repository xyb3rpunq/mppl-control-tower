package render

import (
	"fmt"
	"html/template"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Grafik rincian: pengganti tabel panjang yang isinya lebih cepat dibaca
// sebagai bentuk - sapuan asumsi, sebaran putaran, rentang estimasi, pecahan
// keterlambatan, dan linimasa. Semuanya generik supaya satu bentuk grafik
// bisa dipakai beberapa halaman dengan data berbeda.

// pathOf menulis perintah jalur SVG dari titik layar.
func pathOf(xs, ys []float64) string {
	var b strings.Builder
	for i := range xs {
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		b.WriteString(fmt.Sprintf("%s%s %s ", cmd, f(xs[i]), f(ys[i])))
	}
	return strings.TrimSpace(b.String())
}

// wrapLegend menggambar legenda yang membungkus ke baris baru bila penuh dan
// mengembalikan tinggi yang dipakainya.
func wrapLegend(cv *Canvas, items []LegendItem, x0, y0, maxX float64, swatch func(x, y float64, class string)) float64 {
	x, y := x0, y0
	for _, it := range items {
		w := 19 + textWidth(it.Label, 10.5) + 22
		if x+w > maxX && x > x0 {
			x, y = x0, y+16
		}
		swatch(x, y, it.Class)
		cv.Text(x+19, y+9, it.Label, "axis-label", "start")
		x += w
	}
	if len(items) == 0 {
		return 0
	}
	return y - y0 + 16
}

// legendRows menghitung jumlah baris legenda yang dibungkus pada lebar tertentu.
func legendRows(items []LegendItem, x0, maxX float64) int {
	if len(items) == 0 {
		return 0
	}
	rows, x := 1, x0
	for _, it := range items {
		w := 19 + textWidth(it.Label, 10.5) + 22
		if x+w > maxX && x > x0 {
			rows, x = rows+1, x0
		}
		x += w
	}
	return rows
}

// ---------------------------------------------------------------------------
// Panel sapuan asumsi

// SweepMinSpan adalah rentang tegak minimum panel sapuan, relatif terhadap
// rerata nilainya.
const SweepMinSpan = 0.08

// SweepPanel adalah satu besaran pada grafik sapuan asumsi.
type SweepPanel struct {
	Title  string
	Values []float64
	Format func(float64) string
}

// SweepPanels menggambar beberapa besaran terhadap satu asumsi yang disapu,
// satu panel kecil per besaran dengan sumbu asumsi yang sama. Setiap panel
// punya skala tegaknya sendiri, jadi yang dibandingkan adalah bentuk dan
// label perubahan di pojok: panel yang datar tidak peka terhadap asumsi.
func SweepPanels(xs []float64, xName string, highlight float64, panels []SweepPanel, title, desc, lang string) template.HTML {
	cols := 3
	if len(panels) == 4 || len(panels) == 2 {
		cols = 2
	}
	rows := (len(panels) + cols - 1) / cols
	panelW, panelH := 880/float64(cols), 176.0
	cv := NewCanvas(880, 8+panelH*float64(rows), Padding{})
	cv.Describe(title, desc)
	if len(xs) < 2 || len(panels) == 0 {
		return template.HTML(cv.SVG("sweep-panels"))
	}
	xlo, xhi := xs[0], xs[0]
	for _, x := range xs {
		xlo, xhi = math.Min(xlo, x), math.Max(xhi, x)
	}
	if xhi == xlo {
		xhi = xlo + 1
	}
	for i, p := range panels {
		n := min(len(xs), len(p.Values))
		if n == 0 {
			continue
		}
		format := p.Format
		if format == nil {
			format = func(v float64) string { return Num(v, 2, lang) }
		}
		ox, oy := float64(i%cols)*panelW, 8+float64(i/cols)*panelH
		left, right, top, bottom := ox+34, ox+panelW-30, oy+44, oy+panelH-44
		vs := p.Values[:n]
		lo, hi, sum := vs[0], vs[0], 0.0
		for _, v := range vs {
			lo, hi, sum = math.Min(lo, v), math.Max(hi, v), sum+v
		}
		// Rentang tegak paling sedikit SweepMinSpan dari rerata nilainya:
		// selisih Rp 2 ribu pada biaya Rp 19 juta harus tampak datar, bukan
		// dibesarkan skala otomatis menjadi garis curam.
		mid := (lo + hi) / 2
		span := math.Max(hi-lo, math.Max(math.Abs(sum/float64(n))*SweepMinSpan, 1e-9))
		lo, hi = mid-span*0.8, mid+span*0.8
		X := func(v float64) float64 { return left + (v-xlo)/(xhi-xlo)*(right-left) }
		Y := func(v float64) float64 { return bottom - (v-lo)/(hi-lo)*(bottom-top) }

		cv.Group("sweep-panel")
		cv.Rect(ox+6, oy, panelW-12, panelH-10, "sweep-bg")
		cv.Text(ox+18, oy+20, truncate(p.Title, 34), "row-label", "start")
		cv.Text(ox+18, oy+36, format(vs[0])+" → "+format(vs[n-1]), "bar-value sweep-change", "start")
		cv.Line(left, bottom, right, bottom, "axis-line")
		if highlight >= xlo && highlight <= xhi {
			hx := X(highlight)
			cv.Line(hx, top-4, hx, bottom, "ref-line sweep-default")
		}
		var px, py []float64
		for k := 0; k < n; k++ {
			px, py = append(px, X(xs[k])), append(py, Y(vs[k]))
			cv.Line(px[k], bottom, px[k], bottom+4, "axis-line")
			cv.Text(px[k], bottom+16, Num(xs[k], 2, lang), "axis-label", "middle")
		}
		cv.Path(pathOf(px, py), "sweep-line")
		for k := 0; k < n; k++ {
			cls := "sweep-dot"
			if math.Abs(xs[k]-highlight) < 1e-9 {
				cls += " default"
				ly := py[k] - 10
				if ly < top+2 {
					ly = py[k] + 18
				}
				anchor := "middle"
				if px[k] > right-30 {
					anchor = "end"
				} else if px[k] < left+30 {
					anchor = "start"
				}
				cv.Text(px[k], ly, format(vs[k]), "bar-value", anchor)
			}
			cv.Circle(px[k], py[k], 4.5, cls)
		}
		cv.Text(right, bottom+30, xName, "axis-label dim", "end")
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("sweep-panels"))
}

// ---------------------------------------------------------------------------
// Sebaran putaran rework GERT

// LoopBar adalah satu putaran rework pada grafik GERT.
type LoopBar struct {
	Label     string
	P         float64 // peluang gagal per pemeriksaan
	Analytic  float64 // E[N] bentuk tertutup
	Simulated float64 // E[N] di simulasi
	Q90       int     // putaran pada keyakinan 90%
}

// LoopProb mengembalikan peluang tepat k putaran ulang, (1-p)p^k; bucket
// terakhir (k = maxK) menampung ekor, p^maxK.
func LoopProb(p float64, k, maxK int) float64 {
	if k >= maxK {
		return math.Pow(p, float64(maxK))
	}
	return (1 - p) * math.Pow(p, float64(k))
}

// LoopBars menggambar peluang setiap jumlah putaran ulang untuk beberapa
// putaran rework berdampingan, dengan rerata analitik dan simulasi di legenda.
func LoopBars(loops []LoopBar, maxK int, lang string) template.HTML {
	cv := NewCanvas(880, 300+18*float64(len(loops)), Padding{Top: 34 + 18*float64(len(loops)), Right: 24, Bottom: 50, Left: 64})
	cv.Describe(tr(lang, "Berapa kali pekerjaan harus diulang", "How many times work must be redone"), tr(lang,
		"Untuk setiap putaran rework, batang menunjukkan peluang pemeriksaan harus diulang tepat 0, 1, 2, ... kali menurut model GERT: (1 − p) × p pangkat k. Bucket terakhir menampung ekornya.",
		"For each rework loop, bars show the chance the check must be repeated exactly 0, 1, 2, ... times under the GERT model: (1 − p) × p to the power k. The last bucket holds the tail."))
	if len(loops) == 0 || maxK < 1 {
		return template.HTML(cv.SVG("loop-bars"))
	}
	top := 0.0
	for _, lp := range loops {
		for k := 0; k <= maxK; k++ {
			top = math.Max(top, LoopProb(lp.P, k, maxK))
		}
	}
	cv.SetDomain(0, float64(maxK+1), 0, math.Max(top*1.18, 0.01))
	var yt []Tick
	for _, v := range NiceTicks(0, math.Max(top*1.18, 0.01), 5) {
		yt = append(yt, Tick{v, Pct(v, 0, lang)})
	}
	cv.AxisY(yt, true)
	groupW := cv.plotW() / float64(maxK+1)
	barW := groupW * 0.72 / float64(len(loops))
	for k := 0; k <= maxK; k++ {
		label := fmt.Sprintf("%d", k)
		if k == maxK {
			label = "≥ " + label
		}
		cv.Text(cv.X(float64(k))+groupW/2, cv.H-cv.Pad.Bottom+18, label, "axis-label", "middle")
		for j, lp := range loops {
			v := LoopProb(lp.P, k, maxK)
			x := cv.X(float64(k)) + groupW*0.14 + float64(j)*barW
			cv.Group("loop-col").
				Rect(x, cv.Y(v), math.Max(barW-3, 1), cv.Y(0)-cv.Y(v), fmt.Sprintf("loop-bar loop-%d", j%4)).
				Title(fmt.Sprintf("%s - %s: %s", lp.Label, label, Pct(v, 1, lang))).
				EndGroup()
			if barW >= 30 && v >= 0.005 {
				cv.Text(x+(barW-3)/2, cv.Y(v)-5, Pct(v, 0, lang), "bar-value", "middle")
			}
		}
	}
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-12, tr(lang, "jumlah putaran ulang sampai pemeriksaan lulus", "number of repeats until the check passes"), "axis-title", "middle")
	for j, lp := range loops {
		y := 12 + 18*float64(j)
		cv.Rect(12, y, 14, 10, fmt.Sprintf("loop-bar loop-%d", j%4))
		cv.Text(32, y+9, fmt.Sprintf("%s · p %s · E[N] %s %s, %s %s · 90%%: ≤ %d", truncate(lp.Label, 40), Pct(lp.P, 0, lang),
			tr(lang, "analitik", "analytic"), Num(lp.Analytic, 3, lang), tr(lang, "simulasi", "simulated"), Num(lp.Simulated, 3, lang), lp.Q90), "axis-label", "start")
	}
	return template.HTML(cv.SVG("loop-bars"))
}

// ---------------------------------------------------------------------------
// Batang kolom

// ColumnBar adalah satu kolom pada ColumnBars.
type ColumnBar struct {
	Label string
	Value float64
	Class string
	Note  string
}

// ColumnBars menggambar kolom tegak dengan nilai di atasnya dan garis acuan
// opsional, mis. biaya rencana.
func ColumnBars(bars []ColumnBar, ref float64, refLabel string, money bool, title, desc, lang string) template.HTML {
	cv := NewCanvas(880, 300, Padding{Top: 34, Right: 24, Bottom: 66, Left: 100})
	cv.Describe(title, desc)
	maxV := math.Max(ref, 0)
	for _, b := range bars {
		maxV = math.Max(maxV, b.Value)
	}
	if len(bars) == 0 || maxV <= 0 {
		return template.HTML(cv.SVG("column-bars"))
	}
	format := func(v float64) string {
		if money {
			return RpShort(v, lang)
		}
		return Num(v, 1, lang)
	}
	n := float64(len(bars))
	cv.SetDomain(0, n, 0, maxV*1.16)
	var yt []Tick
	for _, v := range NiceTicks(0, maxV*1.16, 5) {
		yt = append(yt, Tick{v, format(v)})
	}
	cv.AxisY(yt, true)
	barW := math.Min(cv.plotW()/n*0.56, 120)
	for i, b := range bars {
		x := cv.X(float64(i) + 0.5)
		v := math.Max(b.Value, 0)
		cv.Group("col").
			Rect(x-barW/2, cv.Y(v), barW, cv.Y(0)-cv.Y(v), "col-bar "+b.Class).
			Title(b.Label + ": " + format(b.Value)).
			EndGroup()
		cv.Text(x, cv.Y(v)-7, format(b.Value), "bar-value", "middle")
		cv.Text(x, cv.H-cv.Pad.Bottom+18, truncate(b.Label, 30), "axis-label", "middle")
		if b.Note != "" {
			cv.Text(x, cv.H-cv.Pad.Bottom+33, truncate(b.Note, 34), "axis-label dim", "middle")
		}
	}
	if ref > 0 {
		ry := cv.Y(ref)
		cv.Line(cv.Pad.Left, ry, cv.Pad.Left+cv.plotW(), ry, "ref-line target")
		cv.Line(cv.Pad.Left, 14, cv.Pad.Left+26, 14, "ref-line target")
		cv.Text(cv.Pad.Left+32, 18, refLabel+" "+format(ref), "axis-label", "start")
	}
	return template.HTML(cv.SVG("column-bars"))
}

// ---------------------------------------------------------------------------
// Dumbbell peluang

// DumbbellRow adalah satu baris pembanding dua peluang.
type DumbbellRow struct {
	Label string
	A, B  float64 // peluang 0..1
	Note  string  // teks kanan; kosong berarti nilai B
	Muted bool
}

// DumbbellRows membandingkan dua peluang per baris: lingkaran kosong A,
// lingkaran penuh B, disambung garis. Jarak garisnya adalah selisihnya.
func DumbbellRows(rows []DumbbellRow, aLabel, bLabel, title, desc, lang string) template.HTML {
	rowH := 26.0
	cv := NewCanvas(880, 78+rowH*float64(len(rows)), Padding{Top: 46, Right: 150, Bottom: 32, Left: 300})
	cv.Describe(title, desc)
	maxV := 0.0
	for _, r := range rows {
		maxV = math.Max(maxV, math.Max(r.A, r.B))
	}
	if len(rows) == 0 || maxV <= 0 {
		return template.HTML(cv.SVG("dumbbell"))
	}
	hi := math.Min(1, maxV*1.15)
	if hi < maxV {
		hi = maxV
	}
	cv.SetDomain(0, hi, 0, 1)
	var xt []Tick
	for _, v := range NiceTicks(0, hi, 6) {
		xt = append(xt, Tick{v, Pct(v, 0, lang)})
	}
	cv.AxisX(xt, true)
	cv.Circle(18, 17, 5, "db-a")
	cv.Text(28, 21, aLabel, "axis-label", "start")
	bx := 40 + textWidth(aLabel, 10.5) + 20
	cv.Circle(bx, 17, 5, "db-b")
	cv.Text(bx+10, 21, bLabel, "axis-label", "start")
	for i, r := range rows {
		y := cv.Pad.Top + rowH*float64(i) + rowH/2
		muted := ""
		if r.Muted {
			muted = " muted"
		}
		cv.Group("db-row" + muted)
		cv.Title(fmt.Sprintf("%s - %s %s, %s %s", r.Label, aLabel, Pct(r.A, 1, lang), bLabel, Pct(r.B, 1, lang)))
		cv.Text(12, y+4, truncate(r.Label, 46), "row-label", "start")
		xa, xb := cv.X(r.A), cv.X(r.B)
		cv.Line(xa, y, xb, y, "db-line")
		cv.Circle(xa, y, 5, "db-a")
		cv.Circle(xb, y, 5, "db-b"+muted)
		note := r.Note
		if note == "" {
			note = Pct(r.B, 1, lang)
		}
		cv.Text(cv.W-cv.Pad.Right+12, y+4, note, "bar-value", "start")
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("dumbbell"))
}

// ---------------------------------------------------------------------------
// Rentang estimasi tiga titik

// RangeRow adalah satu estimasi tiga titik.
type RangeRow struct {
	Label        string
	Lo, Mode, Hi float64
	Mean         float64
	Class        string // mis. critical
}

// RangeRows menggambar setiap estimasi O-M-P sebagai garis rentang dengan
// garis tegak di M dan belah ketupat di rerata beta-PERT. Rerata yang jatuh
// di kanan M menunjukkan estimasi condong ke keterlambatan.
func RangeRows(rows []RangeRow, unit, title, desc, lang string) template.HTML {
	rowH := 20.0
	cv := NewCanvas(880, 80+rowH*float64(len(rows)), Padding{Top: 46, Right: 120, Bottom: 34, Left: 300})
	cv.Describe(title, desc)
	hi := 0.0
	for _, r := range rows {
		hi = math.Max(hi, r.Hi)
	}
	if len(rows) == 0 || hi <= 0 {
		return template.HTML(cv.SVG("range-rows"))
	}
	cv.SetDomain(0, hi*1.04, 0, 1)
	var xt []Tick
	for _, v := range NiceTicks(0, hi*1.04, 8) {
		xt = append(xt, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisX(xt, true)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-6, unit, "axis-title", "middle")
	cv.Line(14, 17, 44, 17, "rr-range")
	cv.Text(50, 21, tr(lang, "optimistis sampai pesimistis", "optimistic to pessimistic"), "axis-label", "start")
	mx := 64 + textWidth(tr(lang, "optimistis sampai pesimistis", "optimistic to pessimistic"), 10.5)
	cv.Line(mx, 11, mx, 23, "rr-mode")
	cv.Text(mx+8, 21, tr(lang, "paling mungkin", "most likely"), "axis-label", "start")
	dx := mx + 24 + textWidth(tr(lang, "paling mungkin", "most likely"), 10.5)
	cv.Path(fmt.Sprintf("M%s 11 l6 6 l-6 6 l-6 -6 Z", f(dx)), "rr-mean")
	cv.Text(dx+10, 21, tr(lang, "rerata beta-PERT", "beta-PERT mean"), "axis-label", "start")
	cx := dx + 30 + textWidth(tr(lang, "rerata beta-PERT", "beta-PERT mean"), 10.5)
	cv.Line(cx, 17, cx+30, 17, "rr-range critical")
	cv.Text(cx+36, 21, tr(lang, "jalur kritis", "critical path"), "axis-label", "start")
	for i, r := range rows {
		y := cv.Pad.Top + rowH*float64(i) + rowH/2
		cv.Group("rr-row")
		cv.Title(fmt.Sprintf("%s: O %s, M %s, P %s, te %s", r.Label, Num(r.Lo, 0, lang), Num(r.Mode, 0, lang), Num(r.Hi, 0, lang), Num(r.Mean, 2, lang)))
		cv.Text(12, y+4, truncate(r.Label, 46), "row-label small", "start")
		cls := strings.TrimSpace("rr-range " + r.Class)
		cv.Line(cv.X(r.Lo), y, cv.X(r.Hi), y, cls)
		cv.Line(cv.X(r.Mode), y-6, cv.X(r.Mode), y+6, "rr-mode")
		cv.Path(fmt.Sprintf("M%s %s l5 5 l-5 5 l-5 -5 Z", f(cv.X(r.Mean)), f(y-5)), "rr-mean")
		cv.Text(cv.W-cv.Pad.Right+12, y+4, fmt.Sprintf("%s · %s · %s", Num(r.Lo, 0, lang), Num(r.Mode, 0, lang), Num(r.Hi, 0, lang)), "bar-value", "start")
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("range-rows"))
}

// ---------------------------------------------------------------------------
// Batang bertumpuk per baris

// StackPart adalah satu segmen batang bertumpuk.
type StackPart struct {
	Value float64
	Class string
	Label string // ditulis di dalam segmen bila muat
}

// StackRow adalah satu baris batang bertumpuk.
type StackRow struct {
	Label string
	Parts []StackPart
	Note  string // teks kanan; kosong berarti jumlah segmen
	Class string
}

// StackRows menggambar batang mendatar yang tersusun dari beberapa segmen,
// satu baris per entri: pecahan keterlambatan, float, atau anggaran per fase.
func StackRows(rows []StackRow, legend []LegendItem, money bool, unit, title, desc, lang string) template.HTML {
	rowH := 24.0
	legendH := 16 * float64(legendRows(legend, 12, 868))
	cv := NewCanvas(880, 72+legendH+rowH*float64(len(rows)), Padding{Top: 30 + legendH, Right: 130, Bottom: 34, Left: 250})
	cv.Describe(title, desc)
	format := func(v float64) string {
		if money {
			return RpShort(v, lang)
		}
		return Num(v, 0, lang)
	}
	maxTotal := 0.0
	for _, r := range rows {
		t := 0.0
		for _, p := range r.Parts {
			t += math.Max(p.Value, 0)
		}
		maxTotal = math.Max(maxTotal, t)
	}
	if len(rows) == 0 || maxTotal <= 0 {
		return template.HTML(cv.SVG("stack-rows"))
	}
	wrapLegend(cv, legend, 12, 10, 868, func(x, y float64, class string) { cv.Rect(x, y, 14, 10, "stack-seg "+class) })
	cv.SetDomain(0, maxTotal*1.02, 0, 1)
	var xt []Tick
	for _, v := range NiceTicks(0, maxTotal*1.02, 7) {
		xt = append(xt, Tick{v, format(v)})
	}
	cv.AxisX(xt, true)
	if unit != "" {
		cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-4, unit, "axis-title", "middle")
	}
	for i, r := range rows {
		y := cv.Pad.Top + rowH*float64(i)
		cv.Group(strings.TrimSpace("stack-row " + r.Class))
		cv.Text(12, y+rowH/2+4, truncate(r.Label, 38), "row-label small", "start")
		x, total := cv.X(0), 0.0
		for _, p := range r.Parts {
			if p.Value <= 0 {
				continue
			}
			w := cv.X(p.Value) - cv.X(0)
			cv.Group("stack-part").
				Rect(x, y+4, w, rowH-8, "stack-seg "+p.Class).
				Title(r.Label + " - " + p.Label + ": " + format(p.Value)).
				EndGroup()
			if p.Label != "" && textWidth(p.Label, 9.5)+6 < w {
				cv.Text(x+w/2, y+rowH/2+3.5, p.Label, "stack-label", "middle")
			}
			x += w
			total += p.Value
		}
		note := r.Note
		if note == "" {
			note = format(total)
		}
		cv.Text(x+6, y+rowH/2+4, note, "bar-value", "start")
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("stack-rows"))
}

// ---------------------------------------------------------------------------
// Scatter berlabel

// ScatterPoint adalah satu titik pada scatter berlabel.
type ScatterPoint struct {
	X, Y  float64
	Label string // kosong berarti tanpa label
	Class string
	Note  string // tooltip
}

// ScatterLabeled menggambar titik dua besaran dengan label yang ditempatkan
// tanpa saling menimpa. Sumbu tegak bisa rupiah.
func ScatterLabeled(pts []ScatterPoint, xTitle, yTitle string, yMoney bool, legend []LegendItem, title, desc, lang string) template.HTML {
	cv := NewCanvas(880, 390, Padding{Top: 44, Right: 30, Bottom: 58, Left: 104})
	cv.Describe(title, desc)
	if len(pts) == 0 {
		return template.HTML(cv.SVG("scatter"))
	}
	xlo, xhi, yhi := pts[0].X, pts[0].X, 0.0
	for _, p := range pts {
		xlo, xhi, yhi = math.Min(xlo, p.X), math.Max(xhi, p.X), math.Max(yhi, p.Y)
	}
	xlo, xhi = math.Min(xlo, 0)-0.5, xhi+0.8
	if yhi <= 0 {
		yhi = 1
	}
	yhi *= 1.18
	cv.SetDomain(xlo, xhi, 0, yhi)
	format := func(v float64) string {
		if yMoney {
			return RpShort(v, lang)
		}
		return Num(v, 1, lang)
	}
	var yt []Tick
	for _, v := range NiceTicks(0, yhi, 6) {
		yt = append(yt, Tick{v, format(v)})
	}
	cv.AxisY(yt, true)
	var xt []Tick
	for _, v := range NiceTicks(math.Max(xlo, 0), xhi, 8) {
		if v == math.Trunc(v) {
			xt = append(xt, Tick{v, Num(v, 0, lang)})
		}
	}
	cv.AxisX(xt, true)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-12, xTitle, "axis-title", "middle")
	cy := cv.Pad.Top + cv.plotH()/2
	cv.Text(18, cy, yTitle, "axis-title", "middle", "transform", fmt.Sprintf("rotate(-90 18 %s)", f(cy)))
	wrapLegend(cv, legend, cv.Pad.Left, 12, cv.W-12, func(x, y float64, class string) { cv.Circle(x+7, y+5, 5, "sc-pt "+class) })

	lp := &labelPlacer{w: cv.W - 4, h: cv.H - cv.Pad.Bottom}
	for _, p := range pts {
		x, y := cv.X(p.X), cv.Y(p.Y)
		lp.boxes = append(lp.boxes, [4]float64{x - 7, y - 7, x + 7, y + 7})
	}
	order := make([]int, len(pts))
	for i := range order {
		order[i] = i
	}
	// Titik tanpa label digambar dulu supaya titik berlabel berada di atasnya.
	sort.SliceStable(order, func(a, b int) bool { return pts[order[a]].Label == "" && pts[order[b]].Label != "" })
	for _, i := range order {
		p := pts[i]
		x, y := cv.X(p.X), cv.Y(p.Y)
		tip := p.Note
		if tip == "" {
			tip = p.Label
		}
		cv.Group("sc-point").Circle(x, y, 6, "sc-pt "+p.Class).Title(tip).EndGroup()
		if p.Label != "" {
			lx, ly, anchor := lp.place(x, y, []string{p.Label}, 10.5)
			cv.Text(lx, ly, p.Label, "bar-value sc-label "+p.Class, anchor)
		}
	}
	return template.HTML(cv.SVG("scatter"))
}

// ---------------------------------------------------------------------------
// Batang divergen

// DivRow adalah satu baris batang divergen.
type DivRow struct {
	Label string
	Value float64
	Note  string
}

// DivergingRows menggambar nilai bertanda dari garis nol: kiri untuk nilai
// negatif, kanan untuk positif, dengan label arah di atas.
func DivergingRows(rows []DivRow, money bool, negLabel, posLabel, title, desc, lang string) template.HTML {
	rowH := 22.0
	cv := NewCanvas(880, 78+rowH*float64(len(rows)), Padding{Top: 46, Right: 40, Bottom: 32, Left: 270})
	cv.Describe(title, desc)
	m := 0.0
	for _, r := range rows {
		m = math.Max(m, math.Abs(r.Value))
	}
	if len(rows) == 0 {
		return template.HTML(cv.SVG("diverging"))
	}
	if m == 0 {
		m = 1
	}
	format := func(v float64) string {
		if money {
			s := RpShort(math.Abs(v), lang)
			switch {
			case v > 0:
				return "+" + s
			case v < 0:
				return "-" + s
			}
			return s
		}
		return SignedNum(v, 1, lang)
	}
	neg, pos := false, false
	for _, r := range rows {
		neg, pos = neg || r.Value < 0, pos || r.Value > 0
	}
	lo, hi := -m*1.35, m*1.35
	if !pos {
		hi = m * 0.12
	}
	if !neg {
		lo = -m * 0.12
	}
	cv.SetDomain(lo, hi, 0, 1)
	var xt []Tick
	for _, v := range NiceTicks(lo, hi, 7) {
		xt = append(xt, Tick{v, format(v)})
	}
	cv.AxisX(xt, true)
	x0 := cv.X(0)
	cv.Line(x0, cv.Pad.Top-8, x0, cv.H-cv.Pad.Bottom, "ref-line divider")
	cv.Text(x0-10, cv.Pad.Top-14, "← "+negLabel, "axis-label div-neg-label", "end")
	cv.Text(x0+10, cv.Pad.Top-14, posLabel+" →", "axis-label div-pos-label", "start")
	for i, r := range rows {
		y := cv.Pad.Top + rowH*float64(i)
		cls := "div-bar pos"
		if r.Value < 0 {
			cls = "div-bar neg"
		}
		cv.Group("div-row")
		cv.Title(r.Label + ": " + format(r.Value))
		cv.Text(12, y+rowH/2+4, truncate(r.Label, 38), "row-label small", "start")
		xv := cv.X(r.Value)
		cv.Rect(math.Min(x0, xv), y+4, math.Max(math.Abs(xv-x0), 1), rowH-8, cls)
		label := format(r.Value)
		if r.Note != "" {
			label += " · " + r.Note
		}
		w := textWidth(label, 10.5)
		switch {
		case r.Value < 0 && xv-6-w < cv.Pad.Left:
			cv.Text(xv+6, y+rowH/2+4, label, "bar-value div-label-in", "start")
		case r.Value < 0:
			cv.Text(xv-6, y+rowH/2+4, label, "bar-value", "end")
		case xv+6+w > cv.W-4:
			cv.Text(xv-6, y+rowH/2+4, label, "bar-value div-label-in", "end")
		default:
			cv.Text(xv+6, y+rowH/2+4, label, "bar-value", "start")
		}
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("diverging"))
}

// ---------------------------------------------------------------------------
// Linimasa kejadian

// TimelineEvent adalah satu kejadian bertanggal ISO pada linimasa.
type TimelineEvent struct {
	Date  string
	Label string
	Class string
}

// EventTimeline menggambar kejadian pada sumbu tahun dengan label bertumpuk
// di atas dan di bawah sumbu, sehingga jeda panjang antar-kejadian terlihat
// sebagai jarak - bukan hanya sebagai baris berikutnya pada daftar.
func EventTimeline(events []TimelineEvent, title, desc, lang string) template.HTML {
	type placed struct {
		ev     TimelineEvent
		t      time.Time
		x      float64
		lane   int // 0,2,4... di atas; 1,3,5... di bawah
		anchor string
		label  string
	}
	var ps []placed
	for _, e := range events {
		t, err := time.Parse("2006-01-02", e.Date)
		if err != nil {
			continue
		}
		ps = append(ps, placed{ev: e, t: t})
	}
	const width, left, right = 880.0, 40.0, 40.0
	if len(ps) == 0 {
		cv := NewCanvas(width, 120, Padding{})
		cv.Describe(title, desc)
		return template.HTML(cv.SVG("event-timeline"))
	}
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].t.Before(ps[j].t) })
	start := time.Date(ps[0].t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(ps[len(ps)-1].t.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
	span := end.Sub(start).Hours()
	X := func(t time.Time) float64 { return left + t.Sub(start).Hours()/span*(width-left-right) }

	// Setiap label mencoba lajur terdekat sumbu yang belum terisi pada
	// rentang mendatarnya, bergantian atas dan bawah.
	var laneEnd []float64
	lanes := 0
	for i := range ps {
		p := &ps[i]
		p.x = X(p.t)
		p.label = truncate(p.ev.Label, 42)
		w := math.Max(textWidth(p.label, 10.5), textWidth(workcal.FormatDate(p.ev.Date, lang), 10.5)) + 8
		x1, x2, anchor := p.x-4, p.x-4+w, "start"
		if x2 > width-8 {
			x1, x2, anchor = p.x+4-w, p.x+4, "end"
		}
		lane := 0
		for ; lane < len(laneEnd); lane++ {
			if laneEnd[lane] < x1 {
				break
			}
		}
		if lane == len(laneEnd) {
			laneEnd = append(laneEnd, 0)
		}
		laneEnd[lane] = x2
		p.lane, p.anchor = lane, anchor
		lanes = max(lanes, lane+1)
	}
	laneH := 36.0
	above := (lanes + 1) / 2
	below := lanes / 2
	axisY := 24 + laneH*float64(above)
	h := axisY + 30 + laneH*float64(below) + 10
	cv := NewCanvas(width, h, Padding{})
	cv.Describe(title, desc)
	cv.Line(left, axisY, width-right, axisY, "tl-axis")
	for y := start.Year(); y <= end.Year(); y++ {
		x := X(time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC))
		cv.Line(x, axisY-4, x, axisY+4, "axis-line")
		cv.Text(x, axisY+18, fmt.Sprintf("%d", y), "axis-label", "middle")
	}
	for _, p := range ps {
		up := p.lane%2 == 0
		k := float64(p.lane / 2)
		var ly float64
		if up {
			ly = axisY - 12 - laneH*k
		} else {
			ly = axisY + 34 + laneH*k
		}
		cv.Group("tl-event " + p.ev.Class)
		cv.Title(workcal.FormatDate(p.ev.Date, lang) + ": " + p.ev.Label)
		if up {
			cv.Line(p.x, axisY, p.x, ly+4, "tl-leader")
			cv.Text(p.x+map[string]float64{"start": 4, "end": -4}[p.anchor], ly-14, workcal.FormatDate(p.ev.Date, lang), "bar-value tlc-date", p.anchor)
			cv.Text(p.x+map[string]float64{"start": 4, "end": -4}[p.anchor], ly, p.label, "axis-label tlc-label", p.anchor)
		} else {
			cv.Line(p.x, axisY, p.x, ly-12, "tl-leader")
			cv.Text(p.x+map[string]float64{"start": 4, "end": -4}[p.anchor], ly, workcal.FormatDate(p.ev.Date, lang), "bar-value tlc-date", p.anchor)
			cv.Text(p.x+map[string]float64{"start": 4, "end": -4}[p.anchor], ly+14, p.label, "axis-label tlc-label", p.anchor)
		}
		cv.Circle(p.x, axisY, 6, "tl-dot "+p.ev.Class)
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("event-timeline"))
}
