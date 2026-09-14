package compress

import (
	"fmt"
	"math"

	"github.com/xyb3rpunq/mppl-control-tower/internal/cost"
	"github.com/xyb3rpunq/mppl-control-tower/internal/lp"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// Crash serakah memotong satu hari pada setiap langkah dan tidak pernah
// membatalkan potongan lama. Itu cepat, tetapi tidak dijamin optimum: potongan
// yang dulu perlu bisa menjadi mubazir begitu jalur paralel ikut dipotong.
//
// Pemecahan eksaknya adalah pemrograman linear (Kelley, 1961). Untuk tenggat T:
//
//	minimalkan   sum c_i x_i
//	dengan       t_j >= t_i + d_i - x_i      untuk setiap relasi FS i -> j
//	             t_i + d_i - x_i <= E <= T   untuk setiap aktivitas i
//	             0 <= x_i <= d_i - crash_i,  t_i >= 0
//
// Matriks batasannya adalah matriks jaringan yang unimodular total, sehingga
// titik sudut optimum simpleks selalu bilangan bulat - potongan hari yang
// dihasilkan LP langsung bisa dijalankan, tanpa pembulatan yang merusak
// optimalitas. Solusinya tetap diperiksa ulang dengan CPM.
//
// Dengan biaya sewa (cost.Rental), tujuan yang sama diperluas menjadi biaya
// TOTAL: sum c_i x_i + sum tarif_k (E - t_pembeli_k). Titik terendah kurva
// biaya total adalah durasi yang paling murah secara keseluruhan - analisis
// time-cost trade-off klasik pada Modul 4.

// ExactPoint adalah satu titik kurva waktu-biaya eksak.
type ExactPoint struct {
	Duration int
	// CrashCost adalah biaya crashing minimum untuk mencapai Duration.
	CrashCost float64
	// Greedy adalah biaya crashing serakah pada durasi yang sama, atau -1
	// bila serakah tidak mencapai durasi ini.
	Greedy float64
	Cuts   map[string]int
	// Pada solusi biaya total minimum:
	Rental     float64 // biaya sewa & langganan
	TotalCrash float64 // bagian biaya crashing
	Total      float64 // TotalCrash + Rental
}

// TradeOff adalah kurva waktu-biaya eksak beserta titik biaya total terendah.
type TradeOff struct {
	Normal   int
	ExactMin int
	Points   []ExactPoint // dari durasi normal menurun ke ExactMin

	// GreedyOptimal bernilai true bila serakah mencapai biaya eksak pada
	// setiap durasi yang dicapainya.
	GreedyOptimal   bool
	MaxGreedyExcess float64
	// GreedyMissed adalah hari yang bisa dipotong LP tetapi tidak oleh serakah.
	GreedyMissed int

	RentalRate float64
	Optimum    ExactPoint
}

// PointAt mengembalikan titik untuk durasi tertentu.
func (t TradeOff) PointAt(d int) (ExactPoint, bool) {
	for _, p := range t.Points {
		if p.Duration == d {
			return p, true
		}
	}
	return ExactPoint{}, false
}

// BreakEvenPerDay mengembalikan nilai minimum satu hari percepatan agar
// mempercepat proyek ke durasi d tidak merugi: (Total(d) - Total(normal)) / hari.
func (t TradeOff) BreakEvenPerDay(d int) float64 {
	p, ok := t.PointAt(d)
	if !ok || len(t.Points) == 0 || d >= t.Normal {
		return 0
	}
	return (p.Total - t.Points[0].Total) / float64(t.Normal-d)
}

// Exact menghitung kurva crashing eksak lewat LP dan membandingkannya dengan
// kurva serakah.
func Exact(acts []model.Activity, rates map[model.Role]float64, greedy Curve) (TradeOff, error) {
	base, err := schedule.Compute(acts, schedule.Options{})
	if err != nil {
		return TradeOff{}, err
	}
	rentals, _, err := cost.Rentals(acts)
	if err != nil {
		return TradeOff{}, err
	}
	out := TradeOff{Normal: base.Duration, GreedyOptimal: true, RentalRate: cost.DailyRate(rentals)}

	n := len(acts)
	idx := make(map[string]int, n)
	for i, a := range acts {
		idx[a.ID] = i
	}
	plans := make([]model.CrashPlan, n)
	for i, a := range acts {
		plans[i] = a.Crash(rates)
	}

	build := func(T int, withRental bool) ([]float64, [][]float64, []float64, error) {
		nv := 2*n + 1
		E := 2 * n
		var A [][]float64
		var b []float64
		row := func() []float64 { return make([]float64, nv) }
		for j, a := range acts {
			for _, p := range a.Pred {
				i := idx[p.ID]
				r := row()
				switch p.Type {
				case "", "FS":
					r[i], r[n+i], r[j] = 1, -1, -1
					A, b = append(A, r), append(b, -float64(acts[i].Duration)-float64(p.Lag))
				case "SS":
					r[i], r[j] = 1, -1
					A, b = append(A, r), append(b, -float64(p.Lag))
				default:
					return nil, nil, nil, fmt.Errorf("compress: relasi %s belum didukung LP crashing", p.Type)
				}
			}
			r := row()
			r[j], r[n+j], r[E] = 1, -1, -1
			A, b = append(A, r), append(b, -float64(a.Duration))
			if len(a.Pred) == 0 {
				// Tanggal mulai proyek dikunci Project Charter. Tanpa batasan
				// ini LP bisa "menghemat" sewa dengan menggeser seluruh proyek
				// ke kanan - yang tidak mungkin dilakukan.
				r := row()
				r[j] = 1
				A, b = append(A, r), append(b, 0)
			}
		}
		r := row()
		r[E] = 1
		A, b = append(A, r), append(b, float64(T))
		if withRental {
			// Biaya total diukur TEPAT pada durasi T. Tanpa ini LP boleh
			// selesai lebih cepat dari T bila itu lebih murah, dan titik
			// kurvanya tidak lagi mewakili durasi yang tertulis.
			r := row()
			r[E] = -1
			A, b = append(A, r), append(b, -float64(T))
		}
		for i := range acts {
			r := row()
			r[n+i] = 1
			u := 0.0
			if plans[i].Allowed {
				u = float64(acts[i].Duration - plans[i].CrashDur)
			}
			A, b = append(A, r), append(b, u)
		}
		c := make([]float64, nv)
		for i := range acts {
			if plans[i].Allowed {
				c[n+i] = plans[i].SlopePerDay
			}
		}
		if withRental {
			for _, rt := range rentals {
				c[E] += rt.Rate
				c[rt.Index] -= rt.Rate
			}
		}
		return c, A, b, nil
	}

	verify := func(x []float64, T int) (map[string]int, error) {
		cuts := map[string]int{}
		dur := make(map[string]int, n)
		for i, a := range acts {
			k := int(math.Round(x[n+i]))
			if math.Abs(x[n+i]-float64(k)) > 1e-6 {
				return nil, fmt.Errorf("compress: LP memberi potongan pecahan %v pada %s", x[n+i], a.ID)
			}
			dur[a.ID] = a.Duration - k
			if k > 0 {
				cuts[a.ID] = k
			}
		}
		r, err := schedule.Compute(acts, schedule.Options{DurationOf: func(a model.Activity) int { return dur[a.ID] }})
		if err != nil {
			return nil, err
		}
		if r.Duration > T {
			return nil, fmt.Errorf("compress: solusi LP untuk %d hari ternyata %d hari di CPM", T, r.Duration)
		}
		return cuts, nil
	}

	for T := base.Duration; T > 0; T-- {
		c, A, b, err := build(T, false)
		if err != nil {
			return TradeOff{}, err
		}
		pure := lp.Minimize(c, A, b)
		if pure.Status != lp.Optimal {
			break
		}
		cuts, err := verify(pure.X, T)
		if err != nil {
			return TradeOff{}, err
		}
		pt := ExactPoint{Duration: T, CrashCost: clean(pure.Objective), Greedy: -1, Cuts: cuts}
		k := base.Duration - T
		switch {
		case k == 0:
			pt.Greedy = 0
		case k <= len(greedy.Steps):
			pt.Greedy = greedy.Steps[k-1].TotalCost
		}
		if pt.Greedy >= 0 {
			if ex := pt.Greedy - pt.CrashCost; ex > 0.5 {
				out.GreedyOptimal = false
				if ex > out.MaxGreedyExcess {
					out.MaxGreedyExcess = ex
				}
			}
		} else {
			out.GreedyMissed++
		}

		c2, A2, b2, _ := build(T, true)
		tot := lp.Minimize(c2, A2, b2)
		if tot.Status != lp.Optimal {
			return TradeOff{}, fmt.Errorf("compress: LP biaya total untuk %d hari %v", T, tot.Status)
		}
		for _, rt := range rentals {
			pt.Rental += rt.Rate * (tot.X[2*n] - tot.X[rt.Index])
		}
		pt.Total = clean(tot.Objective)
		pt.TotalCrash = clean(pt.Total - pt.Rental)
		out.Points = append(out.Points, pt)
		out.ExactMin = T
	}
	if len(out.Points) == 0 {
		return TradeOff{}, fmt.Errorf("compress: jaringan tidak layak bahkan pada durasi normal")
	}
	out.Optimum = out.Points[0]
	for _, p := range out.Points {
		if p.Total < out.Optimum.Total-0.5 {
			out.Optimum = p
		}
	}
	return out, nil
}

// clean membuang galat pembulatan simpleks di bawah satu per sejuta rupiah.
func clean(v float64) float64 {
	if math.Abs(v) < 1e-6 {
		return 0
	}
	return math.Round(v*1e6) / 1e6
}
