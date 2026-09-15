package render

import (
	"fmt"
	"html/template"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Grafik penjelas: setiap grafik di berkas ini dibuat untuk satu pertanyaan
// yang biasanya hanya dijawab tabel, dan menuliskan jawabannya langsung di
// dalam gambar - label pada titik, bukan legenda yang harus dicocokkan mata.

// labelPlacer menempatkan label tanpa saling menimpa: setiap label mencoba
// beberapa posisi di sekitar titiknya dan memakai yang pertama tidak
// bertabrakan dengan label lain dan masih di dalam kanvas.
type labelPlacer struct {
	w, h  float64
	boxes [][4]float64 // x1, y1, x2, y2
}

// textWidth memperkirakan lebar teks 11px dalam piksel.
func textWidth(s string, size float64) float64 { return float64(len([]rune(s))) * size * 0.56 }

// place mengembalikan posisi baseline dan anchor untuk sekumpulan baris teks
// di dekat titik (x, y). Kandidat disusun makin jauh dari titik dengan jarak
// setinggi label, supaya label berbaris banyak tidak saling menimpa hanya
// karena kandidatnya terlalu rapat. Bila tidak ada yang bebas, dipilih
// kandidat di dalam kanvas dengan tumpang-tindih terkecil.
func (lp *labelPlacer) place(x, y float64, lines []string, size float64) (float64, float64, string) {
	w := 0.0
	for _, l := range lines {
		w = math.Max(w, textWidth(l, size))
	}
	h := float64(len(lines)) * (size + 3)
	step := h + 4
	below := math.Max(18, -8+step)
	var dys []float64
	for k := 0.0; k < 4; k++ {
		dys = append(dys, -8-k*step, below+k*step)
	}
	boxAt := func(dx, dy float64, anchor string) [4]float64 {
		x1 := x + dx
		if anchor == "end" {
			x1 -= w
		}
		return [4]float64{x1 - 2, y + dy - size - 2, x1 + w + 2, y + dy - size + h}
	}
	overlap := func(box [4]float64) float64 {
		area := 0.0
		for _, o := range lp.boxes {
			ow := math.Min(box[2], o[2]) - math.Max(box[0], o[0])
			oh := math.Min(box[3], o[3]) - math.Max(box[1], o[1])
			if ow > 0 && oh > 0 {
				area += ow * oh
			}
		}
		return area
	}
	bestArea, bestDx, bestDy, bestAnchor := math.Inf(1), 12.0, -8.0, "start"
	for _, dy := range dys {
		for _, side := range []struct {
			dx     float64
			anchor string
		}{{12, "start"}, {-12, "end"}} {
			box := boxAt(side.dx, dy, side.anchor)
			if box[0] < 0 || box[2] > lp.w || box[1] < 0 || box[3] > lp.h {
				continue
			}
			area := overlap(box)
			if area == 0 {
				lp.boxes = append(lp.boxes, box)
				return x + side.dx, y + dy, side.anchor
			}
			if area < bestArea {
				bestArea, bestDx, bestDy, bestAnchor = area, side.dx, dy, side.anchor
			}
		}
	}
	lp.boxes = append(lp.boxes, boxAt(bestDx, bestDy, bestAnchor))
	return x + bestDx, y + bestDy, bestAnchor
}

// avoidSegment mendaftarkan garis sebagai rintangan: kotak kecil setiap
// beberapa piksel sepanjang garis. Label yang hanya bisa ditempatkan di atas
// garis tetap boleh, tetapi kandidat yang bebas garis dipilih lebih dulu.
func (lp *labelPlacer) avoidSegment(x1, y1, x2, y2 float64) {
	n := int(math.Ceil(math.Hypot(x2-x1, y2-y1) / 6))
	for i := 0; i <= n; i++ {
		t := float64(i) / math.Max(float64(n), 1)
		x, y := x1+(x2-x1)*t, y1+(y2-y1)*t
		lp.boxes = append(lp.boxes, [4]float64{x - 2, y - 2, x + 2, y + 2})
	}
}

// spreadY menjauhkan posisi tegak label sedikitnya gap piksel dengan urutan
// tetap dan menjaganya di dalam [lo, hi]. Tiga lintasan: dorong ke bawah
// untuk jarak, tarik ke atas dari batas bawah, lalu dorong lagi dari batas
// atas. Hanya bila ruangnya memang kurang, label boleh melewati hi.
func spreadY(ys []float64, gap, lo, hi float64) []float64 {
	idx := make([]int, len(ys))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return ys[idx[a]] < ys[idx[b]] })
	out := append([]float64(nil), ys...)
	n := len(idx)
	if n == 0 {
		return out
	}
	for k := 1; k < n; k++ {
		out[idx[k]] = math.Max(out[idx[k]], out[idx[k-1]]+gap)
	}
	out[idx[n-1]] = math.Min(out[idx[n-1]], hi)
	for k := n - 2; k >= 0; k-- {
		out[idx[k]] = math.Min(out[idx[k]], out[idx[k+1]]-gap)
	}
	out[idx[0]] = math.Max(out[idx[0]], lo)
	for k := 1; k < n; k++ {
		out[idx[k]] = math.Max(out[idx[k]], out[idx[k-1]]+gap)
	}
	return out
}

// dateTicks membuat penanda sumbu tanggal dari indeks hari kerja.
func dateTicks(cal *workcal.Calendar, lo, hi float64, count int, lang string) []Tick {
	var out []Tick
	for _, v := range NiceTicks(lo, hi, count) {
		d := int(math.Round(v)) - 1
		if d < 0 || cal == nil || d >= cal.Len() {
			continue
		}
		out = append(out, Tick{v, workcal.FormatDateShort(cal.ISOAt(d), lang)})
	}
	return out
}

// finishLabel menulis tanggal selesai untuk durasi dalam hari kerja.
func finishLabel(cal *workcal.Calendar, day float64, lang string) string {
	d := int(math.Round(day)) - 1
	if cal == nil || d < 0 || d >= cal.Len() {
		return Num(day, 0, lang)
	}
	return workcal.FormatDate(cal.ISOAt(d), lang)
}

