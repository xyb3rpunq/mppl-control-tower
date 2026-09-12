// Package render membangun seluruh grafik sebagai SVG inline.
//
// Keputusan penting: grafik digambar di sisi server dan dikirim sebagai SVG di
// dalam HTML, bukan digambar ulang oleh JavaScript setelah halaman dimuat.
// Konsekuensinya baik: halaman tetap utuh tanpa JavaScript, bisa dicetak jadi
// PDF apa adanya, terbaca mesin pengindeks, dan tidak ada kedipan saat muat.
// Interaksi yang benar-benar butuh hitung ulang - misalnya menggeser tanggal
// data - ditangani WebAssembly yang mengganti SVG-nya, bukan membangunnya
// dari nol.
//
// Semua warna memakai CSS custom property sehingga satu berkas SVG yang sama
// tampil benar di tema terang maupun gelap tanpa digambar ulang.
package render

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// Canvas adalah pembangun SVG dengan sistem koordinat data.
type Canvas struct {
	W, H       float64
	Pad        Padding
	sb         strings.Builder
	xMin, xMax float64
	yMin, yMax float64
	title      string
	desc       string
}

// Padding adalah ruang tepi untuk sumbu dan label.
type Padding struct{ Top, Right, Bottom, Left float64 }

// NewCanvas membuat kanvas baru dengan rentang data tertentu.
func NewCanvas(w, h float64, pad Padding) *Canvas {
	return &Canvas{W: w, H: h, Pad: pad, xMin: 0, xMax: 1, yMin: 0, yMax: 1}
}

// SetDomain menetapkan rentang data sumbu X dan Y.
func (c *Canvas) SetDomain(xMin, xMax, yMin, yMax float64) *Canvas {
	if xMax == xMin {
		xMax = xMin + 1
	}
	if yMax == yMin {
		yMax = yMin + 1
	}
	c.xMin, c.xMax, c.yMin, c.yMax = xMin, xMax, yMin, yMax
	return c
}

// Describe memberi judul dan deskripsi aksesibilitas pada grafik. Keduanya
// wajib: pembaca layar tidak bisa melihat garis, dan grafik tanpa deskripsi
// sama saja dengan kotak kosong bagi sebagian pembaca.
func (c *Canvas) Describe(title, desc string) *Canvas {
	c.title, c.desc = title, desc
	return c
}

// Plot area helpers.
func (c *Canvas) plotW() float64 { return c.W - c.Pad.Left - c.Pad.Right }
func (c *Canvas) plotH() float64 { return c.H - c.Pad.Top - c.Pad.Bottom }

// X memetakan nilai data ke koordinat layar.
func (c *Canvas) X(v float64) float64 {
	return c.Pad.Left + (v-c.xMin)/(c.xMax-c.xMin)*c.plotW()
}

// Y memetakan nilai data ke koordinat layar (sumbu Y terbalik).
func (c *Canvas) Y(v float64) float64 {
	return c.Pad.Top + c.plotH() - (v-c.yMin)/(c.yMax-c.yMin)*c.plotH()
}

// Raw menambahkan markup SVG mentah.
func (c *Canvas) Raw(s string) *Canvas { c.sb.WriteString(s); return c }

// Rect menggambar persegi panjang dalam koordinat layar.
func (c *Canvas) Rect(x, y, w, h float64, class string, extra ...string) *Canvas {
	if w < 0 {
		x, w = x+w, -w
	}
	if h < 0 {
		y, h = y+h, -h
	}
	c.sb.WriteString(fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" class="%s"%s/>`,
		f(x), f(y), f(w), f(h), class, attrs(extra)))
	return c
}

// Line menggambar garis lurus dalam koordinat layar.
func (c *Canvas) Line(x1, y1, x2, y2 float64, class string, extra ...string) *Canvas {
	c.sb.WriteString(fmt.Sprintf(`<line x1="%s" y1="%s" x2="%s" y2="%s" class="%s"%s/>`,
		f(x1), f(y1), f(x2), f(y2), class, attrs(extra)))
	return c
}

// Path menggambar jalur dari perintah SVG mentah.
func (c *Canvas) Path(d, class string, extra ...string) *Canvas {
	c.sb.WriteString(fmt.Sprintf(`<path d="%s" class="%s"%s/>`, d, class, attrs(extra)))
	return c
}

// Circle menggambar lingkaran.
func (c *Canvas) Circle(cx, cy, r float64, class string, extra ...string) *Canvas {
	c.sb.WriteString(fmt.Sprintf(`<circle cx="%s" cy="%s" r="%s" class="%s"%s/>`,
		f(cx), f(cy), f(r), class, attrs(extra)))
	return c
}

// Text menulis teks. anchor: "start", "middle", "end".
func (c *Canvas) Text(x, y float64, s, class, anchor string, extra ...string) *Canvas {
	c.sb.WriteString(fmt.Sprintf(`<text x="%s" y="%s" class="%s" text-anchor="%s"%s>%s</text>`,
		f(x), f(y), class, anchor, attrs(extra), html.EscapeString(s)))
	return c
}

// Title menyisipkan elemen title sebagai tooltip bawaan peramban.
func (c *Canvas) Title(s string) *Canvas {
	c.sb.WriteString("<title>" + html.EscapeString(s) + "</title>")
	return c
}

