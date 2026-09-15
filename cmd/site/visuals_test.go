package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/render"
	"github.com/xyb3rpunq/mppl-control-tower/internal/site"
)

// blocks memotong semua blok dari open sampai close pertama sesudahnya.
// Sengaja tanpa regexp: halaman berukuran megabita dan regexp berkelompok
// di atasnya membuat uji berjalan bermenit-menit.
func blocks(s, open, close string) []string {
	var out []string
	for {
		i := strings.Index(s, open)
		if i < 0 {
			return out
		}
		j := strings.Index(s[i:], close)
		if j < 0 {
			return out
		}
		out = append(out, s[i:i+j+len(close)])
		s = s[i+j+len(close):]
	}
}

// between mengembalikan teks di antara open dan close pertama.
func between(s, open, close string) string {
	i := strings.Index(s, open)
	if i < 0 {
		return ""
	}
	s = s[i+len(open):]
	if j := strings.Index(s, close); j >= 0 {
		return s[:j]
	}
	return ""
}

// hasChart memeriksa apakah halaman memuat SVG grafik dengan kelas c di
// antara kelas-kelasnya, mis. "chart gantt gantt-levelled".
func hasChart(page, c string) bool {
	for _, b := range blocks(page, `<svg `, `>`) {
		for _, cls := range strings.Fields(between(b, `class="`, `"`)) {
			if cls == c && strings.Contains(b, `class="chart `) {
				return true
			}
		}
	}
	return false
}

// guides mengurai setiap panduan baca menjadi judul, butir, judul arti, dan arti.
func guides(page string) [][4]string {
	var out [][4]string
	for _, g := range blocks(page, `<div class="reading-guide`, `</p></div>`) {
		out = append(out, [4]string{
			between(g, "<h4>", "</h4>"),
			between(g, "<ul>", "</ul>"),
			between(g[strings.Index(g, "</ul>"):], "<h4>", "</h4>"),
			between(g, `<p class="takeaway">`, "</p>"),
		})
	}
	return out
}

// TestEveryChartHasAReadingGuide menjaga janji fitur ini: tidak ada grafik
// yang dibiarkan tanpa penjelasan cara membaca dan artinya. Grafik yang
// berpasangan (dua kolom atau berulang) boleh berbagi satu panduan mandiri
// di bagian yang sama.
func TestEveryChartHasAReadingGuide(t *testing.T) {
	pages := renderAll(t)
	for key, page := range pages {
		for _, sec := range blocks(page, `<section class="band">`, "</section>") {
			shared := strings.Contains(sec, `class="reading-guide standalone"`)
			for _, fig := range blocks(sec, `<figure class="figure">`, "</figure>") {
				if !strings.Contains(fig, "<svg") {
					continue
				}
				if !strings.Contains(fig, `class="reading-guide"`) && !shared {
					t.Errorf("%s: grafik %q tidak punya panduan baca", key, between(fig, `">`, "</title>"))
				}
			}
		}
	}
}

// TestReadingGuidesAreCompleteAndTranslated memeriksa isi setiap panduan:
// judul sesuai bahasa, paling sedikit dua butir cara membaca, dan arti yang
// benar-benar berupa kalimat - bukan templat kosong.
func TestReadingGuidesAreCompleteAndTranslated(t *testing.T) {
	pages := renderAll(t)
	total := 0
	for _, lang := range i18n.Langs {
		for _, p := range site.Pages {
			key := lang + " " + p.Route
			for _, g := range guides(pages[key]) {
				total++
				if g[0] != i18n.T(lang, "t.reading") || g[2] != i18n.T(lang, "t.takeaway") {
					t.Errorf("%s: judul panduan %q/%q tidak sesuai bahasa", key, g[0], g[2])
				}
				if n := strings.Count(g[1], "<li>"); n < 2 {
					t.Errorf("%s: panduan hanya punya %d butir cara membaca", key, n)
				}
				meaning := stripTags(g[3])
				if len(meaning) < 60 {
					t.Errorf("%s: arti terlalu pendek untuk menjelaskan apa pun: %q", key, meaning)
				}
				if strings.Contains(meaning, "<no value>") || strings.Contains(meaning, "%!") {
					t.Errorf("%s: arti memuat nilai templat rusak: %q", key, meaning)
				}
			}
		}
	}
	// Semua panduan harus tertangkap pola; kalau strukturnya berubah, uji di
	// atas diam-diam tidak memeriksa apa-apa.
	raw := 0
	for _, page := range pages {
		raw += strings.Count(page, `<div class="reading-guide`)
	}
	if total != raw || total < 2*40 {
		t.Errorf("%d panduan terbaca pola dari %d di halaman (minimal 80 untuk dua bahasa)", total, raw)
	}
}

