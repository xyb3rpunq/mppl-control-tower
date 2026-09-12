// Command site membangun seluruh situs statis Control Tower MPPL.
//
// Keluarannya adalah direktori yang bisa dilayani apa adanya oleh GitHub Pages
// atau server berkas statis mana pun: tidak ada basis data, tidak ada proses
// yang harus tetap hidup, dan tidak ada langkah render di sisi klien untuk
// menampilkan isi halaman.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyb3rpunq/mppl-control-tower/internal/coretax"
	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/risk"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/site"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// PageData adalah seluruh konteks yang dilihat sebuah templat.
type PageData struct {
	Lang      string
	OtherLang string
	Page      site.Page
	Pages     []site.Page
	Canonical string
	AltURL    string
	BaseURL   string
	Title     string
	Desc      string
	BuildTime string
	Content   template.HTML
	Analysis  *site.Analysis
	Examples  map[string]site.WorkedExample
	Charter   model.Charter
	Risk      risk.Register
}

// T menerjemahkan kunci i18n dalam bahasa halaman.
func (d PageData) T(key string) string { return i18n.T(d.Lang, key) }

// Tx mengambil teks dwibahasa dalam bahasa halaman.
func (d PageData) Tx(t model.Text) string { return t.Get(d.Lang) }

func main() {
	out := flag.String("out", "dist", "direktori keluaran")
	baseURL := flag.String("base", "", "URL dasar situs, mis. https://xyb3rpunq.github.io/mppl-control-tower")
	statusDate := flag.String("status", model.DefaultStatusDate, "tanggal data untuk pelaporan Earned Value")
	flag.Parse()

	start := time.Now()
	analysis, err := site.Build(*statusDate)
	if err != nil {
		log.Fatalf("gagal menjalankan analisis: %v", err)
	}
	log.Printf("analisis selesai dalam %s: %d aktivitas, durasi %d hari kerja, %d temuan",
		time.Since(start).Round(time.Millisecond),
		len(model.Activities), analysis.Plan.Duration, len(analysis.Findings))

	if err := os.RemoveAll(*out); err != nil {
		log.Fatalf("gagal membersihkan %s: %v", *out, err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatalf("gagal membuat %s: %v", *out, err)
	}

	tmpl, err := loadTemplates(analysis)
	if err != nil {
		log.Fatalf("gagal memuat templat: %v", err)
	}

	pages := 0
	for _, lang := range i18n.Langs {
		examples := analysis.Examples(lang)
		for _, p := range site.Pages {
			data := PageData{
				Lang:      lang,
				OtherLang: i18n.OtherLang(lang),
				Page:      p,
				Pages:     site.Pages,
				Canonical: strings.TrimRight(*baseURL, "/") + site.PathFor(p.Route, lang),
				AltURL:    strings.TrimRight(*baseURL, "/") + site.PathFor(p.Route, i18n.OtherLang(lang)),
				BaseURL:   strings.TrimRight(*baseURL, "/"),
				Title:     p.NavLabel(lang) + " - " + i18n.T(lang, "site.name"),
				Desc:      p.Summary.Get(lang),
				BuildTime: time.Now().UTC().Format("2006-01-02 15:04 MST"),
				Analysis:  analysis,
				Examples:  examples,
				Charter:   model.ProjectCharter,
				Risk:      analysis.Risk,
			}
			if p.Route == "/" {
				data.Title = i18n.T(lang, "site.name") + " - " + i18n.T(lang, "site.tagline")
			}

			// Isi halaman dirender lebih dulu, lalu disisipkan ke kerangka.
			// Templat Go tidak bisa memilih templat lewat nama variabel, jadi
			// pemilihannya dilakukan di sini.
			var body bytes.Buffer
			if err := tmpl.ExecuteTemplate(&body, p.Template, data); err != nil {
				log.Fatalf("gagal merender isi %s (%s): %v", p.Route, lang, err)
			}
			data.Content = template.HTML(body.String())

			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
				log.Fatalf("gagal merender kerangka %s (%s): %v", p.Route, lang, err)
			}
			dir := filepath.Join(*out, filepath.FromSlash(strings.Trim(site.PathFor(p.Route, lang), "/")))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				log.Fatalf("gagal membuat direktori %s: %v", dir, err)
			}
			if err := os.WriteFile(filepath.Join(dir, "index.html"), buf.Bytes(), 0o644); err != nil {
				log.Fatalf("gagal menulis %s: %v", dir, err)
			}
			pages++
		}
	}

	if err := copyStatic("web/static", filepath.Join(*out, "assets")); err != nil {
		log.Fatalf("gagal menyalin aset statis: %v", err)
	}
	if err := writeExtras(*out, *baseURL, analysis); err != nil {
		log.Fatalf("gagal menulis berkas pelengkap: %v", err)
	}

	log.Printf("selesai: %d halaman ditulis ke %s dalam %s", pages, *out, time.Since(start).Round(time.Millisecond))
}

