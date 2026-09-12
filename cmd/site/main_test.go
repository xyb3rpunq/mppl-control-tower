package main

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/site"
)

// Uji di berkas ini merender HTML yang SEBENARNYA, bukan memeriksa kode yang
// menghasilkannya. Membaca kode hanya membuktikan niat; merender membuktikan
// hasil. Kebocoran bahasa, kunci i18n yang hilang, dan templat yang salah
// hanya muncul di keluaran jadi.

func init() {
	// Templat dimuat dengan jalur relatif terhadap akar repositori.
	if _, err := os.Stat("web/templates"); err != nil {
		if err := os.Chdir("../.."); err != nil {
			panic("tidak bisa pindah ke akar repositori: " + err.Error())
		}
	}
}

// renderAll merender seluruh halaman dalam kedua bahasa dan mengembalikan
// petanya: "lang route" -> HTML.
func renderAll(t *testing.T) map[string]string {
	t.Helper()
	analysis, err := site.Build(model.DefaultStatusDate)
	if err != nil {
		t.Fatalf("site.Build: %v", err)
	}
	tmpl, err := loadTemplates(analysis)
	if err != nil {
		t.Fatalf("loadTemplates: %v", err)
	}

	out := map[string]string{}
	for _, lang := range i18n.Langs {
		examples := analysis.Examples(lang)
		for _, p := range site.Pages {
			data := PageData{
				Lang: lang, OtherLang: i18n.OtherLang(lang),
				Page: p, Pages: site.Pages,
				Canonical: site.PathFor(p.Route, lang),
				AltURL:    site.PathFor(p.Route, i18n.OtherLang(lang)),
				Title:     p.NavLabel(lang), Desc: p.Summary.Get(lang),
				BuildTime: time.Now().Format(time.RFC3339),
				Analysis:  analysis, Examples: examples,
				Charter: model.ProjectCharter, Risk: analysis.Risk,
			}
			var body bytes.Buffer
			if err := tmpl.ExecuteTemplate(&body, p.Template, data); err != nil {
				t.Fatalf("render isi %s (%s): %v", p.Route, lang, err)
			}
			data.Content = template.HTML(body.String())
			var full bytes.Buffer
			if err := tmpl.ExecuteTemplate(&full, "base", data); err != nil {
				t.Fatalf("render kerangka %s (%s): %v", p.Route, lang, err)
			}
			out[lang+" "+p.Route] = full.String()
		}
	}
	return out
}

func TestEveryPageRendersWithoutError(t *testing.T) {
	pages := renderAll(t)
	want := len(site.Pages) * len(i18n.Langs)
	if len(pages) != want {
		t.Fatalf("%d halaman dirender, mau %d", len(pages), want)
	}
	for key, html := range pages {
		if len(html) < 3000 {
			t.Errorf("%s: hanya %d bait, halaman kemungkinan kosong", key, len(html))
		}
		if !strings.Contains(html, "<!doctype html>") {
			t.Errorf("%s: tidak punya doctype", key)
		}
		if !strings.Contains(html, `id="utama"`) {
			t.Errorf("%s: tidak punya landmark main", key)
		}
	}
}

// TestNoMissingTranslationKeys menangkap kunci i18n yang salah tulis. Fungsi
// i18n.T sengaja mengembalikan "!kunci" alih-alih string kosong supaya
// kesalahan semacam ini terlihat, dan uji inilah yang melihatnya.
func TestNoMissingTranslationKeys(t *testing.T) {
	missing := regexp.MustCompile(`!(site|nav|t|h)\.[A-Za-z]+`)
	for key, html := range renderAll(t) {
		if m := missing.FindAllString(html, 5); len(m) > 0 {
			t.Errorf("%s: kunci terjemahan tidak ditemukan: %v", key, m)
		}
	}
}

