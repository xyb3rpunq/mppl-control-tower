package simulate

import (
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Sampler mengambil sampel durasi seluruh aktivitas dalam satu iterasi.
//
// Setiap sampel dibuat lewat transformasi invers: bilangan seragam u diubah
// menjadi durasi x = F^-1(u). Pendekatan ini dipilih karena satu alasan -
// korelasi. Dengan transformasi invers, korelasi antar-aktivitas cukup
// disuntikkan pada tingkat u lewat kopula Gauss, sementara sebaran marginal
// setiap aktivitas tetap persis beta-PERT atau segitiga seperti yang
// diestimasi. Metode gamma yang dipakai sebelumnya tidak punya pegangan untuk
// itu.
//
// Model korelasinya: setiap peran punya satu faktor kinerja laten per iterasi
// (z_peran), dan setiap aktivitas mengikuti faktor peran dominannya:
//
//	u = Phi(rho * z_peran + sqrt(1 - rho^2) * epsilon)
//
// Maknanya konkret: kalau Backend Developer sedang lambat pada iterasi ini,
// ia lambat pada SEMUA aktivitasnya, bukan hanya pada satu. Dengan rho = 0
// model ini kembali menjadi sampel independen.
type Sampler struct {
	acts  []model.Activity
	index map[string]int
	roles []model.Role
	dom   []int        // indeks peran dominan per aktivitas; -1 bila tanpa tim
	quant []*betaTable // tabel kuantil beta-PERT per aktivitas; nil bila tetap
	z     []float64    // faktor laten per peran dari Draw terakhir
	u     []float64    // bilangan seragam per aktivitas dari Draw terakhir
}

// NewSampler menyiapkan tabel kuantil untuk seluruh aktivitas. Tabel yang
// estimasi tiga titiknya sama dipakai bersama.
func NewSampler(acts []model.Activity) *Sampler {
	s := &Sampler{
		acts:  acts,
		index: make(map[string]int, len(acts)),
		dom:   make([]int, len(acts)),
		quant: make([]*betaTable, len(acts)),
		u:     make([]float64, len(acts)),
	}
	roleIdx := map[model.Role]int{}
	cache := map[[3]int]*betaTable{}
	for i, a := range acts {
		s.index[a.ID] = i
		s.dom[i] = -1
		best := -1.0
		for _, slot := range a.Team {
			if _, ok := roleIdx[slot.Role]; !ok {
				roleIdx[slot.Role] = len(s.roles)
				s.roles = append(s.roles, slot.Role)
			}
			if slot.Alloc > best {
				best = slot.Alloc
				s.dom[i] = roleIdx[slot.Role]
			}
		}
		if a.Milestone || a.Pessimistic <= a.Optimistic {
			continue
		}
		key := [3]int{a.Optimistic, a.Duration, a.Pessimistic}
		tbl, ok := cache[key]
		if !ok {
			tbl = newBetaTable(float64(a.Optimistic), float64(a.Duration), float64(a.Pessimistic))
			cache[key] = tbl
		}
		s.quant[i] = tbl
	}
	s.z = make([]float64, len(s.roles))
	return s
}

// Factor mengembalikan faktor kinerja laten sebuah peran pada Draw terakhir,
// atau 0 bila peran itu tidak dikenal. Nilai positif berarti peran itu sedang
// lambat pada iterasi ini. Kopula risiko memakainya agar risiko yang
// bersumber dari peran itu ikut berkorelasi dengan durasi aktivitasnya.
func (s *Sampler) Factor(r model.Role) float64 {
	for i, role := range s.roles {
		if role == r {
			return s.z[i]
		}
	}
	return 0
}

// U mengembalikan bilangan seragam aktivitas i pada Draw terakhir.
func (s *Sampler) U(i int) float64 { return s.u[i] }

// Value mengembalikan kuantil kontinu durasi aktivitas i untuk bilangan
// seragam u, sebelum dibulatkan ke hari.
func (s *Sampler) Value(i int, u float64, distribution string) float64 {
	a := s.acts[i]
	if a.Milestone {
		return 0
	}
	if a.Pessimistic <= a.Optimistic {
		return float64(a.Duration)
	}
	if distribution == "triangular" {
		return triangularInverse(u, float64(a.Optimistic), float64(a.Duration), float64(a.Pessimistic))
	}
	return s.quant[i].quantile(u)
}

// CDF mengembalikan P(durasi aktivitas i <= x) menurut sebaran beta-PERT-nya.
// Prakiraan berjalan memakainya untuk menarik durasi bersyarat: aktivitas
// yang sudah berjalan e hari pasti tidak berdurasi kurang dari e.
func (s *Sampler) CDF(i int, x float64) float64 {
	a := s.acts[i]
	o, p := float64(a.Optimistic), float64(a.Pessimistic)
	if a.Milestone || p <= o {
		if x >= float64(a.Duration) {
			return 1
		}
		return 0
	}
	alpha, beta := BetaPERTParams(o, float64(a.Duration), p)
	return RegIncBeta(alpha, beta, (x-o)/(p-o))
}

// PERTMean mengembalikan rerata beta-PERT aktivitas i: (O + 4M + P) / 6.
func (s *Sampler) PERTMean(i int) float64 {
	a := s.acts[i]
	if a.Milestone {
		return 0
	}
	if a.Pessimistic <= a.Optimistic {
		return float64(a.Duration)
	}
	return (float64(a.Optimistic) + 4*float64(a.Duration) + float64(a.Pessimistic)) / 6
}

// Index mengembalikan indeks aktivitas.
func (s *Sampler) Index(id string) int { return s.index[id] }

// Roles mengembalikan peran yang dikenali sampler, dalam urutan indeksnya.
func (s *Sampler) Roles() []model.Role { return s.roles }

// DominantRole mengembalikan peran dominan sebuah aktivitas, atau string kosong.
func (s *Sampler) DominantRole(i int) model.Role {
	if s.dom[i] < 0 {
		return ""
	}
	return s.roles[s.dom[i]]
}

// Draw mengisi out dengan durasi sampel satu iterasi.
//
// Urutan konsumsi bilangan acak dikunci: faktor laten seluruh peran lebih
// dulu, lalu satu epsilon per aktivitas yang tidak tetap. Urutan yang tetap
// inilah yang membuat rho = 0 menghasilkan sampel yang sama persis di mana
// pun sampler ini dipakai.
func (s *Sampler) Draw(rng *PRNG, rho float64, distribution string, out []int) {
	if rho < 0 {
		rho = 0
	}
	if rho > 0.999 {
		rho = 0.999
	}
	z := s.z
	for i := range z {
		z[i] = NormalSample(rng)
	}
	idio := math.Sqrt(1 - rho*rho)
	for i, a := range s.acts {
		if a.Milestone {
			out[i] = 0
			s.u[i] = 0
			continue
		}
		if a.Pessimistic <= a.Optimistic {
			out[i] = a.Duration
			s.u[i] = 0.5
			continue
		}
		latent := NormalSample(rng) * idio
		if s.dom[i] >= 0 {
			latent += rho * z[s.dom[i]]
		}
		u := stdNormalCDF(latent)
		s.u[i] = u
		v := s.Value(i, u, distribution)
		d := int(math.Round(v))
		if d < 1 {
			d = 1
		}
		out[i] = d
	}
}

// BetaPERTParams mengembalikan parameter bentuk beta-PERT baku:
//
//	alpha = 1 + 4(M - O)/(P - O)      beta = 1 + 4(P - M)/(P - O)
//
// Parameterisasi ini menjamin rerata (O + 4M + P)/6 dan kedua parameter
// selalu >= 1, sehingga densitasnya terbatas dan tidak perlu jalur cadangan.
func BetaPERTParams(o, m, p float64) (alpha, beta float64) {
	span := p - o
	if span <= 0 {
		return 1, 1
	}
	return 1 + 4*(m-o)/span, 1 + 4*(p-m)/span
}

// betaTable menyimpan CDF beta-PERT pada grid rapat untuk inversi cepat.
type betaTable struct {
	o, p float64
	cdf  []float64 // CDF pada titik x_k = k/(n-1), dinormalisasi ke [0,1]
}

const betaGrid = 2049

func newBetaTable(o, m, p float64) *betaTable {
	a, b := BetaPERTParams(o, m, p)
	t := &betaTable{o: o, p: p, cdf: make([]float64, betaGrid)}
	for k := 0; k < betaGrid; k++ {
		t.cdf[k] = RegIncBeta(a, b, float64(k)/float64(betaGrid-1))
	}
	t.cdf[0], t.cdf[betaGrid-1] = 0, 1
	return t
}

// quantile mengembalikan F^-1(u) dengan pencarian biner dan interpolasi linear.
func (t *betaTable) quantile(u float64) float64 {
	if u <= 0 {
		return t.o
	}
	if u >= 1 {
		return t.p
	}
	k := sort.SearchFloat64s(t.cdf, u)
	if k <= 0 {
		return t.o
	}
	lo, hi := t.cdf[k-1], t.cdf[k]
	frac := 0.0
	if hi > lo {
		frac = (u - lo) / (hi - lo)
	}
	x := (float64(k-1) + frac) / float64(betaGrid-1)
	return t.o + x*(t.p-t.o)
}

// BetaPERTQuantile mengembalikan kuantil beta-PERT tanpa tabel - lebih lambat,
// dipakai untuk uji dan contoh hitung, bukan di dalam simulasi.
func BetaPERTQuantile(o, m, p, u float64) float64 {
	return newBetaTable(o, m, p).quantile(u)
}

// RegIncBeta menghitung fungsi beta tak lengkap teregularisasi I_x(a, b),
// yaitu CDF sebaran Beta(a, b) pada x. Algoritma pecahan berlanjut Lentz
// (Numerical Recipes, bagian 6.4).
func RegIncBeta(a, b, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	la, _ := math.Lgamma(a)
	lb, _ := math.Lgamma(b)
	lab, _ := math.Lgamma(a + b)
	front := math.Exp(lab - la - lb + a*math.Log(x) + b*math.Log(1-x))
	if x < (a+1)/(a+b+2) {
		return front * betaCF(a, b, x) / a
	}
	return 1 - front*betaCF(b, a, 1-x)/b
}

func betaCF(a, b, x float64) float64 {
	const (
		maxIter = 300
		epsCF   = 3e-14
		fpMin   = 1e-300
	)
	qab, qap, qam := a+b, a+1, a-1
	c := 1.0
	d := 1 - qab*x/qap
	if math.Abs(d) < fpMin {
		d = fpMin
	}
	d = 1 / d
	h := d
	for m := 1; m <= maxIter; m++ {
		fm := float64(m)
		m2 := 2 * fm
		aa := fm * (b - fm) * x / ((qam + m2) * (a + m2))
		d = 1 + aa*d
		if math.Abs(d) < fpMin {
			d = fpMin
		}
		c = 1 + aa/c
		if math.Abs(c) < fpMin {
			c = fpMin
		}
		d = 1 / d
		h *= d * c
		aa = -(a + fm) * (qab + fm) * x / ((a + m2) * (qap + m2))
		d = 1 + aa*d
		if math.Abs(d) < fpMin {
			d = fpMin
		}
		c = 1 + aa/c
		if math.Abs(c) < fpMin {
			c = fpMin
		}
		d = 1 / d
		del := d * c
		h *= del
		if math.Abs(del-1) < epsCF {
			break
		}
	}
	return h
}

// triangularInverse adalah invers CDF sebaran segitiga.
func triangularInverse(u, o, m, p float64) float64 {
	if p <= o {
		return o
	}
	fc := (m - o) / (p - o)
	if u < fc {
		return o + math.Sqrt(u*(p-o)*(m-o))
	}
	return p - math.Sqrt((1-u)*(p-o)*(p-m))
}

func stdNormalCDF(z float64) float64 { return 0.5 * (1 + math.Erf(z/math.Sqrt2)) }