func loadTemplates(a *site.Analysis) (*template.Template, error) {
	funcs := template.FuncMap{
		// --- format angka ------------------------------------------------
		"rp":        render.Rp,
		"rpShort":   render.RpShort,
		"num":       render.Num,
		"pct":       render.Pct,
		"pctPts":    render.PctPoints,
		"idx":       render.Index,
		"signed":    render.Signed,
		"signedN":   render.SignedNum,
		"ratio":     render.Ratio,
		"weeks":     render.Weeks,
		"date":      workcal.FormatDate,
		"dateShort": workcal.FormatDateShort,

		// --- bantu templat -----------------------------------------------
		"tx": func(t model.Text, lang string) string { return t.Get(lang) },
		"t":  i18n.T,
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"add":  func(a, b int) int { return a + b },
		"sub":  func(a, b int) int { return a - b },
		"mul":  func(a, b float64) float64 { return a * b },
		"subf": func(a, b float64) float64 { return a - b },
		"addf": func(a, b float64) float64 { return a + b },
		"f":    func(i int) float64 { return float64(i) },
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"neg": func(a float64) float64 { return -a },
		"abs": func(a float64) float64 {
			if a < 0 {
				return -a
			}
			return a
		},
		"lt":      func(a, b float64) bool { return a < b },
		"gt":      func(a, b float64) bool { return a > b },
		"pathFor": site.PathFor,
		"dict": func(kv ...interface{}) map[string]interface{} {
			m := map[string]interface{}{}
			for i := 0; i+1 < len(kv); i += 2 {
				m[fmt.Sprint(kv[i])] = kv[i+1]
			}
			return m
		},
		"has":     func(s, sub string) bool { return strings.Contains(s, sub) },
		"upper":   strings.ToUpper,
		"join":    strings.Join,
		"safeCSS": func(s string) template.CSS { return template.CSS(s) },

		// --- akses model --------------------------------------------------
		"activities":      func() []model.Activity { return model.Activities },
		"wbsPhases":       func() []model.WBSPhase { return model.WBSPhases },
		"risks":           func() []model.Risk { return model.Risks },
		"team":            func() []model.Member { return model.Team },
		"raci":            func() []model.RACIRow { return model.RACI },
		"raciRoles":       func() []model.Role { return model.RACIRoles },
		"stakeholders":    func() []model.Stakeholder { return model.Stakeholders },
		"commsPlan":       func() []model.CommsChannel { return model.CommsPlan },
		"lifecycle":       func() []model.LifecyclePhase { return model.Lifecycle },
		"knowledgeAreas":  func() []model.KnowledgeArea { return model.KnowledgeAreas },
		"defects":         func() []model.DefectRecord { return model.Defects },
		"coqItems":        func() []model.COQItem { return model.CostOfQuality },
		"fishbones":       func() []model.FishboneProblem { return model.Fishbones },
		"formulas":        func() []model.Formula { return model.Formulas },
		"formulaGroups":   func() interface{} { return model.FormulaGroups },
		"formulasIn":      model.FormulasByGroup,
		"coretaxFacts":    func() []model.CoretaxFact { return model.CoretaxFacts },
		"coretaxSources":  func() []model.Source { return model.CoretaxSources },
		"coretaxTimeline": func() []model.CoretaxMilestone { return model.CoretaxTimeline },
		"coretaxLessons":  func() []model.CoretaxLesson { return model.CoretaxLessons },
		"charterAlloc":    func() []model.BudgetLine { return model.CharterAllocation },
		"rateCard":        func() map[model.Role]float64 { return model.RateCard },
		"capacity":        func() map[model.Role]float64 { return model.Capacity },
		"holidays":        func() []workcal.Holiday { return workcal.Holidays },
		"totalBudget":     func() float64 { return model.TotalAuthorised },
		"contingency":     func() float64 { return model.ContingencyReserve },
		"activityByID":    model.ActivityByID,
		"packageName":     model.PackageName,
		"phaseOf":         model.PhaseOf,
		"sortedRoles":     render.SortedRoles,

		// --- grafik --------------------------------------------------------
		"sCurve": func(lang string) template.HTML {
			return render.SCurve(a.Curves, a.Calendar, lang, a.BAC)
		},
		"gantt": func(lang string) template.HTML {
			return render.Gantt(model.Activities, a.Plan, a.Calendar, a.StatusDay, lang)
		},
		"network": func(lang string) template.HTML {
			return render.Network(model.Activities, a.Plan, lang)
		},
		"simHistogram": func(lang string) template.HTML { return render.SimHistogram(a.Sim, lang) },
		"tornado": func(lang string, limit int) template.HTML {
			return render.Tornado(a.Sim.Sensitivity, limit, lang)
		},
		"riskMatrix": func(lang, caption string, residual bool) template.HTML {
			if residual {
				return render.RiskMatrix(a.Risk.ResidualMatrix, lang, caption)
			}
			return render.RiskMatrix(a.Risk.Matrix, lang, caption)
		},
		"controlChart": func(lang string) template.HTML {
			return render.ControlChartSVG(a.Control, model.ResponseSamples, lang)
		},
		"paretoChart": func(lang string) template.HTML { return render.ParetoSVG(a.Pareto, lang) },
		"budgetWaterfall": func(lang string) template.HTML {
			return render.BudgetWaterfall(a.BAC, model.ContingencyReserve, a.MgmtReserve, model.TotalAuthorised, lang)
		},
		"resourceHistogram": func(lang string) template.HTML {
			return render.ResourceHistogram(a.Resources, lang, a.Calendar)
		},
		"powerInterest": func(lang string) template.HTML {
			return render.PowerInterestGrid(model.Stakeholders, lang)
		},
		"metricBars": func(lang string) template.HTML { return render.MetricBars(a.Metrics, lang) },
		"coqBars":    func(lang string) template.HTML { return render.COQBars(a.COQ, lang) },
		"orgChart": func(lang string) template.HTML {
			return render.OrgChart(model.Team, model.ProjectCharter.Sponsor, lang)
		},
		"fishbone": func(p model.FishboneProblem, lang string) template.HTML {
			return render.Fishbone(p, lang)
		},
		"scenarioBars": func(lang string) template.HTML {
			var labels []string
			var extra, expected []float64
			for _, s := range a.Scenarios {
				labels = append(labels, s.Name.Get(lang))
				extra = append(extra, s.ExtraCost)
				expected = append(expected, s.ExpectedLoss)
			}
			return render.ScenarioBars(labels, extra, expected, lang)
		},

		// --- turunan analisis ---------------------------------------------
		"taskOf":        func(id string) interface{} { return a.Plan.Task(id) },
		"criticalPath":  func() []string { return a.Plan.CriticalPath },
		"isCritical":    func(id string) bool { return a.Plan.Task(id).Critical },
		"startDate":     func(id string) string { return a.Calendar.ISOAt(a.Plan.Task(id).StartX) },
		"finishDate":    func(id string) string { return a.Calendar.ISOAt(a.Plan.Task(id).FinishX - 1) },
		"dateAt":        func(d int) string { return a.Calendar.ISOAt(d) },
		"budgetOf":      func(act model.Activity) float64 { return act.Budget(model.RateCard) },
		"labourOf":      func(act model.Activity) float64 { return act.LabourCost(model.RateCard) },
		"emvOf":         func(r model.Risk) float64 { return r.EMV() },
		"residualEMVOf": func(r model.Risk) float64 { return r.ResidualEMV() },
		"probLevel":     func(p float64) int { return int(risk.ProbabilityLevel(p)) },
		"impactLevel":   func(i float64) int { return int(risk.ImpactLevel(i, a.BAC)) },
		"severityOf":    func(score int) string { return string(risk.SeverityOf(score)) },

		// --- studi kasus ---------------------------------------------------
		// breakEven menghitung pada peluang kegagalan berapa sebuah strategi
		// transisi mulai lebih murah daripada cutover serentak.
		"breakEven": func(extraShare, residualShare float64) float64 {
			return coretax.BreakEvenFailureProb(
				a.Coretax.TotalContract, model.CoretaxLostRevenueJan2025, extraShare, residualShare)
		},

		// --- PERT per aktivitas -------------------------------------------
		"pertTe": schedule.ActivityExpected,
		"pertSd": func(act model.Activity) float64 {
			return schedule.StdDev(float64(act.Optimistic), float64(act.Pessimistic))
		},
		"pertVar": func(act model.Activity) float64 {
			return schedule.Variance(float64(act.Optimistic), float64(act.Pessimistic))
		},
		// pertSkew membandingkan panjang ekor pesimistis terhadap ekor
		// optimistis. Di atas 1 berarti estimasinya condong ke arah lama.
		"pertSkew": func(act model.Activity) float64 {
			left := float64(act.Duration - act.Optimistic)
			if left <= 0 {
				return 0
			}
			return float64(act.Pessimistic-act.Duration) / left
		},
	}

	tmpl := template.New("base").Funcs(funcs)
	return tmpl.ParseGlob(filepath.Join("web", "templates", "*.gohtml"))
}