// DateItem adalah satu baris pada DateDots.
type DateItem struct {
	Label  string
	Status string  // teks kecil di bawah label
	Day    float64 // durasi dalam hari kerja; tanggal selesai = hari ke-Day
	Note   string  // teks di samping titik, mis. anggaran
	Class  string  // kelas CSS titik: inforce, superseded, invalid, neutral
}

// DateRef adalah garis tegak acuan, mis. tanggal data.
type DateRef struct {
	Day   float64
	Label string
	Class string
}

// DateDots menggambar beberapa angka durasi sebagai titik pada satu garis
// waktu, satu baris per angka, supaya jarak antar-janji terlihat dalam hari.
func DateDots(items []DateItem, refs []DateRef, cal *workcal.Calendar, title, desc, lang string) template.HTML {
	rowH := 40.0
	cv := NewCanvas(880, 70+rowH*float64(len(items)), Padding{Top: 44, Right: 150, Bottom: 34, Left: 250})
	cv.Describe(title, desc)
	if len(items) == 0 {
		return template.HTML(cv.SVG("date-dots"))
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, it := range items {
		lo, hi = math.Min(lo, it.Day), math.Max(hi, it.Day)
	}
	for _, r := range refs {
		lo, hi = math.Min(lo, r.Day), math.Max(hi, r.Day)
	}
	span := math.Max(hi-lo, 10)
	lo, hi = lo-span*0.06, hi+span*0.06
	cv.SetDomain(lo, hi, 0, 1)
	cv.AxisX(dateTicks(cal, lo, hi, 8, lang), true)
	for i, r := range refs {
		x := cv.X(r.Day)
		cv.Line(x, cv.Pad.Top-8, x, cv.H-cv.Pad.Bottom, "ref-line "+r.Class)
		cv.Text(x, cv.Pad.Top-14-float64(i%2)*13, r.Label, "axis-label ref-label "+r.Class, "middle")
	}
	for i, it := range items {
		y := cv.Pad.Top + rowH*float64(i) + rowH/2
		cv.Group("dd-row " + it.Class)
		if i%2 == 0 {
			cv.Rect(0, y-rowH/2, cv.W, rowH, "dd-band")
		}
		cv.Text(12, y-2, it.Label, "row-label dd-label", "start")
		if it.Status != "" {
			cv.Text(12, y+12, it.Status, "axis-label dd-status "+it.Class, "start")
		}
		x := cv.X(it.Day)
		cv.Line(cv.Pad.Left, y, x, y, "dd-guide")
		cv.Circle(x, y, 7, "dd-dot "+it.Class).Title(it.Label + ": " + finishLabel(cal, it.Day, lang) + " " + it.Note)
		label := finishLabel(cal, it.Day, lang)
		if it.Note != "" {
			label += " · " + it.Note
		}
		if x > cv.W-cv.Pad.Right-40 {
			cv.Text(x-12, y+4, label, "bar-value dd-value", "end")
		} else {
			cv.Text(x+12, y+4, label, "bar-value dd-value", "start")
		}
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("date-dots"))
}

// OptionPoint adalah satu opsi pada peta opsi.
type OptionPoint struct {
	Label                            string
	Day, Budget                      float64
	DayLo, DayHi, BudgetLo, BudgetHi float64 // interval 90%; nol berarti tidak ada
	Note                             string
	Class                            string // base, cheapest, fastest, other, assumed
}

// OptionMap menggambar setiap opsi sebagai titik tanggal selesai x anggaran,
// dengan interval bootstrap 90% dan panah dari opsi tanpa percepatan: panah
// ke kiri berarti lebih cepat, ke atas berarti lebih mahal.
func OptionMap(pts []OptionPoint, cal *workcal.Calendar, lang string) template.HTML {
	cv := NewCanvas(880, 440, Padding{Top: 30, Right: 30, Bottom: 58, Left: 120})
	cv.Describe(tr(lang, "Peta opsi percepatan", "Acceleration options map"), tr(lang,
		"Setiap titik adalah satu opsi pada titik JCL 70%: tanggal selesai mendatar, anggaran tegak. Garis silang adalah interval bootstrap 90%. Panah berangkat dari opsi tanpa percepatan: makin ke kiri makin cepat, makin ke atas makin mahal.",
		"Each point is one option at the 70% JCL: finish date across, budget up. The cross lines are the 90% bootstrap interval. Arrows start from the no-acceleration option: further left is faster, further up is more expensive."))
	if len(pts) == 0 {
		return template.HTML(cv.SVG("option-map"))
	}
	xlo, xhi, ylo, yhi := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		for _, d := range []float64{p.Day, p.DayLo, p.DayHi} {
			if d > 0 {
				xlo, xhi = math.Min(xlo, d), math.Max(xhi, d)
			}
		}
		for _, b := range []float64{p.Budget, p.BudgetLo, p.BudgetHi} {
			if b > 0 {
				ylo, yhi = math.Min(ylo, b), math.Max(yhi, b)
			}
		}
	}
	if math.IsInf(xlo, 0) || math.IsInf(ylo, 0) {
		return template.HTML(cv.SVG("option-map"))
	}
	xs, ys := math.Max(xhi-xlo, 4), math.Max(yhi-ylo, 1)
	xlo, xhi = xlo-xs*0.3, xhi+xs*0.3
	ylo, yhi = ylo-ys*0.12, yhi+ys*0.12
	cv.SetDomain(xlo, xhi, ylo, yhi)
	var yt []Tick
	for _, v := range NiceTicks(ylo, yhi, 6) {
		yt = append(yt, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yt, true)
	cv.AxisX(dateTicks(cal, xlo, xhi, 7, lang), true)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-10, tr(lang, "tanggal selesai pada JCL 70%   ← lebih cepat", "finish date at the 70% JCL   ← faster"), "axis-title", "middle")
	cv.Text(18, cv.Pad.Top+cv.plotH()/2, tr(lang, "anggaran JCL 70%  ↑ lebih mahal", "70% JCL budget  ↑ more expensive"), "axis-title", "middle",
		"transform", fmt.Sprintf("rotate(-90 18 %s)", f(cv.Pad.Top+cv.plotH()/2)))

	var base *OptionPoint
	for i := range pts {
		if pts[i].Class == "base" {
			base = &pts[i]
		}
	}
	for _, p := range pts {
		if base != nil && p.Class != "base" {
			cv.Line(cv.X(base.Day), cv.Y(base.Budget), cv.X(p.Day), cv.Y(p.Budget), "om-link")
		}
	}
	lp := &labelPlacer{w: cv.W, h: cv.H - cv.Pad.Bottom}
	for _, p := range pts {
		x, y := cv.X(p.Day), cv.Y(p.Budget)
		lp.boxes = append(lp.boxes, [4]float64{x - 9, y - 9, x + 9, y + 9})
		if base != nil && p.Class != "base" {
			lp.avoidSegment(cv.X(base.Day), cv.Y(base.Budget), x, y)
		}
		if p.DayLo > 0 && p.DayHi > p.DayLo {
			lp.avoidSegment(cv.X(p.DayLo), y, cv.X(p.DayHi), y)
		}
		if p.BudgetLo > 0 && p.BudgetHi > p.BudgetLo {
			lp.avoidSegment(x, cv.Y(p.BudgetLo), x, cv.Y(p.BudgetHi))
		}
	}
	for _, p := range pts {
		x, y := cv.X(p.Day), cv.Y(p.Budget)
		cv.Group("om-option " + p.Class)
		if p.DayLo > 0 && p.DayHi > p.DayLo {
			cv.Line(cv.X(p.DayLo), y, cv.X(p.DayHi), y, "om-ci")
		}
		if p.BudgetLo > 0 && p.BudgetHi > p.BudgetLo {
			cv.Line(x, cv.Y(p.BudgetLo), x, cv.Y(p.BudgetHi), "om-ci")
		}
		cv.Circle(x, y, 8, "om-pt "+p.Class).Title(p.Label + ": " + finishLabel(cal, p.Day, lang) + ", " + Rp(p.Budget, lang))
		lines := []string{p.Label}
		if p.Note != "" {
			lines = append(lines, p.Note)
		}
		lx, ly, anchor := lp.place(x, y, lines, 11.5)
		cv.Text(lx, ly, p.Label, "row-label om-label", anchor)
		if p.Note != "" {
			cv.Text(lx, ly+14, p.Note, "axis-label om-note", anchor)
		}
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("option-map"))
}

