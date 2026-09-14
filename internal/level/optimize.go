package level

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// OptimizeOptions mengatur pencarian jadwal levelling terbaik.
type OptimizeOptions struct {
	Options
	// Samples adalah jumlah daftar aktivitas acak berbias yang dicoba.
	Samples int
	// Seed membuat sampel acak bisa diulang persis.
	Seed uint32
	// Rounds adalah batas putaran justifikasi per jadwal.
	Rounds int
}

// RuleRun adalah hasil satu aturan prioritas, sebelum dan sesudah justifikasi.
type RuleRun struct {
	Rule      Rule
	Duration  int
	Justified int
}

// Optimized adalah hasil pencarian jadwal levelling terbaik.
type Optimized struct {
	Best Result
	// Source menjelaskan dari mana jadwal terbaik berasal, mis. "GRPW" atau
	// "sampel" (daftar acak berbias).
	Source    string
	Justified bool // jadwal terbaik lahir dari justifikasi, bukan SGS polos

	// Baseline adalah SGS polos dengan aturan LST, pembanding bagi perbaikan.
	Baseline Result
	Rules    []RuleRun

	Samples     int
	SampleBest  int
	SampleWorst int
	// SampleCounts memetakan durasi ke jumlah sampel (setelah justifikasi).
	SampleCounts map[int]int

	Bound  Bound
	Gap    int  // Best.Duration - Bound.Value
	Proven bool // Gap == 0: tidak ada jadwal yang lebih pendek dalam model ini
}

// Optimize mencari jadwal levelling terpendek dengan tiga cara berlapis:
//
//  1. SGS dengan enam aturan prioritas baku;
//  2. SGS dengan daftar aktivitas acak berbias (regret-based biased random
//     sampling): aktivitas dengan latest start lebih kecil lebih mungkin
//     dipilih, tetapi tidak selalu;
//  3. setiap jadwal dari (1) dan (2) diperbaiki dengan justifikasi maju-mundur
//     (Valls, Ballestín & Quintanilla, 2005).
//
// Jadwal terbaik kemudian dibandingkan dengan LowerBound. Pencarian ini tetap
// heuristik; yang membuatnya jujur adalah batas bawahnya.
func Optimize(acts []model.Activity, o OptimizeOptions) (Optimized, error) {
	if o.Samples <= 0 {
		o.Samples = 300
	}
	if o.Rounds <= 0 {
		o.Rounds = 8
	}
	if o.Seed == 0 {
		o.Seed = 20210801
	}
	base := o.Options
	base.Lite = true
	base.Order = nil
	if base.Calendar == nil || base.Capacity == nil {
		return Optimized{}, fmt.Errorf("level: kalender dan kapasitas wajib diisi")
	}
	if base.Horizon <= 0 {
		base.Horizon = 300
	}
	if base.Horizon > base.Calendar.Len() {
		base.Horizon = base.Calendar.Len()
	}
	if base.CapGrid == nil {
		base.CapGrid = CapacityGridWith(base.Calendar, base.Capacity, base.UseWindows, base.Horizon, base.ExamFactor)
	}

	out := Optimized{SampleCounts: map[int]int{}}
	var bestOrder []string
	bestDur := math.MaxInt32
	consider := func(r Result, source string, justified bool) {
		if r.Duration < bestDur {
			bestDur = r.Duration
			bestOrder = r.Order
			out.Source = source
			out.Justified = justified
		}
	}

	for _, rule := range Rules {
		opt := base
		opt.Rule = rule
		r, err := Run(acts, opt)
		if err != nil {
			return Optimized{}, err
		}
		j := Justify(acts, base, r, o.Rounds)
		out.Rules = append(out.Rules, RuleRun{Rule: rule, Duration: r.Duration, Justified: j.Duration})
		consider(r, string(rule), false)
		consider(j, string(rule), j.Duration < r.Duration)
	}

	rng := newRand(o.Seed)
	out.Samples = o.Samples
	out.SampleBest = math.MaxInt32
	for s := 0; s < o.Samples; s++ {
		order, err := biasedOrder(acts, base, rng)
		if err != nil {
			return Optimized{}, err
		}
		opt := base
		opt.Order = order
		r, err := Run(acts, opt)
		if err != nil {
			return Optimized{}, err
		}
		j := Justify(acts, base, r, o.Rounds)
		out.SampleCounts[j.Duration]++
		if j.Duration < out.SampleBest {
			out.SampleBest = j.Duration
		}
		if j.Duration > out.SampleWorst {
			out.SampleWorst = j.Duration
		}
		consider(r, "sampel", false)
		consider(j, "sampel", j.Duration < r.Duration)
	}

	// Jadwal terbaik dibangun ulang dalam mode lengkap agar Gantt dan
	// histogram punya rincian penyebab per aktivitas.
	full := o.Options
	full.Lite = false
	full.Rule = ""
	full.Order = bestOrder
	best, err := Run(acts, full)
	if err != nil {
		return Optimized{}, err
	}
	out.Best = best

	baseFull := o.Options
	baseFull.Lite = false
	baseFull.Rule = RuleLST
	baseFull.Order = nil
	if out.Baseline, err = Run(acts, baseFull); err != nil {
		return Optimized{}, err
	}

	if out.Bound, err = LowerBound(acts, o.Options); err != nil {
		return Optimized{}, err
	}
	out.Gap = out.Best.Duration - out.Bound.Value
	out.Proven = out.Gap == 0
	return out, nil
}