func copyStatic(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

// writeExtras menulis berkas pelengkap yang dibutuhkan situs produksi:
// sitemap, robots.txt, penanda .nojekyll untuk GitHub Pages, ekspor CSV, dan
// satu berkas JSON berisi seluruh metrik untuk dipakai ulang siapa pun.
func writeExtras(out, baseURL string, a *site.Analysis) error {
	if err := os.WriteFile(filepath.Join(out, ".nojekyll"), nil, 0o644); err != nil {
		return err
	}

	// Service worker harus berada di akar situs: sebuah worker hanya boleh
	// mengendalikan jalur pada atau di bawah direktorinya sendiri, sehingga
	// worker di /assets/js/ tidak akan pernah bisa melayani /piagam/.
	sw, err := os.ReadFile(filepath.Join("web", "static", "js", "sw.js"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "sw.js"), sw, 0o644); err != nil {
		return err
	}

	var sm strings.Builder
	sm.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sm.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">` + "\n")
	base := strings.TrimRight(baseURL, "/")
	today := time.Now().UTC().Format("2006-01-02")
	for _, lang := range i18n.Langs {
		for _, p := range site.Pages {
			sm.WriteString("  <url>\n")
			sm.WriteString("    <loc>" + base + site.PathFor(p.Route, lang) + "</loc>\n")
			for _, alt := range i18n.Langs {
				sm.WriteString(fmt.Sprintf("    <xhtml:link rel=\"alternate\" hreflang=\"%s\" href=\"%s\"/>\n",
					alt, base+site.PathFor(p.Route, alt)))
			}
			sm.WriteString("    <lastmod>" + today + "</lastmod>\n")
			sm.WriteString("  </url>\n")
		}
	}
	sm.WriteString("</urlset>\n")
	if err := os.WriteFile(filepath.Join(out, "sitemap.xml"), []byte(sm.String()), 0o644); err != nil {
		return err
	}

	robots := "User-agent: *\nAllow: /\n\nSitemap: " + base + "/sitemap.xml\n"
	if err := os.WriteFile(filepath.Join(out, "robots.txt"), []byte(robots), 0o644); err != nil {
		return err
	}

	dataDir := filepath.Join(out, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dataDir, "aktivitas.csv"), []byte(activitiesCSV(a)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dataDir, "risiko.csv"), []byte(risksCSV(a)), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, "metrik.json"), []byte(metricsJSON(a)), 0o644)
}

