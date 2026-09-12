package simulate

import (
	"math"
	"sort"
)

// PRNG adalah generator mulberry32 - 32 bit, cepat, periode 2^32. Cukup untuk
// simulasi jadwal dan yang terpenting: deterministik untuk benih yang sama,
// baik saat dijalankan di server maupun di dalam WebAssembly di peramban.
type PRNG struct{ state uint32 }

// NewPRNG membuat generator dengan benih tertentu.
func NewPRNG(seed uint32) *PRNG { return &PRNG{state: seed} }

// Float64 mengembalikan bilangan acak seragam di [0, 1).
func (p *PRNG) Float64() float64 {
	p.state += 0x6d2b79f5
	t := p.state
	t = (t ^ (t >> 15)) * (t | 1)
	t ^= t + (t^(t>>7))*(t|61)
	return float64(t^(t>>14)) / 4294967296.0
}

// TriangularSample mengambil sampel distribusi segitiga (o, m, p) lewat
// metode transformasi invers.
func TriangularSample(rng *PRNG, o, m, p float64) float64 {
	if p <= o {
		return o
	}
	u := rng.Float64()
	fc := (m - o) / (p - o)
	if u < fc {
		return o + math.Sqrt(u*(p-o)*(m-o))
	}
	return p - math.Sqrt((1-u)*(p-o)*(p-m))
}

// BetaPERTSample mengambil sampel distribusi beta-PERT.
//
// Beta-PERT lebih setia pada semangat estimasi tiga titik ketimbang segitiga:
// bobotnya menumpuk di sekitar nilai paling mungkin dan ekornya melandai,
// bukan menyiku. Lambda mengatur seberapa percaya kita pada nilai M; lambda 4
// adalah nilai baku PERT.
func BetaPERTSample(rng *PRNG, o, m, p, lambda float64) float64 {
	if p <= o {
		return o
	}
	mu := (o + lambda*m + p) / (lambda + 2)
	var alpha, beta float64
	if math.Abs(mu-m) < 1e-9 {
		alpha = lambda/2 + 1
		beta = lambda/2 + 1
	} else {
		alpha = ((mu - o) * (2*m - o - p)) / ((m - mu) * (p - o))
		beta = (alpha * (p - mu)) / (mu - o)
	}
	if alpha <= 0 || beta <= 0 || math.IsNaN(alpha) || math.IsNaN(beta) {
		return TriangularSample(rng, o, m, p)
	}
	x := GammaSample(rng, alpha)
	y := GammaSample(rng, beta)
	if x+y == 0 {
		return m
	}
	return o + (x/(x+y))*(p-o)
}

// GammaSample mengambil sampel gamma(shape, 1) dengan metode Marsaglia-Tsang.
func GammaSample(rng *PRNG, shape float64) float64 {
	if shape < 1 {
		u := rng.Float64()
		if u <= 0 {
			u = 1e-12
		}
		return GammaSample(rng, shape+1) * math.Pow(u, 1/shape)
	}
	d := shape - 1.0/3.0
	c := 1 / math.Sqrt(9*d)
	for guard := 0; guard < 1000; guard++ {
		var x, v float64
		for {
			x = NormalSample(rng)
			v = 1 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1-0.0331*x*x*x*x {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v
		}
	}
	return d
}

// NormalSample mengambil sampel normal baku lewat transformasi Box-Muller.
func NormalSample(rng *PRNG) float64 {
	u, v := 0.0, 0.0
	for u == 0 {
		u = rng.Float64()
	}
	for v == 0 {
		v = rng.Float64()
	}
	return math.Sqrt(-2*math.Log(u)) * math.Cos(2*math.Pi*v)
}

// Quantile mengembalikan kuantil dari larik yang SUDAH terurut menaik,
// dengan interpolasi linear (metode 7 pada R).
func Quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := float64(len(sorted)-1) * q
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	return sorted[lo] + (sorted[hi]-sorted[lo])*(pos-float64(lo))
}

// Mean mengembalikan rata-rata aritmetik.
func Mean(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	var s float64
	for _, v := range values {
		s += v
	}
	return s / float64(len(values))
}

// StdDev mengembalikan simpangan baku sampel (pembagi n-1).
func StdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := Mean(values)
	var ss float64
	for _, v := range values {
		ss += (v - m) * (v - m)
	}
	return math.Sqrt(ss / float64(len(values)-1))
}

// Spearman menghitung koefisien korelasi peringkat.
//
// Dipakai untuk diagram tornado: seberapa kuat durasi satu aktivitas
// menggerakkan durasi total proyek. Peringkat lebih tahan terhadap hubungan
// non-linear ketimbang korelasi Pearson - dan hubungan antara durasi aktivitas
// non-kritis dengan durasi proyek memang sangat non-linear: nol pengaruh
// sampai float-nya habis, lalu tiba-tiba satu-untuk-satu.
func Spearman(xs, ys []float64) float64 {
	n := len(xs)
	if n != len(ys) || n < 2 {
		return 0
	}
	rx := rank(xs)
	ry := rank(ys)
	mx, my := Mean(rx), Mean(ry)
	var num, dx, dy float64
	for i := 0; i < n; i++ {
		a := rx[i] - mx
		b := ry[i] - my
		num += a * b
		dx += a * a
		dy += b * b
	}
	if dx == 0 || dy == 0 {
		return 0
	}
	return num / math.Sqrt(dx*dy)
}

// rank mengembalikan peringkat dengan rata-rata untuk nilai yang sama.
func rank(values []float64) []float64 {
	n := len(values)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return values[idx[a]] < values[idx[b]] })
	out := make([]float64, n)
	for i := 0; i < n; {
		j := i
		for j+1 < n && values[idx[j+1]] == values[idx[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			out[idx[k]] = avg
		}
		i = j + 1
	}
	return out
}
