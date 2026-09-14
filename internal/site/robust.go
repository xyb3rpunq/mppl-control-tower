package site

import (
	"math"
	"sort"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// Keputusan percepatan bergantung pada satu angka yang tidak ada di data:
// berapa nilai satu hari lebih cepat bagi sponsor. Pita nilai menjawabnya
// tanpa angka itu - untuk setiap rentang nilai, opsi mana yang memberi manfaat
// bersih terbesar. Lalu dua pertanyaan kejujuran: apakah jawabannya bertahan
// terhadap derau Monte Carlo (bootstrap berpasangan), dan terhadap asumsi yang
// belum punya data (skenario rho, lambda risiko, dan peluang gagal GERT)?

// ValueBand adalah rentang nilai satu hari lebih cepat (rupiah per hari) tempat
// satu opsi memberi manfaat bersih terbesar: nilai x hari lebih cepat -
// tambahan anggaran pada titik JCL 70%.
type ValueBand struct {
	Option int     // indeks di Decision.Options
	From   float64 // inklusif
	To     float64 // eksklusif; +Inf untuk rentang terakhir
}

// Open melaporkan apakah rentang tidak punya batas atas.
func (b ValueBand) Open() bool { return math.IsInf(b.To, 1) }

// valueBands menghitung selubung atas garis manfaat bersih v*days[i] -
// extra[i] untuk v >= 0 atas opsi yang eligible. Seri dipecah ke anggaran
// tambahan yang lebih kecil, lalu indeks yang lebih kecil.
func valueBands(days, extra []float64, eligible []bool) []ValueBand {
	var idx []int
	for i := range days {
		if eligible[i] {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return nil
	}
	cuts := []float64{0}
	for _, i := range idx {
		for _, j := range idx {
			if days[j] > days[i] {
				if v := (extra[j] - extra[i]) / (days[j] - days[i]); v > 0 {
					cuts = append(cuts, v)
				}
			}
		}
	}
	sort.Float64s(cuts)
	best := func(v float64) int {
		b := idx[0]
		for _, i := range idx[1:] {
			ni, nb := v*days[i]-extra[i], v*days[b]-extra[b]
			if ni > nb+1e-9 || (math.Abs(ni-nb) <= 1e-9 && extra[i] < extra[b]-1e-9) {
				b = i
			}
		}
		return b
	}
	var out []ValueBand
	for k, from := range cuts {
		if k > 0 && from-cuts[k-1] < 1e-9 {
			continue
		}
		probe := from + 1
		to := math.Inf(1)
		for _, c := range cuts[k+1:] {
			if c-from >= 1e-9 {
				to = c
				probe = (from + to) / 2
				break
			}
		}
		o := best(probe)
		if n := len(out); n > 0 && out[n-1].Option == o {
			out[n-1].To = to
			continue
		}
		out = append(out, ValueBand{Option: o, From: from, To: to})
	}
	return out
}

// pricing menurunkan hari lebih cepat, tambahan anggaran, harga per hari, dan
// opsi termurah dari titik JCL 70% setiap opsi (indeks 0 = tanpa percepatan).
func pricing(pts []simulate.FrontierPoint, assumed []bool) (days, extra, price []float64, cheapest int) {
	n := len(pts)
	days, extra, price = make([]float64, n), make([]float64, n), make([]float64, n)
	cheapest = -1
	if n == 0 || !pts[0].Feasible {
		return
	}
	for i := 1; i < n; i++ {
		if !pts[i].Feasible {
			continue
		}
		days[i] = pts[0].Duration - pts[i].Duration
		extra[i] = pts[i].Budget - pts[0].Budget
		if days[i] > 0 {
			price[i] = extra[i] / days[i]
			if !assumed[i] && (cheapest < 0 || price[i] < price[cheapest]) {
				cheapest = i
			}
		}
	}
	return
}

// bandOptions mengembalikan urutan opsi pada pita nilai, untuk membandingkan
// rekomendasi antar-ulangan dan antar-skenario.
func bandOptions(bands []ValueBand) []int {
	out := make([]int, len(bands))
	for i, b := range bands {
		out[i] = b.Option
	}
	return out
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Robustness adalah sebaran angka keputusan menurut bootstrap berpasangan.
type Robustness struct {
	Reps int
	// Interval 90% per opsi (sejajar Decision.Options).
	DurLo, DurHi, BudgetLo, BudgetHi, PriceLo, PriceHi []float64
	// CheapestSame dan BandsSame adalah porsi ulangan dengan opsi termurah per
	// hari dan urutan opsi pada pita nilai yang sama dengan angka titik.
	CheapestSame, BandsSame float64
	// EdgeLo dan EdgeHi adalah interval 90% setiap batas antar-pita, dihitung
	// dari ulangan yang urutan pitanya sama.
	EdgeLo, EdgeHi []float64
}

// BootstrapReps adalah jumlah ulangan bootstrap berpasangan keputusan.
const BootstrapReps = 200

// robustness menjalankan bootstrap berpasangan atas seluruh opsi.
func robustness(d *Decision) (*Robustness, error) {
	results := make([]simulate.IntegratedResult, len(d.Options))
	assumed := make([]bool, len(d.Options))
	for i, o := range d.Options {
		results[i] = o.Sim
		assumed[i] = o.Assumption.ID != ""
	}
	reps, err := simulate.PairedBootstrap(results, 0.7, BootstrapReps, 20210801)
	if err != nil || reps == nil {
		return nil, err
	}
	n := len(d.Options)
	r := &Robustness{Reps: len(reps)}
	durs, budgets, prices := make([][]float64, n), make([][]float64, n), make([][]float64, n)
	pointBands := bandOptions(d.Bands)
	edges := make([][]float64, max(0, len(d.Bands)-1))
	var cheapSame, bandSame int
	eligible := make([]bool, n)
	for i := range eligible {
		eligible[i] = !assumed[i]
	}
	for _, row := range reps {
		days, extra, price, cheapest := pricing(row, assumed)
		for i, p := range row {
			durs[i] = append(durs[i], p.Duration)
			budgets[i] = append(budgets[i], p.Budget)
			if days[i] > 0 {
				prices[i] = append(prices[i], price[i])
			}
		}
		if cheapest == d.Cheapest {
			cheapSame++
		}
		bands := valueBands(days, extra, eligible)
		if sameInts(bandOptions(bands), pointBands) {
			bandSame++
			for k := range edges {
				edges[k] = append(edges[k], bands[k].To)
			}
		}
	}
	q := func(xs []float64, p float64) float64 {
		if len(xs) == 0 {
			return 0
		}
		s := append([]float64(nil), xs...)
		sort.Float64s(s)
		return simulate.Quantile(s, p)
	}
	for i := 0; i < n; i++ {
		r.DurLo, r.DurHi = append(r.DurLo, q(durs[i], 0.05)), append(r.DurHi, q(durs[i], 0.95))
		r.BudgetLo, r.BudgetHi = append(r.BudgetLo, q(budgets[i], 0.05)), append(r.BudgetHi, q(budgets[i], 0.95))
		r.PriceLo, r.PriceHi = append(r.PriceLo, q(prices[i], 0.05)), append(r.PriceHi, q(prices[i], 0.95))
	}
	for _, e := range edges {
		r.EdgeLo, r.EdgeHi = append(r.EdgeLo, q(e, 0.05)), append(r.EdgeHi, q(e, 0.95))
	}
	r.CheapestSame = float64(cheapSame) / float64(len(reps))
	r.BandsSame = float64(bandSame) / float64(len(reps))
	return r, nil
}

// Scenario adalah keputusan yang dihitung ulang dengan satu asumsi diganti.
type Scenario struct {
	Key  string
	Name model.Text
	// Config mengubah konfigurasi simulasi dasar.
	Config func(*simulate.IntegratedConfig) `json:"-"`
	// Sims dan JCL70 sejajar dengan Decision.ScenarioOptions.
	Sims  []simulate.IntegratedResult
	JCL70 []simulate.FrontierPoint
	// Hasil turunan, dengan indeks opsi di Decision.Options.
	Cheapest int
	Price    float64
	Bands    []ValueBand
	Same     bool // opsi termurah dan urutan pita sama dengan keputusan dasar
}

// ScenarioIterations adalah iterasi setiap simulasi skenario. Skenario memakai
// levelling cepat tanpa pembuktian per iterasi supaya build tetap wajar, dan
// dibandingkan dengan skenario pertama - asumsi dasar dengan cara yang sama -
// sehingga perubahan rekomendasi hanya bisa lahir dari asumsinya, bukan dari
// cara hitung atau jumlah iterasi.
const ScenarioIterations = 3000

// scenarios adalah asumsi tanpa data yang diuji: nilai rendah dan tinggi dari
// rentang yang dipakai halaman kepekaan masing-masing. Skenario pertama adalah
// pembanding dengan asumsi dasar.
func scenarios() []Scenario {
	return []Scenario{
		{Key: "dasar", Name: model.Text{ID: "Asumsi dasar (cara skenario)", EN: "Base assumptions (scenario method)"}, Config: func(c *simulate.IntegratedConfig) {}},
		{Key: "rho-rendah", Name: model.Text{ID: "Korelasi peran ρ 0,25", EN: "Role correlation ρ 0.25"}, Config: func(c *simulate.IntegratedConfig) { c.Rho = 0.25 }},
		{Key: "rho-tinggi", Name: model.Text{ID: "Korelasi peran ρ 0,75", EN: "Role correlation ρ 0.75"}, Config: func(c *simulate.IntegratedConfig) { c.Rho = 0.75 }},
		{Key: "lambda-rendah", Name: model.Text{ID: "Risiko bergerombol λ 0,3", EN: "Clustered risk λ 0.3"}, Config: func(c *simulate.IntegratedConfig) { c.RiskLoading = 0.3 }},
		{Key: "lambda-tinggi", Name: model.Text{ID: "Risiko bergerombol λ 0,9", EN: "Clustered risk λ 0.9"}, Config: func(c *simulate.IntegratedConfig) { c.RiskLoading = 0.9 }},
		{Key: "gert-rendah", Name: model.Text{ID: "Peluang gagal GERT x 0,5", EN: "GERT failure chance x 0.5"}, Config: func(c *simulate.IntegratedConfig) { c.ReworkScale = 0.5 }},
		{Key: "gert-tinggi", Name: model.Text{ID: "Peluang gagal GERT x 1,5", EN: "GERT failure chance x 1.5"}, Config: func(c *simulate.IntegratedConfig) { c.ReworkScale = 1.5 }},
	}
}

// finishScenario menurunkan titik JCL 70%, opsi termurah, dan pita nilai, lalu
// membandingkannya dengan acuan: keputusan utama untuk skenario pembanding, dan
// skenario pembanding untuk skenario lain.
func finishScenario(d *Decision, s *Scenario, refCheapest int, refBands []ValueBand) {
	pts := make([]simulate.FrontierPoint, len(d.Options))
	assumed := make([]bool, len(d.Options))
	eligible := make([]bool, len(d.Options))
	for i := range d.Options {
		assumed[i] = true // opsi yang tidak dijalankan pada skenario diabaikan
	}
	s.JCL70 = make([]simulate.FrontierPoint, len(s.Sims))
	for k, oi := range d.ScenarioOptions {
		s.JCL70[k] = jcl70(s.Sims[k])
		pts[oi] = s.JCL70[k]
		assumed[oi] = false
		eligible[oi] = true
	}
	days, extra, price, cheapest := pricing(pts, assumed)
	s.Cheapest = cheapest
	if cheapest >= 0 {
		s.Price = price[cheapest]
	}
	s.Bands = valueBands(days, extra, eligible)
	s.Same = cheapest == refCheapest && sameInts(bandOptions(s.Bands), bandOptions(refBands))
}

// BandRuleID dan BandRuleEN menuliskan pita nilai sebagai aturan keputusan.
func (d *Decision) BandRuleID() string { return d.bandRule("id") }

func (d *Decision) BandRuleEN() string { return d.bandRule("en") }

func (d *Decision) bandRule(lang string) string {
	if len(d.Bands) == 0 {
		return ""
	}
	rp := func(v float64) string { return render.Rp(v, lang) }
	var parts []string
	for _, b := range d.Bands {
		o := d.Options[b.Option]
		var rangeText string
		switch {
		case b.From == 0 && b.Open():
			rangeText = map[string]string{"id": "berapa pun nilainya", "en": "whatever its value"}[lang]
		case b.From == 0:
			rangeText = map[string]string{"id": "di bawah ", "en": "below "}[lang] + rp(b.To)
		case b.Open():
			rangeText = rp(b.From) + map[string]string{"id": " ke atas", "en": " and above"}[lang]
		default:
			rangeText = rp(b.From) + map[string]string{"id": " sampai ", "en": " to "}[lang] + rp(b.To)
		}
		what := lowerFirst(o.Name.Get(lang))
		if b.Option > 0 {
			what += " (" + render.Num(o.JCL70.Duration, 0, lang) + map[string]string{"id": " hari kerja, ", "en": " working days, "}[lang] + rp(o.JCL70.Budget) + ")"
		}
		parts = append(parts, rangeText+": "+what)
	}
	head := map[string]string{"id": "Menurut nilai satu hari lebih cepat bagi sponsor - ", "en": "By what a day earlier is worth to the sponsor - "}[lang]
	out := head + strings.Join(parts, "; ") + "."
	if never := d.NeverBest(); len(never) > 0 {
		var names []string
		for _, o := range never {
			names = append(names, lowerFirst(o.Name.Get(lang)))
		}
		out += map[string]string{"id": " Opsi yang tidak pernah terbaik: ", "en": " Never the best option: "}[lang] + strings.Join(names, ", ") + "."
	}
	if r := d.Robust; r != nil {
		out += map[string]string{"id": " Urutan ini bertahan pada ", "en": " This order holds in "}[lang] + render.Pct(r.BandsSame, 1, lang) +
			map[string]string{"id": " ulangan bootstrap dan ", "en": " of bootstrap resamples and "}[lang] +
			render.Num(float64(d.ScenariosSame()), 0, lang) + map[string]string{"id": " dari ", "en": " of "}[lang] + render.Num(float64(len(d.AssumptionScenarios())), 0, lang) +
			map[string]string{"id": " skenario asumsi.", "en": " assumption scenarios."}[lang]
	}
	return out
}

// Iterations mengembalikan jumlah iterasi simulasi skenario.
func (s Scenario) Iterations() int {
	if len(s.Sims) == 0 {
		return 0
	}
	return s.Sims[0].Config.Iterations
}
