package main

import (
	"fmt"
	"html"
	"html/template"
	"math"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
)

// TestDetailChartsReachTheirPages memastikan setiap grafik rincian dirender
// di halamannya dalam kedua bahasa.
func TestDetailChartsReachTheirPages(t *testing.T) {
	want := map[string][]string{
		"/simulasi-terpadu/": {"sweep-panels", "loop-bars", "column-bars", "dumbbell"},
		"/prakiraan/":        {"dumbbell"},
		"/pert/":             {"range-rows"},
		"/jadwal/":           {"stack-rows"},
		"/optimasi/":         {"stack-rows", "scatter"},
		"/biaya/":            {"diverging"},
		"/piagam/":           {"stack-rows"},
		"/coretax/":          {"event-timeline", "column-bars"},
	}
	pages := renderAll(t)
	for _, lang := range i18n.Langs {
		for route, classes := range want {
			for _, c := range classes {
				if !hasChart(pages[lang+" "+route], c) {
					t.Errorf("%s %s: grafik %q tidak dirender", lang, route, c)
				}
			}
		}
		if n := strings.Count(pages[lang+" /simulasi-terpadu/"], `class="chart sweep-panels"`); n != 2 {
			t.Errorf("%s: sapuan rho dan lambda harus dua grafik, dapat %d", lang, n)
		}
	}
}

// plainText membuang tag tanpa menyisipkan spasi dan menerjemahkan entitas,
// sehingga "<code>G1</code> lulus" terbaca "G1 lulus" dan "&amp;" terbaca "&".
func plainText(s string) string {
	return html.UnescapeString(tagRe.ReplaceAllString(s, ""))
}

// TestDetailTakeawaysQuoteTheAnalysis: arti grafik rincian harus mengutip
// angka yang sama dengan analisis.
func TestDetailTakeawaysQuoteTheAnalysis(t *testing.T) {
	a := analysisFor(t)
	pages := renderAll(t)
	rs, ks := a.RhoSweep, a.RiskSweep
	last := a.Ladder[len(a.Ladder)-1]
	gap := riskFrequencyGap(a)
	est, fs, mv, cv, wb := estimateSummary(a), floatSummary(a), movedSummary(a), cvSummary(a), wbsSummary()
	closed := 0
	for _, r := range model.Risks {
		if a.InFlight.ClosedRisk[r.ID] {
			closed++
		}
	}
	same := 0
	for _, ft := range a.FastTracks {
		if ft.SameResource {
			same++
		}
	}
	cvText := fmt.Sprintf("%d dari %d aktivitas yang sudah berbiaya boros", cv.Over, cv.WithAC)
	if cv.Over == cv.WithAC {
		cvText = fmt.Sprintf("Semua %d aktivitas yang sudah berbiaya boros", cv.WithAC)
	}
	checks := []struct{ key, want string }{
		{"id /simulasi-terpadu/", "menggeser P80 dari " + render.Num(rs[0].P80, 0, "id") + " ke " + render.Num(rs[len(rs)-1].P80, 0, "id") + " hari kerja"},
		{"id /simulasi-terpadu/", "simpangan baku berubah dari " + render.Num(rs[0].StdDev, 2, "id") + " ke " + render.Num(rs[len(rs)-1].StdDev, 2, "id")},
		{"en /simulasi-terpadu/", "four risks at once goes from " + render.Pct(ks[0].ManyRisks, 1, "en") + " to " + render.Pct(ks[len(ks)-1].ManyRisks, 1, "en")},
		{"id /simulasi-terpadu/", "G1 lulus tanpa diulang pada " + render.Pct(1-a.GERT[0].Loop.FailProb, 0, "id")},
		{"id /simulasi-terpadu/", "rerata sewa " + render.Rp(last.TimeCost, "id")},
		{"id /simulasi-terpadu/", "Selisih terbesar hanya " + render.Num(gap.Gap*100, 1, "id") + " poin persentase (" + gap.ID + ")"},
		{"id /prakiraan/", fmt.Sprintf("%d dari %d risiko sudah ditutup", closed, len(model.Risks))},
		{"id /pert/", fmt.Sprintf("%d dari %d aktivitas punya rerata di atas durasi paling mungkin", est.RightSkewed, est.Total)},
		{"id /pert/", est.Widest.ID + " " + est.Widest.Name.Get("id")},
		{"id /jadwal/", fmt.Sprintf("Hanya %d dari %d aktivitas punya ruang gerak, dan %d di antaranya", fs.WithFloat, fs.Total, fs.FreeOnly)},
		{"id /optimasi/", fmt.Sprintf("%d aktivitas bergeser: %d hari terbawa, %d hari menunggu orang, %d hari memanjang", mv.Count, mv.Carried, mv.Wait, mv.Stretch)},
		{"en /optimasi/", fmt.Sprintf("Of %d candidates, %d are viable and %d are rejected", len(a.FastTracks), len(a.FastViable), same)},
		{"id /biaya/", cvText},
		{"id /biaya/", cv.WorstID + " " + cv.Worst.Get("id")},
		{"id /piagam/", "Fase " + wb.Top.Code + " " + wb.Top.Name.Get("id") + " menyerap " + render.Pct(wb.Share, 0, "id")},
		{"id /coretax/", "Kerugian sebulan " + render.Ratio(a.Coretax.ExposureRatio, "id")},
		{"en /coretax/", "It took " + render.Num(a.Coretax.PerpresToGoLiveYears, 1, "en") + " years"},
	}
	for _, c := range checks {
		takeaways := plainText(strings.Join(blocks(pages[c.key], `<p class="takeaway">`, "</p>"), "\n"))
		if !strings.Contains(takeaways, c.want) {
			t.Errorf("%s: arti tidak memuat %q", c.key, c.want)
		}
	}
}