func activitiesCSV(a *site.Analysis) string {
	var sb strings.Builder
	sb.WriteString("id,wbs,nama,durasi,es,ef,ls,lf,total_float,free_float,kritis,anggaran,pv,ev,ac\n")
	rows := map[string]interface{}{}
	_ = rows
	byRow := map[string]int{}
	for i, r := range a.Rows {
		byRow[r.ID] = i
	}
	for _, act := range model.Activities {
		t := a.Plan.Task(act.ID)
		r := a.Rows[byRow[act.ID]]
		sb.WriteString(fmt.Sprintf("%s,%s,%q,%d,%d,%d,%d,%d,%d,%d,%t,%.0f,%.0f,%.0f,%.0f\n",
			act.ID, act.WBS, act.Name.ID, t.Duration, t.ES, t.EF, t.LS, t.LF,
			t.TotalFloat, t.FreeFloat, t.Critical, r.Budget, r.PV, r.EV, r.AC))
	}
	return sb.String()
}

func risksCSV(a *site.Analysis) string {
	var sb strings.Builder
	sb.WriteString("id,kategori,judul,peluang,dampak,emv,peluang_residual,dampak_residual,emv_residual,skor,keparahan,respons,pemilik,status\n")
	for _, r := range a.Risk.Risks {
		sb.WriteString(fmt.Sprintf("%s,%q,%q,%.2f,%.0f,%.0f,%.2f,%.0f,%.0f,%d,%s,%s,%s,%s\n",
			r.ID, r.Category.ID, r.Title.ID, r.Probability, r.Impact, r.EMV,
			r.ResidualProb, r.ResidualImpact, r.ResidualEMV, r.Score, r.Severity,
			r.Response, r.Owner, r.Status))
	}
	return sb.String()
}

