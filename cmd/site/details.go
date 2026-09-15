package main

import (
	"html/template"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/site"
)

// detailFuncs menyiapkan grafik rincian untuk bagian yang sebelumnya hanya
// berupa tabel. Seperti visualFuncs, fungsi di sini hanya menerjemahkan
// struct analisis ke masukan grafik.
func detailFuncs(a *site.Analysis) template.FuncMap {
	tl := func(lang, id, en string) string {
		if lang == "id" {
			return id
		}
		return en
	}
	pct := func(lang string, dec int) func(float64) string {
		return func(v float64) string { return render.Pct(v, dec, lang) }
	}
	num := func(lang string, dec int) func(float64) string {
		return func(v float64) string { return render.Num(v, dec, lang) }
	}
	money := func(lang string) func(float64) string {
		return func(v float64) string { return render.RpShort(v, lang) }
	}
	return template.FuncMap{
		// --- ringkasan untuk kolom "Artinya" -----------------------------------
		"riskFrequencyGap": func() riskGap { return riskFrequencyGap(a) },
		"estimateSummary":  func() estimateStats { return estimateSummary(a) },
		"floatSummary":     func() floatStats { return floatSummary(a) },
		"movedSummary":     func() movedStats { return movedSummary(a) },
		"cvSummary":        func() cvStats { return cvSummary(a) },
		"wbsSummary":       func() wbsStats { return wbsSummary() },

		// --- simulasi terpadu ------------------------------------------------
		"rhoSweepChart": func(lang string) template.HTML {
			var xs, p80, sd, onTime, realised []float64
			for _, p := range a.RhoSweep {
				xs = append(xs, p.Rho)
				p80, sd = append(p80, p.P80), append(sd, p.StdDev)
				onTime, realised = append(onTime, p.OnTime), append(realised, p.Realised)
			}
			return render.SweepPanels(xs, "rho", simulate.DefaultRho, []render.SweepPanel{
				{Title: tl(lang, "P80 durasi (hari kerja)", "P80 duration (working days)"), Values: p80, Format: num(lang, 0)},
				{Title: tl(lang, "Simpangan baku durasi", "Duration standard deviation"), Values: sd, Format: num(lang, 2)},
				{Title: tl(lang, "Peluang tepat waktu", "On-time probability"), Values: onTime, Format: pct(lang, 1)},
				{Title: tl(lang, "Korelasi terealisasi", "Realised correlation"), Values: realised, Format: num(lang, 3)},
			}, tl(lang, "Kepekaan terhadap asumsi korelasi rho", "Sensitivity to the correlation assumption rho"),
				tl(lang, "Empat panel memakai sumbu rho yang sama. Garis putus-putus adalah rho yang dipakai situs. Label di pojok setiap panel menulis perubahan dari rho terkecil ke terbesar.", "Four panels share the same rho axis. The dashed line is the rho the site uses. The label in each panel's corner states the change from the smallest to the largest rho."), lang)
		},
		"riskSweepChart": func(lang string) template.HTML {
			var xs, phi, mean, p95, dur, many []float64
			for _, p := range a.RiskSweep {
				xs = append(xs, p.Loading)
				phi, mean, p95 = append(phi, p.Phi), append(mean, p.CostMean), append(p95, p.CostP95)
				dur, many = append(dur, p.DurP95), append(many, p.ManyRisks)
			}
			return render.SweepPanels(xs, "lambda", model.RiskLoading, []render.SweepPanel{
				{Title: tl(lang, "Rerata biaya", "Mean cost"), Values: mean, Format: money(lang)},
				{Title: tl(lang, "P95 biaya", "P95 cost"), Values: p95, Format: money(lang)},
				{Title: tl(lang, "P95 durasi (hari kerja)", "P95 duration (working days)"), Values: dur, Format: num(lang, 0)},
				{Title: tl(lang, "phi terealisasi", "Realised phi"), Values: phi, Format: num(lang, 3)},
				{Title: tl(lang, "Peluang ≥ 4 risiko sekaligus", "Chance of ≥ 4 risks at once"), Values: many, Format: pct(lang, 1)},
			}, tl(lang, "Kepekaan terhadap kekuatan penggerak bersama lambda", "Sensitivity to the shared-driver strength lambda"),
				tl(lang, "Lima panel memakai sumbu lambda yang sama. Rerata biaya yang datar dan ekor yang menebal menunjukkan kopula menata ulang kapan risiko terjadi, bukan menambah risiko.", "Five panels share the same lambda axis. A flat mean cost with a thickening tail shows the copula rearranges when risks strike rather than adding risk."), lang)
		},
		"gertLoopChart": func(lang string) template.HTML {
			var loops []render.LoopBar
			for _, g := range a.GERT {
				loops = append(loops, render.LoopBar{Label: g.Loop.ID + " " + g.Loop.Label.Get(lang), P: g.Loop.FailProb, Analytic: g.Cycles, Simulated: g.MCCycles, Q90: g.Q90})
			}
			return render.LoopBars(loops, 4, lang)
		},
		"timeCostChart": func(lang string) template.HTML {
			var bars []render.ColumnBar
			plan := 0.0
			for i, r := range a.Ladder {
				plan = r.TimeCostPlan
				bars = append(bars, render.ColumnBar{Label: "L" + render.Num(float64(i), 0, lang), Value: r.TimeCost, Class: "step-" + render.Num(float64(i), 0, lang), Note: site.LadderNames[i].Get(lang)})
			}
			return render.ColumnBars(bars, plan, tl(lang, "pada jadwal rencana", "on the planned schedule"), true,
				tl(lang, "Rerata biaya sewa dan langganan per lapisan", "Mean rental and subscription cost per layer"),
				tl(lang, "Setiap kolom adalah rerata biaya sewa dan langganan pada satu lapisan realisme. Garis putus-putus adalah nilainya pada jadwal rencana, yang juga tercatat di BAC.", "Each column is the mean rental and subscription cost at one realism layer. The dashed line is its value on the planned schedule, which is also what the BAC records."), lang)
		},
		"riskFrequencyChart": func(lang string) template.HTML {
			fin := a.Final()
			var rows []render.DumbbellRow
			for _, r := range model.Risks {
				freq := float64(fin.RiskHits[r.ID]) / float64(fin.Config.Iterations)
				rows = append(rows, render.DumbbellRow{Label: r.ID + " " + r.Title.Get(lang), A: r.ResidualProb, B: freq})
			}
			return render.DumbbellRows(rows, tl(lang, "peluang residual di register", "residual probability in the register"), tl(lang, "frekuensi di simulasi L4", "frequency in the L4 simulation"),
				tl(lang, "Peluang register dibanding frekuensi simulasi", "Register probability versus simulated frequency"),
				tl(lang, "Lingkaran kosong adalah peluang residual yang ditulis di register; lingkaran penuh adalah seberapa sering risiko itu benar-benar terjadi di sepuluh ribu iterasi. Bila kopula bekerja benar, keduanya berimpit.", "The hollow circle is the residual probability written in the register; the filled circle is how often the risk actually occurred across ten thousand iterations. If the copula works, the two coincide."), lang)
		},

		// --- prakiraan berjalan ------------------------------------------------
		"forecastRiskChart": func(lang string) template.HTML {
			fl, fin, fc := a.InFlight, a.Final(), a.Forecast
			if fl == nil {
				return ""
			}
			var rows []render.DumbbellRow
			for _, r := range model.Risks {
				row := render.DumbbellRow{
					Label: r.ID + " " + r.Title.Get(lang),
					A:     float64(fin.RiskHits[r.ID]) / float64(fin.Config.Iterations),
					B:     float64(fc.RiskHits[r.ID]) / float64(fc.Config.Iterations),
				}
				if fl.ClosedRisk[r.ID] {
					row.Muted = true
					row.Note = tl(lang, "ditutup: "+r.Status, "closed: "+map[string]string{"terjadi": "occurred", "terpantau": "monitored", "terbuka": "open"}[r.Status])
				}
				rows = append(rows, row)
			}
			return render.DumbbellRows(rows, tl(lang, "frekuensi saat perencanaan", "frequency at planning"), tl(lang, "frekuensi dari tanggal data", "frequency from the data date"),
				tl(lang, "Peluang risiko sebelum dan sesudah proyek berjalan", "Risk chances before and after the project started"),
				tl(lang, "Lingkaran kosong adalah frekuensi risiko di simulasi perencanaan; lingkaran penuh adalah frekuensinya di prakiraan dari tanggal data. Baris pudar adalah risiko yang sudah ditutup register dan tidak disampel lagi.", "The hollow circle is the risk's frequency in the planning simulation; the filled circle is its frequency in the forecast from the data date. Faded rows are risks the register has closed and no longer samples."), lang)
		},

		// --- PERT -------------------------------------------------------------
		"threePointChart": func(lang string) template.HTML {
			var rows []render.RangeRow
			for _, act := range model.Activities {
				if act.Milestone {
					continue
				}
				cls := ""
				if a.Plan.Task(act.ID).Critical {
					cls = "critical"
				}
				rows = append(rows, render.RangeRow{Label: act.ID + " " + act.Name.Get(lang), Lo: float64(act.Optimistic), Mode: float64(act.Duration), Hi: float64(act.Pessimistic), Mean: schedule.ActivityExpected(act), Class: cls})
			}
			return render.RangeRows(rows, tl(lang, "hari kerja", "working days"),
				tl(lang, "Estimasi tiga titik setiap aktivitas", "Three-point estimate of every activity"),
				tl(lang, "Setiap baris adalah satu aktivitas: garis dari durasi optimistis ke pesimistis, garis tegak di durasi paling mungkin, dan belah ketupat di rerata beta-PERT. Garis merah berada di jalur kritis.", "Each row is one activity: a line from the optimistic to the pessimistic duration, a tick at the most likely duration, and a diamond at the beta-PERT mean. Red lines are on the critical path."), lang)
		},

		// --- jadwal -------------------------------------------------------------
		"floatChart": func(lang string) template.HTML {
			type fr struct {
				act         model.Activity
				total, free int
			}
			var list []fr
			for _, act := range model.Activities {
				t := a.Plan.Task(act.ID)
				if t.TotalFloat > 0 {
					list = append(list, fr{act, t.TotalFloat, t.FreeFloat})
				}
			}
			sort.SliceStable(list, func(i, j int) bool { return list[i].total > list[j].total })
			var rows []render.StackRow
			for _, x := range list {
				rows = append(rows, render.StackRow{
					Label: x.act.ID + " " + x.act.Name.Get(lang),
					Parts: []render.StackPart{
						{Value: float64(x.free), Class: "seg-free", Label: tl(lang, "bebas", "free")},
						{Value: float64(x.total - x.free), Class: "seg-shared", Label: tl(lang, "bersama", "shared")},
					},
					Note: render.Num(float64(x.total), 0, lang) + tl(lang, " hari", " days"),
				})
			}
			return render.StackRows(rows, []render.LegendItem{
				{Label: tl(lang, "float bebas: dipakai tanpa menggeser penerus", "free float: usable without moving successors"), Class: "seg-free"},
				{Label: tl(lang, "sisa float total: dipakai menggeser penerus", "rest of total float: using it moves successors"), Class: "seg-shared"},
			}, false, tl(lang, "hari kerja", "working days"),
				tl(lang, "Ruang gerak setiap aktivitas non-kritis", "Room to move for every non-critical activity"),
				tl(lang, "Panjang batang adalah total float. Bagian hijau adalah float bebas yang tidak mengganggu aktivitas lain; bagian kuning hanya bisa dipakai dengan menggeser penerusnya.", "Bar length is total float. The green part is free float that disturbs no other activity; the yellow part can only be used by moving its successors."), lang)
		},

		// --- optimasi -----------------------------------------------------------
		"movedTasksChart": func(lang string) template.HTML {
			// MovedTasks sudah terurut dari pergeseran terbesar.
			moved := a.MovedTasks()
			byID := map[string]model.Activity{}
			for _, act := range model.Activities {
				byID[act.ID] = act
			}
			var rows []render.StackRow
			for _, t := range moved {
				rows = append(rows, render.StackRow{
					Label: t.ID + " " + byID[t.ID].Name.Get(lang),
					Parts: []render.StackPart{
						{Value: float64(t.CarriedDays), Class: "seg-carried", Label: render.Num(float64(t.CarriedDays), 0, lang)},
						{Value: float64(t.WaitDays), Class: "seg-wait", Label: render.Num(float64(t.WaitDays), 0, lang)},
						{Value: float64(t.StretchDays), Class: "seg-stretch", Label: render.Num(float64(t.StretchDays), 0, lang)},
					},
					Note: "+" + render.Num(float64(t.Delay()), 0, lang),
				})
			}
			return render.StackRows(rows, []render.LegendItem{
				{Label: tl(lang, "terbawa dari pendahulu", "carried from predecessors"), Class: "seg-carried"},
				{Label: tl(lang, "menunggu orang", "waiting for people"), Class: "seg-wait"},
				{Label: tl(lang, "memanjang saat dikerjakan", "stretched while in progress"), Class: "seg-stretch"},
			}, false, tl(lang, "hari kerja", "working days"),
				tl(lang, "Dari mana setiap hari keterlambatan datang", "Where every day of delay comes from"),
				tl(lang, "Setiap baris adalah aktivitas yang bergeser akibat levelling, diurutkan dari pergeseran terbesar. Segmen memecah pergeseran menjadi hari yang terbawa dari pendahulu, hari menunggu orang, dan hari memanjang karena paruh waktu atau ujian.", "Each row is an activity moved by levelling, sorted by the largest shift. Segments split the shift into days carried from predecessors, days waiting for people, and days stretched by half-time roles or exams."), lang)
		},
		"fastTrackChart": func(lang string) template.HTML {
			var pts []render.ScatterPoint
			for _, ft := range a.FastTracks {
				p := render.ScatterPoint{X: float64(ft.DaysSaved), Y: ft.ExpectedRework, Class: "ft-none",
					Note: ft.Pred + " → " + ft.Succ + ": " + render.Num(float64(ft.DaysSaved), 0, lang) + tl(lang, " hari, rework ", " days, rework ") + render.Rp(ft.ExpectedRework, lang)}
				switch {
				case ft.Viable():
					p.Class, p.Label = "ft-viable", ft.Pred+"→"+ft.Succ
				case ft.SameResource:
					p.Class = "ft-same"
				}
				pts = append(pts, p)
			}
			return render.ScatterLabeled(pts, tl(lang, "hari yang dihemat pada jaringan CPM", "days saved on the CPM network"), tl(lang, "rework harapan", "expected rework"), true,
				[]render.LegendItem{
					{Label: tl(lang, "layak", "viable"), Class: "ft-viable"},
					{Label: tl(lang, "orang yang sama: tidak mungkin", "same person: impossible"), Class: "ft-same"},
					{Label: tl(lang, "tidak menghemat hari", "saves no days"), Class: "ft-none"},
				},
				tl(lang, "Kandidat fast-tracking: hari dihemat dibanding rework", "Fast-tracking candidates: days saved versus rework"),
				tl(lang, "Setiap titik adalah satu pasangan pendahulu-penerus yang dikerjakan tumpang tindih. Makin ke kanan makin banyak hari dihemat, makin ke atas makin mahal rework harapannya. Hanya titik hijau yang benar-benar bisa dijalankan.", "Each dot is one predecessor-successor pair worked in overlap. Further right saves more days, higher up costs more expected rework. Only green dots can actually run."), lang)
		},

		// --- biaya --------------------------------------------------------------
		"activityCVChart": func(lang string) template.HTML {
			var rows []render.DivRow
			for _, r := range a.Rows {
				if r.AC == 0 {
					continue
				}
				rows = append(rows, render.DivRow{Label: r.ID + " " + r.Name.Get(lang), Value: r.CV, Note: "CPI " + render.Num(r.CPI, 2, lang)})
			}
			sort.SliceStable(rows, func(i, j int) bool { return rows[i].Value < rows[j].Value })
			return render.DivergingRows(rows, true, tl(lang, "boros", "over cost"), tl(lang, "hemat", "under cost"),
				tl(lang, "Varians biaya per aktivitas yang sudah berbiaya", "Cost variance of every activity with actual cost"),
				tl(lang, "Setiap batang adalah CV = EV − AC satu aktivitas, diurutkan dari yang paling boros. Batang ke kiri berarti uang keluar lebih banyak dari nilai pekerjaan yang dihasilkan.", "Each bar is CV = EV − AC for one activity, sorted from the most over cost. Bars to the left mean more money went out than the work earned."), lang)
		},

		// --- piagam -------------------------------------------------------------
		"wbsBudgetChart": func(lang string) template.HTML {
			classes := []string{"seg-a", "seg-b", "seg-c", "seg-d", "seg-e"}
			var rows []render.StackRow
			var legend []render.LegendItem
			for _, ph := range model.WBSPhases {
				row := render.StackRow{Label: ph.Code + " " + ph.Name.Get(lang)}
				for k, pk := range ph.Packages {
					sum := 0.0
					for _, act := range model.Activities {
						if act.WBS == pk.Code {
							sum += act.Budget(model.RateCard)
						}
					}
					row.Parts = append(row.Parts, render.StackPart{Value: sum, Class: classes[k%len(classes)], Label: pk.Code})
				}
				rows = append(rows, row)
			}
			for k := range classes {
				legend = append(legend, render.LegendItem{Label: tl(lang, "paket ke-", "package ") + render.Num(float64(k+1), 0, lang), Class: classes[k]})
			}
			return render.StackRows(rows, legend, true, "",
				tl(lang, "Anggaran aktivitas per fase dan paket kerja", "Activity budget by phase and work package"),
				tl(lang, "Setiap baris adalah satu fase WBS; segmen adalah paket kerjanya dengan kode tertulis di dalam. Panjang batang adalah anggaran bottom-up fase itu.", "Each row is one WBS phase; segments are its work packages with codes written inside. Bar length is the phase's bottom-up budget."), lang)
		},

		// --- coretax ------------------------------------------------------------
		"coretaxTimelineChart": func(lang string) template.HTML {
			var evs []render.TimelineEvent
			for _, m := range model.CoretaxTimeline {
				evs = append(evs, render.TimelineEvent{Date: m.Date, Label: m.Label.Get(lang), Class: "kind-" + m.Kind})
			}
			return render.EventTimeline(evs, tl(lang, "Linimasa Coretax pada skala waktu sebenarnya", "The Coretax timeline on its real time scale"),
				tl(lang, "Setiap titik adalah satu tonggak bersumber, diletakkan pada tanggalnya. Jarak antar-titik adalah jeda waktu sebenarnya: bertahun-tahun membangun, lalu dampak dalam tiga bulan pertama setelah go-live.", "Each dot is one sourced milestone, placed on its date. The gaps between dots are the real time gaps: years of building, then impact within the first three months after go-live."), lang)
		},
		"coretaxExposureChart": func(lang string) template.HTML {
			d := a.Coretax
			return render.ColumnBars([]render.ColumnBar{
				{Label: tl(lang, "Seluruh kontrak Coretax", "The whole Coretax contract"), Value: d.TotalContract, Class: "exposure-build", Note: tl(lang, "sekali, selama bertahun-tahun", "once, over years")},
				{Label: tl(lang, "Penerimaan berisiko dalam sebulan", "Revenue at risk in one month"), Value: d.TotalContract * d.ExposureRatio, Class: "exposure-loss", Note: render.Ratio(d.ExposureRatio, lang) + tl(lang, " nilai kontrak", " the contract value")},
			}, 0, "", true,
				tl(lang, "Biaya membangun dibanding kerugian bila gagal", "Cost to build versus loss if it fails"),
				tl(lang, "Kolom kiri adalah seluruh nilai kontrak pembangunan sistem. Kolom kanan adalah potensi penerimaan pajak yang hilang dalam satu bulan gangguan. Keduanya pada skala yang sama.", "The left column is the whole system build contract. The right column is the tax revenue potentially lost in one month of disruption. Both share one scale."), lang)
		},
	}
}

