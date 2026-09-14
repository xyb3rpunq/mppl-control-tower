package compress

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/cost"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Lembur pada jadwal yang bisa dijalankan.
//
// Crashing CPM memotong durasi aktivitas pada jaringan tanpa batas sumber daya,
// padahal jaringan itu tidak bisa dijalankan. Pada jadwal levelling berbasis isi
// pekerjaan, lembur tidak memendekkan isi pekerjaan - lembur menambah jam
// kerja: satu jam lembur seorang pekerja penuh waktu menambah 1/8 hari-orang
// pada hari itu, baik untuk pekerjaan lain yang menunggu maupun untuk
// mempercepat pekerjaannya sendiri (laju 1 + h/8). Batas sahnya menurut
// PP 35/2021 Pasal 26:
//
//	per hari   <= 4 jam
//	per minggu <= 18 jam, jadi 3,6 jam per hari kerja bila lembur setiap hari
//
// Lembur tidak ditambahkan pada hari jendela ketersediaan (periode ujian): tim
// yang sedang ujian tidak menjadi lebih tersedia karena dibayar lembur. Peran
// paruh waktu tidak diberi lembur, karena jam di atas kontrak paruh waktu belum
// tentu lembur menurut aturan.
//
// Hasilnya punya dua status bukti yang berbeda:
//
//  1. Durasi minimum dengan lembur sah maksimum dibuktikan dengan batas bawah
//     pada grid kapasitas maksimum. Setiap rencana lembur sah memakai grid yang
//     tidak lebih besar, sehingga batas itu berlaku untuk semuanya.
//  2. Biaya per durasi adalah biaya rencana yang ditemukan dengan memangkas
//     lembur dari grid maksimum selama durasinya tetap tercapai - batas atas
//     biaya minimum, bukan minimum yang terbukti.

// OvertimePoint adalah satu rencana lembur untuk satu durasi.
type OvertimePoint struct {
	Duration int
	Cost     float64                // upah lembur menurut PP 35/2021
	Hours    map[model.Role]float64 // jam lembur yang terpakai per peran
	Days     map[model.Role]int     // hari yang memakai lembur per peran
	// MaxWeek adalah jam lembur terbanyak satu orang dalam satu minggu kalender.
	MaxWeek float64
	// MaxDay adalah jam lembur terbanyak satu orang dalam satu hari.
	MaxDay      float64
	Rental      float64 // sewa & langganan pada jadwal ini
	RentalSaved float64 // dibanding jadwal tanpa lembur
	Net         float64 // Cost - RentalSaved
	Schedule    level.Result
}