func metricsJSON(a *site.Analysis) string {
	s := a.Snapshot
	return fmt.Sprintf(`{
  "proyek": %q,
  "tanggal_data": %q,
  "hari_kerja_berjalan": %.2f,
  "durasi_rencana_hari_kerja": %d,
  "tanggal_selesai_rencana": %q,
  "earned_value": {
    "pv": %.0f, "ev": %.0f, "ac": %.0f, "bac": %.0f,
    "sv": %.0f, "cv": %.0f, "spi": %.6f, "cpi": %.6f,
    "es": %.4f, "sv_waktu_hari": %.4f, "spi_waktu": %.6f,
    "eac_optimistis": %.0f, "eac_tipikal": %.0f, "eac_pesimistis": %.0f,
    "etc": %.0f, "vac": %.0f, "tcpi": %.6f
  },
  "anggaran": {
    "bac": %.0f, "cadangan_kontinjensi": %.0f, "cost_baseline": %.0f,
    "cadangan_manajemen": %.0f, "pagu_total": %.0f
  },
  "simulasi": {
    "iterasi": %d, "rerata": %.4f, "simpangan_baku": %.4f,
    "p50": %.0f, "p80": %.0f, "p90": %.0f, "peluang_tepat_waktu": %.6f
  },
  "risiko": {
    "emv_inheren": %.0f, "emv_residual": %.0f,
    "cakupan_cadangan": %.6f, "kekurangan_cadangan": %.0f,
    "paparan_jadwal_hari": %.4f
  },
  "mutu": {
    "terkendali": %t, "pelanggaran_aturan": %d, "cpk": %.4f,
    "coq_kesesuaian": %.0f, "coq_ketidaksesuaian": %.0f, "coq_rasio": %.4f
  }
}
`,
		model.ProjectCharter.Name.ID, a.StatusDate, s.AtDay, a.Plan.Duration, a.FinishDatePlan(),
		s.PV, s.EV, s.AC, s.BAC, s.SV, s.CV, s.SPI, s.CPI,
		s.ES, s.SVt, s.SPIt, s.EACOptimistic, s.EACTypical, s.EACPessimistic,
		s.ETC, s.VAC, s.TCPI,
		a.BAC, model.ContingencyReserve, a.Baseline, a.MgmtReserve, model.TotalAuthorised,
		a.Sim.Iterations, a.Sim.Mean, a.Sim.StdDev, a.Sim.P50, a.Sim.P80, a.Sim.P90, a.Sim.OnTimeProb,
		a.Risk.TotalEMV, a.Risk.TotalResidualEMV, a.Risk.ReserveCoverage, a.Risk.ReserveGap, a.Risk.ScheduleExposure,
		a.Control.InControl, len(a.Control.Violations), a.Control.Cpk,
		a.COQ.Conformance, a.COQ.Nonconformance, a.COQ.Ratio)
}