// TestDetailClaimsHold memeriksa klaim yang ditulis kalimat arti terhadap
// hitungan independen, bukan terhadap fungsi ringkasan yang sama.
func TestDetailClaimsHold(t *testing.T) {
	a := analysisFor(t)

	// "Selisih terbesar hanya ... lingkaran praktis berimpit": frekuensi
	// sepuluh ribu iterasi harus berada dalam derau binomial register.
	fin := a.Final()
	n := float64(fin.Config.Iterations)
	for _, r := range model.Risks {
		freq := float64(fin.RiskHits[r.ID]) / n
		if limit := 4 * math.Sqrt(r.ResidualProb*(1-r.ResidualProb)/n); math.Abs(freq-r.ResidualProb) > limit+1e-9 {
			t.Errorf("%s: frekuensi %v jauh dari peluang register %v (batas %v)", r.ID, freq, r.ResidualProb, limit)
		}
	}
	if g := riskFrequencyGap(a); g.Gap > 0.02 {
		t.Errorf("selisih terbesar %v terlalu besar untuk ditulis 'praktis berimpit'", g.Gap)
	}

	// "risiko sudah ditutup dan frekuensinya nol".
	for _, r := range model.Risks {
		if a.InFlight.ClosedRisk[r.ID] && a.Forecast.RiskHits[r.ID] != 0 {
			t.Errorf("%s ditutup tetapi terjadi %d kali di prakiraan", r.ID, a.Forecast.RiskHits[r.ID])
		}
	}

	// Rerata beta-PERT di atas M persis bila O + P > 2M.
	est := estimateSummary(a)
	skew := 0
	for _, act := range model.Activities {
		if !act.Milestone && act.Optimistic+act.Pessimistic > 2*act.Duration {
			skew++
		}
	}
	if skew != est.RightSkewed {
		t.Errorf("condong kanan %d, hitung tangan O + P > 2M memberi %d", est.RightSkewed, skew)
	}

	// Pecahan pergeseran menjumlah ke pergeseran total setiap aktivitas.
	mv, total := movedSummary(a), 0
	for _, mt := range a.MovedTasks() {
		total += mt.Delay()
	}
	if mv.Carried+mv.Wait+mv.Stretch != total {
		t.Errorf("terbawa %d + menunggu %d + memanjang %d != total pergeseran %d", mv.Carried, mv.Wait, mv.Stretch, total)
	}

	// Anggaran fase WBS menjumlah ke BAC aktivitas.
	wb := wbsSummary()
	if wb.Share <= 0 || math.Abs(wb.Budget/wb.Share-a.BAC) > 1 {
		t.Errorf("anggaran fase menjumlah %v, BAC %v", wb.Budget/wb.Share, a.BAC)
	}

	// Kandidat fast-tracking: layak + orang sama + tanpa hari = semua.
	viable, same, none := 0, 0, 0
	for _, ft := range a.FastTracks {
		switch {
		case ft.Viable():
			viable++
		case ft.SameResource:
			same++
		default:
			none++
		}
	}
	if viable != len(a.FastViable) || viable+same+none != len(a.FastTracks) {
		t.Errorf("kandidat %d layak %d orang sama %d tanpa hari %d, FastViable %d", len(a.FastTracks), viable, same, none, len(a.FastViable))
	}

	// "butuh paling banyak Q90 putaran pada 90% proyek".
	for _, g := range a.GERT {
		cum := 0.0
		for k := 0; k <= g.Q90; k++ {
			cum += render.LoopProb(g.Loop.FailProb, k, 1000)
		}
		below := cum - render.LoopProb(g.Loop.FailProb, g.Q90, 1000)
		if cum < 0.9 || below >= 0.9 {
			t.Errorf("%s: P(N <= %d) = %v, P(N <= %d) = %v - Q90 tidak konsisten", g.Loop.ID, g.Q90, cum, g.Q90-1, below)
		}
	}

	// Sewa pada lapisan paling realistis di atas rencana, dan varians biaya
	// terburuk memang yang paling negatif.
	if last := a.Ladder[len(a.Ladder)-1]; last.TimeCost <= last.TimeCostPlan {
		t.Errorf("sewa L4 %v tidak di atas rencana %v", last.TimeCost, last.TimeCostPlan)
	}
	cv := cvSummary(a)
	for _, r := range a.Rows {
		if r.AC != 0 && r.CV < cv.WorstCV {
			t.Errorf("%s punya CV %v lebih buruk dari yang disebut terburuk %v", r.ID, r.CV, cv.WorstCV)
		}
	}
}