// TestEnglishPagesAreActuallyEnglish adalah uji dwibahasa yang sesungguhnya.
// Memeriksa kelengkapan kamus tidak cukup: teks bisa saja lengkap tetapi
// templatnya lupa memilih cabang bahasa. Yang membuktikan hanyalah membaca
// halaman jadinya.
//
// Deteksinya memakai kata fungsi Indonesia, bukan perbandingan identik: kata
// benda dan istilah teknis memang sama di kedua bahasa, sedangkan "yang" atau
// "dengan" tidak akan pernah muncul di kalimat Inggris yang benar.
func TestEnglishPagesAreActuallyEnglish(t *testing.T) {
	indonesianOnly := []string{
		"yang", "dengan", "untuk", "adalah", "tidak", "sehingga", "karena",
		"dari", "pada", "akan", "sudah", "belum", "harus", "hanya", "setiap",
		"terhadap", "sebagai", "dalam", "lebih", "agar",
	}
	word := map[string]*regexp.Regexp{}
	for _, w := range indonesianOnly {
		word[w] = regexp.MustCompile(`(?i)\b` + w + `\b`)
	}

	pages := renderAll(t)
	for _, p := range site.Pages {
		html := pages["en "+p.Route]
		text := stripTags(html)
		for w, re := range word {
			if m := re.FindAllString(text, 3); len(m) > 0 {
				t.Errorf("halaman Inggris %s memuat kata Indonesia %q %d kali - "+
					"ada cabang templat yang lupa diterjemahkan", p.Route, w, len(m))
			}
		}
	}
}

func TestIndonesianPagesAreActuallyIndonesian(t *testing.T) {
	// Sisi sebaliknya, dengan kata fungsi Inggris yang tidak dipakai dalam
	// bahasa Indonesia. Istilah teknis seperti "the critical path" memang
	// lazim dipertahankan, jadi ambangnya dibuat longgar dan hanya menangkap
	// kalimat utuh yang tertinggal dalam bahasa Inggris.
	markers := []string{`\bthe project\b`, `\bshould be\b`, `\bwhich means\b`, `\bin order to\b`}
	pages := renderAll(t)
	for _, p := range site.Pages {
		text := stripTags(pages["id "+p.Route])
		for _, m := range markers {
			re := regexp.MustCompile(`(?i)` + m)
			if hits := re.FindAllString(text, 2); len(hits) > 0 {
				t.Errorf("halaman Indonesia %s memuat frasa Inggris %q - cabang templat tertukar", p.Route, hits[0])
			}
		}
	}
}

// TestKeyNumbersAppearOnTheirPages memastikan angka hasil hitungan benar-benar
// sampai ke halaman, bukan berhenti di struct.
func TestKeyNumbersAppearOnTheirPages(t *testing.T) {
	pages := renderAll(t)
	checks := []struct {
		route string
		want  string
		why   string
	}{
		{"/", "0,915", "SPI harus tampil di dasbor"},
		{"/", "0,918", "CPI harus tampil di dasbor"},
		{"/biaya/", "16.157.314", "EAC tipikal harus tampil di halaman biaya"},
		{"/biaya/", "14.832.000", "BAC harus tampil di halaman biaya"},
		{"/risiko/", "2.880.000", "EMV residual harus tampil di halaman risiko"},
		{"/coretax/", "47,8x", "rasio paparan harus tampil di studi kasus"},
		{"/coretax/", "1.228", "nilai kontrak harus tampil di studi kasus"},
		{"/piagam/", "20 Feb 2026", "tanggal selesai kalender kerja harus tampil di piagam"},
		{"/jadwal/", "A17", "kode aktivitas harus tampil di tabel CPM"},
	}
	for _, c := range checks {
		if !strings.Contains(pages["id "+c.route], c.want) {
			t.Errorf("%s: %q tidak ditemukan - %s", c.route, c.want, c.why)
		}
	}
}

// TestCoretaxSourcesAreLinked memastikan setiap fakta pada studi kasus
// benar-benar tertaut ke sumbernya di HTML jadi, bukan hanya di data.
func TestCoretaxSourcesAreLinked(t *testing.T) {
	html := renderAll(t)["id /coretax/"]
	for _, s := range model.CoretaxSources {
		if !strings.Contains(html, s.URL) {
			t.Errorf("URL sumber tidak muncul di halaman: %s", s.URL)
		}
	}
	// Semua tautan keluar harus membawa rel yang aman.
	extLinks := regexp.MustCompile(`<a href="https?://[^"]+"[^>]*>`).FindAllString(html, -1)
	for _, link := range extLinks {
		if !strings.Contains(link, "noopener") {
			t.Errorf("tautan keluar tanpa rel=\"noopener\": %s", link)
		}
	}
}

func TestNavigationIsCompleteAndLinked(t *testing.T) {
	pages := renderAll(t)
	for _, lang := range i18n.Langs {
		html := pages[lang+" /"]
		for _, p := range site.Pages {
			href := site.PathFor(p.Route, lang)
			if !strings.Contains(html, `href="`+href+`"`) {
				t.Errorf("navigasi %s kehilangan tautan ke %s", lang, href)
			}
			// Label dibandingkan setelah di-escape: "Piagam & Lingkup" muncul
			// di HTML sebagai "Piagam &amp; Lingkup".
			label := template.HTMLEscapeString(p.NavLabel(lang))
			if !strings.Contains(html, label) {
				t.Errorf("navigasi %s kehilangan label %q", lang, p.NavLabel(lang))
			}
		}
	}
}