// BenefitLine adalah garis manfaat bersih satu opsi: v x Days - Extra.
type BenefitLine struct {
	Label       string
	Days, Extra float64
	Class       string
}

// BandSpan adalah satu rentang nilai tempat satu opsi terbaik.
type BandSpan struct {
	From, To       float64 // To boleh +Inf
	Label          string
	Class          string
	EdgeLo, EdgeHi float64 // interval 90% batas bawah rentang; nol berarti tidak ada
}

// ValueBandsChart menggambar garis manfaat bersih setiap opsi terhadap nilai
// satu hari lebih cepat, dengan selubung atas yang tebal dan rentang tempat
// setiap opsi menjadi pilihan terbaik.
func ValueBandsChart(lines []BenefitLine, bands []BandSpan, lang string) template.HTML {
	cv := NewCanvas(880, 430, Padding{Top: 76, Right: 190, Bottom: 58, Left: 112})
	cv.Describe(tr(lang, "Pita nilai satu hari lebih cepat", "Value-of-a-day bands"), tr(lang,
		"Setiap garis adalah manfaat bersih satu opsi: nilai satu hari lebih cepat dikali hari yang dimajukan, dikurangi tambahan anggaran. Garis tebal adalah opsi terbaik pada setiap nilai; latar berwarna menandai rentangnya, dan batas diberi interval bootstrap 90%.",
		"Each line is one option's net benefit: the value of a day earlier times days gained, minus extra budget. The thick line is the best option at every value; shaded backgrounds mark its range, and boundaries carry a 90% bootstrap interval."))
	if len(lines) == 0 || len(bands) == 0 {
		return template.HTML(cv.SVG("value-bands"))
	}
	vmax := 400000.0
	if last := bands[len(bands)-1].From; last > 0 {
		vmax = last * 1.6
	}
	for _, b := range bands {
		vmax = math.Max(vmax, b.EdgeHi*1.05)
	}
	net := func(l BenefitLine, v float64) float64 { return v*l.Days - l.Extra }
	ylo, yhi := 0.0, 0.0
	for _, l := range lines {
		for _, v := range []float64{0, vmax} {
			ylo, yhi = math.Min(ylo, net(l, v)), math.Max(yhi, net(l, v))
		}
	}
	pad := (yhi - ylo) * 0.08
	cv.SetDomain(0, vmax, ylo-pad, yhi+pad)
	for i, b := range bands {
		to := math.Min(b.To, vmax)
		x1, x2 := cv.X(b.From), cv.X(to)
		cv.Rect(x1, cv.Pad.Top, x2-x1, cv.plotH(), "vb-band "+b.Class)
		label := truncate(b.Label, 38)
		cx := (x1 + x2) / 2
		half := textWidth(label, 11.5) / 2
		cx = math.Max(half+4, math.Min(cv.W-half-4, cx))
		ly := cv.Pad.Top - 10 - float64(len(bands)-1-i)*17
		cv.Line(cx, ly+4, (x1+x2)/2, cv.Pad.Top, "vb-leader "+b.Class)
		cv.Text(cx, ly, label, "row-label vb-band-label "+b.Class, "middle")
	}
	var yt []Tick
	for _, v := range NiceTicks(ylo, yhi, 6) {
		yt = append(yt, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yt, true)
	var xt []Tick
	for _, v := range NiceTicks(0, vmax, 6) {
		xt = append(xt, Tick{v, RpShort(v, lang)})
	}
	cv.AxisX(xt, false)
	cv.Line(cv.Pad.Left, cv.Y(0), cv.Pad.Left+cv.plotW(), cv.Y(0), "ref-line")

	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-12, tr(lang, "nilai satu hari lebih cepat bagi sponsor (rupiah per hari)", "what a day earlier is worth to the sponsor (rupiah per day)"), "axis-title", "middle")
	for k, b := range bands {
		if b.From <= 0 {
			continue
		}
		x := cv.X(b.From)
		cv.Line(x, cv.Pad.Top, x, cv.Pad.Top+cv.plotH(), "vb-edge")
		ey := cv.Pad.Top + cv.plotH() - 14 - float64(k%2)*16
		if b.EdgeHi > b.EdgeLo {
			x1, x2 := cv.X(b.EdgeLo), cv.X(math.Min(b.EdgeHi, vmax))
			cv.Line(x1, ey, x2, ey, "vb-ci").Line(x1, ey-4, x1, ey+4, "vb-ci").Line(x2, ey-4, x2, ey+4, "vb-ci")
		}
		cv.Text(x+5, ey-6, Rp(b.From, lang), "bar-value vb-edge-label", "start")
	}
	var endY []float64
	for _, l := range lines {
		cv.PolyLine([]float64{0, vmax}, []float64{net(l, 0), net(l, vmax)}, "vb-line "+l.Class)
		endY = append(endY, cv.Y(net(l, vmax))+4)
	}
	endY = spreadY(endY, 14, cv.Pad.Top+10, cv.Pad.Top+cv.plotH())
	for i, l := range lines {
		cv.Text(cv.Pad.Left+cv.plotW()+6, endY[i], truncate(l.Label, 32), "axis-label vb-line-label "+l.Class, "start")
	}
	// Selubung atas: nilai terbaik pada setiap batas dan ujung sumbu.
	knots := []float64{0}
	for _, b := range bands {
		if b.From > 0 {
			knots = append(knots, b.From)
		}
	}
	knots = append(knots, vmax)
	sort.Float64s(knots)
	var ex, ey []float64
	for _, v := range knots {
		best := math.Inf(-1)
		for _, l := range lines {
			best = math.Max(best, net(l, v))
		}
		ex, ey = append(ex, v), append(ey, best)
	}
	cv.PolyLine(ex, ey, "vb-envelope")
	return template.HTML(cv.SVG("value-bands"))
}

