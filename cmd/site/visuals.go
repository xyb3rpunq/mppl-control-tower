package main

import (
	"html/template"
	"math"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/site"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// visualFuncs menyiapkan grafik penjelas dari analisis. Setiap fungsi hanya
// menerjemahkan struct analisis ke masukan grafik; tidak ada angka yang
// dihitung ulang di sini.
func visualFuncs(a *site.Analysis) template.FuncMap {
	tl := func(lang, id, en string) string {
		if lang == "id" {
			return id
		}
		return en
	}
	return template.FuncMap{
		"commitmentTimeline": func(lang string) template.HTML {
			fj := a.ForecastJCL70
			items := []render.DateItem{
				{Label: tl(lang, "Rencana CPM", "CPM plan"), Status: tl(lang, "tidak berlaku: mengabaikan kapasitas orang", "not in force: ignores people's capacity"), Day: float64(a.Plan.Duration), Note: render.RpShort(a.BAC, lang), Class: "invalid"},
				{Label: "IEAC(t) Earned Schedule", Status: tl(lang, "tidak berlaku: hanya tren SPI", "not in force: SPI trend only"), Day: a.IEACt, Note: render.RpShort(a.Snapshot.EACTypical, lang), Class: "invalid"},
				{Label: "P80 PERT", Status: tl(lang, "tidak berlaku: hanya ketidakpastian durasi", "not in force: duration uncertainty only"), Day: a.Sim.P80, Class: "invalid"},
				{Label: tl(lang, "Jadwal levelling", "Levelled schedule"), Status: tl(lang, "tidak berlaku: tanpa ketidakpastian", "not in force: no uncertainty"), Day: float64(a.Level.Duration), Class: "invalid"},
				{Label: tl(lang, "JCL 70% berjalan", "In-flight 70% JCL"), Status: tl(lang, "BERLAKU: memakai realisasi", "IN FORCE: uses actuals"), Day: fj.Duration, Note: render.RpShort(fj.Budget, lang), Class: "inforce"},
				{Label: tl(lang, "JCL 70% perencanaan", "Planning 70% JCL"), Status: tl(lang, "digantikan: sebelum realisasi", "superseded: before actuals"), Day: a.JCL70.Duration, Note: render.RpShort(a.JCL70.Budget, lang), Class: "superseded"},
			}
			sortItems(items)
			return render.DateDots(items, []render.DateRef{{Day: a.StatusDay, Label: tl(lang, "tanggal data", "data date"), Class: "data"}}, a.Calendar,
				tl(lang, "Enam angka komitmen pada satu garis waktu", "Six commitment figures on one timeline"),
				tl(lang, "Setiap baris adalah satu angka komitmen yang pernah muncul di situs, diletakkan pada tanggal selesainya. Titik hijau adalah satu-satunya yang berlaku.", "Each row is one commitment figure the site has shown, placed at its finish date. The green dot is the only one in force."), lang)
		},
		"forecastDots": func(lang string) template.HTML {
			fc, fin, pr := a.Forecast, a.Final(), a.ForecastPrior
			items := []render.DateItem{
				{Label: tl(lang, "Rencana CPM", "CPM plan"), Status: tl(lang, "buta: ketidakpastian, kapasitas, risiko, realisasi", "blind to: uncertainty, capacity, risk, actuals"), Day: float64(a.Plan.Duration), Class: "invalid"},
				{Label: "Earned Value IEAC(t)", Status: tl(lang, "buta: kapasitas, ujian, risiko, rework", "blind to: capacity, exams, risk, rework"), Day: a.IEACt, Class: "invalid"},
				{Label: tl(lang, "P80 simulasi perencanaan", "Planning simulation P80"), Status: tl(lang, "buta: apa yang sudah terjadi", "blind to: what already happened"), Day: fin.DurP80, Class: "superseded"},
				{Label: tl(lang, "P80 berjalan tanpa belajar", "In-flight P80, no learning"), Status: tl(lang, "buta: pelajaran dari realisasi", "blind to: lessons from actuals"), Day: pr.DurP80, Class: "neutral"},
				{Label: tl(lang, "P80 berjalan terkalibrasi", "Calibrated in-flight P80"), Status: tl(lang, "paling lengkap", "most complete"), Day: fc.DurP80, Class: "inforce"},
			}
			sortItems(items)
			return render.DateDots(items, []render.DateRef{{Day: a.StatusDay, Label: tl(lang, "tanggal data", "data date"), Class: "data"}}, a.Calendar,
				tl(lang, "Lima prakiraan tanggal selesai", "Five finish-date forecasts"),
				tl(lang, "Setiap baris adalah satu metode prakiraan pada data yang sama; label kecil menyebut apa yang tidak dilihat metode itu.", "Each row is one forecasting method on the same data; the small label names what that method cannot see."), lang)
		},
		"optionMap": func(lang string) template.HTML {
			d := a.Decision
			if d == nil {
				return ""
			}
			var pts []render.OptionPoint
			for i, o := range d.Options {
				if o.Assumption.ID != "" || !o.JCL70.Feasible {
					continue
				}
				p := render.OptionPoint{Label: o.Name.Get(lang), Day: o.JCL70.Duration, Budget: o.JCL70.Budget, Class: "other"}
				if r := d.Robust; r != nil {
					p.DayLo, p.DayHi, p.BudgetLo, p.BudgetHi = r.DurLo[i], r.DurHi[i], r.BudgetLo[i], r.BudgetHi[i]
				}
				switch {
				case i == 0:
					p.Class = "base"
					p.Note = tl(lang, "komitmen yang berlaku", "commitment in force")
				case i == d.Cheapest:
					p.Class = "cheapest"
				case i == d.Fastest:
					p.Class = "fastest"
				}
				if i > 0 && o.DaysEarlier > 0 {
					p.Note = "−" + render.Num(o.DaysEarlier, 0, lang) + tl(lang, " hari · +", " days · +") + render.RpShort(o.ExtraBudget, lang) + " · " + render.RpShort(o.PricePerDay, lang) + tl(lang, "/hari", "/day")
				}
				pts = append(pts, p)
			}
			return render.OptionMap(pts, a.Calendar, lang)
		},
		"valueBandsChart": func(lang string) template.HTML {
			d := a.Decision
			if d == nil || len(d.Bands) == 0 {
				return ""
			}
			var lines []render.BenefitLine
			for _, o := range d.Options {
				if o.Assumption.ID != "" || !o.JCL70.Feasible {
					continue
				}
				lines = append(lines, render.BenefitLine{Label: o.Name.Get(lang), Days: o.DaysEarlier, Extra: o.ExtraBudget, Class: "opt-" + o.Key})
			}
			return render.ValueBandsChart(lines, bandSpans(d, d.Bands, lang, true), lang)
		},
		"scenarioStrips": func(lang string) template.HTML {
			d := a.Decision
			if d == nil || len(d.Bands) == 0 {
				return ""
			}
			rows := []render.StripRow{{Label: tl(lang, "Keputusan utama", "Main decision"), Bands: bandSpans(d, d.Bands, lang, false), Same: true, Base: true}}
			vmax := 0.0
			for _, b := range d.Bands {
				vmax = math.Max(vmax, b.From)
			}
			for _, s := range d.Scenarios {
				rows = append(rows, render.StripRow{Label: s.Name.Get(lang), Bands: bandSpans(d, s.Bands, lang, false), Same: s.Same})
				for _, b := range s.Bands {
					vmax = math.Max(vmax, b.From)
				}
			}
			var legend []render.LegendItem
			for _, o := range d.Options {
				if o.Assumption.ID == "" {
					legend = append(legend, render.LegendItem{Label: o.Name.Get(lang), Class: "opt-" + o.Key})
				}
			}
			return render.ScenarioStrips(rows, vmax*1.35, legend, lang)
		},
		"budgetBridge": func(lang string) template.HTML {
			fj, s := a.ForecastJCL70, a.Snapshot
			if a.Decision == nil || !fj.Feasible {
				return ""
			}
			steps := []render.BridgeStep{
				{Label: tl(lang, "Pagu Project Charter", "Project Charter cap"), Value: model.TotalAuthorised, Kind: "base"},
				{Label: tl(lang, "Tren biaya (EAC)", "Cost trend (EAC)"), Value: s.EACTypical - model.TotalAuthorised, Kind: "delta", Note: tl(lang, "CPI saja", "CPI only")},
				{Label: tl(lang, "Risiko, kapasitas, ujian", "Risk, capacity, exams"), Value: fj.Budget - s.EACTypical, Kind: "delta", Note: tl(lang, "yang tidak dilihat EAC", "what the EAC misses")},
				{Label: tl(lang, "Anggaran JCL 70%", "70% JCL budget"), Value: fj.Budget, Kind: "total", Note: tl(lang, "permintaan +", "request +") + render.RpShort(a.Decision.BudgetRequest, lang)},
			}
			return render.BridgeChart(steps, tl(lang, "Dari pagu ke anggaran yang dibutuhkan", "From the cap to the budget needed"),
				tl(lang, "Batang pertama adalah pagu piagam. Batang merah menumpuk apa yang ditambahkan tren biaya, lalu risiko, kapasitas, dan ujian. Batang biru adalah anggaran untuk keyakinan bersama 70%.", "The first bar is the charter cap. Red bars stack what the cost trend, then risk, capacity, and exams add. The blue bar is the budget for 70% joint confidence."), lang)
		},
		"decisionOvertimeCurve": func(lang string) template.HTML {
			d := a.Decision
			if d == nil {
				return ""
			}
			return render.OvertimeCurveChart(float64(d.Overtime.Levelled), curvePoints(a, d.Overtime.Points, lang), nil, "", a.Calendar, lang)
		},
		"planningOvertimeCurve": func(lang string) template.HTML {
			ot := a.Overtime
			var compare []render.CurvePoint
			for _, p := range ot.Points {
				k := ot.Levelled - p.Duration
				if cp, ok := a.Exact.PointAt(a.Exact.Normal - k); ok {
					compare = append(compare, render.CurvePoint{Day: float64(p.Duration), Cost: cp.CrashCost})
				}
			}
			return render.OvertimeCurveChart(float64(ot.Levelled), curvePoints(a, ot.Points, lang), compare, tl(lang, "crashing CPM (tidak terjalankan)", "CPM crashing (not executable)"), a.Calendar, lang)
		},
		"credibilityPanels": func(lang string) template.HTML {
			fl := a.InFlight
			if fl == nil {
				return ""
			}
			meaning := func(z float64) string {
				switch {
				case z < 0.05:
					return tl(lang, "masih derau: rencana tetap", "within noise: plan kept")
				case z > 0.9:
					return tl(lang, "di luar derau: data dipakai", "beyond noise: data used")
				}
				return tl(lang, "sebagian dipercaya", "partly trusted")
			}
			rows := []render.CredRow{
				{Label: tl(lang, "Durasi aktual / rencana", "Duration actual / plan"), Prior: 1, Observed: fl.ObservedRatio, Noise: math.Sqrt(fl.DurationVar), Z: fl.Credibility, Blended: fl.DurationFactor, Meaning: meaning(fl.Credibility)},
				{Label: tl(lang, "Biaya harian aktual / rencana", "Daily cost actual / plan"), Prior: 1, Observed: fl.ObservedCostRatio, Noise: math.Sqrt(fl.CostVar), Z: fl.CostCredibility, Blended: fl.CostFactor, Meaning: meaning(fl.CostCredibility)},
				{Label: tl(lang, "Kapasitas saat ujian", "Capacity during exams"), Prior: model.ExamCapacityFactor, Observed: math.Min(1, fl.ExamObserved), Noise: math.Sqrt(fl.ExamVar), Z: fl.ExamCredibility, Blended: fl.ExamFactor, Meaning: meaning(fl.ExamCredibility)},
			}
			return render.CredibilityPanels(rows, lang)
		},
		"forecastFrontierLine": func(lang string) template.HTML {
			var days, budgets []float64
			for _, p := range a.ForecastFrontier {
				if p.Feasible {
					days, budgets = append(days, p.Duration), append(budgets, p.Budget)
				}
			}
			fc, fj := a.Forecast, a.ForecastJCL70
			marks := []render.FrontierMark{
				{Day: fj.Duration, Budget: fj.Budget, Label: tl(lang, "komitmen JCL 70%", "70% JCL commitment"), Class: "inforce"},
				{Day: fc.DurP80, Budget: fc.CostP80, Label: tl(lang, "P80 × P80 (peluang bersama lebih kecil)", "P80 × P80 (lower joint chance)"), Class: "p80"},
			}
			return render.FrontierLine(days, budgets, marks, a.Calendar, lang)
		},
		"capacityTimeline": func(lang string) template.HTML {
			var spans []render.WindowSpan
			end := a.Level.Duration + 5
			for _, w := range model.AvailabilityWindows {
				from, to := a.Calendar.IndexOf(w.From), a.Calendar.IndexOf(w.To)
				if a.Calendar.ISOAt(to) != w.To {
					to--
				}
				roles := tl(lang, "seluruh tim", "whole team")
				if len(w.Roles) < len(model.Capacity) {
					var rs []string
					for _, r := range w.Roles {
						rs = append(rs, string(r))
					}
					roles = strings.Join(rs, ", ")
				}
				label := w.Label.Get(lang)
				if k := strings.Index(label, " ("); k > 0 {
					label = label[:k]
				}
				spans = append(spans, render.WindowSpan{FromDay: from, ToDay: to, Label: label, Factor: w.Factor, Roles: workcalRange(w.From, w.To, lang) + " · " + roles, Assumed: w.Asumsi})
				if to+5 > end {
					end = to + 5
				}
			}
			spans = append(spans, render.WindowSpan{FromDay: 0, ToDay: a.Level.Duration - 1, Label: tl(lang, "DevOps paruh waktu (piagam)", "Part-time DevOps (charter)"), Factor: model.Capacity[model.RoleOPS], Roles: "OPS"})
			return render.CapacityTimeline(spans, a.Level.Duration, end, a.StatusDay, a.Calendar, lang)
		},
		"phaseIndexDots": func(lang string) template.HTML {
			var rows []render.IndexRow
			for _, p := range a.Phases {
				rows = append(rows, render.IndexRow{Label: p.Code + " " + p.Name.Get(lang), SPI: p.SPI, CPI: p.CPI})
			}
			return render.IndexDots(rows, lang)
		},
		"riskCategoryBars": func(lang string) template.HTML {
			var labels []string
			var before, after []float64
			for _, c := range a.Risk.ByCategory {
				labels = append(labels, c.Category.Get(lang))
				before, after = append(before, c.EMV), append(after, c.ResidualEMV)
			}
			return render.PairBars(labels, before, after, tl(lang, "EMV sebelum mitigasi", "EMV before mitigation"), tl(lang, "EMV setelah mitigasi", "EMV after mitigation"),
				tl(lang, "Paparan risiko per kategori", "Risk exposure by category"),
				tl(lang, "Batang muda adalah EMV sebelum mitigasi, batang pekat setelah mitigasi. Selisih panjangnya adalah nilai rupiah yang dibeli rencana mitigasi.", "The light bar is EMV before mitigation, the dark bar after. The length difference is the rupiah value the mitigation plan buys."), lang)
		},
	}
}

// sortItems mengurutkan baris garis waktu dari tanggal paling awal.
func sortItems(items []render.DateItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Day < items[j-1].Day; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// bandSpans menerjemahkan pita nilai keputusan ke masukan grafik. withCI
// menempelkan interval bootstrap batas bila ada.
func bandSpans(d *site.Decision, bands []site.ValueBand, lang string, withCI bool) []render.BandSpan {
	var out []render.BandSpan
	for k, b := range bands {
		o := d.Options[b.Option]
		s := render.BandSpan{From: b.From, To: b.To, Label: o.Name.Get(lang), Class: "opt-" + o.Key}
		if withCI && d.Robust != nil && k > 0 && k-1 < len(d.Robust.EdgeLo) {
			s.EdgeLo, s.EdgeHi = d.Robust.EdgeLo[k-1], d.Robust.EdgeHi[k-1]
		}
		out = append(out, s)
	}
	return out
}

// curvePoints menerjemahkan rencana lembur ke titik kurva.
func curvePoints(a *site.Analysis, pts []compress.OvertimePoint, lang string) []render.CurvePoint {
	var out []render.CurvePoint
	for _, p := range pts {
		out = append(out, render.CurvePoint{Day: float64(p.Duration), Cost: p.Cost, Net: p.Net, Note: a.OvertimeRoleText(p, lang)})
	}
	return out
}

// workcalRange menulis rentang tanggal pendek, mis. "3 Nov – 15 Nov 2025".
func workcalRange(from, to, lang string) string {
	return workcal.FormatDateShort(from, lang) + " – " + workcal.FormatDate(to, lang)
}