// riskGap adalah selisih terbesar antara peluang register dan frekuensi simulasi.
type riskGap struct {
	ID  string
	Gap float64 // mutlak, 0..1
}

func riskFrequencyGap(a *site.Analysis) riskGap {
	fin := a.Final()
	var g riskGap
	for _, r := range model.Risks {
		freq := float64(fin.RiskHits[r.ID]) / float64(fin.Config.Iterations)
		if d := math.Abs(freq - r.ResidualProb); d > g.Gap || g.ID == "" {
			g = riskGap{ID: r.ID, Gap: d}
		}
	}
	return g
}

// estimateStats meringkas estimasi tiga titik.
type estimateStats struct {
	Widest      model.Activity
	Total       int // aktivitas non-milestone
	RightSkewed int // rerata beta-PERT di atas durasi paling mungkin
}

func estimateSummary(a *site.Analysis) estimateStats {
	var st estimateStats
	for _, act := range model.Activities {
		if act.Milestone {
			continue
		}
		st.Total++
		if schedule.ActivityExpected(act) > float64(act.Duration)+1e-9 {
			st.RightSkewed++
		}
		if st.Widest.ID == "" || act.Pessimistic-act.Optimistic > st.Widest.Pessimistic-st.Widest.Optimistic {
			st.Widest = act
		}
	}
	return st
}