// StripRow adalah satu baris skenario: rentang nilai per opsi terbaik.
type StripRow struct {
	Label string
	Bands []BandSpan
	Same  bool
	Base  bool
}

// LegendItem adalah satu entri legenda.
type LegendItem struct {
	Label string
	Class string
}

// ScenarioStrips menggambar rentang opsi terbaik untuk setiap skenario
// sebagai pita bertumpuk pada sumbu nilai yang sama, sehingga pergeseran
// batas dan perubahan urutan terlihat sekali pandang.
func ScenarioStrips(rows []StripRow, vmax float64, legend []LegendItem, lang string) template.HTML {
	rowH := 34.0
	cv := NewCanvas(880, 112+rowH*float64(len(rows)), Padding{Top: 66, Right: 70, Bottom: 44, Left: 230})
	cv.Describe(tr(lang, "Rekomendasi per skenario asumsi", "Recommendation per assumption scenario"), tr(lang,
		"Setiap baris adalah satu skenario. Warna menunjukkan opsi terbaik pada setiap nilai satu hari lebih cepat; tanda di kanan menunjukkan apakah urutan opsinya sama dengan pembanding.",
		"Each row is one scenario. Colour shows the best option at each value of a day earlier; the mark on the right shows whether its option order matches the baseline."))
	if len(rows) == 0 || vmax <= 0 {
		return template.HTML(cv.SVG("scenario-strips"))
	}
	cv.SetDomain(0, vmax, 0, 1)
	x0, y0 := 12.0, 12.0
	for _, lg := range legend {
		w := 19 + textWidth(lg.Label, 10.5) + 22
		if x0+w > cv.W-12 {
			x0, y0 = 12, y0+16
		}
		cv.Rect(x0, y0, 14, 10, "strip-seg "+lg.Class)
		cv.Text(x0+19, y0+9, lg.Label, "axis-label", "start")
		x0 += w
	}
	var xt []Tick
	for _, v := range NiceTicks(0, vmax, 6) {
		xt = append(xt, Tick{v, RpShort(v, lang)})
	}
	cv.AxisX(xt, true)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-8, tr(lang, "nilai satu hari lebih cepat (rupiah per hari)", "value of a day earlier (rupiah per day)"), "axis-title", "middle")
	for i, r := range rows {
		y := cv.Pad.Top + rowH*float64(i) + 6
		cls := "strip-row"
		if r.Base {
			cls += " base"
		}
		cv.Group(cls)
		cv.Text(12, y+15, truncate(r.Label, 34), "row-label", "start")
		for _, b := range r.Bands {
			if b.From >= vmax {
				continue
			}
			x1, x2 := cv.X(b.From), cv.X(math.Min(b.To, vmax))
			cv.Rect(x1, y, math.Max(x2-x1, 0.5), rowH-12, "strip-seg "+b.Class).Title(b.Label + ": " + Rp(b.From, lang))
		}
		mark, mcls := "✓", "strip-mark same"
		if !r.Same {
			mark, mcls = "✗", "strip-mark diff"
		}
		cv.Text(cv.W-cv.Pad.Right+22, y+16, mark, mcls, "middle")
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("scenario-strips"))
}

// BridgeStep adalah satu langkah jembatan anggaran.
type BridgeStep struct {
	Label string
	Value float64
	Kind  string // base, delta, total
	Note  string
}