// Justify memperbaiki sebuah jadwal dengan justifikasi ganda: aktivitas
// digeser sekanan mungkin (urutan selesai menurun, di atas jaringan dan
// kalender yang dibalik), lalu dijadwalkan ulang sekiri mungkin mengikuti
// urutan mulainya. Pemadatan dua arah ini sering menutup celah yang
// ditinggalkan SGS. Putaran berhenti bila durasi tidak lagi membaik.
//
// Hasil akhir selalu keluaran SGS maju yang sah; putaran mundur hanya dipakai
// untuk menyusun daftar aktivitas. Jaringan dengan relasi selain FS, atau
// dengan tanggal rilis, dikembalikan apa adanya.
func Justify(acts []model.Activity, opts Options, r Result, rounds int) Result {
	if opts.ReleaseOf != nil {
		return r
	}
	for _, a := range acts {
		for _, p := range a.Pred {
			if p.Type != "" && p.Type != "FS" {
				return r
			}
		}
	}
	best := r
	for round := 0; round < rounds; round++ {
		next, ok := justifyOnce(acts, opts, best)
		if !ok || next.Duration >= best.Duration {
			break
		}
		best = next
	}
	return best
}

func justifyOnce(acts []model.Activity, opts Options, r Result) (Result, bool) {
	D := r.Duration
	if D <= 0 {
		return r, false
	}
	durationOf := opts.DurationOf
	if durationOf == nil {
		durationOf = func(a model.Activity) int { return a.Duration }
	}

	// Jaringan terbalik: penerus menjadi pendahulu.
	succ := map[string][]model.Predecessor{}
	for _, a := range acts {
		for _, p := range a.Pred {
			succ[p.ID] = append(succ[p.ID], model.Predecessor{ID: a.ID, Type: "FS", Lag: p.Lag})
		}
	}
	rev := make([]model.Activity, len(acts))
	durs := make(map[string]int, len(acts))
	for i, a := range acts {
		b := a
		b.Pred = succ[a.ID]
		rev[i] = b
		durs[a.ID] = durationOf(a)
	}

	// Kalender terbalik sepanjang D hari: hari ke-k mundur = hari D-1-k maju.
	grid := make(map[model.Role][]float64, len(r.Cap))
	for role, row := range r.Cap {
		rr := make([]float64, D)
		for k := 0; k < D && D-1-k < len(row); k++ {
			rr[k] = row[D-1-k]
		}
		grid[role] = rr
	}

	order := append([]string(nil), r.Order...)
	sort.SliceStable(order, func(i, j int) bool {
		ti, tj := r.Tasks[order[i]], r.Tasks[order[j]]
		if ti.Finish != tj.Finish {
			return ti.Finish > tj.Finish
		}
		return ti.Start > tj.Start
	})
	back := opts
	back.Lite = true
	back.Horizon = D
	back.CapGrid = grid
	back.Order = order
	back.Rule = ""
	back.DurationOf = func(a model.Activity) int { return durs[a.ID] }
	rb, err := Run(rev, back)
	if err != nil {
		return r, false
	}

	// Mulai asli = D - selesai mundur. Urutkan menurut mulai asli.
	fwd := append([]string(nil), rb.Order...)
	sort.SliceStable(fwd, func(i, j int) bool {
		si, sj := D-rb.Tasks[fwd[i]].Finish, D-rb.Tasks[fwd[j]].Finish
		if si != sj {
			return si < sj
		}
		return D-rb.Tasks[fwd[i]].Start < D-rb.Tasks[fwd[j]].Start
	})
	front := opts
	front.Lite = true
	front.Order = fwd
	front.Rule = ""
	rf, err := Run(acts, front)
	if err != nil {
		return r, false
	}
	return rf, true
}

// biasedOrder menyusun satu daftar aktivitas yang layak-urutan secara acak.
// Peluang aktivitas layak j terpilih sebanding dengan (LSmaks - LS_j + 1)^2:
// aktivitas yang lebih mendesak lebih sering didahulukan, tetapi urutan lain
// tetap punya kesempatan.
func biasedOrder(acts []model.Activity, opts Options, rng *randSource) ([]string, error) {
	plan, err := scheduleFor(acts, opts)
	if err != nil {
		return nil, err
	}
	done := make(map[string]bool, len(acts))
	order := make([]string, 0, len(acts))
	var elig []string
	var weights []float64
	for len(order) < len(acts) {
		elig = elig[:0]
		weights = weights[:0]
		maxLS := math.MinInt32
		for _, a := range acts {
			if done[a.ID] {
				continue
			}
			ok := true
			for _, p := range a.Pred {
				if !done[p.ID] {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			elig = append(elig, a.ID)
			if ls := plan.Task(a.ID).LS; ls > maxLS {
				maxLS = ls
			}
		}
		if len(elig) == 0 {
			return nil, fmt.Errorf("level: jaringan tidak sah saat menyusun daftar acak")
		}
		var total float64
		for _, id := range elig {
			w := float64(maxLS-plan.Task(id).LS) + 1
			w *= w
			weights = append(weights, w)
			total += w
		}
		u := rng.float64() * total
		pick := elig[len(elig)-1]
		for i, w := range weights {
			if u < w {
				pick = elig[i]
				break
			}
			u -= w
		}
		done[pick] = true
		order = append(order, pick)
	}
	return order, nil
}

// randSource adalah mulberry32 - generator yang sama dengan paket simulate,
// ditulis ulang di sini agar paket level tidak bergantung pada simulate.
type randSource struct{ state uint32 }

func newRand(seed uint32) *randSource { return &randSource{state: seed} }

func (r *randSource) float64() float64 {
	r.state += 0x6D2B79F5
	z := r.state
	z = (z ^ (z >> 15)) * (z | 1)
	z ^= z + (z^(z>>7))*(z|61)
	z ^= z >> 14
	return float64(z) / 4294967296.0
}
