package render

import (
	"fmt"
	"html/template"
	"math"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Gantt menggambar diagram Gantt lengkap: batang rencana, batang realisasi,
// float sebagai batang bayangan, milestone sebagai belah ketupat, panah
// ketergantungan untuk jalur kritis, dan garis tanggal data.
//
// Float digambar eksplisit karena di situlah letak ruang gerak proyek. Gantt
// yang hanya menampilkan batang rencana menyembunyikan informasi terpenting:
// aktivitas mana yang boleh molor tanpa akibat, dan mana yang tidak.
func Gantt(acts []model.Activity, plan schedule.Result, cal *workcal.Calendar, statusDay float64, lang string) template.HTML {
	rowH := 26.0
	labelW := 330.0
	headerH := 54.0
	n := float64(plan.Duration)

	// Sisipkan baris judul fase di antara aktivitas.
	type row struct {
		phase    bool
		code     string
		label    string
		activity model.Activity
	}
	var rows []row
	lastPhase := ""
	for _, a := range acts {
		ph := model.PhaseOf(a.WBS)
		if ph.Code != lastPhase {
			lastPhase = ph.Code
			rows = append(rows, row{phase: true, code: ph.Code, label: ph.Code + " " + ph.Name.Get(lang)})
		}
		rows = append(rows, row{code: a.ID, label: a.ID + " - " + truncate(a.Name.Get(lang), 40), activity: a})
	}

	h := headerH + rowH*float64(len(rows)) + 26
	cv := NewCanvas(1080, h, Padding{Top: headerH, Right: 26, Bottom: 24, Left: labelW})
	cv.SetDomain(0, n, 0, 1).Describe(
		tr(lang, "Diagram Gantt dengan jalur kritis", "Gantt chart with critical path"),
		tr(lang,
			"Batang rencana per aktivitas sepanjang 85 hari kerja; batang jalur kritis ditandai berbeda, float digambar sebagai batang bayangan, dan batang realisasi ditumpuk di bawahnya.",
			"Planned bars per activity across 85 working days; critical-path bars are marked differently, float is drawn as a shadow bar, and actual bars sit beneath."))

	dayW := cv.plotW() / n

	// Kepala sumbu: bulan dan penanda awal minggu.
	lastMonth := -1
	for d := 0; d <= plan.Duration; d++ {
		t := cal.Date(d)
		if int(t.Month()) != lastMonth {
			lastMonth = int(t.Month())
			x := cv.Pad.Left + float64(d)*dayW
			cv.Line(x, headerH-20, x, cv.H-cv.Pad.Bottom, "grid-line month")
			cv.Text(x+4, headerH-26, fmt.Sprintf("%s %d", workcal.MonthShort(t.Month(), lang), t.Year()), "axis-label month", "start")
		}
		if d%5 == 0 {
			x := cv.Pad.Left + float64(d)*dayW
			cv.Line(x, headerH-8, x, cv.H-cv.Pad.Bottom, "grid-line week")
			cv.Text(x, headerH-12, fmt.Sprintf("M%d", d/5+1), "axis-label week", "middle")
		}
	}

	for i, r := range rows {
		y := headerH + float64(i)*rowH
		if r.phase {
			cv.Rect(0, y, cv.W, rowH, "gantt-phase-bg")
			cv.Text(10, y+rowH/2+4, r.label, "gantt-phase-label", "start")
			continue
		}
		a := r.activity
		task := plan.Task(a.ID)
		cls := "gantt-label"
		if task.Critical {
			cls += " critical"
		}
		cv.Text(labelW-12, y+rowH/2+4, r.label, cls, "end")

		x1 := cv.Pad.Left + float64(task.StartX)*dayW
		x2 := cv.Pad.Left + float64(task.FinishX)*dayW

		if a.Milestone {
			mx := x1
			my := y + rowH/2
			s := 7.0
			mcls := "gantt-milestone"
			if task.Critical {
				mcls += " critical"
			}
			cv.Group("ms").
				Path(fmt.Sprintf("M%s %s L%s %s L%s %s L%s %s Z",
					f(mx), f(my-s), f(mx+s), f(my), f(mx), f(my+s), f(mx-s), f(my)), mcls).
				Title(fmt.Sprintf("%s - %s", a.Name.Get(lang), workcal.FormatDate(cal.ISOAt(task.StartX), lang))).
				EndGroup()
			continue
		}

		// Batang float lebih dulu supaya berada di belakang batang rencana.
		if task.TotalFloat > 0 {
			fx := x2
			fw := float64(task.TotalFloat) * dayW
			cv.Group("fl").
				Rect(fx, y+8, fw, rowH-18, "gantt-float").
				Title(fmt.Sprintf("%s: %d %s", tr(lang, "total float", "total float"), task.TotalFloat, tr(lang, "hari kerja", "working days"))).
				EndGroup()
		}

		barCls := "gantt-bar"
		if task.Critical {
			barCls += " critical"
		}
		cv.Group("gb").
			Rect(x1, y+4, math.Max(x2-x1, 2), rowH-14, barCls).
			Title(fmt.Sprintf("%s\n%s: %s - %s (%d %s)\n%s: %d, %s: %d",
				a.Name.Get(lang), tr(lang, "rencana", "plan"),
				workcal.FormatDate(cal.ISOAt(task.StartX), lang),
				workcal.FormatDate(cal.ISOAt(task.FinishX-1), lang),
				task.Duration, tr(lang, "hari kerja", "working days"),
				tr(lang, "total float", "total float"), task.TotalFloat,
				tr(lang, "free float", "free float"), task.FreeFloat)).
			EndGroup()

		if a.Actual.Started && a.Actual.Duration > 0 {
			ax1 := cv.Pad.Left + float64(a.Actual.Start)*dayW
			ax2 := cv.Pad.Left + float64(a.Actual.Start+a.Actual.Duration)*dayW
			acls := "gantt-actual"
			if a.Actual.Duration > task.Duration {
				acls += " overrun"
			}
			cv.Group("ga").
				Rect(ax1, y+rowH-9, math.Max(ax2-ax1, 2), 5, acls).
				Title(fmt.Sprintf("%s: %s - %s (%d %s)", tr(lang, "realisasi", "actual"),
					workcal.FormatDate(cal.ISOAt(a.Actual.Start), lang),
					workcal.FormatDate(cal.ISOAt(a.Actual.Start+a.Actual.Duration-1), lang),
					a.Actual.Duration, tr(lang, "hari kerja", "working days"))).
				EndGroup()
		}
	}

	// Panah ketergantungan, dibatasi pada jalur kritis supaya tidak jadi
	// sarang laba-laba yang justru menyembunyikan strukturnya.
	rowOf := map[string]int{}
	for i, r := range rows {
		if !r.phase {
			rowOf[r.code] = i
		}
	}
	critical := map[string]bool{}
	for _, id := range plan.CriticalPath {
		critical[id] = true
	}
	for i := 1; i < len(plan.CriticalPath); i++ {
		from, to := plan.CriticalPath[i-1], plan.CriticalPath[i]
		fr, ok1 := rowOf[from]
		tr2, ok2 := rowOf[to]
		if !ok1 || !ok2 {
			continue
		}
		ft := plan.Task(from)
		tt := plan.Task(to)
		x1 := cv.Pad.Left + float64(ft.FinishX)*dayW
		y1 := headerH + float64(fr)*rowH + rowH/2
		x2 := cv.Pad.Left + float64(tt.StartX)*dayW
		y2 := headerH + float64(tr2)*rowH + rowH/2
		mid := x1 + math.Max(6, (x2-x1)/2)
		cv.Path(fmt.Sprintf("M%s %s H%s V%s H%s", f(x1), f(y1), f(mid), f(y2), f(x2)), "gantt-link")
		cv.Path(fmt.Sprintf("M%s %s l-5 -3 l0 6 Z", f(x2), f(y2)), "gantt-arrow")
	}

	sx := cv.Pad.Left + statusDay*dayW
	cv.Line(sx, headerH-20, sx, cv.H-cv.Pad.Bottom, "status-line")
	cv.Text(sx, headerH-26, tr(lang, "tanggal data", "data date"), "axis-label status", "middle")
	return template.HTML(cv.SVG("gantt"))
}

// Network menggambar diagram jaringan Activity-on-Node.
//
// Tata letaknya berlapis menurut early start, sehingga sumbu mendatar tetap
// bermakna waktu - berbeda dari tata letak graf umum yang menempatkan simpul
// sekadar agar rapi. Simpul memuat enam angka standar CPM: ES, durasi, EF di
// baris atas, dan LS, float, LF di baris bawah.
func Network(acts []model.Activity, plan schedule.Result, lang string) template.HTML {
	nodeW, nodeH := 116.0, 62.0
	gapX, gapY := 44.0, 22.0

	// Kelompokkan simpul menurut kolom = early start.
	cols := map[int][]string{}
	var colKeys []int
	for _, a := range acts {
		t := plan.Task(a.ID)
		if _, ok := cols[t.StartX]; !ok {
			colKeys = append(colKeys, t.StartX)
		}
		cols[t.StartX] = append(cols[t.StartX], a.ID)
	}
	sortInts(colKeys)

	pos := map[string][2]float64{}
	maxRows := 0
	for ci, key := range colKeys {
		ids := cols[key]
		if len(ids) > maxRows {
			maxRows = len(ids)
		}
		for ri, id := range ids {
			pos[id] = [2]float64{
				40 + float64(ci)*(nodeW+gapX),
				60 + float64(ri)*(nodeH+gapY),
			}
		}
	}
	w := 40 + float64(len(colKeys))*(nodeW+gapX) + 20
	h := 60 + float64(maxRows)*(nodeH+gapY) + 30

	cv := NewCanvas(w, h, Padding{})
	cv.Describe(
		tr(lang, "Diagram jaringan Activity-on-Node", "Activity-on-Node network diagram"),
		tr(lang,
			"Setiap kotak adalah satu aktivitas dengan early start, durasi, early finish di baris atas, serta late start, total float, late finish di baris bawah. Jalur kritis ditandai dengan garis tebal.",
			"Each box is one activity showing early start, duration, early finish on the top row, and late start, total float, late finish on the bottom. The critical path is drawn with heavy lines."))

	critical := map[string]bool{}
	for _, id := range plan.CriticalPath {
		critical[id] = true
	}

	// Gambar sisi lebih dulu agar berada di belakang simpul.
	for _, a := range acts {
		t := plan.Task(a.ID)
		p := pos[a.ID]
		for _, pred := range t.Predecessors {
			q, ok := pos[pred.ID]
			if !ok {
				continue
			}
			x1 := q[0] + nodeW
			y1 := q[1] + nodeH/2
			x2 := p[0]
			y2 := p[1] + nodeH/2
			cls := "net-edge"
			if critical[a.ID] && critical[pred.ID] {
				cls += " critical"
			}
			mid := x1 + math.Max(10, (x2-x1)/2)
			cv.Path(fmt.Sprintf("M%s %s C%s %s %s %s %s %s",
				f(x1), f(y1), f(mid), f(y1), f(mid), f(y2), f(x2), f(y2)), cls)
			cv.Path(fmt.Sprintf("M%s %s l-6 -3.5 l0 7 Z", f(x2), f(y2)), "net-arrow")
		}
	}

	byID := model.ActivityByID()
	for id, p := range pos {
		t := plan.Task(id)
		a := byID[id]
		cls := "net-node"
		if critical[id] {
			cls += " critical"
		}
		if a.Milestone {
			cls += " milestone"
		}
		cv.Group("nn").
			Rect(p[0], p[1], nodeW, nodeH, cls).
			Title(fmt.Sprintf("%s - %s\nES %d - EF %d | LS %d - LF %d | TF %d | FF %d",
				id, a.Name.Get(lang), t.ES, t.EF, t.LS, t.LF, t.TotalFloat, t.FreeFloat)).
			EndGroup()
		cv.Line(p[0], p[1]+18, p[0]+nodeW, p[1]+18, "net-divider")
		cv.Line(p[0], p[1]+44, p[0]+nodeW, p[1]+44, "net-divider")
		cv.Line(p[0]+nodeW/3, p[1], p[0]+nodeW/3, p[1]+18, "net-divider")
		cv.Line(p[0]+nodeW*2/3, p[1], p[0]+nodeW*2/3, p[1]+18, "net-divider")
		cv.Line(p[0]+nodeW/3, p[1]+44, p[0]+nodeW/3, p[1]+nodeH, "net-divider")
		cv.Line(p[0]+nodeW*2/3, p[1]+44, p[0]+nodeW*2/3, p[1]+nodeH, "net-divider")

		cv.Text(p[0]+nodeW/6, p[1]+13, fmt.Sprintf("%d", t.ES), "net-cell", "middle")
		cv.Text(p[0]+nodeW/2, p[1]+13, fmt.Sprintf("%d", t.Duration), "net-cell dur", "middle")
		cv.Text(p[0]+nodeW*5/6, p[1]+13, fmt.Sprintf("%d", t.EF), "net-cell", "middle")
		cv.Text(p[0]+nodeW/2, p[1]+35, id, "net-id", "middle")
		cv.Text(p[0]+nodeW/6, p[1]+58, fmt.Sprintf("%d", t.LS), "net-cell", "middle")
		cv.Text(p[0]+nodeW/2, p[1]+58, fmt.Sprintf("%d", t.TotalFloat), "net-cell float", "middle")
		cv.Text(p[0]+nodeW*5/6, p[1]+58, fmt.Sprintf("%d", t.LF), "net-cell", "middle")
	}

	cv.Text(40, 26, tr(lang,
		"Baris atas: ES | durasi | EF   -   Baris bawah: LS | total float | LF",
		"Top row: ES | duration | EF   -   Bottom row: LS | total float | LF"), "chart-title", "start")
	return template.HTML(cv.SVG("network"))
}

// OrgChart menggambar bagan organisasi proyek lima tingkat.
func OrgChart(team []model.Member, sponsor model.Text, lang string) template.HTML {
	boxW, boxH := 168.0, 54.0
	cv := NewCanvas(1000, 420, Padding{})
	cv.Describe(
		tr(lang, "Bagan organisasi proyek", "Project organisation chart"),
		tr(lang,
			"Lima tingkat dari sponsor sampai tim pelaksana, dengan garis pelaporan yang menunjukkan siapa bertanggung jawab kepada siapa.",
			"Five levels from sponsor to delivery team, with reporting lines showing who answers to whom."))

	center := cv.W / 2
	// Tingkat 1: sponsor.
	sx := center - boxW/2
	cv.Rect(sx, 16, boxW, boxH, "org-box sponsor")
	cv.Text(center, 38, tr(lang, "Project Sponsor", "Project Sponsor"), "org-role", "middle")
	cv.Text(center, 54, truncate(sponsor.Get(lang), 26), "org-person", "middle")

	// Tingkat 2: project manager.
	var pm model.Member
	for _, m := range team {
		if m.Level == 2 {
			pm = m
		}
	}
	py := 110.0
	cv.Rect(sx, py, boxW, boxH, "org-box pm")
	cv.Text(center, py+22, pm.Title.Get(lang), "org-role", "middle")
	cv.Text(center, py+38, pm.Person, "org-person", "middle")
	cv.Line(center, 16+boxH, center, py, "org-link")

	// Tingkat 3: core lead.
	var leads, delivery []model.Member
	for _, m := range team {
		switch m.Level {
		case 3:
			leads = append(leads, m)
		case 4:
			delivery = append(delivery, m)
		}
	}
	ly := 210.0
	spacing := cv.W / float64(len(leads)+1)
	leadX := map[model.Role]float64{}
	for i, m := range leads {
		x := spacing * float64(i+1)
		leadX[m.Role] = x
		cv.Rect(x-boxW/2, ly, boxW, boxH, "org-box lead")
		cv.Text(x, ly+22, m.Title.Get(lang), "org-role", "middle")
		cv.Text(x, ly+38, string(m.Role), "org-person", "middle")
		cv.Path(fmt.Sprintf("M%s %s V%s H%s V%s", f(center), f(py+boxH), f(ly-18), f(x), f(ly)), "org-link")
	}

	// Tingkat 4: tim pelaksana, semuanya melapor ke Technical Lead.
	dy := 310.0
	dspacing := cv.W / float64(len(delivery)+1)
	tlX := leadX[model.RoleTL]
	for i, m := range delivery {
		x := dspacing * float64(i+1)
		cls := "org-box delivery"
		if m.Optional {
			cls += " optional"
		}
		cv.Rect(x-boxW/2*0.82, dy, boxW*0.82, boxH, cls)
		cv.Text(x, dy+21, truncate(m.Title.Get(lang), 20), "org-role small", "middle")
		if m.Person != "" {
			cv.Text(x, dy+37, truncate(m.Person, 20), "org-person small", "middle")
		} else {
			cv.Text(x, dy+37, string(m.Role), "org-person small dim", "middle")
		}
		cv.Path(fmt.Sprintf("M%s %s V%s H%s V%s", f(tlX), f(ly+boxH), f(dy-18), f(x), f(dy)), "org-link")
	}
	return template.HTML(cv.SVG("org-chart"))
}

// Fishbone menggambar diagram sebab-akibat Ishikawa.
func Fishbone(p model.FishboneProblem, lang string) template.HTML {
	cv := NewCanvas(1000, 460, Padding{})
	cv.Describe(
		tr(lang, "Diagram sebab-akibat", "Cause-and-effect diagram"),
		p.Effect.Get(lang))

	spineY := 230.0
	spineX1, spineX2 := 60.0, 780.0
	cv.Line(spineX1, spineY, spineX2, spineY, "fish-spine")
	cv.Path(fmt.Sprintf("M%s %s l-14 -8 l0 16 Z", f(spineX2), f(spineY)), "fish-arrow")
	cv.Rect(spineX2+6, spineY-40, 200, 80, "fish-effect-box")
	wrapText(cv, spineX2+16, spineY-18, 186, p.Effect.Get(lang), "fish-effect", 3)

	n := len(p.Branches)
	step := (spineX2 - spineX1 - 90) / float64((n+1)/2)
	for i, b := range p.Branches {
		up := i%2 == 0
		col := i / 2
		bx := spineX1 + 70 + float64(col)*step
		var by, ty float64
		if up {
			by, ty = spineY-95, spineY-108
		} else {
			by, ty = spineY+95, spineY+108
		}
		cv.Line(bx-46, by, bx, spineY, "fish-bone")
		cv.Text(bx-46, ty, b.Category.Get(lang), "fish-category", "middle")
		for j, cause := range b.Causes {
			off := float64(j+1) * 24
			cx := bx - 46 + off*0.46
			cy := by + off*(map[bool]float64{true: 1, false: -1})[up]
			cv.Line(cx, cy, cx+54, cy, "fish-twig")
			anchor := "start"
			cv.Text(cx+58, cy+4, truncate(cause.Get(lang), 30), "fish-cause", anchor)
		}
	}
	return template.HTML(cv.SVG("fishbone"))
}

// wrapText memecah teks menjadi beberapa baris tspan sederhana.
func wrapText(cv *Canvas, x, y, width float64, s, class string, maxLines int) {
	words := splitWords(s)
	perLine := int(width / 6.4)
	var lines []string
	cur := ""
	for _, w := range words {
		if len(cur)+len(w)+1 > perLine && cur != "" {
			lines = append(lines, cur)
			cur = w
			if len(lines) == maxLines {
				break
			}
		} else if cur == "" {
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" && len(lines) < maxLines {
		lines = append(lines, cur)
	}
	for i, ln := range lines {
		cv.Text(x, y+float64(i)*15, ln, class, "start")
	}
}

func splitWords(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func sortInts(v []int) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