// Group membuka grup SVG.
func (c *Canvas) Group(class string, extra ...string) *Canvas {
	c.sb.WriteString(fmt.Sprintf(`<g class="%s"%s>`, class, attrs(extra)))
	return c
}

// EndGroup menutup grup SVG.
func (c *Canvas) EndGroup() *Canvas { c.sb.WriteString("</g>"); return c }

// PolyLine menggambar garis patah dari titik data.
func (c *Canvas) PolyLine(xs, ys []float64, class string, extra ...string) *Canvas {
	var d strings.Builder
	started := false
	for i := range xs {
		if i >= len(ys) || math.IsNaN(ys[i]) {
			started = false
			continue
		}
		cmd := "L"
		if !started {
			cmd = "M"
			started = true
		}
		d.WriteString(fmt.Sprintf("%s%s %s ", cmd, f(c.X(xs[i])), f(c.Y(ys[i]))))
	}
	if d.Len() == 0 {
		return c
	}
	return c.Path(strings.TrimSpace(d.String()), class, extra...)
}

// Area menggambar area di bawah kurva sampai garis dasar.
func (c *Canvas) Area(xs, ys []float64, baseline float64, class string) *Canvas {
	var d strings.Builder
	var first, last float64
	started := false
	for i := range xs {
		if i >= len(ys) || math.IsNaN(ys[i]) {
			continue
		}
		if !started {
			first = c.X(xs[i])
			d.WriteString(fmt.Sprintf("M%s %s ", f(first), f(c.Y(baseline))))
			started = true
		}
		last = c.X(xs[i])
		d.WriteString(fmt.Sprintf("L%s %s ", f(last), f(c.Y(ys[i]))))
	}
	if !started {
		return c
	}
	d.WriteString(fmt.Sprintf("L%s %s Z", f(last), f(c.Y(baseline))))
	return c.Path(strings.TrimSpace(d.String()), class)
}

// Tick adalah satu penanda sumbu.
type Tick struct {
	Value float64
	Label string
}

// AxisX menggambar sumbu horizontal beserta garis bantu.
func (c *Canvas) AxisX(ticks []Tick, grid bool) *Canvas {
	y := c.Pad.Top + c.plotH()
	c.Line(c.Pad.Left, y, c.Pad.Left+c.plotW(), y, "axis-line")
	for _, t := range ticks {
		x := c.X(t.Value)
		if grid {
			c.Line(x, c.Pad.Top, x, y, "grid-line")
		}
		c.Line(x, y, x, y+4, "axis-line")
		if t.Label != "" {
			c.Text(x, y+16, t.Label, "axis-label", "middle")
		}
	}
	return c
}

// AxisY menggambar sumbu vertikal beserta garis bantu.
func (c *Canvas) AxisY(ticks []Tick, grid bool) *Canvas {
	x := c.Pad.Left
	c.Line(x, c.Pad.Top, x, c.Pad.Top+c.plotH(), "axis-line")
	for _, t := range ticks {
		y := c.Y(t.Value)
		if grid {
			c.Line(x, y, x+c.plotW(), y, "grid-line")
		}
		c.Line(x-4, y, x, y, "axis-line")
		if t.Label != "" {
			c.Text(x-8, y+4, t.Label, "axis-label", "end")
		}
	}
	return c
}

// NiceTicks menghasilkan penanda sumbu pada angka bulat yang enak dibaca.
func NiceTicks(min, max float64, count int) []float64 {
	if count < 2 {
		count = 2
	}
	span := max - min
	if span <= 0 {
		return []float64{min}
	}
	raw := span / float64(count-1)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	norm := raw / mag
	var step float64
	switch {
	case norm <= 1:
		step = 1
	case norm <= 2:
		step = 2
	case norm <= 2.5:
		step = 2.5
	case norm <= 5:
		step = 5
	default:
		step = 10
	}
	step *= mag
	start := math.Ceil(min/step) * step
	var out []float64
	for v := start; v <= max+step*1e-9; v += step {
		out = append(out, math.Round(v/step)*step)
	}
	return out
}

// SVG menutup kanvas dan mengembalikan markup lengkap.
func (c *Canvas) SVG(class string) string {
	var head strings.Builder
	head.WriteString(fmt.Sprintf(
		`<svg viewBox="0 0 %s %s" class="chart %s" role="img" preserveAspectRatio="xMidYMid meet"`,
		f(c.W), f(c.H), class))
	if c.title != "" {
		head.WriteString(` aria-labelledby="t` + slug(c.title) + `"`)
	}
	head.WriteString(">")
	if c.title != "" {
		head.WriteString(`<title id="t` + slug(c.title) + `">` + html.EscapeString(c.title) + `</title>`)
	}
	if c.desc != "" {
		head.WriteString(`<desc>` + html.EscapeString(c.desc) + `</desc>`)
	}
	return head.String() + c.sb.String() + "</svg>"
}

// f memformat angka untuk atribut SVG: maksimal dua desimal, tanpa nol ekor.
func f(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func attrs(extra []string) string {
	if len(extra) == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i+1 < len(extra); i += 2 {
		sb.WriteString(fmt.Sprintf(` %s="%s"`, extra[i], html.EscapeString(extra[i+1])))
	}
	return sb.String()
}

func slug(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			sb.WriteRune('-')
		}
	}
	out := sb.String()
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}