// TestExplanatoryChartsReachTheirPages memastikan setiap grafik penjelas
// benar-benar dirender di halamannya dalam kedua bahasa.
func TestExplanatoryChartsReachTheirPages(t *testing.T) {
	want := map[string][]string{
		"/keputusan/":  {"date-dots", "option-map", "value-bands", "scenario-strips", "overtime-curve", "bridge"},
		"/prakiraan/":  {"date-dots", "credibility", "frontier-line"},
		"/optimasi/":   {"capacity-timeline", "overtime-curve", "bound-chart", "gantt-levelled"},
		"/biaya/":      {"index-dots"},
		"/risiko/":     {"pair-bars"},
		"/kualitas/":   {"metric-bars", "control-chart", "coq"},
		"/organisasi/": {"resource-histogram"},
	}
	pages := renderAll(t)
	for _, lang := range i18n.Langs {
		for route, classes := range want {
			page := pages[lang+" "+route]
			for _, c := range classes {
				if !hasChart(page, c) {
					t.Errorf("%s %s: grafik %q tidak dirender", lang, route, c)
				}
			}
		}
	}
}

// TestTakeawaysQuoteTheComputedNumbers: arti setiap grafik harus berasal dari
// hasil hitungan, jadi angka yang ditulis harus sama dengan angka di analisis.
func TestTakeawaysQuoteTheComputedNumbers(t *testing.T) {
	a := analysisFor(t)
	pages := renderAll(t)
	miss := 0
	for _, m := range a.Metrics {
		if !m.Met {
			miss++
		}
	}
	checks := []struct{ key, want string }{
		{"id /kualitas/", fmt.Sprintf("%d dari %d metrik belum memenuhi target", miss, len(a.Metrics))},
		{"en /kualitas/", fmt.Sprintf("%d of %d metrics miss their target", miss, len(a.Metrics))},
		{"id /kualitas/", fmt.Sprintf("%d modul", a.Pareto.VitalFewCount)},
		{"id /optimasi/", fmt.Sprintf("%d hari tambahan datang dari bentrokan kapasitas dan %d hari dari periode ujian", a.LevelWhy.CapacityOnly-a.LevelWhy.CPM, a.LevelWhy.WithWindows-a.LevelWhy.CapacityOnly)},
		{"en /optimasi/", fmt.Sprintf("%d extra days come from capacity clashes and %d from exam periods", a.LevelWhy.CapacityOnly-a.LevelWhy.CPM, a.LevelWhy.WithWindows-a.LevelWhy.CapacityOnly)},
		{"id /pert/", "Hanya " + render.Pct(a.Sim.OnTimeProb, 1, "id") + " iterasi selesai"},
		{"en /pert/", "Only " + render.Pct(a.Sim.OnTimeProb, 1, "en") + " of iterations finish"},
		{"id /organisasi/", fmt.Sprintf("%d hari-peran melebihi kapasitas", len(a.Resources.Conflicts))},
		{"id /simulasi-terpadu/", "ke " + render.Num(a.Final().DurP80, 0, "id") + " hari kerja (L4)"},
		{"id /risiko/", "EMV turun dari " + render.Rp(a.Risk.TotalEMV, "id") + " ke " + render.Rp(a.Risk.TotalResidualEMV, "id")},
		{"id /biaya/", "SPI " + render.Num(a.Snapshot.SPI, 3, "id")},
	}
	for _, c := range checks {
		takeaways := strings.Join(blocks(pages[c.key], `<p class="takeaway">`, "</p>"), "\n")
		if !strings.Contains(stripTags(takeaways), c.want) {
			t.Errorf("%s: arti tidak memuat %q", c.key, c.want)
		}
	}
}