// Roles mengembalikan peran yang benar-benar lembur, urut abjad.
func (p OvertimePoint) Roles() []model.Role {
	var out []model.Role
	for r, h := range p.Hours {
		if h > 0 {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// OvertimeCurve adalah kurva durasi-biaya lembur pada jadwal levelling.
type OvertimeCurve struct {
	Levelled    int          // durasi terbaik tanpa lembur
	Base        level.Result // jadwal tanpa lembur
	BaseRental  float64
	MinDuration int  // durasi terbaik dengan lembur sah maksimum
	MinProven   bool // batas bawah grid maksimum = MinDuration
	Bound       level.Bound
	HoursPerDay float64
	Persons     map[model.Role]float64
	Eligible    []model.Role // peran penuh waktu yang boleh lembur
	Excluded    []model.Role // peran paruh waktu
	Points      []OvertimePoint
}

// PointAt mengembalikan rencana untuk durasi d.
func (c OvertimeCurve) PointAt(d int) (OvertimePoint, bool) {
	for _, p := range c.Points {
		if p.Duration == d {
			return p, true
		}
	}
	return OvertimePoint{}, false
}

// overtimeSwapWindow adalah jarak hari terjauh saat lembur dipindahkan.
const overtimeSwapWindow = 5

// SustainedOvertimeHours adalah jam lembur per hari kerja yang tetap sah bila
// dilakukan setiap hari dalam lima hari kerja seminggu.
func SustainedOvertimeHours() float64 {
	return math.Min(model.OvertimeMaxDaily, model.OvertimeMaxWeekly/5)
}

// LevelledOvertime menghitung berapa hari jadwal levelling bisa dipotong dengan
// lembur sah mulai hari kerja from, dan berapa biayanya. from = 0 untuk
// perencanaan; untuk keputusan pada tanggal data, from adalah hari itu dan
// acts adalah jaringan sisa dengan tanggal rilis pada o.ReleaseOf.
func LevelledOvertime(acts []model.Activity, o level.OptimizeOptions, rates map[model.Role]float64, from int) (OvertimeCurve, error) {
	opts := o.Options
	if opts.Calendar == nil || opts.Capacity == nil {
		return OvertimeCurve{}, fmt.Errorf("compress: lembur butuh kalender dan kapasitas")
	}
	if opts.Horizon <= 0 {
		opts.Horizon = 300
	}
	if opts.Horizon > opts.Calendar.Len() {
		opts.Horizon = opts.Calendar.Len()
	}
	H := opts.Horizon
	baseGrid := level.CapacityGridWith(opts.Calendar, opts.Capacity, opts.UseWindows, H, opts.ExamFactor)

	out := OvertimeCurve{HoursPerDay: SustainedOvertimeHours(), Persons: map[model.Role]float64{}}
	for r, c := range opts.Capacity {
		switch {
		case c >= 1:
			out.Eligible = append(out.Eligible, r)
			out.Persons[r] = math.Ceil(c)
		case c > 0:
			out.Excluded = append(out.Excluded, r)
		}
	}
	// Peran termahal dipangkas lebih dulu, supaya lembur yang tersisa murah.
	sort.Slice(out.Eligible, func(i, j int) bool {
		ri, rj := rates[out.Eligible[i]], rates[out.Eligible[j]]
		if ri != rj {
			return ri > rj
		}
		return out.Eligible[i] < out.Eligible[j]
	})
	sort.Slice(out.Excluded, func(i, j int) bool { return out.Excluded[i] < out.Excluded[j] })

	// extra[r][d] adalah kapasitas lembur yang boleh ditambahkan pada hari d.
	extra := map[model.Role][]float64{}
	for _, r := range out.Eligible {
		row := make([]float64, H)
		for d := from; d < H; d++ {
			if math.Abs(baseGrid[r][d]-opts.Capacity[r]) < 1e-9 {
				row[d] = out.HoursPerDay / model.RegularHoursPerDay * out.Persons[r]
			}
		}
		extra[r] = row
	}
	// gridWith mengembalikan kapasitas dan laju maksimum untuk lembur yang aktif.
	gridWith := func(on map[model.Role][]bool) (map[model.Role][]float64, map[model.Role][]float64) {
		g := make(map[model.Role][]float64, len(baseGrid))
		rc := make(map[model.Role][]float64, len(on))
		for r, row := range baseGrid {
			nr := append([]float64(nil), row...)
			if flags, ok := on[r]; ok {
				rate := make([]float64, len(nr))
				for d := range nr {
					rate[d] = 1
					if flags[d] {
						nr[d] += extra[r][d]
						rate[d] = 1 + out.HoursPerDay/model.RegularHoursPerDay
					}
				}
				rc[r] = rate
			}
			g[r] = nr
		}
		return g, rc
	}

	rentals, _, err := cost.Rentals(acts)
	if err != nil {
		return OvertimeCurve{}, err
	}
	rentalOf := func(r level.Result) float64 {
		var c float64
		for _, rt := range rentals {
			c += rt.Rate * float64(r.Duration-r.Tasks[acts[rt.Index].ID].Start)
		}
		return c
	}

	optimize := func(g, rc map[model.Role][]float64, samples int) (level.Optimized, error) {
		q := o
		q.Options = opts
		q.CapGrid, q.RateCap = g, rc
		q.Samples = samples
		return level.Optimize(acts, q)
	}
	none, err := optimize(baseGrid, nil, o.Samples)
	if err != nil {
		return OvertimeCurve{}, err
	}
	out.Levelled, out.Base, out.BaseRental = none.Best.Duration, none.Best, rentalOf(none.Best)

	on := map[model.Role][]bool{}
	for _, r := range out.Eligible {
		flags := make([]bool, H)
		for d := range flags {
			flags[d] = extra[r][d] > 0
		}
		on[r] = flags
	}
	fg, frc := gridWith(on)
	full, err := optimize(fg, frc, o.Samples)
	if err != nil {
		return OvertimeCurve{}, err
	}
	out.MinDuration, out.Bound, out.MinProven = full.Best.Duration, full.Bound, full.Proven
	if out.MinDuration >= out.Levelled {
		return out, nil
	}

	order := full.Best.Order
	run := func(g, rc map[model.Role][]float64) (level.Result, error) {
		q := opts
		q.Lite, q.CapGrid, q.RateCap, q.Order, q.Rule = true, g, rc, order, ""
		return level.Run(acts, q)
	}
	best := map[int]OvertimePoint{}
	for T := out.MinDuration; T < out.Levelled; T++ {
		// Lembur di luar tenggat tidak pernah terpakai.
		for _, r := range out.Eligible {
			for d := T; d < H; d++ {
				on[r][d] = false
			}
		}
		// Tahap 1: lepas lembur satu peran utuh. Coba urutan yang sama dulu
		// (murah); bila memanjang, beri kesempatan urutan lain lewat Optimize.
		for _, r := range out.Eligible {
			saved := append([]bool(nil), on[r]...)
			any := false
			for d := range on[r] {
				any = any || on[r][d]
				on[r][d] = false
			}
			if !any {
				continue
			}
			g, rc := gridWith(on)
			res, err := run(g, rc)
			if err != nil {
				return OvertimeCurve{}, err
			}
			if res.Duration <= T {
				continue
			}
			alt, err := optimize(g, rc, 60)
			if err != nil {
				return OvertimeCurve{}, err
			}
			if alt.Best.Duration <= T {
				order = alt.Best.Order
				continue
			}
			on[r] = saved
		}
		// costOf menjalankan urutan tetap pada lembur yang aktif dan melaporkan
		// upahnya, atau ok = false bila tenggat T terlewati.
		costOf := func(flags map[model.Role][]bool) (level.Result, float64, bool, error) {
			sched, err := run(gridWith(flags))
			if err != nil || sched.Duration > T {
				return sched, 0, false, err
			}
			return sched, overtimePoint(sched, baseGrid, out, opts).Cost, true, nil
		}
		clone := func(src map[model.Role][]bool) map[model.Role][]bool {
			dst := make(map[model.Role][]bool, len(src))
			for r, row := range src {
				dst[r] = append([]bool(nil), row...)
			}
			return dst
		}
		// Tahap 2: lepas lembur hari demi hari dengan urutan tetap, dari belakang
		// dan dari depan. Arah memengaruhi hari mana yang tersisa; yang lebih
		// murah dipakai.
		var bestOn map[model.Role][]bool
		bestCost := math.Inf(1)
		for _, backward := range []bool{true, false} {
			cand := clone(on)
			for _, r := range out.Eligible {
				for k := 0; k < T; k++ {
					d := k
					if backward {
						d = T - 1 - k
					}
					if !cand[r][d] {
						continue
					}
					cand[r][d] = false
					_, _, ok, err := costOf(cand)
					if err != nil {
						return OvertimeCurve{}, err
					}
					if !ok {
						cand[r][d] = true
					}
				}
			}
			_, c, ok, err := costOf(cand)
			if err != nil {
				return OvertimeCurve{}, err
			}
			if ok && c < bestCost {
				bestOn, bestCost = cand, c
			}
		}
		if bestOn == nil {
			return OvertimeCurve{}, fmt.Errorf("compress: rencana lembur untuk %d hari tidak ditemukan", T)
		}
		// Tahap 3: pencarian lokal - lepas satu hari, atau pindahkan lembur satu
		// hari ke hari lain di dekatnya, selama tenggat tetap tercapai dan
		// upahnya turun. Hari terakhir yang hanya terpakai sebagian sering lebih
		// murah daripada hari penuh di tengah.
		for pass := 0; pass < 20; pass++ {
			improved := false
			for _, r := range out.Eligible {
				for d := 0; d < T; d++ {
					if !bestOn[r][d] {
						continue
					}
					bestOn[r][d] = false
					if _, c, ok, err := costOf(bestOn); err != nil {
						return OvertimeCurve{}, err
					} else if ok && c < bestCost-0.5 {
						bestCost, improved = c, true
						continue
					}
					moved := false
					for e := d - overtimeSwapWindow; e <= d+overtimeSwapWindow && !moved; e++ {
						if e < from || e >= T || e == d || bestOn[r][e] || extra[r][e] == 0 {
							continue
						}
						bestOn[r][e] = true
						_, c, ok, err := costOf(bestOn)
						if err != nil {
							return OvertimeCurve{}, err
						}
						if ok && c < bestCost-0.5 {
							bestCost, improved, moved = c, true, true
						} else {
							bestOn[r][e] = false
						}
					}
					if !moved {
						bestOn[r][d] = true
					}
				}
			}
			if !improved {
				break
			}
		}
		on = bestOn
		sched, err := run(gridWith(on))
		if err != nil {
			return OvertimeCurve{}, err
		}
		if sched.Duration > T {
			return OvertimeCurve{}, fmt.Errorf("compress: rencana lembur untuk %d hari memanjang ke %d", T, sched.Duration)
		}
		pt := overtimePoint(sched, baseGrid, out, opts)
		pt.Rental = rentalOf(sched)
		pt.RentalSaved = out.BaseRental - pt.Rental
		pt.Net = pt.Cost - pt.RentalSaved
		if prev, ok := best[pt.Duration]; !ok || pt.Cost < prev.Cost {
			best[pt.Duration] = pt
		}
	}
	for _, p := range best {
		out.Points = append(out.Points, p)
	}
	sort.Slice(out.Points, func(i, j int) bool { return out.Points[i].Duration > out.Points[j].Duration })
	return out, nil
}

// overtimePoint menghitung jam dan upah lembur dari pemakaian kapasitas
// jadwal di atas kapasitas normal, atau dari kelebihan di atas alokasi rencana
// (level.Result.Excess) bila lebih besar. Lembur dibagi rata ke orang dalam
// peran itu - pembagian termurah, karena jam pertama paling murah.
func overtimePoint(sched level.Result, baseGrid map[model.Role][]float64, c OvertimeCurve, opts level.Options) OvertimePoint {
	pt := OvertimePoint{Duration: sched.Duration, Hours: map[model.Role]float64{}, Days: map[model.Role]int{}, Schedule: sched}
	for _, r := range c.Eligible {
		n := c.Persons[r]
		week := map[string]float64{}
		for d := 0; d < sched.Duration && d < len(sched.Usage[r]); d++ {
			over := sched.Usage[r][d] - baseGrid[r][d]
			if ex := sched.Excess[r]; d < len(ex) && ex[d] > over {
				over = ex[d]
			}
			if over <= 1e-9 {
				continue
			}
			perPerson := over * model.RegularHoursPerDay / n
			pt.Hours[r] += over * model.RegularHoursPerDay
			pt.Days[r]++
			pt.Cost += n * model.OvertimeUnits(perPerson) * model.RateCard[r] / model.RegularHoursPerDay
			pt.MaxDay = math.Max(pt.MaxDay, perPerson)
			y, w := opts.Calendar.Date(d).ISOWeek()
			key := fmt.Sprintf("%d-%02d", y, w)
			week[key] += perPerson
			pt.MaxWeek = math.Max(pt.MaxWeek, week[key])
		}
	}
	return pt
}