// BridgeChart menggambar jembatan dari satu angka ke angka lain: batang dasar,
// kenaikan yang menumpuk, dan batang total, dengan nilai tertulis di setiap
// batang.
func BridgeChart(steps []BridgeStep, title, desc, lang string) template.HTML {
	cv := NewCanvas(880, 340, Padding{Top: 34, Right: 24, Bottom: 70, Left: 112})
	cv.Describe(title, desc)
	if len(steps) == 0 {
		return template.HTML(cv.SVG("bridge"))
	}
	maxV, run := 0.0, 0.0
	for _, s := range steps {
		switch s.Kind {
		case "delta":
			run += s.Value
		default:
			run = s.Value
		}
		maxV = math.Max(maxV, run)
	}
	cv.SetDomain(0, float64(len(steps)), 0, maxV*1.15)
	var yt []Tick
	for _, v := range NiceTicks(0, maxV*1.15, 6) {
		yt = append(yt, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yt, true)
	w := cv.plotW() / float64(len(steps)) * 0.56
	run = 0
	prevTop := 0.0
	for i, s := range steps {
		cx := cv.X(float64(i) + 0.5)
		var lo, hi float64
		switch s.Kind {
		case "delta":
			lo, hi = run, run+s.Value
			run = hi
		default:
			lo, hi = 0, s.Value
			run = s.Value
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		cls := "bridge-bar " + s.Kind
		if s.Kind == "delta" && s.Value < 0 {
			cls += " down"
		}
		cv.Rect(cx-w/2, cv.Y(hi), w, cv.Y(lo)-cv.Y(hi), cls).Title(s.Label + ": " + Rp(s.Value, lang))
		if i > 0 {
			cv.Line(cv.X(float64(i)-0.5)+w/2, cv.Y(prevTop), cx-w/2, cv.Y(prevTop), "bridge-link")
		}
		prevTop = run
		val := RpShort(s.Value, lang)
		if s.Kind == "delta" && s.Value >= 0 {
			val = "+" + val
		}
		cv.Text(cx, cv.Y(hi)-7, val, "bar-value bridge-value", "middle")
		cv.Text(cx, cv.H-cv.Pad.Bottom+18, truncate(s.Label, 30), "axis-label", "middle")
		if s.Note != "" {
			cv.Text(cx, cv.H-cv.Pad.Bottom+32, truncate(s.Note, 34), "axis-label dim", "middle")
		}
	}
	return template.HTML(cv.SVG("bridge"))
}

// CurvePoint adalah satu titik kurva durasi-biaya.
type CurvePoint struct {
	Day       float64
	Cost, Net float64
	Note      string
}

// OvertimeCurveChart menggambar upah lembur dan biaya bersih setelah sewa untuk
// setiap durasi yang bisa dibeli, mulai dari durasi tanpa lembur. compare,
// bila diisi, adalah kurva pembanding (mis. crashing CPM untuk hari yang sama).
func OvertimeCurveChart(base float64, pts, compare []CurvePoint, compareLabel string, cal *workcal.Calendar, lang string) template.HTML {
	cv := NewCanvas(880, 340, Padding{Top: 30, Right: 160, Bottom: 58, Left: 112})
	cv.Describe(tr(lang, "Harga setiap hari yang dibeli dengan lembur", "The price of each day bought with overtime"), tr(lang,
		"Titik paling kanan adalah jadwal tanpa lembur. Setiap titik ke kiri adalah satu durasi lebih pendek: garis tegas adalah upah lembur, garis tipis adalah biaya bersih setelah sewa yang ikut memendek. Makin curam garisnya, makin mahal hari tambahan.",
		"The rightmost point is the schedule without overtime. Each point to the left is a shorter duration: the bold line is overtime pay, the thin line the net cost after rentals that shorten too. The steeper the line, the pricier each extra day."))
	if len(pts) == 0 {
		return template.HTML(cv.SVG("overtime-curve"))
	}
	all := append([]CurvePoint{{Day: base}}, pts...)
	sort.Slice(all, func(i, j int) bool { return all[i].Day < all[j].Day })
	lo, hi := all[0].Day, all[len(all)-1].Day
	maxY := 0.0
	for _, p := range append(append([]CurvePoint(nil), all...), compare...) {
		maxY = math.Max(maxY, math.Max(p.Cost, p.Net))
	}
	cv.SetDomain(lo-1, hi+1, 0, maxY*1.15)
	var yt []Tick
	for _, v := range NiceTicks(0, maxY*1.15, 6) {
		yt = append(yt, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yt, true)
	cv.AxisX(dateTicks(cal, lo-1, hi+1, 7, lang), false)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-10, tr(lang, "tanggal selesai   ← lebih cepat", "finish date   ← faster"), "axis-title", "middle")
	var xs, cost, netv []float64
	for _, p := range all {
		xs, cost, netv = append(xs, p.Day), append(cost, p.Cost), append(netv, p.Net)
	}
	if len(compare) > 0 {
		cp := append([]CurvePoint{{Day: base}}, compare...)
		sort.Slice(cp, func(i, j int) bool { return cp[i].Day < cp[j].Day })
		var cx, cy []float64
		for _, p := range cp {
			cx, cy = append(cx, p.Day), append(cy, p.Cost)
		}
		cv.PolyLine(cx, cy, "oc-compare")
		mid := len(cx) / 2
		cv.Text(cv.X(cx[mid]), cv.Y(cy[mid])+18, compareLabel, "axis-label oc-compare-label", "middle")
	}
	cv.PolyLine(xs, netv, "oc-net")
	cv.PolyLine(xs, cost, "oc-pay")
	for _, p := range all {
		x := cv.X(p.Day)
		cv.Circle(x, cv.Y(p.Cost), 4.5, "oc-dot").Title(finishLabel(cal, p.Day, lang) + ": " + Rp(p.Cost, lang) + " " + p.Note)
	}
	first := all[0]
	cv.Text(cv.X(first.Day)+8, cv.Y(first.Cost)-10, tr(lang, "lantai: ", "floor: ")+Num(first.Day, 0, lang)+" · "+RpShort(first.Cost, lang), "bar-value oc-floor", "start")
	cv.Text(cv.Pad.Left+cv.plotW()+8, cv.Y(first.Cost), tr(lang, "upah lembur", "overtime pay"), "series-label oc-pay-label", "start")
	cv.Text(cv.Pad.Left+cv.plotW()+8, cv.Y(first.Net)+14, tr(lang, "bersih setelah sewa", "net after rentals"), "axis-label oc-net-label", "start")
	return template.HTML(cv.SVG("overtime-curve"))
}

// CredRow adalah satu panel kredibilitas.
type CredRow struct {
	Label           string
	Prior, Observed float64 // nilai rencana dan nilai teramati
	Noise           float64 // simpangan baku derau estimasi
	Z, Blended      float64 // bobot kredibilitas dan nilai yang dipakai
	Meaning         string
}

// CredibilityPanels menggambar kredibilitas Buhlmann sebagai tiga garis
// bilangan: pita derau di sekitar rencana, nilai teramati, dan nilai yang
// akhirnya dipakai. Bila titik teramati jatuh di dalam pita, data belum cukup
// kuat untuk menggeser rencana.
func CredibilityPanels(rows []CredRow, lang string) template.HTML {
	panelH := 118.0
	cv := NewCanvas(880, 30+panelH*float64(len(rows)), Padding{Top: 20, Right: 40, Bottom: 10, Left: 230})
	cv.Describe(tr(lang, "Apa yang dipelajari dari realisasi", "What the actuals teach"), tr(lang,
		"Untuk durasi, biaya, dan kapasitas ujian: pita abu-abu adalah rentang yang bisa dijelaskan derau estimasi (95%), garis tegak adalah rencana, titik adalah hasil teramati, dan belah ketupat adalah nilai yang dipakai prakiraan. Z adalah seberapa jauh prakiraan bergeser dari rencana ke hasil teramati.",
		"For duration, cost, and exam capacity: the grey band is the range estimation noise can explain (95%), the vertical line is the plan, the dot is the observed result, and the diamond is the value the forecast uses. Z is how far the forecast moves from plan toward the observation."))
	for i, r := range rows {
		top := cv.Pad.Top + panelH*float64(i)
		lo := math.Min(math.Min(r.Prior-2.4*r.Noise, r.Observed), r.Blended)
		hi := math.Max(math.Max(r.Prior+2.4*r.Noise, r.Observed), r.Blended)
		span := math.Max(hi-lo, 0.02)
		lo, hi = lo-span*0.12, hi+span*0.12
		x := func(v float64) float64 { return cv.Pad.Left + (v-lo)/(hi-lo)*(cv.W-cv.Pad.Left-cv.Pad.Right) }
		axisY := top + 64
		cv.Group("cred-panel")
		cv.Text(12, top+30, r.Label, "row-label", "start")
		cv.Text(12, top+46, "Z = "+Pct(r.Z, 0, lang), "bar-value cred-z", "start")
		if r.Meaning != "" {
			cv.Text(12, top+62, truncate(r.Meaning, 40), "axis-label dim", "start")
		}
		if r.Noise > 0 {
			cv.Rect(x(r.Prior-1.96*r.Noise), axisY-16, x(r.Prior+1.96*r.Noise)-x(r.Prior-1.96*r.Noise), 32, "cred-noise")
		}
		cv.Line(cv.Pad.Left, axisY, cv.W-cv.Pad.Right, axisY, "axis-line")
		for _, v := range NiceTicks(lo, hi, 6) {
			cv.Line(x(v), axisY, x(v), axisY+4, "axis-line")
			cv.Text(x(v), axisY+16, Num(v, 2, lang), "axis-label", "middle")
		}
		px, ox, bx := x(r.Prior), x(r.Observed), x(r.Blended)
		cv.Line(px, axisY-22, px, axisY+2, "cred-prior")
		cv.Text(px, axisY+34, tr(lang, "rencana ", "plan ")+Num(r.Prior, 2, lang), "axis-label cred-prior-label", "middle")
		if math.Abs(bx-px) > 2 {
			cv.Path(fmt.Sprintf("M%s %s L%s %s", f(px), f(axisY-8), f(bx), f(axisY-8)), "cred-arrow")
		}
		cv.Circle(ox, axisY, 6, "cred-obs").Title(tr(lang, "teramati ", "observed ") + Num(r.Observed, 3, lang))
		cv.Path(fmt.Sprintf("M%s %s l7 7 l-7 7 l-7 -7 Z", f(bx), f(axisY-7)), "cred-blend")
		obsY := axisY + 34
		if math.Abs(ox-px) < 90 {
			obsY = axisY + 48
		}
		cv.Text(ox, obsY, tr(lang, "teramati ", "observed ")+Num(r.Observed, 3, lang), "axis-label cred-obs-label", "middle")
		cv.Text(bx, axisY-28, tr(lang, "dipakai ", "used ")+Num(r.Blended, 3, lang), "bar-value cred-blend-label", "middle")
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("credibility"))
}

// FrontierMark adalah satu titik yang ditandai pada grafik frontier.
type FrontierMark struct {
	Day, Budget float64
	Label       string
	Class       string
}

// FrontierLine menggambar anggaran minimum untuk keyakinan bersama 70% pada
// setiap tanggal selesai, dengan titik-titik penting ditandai.
func FrontierLine(days, budgets []float64, marks []FrontierMark, cal *workcal.Calendar, lang string) template.HTML {
	cv := NewCanvas(880, 340, Padding{Top: 30, Right: 40, Bottom: 58, Left: 112})
	cv.Describe(tr(lang, "Frontier JCL 70%", "70% JCL frontier"), tr(lang,
		"Garis menunjukkan anggaran terkecil yang memberi peluang 70% selesai tepat waktu DAN tepat anggaran untuk setiap tanggal selesai. Makin ke kiri tanggalnya, makin tinggi anggaran yang dibutuhkan; di kiri titik pertama, 70% tidak tercapai berapa pun anggarannya.",
		"The line shows the smallest budget that gives a 70% chance of finishing on time AND on budget for every finish date. The earlier the date, the higher the budget needed; left of the first point, 70% cannot be reached at any budget."))
	if n := min(len(days), len(budgets)); n < len(days) || n < len(budgets) {
		days, budgets = days[:n], budgets[:n]
	}
	if len(days) == 0 {
		return template.HTML(cv.SVG("frontier-line"))
	}
	// Ekor yang sudah datar tidak memberi informasi; potong beberapa hari
	// setelah anggaran turun ke 1% dari rentangnya.
	floor, top := budgets[len(budgets)-1], budgets[0]
	cut := len(days)
	for i, b := range budgets {
		if b-floor <= (top-floor)*0.01 {
			cut = i + 8
			break
		}
	}
	if cut < len(days) {
		days, budgets = days[:cut], budgets[:cut]
	}
	xlo, xhi := days[0], days[len(days)-1]
	ylo, yhi := budgets[len(budgets)-1], budgets[0]
	for _, m := range marks {
		xlo, xhi = math.Min(xlo, m.Day), math.Max(xhi, m.Day)
		ylo, yhi = math.Min(ylo, m.Budget), math.Max(yhi, m.Budget)
	}
	ys := math.Max(yhi-ylo, 1)
	cv.SetDomain(xlo-2, xhi+2, ylo-ys*0.1, yhi+ys*0.15)
	var yt []Tick
	for _, v := range NiceTicks(ylo-ys*0.1, yhi+ys*0.15, 6) {
		yt = append(yt, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yt, true)
	cv.AxisX(dateTicks(cal, xlo-2, xhi+2, 7, lang), false)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-10, tr(lang, "tanggal selesai", "finish date"), "axis-title", "middle")
	cv.Area(days, budgets, ylo-ys*0.1, "frontier-area")
	cv.PolyLine(days, budgets, "series frontier")
	cv.Text(cv.X(days[len(days)-1]), cv.Y(budgets[len(budgets)-1])-10, tr(lang, "di atas garis: peluang ≥ 70%", "above the line: chance ≥ 70%"), "axis-label", "end")
	lp := &labelPlacer{w: cv.W, h: cv.H - cv.Pad.Bottom}
	for i := 1; i < len(days); i++ {
		lp.avoidSegment(cv.X(days[i-1]), cv.Y(budgets[i-1]), cv.X(days[i]), cv.Y(budgets[i]))
	}
	for _, m := range marks {
		x, y := cv.X(m.Day), cv.Y(m.Budget)
		cv.Circle(x, y, 6, "fl-mark "+m.Class).Title(m.Label + ": " + finishLabel(cal, m.Day, lang) + ", " + Rp(m.Budget, lang))
		text := m.Label + " · " + RpShort(m.Budget, lang)
		lx, ly, anchor := lp.place(x, y, []string{text}, 10.5)
		cv.Text(lx, ly, text, "bar-value fl-label "+m.Class, anchor)
	}
	return template.HTML(cv.SVG("frontier-line"))
}

// WindowSpan adalah satu jendela ketersediaan pada linimasa kapasitas.
type WindowSpan struct {
	FromDay, ToDay int // indeks hari kerja, inklusif
	Label          string
	Factor         float64
	Roles          string
	Assumed        bool
}

// CapacityTimeline menggambar setiap jendela ketersediaan sebagai batang pada
// linimasa proyek: makin pekat batangnya, makin sedikit kapasitas tersisa.
func CapacityTimeline(windows []WindowSpan, projectEnd, endDay int, statusDay float64, cal *workcal.Calendar, lang string) template.HTML {
	rowH := 38.0
	cv := NewCanvas(880, 96+rowH*float64(len(windows)), Padding{Top: 46, Right: 90, Bottom: 34, Left: 250})
	cv.Describe(tr(lang, "Linimasa ketersediaan tim", "Team availability timeline"), tr(lang,
		"Setiap batang adalah satu periode ketika kapasitas tim turun. Panjang batang adalah lamanya, kepekatannya menunjukkan kapasitas yang tersisa. Garis merah adalah tanggal data.",
		"Each bar is a period when team capacity drops. Its length is the duration, its shade shows the capacity left. The red line is the data date."))
	if len(windows) == 0 || endDay <= 0 {
		return template.HTML(cv.SVG("capacity-timeline"))
	}
	cv.SetDomain(0, float64(endDay), 0, 1)
	if projectEnd > 0 {
		x2 := cv.X(float64(min(projectEnd, endDay)))
		cv.Rect(cv.X(0), cv.Pad.Top-6, x2-cv.X(0), cv.plotH()+6, "cap-project")
		cv.Text(x2-4, cv.Pad.Top-10, tr(lang, "jadwal levelling selesai", "levelled schedule ends"), "axis-label", "end")
	}
	cv.AxisX(dateTicks(cal, 1, float64(endDay), 11, lang), true)
	if statusDay > 0 && statusDay < float64(endDay) {
		x := cv.X(statusDay)
		cv.Line(x, cv.Pad.Top-10, x, cv.H-cv.Pad.Bottom, "status-line")
		cv.Text(x, cv.Pad.Top-24, tr(lang, "tanggal data", "data date"), "axis-label status", "middle")
	}
	for i, w := range windows {
		y := cv.Pad.Top + rowH*float64(i) + 6
		cv.Group("cap-row")
		cv.Text(12, y+13, truncate(w.Label, 38), "row-label", "start")
		cv.Text(12, y+26, truncate(w.Roles, 44), "axis-label dim", "start")
		x1, x2 := cv.X(float64(w.FromDay)), cv.X(float64(w.ToDay+1))
		level := max(1, min(5, int(math.Round((1-w.Factor)*5))))
		cls := fmt.Sprintf("cap-bar cap-%d", level)
		if w.Assumed {
			cls += " assumed"
		}
		cv.Rect(x1, y, math.Max(x2-x1, 2), rowH-14, cls).Title(w.Label + ": " + Pct(w.Factor, 0, lang))
		cv.Text(cv.W-cv.Pad.Right+10, y+16, tr(lang, "sisa ", "left ")+Pct(w.Factor, 0, lang), "bar-value", "start")
		cv.EndGroup()
	}
	return template.HTML(cv.SVG("capacity-timeline"))
}

// IndexRow adalah SPI dan CPI satu baris.
type IndexRow struct {
	Label    string
	SPI, CPI float64 // nol berarti belum ada
}

// IndexDots menggambar SPI dan CPI setiap fase pada garis yang sama dengan
// batas 1,0: di kiri garis berarti terlambat atau boros.
func IndexDots(rows []IndexRow, lang string) template.HTML {
	rowH := 34.0
	cv := NewCanvas(880, 90+rowH*float64(len(rows)), Padding{Top: 44, Right: 40, Bottom: 38, Left: 250})
	cv.Describe(tr(lang, "SPI dan CPI per fase", "SPI and CPI by phase"), tr(lang,
		"Lingkaran adalah SPI (jadwal), kotak adalah CPI (biaya). Garis tegak di 1,0 adalah sesuai rencana; area merah di kiri berarti terlambat atau boros.",
		"Circles are SPI (schedule), squares are CPI (cost). The vertical line at 1.0 means on plan; the red area to the left means late or over cost."))
	lo, hi := 0.7, 1.15
	for _, r := range rows {
		for _, v := range []float64{r.SPI, r.CPI} {
			if v > 0 {
				lo, hi = math.Min(lo, v-0.05), math.Max(hi, v+0.05)
			}
		}
	}
	cv.SetDomain(lo, hi, 0, 1)
	cv.Rect(cv.X(lo), cv.Pad.Top, cv.X(1)-cv.X(lo), cv.plotH(), "idx-bad")
	var xt []Tick
	for _, v := range NiceTicks(lo, hi, 7) {
		xt = append(xt, Tick{v, Num(v, 2, lang)})
	}
	cv.AxisX(xt, true)
	x1 := cv.X(1)
	cv.Line(x1, cv.Pad.Top-8, x1, cv.H-cv.Pad.Bottom, "ref-line divider")
	cv.Text(x1, cv.Pad.Top-12, tr(lang, "1,0 = sesuai rencana", "1.0 = on plan"), "axis-label", "middle")
	cv.Circle(14, 16, 5, "idx-spi")
	cv.Text(24, 20, tr(lang, "SPI jadwal", "SPI schedule"), "axis-label", "start")
	cv.Rect(110, 11, 10, 10, "idx-cpi")
	cv.Text(126, 20, tr(lang, "CPI biaya", "CPI cost"), "axis-label", "start")
	for i, r := range rows {
		y := cv.Pad.Top + rowH*float64(i) + rowH/2
		cv.Text(12, y+4, truncate(r.Label, 38), "row-label", "start")
		if r.SPI <= 0 && r.CPI <= 0 {
			cv.Text(cv.Pad.Left+6, y+4, tr(lang, "belum dimulai", "not started"), "axis-label dim", "start")
			continue
		}
		if r.SPI > 0 && r.CPI > 0 {
			cv.Line(cv.X(r.SPI), y, cv.X(r.CPI), y, "dd-guide")
		}
		if r.SPI > 0 {
			cv.Circle(cv.X(r.SPI), y, 6, "idx-spi").Title("SPI " + Num(r.SPI, 3, lang))
			cv.Text(cv.X(r.SPI), y-10, Num(r.SPI, 2, lang), "axis-label", "middle")
		}
		if r.CPI > 0 {
			cv.Rect(cv.X(r.CPI)-5, y-5, 10, 10, "idx-cpi").Title("CPI " + Num(r.CPI, 3, lang))
			cv.Text(cv.X(r.CPI), y+18, Num(r.CPI, 2, lang), "axis-label", "middle")
		}
	}
	return template.HTML(cv.SVG("index-dots"))
}

// PairBars menggambar dua nilai per kategori sebagai batang bertumpuk tipis:
// batang muda untuk nilai sebelum, batang pekat untuk nilai sesudah.
func PairBars(labels []string, before, after []float64, labelBefore, labelAfter, title, desc, lang string) template.HTML {
	if n := min(len(labels), len(before), len(after)); n < len(labels) {
		labels = labels[:n]
	}
	rowH := 38.0
	cv := NewCanvas(880, 80+rowH*float64(len(labels)), Padding{Top: 40, Right: 130, Bottom: 30, Left: 230})
	cv.Describe(title, desc)
	maxV := 0.0
	for i := range labels {
		maxV = math.Max(maxV, math.Max(before[i], after[i]))
	}
	if maxV == 0 {
		return template.HTML(cv.SVG("pair-bars"))
	}
	cv.SetDomain(0, maxV*1.05, 0, 1)
	cv.Rect(12, 12, 14, 10, "pair-bar before")
	cv.Text(31, 21, labelBefore, "axis-label", "start")
	cv.Rect(190, 12, 14, 10, "pair-bar after")
	cv.Text(209, 21, labelAfter, "axis-label", "start")
	for i, l := range labels {
		y := cv.Pad.Top + rowH*float64(i) + 6
		cv.Text(12, y+15, truncate(l, 34), "row-label", "start")
		cv.Rect(cv.X(0), y, cv.X(before[i])-cv.X(0), rowH-14, "pair-bar before").Title(labelBefore + ": " + Rp(before[i], lang))
		cv.Rect(cv.X(0), y+5, cv.X(after[i])-cv.X(0), rowH-24, "pair-bar after").Title(labelAfter + ": " + Rp(after[i], lang))
		drop := ""
		switch {
		case before[i] > 0 && after[i] < before[i]:
			drop = " (" + Pct(1-after[i]/before[i], 0, lang) + tr(lang, " turun)", " lower)")
		case before[i] > 0 && after[i] > before[i]:
			drop = " (" + Pct(after[i]/before[i]-1, 0, lang) + tr(lang, " naik)", " higher)")
		}
		label := RpShort(after[i], lang) + drop
		if lx := cv.X(before[i]) + 8; lx+textWidth(label, 10.5) > cv.W-8 {
			cv.Text(cv.X(before[i])-8, y+16, label, "bar-value", "end")
		} else {
			cv.Text(lx, y+16, label, "bar-value", "start")
		}
	}
	return template.HTML(cv.SVG("pair-bars"))
}
