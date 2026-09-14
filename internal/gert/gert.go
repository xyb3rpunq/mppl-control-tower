// Package gert mengimplementasikan reduksi jaringan GERT (Graphical Evaluation
// and Review Technique, Pritsker 1966) untuk cabang berwaktu tetap.
//
// Setiap cabang punya peluang p dan waktu t. Fungsi transmitansinya
//
//	W(s) = p * exp(s t)
//
// adalah fungsi pembangkit momen yang dikalikan peluang. Tiga aturan reduksi
// cukup untuk jaringan seri, paralel, dan putaran:
//
//	seri      W = W1 * W2
//	paralel   W = W1 + W2            (cabang saling meniadakan)
//	putaran   W = W_maju / (1 - W_putar)   (aturan Mason untuk satu loop)
//
// Dari W_E hasil reduksi: peluang mencapai simpul akhir p_E = W_E(0), dan
// momen waktu diperoleh dari turunan M_E(s) = W_E(s) / W_E(0) di s = 0.
//
// Alih-alih menurunkan fungsi secara numerik, paket ini membawa deret Taylor
// orde dua (nilai, turunan pertama, dan turunan kedua W di s = 0) melalui setiap aturan. Aljabarnya eksak,
// sehingga rerata dan varians hasil reduksi tidak mengandung galat diferensiasi.
package gert

import "math"

// W adalah transmitansi GERT yang diwakili koefisien Taylor orde dua di s = 0.
type W struct {
	A0 float64 // W(0)   = peluang
	A1 float64 // W'(0)  = E[T; lulus]
	A2 float64 // W''(0) = E[T^2; lulus]
}

// Arc membuat satu cabang dengan peluang p dan waktu tetap t.
func Arc(p, t float64) W { return W{A0: p, A1: p * t, A2: p * t * t} }

// Then menyambung dua bagian secara seri: W = w * o.
func (w W) Then(o W) W {
	return W{
		A0: w.A0 * o.A0,
		A1: w.A1*o.A0 + w.A0*o.A1,
		A2: w.A2*o.A0 + 2*w.A1*o.A1 + w.A0*o.A2,
	}
}

// Or menggabungkan cabang yang saling meniadakan: W = jumlah.
func Or(ws ...W) W {
	var out W
	for _, w := range ws {
		out.A0 += w.A0
		out.A1 += w.A1
		out.A2 += w.A2
	}
	return out
}

// Loop mereduksi satu putaran dengan aturan Mason: W = maju / (1 - putar).
func Loop(forward, back W) W {
	d0, d1, d2 := 1-back.A0, -back.A1, -back.A2
	if d0 <= 0 {
		return W{A0: math.Inf(1), A1: math.Inf(1), A2: math.Inf(1)}
	}
	q0 := forward.A0 / d0
	q1 := (forward.A1 - q0*d1) / d0
	q2 := (forward.A2 - 2*q1*d1 - q0*d2) / d0
	return W{A0: q0, A1: q1, A2: q2}
}

// Prob mengembalikan peluang simpul akhir tercapai.
func (w W) Prob() float64 { return w.A0 }

// Mean mengembalikan rerata waktu bersyarat pada simpul akhir tercapai.
func (w W) Mean() float64 {
	if w.A0 == 0 {
		return 0
	}
	return w.A1 / w.A0
}

// Var mengembalikan varians waktu bersyarat pada simpul akhir tercapai.
func (w W) Var() float64 {
	if w.A0 == 0 {
		return 0
	}
	m := w.Mean()
	v := w.A2/w.A0 - m*m
	if v < 0 && v > -1e-9 {
		return 0
	}
	return v
}

// SD mengembalikan simpangan baku waktu.
func (w W) SD() float64 { return math.Sqrt(w.Var()) }

// SelfLoop adalah pemeriksaan yang lulus dengan peluang 1 - p tanpa tambahan
// waktu, atau gagal dengan peluang p dan menghabiskan waktu rework sebelum
// diperiksa lagi. Hasilnya adalah WAKTU TAMBAHAN akibat putaran.
//
// Bentuk tertutupnya: rerata = p*r/(1-p), varians = p*r^2/(1-p)^2.
func SelfLoop(p, rework float64) W { return Loop(Arc(1-p, 0), Arc(p, rework)) }

// ExpectedCycles mengembalikan rerata jumlah putaran tambahan, p/(1-p).
func ExpectedCycles(p float64) float64 {
	if p >= 1 {
		return math.Inf(1)
	}
	return p / (1 - p)
}

// AtLeast mengembalikan peluang terjadi paling sedikit k putaran tambahan, p^k.
func AtLeast(p float64, k int) float64 { return math.Pow(p, float64(k)) }

// CycleQuantile mengembalikan jumlah putaran n terkecil dengan
// P(N <= n) >= q, yaitu 1 - p^(n+1) >= q.
func CycleQuantile(p, q float64) int {
	if p <= 0 || q <= 0 {
		return 0
	}
	if p >= 1 || q >= 1 {
		return math.MaxInt32
	}
	n := int(math.Ceil(math.Log(1-q)/math.Log(p) - 1 - 1e-12))
	if n < 0 {
		n = 0
	}
	return n
}

// MaxCycles membatasi sampel putaran agar simulasi tidak berputar tanpa akhir
// pada peluang gagal yang ekstrem. Untuk p = 0,3 peluang terpotongnya 0,3^20,
// sekitar 3,5e-11.
const MaxCycles = 20

// SampleCycles mengubah bilangan seragam u di (0, 1) menjadi jumlah putaran
// tambahan lewat transformasi invers: N = floor(ln u / ln p), sehingga
// P(N >= n) = p^n tepat.
func SampleCycles(u, p float64) int {
	if p <= 0 {
		return 0
	}
	if u <= 0 {
		return MaxCycles
	}
	if u >= 1 || p >= 1 {
		if p >= 1 {
			return MaxCycles
		}
		return 0
	}
	n := int(math.Floor(math.Log(u) / math.Log(p)))
	if n > MaxCycles {
		n = MaxCycles
	}
	return n
}