// floatStats meringkas ruang gerak aktivitas.
type floatStats struct {
	WithFloat, Total int
	Max              model.Activity
	MaxFloat         int
	FreeOnly         int // aktivitas yang seluruh floatnya float bebas
}

func floatSummary(a *site.Analysis) floatStats {
	var st floatStats
	for _, act := range model.Activities {
		st.Total++
		t := a.Plan.Task(act.ID)
		if t.TotalFloat <= 0 {
			continue
		}
		st.WithFloat++
		if t.FreeFloat == t.TotalFloat {
			st.FreeOnly++
		}
		if t.TotalFloat > st.MaxFloat {
			st.Max, st.MaxFloat = act, t.TotalFloat
		}
	}
	return st
}

// movedStats menjumlahkan pecahan keterlambatan akibat levelling.
type movedStats struct {
	Count                  int
	Carried, Wait, Stretch int
	// Biggest adalah kunci penyebab terbesar: carried, wait, atau stretch.
	Biggest string
}

func movedSummary(a *site.Analysis) movedStats {
	var st movedStats
	for _, t := range a.MovedTasks() {
		st.Count++
		st.Carried += t.CarriedDays
		st.Wait += t.WaitDays
		st.Stretch += t.StretchDays
	}
	st.Biggest = biggestCause(st.Carried, st.Wait, st.Stretch)
	return st
}

