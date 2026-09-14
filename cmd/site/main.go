// Command site membangun seluruh situs statis Control Tower MPPL.
//
// Keluarannya adalah direktori yang bisa dilayani apa adanya oleh GitHub Pages
// atau server berkas statis mana pun: tidak ada basis data, tidak ada proses
// yang harus tetap hidup, dan tidak ada langkah render di sisi klien untuk
// menampilkan isi halaman.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/coretax"
	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/risk"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
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
	// AssetVersion adalah hash isi web/static; ditempel pada URL aset agar
	// cache peramban tidak pernah memasangkan HTML baru dengan aset lama.
	AssetVersion string
	Content      template.HTML
	Analysis     *site.Analysis
	Examples     map[string]site.WorkedExample
	Charter      model.Charter
	Risk         risk.Register
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
	version, err := assetVersion(filepath.Join("web", "static"))
	if err != nil {
		log.Fatalf("gagal menghitung versi aset: %v", err)
	}

	pages := 0
	for _, lang := range i18n.Langs {
		examples := analysis.Examples(lang)
		for _, p := range site.Pages {
			data := PageData{
				Lang:         lang,
				OtherLang:    i18n.OtherLang(lang),
				Page:         p,
				Pages:        site.Pages,
				Canonical:    strings.TrimRight(*baseURL, "/") + site.PathFor(p.Route, lang),
				AltURL:       strings.TrimRight(*baseURL, "/") + site.PathFor(p.Route, i18n.OtherLang(lang)),
				BaseURL:      strings.TrimRight(*baseURL, "/"),
				Title:        p.NavLabel(lang) + " - " + i18n.T(lang, "site.name"),
				Desc:         p.Summary.Get(lang),
				BuildTime:    time.Now().UTC().Format("2006-01-02 15:04 MST"),
				AssetVersion: version,
				Analysis:     analysis,
				Examples:     examples,
				Charter:      model.ProjectCharter,
				Risk:         analysis.Risk,
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

		// --- grafik analisis lanjutan ----------------------------------------
		"ganttLevelled": func(lang string) template.HTML {
			return render.GanttLevelled(model.Activities, a.Plan, a.Level, a.Calendar, lang)
		},
		"levelHistogram": func(lang string) template.HTML {
			return render.ResourceHistogram(a.LevelProfile, lang, a.Calendar)
		},
		"crashCurve": func(lang string) template.HTML { return render.CrashCurve(a.Crash, lang) },
		"ladderChart": func(metric, lang string) template.HTML {
			target := float64(a.Plan.Duration)
			if metric == "cost" {
				target = model.TotalAuthorised
			}
			return render.LadderChart(a.Ladder, site.LadderNames, metric, target, lang)
		},
		"jclHeatmap": func(lang string) template.HTML {
			fin := a.Final()
			return render.JCLHeatmap(a.Density, a.Frontier, float64(a.Plan.Duration), model.TotalAuthorised, fin.DurP80, fin.CostP80, lang)
		},
		"costHistogram": func(lang string) template.HTML {
			fin := a.Final()
			return render.ValueHistogram(fin.Costs, 32, []render.Marker{
				{Value: model.TotalAuthorised, Label: map[bool]string{true: "pagu", false: "cap"}[lang == "id"], Class: "budget"},
				{Value: fin.CostP50, Label: "P50", Class: "p50"},
				{Value: fin.CostP80, Label: "P80", Class: "p80"},
			}, true, lang)
		},
		"ladderNames":   func() []model.Text { return site.LadderNames },
		"tradeOffChart": func(lang string) template.HTML { return render.TradeOffChart(a.Exact, lang) },
		"boundChart":    func(lang string) template.HTML { return render.BoundChart(a.LevelOpt, lang) },
		"riskCountBars": func(lang string) template.HTML {
			if len(a.RiskSweep) == 0 {
				return ""
			}
			def := a.RiskSweep[0]
			for _, p := range a.RiskSweep {
				if p.Loading == model.RiskLoading {
					def = p
				}
			}
			return render.CountBars(a.RiskSweep[0].Count, def.Count,
				map[bool]string{true: "saling bebas (lambda 0)", false: "independent (lambda 0)"}[lang == "id"],
				map[bool]string{true: "bergerombol (lambda " + render.Num(model.RiskLoading, 1, lang) + ")", false: "clustered (lambda " + render.Num(model.RiskLoading, 1, lang) + ")"}[lang == "id"],
				8, lang)
		},
		"forecastHistogram": func(lang string) template.HTML {
			fc := a.Forecast
			return render.ValueHistogram(fc.Durations, 30, []render.Marker{
				{Value: float64(a.Plan.Duration), Label: map[bool]string{true: "rencana", false: "plan"}[lang == "id"], Class: "plan"},
				{Value: a.IEACt, Label: "IEAC(t)", Class: "ieac"},
				{Value: fc.DurP50, Label: "P50", Class: "p50"},
				{Value: fc.DurP80, Label: "P80", Class: "p80"},
				{Value: a.Final().DurP80, Label: map[bool]string{true: "P80 perencanaan", false: "planning P80"}[lang == "id"], Class: "plan-p80"},
			}, false, lang)
		},
		"forecastFrontierRows": func(step int) []simulate.FrontierPoint {
			return thinFrontier(a.ForecastFrontier, step)
		},
		"exposedActivity": func(id string) string {
			if a.InFlight == nil {
				return ""
			}
			if i, ok := a.InFlight.Exposed[id]; ok && i < len(model.Activities) {
				return model.Activities[i].ID
			}
			return ""
		},
		"openRiskCount": func() int {
			n := 0
			for _, r := range model.Risks {
				if a.InFlight == nil || !a.InFlight.ClosedRisk[r.ID] {
					n++
				}
			}
			return n
		},
		"riskDrivers":  func() []model.RiskDriver { return model.RiskDrivers },
		"reworkLoops":  func() []model.ReworkLoop { return model.ReworkLoops },
		"riskLoading":  func() float64 { return model.RiskLoading },
		"examFactor":   func() float64 { return model.ExamCapacityFactor },
		"priorWeight":  func() float64 { return simulate.DefaultPriorWeight },
		"minStartRate": func() float64 { return level.DefaultMinStartRate },
		"calendarURL":  func() string { return model.AcademicCalendarURL },
		"skbURL":       func() string { return workcal.SKB2026URL },
		"skb2025URL":   func() string { return workcal.SKB2025URL },
		"manyRisks":    func() int { return site.ManyRiskThreshold },
		"driverOf":     func(r model.Risk) string { return r.DriverOf() },
		"driverLabel": func(key, lang string) string {
			if d, ok := model.DriverByKey(key); ok {
				return d.Label.Get(lang)
			}
			return ""
		},
		"risksOfDriver": func(key string) []string {
			var out []string
			for _, r := range model.Risks {
				if r.DriverOf() == key {
					out = append(out, r.ID)
				}
			}
			return out
		},
		"minf": func(x, y float64) float64 {
			if x < y {
				return x
			}
			return y
		},
		"netCrash": func(days int) float64 { return a.TradeOffNet(days) },
		"exactAt": func(d int) compress.ExactPoint {
			p, _ := a.Exact.PointAt(d)
			return p
		},
		"cutsText": func(m map[string]int) string {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var parts []string
			for _, k := range keys {
				if m[k] > 1 {
					parts = append(parts, fmt.Sprintf("%s×%d", k, m[k]))
				} else {
					parts = append(parts, k)
				}
			}
			return strings.Join(parts, ", ")
		},
		"ruleName":     func(r level.Rule, lang string) string { return ruleName(r, lang) },
		"availability": func() []model.AvailabilityWindow { return model.AvailabilityWindows },
		"crashPlanOf":  func(act model.Activity) model.CrashPlan { return act.Crash(model.RateCard) },
		"crashPremium": func() float64 { return model.CrashPremium },
		"reworkProb":   func() float64 { return compress.ReworkProbability },
		"defaultRho":   func() float64 { return simulate.DefaultRho },
		// frontierRows menipiskan tabel frontier, tetapi selalu dimulai dari
		// titik layak PERTAMA - itulah komitmen JCL 70% dengan tenggat
		// terpendek, dan tidak boleh sampai terlewat oleh penipisan.
		"frontierRows": func(step int) []simulate.FrontierPoint {
			return thinFrontier(a.Frontier, step)
		},
		"riskByID": func(id string) model.Risk {
			for _, r := range model.Risks {
				if r.ID == id {
					return r
				}
			}
			return model.Risk{}
		},
		"iterf": func(n int) float64 { return float64(n) },
		"int":   func(v float64) int { return int(v) },
		"sumDur": func(ids ...string) int {
			byID := model.ActivityByID()
			total := 0
			for _, id := range ids {
				total += byID[id].Duration
			}
			return total
		},
		"levelTask": func(id string) interface{} { return a.Level.Tasks[id] },
		"sdOf":      simulate.StdDev,

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

// assetVersion menghitung sidik sepuluh heksadesimal atas nama dan isi seluruh
// berkas di bawah root, dalam urutan jalur yang tetap. Berkas WebAssembly ikut
// dihitung bila sudah dibangun, sehingga mesin hitung yang berubah pasti
// mengganti versi.
func assetVersion(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		rel, _ := filepath.Rel(root, p)
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		f, err := os.Open(p)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:10], nil
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
	version, err := assetVersion(filepath.Join("web", "static"))
	if err != nil {
		return err
	}
	sw = []byte(strings.ReplaceAll(string(sw), "__ASSET_VERSION__", version))
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
	sb.WriteString("id,wbs,nama,durasi,es,ef,ls,lf,total_float,free_float,kritis,mulai_levelling,selesai_levelling,geser_levelling,anggaran,pv,ev,ac\n")
	rows := map[string]interface{}{}
	_ = rows
	byRow := map[string]int{}
	for i, r := range a.Rows {
		byRow[r.ID] = i
	}
	for _, act := range model.Activities {
		t := a.Plan.Task(act.ID)
		r := a.Rows[byRow[act.ID]]
		lt := a.Level.Tasks[act.ID]
		sb.WriteString(fmt.Sprintf("%s,%s,%q,%d,%d,%d,%d,%d,%d,%d,%t,%d,%d,%d,%.0f,%.0f,%.0f,%.0f\n",
			act.ID, act.WBS, act.Name.ID, t.Duration, t.ES, t.EF, t.LS, t.LF,
			t.TotalFloat, t.FreeFloat, t.Critical, lt.Start, lt.Finish-1, lt.Delay(),
			r.Budget, r.PV, r.EV, r.AC))
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

// metricsJSON menulis seluruh metrik utama sebagai JSON terstruktur. Urutan
// kunci mengikuti urutan field struct, sehingga berkasnya stabil antar-build.
func metricsJSON(a *site.Analysis) string {
	s := a.Snapshot
	type ev struct {
		PV     float64 `json:"pv"`
		EV     float64 `json:"ev"`
		AC     float64 `json:"ac"`
		BAC    float64 `json:"bac"`
		SV     float64 `json:"sv"`
		CV     float64 `json:"cv"`
		SPI    float64 `json:"spi"`
		CPI    float64 `json:"cpi"`
		ES     float64 `json:"es"`
		SVt    float64 `json:"sv_waktu_hari"`
		SPIt   float64 `json:"spi_waktu"`
		EACOpt float64 `json:"eac_optimistis"`
		EACTyp float64 `json:"eac_tipikal"`
		EACPes float64 `json:"eac_pesimistis"`
		ETC    float64 `json:"etc"`
		VAC    float64 `json:"vac"`
		TCPI   float64 `json:"tcpi"`
		IEACt  float64 `json:"ieac_t_hari"`
	}
	type layer struct {
		Name       string    `json:"lapisan"`
		DurP50     float64   `json:"p50_durasi"`
		DurP80     float64   `json:"p80_durasi"`
		DurP95     float64   `json:"p95_durasi"`
		CostP80    float64   `json:"p80_biaya"`
		CostP95    float64   `json:"p95_biaya"`
		CostMean   float64   `json:"rerata_biaya"`
		JCL        float64   `json:"jcl_target_piagam"`
		JointAtP80 float64   `json:"peluang_bersama_di_p80"`
		RiskPhi    float64   `json:"phi_risiko_terealisasi"`
		Rework     float64   `json:"rerata_hari_rework"`
		TimeCost   float64   `json:"rerata_biaya_sewa"`
		RiskCount  []float64 `json:"sebaran_jumlah_risiko,omitempty"`
	}
	var ladder []layer
	for i, r := range a.Ladder {
		ladder = append(ladder, layer{
			Name: site.LadderNames[i].ID, DurP50: r.DurP50, DurP80: r.DurP80, DurP95: r.DurP95,
			CostP80: r.CostP80, CostP95: r.CostP95, CostMean: r.CostMean, JCL: r.JCL, JointAtP80: r.JointAtP80,
			RiskPhi: r.RiskPhi, Rework: r.ReworkDays, TimeCost: r.TimeCost,
		})
	}
	type rule struct {
		Rule      level.Rule `json:"aturan"`
		Duration  int        `json:"durasi"`
		Justified int        `json:"setelah_justifikasi"`
	}
	var rules []rule
	for _, r := range a.LevelOpt.Rules {
		rules = append(rules, rule{r.Rule, r.Duration, r.Justified})
	}
	type exact struct {
		Duration  int     `json:"durasi"`
		CrashCost float64 `json:"biaya_crash_eksak"`
		Greedy    float64 `json:"biaya_crash_serakah"`
		Rental    float64 `json:"biaya_sewa"`
		Total     float64 `json:"biaya_total"`
	}
	var curve []exact
	for _, p := range a.Exact.Points {
		curve = append(curve, exact{p.Duration, p.CrashCost, p.Greedy, p.Rental, p.Total})
	}
	type gertRow struct {
		ID       string  `json:"id"`
		Check    string  `json:"pemeriksaan"`
		P        float64 `json:"peluang_gagal"`
		Rework   float64 `json:"hari_per_putaran"`
		Cycles   float64 `json:"rerata_putaran_analitik"`
		MC       float64 `json:"rerata_putaran_simulasi"`
		MeanAdd  float64 `json:"rerata_hari_tambahan"`
		SDAdd    float64 `json:"simpangan_baku_hari_tambahan"`
		AtLeast2 float64 `json:"peluang_dua_putaran_atau_lebih"`
	}
	var gerts []gertRow
	for _, g := range a.GERT {
		gerts = append(gerts, gertRow{g.Loop.ID, g.Loop.Check, g.Loop.FailProb, g.ReworkDays, g.Cycles, g.MCCycles, g.Reduced.Mean(), g.Reduced.SD(), g.AtLeastTwo})
	}
	fin := a.Final()
	fl := a.InFlight
	fc := a.Forecast
	ordered := []struct {
		key string
		val interface{}
	}{
		{"proyek", model.ProjectCharter.Name.ID},
		{"tanggal_data", a.StatusDate},
		{"hari_kerja_berjalan", s.AtDay},
		{"durasi_rencana_hari_kerja", a.Plan.Duration},
		{"tanggal_selesai_rencana", a.FinishDatePlan()},
		{"earned_value", ev{s.PV, s.EV, s.AC, s.BAC, s.SV, s.CV, s.SPI, s.CPI, s.ES, s.SVt, s.SPIt, s.EACOptimistic, s.EACTypical, s.EACPessimistic, s.ETC, s.VAC, s.TCPI, a.IEACt}},
		{"anggaran", map[string]float64{"bac": a.BAC, "cadangan_kontinjensi": model.ContingencyReserve, "cost_baseline": a.Baseline, "cadangan_manajemen": a.MgmtReserve, "pagu_total": model.TotalAuthorised, "tarif_sewa_harian": a.RentalDaily()}},
		{"simulasi", map[string]interface{}{"iterasi": a.Sim.Iterations, "rerata": a.Sim.Mean, "simpangan_baku": a.Sim.StdDev, "p50": a.Sim.P50, "p80": a.Sim.P80, "p90": a.Sim.P90, "peluang_tepat_waktu": a.Sim.OnTimeProb}},
		{"risiko", map[string]interface{}{"emv_inheren": a.Risk.TotalEMV, "emv_residual": a.Risk.TotalResidualEMV, "cakupan_cadangan": a.Risk.ReserveCoverage, "kekurangan_cadangan": a.Risk.ReserveGap, "paparan_jadwal_hari": a.Risk.ScheduleExposure}},
		{"mutu", map[string]interface{}{"terkendali": a.Control.InControl, "pelanggaran_aturan": len(a.Control.Violations), "cpk": a.Control.Cpk, "coq_kesesuaian": a.COQ.Conformance, "coq_ketidaksesuaian": a.COQ.Nonconformance, "coq_rasio": a.COQ.Ratio}},
		{"levelling", map[string]interface{}{
			"durasi_cpm": a.LevelWhy.CPM, "durasi_kapasitas_saja": a.LevelWhy.CapacityOnly, "durasi_dengan_ujian": a.LevelWhy.WithWindows,
			"tanggal_selesai": a.FinishISO(a.Level.Duration), "peran_kritis": string(a.CriticalRole),
			"laju_mulai_minimum": level.DefaultMinStartRate,
			"aturan_prioritas":   rules, "sampel_acak": a.LevelOpt.Samples, "sumber_jadwal_terbaik": a.LevelOpt.Source,
			"durasi_sgs_lst":    a.LevelOpt.Baseline.Duration,
			"batas_bawah":       map[string]interface{}{"cpm": a.LevelOpt.Bound.CPM, "solo": a.LevelOpt.Bound.Solo, "energetik": a.LevelOpt.Bound.Energetic, "akhir": a.LevelOpt.Bound.Value},
			"celah_optimalitas": a.LevelOpt.Gap, "terbukti_optimal": a.LevelOpt.Proven,
			"audit_simulasi": map[string]interface{}{"iterasi_teraudit": fin.Audit.Audited, "rerata_selisih_sgs": fin.Audit.SGSGapMean, "maks_selisih_sgs": fin.Audit.SGSGapMax, "porsi_terbukti_optimal": fin.Audit.ProvenShare, "porsi_sgs_sudah_optimal": fin.Audit.SGSOptimalShare},
		}},
		{"kompresi", map[string]interface{}{
			"durasi_minimum_crash": a.Crash.MinDuration, "biaya_crash_5_hari": crashCost(a, 5), "biaya_crash_penuh": crashCost(a, len(a.Crash.Steps)),
			"serakah_optimal": a.Exact.GreedyOptimal, "kelebihan_serakah_maks": a.Exact.MaxGreedyExcess,
			"biaya_bersih_5_hari": a.TradeOffNet(5), "durasi_biaya_total_terendah": a.Exact.Optimum.Duration,
			"kurva_eksak":               curve,
			"kandidat_fast_track_layak": len(a.FastViable), "durasi_semua_fast_track": a.FastAllDur,
		}},
		{"simulasi_terpadu", map[string]interface{}{
			"rho": simulate.DefaultRho, "lambda_risiko": model.RiskLoading, "iterasi": fin.Config.Iterations,
			"lapisan": ladder, "jcl70_tenggat": a.JCL70.Duration, "jcl70_anggaran": a.JCL70.Budget,
			"putaran_gert": gerts,
		}},
	}
	if fl != nil {
		ordered = append(ordered, struct {
			key string
			val interface{}
		}{"prakiraan_berjalan", map[string]interface{}{
			"selesai": fl.Completed, "berjalan": fl.InProgress, "belum_mulai": len(fl.NotStarted),
			"kredibilitas_z": fl.Credibility, "rasio_durasi_teramati": fl.ObservedRatio, "faktor_durasi": fl.DurationFactor,
			"rasio_biaya_teramati": fl.ObservedCostRatio, "faktor_biaya": fl.CostFactor,
			"faktor_ujian_teramati": fl.ExamObserved, "faktor_ujian_terkalibrasi": fl.ExamFactor,
			"ac_sampai_tanggal_data": fl.ACToDate,
			"p50_durasi":             fc.DurP50, "p80_durasi": fc.DurP80, "p80_biaya": fc.CostP80, "jcl_target_piagam": fc.JCL,
			"p80_durasi_tanpa_belajar": a.ForecastPrior.DurP80,
			"jcl70_tenggat":            a.ForecastJCL70.Duration, "jcl70_anggaran": a.ForecastJCL70.Budget,
		}})
	}
	// encoding/json mengurutkan kunci map; urutan dokumen dijaga dengan
	// menulis objek terluar secara manual.
	var sb strings.Builder
	sb.WriteString("{\n")
	for i, kv := range ordered {
		b, err := json.MarshalIndent(kv.val, "  ", "  ")
		if err != nil {
			b = []byte("null")
		}
		k, _ := json.Marshal(kv.key)
		sb.WriteString("  " + string(k) + ": " + string(b))
		if i < len(ordered)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}

// thinFrontier menipiskan tabel frontier, tetapi selalu dimulai dari titik
// layak PERTAMA - itulah komitmen JCL 70% dengan tenggat terpendek, dan tidak
// boleh sampai terlewat oleh penipisan.
func thinFrontier(fr []simulate.FrontierPoint, step int) []simulate.FrontierPoint {
	if step < 1 {
		step = 1
	}
	var out []simulate.FrontierPoint
	k := 0
	for _, p := range fr {
		if !p.Feasible {
			continue
		}
		if k%step == 0 {
			out = append(out, p)
		}
		k++
	}
	return out
}

// ruleName memberi nama aturan prioritas yang bisa dibaca manusia.
func ruleName(r level.Rule, lang string) string {
	names := map[level.Rule][2]string{
		level.RuleLST:  {"latest start terkecil", "smallest latest start"},
		level.RuleLFT:  {"latest finish terkecil", "smallest latest finish"},
		level.RuleMSLK: {"total float terkecil", "smallest total float"},
		level.RuleGRPW: {"bobot posisi terbesar", "greatest positional weight"},
		level.RuleMTS:  {"penerus terbanyak", "most total successors"},
		level.RuleSPT:  {"durasi terpendek", "shortest duration"},
	}
	n, ok := names[r]
	if !ok {
		return string(r)
	}
	if lang == "en" {
		return n[1]
	}
	return n[0]
}

func crashCost(a *site.Analysis, days int) float64 {
	v, _ := a.Crash.CostToSave(days)
	return v
}