// TestChartFuncsDegradeWithoutOptionalAnalysis: fungsi grafik yang bergantung
// pada paket keputusan atau prakiraan berjalan harus mengembalikan kosong,
// bukan panik, bila bagian analisis itu tidak ada.
func TestChartFuncsDegradeWithoutOptionalAnalysis(t *testing.T) {
	b := *analysisFor(t)
	b.Decision, b.InFlight = nil, nil
	funcs := template.FuncMap{}
	for k, v := range visualFuncs(&b) {
		funcs[k] = v
	}
	for k, v := range detailFuncs(&b) {
		funcs[k] = v
	}
	for _, name := range []string{"optionMap", "valueBandsChart", "scenarioStrips", "budgetBridge", "decisionOvertimeCurve", "credibilityPanels", "forecastRiskChart"} {
		fn, ok := funcs[name].(func(string) template.HTML)
		if !ok {
			t.Fatalf("%s tidak terdaftar sebagai func(string) template.HTML", name)
		}
		if got := fn("id"); got != "" {
			t.Errorf("%s tanpa analisis opsional harus kosong, dapat %d bait", name, len(got))
		}
	}
}

func TestBiggestCause(t *testing.T) {
	cases := []struct {
		c, w, s int
		want    string
	}{
		{563, 25, 37, "carried"},
		{5, 20, 3, "wait"},
		{5, 3, 20, "stretch"},
		{10, 10, 10, "carried"},
		{1, 10, 10, "wait"},
		{0, 0, 0, "carried"},
	}
	for _, c := range cases {
		if got := biggestCause(c.c, c.w, c.s); got != c.want {
			t.Errorf("biggestCause(%d, %d, %d) = %q, mau %q", c.c, c.w, c.s, got, c.want)
		}
	}
}