// biggestCause memilih penyebab pergeseran terbesar; seri dimenangkan
// terbawa, lalu menunggu.
func biggestCause(carried, wait, stretch int) string {
	switch {
	case wait > carried && wait >= stretch:
		return "wait"
	case stretch > carried && stretch > wait:
		return "stretch"
	}
	return "carried"
}

// cvStats meringkas varians biaya per aktivitas.
type cvStats struct {
	WithAC, Over int
	Worst        model.Text
	WorstID      string
	WorstCV      float64
}

func cvSummary(a *site.Analysis) cvStats {
	var st cvStats
	for _, r := range a.Rows {
		if r.AC == 0 {
			continue
		}
		st.WithAC++
		if r.CV < 0 {
			st.Over++
		}
		if st.WorstID == "" || r.CV < st.WorstCV {
			st.WorstID, st.Worst, st.WorstCV = r.ID, r.Name, r.CV
		}
	}
	return st
}

// wbsStats menyebut fase dengan anggaran aktivitas terbesar.
type wbsStats struct {
	Top    model.WBSPhase
	Budget float64
	Share  float64
}

func wbsSummary() wbsStats {
	var st wbsStats
	total := 0.0
	for _, ph := range model.WBSPhases {
		sum := 0.0
		for _, act := range model.Activities {
			if model.PhaseOf(act.WBS).Code == ph.Code {
				sum += act.Budget(model.RateCard)
			}
		}
		total += sum
		if sum > st.Budget {
			st.Top, st.Budget = ph, sum
		}
	}
	if total > 0 {
		st.Share = st.Budget / total
	}
	return st
}