func TestHreflangPairsPointAtEachOther(t *testing.T) {
	pages := renderAll(t)
	for _, p := range site.Pages {
		id := pages["id "+p.Route]
		en := pages["en "+p.Route]
		if !strings.Contains(id, `hreflang="en" href="`+site.PathFor(p.Route, "en")+`"`) {
			t.Errorf("%s (id) tidak menunjuk versi Inggrisnya", p.Route)
		}
		if !strings.Contains(en, `hreflang="id" href="`+site.PathFor(p.Route, "id")+`"`) {
			t.Errorf("%s (en) tidak menunjuk versi Indonesianya", p.Route)
		}
	}
}

func TestNoUnrenderedTemplateSyntaxLeaks(t *testing.T) {
	for key, html := range renderAll(t) {
		for _, bad := range []string{"{{", "}}", "<no value>", "%!s(", "%!d("} {
			if strings.Contains(html, bad) {
				t.Errorf("%s: memuat sisa sintaks templat atau format %q", key, bad)
			}
		}
	}
}

func TestGeneratedFilesAreWritten(t *testing.T) {
	dir := t.TempDir()
	analysis, err := site.Build(model.DefaultStatusDate)
	if err != nil {
		t.Fatalf("site.Build: %v", err)
	}
	if err := writeExtras(dir, "https://example.test", analysis); err != nil {
		t.Fatalf("writeExtras: %v", err)
	}
	for _, name := range []string{
		".nojekyll", "sw.js", "sitemap.xml", "robots.txt",
		filepath.Join("data", "aktivitas.csv"),
		filepath.Join("data", "risiko.csv"),
		filepath.Join("data", "metrik.json"),
	} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s tidak ditulis: %v", name, err)
			continue
		}
		if name != ".nojekyll" && info.Size() == 0 {
			t.Errorf("%s kosong", name)
		}
	}

	// CSV aktivitas harus memuat satu baris per simpul, plus judul kolom.
	csv, err := os.ReadFile(filepath.Join(dir, "data", "aktivitas.csv"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(csv)), "\n")
	if len(lines) != len(model.Activities)+1 {
		t.Errorf("aktivitas.csv punya %d baris, mau %d", len(lines), len(model.Activities)+1)
	}

	// Sitemap harus memuat setiap rute dalam kedua bahasa.
	sm, _ := os.ReadFile(filepath.Join(dir, "sitemap.xml"))
	for _, lang := range i18n.Langs {
		for _, p := range site.Pages {
			loc := "https://example.test" + site.PathFor(p.Route, lang)
			if !strings.Contains(string(sm), "<loc>"+loc+"</loc>") {
				t.Errorf("sitemap kehilangan %s", loc)
			}
		}
	}
}

func TestFindingsReachTheDashboard(t *testing.T) {
	analysis, err := site.Build(model.DefaultStatusDate)
	if err != nil {
		t.Fatalf("site.Build: %v", err)
	}
	if len(analysis.Findings) == 0 {
		t.Fatal("tidak ada temuan sama sekali")
	}
	html := renderAll(t)["id /"]
	for _, f := range analysis.Findings {
		if !strings.Contains(html, f.Title.ID) {
			t.Errorf("temuan %q tidak muncul di dasbor", f.Key)
		}
		if f.Metric.ID == "" || f.Metric.EN == "" || f.Action.ID == "" {
			t.Errorf("temuan %q tidak lengkap: metrik atau rekomendasinya kosong", f.Key)
		}
	}
}

// TestEveryFormulaHasAWorkedExample memastikan halaman rumus tidak memuat
// rumus yang menggantung tanpa contoh hitung dari data hidup.
func TestEveryFormulaHasAWorkedExample(t *testing.T) {
	analysis, err := site.Build(model.DefaultStatusDate)
	if err != nil {
		t.Fatalf("site.Build: %v", err)
	}
	for _, lang := range i18n.Langs {
		ex := analysis.Examples(lang)
		for _, f := range model.Formulas {
			e, ok := ex[f.Key]
			if !ok || e.Substitution == "" || e.Result == "" {
				t.Errorf("rumus %q (%s) tidak punya contoh hitung", f.Key, lang)
			}
		}
	}
}

var tagRe = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>|<[^>]+>`)

func stripTags(html string) string {
	return tagRe.ReplaceAllString(html, " ")
}
