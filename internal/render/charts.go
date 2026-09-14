package render

import (
	"fmt"
	"html/template"
	"math"
	"sort"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/evm"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/quality"
	"github.com/xyb3rpunq/mppl-control-tower/internal/resource"
	"github.com/xyb3rpunq/mppl-control-tower/internal/risk"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// SCurve menggambar kurva-S Earned Value: PV, EV, AC, dan proyeksi biaya.
//
// Kurva-S adalah grafik paling padat informasi dalam manajemen proyek. Jarak
// vertikal antara EV dan PV adalah varians jadwal; jarak antara EV dan AC
// adalah varians biaya; dan jarak HORIZONTAL antara EV dan PV adalah
// keterlambatan dalam satuan waktu - yang terakhir inilah yang biasanya
// terlewat karena orang hanya membaca jarak vertikal.
func SCurve(c Curves, cal *workcal.Calendar, lang string, bac float64) template.HTML {
	cv := NewCanvas(880, 380, Padding{Top: 24, Right: 120, Bottom: 46, Left: 78})
	n := float64(len(c.Days) - 1)
	maxY := bac
	for _, v := range c.Forecast {
		if !math.IsNaN(v) && v > maxY {
			maxY = v
		}
	}
	maxY *= 1.06
	cv.SetDomain(0, n, 0, maxY).Describe(
		tr(lang, "Kurva-S Earned Value", "Earned Value S-curve"),
		tr(lang,
			"Tiga kurva kumulatif: nilai rencana, nilai yang diperoleh, dan biaya aktual, ditambah proyeksi biaya sampai akhir proyek.",
			"Three cumulative curves: planned value, earned value, and actual cost, plus the cost forecast to completion."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, maxY, 6) {
		yTicks = append(yTicks, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yTicks, true)

	var xTicks []Tick
	for d := 0; d <= int(n); d += 10 {
		xTicks = append(xTicks, Tick{float64(d), workcal.FormatDateShort(cal.ISOAt(d), lang)})
	}
	cv.AxisX(xTicks, false)

	// Garis BAC sebagai acuan tetap.
	cv.Line(cv.X(0), cv.Y(bac), cv.X(n), cv.Y(bac), "ref-line")
	cv.Text(cv.X(n)+6, cv.Y(bac)+4, "BAC", "series-label bac", "start")

	cv.PolyLine(c.Days, c.PV, "series pv")
	cv.PolyLine(c.Days[:len(c.EV)], c.EV, "series ev")
	cv.PolyLine(c.Days[:len(c.AC)], c.AC, "series ac")
	cv.PolyLine(c.Days, c.Forecast, "series forecast")

	// Garis tanggal data.
	sx := cv.X(float64(c.CutoffIndex))
	cv.Line(sx, cv.Pad.Top, sx, cv.H-cv.Pad.Bottom, "status-line")
	cv.Text(sx, cv.Pad.Top-8, tr(lang, "tanggal data", "data date"), "axis-label status", "middle")

	// Label seri di ujung kanan, supaya legenda tidak perlu dibaca bolak-balik.
	if len(c.EV) > 0 {
		cv.Text(cv.X(float64(len(c.EV)-1))+6, cv.Y(c.EV[len(c.EV)-1]), "EV", "series-label ev", "start")
		cv.Text(cv.X(float64(len(c.AC)-1))+6, cv.Y(c.AC[len(c.AC)-1])+12, "AC", "series-label ac", "start")
	}
	cv.Text(cv.X(n)+6, cv.Y(c.PV[len(c.PV)-1])-10, "PV", "series-label pv", "start")
	if v := c.Forecast[len(c.Forecast)-1]; !math.IsNaN(v) {
		cv.Text(cv.X(n)+6, cv.Y(v), "EAC", "series-label forecast", "start")
	}
	return template.HTML(cv.SVG("s-curve"))
}

// Curves adalah alias lokal agar templat tidak perlu mengimpor paket evm.
type Curves = evm.Curves

// SimHistogram menggambar histogram hasil Monte Carlo dengan kurva kumulatif
// dan penanda P50/P80/P90 serta durasi rencana.
func SimHistogram(r simulate.Result, lang string) template.HTML {
	cv := NewCanvas(880, 360, Padding{Top: 28, Right: 56, Bottom: 52, Left: 62})
	if len(r.Histogram) == 0 {
		return template.HTML(cv.SVG("histogram"))
	}
	maxCount := 0
	for _, b := range r.Histogram {
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}
	lo := r.Histogram[0].From
	hi := r.Histogram[len(r.Histogram)-1].To
	cv.SetDomain(lo, hi, 0, float64(maxCount)*1.12).Describe(
		tr(lang, "Sebaran durasi proyek hasil simulasi", "Simulated project duration distribution"),
		tr(lang,
			"Histogram durasi akhir proyek dari ribuan iterasi Monte Carlo, dengan kurva probabilitas kumulatif dan penanda P50, P80, serta P90.",
			"Histogram of final project duration across thousands of Monte Carlo iterations, with the cumulative probability curve and P50, P80, and P90 markers."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, float64(maxCount), 5) {
		yTicks = append(yTicks, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisY(yTicks, true)
	var xTicks []Tick
	for _, v := range NiceTicks(lo, hi, 8) {
		xTicks = append(xTicks, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisX(xTicks, false)

	for _, b := range r.Histogram {
		x1, x2 := cv.X(b.From), cv.X(b.To)
		y := cv.Y(float64(b.Count))
		base := cv.Y(0)
		cv.Group("bin").
			Rect(x1+0.5, y, x2-x1-1, base-y, "hist-bar").
			Title(fmt.Sprintf("%s-%s %s: %s %s",
				Num(b.From, 0, lang), Num(b.To, 0, lang), tr(lang, "hari", "days"),
				Num(float64(b.Count), 0, lang), tr(lang, "iterasi", "iterations"))).
			EndGroup()
	}

	// Kurva kumulatif memakai sumbu Y kanan secara implisit (0..maxCount).
	var cx, cy []float64
	for _, b := range r.Histogram {
		cx = append(cx, (b.From+b.To)/2)
		cy = append(cy, b.Cumulative*float64(maxCount)*1.12)
	}
	cv.PolyLine(cx, cy, "series cumulative")

	markers := []struct {
		v     float64
		label string
		class string
	}{
		{float64(r.Deterministic), tr(lang, "rencana", "plan"), "marker plan"},
		{r.P50, "P50", "marker p50"},
		{r.P80, "P80", "marker p80"},
		{r.P90, "P90", "marker p90"},
	}
	for i, m := range markers {
		x := cv.X(m.v)
		cv.Line(x, cv.Pad.Top, x, cv.H-cv.Pad.Bottom, m.class)
		cv.Text(x, cv.Pad.Top-10+float64(i%2)*-0. /*sejajar*/, m.label+" "+Num(m.v, 0, lang), "axis-label "+m.class, "middle")
	}
	return template.HTML(cv.SVG("histogram"))
}

// Tornado menggambar diagram sensitivitas: aktivitas mana yang paling kuat
// menggerakkan durasi total proyek.
func Tornado(sens []simulate.Sensitivity, limit int, lang string) template.HTML {
	if len(sens) > limit {
		sens = sens[:limit]
	}
	rowH := 26.0
	cv := NewCanvas(880, 40+rowH*float64(len(sens))+34, Padding{Top: 30, Right: 150, Bottom: 30, Left: 320})
	maxAbs := 0.0
	for _, s := range sens {
		if a := math.Abs(s.Correlation); a > maxAbs {
			maxAbs = a
		}
	}
	if maxAbs == 0 {
		maxAbs = 1
	}
	cv.SetDomain(0, maxAbs*1.05, 0, float64(len(sens))).Describe(
		tr(lang, "Diagram tornado sensitivitas jadwal", "Schedule sensitivity tornado"),
		tr(lang,
			"Korelasi peringkat Spearman antara durasi tiap aktivitas dan durasi total proyek, diurutkan dari yang paling berpengaruh.",
			"Spearman rank correlation between each activity duration and total project duration, strongest first."))

	for i, s := range sens {
		y := cv.Pad.Top + float64(i)*rowH
		w := math.Abs(s.Correlation) / (maxAbs * 1.05) * cv.plotW()
		cls := "tornado-bar"
		if s.CriticalRate > 0.9 {
			cls += " always-critical"
		}
		cv.Group("tornado-row").
			Rect(cv.Pad.Left, y+4, w, rowH-10, cls).
			Title(fmt.Sprintf("%s - r = %s, %s %s",
				s.ID, Num(s.Correlation, 3, lang),
				tr(lang, "berada di jalur kritis pada", "on the critical path in"),
				Pct(s.CriticalRate, 0, lang))).
			EndGroup()
		label := s.ID + " - " + truncate(s.Name.Get(lang), 42)
		cv.Text(cv.Pad.Left-10, y+rowH/2+1, label, "row-label", "end")
		cv.Text(cv.Pad.Left+w+8, y+rowH/2+1,
			fmt.Sprintf("r=%s - %s %s", Num(s.Correlation, 2, lang), Pct(s.CriticalRate, 0, lang),
				tr(lang, "kritis", "critical")), "row-value", "start")
	}
	return template.HTML(cv.SVG("tornado"))
}

// RiskMatrix menggambar peta panas probabilitas-dampak 5x5.
func RiskMatrix(matrix [5][5][]string, lang, caption string) template.HTML {
	cell := 92.0
	cv := NewCanvas(cell*5+150, cell*5+86, Padding{Top: 42, Right: 20, Bottom: 44, Left: 150})
	cv.Describe(caption, tr(lang,
		"Peta panas lima kali lima; sumbu mendatar adalah tingkat peluang, sumbu tegak adalah tingkat dampak, dan setiap sel memuat kode risiko yang jatuh di sana.",
		"A five-by-five heat map; the horizontal axis is probability level, the vertical axis is impact level, and each cell lists the risk codes falling there."))

	probLabels := []string{
		tr(lang, "Sangat rendah", "Very low"), tr(lang, "Rendah", "Low"),
		tr(lang, "Sedang", "Medium"), tr(lang, "Tinggi", "High"), tr(lang, "Sangat tinggi", "Very high"),
	}
	impLabels := []string{
		tr(lang, "Dapat diabaikan", "Negligible"), tr(lang, "Kecil", "Minor"),
		tr(lang, "Sedang", "Moderate"), tr(lang, "Besar", "Major"), tr(lang, "Sangat besar", "Severe"),
	}

	for imp := 4; imp >= 0; imp-- {
		row := 4 - imp
		y := cv.Pad.Top + float64(row)*cell
		cv.Text(cv.Pad.Left-12, y+cell/2, impLabels[imp], "matrix-axis", "end")
		cv.Text(cv.Pad.Left-12, y+cell/2+15, fmt.Sprintf("%s %d", tr(lang, "dampak", "impact"), imp+1), "matrix-axis dim", "end")
		for prob := 0; prob < 5; prob++ {
			x := cv.Pad.Left + float64(prob)*cell
			score := (imp + 1) * (prob + 1)
			sev := string(risk.SeverityOf(score))
			ids := matrix[imp][prob]
			cv.Group("matrix-cell sev-"+sev).
				Rect(x+2, y+2, cell-4, cell-4, "matrix-box").
				Title(fmt.Sprintf("%s %d x %s %d = %d (%s)",
					tr(lang, "dampak", "impact"), imp+1, tr(lang, "peluang", "probability"), prob+1, score, sev)).
				EndGroup()
			cv.Text(x+cell-8, y+16, fmt.Sprintf("%d", score), "matrix-score", "end")
			for i, id := range ids {
				if i >= 4 {
					cv.Text(x+cell/2, y+30+float64(i)*17, "+"+fmt.Sprint(len(ids)-4), "matrix-id", "middle")
					break
				}
				cv.Text(x+cell/2, y+30+float64(i)*17, id, "matrix-id", "middle")
			}
		}
	}
	for prob := 0; prob < 5; prob++ {
		x := cv.Pad.Left + float64(prob)*cell + cell/2
		cv.Text(x, cv.Pad.Top+5*cell+20, probLabels[prob], "matrix-axis", "middle")
		cv.Text(x, cv.Pad.Top+5*cell+35, fmt.Sprintf("%s %d", tr(lang, "peluang", "prob"), prob+1), "matrix-axis dim", "middle")
	}
	cv.Text(cv.Pad.Left, 22, caption, "chart-title", "start")
	return template.HTML(cv.SVG("risk-matrix"))
}

// ControlChartSVG menggambar peta kendali X-bar beserta batas kendali, batas
// spesifikasi, dan penanda pelanggaran aturan.
func ControlChartSVG(cc quality.ControlChart, samples []model.ResponseSample, lang string) template.HTML {
	cv := NewCanvas(880, 360, Padding{Top: 26, Right: 96, Bottom: 46, Left: 64})
	n := float64(len(cc.Values))
	maxY := cc.UCL
	for _, v := range cc.Values {
		if v > maxY {
			maxY = v
		}
	}
	if cc.HasSpec && cc.Spec > maxY {
		maxY = cc.Spec
	}
	maxY *= 1.1
	cv.SetDomain(1, n, 0, maxY).Describe(
		tr(lang, "Peta kendali waktu respons", "Response time control chart"),
		tr(lang,
			"Rata-rata subgrup waktu respons per minggu dengan garis tengah, batas kendali atas dan bawah, serta batas spesifikasi tiga detik dari Project Charter.",
			"Weekly subgroup mean response time with the centre line, upper and lower control limits, and the three-second specification limit from the Project Charter."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, maxY, 6) {
		yTicks = append(yTicks, Tick{v, Num(v, 1, lang) + "s"})
	}
	cv.AxisY(yTicks, true)
	var xTicks []Tick
	for i := 1; i <= int(n); i++ {
		xTicks = append(xTicks, Tick{float64(i), fmt.Sprintf("%d", i)})
	}
	cv.AxisX(xTicks, false)

	bands := []struct {
		v     float64
		label string
		class string
	}{
		{cc.UCL, "UCL " + Num(cc.UCL, 2, lang), "limit ucl"},
		{cc.CenterLine, "CL " + Num(cc.CenterLine, 2, lang), "limit cl"},
		{cc.LCL, "LCL " + Num(cc.LCL, 2, lang), "limit lcl"},
	}
	if cc.HasSpec {
		bands = append(bands, struct {
			v     float64
			label string
			class string
		}{cc.Spec, tr(lang, "Batas spesifikasi ", "Spec limit ") + Num(cc.Spec, 1, lang) + "s", "limit spec"})
	}
	for _, b := range bands {
		y := cv.Y(b.v)
		cv.Line(cv.Pad.Left, y, cv.Pad.Left+cv.plotW(), y, b.class)
		cv.Text(cv.Pad.Left+cv.plotW()+6, y+4, b.label, "limit-label "+b.class, "start")
	}

	violated := map[int]bool{}
	for _, v := range cc.Violations {
		violated[v.Index] = true
	}
	var xs, ys []float64
	for i, v := range cc.Values {
		xs = append(xs, float64(i+1))
		ys = append(ys, v)
	}
	cv.PolyLine(xs, ys, "series measure")
	for i, v := range cc.Values {
		cls := "point"
		if violated[i] {
			cls += " violation"
		}
		note := ""
		if i < len(samples) && samples[i].Note.ID != "" {
			note = " - " + samples[i].Note.Get(lang)
		}
		cv.Group("pt").
			Circle(cv.X(float64(i+1)), cv.Y(v), 4.5, cls).
			Title(fmt.Sprintf("%s %d: %ss%s", tr(lang, "minggu", "week"), i+1, Num(v, 2, lang), note)).
			EndGroup()
	}
	return template.HTML(cv.SVG("control-chart"))
}

// ParetoSVG menggambar diagram Pareto dengan kurva kumulatif dan garis 80%.
func ParetoSVG(p quality.Pareto, lang string) template.HTML {
	cv := NewCanvas(880, 400, Padding{Top: 26, Right: 56, Bottom: 118, Left: 64})
	n := float64(len(p.Items))
	maxV := 0.0
	for _, it := range p.Items {
		if it.Value > maxV {
			maxV = it.Value
		}
	}
	cv.SetDomain(0, n, 0, maxV*1.15).Describe(
		tr(lang, "Diagram Pareto cacat berbobot", "Weighted defect Pareto chart"),
		tr(lang,
			"Batang menunjukkan cacat berbobot keparahan per modul dari yang terbesar, garis menunjukkan porsi kumulatif, dan garis putus-putus menandai ambang delapan puluh persen.",
			"Bars show severity-weighted defects per module in descending order, the line shows cumulative share, and the dashed line marks the eighty percent threshold."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, maxV, 5) {
		yTicks = append(yTicks, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisY(yTicks, true)

	barW := cv.plotW() / n * 0.72
	for i, it := range p.Items {
		x := cv.X(float64(i) + 0.5)
		y := cv.Y(it.Value)
		cls := "pareto-bar"
		if it.InVitalFew {
			cls += " vital-few"
		}
		cv.Group("pareto-col").
			Rect(x-barW/2, y, barW, cv.Y(0)-y, cls).
			Title(fmt.Sprintf("%s: %s (%s, %s %s)", it.Label.Get(lang),
				Num(it.Value, 0, lang), Pct(it.Share, 1, lang),
				tr(lang, "kumulatif", "cumulative"), Pct(it.Cumulative, 1, lang))).
			EndGroup()
		cv.Raw(fmt.Sprintf(`<text x="%s" y="%s" class="axis-label rotated" text-anchor="end" transform="rotate(-38 %s %s)">%s</text>`,
			f(x), f(cv.Y(0)+14), f(x), f(cv.Y(0)+14), template.HTMLEscapeString(truncate(it.Label.Get(lang), 34))))
	}

	var cx, cy []float64
	for i, it := range p.Items {
		cx = append(cx, float64(i)+0.5)
		cy = append(cy, it.Cumulative*maxV*1.15)
	}
	cv.PolyLine(cx, cy, "series cumulative")
	y80 := cv.Y(0.8 * maxV * 1.15)
	cv.Line(cv.Pad.Left, y80, cv.Pad.Left+cv.plotW(), y80, "ref-line eighty")
	cv.Text(cv.Pad.Left+cv.plotW(), y80-6, "80%", "axis-label", "end")
	return template.HTML(cv.SVG("pareto"))
}

// BudgetWaterfall menggambar struktur anggaran berlapis: biaya aktivitas,
// cadangan kontinjensi, cadangan manajemen, sampai pagu total.
func BudgetWaterfall(bac, contingency, management, total float64, lang string) template.HTML {
	cv := NewCanvas(880, 300, Padding{Top: 40, Right: 24, Bottom: 72, Left: 76})
	cv.SetDomain(0, 4, 0, total*1.12).Describe(
		tr(lang, "Struktur anggaran berlapis", "Layered budget structure"),
		tr(lang,
			"Biaya aktivitas ditambah cadangan kontinjensi menghasilkan cost baseline; ditambah cadangan manajemen menghasilkan pagu total yang disetujui.",
			"Activity cost plus contingency reserve gives the cost baseline; plus management reserve gives the total authorised budget."))

	steps := []struct {
		label string
		from  float64
		to    float64
		class string
	}{
		{tr(lang, "Biaya aktivitas (BAC)", "Activity cost (BAC)"), 0, bac, "wf-bac"},
		{tr(lang, "Cadangan kontinjensi", "Contingency reserve"), bac, bac + contingency, "wf-cont"},
		{tr(lang, "Cadangan manajemen", "Management reserve"), bac + contingency, total, "wf-mgmt"},
		{tr(lang, "Pagu disetujui", "Authorised budget"), 0, total, "wf-total"},
	}
	barW := cv.plotW() / 4 * 0.58
	for i, s := range steps {
		x := cv.X(float64(i) + 0.5)
		y1, y2 := cv.Y(s.from), cv.Y(s.to)
		cv.Group("wf-step").
			Rect(x-barW/2, y2, barW, y1-y2, "wf-bar "+s.class).
			Title(fmt.Sprintf("%s: %s", s.label, Rp(s.to-s.from, lang))).
			EndGroup()
		cv.Text(x, y2-8, RpShort(s.to-s.from, lang), "bar-value", "middle")
		cv.Text(x, cv.Y(0)+18, truncate(s.label, 26), "axis-label", "middle")
		if i < 3 {
			nx := cv.X(float64(i) + 1.5)
			cv.Line(x+barW/2, y2, nx-barW/2, y2, "wf-connector")
		}
	}
	var yTicks []Tick
	for _, v := range NiceTicks(0, total, 5) {
		yTicks = append(yTicks, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yTicks, true)
	cv.Text(cv.Pad.Left, 24, tr(lang, "Dari estimasi bottom-up ke pagu Project Charter", "From bottom-up estimate to the charter cap"), "chart-title", "start")
	return template.HTML(cv.SVG("waterfall"))
}

// ResourceHistogram menggambar beban harian satu peran terhadap kapasitasnya.
func ResourceHistogram(p resource.Profile, lang string, cal *workcal.Calendar) template.HTML {
	rowH := 46.0
	cv := NewCanvas(880, 46+rowH*float64(len(p.Roles))+36, Padding{Top: 34, Right: 72, Bottom: 36, Left: 66})
	cv.SetDomain(0, float64(p.Horizon), 0, 1).Describe(
		tr(lang, "Histogram pembebanan sumber daya", "Resource loading histogram"),
		tr(lang,
			"Satu baris per peran sepanjang 85 hari kerja; batang yang melewati garis kapasitas menandakan satu orang dijadwalkan mengerjakan lebih dari satu pekerjaan penuh waktu pada hari yang sama.",
			"One row per role across 85 working days; bars crossing the capacity line mark a person scheduled for more than one full-time job on the same day."))

	dayW := cv.plotW() / float64(p.Horizon)
	for i, rl := range p.Roles {
		top := cv.Pad.Top + float64(i)*rowH
		lane := rowH - 12
		maxLoad := math.Max(rl.PeakLoad, rl.Capacity)
		cv.Text(cv.Pad.Left-10, top+lane/2+4, string(rl.Role), "row-label", "end")
		cv.Rect(cv.Pad.Left, top, cv.plotW(), lane, "lane-bg")
		// Kapasitas digambar per hari sebagai garis bertangga, bukan satu garis
		// datar: pada jadwal levelling kapasitas turun selama periode ujian,
		// dan garis datar akan menyembunyikan justru hal yang ingin ditunjukkan.
		var capPath strings.Builder
		for i, d := range rl.Days {
			x1 := cv.Pad.Left + float64(d.Day)*dayW
			x2 := x1 + dayW
			y := top + lane - (d.Capacity/maxLoad)*lane
			if i == 0 {
				capPath.WriteString(fmt.Sprintf("M%s %s ", f(x1), f(y)))
			} else {
				capPath.WriteString(fmt.Sprintf("L%s %s ", f(x1), f(y)))
			}
			capPath.WriteString(fmt.Sprintf("L%s %s ", f(x2), f(y)))
		}
		if capPath.Len() > 0 {
			cv.Path(strings.TrimSpace(capPath.String()), "capacity-line")
		}
		for _, d := range rl.Days {
			if d.Load <= 0 {
				continue
			}
			h := (d.Load / maxLoad) * lane
			cls := "load-bar"
			if d.Over {
				cls += " over"
			}
			cv.Group("load").
				Rect(cv.Pad.Left+float64(d.Day)*dayW, top+lane-h, math.Max(dayW-0.4, 0.8), h, cls).
				Title(fmt.Sprintf("%s - %s: %s %s / %s (%v)", string(rl.Role),
					workcal.FormatDateShort(cal.ISOAt(d.Day), lang),
					Num(d.Load, 1, lang), tr(lang, "hari-orang", "person-days"),
					Num(d.Capacity, 1, lang), d.Activities)).
				EndGroup()
		}
		cv.Text(cv.Pad.Left+cv.plotW()+8, top+lane/2+4,
			fmt.Sprintf("%s - %s%%", Num(rl.TotalDays, 1, lang), Num(rl.Utilisation*100, 0, lang)),
			"row-value", "start")
	}
	var xTicks []Tick
	for d := 0; d <= p.Horizon; d += 10 {
		xTicks = append(xTicks, Tick{float64(d), workcal.FormatDateShort(cal.ISOAt(d), lang)})
	}
	cv.AxisX(xTicks, false)
	cv.Text(cv.Pad.Left, 22, tr(lang, "Beban hari-orang per peran, garis putus-putus adalah kapasitas", "Person-day load per role; the dashed line is capacity"), "chart-title", "start")
	return template.HTML(cv.SVG("resource-histogram"))
}

// PowerInterestGrid menggambar grid kuasa-kepentingan pemangku kepentingan.
func PowerInterestGrid(sh []model.Stakeholder, lang string) template.HTML {
	cv := NewCanvas(700, 560, Padding{Top: 40, Right: 30, Bottom: 60, Left: 74})
	cv.SetDomain(0.5, 5.5, 0.5, 5.5).Describe(
		tr(lang, "Grid kuasa-kepentingan", "Power-interest grid"),
		tr(lang,
			"Sepuluh pemangku kepentingan ditempatkan menurut tingkat kuasa dan kepentingan; kuadran menentukan strategi pengelolaan.",
			"Ten stakeholders placed by power and interest; the quadrant sets the engagement strategy."))

	midX, midY := cv.X(3), cv.Y(3)
	quads := []struct {
		x, y   float64
		id, en string
	}{
		{cv.Pad.Left + 10, cv.Pad.Top + 18, "Jaga kepuasan", "Keep satisfied"},
		{cv.Pad.Left + cv.plotW() - 10, cv.Pad.Top + 18, "Kelola erat", "Manage closely"},
		{cv.Pad.Left + 10, cv.Pad.Top + cv.plotH() - 10, "Pantau", "Monitor"},
		{cv.Pad.Left + cv.plotW() - 10, cv.Pad.Top + cv.plotH() - 10, "Beri informasi", "Keep informed"},
	}
	cv.Rect(cv.Pad.Left, cv.Pad.Top, cv.plotW()/2, cv.plotH()/2, "quad q-satisfy")
	cv.Rect(midX, cv.Pad.Top, cv.plotW()/2, cv.plotH()/2, "quad q-manage")
	cv.Rect(cv.Pad.Left, midY, cv.plotW()/2, cv.plotH()/2, "quad q-monitor")
	cv.Rect(midX, midY, cv.plotW()/2, cv.plotH()/2, "quad q-inform")
	for i, q := range quads {
		anchor := "start"
		if i%2 == 1 {
			anchor = "end"
		}
		cv.Text(q.x, q.y, tr(lang, q.id, q.en), "quad-label", anchor)
	}
	cv.Line(midX, cv.Pad.Top, midX, cv.Pad.Top+cv.plotH(), "grid-line strong")
	cv.Line(cv.Pad.Left, midY, cv.Pad.Left+cv.plotW(), midY, "grid-line strong")

	var ticks []Tick
	for v := 1.0; v <= 5; v++ {
		ticks = append(ticks, Tick{v, Num(v, 0, lang)})
	}
	cv.AxisX(ticks, false)
	cv.AxisY(ticks, false)
	cv.Text(cv.Pad.Left+cv.plotW()/2, cv.H-14, tr(lang, "Kepentingan", "Interest"), "axis-title", "middle")
	cv.Raw(fmt.Sprintf(`<text x="18" y="%s" class="axis-title" text-anchor="middle" transform="rotate(-90 18 %s)">%s</text>`,
		f(cv.Pad.Top+cv.plotH()/2), f(cv.Pad.Top+cv.plotH()/2), tr(lang, "Kuasa", "Power")))

	// Sebar titik yang berimpit agar tidak saling menutupi.
	seen := map[[2]int]int{}
	for _, s := range sh {
		key := [2]int{s.Interest, s.Power}
		k := seen[key]
		seen[key]++
		off := float64(k) * 0.17
		x := cv.X(float64(s.Interest) + off)
		y := cv.Y(float64(s.Power) - off)
		cls := "sh-dot"
		if s.Internal {
			cls += " internal"
		}
		cv.Group("sh").
			Circle(x, y, 15, cls).
			Title(fmt.Sprintf("%s - %s", s.Name.Get(lang), s.Strategy.Get(lang))).
			EndGroup()
		cv.Text(x, y+4, s.ID, "sh-label", "middle")
	}
	return template.HTML(cv.SVG("power-interest"))
}

// MetricBars menggambar capaian metrik mutu terhadap targetnya.
func MetricBars(metrics []quality.MetricStatus, lang string) template.HTML {
	rowH := 38.0
	cv := NewCanvas(880, 28+rowH*float64(len(metrics))+18, Padding{Top: 22, Right: 120, Bottom: 18, Left: 300})
	cv.SetDomain(0, 1.25, 0, 1).Describe(
		tr(lang, "Capaian metrik mutu terhadap target", "Quality metric attainment against target"),
		tr(lang,
			"Setiap batang menunjukkan rasio capaian terhadap target; garis tegak adalah posisi target seratus persen.",
			"Each bar shows attainment relative to target; the vertical line marks the hundred percent target position."))

	targetX := cv.X(1)
	for i, m := range metrics {
		y := cv.Pad.Top + float64(i)*rowH
		ratio := math.Min(m.Ratio, 1.25)
		w := (ratio / 1.25) * cv.plotW()
		cls := "metric-bar"
		if m.Met {
			cls += " met"
		} else if m.Ratio < 0.6 {
			cls += " far"
		}
		cv.Group("metric-row").
			Rect(cv.Pad.Left, y+6, w, rowH-16, cls).
			Title(fmt.Sprintf("%s: %s %s / %s %s", m.Name.Get(lang),
				Num(m.Actual, 2, lang), m.Unit, Num(m.Target, 2, lang), m.Unit)).
			EndGroup()
		cv.Text(cv.Pad.Left-10, y+rowH/2+1, truncate(m.Name.Get(lang), 42), "row-label", "end")
		cv.Text(cv.Pad.Left+w+8, y+rowH/2+1,
			fmt.Sprintf("%s / %s %s", Num(m.Actual, 2, lang), Num(m.Target, 2, lang), m.Unit),
			"row-value", "start")
	}
	cv.Line(targetX, cv.Pad.Top, targetX, cv.H-cv.Pad.Bottom, "ref-line target")
	return template.HTML(cv.SVG("metric-bars"))
}

// COQBars menggambar biaya kualitas: kesesuaian dibanding ketidaksesuaian.
func COQBars(s quality.COQSummary, lang string) template.HTML {
	cv := NewCanvas(880, 250, Padding{Top: 40, Right: 24, Bottom: 60, Left: 76})
	cats := []struct {
		label string
		value float64
		class string
	}{
		{tr(lang, "Pencegahan", "Prevention"), s.Prevention, "coq-prevention"},
		{tr(lang, "Penilaian", "Appraisal"), s.Appraisal, "coq-appraisal"},
		{tr(lang, "Kegagalan internal", "Internal failure"), s.InternalFailure, "coq-internal"},
		{tr(lang, "Kegagalan eksternal", "External failure"), s.ExternalFailure, "coq-external"},
	}
	maxV := 0.0
	for _, c := range cats {
		if c.value > maxV {
			maxV = c.value
		}
	}
	cv.SetDomain(0, 4, 0, maxV*1.2).Describe(
		tr(lang, "Biaya kualitas menurut kategori", "Cost of quality by category"),
		tr(lang,
			"Dua kategori pertama adalah biaya kesesuaian yang dikeluarkan untuk mencegah cacat; dua terakhir adalah biaya ketidaksesuaian yang muncul karena cacat terjadi.",
			"The first two are conformance costs spent to prevent defects; the last two are non-conformance costs incurred because defects happened."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, maxV, 5) {
		yTicks = append(yTicks, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yTicks, true)
	barW := cv.plotW() / 4 * 0.5
	for i, c := range cats {
		x := cv.X(float64(i) + 0.5)
		y := cv.Y(c.value)
		cv.Group("coq-col").
			Rect(x-barW/2, y, barW, cv.Y(0)-y, "coq-bar "+c.class).
			Title(fmt.Sprintf("%s: %s", c.label, Rp(c.value, lang))).
			EndGroup()
		cv.Text(x, y-8, RpShort(c.value, lang), "bar-value", "middle")
		cv.Text(x, cv.Y(0)+18, c.label, "axis-label", "middle")
	}
	// Pemisah kesesuaian dan ketidaksesuaian.
	sep := cv.X(2)
	cv.Line(sep, cv.Pad.Top, sep, cv.Y(0), "ref-line divider")
	cv.Text(cv.X(1), 24, tr(lang, "Biaya kesesuaian", "Conformance"), "chart-title", "middle")
	cv.Text(cv.X(3), 24, tr(lang, "Biaya ketidaksesuaian", "Non-conformance"), "chart-title", "middle")
	return template.HTML(cv.SVG("coq"))
}

// ScenarioBars membandingkan biaya harapan strategi transisi pada studi kasus.
func ScenarioBars(labels []string, extra, expected []float64, lang string) template.HTML {
	n := float64(len(labels))
	cv := NewCanvas(880, 320, Padding{Top: 42, Right: 24, Bottom: 76, Left: 96})
	maxV := 0.0
	for i := range extra {
		if t := extra[i] + expected[i]; t > maxV {
			maxV = t
		}
	}
	cv.SetDomain(0, n, 0, maxV*1.15).Describe(
		tr(lang, "Biaya harapan strategi transisi", "Expected cost of transition strategies"),
		tr(lang,
			"Batang bertumpuk: bagian bawah adalah tambahan biaya strategi, bagian atas adalah kerugian harapan bila cutover gagal.",
			"Stacked bars: the lower segment is the strategy's extra cost, the upper segment is the expected loss if the cutover fails."))

	var yTicks []Tick
	for _, v := range NiceTicks(0, maxV, 5) {
		yTicks = append(yTicks, Tick{v, RpShort(v, lang)})
	}
	cv.AxisY(yTicks, true)
	barW := cv.plotW() / n * 0.5
	for i := range labels {
		x := cv.X(float64(i) + 0.5)
		y0 := cv.Y(0)
		y1 := cv.Y(extra[i])
		y2 := cv.Y(extra[i] + expected[i])
		cv.Group("sc-col").
			Rect(x-barW/2, y1, barW, y0-y1, "sc-bar extra").
			Title(fmt.Sprintf("%s - %s: %s", labels[i], tr(lang, "tambahan biaya", "extra cost"), RpShort(extra[i], lang))).
			EndGroup()
		cv.Group("sc-col").
			Rect(x-barW/2, y2, barW, y1-y2, "sc-bar expected").
			Title(fmt.Sprintf("%s - %s: %s", labels[i], tr(lang, "kerugian harapan", "expected loss"), RpShort(expected[i], lang))).
			EndGroup()
		cv.Text(x, y2-8, RpShort(extra[i]+expected[i], lang), "bar-value", "middle")
		cv.Text(x, y0+18, truncate(labels[i], 24), "axis-label", "middle")
	}
	return template.HTML(cv.SVG("scenario-bars"))
}

// Sparkline menggambar garis mini tanpa sumbu, untuk disisipkan di dalam tabel.
func Sparkline(values []float64, w, h float64, class string) template.HTML {
	if len(values) < 2 {
		return ""
	}
	cv := NewCanvas(w, h, Padding{Top: 2, Right: 2, Bottom: 2, Left: 2})
	min, max := values[0], values[0]
	for _, v := range values {
		min = math.Min(min, v)
		max = math.Max(max, v)
	}
	cv.SetDomain(0, float64(len(values)-1), min, max)
	xs := make([]float64, len(values))
	for i := range values {
		xs[i] = float64(i)
	}
	cv.PolyLine(xs, values, "spark-line")
	return template.HTML(cv.SVG("sparkline " + class))
}

// tr memilih salah satu dari dua string menurut bahasa. Dipakai untuk teks
// pendek khas grafik yang tidak layak masuk kamus i18n.
func tr(lang, id, en string) string {
	if lang == "en" {
		return en
	}
	return id
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// SortedRoles mengembalikan peran terurut - dipakai templat agar urutan kolom
// tabel selalu sama di setiap muat.
func SortedRoles(m map[model.Role]float64) []model.Role {
	out := make([]model.Role, 0, len(m))
	for r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
