package render

import (
	"fmt"
	"html/template"
	"math"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// GanttLevelled menggambar dua jadwal berdampingan untuk setiap aktivitas:
// batang bayangan menurut CPM dan batang padat menurut levelling. Jendela
// ketersediaan (periode ujian) diarsir di latar belakang, sehingga pembaca
// bisa melihat sendiri pekerjaan mana yang tertahan di sana.
func GanttLevelled(acts []model.Activity, plan schedule.Result, lv level.Result, cal *workcal.Calendar, lang string) template.HTML {
	rowH := 22.0
	labelW := 300.0
	headerH := 50.0
	n := float64(lv.Duration)
	if n < float64(plan.Duration) {
		n = float64(plan.Duration)
	}

	h := headerH + rowH*float64(len(acts)) + 24
	cv := NewCanvas(1080, h, Padding{Top: headerH, Right: 20, Bottom: 24, Left: labelW})
	cv.SetDomain(0, n, 0, 1).Describe(
		tr(lang, "Jadwal CPM dibanding jadwal levelling", "CPM schedule versus levelled schedule"),
		tr(lang,
			"Setiap aktivitas punya dua batang: batang tipis transparan adalah jadwal CPM yang mengabaikan kapasitas, batang padat adalah jadwal setelah levelling. Area berarsir adalah periode ketersediaan terbatas.",
			"Each activity has two bars: the thin translucent bar is the CPM schedule that ignores capacity, the solid bar is the schedule after levelling. Hatched areas are periods of reduced availability."))
	dayW := cv.plotW() / n

	for _, w := range model.AvailabilityWindows {
		from := cal.FractionalIndexOf(w.From)
		to := cal.FractionalIndexOf(w.To) + 1
		x1 := cv.Pad.Left + from*dayW
		x2 := cv.Pad.Left + math.Min(to, n)*dayW
		cv.Group("window").
			Rect(x1, headerH-6, x2-x1, cv.H-cv.Pad.Bottom-headerH+6, "window-band").
			Title(w.Label.Get(lang)).
			EndGroup()
		cv.Text((x1+x2)/2, headerH-10, tr(lang, "ujian", "exams"), "axis-label window", "middle")
	}

	lastMonth := -1
	for d := 0; d <= int(n); d++ {
		t := cal.Date(d)
		if int(t.Month()) != lastMonth {
			lastMonth = int(t.Month())
			x := cv.Pad.Left + float64(d)*dayW
			cv.Line(x, headerH-24, x, cv.H-cv.Pad.Bottom, "grid-line month")
			cv.Text(x+4, headerH-28, fmt.Sprintf("%s %d", workcal.MonthShort(t.Month(), lang), t.Year()), "axis-label month", "start")
		}
	}
	cpmEnd := cv.Pad.Left + float64(plan.Duration)*dayW
	cv.Line(cpmEnd, headerH-6, cpmEnd, cv.H-cv.Pad.Bottom, "ref-line")
	cv.Text(cpmEnd, cv.H-8, fmt.Sprintf("CPM %d", plan.Duration), "axis-label", "middle")
	lvEnd := cv.Pad.Left + float64(lv.Duration)*dayW
	cv.Line(lvEnd, headerH-6, lvEnd, cv.H-cv.Pad.Bottom, "status-line")
	cv.Text(lvEnd, cv.H-8, fmt.Sprintf("%s %d", tr(lang, "levelling", "levelled"), lv.Duration), "axis-label status", "middle")

	for i, a := range acts {
		y := headerH + float64(i)*rowH
		ct := plan.Task(a.ID)
		lt := lv.Tasks[a.ID]
		cls := "gantt-label"
		if lt.Delay() > 0 {
			cls += " moved"
		}
		cv.Text(labelW-10, y+rowH/2+4, a.ID+" - "+truncate(a.Name.Get(lang), 36), cls, "end")

		if a.Milestone {
			mx := cv.Pad.Left + float64(lt.Start)*dayW
			my := y + rowH/2
			cv.Path(fmt.Sprintf("M%s %s L%s %s L%s %s L%s %s Z", f(mx), f(my-6), f(mx+6), f(my), f(mx), f(my+6), f(mx-6), f(my)), "gantt-milestone")
			continue
		}
		cx1 := cv.Pad.Left + float64(ct.StartX)*dayW
		cx2 := cv.Pad.Left + float64(ct.FinishX)*dayW
		cv.Rect(cx1, y+3, math.Max(cx2-cx1, 2), 5, "ghost-bar")

		lx1 := cv.Pad.Left + float64(lt.Start)*dayW
		lx2 := cv.Pad.Left + float64(lt.Finish)*dayW
		bar := "lv-bar"
		switch {
		case lt.Window:
			bar += " cause-window"
		case lt.HalfTime:
			bar += " cause-halftime"
		case lt.WaitDays > 0 || lt.Shared:
			bar += " cause-shared"
		case lt.CarriedDays > 0:
			bar += " cause-carried"
		}
		cv.Group("lv").
			Rect(lx1, y+9, math.Max(lx2-lx1, 2), rowH-13, bar).
			Title(fmt.Sprintf("%s\nCPM: %s - %s\n%s: %s - %s\n%s %d | %s %d | %s %d",
				a.Name.Get(lang),
				workcal.FormatDateShort(cal.ISOAt(ct.StartX), lang), workcal.FormatDateShort(cal.ISOAt(ct.FinishX-1), lang),
				tr(lang, "levelling", "levelled"),
				workcal.FormatDateShort(cal.ISOAt(lt.Start), lang), workcal.FormatDateShort(cal.ISOAt(lt.Finish-1), lang),
				tr(lang, "terbawa", "carried"), lt.CarriedDays,
				tr(lang, "menunggu", "waiting"), lt.WaitDays,
				tr(lang, "memanjang", "stretched"), lt.StretchDays)).
			EndGroup()
	}
	return template.HTML(cv.SVG("gantt gantt-levelled"))
}

// CrashCurve menggambar kurva waktu-biaya: di sumbu mendatar durasi proyek
// yang makin pendek, di sumbu tegak biaya crashing kumulatif. Kelengkungan
// kurvanya adalah pesan utamanya - hari-hari pertama murah, hari-hari terakhir
// mahal, dan ada titik di mana uang tidak bisa membeli hari lagi.
func CrashCurve(c compress.Curve, lang string) template.HTML {
	cv := NewCanvas(880, 340, Padding{Top: 28, Right: 30, Bottom: 50, Left: 90})
	if len(c.Steps) == 0 {
		return template.HTML(cv.SVG("crash-curve"))
	}
	maxCost := c.Steps[len(c.Steps)-1].TotalCost * 1.1
	cv.SetDomain(float64(c.MinDuration)-1, float64(c.NormalDuration)+1, 0, maxCost).Describe(
		tr(lang, "Kurva waktu-biaya crashing", "Crashing time-cost curve"),
		tr(lang,
			"Durasi proyek setelah setiap langkah crashing terhadap biaya tambahan kumulatif; setiap titik adalah satu hari yang dibeli.",
			"Project duration after each crashing step against cumulative extra cost; every point is one day bought."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, maxCost, 6) {
		yTicks = append(yTicks, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yTicks, true)
	var xTicks []Tick
	for d := c.MinDuration; d <= c.NormalDuration; d += 2 {
		xTicks = append(xTicks, Tick{float64(d), fmt.Sprintf("%d", d)})
	}
	cv.AxisX(xTicks, false)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-12, tr(lang, "durasi proyek (hari kerja) - makin ke kiri makin dipercepat", "project duration (working days) - further left is faster"), "axis-title", "middle")

	xs := []float64{float64(c.NormalDuration)}
	ys := []float64{0}
	for _, s := range c.Steps {
		xs = append(xs, float64(s.Duration))
		ys = append(ys, s.TotalCost)
	}
	cv.Area(xs, ys, 0, "crash-area")
	cv.PolyLine(xs, ys, "series crash")
	for _, s := range c.Steps {
		cls := "point crash-pt"
		if len(s.Crashed) > 1 {
			cls += " multi"
		}
		cv.Group("pt").
			Circle(cv.X(float64(s.Duration)), cv.Y(s.TotalCost), 4, cls).
			Title(fmt.Sprintf("%d %s: %s %v, %s %s, %s %s",
				s.Duration, tr(lang, "hari", "days"),
				tr(lang, "potong", "cut"), s.Crashed,
				tr(lang, "langkah", "step"), Rp(s.StepCost, lang),
				tr(lang, "kumulatif", "cumulative"), Rp(s.TotalCost, lang))).
			EndGroup()
	}
	return template.HTML(cv.SVG("crash-curve"))
}

// LadderChart menggambar tangga realisme: bagaimana P50 dan P80 bergerak saat
// setiap sumber ketidakpastian ditambahkan. metric bernilai "dur" atau "cost".
func LadderChart(ladder []simulate.IntegratedResult, names []model.Text, metric string, target float64, lang string) template.HTML {
	cv := NewCanvas(880, 320, Padding{Top: 34, Right: 24, Bottom: 62, Left: 90})
	if len(ladder) == 0 {
		return template.HTML(cv.SVG("ladder"))
	}
	val := func(r simulate.IntegratedResult) (p50, p80, p90 float64) {
		if metric == "cost" {
			return r.CostP50, r.CostP80, r.CostP90
		}
		return r.DurP50, r.DurP80, r.DurP90
	}
	label := func(v float64) string {
		if metric == "cost" {
			return RpShort(v, lang)
		}
		return Num(v, 0, lang)
	}
	lo, hi := target, target
	for _, r := range ladder {
		p50, _, p90 := val(r)
		lo = math.Min(lo, p50)
		hi = math.Max(hi, p90)
	}
	span := hi - lo
	lo -= span * 0.15
	hi += span * 0.12
	if lo < 0 {
		lo = 0
	}
	n := float64(len(ladder))
	cv.SetDomain(0, n, lo, hi)
	title := tr(lang, "Tangga realisme: durasi", "Realism ladder: duration")
	if metric == "cost" {
		title = tr(lang, "Tangga realisme: biaya", "Realism ladder: cost")
	}
	cv.Describe(title, tr(lang,
		"Untuk setiap lapisan realisme, batang menunjukkan rentang P50 sampai P90 dan penanda menunjukkan P80; garis putus-putus adalah target Project Charter.",
		"For each realism layer the bar spans P50 to P90 and the marker shows P80; the dashed line is the Project Charter target."))

	var yTicks []Tick
	for _, v := range NiceTicks(lo, hi, 6) {
		yTicks = append(yTicks, Tick{v, label(v)})
	}
	cv.AxisY(yTicks, true)
	ty := cv.Y(target)
	cv.Line(cv.Pad.Left, ty, cv.Pad.Left+cv.plotW(), ty, "ref-line target")
	cv.Text(cv.Pad.Left+cv.plotW()-4, ty-6, tr(lang, "target piagam ", "charter target ")+label(target), "axis-label", "end")

	barW := cv.plotW() / n * 0.42
	var prevX, prevY float64
	for i, r := range ladder {
		p50, p80, p90 := val(r)
		x := cv.X(float64(i) + 0.5)
		cv.Group("ladder-step").
			Rect(x-barW/2, cv.Y(p90), barW, cv.Y(p50)-cv.Y(p90), fmt.Sprintf("ladder-bar step-%d", i)).
			Title(fmt.Sprintf("%s: P50 %s, P80 %s, P90 %s", names[i].Get(lang), label(p50), label(p80), label(p90))).
			EndGroup()
		y80 := cv.Y(p80)
		cv.Line(x-barW/2-4, y80, x+barW/2+4, y80, "ladder-p80")
		cv.Text(x, y80-7, "P80 "+label(p80), "bar-value", "middle")
		if i > 0 {
			cv.Path(fmt.Sprintf("M%s %s L%s %s", f(prevX+barW/2+4), f(prevY), f(x-barW/2-4), f(y80)), "ladder-link")
		}
		prevX, prevY = x, y80
		cv.Text(x, cv.Y(lo)+18, names[i].Get(lang), "axis-label", "middle")
		cv.Text(x, cv.Y(lo)+33, fmt.Sprintf("L%d", i), "axis-label dim", "middle")
	}
	return template.HTML(cv.SVG("ladder"))
}

// JCLHeatmap menggambar peta kepadatan pasangan durasi-biaya dari simulasi
// terpadu, beserta kurva frontier JCL 70% dan titik target Project Charter.
//
// Sepuluh ribu titik sebar akan membuat halaman bengkak ratusan kilobita,
// jadi yang digambar adalah histogram dua dimensi: satu sel per rentang
// durasi dan biaya, dengan kepekatan menurut jumlah iterasi di dalamnya.
func JCLHeatmap(g simulate.Grid, frontier []simulate.FrontierPoint, deadline, budget, p80Dur, p80Cost float64, lang string) template.HTML {
	cv := NewCanvas(880, 460, Padding{Top: 30, Right: 30, Bottom: 56, Left: 96})
	durLo := math.Min(g.DurMin, deadline) - 2
	durHi := g.DurMax + 1
	costLo := math.Min(g.CostMin, budget) * 0.97
	costHi := g.CostMax * 1.02
	cv.SetDomain(durLo, durHi, costLo, costHi).Describe(
		tr(lang, "Peta kepadatan Joint Confidence Level", "Joint Confidence Level density map"),
		tr(lang,
			"Setiap sel menunjukkan berapa iterasi simulasi yang berakhir pada kombinasi durasi dan biaya tersebut. Garis adalah anggaran minimum untuk keyakinan bersama 70% pada setiap tenggat; tanda silang adalah target Project Charter.",
			"Each cell shows how many simulation iterations ended at that duration-cost combination. The line is the minimum budget for 70% joint confidence at each deadline; the cross marks the Project Charter target."))

	var yTicks []Tick
	for _, v := range NiceTicks(costLo, costHi, 6) {
		yTicks = append(yTicks, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yTicks, true)
	var xTicks []Tick
	for _, v := range NiceTicks(durLo, durHi, 9) {
		xTicks = append(xTicks, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisX(xTicks, false)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-12, tr(lang, "durasi proyek (hari kerja)", "project duration (working days)"), "axis-title", "middle")

	cellW := (g.DurMax - g.DurMin) / float64(g.Cols)
	cellH := (g.CostMax - g.CostMin) / float64(g.Rows)
	for r := 0; r < g.Rows; r++ {
		for c := 0; c < g.Cols; c++ {
			cnt := g.Counts[r][c]
			if cnt == 0 {
				continue
			}
			level := 1 + int(math.Floor(5*math.Sqrt(float64(cnt)/float64(g.Max))))
			if level > 6 {
				level = 6
			}
			d0 := g.DurMin + float64(c)*cellW
			c0 := g.CostMin + float64(r)*cellH
			x1, x2 := cv.X(d0), cv.X(d0+cellW)
			y1, y2 := cv.Y(c0+cellH), cv.Y(c0)
			cv.Rect(x1, y1, x2-x1+0.4, y2-y1+0.4, fmt.Sprintf("dens dens-%d", level))
		}
	}

	var fx, fy []float64
	for _, p := range frontier {
		if p.Feasible {
			fx = append(fx, p.Duration)
			fy = append(fy, p.Budget)
		}
	}
	if len(fx) > 1 {
		cv.PolyLine(fx, fy, "series frontier")
		cv.Text(cv.X(fx[len(fx)-1]), cv.Y(fy[len(fy)-1])-8, "JCL 70%", "series-label frontier", "end")
	}

	tx, tyy := cv.X(deadline), cv.Y(budget)
	cv.Line(tx, cv.Pad.Top, tx, cv.H-cv.Pad.Bottom, "ref-line target")
	cv.Line(cv.Pad.Left, tyy, cv.Pad.Left+cv.plotW(), tyy, "ref-line target")
	cv.Path(fmt.Sprintf("M%s %s l8 8 M%s %s l-8 8", f(tx-4), f(tyy-4), f(tx+4), f(tyy-4)), "target-cross")
	cv.Text(tx+8, tyy-8, tr(lang, "target piagam", "charter target"), "axis-label target-label", "start")

	px, py := cv.X(p80Dur), cv.Y(p80Cost)
	cv.Circle(px, py, 5, "p80-marker")
	cv.Text(px+8, py+4, "P80 × P80", "axis-label", "start")
	return template.HTML(cv.SVG("jcl-heatmap"))
}

// ValueHistogram menggambar histogram nilai sembarang dengan penanda vertikal.
// Dipakai untuk sebaran biaya, yang sumbunya rupiah, bukan hari.
func ValueHistogram(sorted []float64, bins int, markers []Marker, money bool, lang string) template.HTML {
	cv := NewCanvas(880, 300, Padding{Top: 30, Right: 30, Bottom: 48, Left: 70})
	if len(sorted) == 0 {
		return template.HTML(cv.SVG("histogram"))
	}
	lo, hi := sorted[0], sorted[len(sorted)-1]
	for _, m := range markers {
		lo = math.Min(lo, m.Value)
		hi = math.Max(hi, m.Value)
	}
	if hi == lo {
		hi = lo + 1
	}
	width := (hi - lo) / float64(bins)
	counts := make([]int, bins)
	maxC := 0
	for _, v := range sorted {
		i := int((v - lo) / width)
		if i >= bins {
			i = bins - 1
		}
		if i < 0 {
			i = 0
		}
		counts[i]++
		if counts[i] > maxC {
			maxC = counts[i]
		}
	}
	cv.SetDomain(lo, hi, 0, float64(maxC)*1.15).Describe(
		tr(lang, "Sebaran hasil simulasi", "Simulated outcome distribution"),
		tr(lang, "Histogram hasil simulasi dengan penanda persentil dan target.", "Histogram of simulation outcomes with percentile and target markers."))
	fmtV := func(v float64) string {
		if money {
			return RpShort(v, lang)
		}
		return Num(v, 0, lang)
	}
	var yTicks []Tick
	for _, v := range NiceTicks(0, float64(maxC), 5) {
		yTicks = append(yTicks, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisY(yTicks, true)
	var xTicks []Tick
	for _, v := range NiceTicks(lo, hi, 7) {
		xTicks = append(xTicks, Tick{v, fmtV(v)})
	}
	cv.AxisX(xTicks, false)
	for i, c := range counts {
		x1 := cv.X(lo + float64(i)*width)
		x2 := cv.X(lo + float64(i+1)*width)
		y := cv.Y(float64(c))
		cv.Rect(x1+0.5, y, math.Max(x2-x1-1, 0.5), cv.Y(0)-y, "hist-bar")
	}
	for i, m := range markers {
		x := cv.X(m.Value)
		cv.Line(x, cv.Pad.Top, x, cv.H-cv.Pad.Bottom, "marker "+m.Class)
		cv.Text(x, cv.Pad.Top-8-float64(i%2)*12, m.Label+" "+fmtV(m.Value), "axis-label marker "+m.Class, "middle")
	}
	return template.HTML(cv.SVG("histogram"))
}

// Marker adalah satu garis penanda pada histogram.
type Marker struct {
	Value float64
	Label string
	Class string
}
