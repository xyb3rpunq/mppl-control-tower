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
// pekerjaan, lembur tidak memendekkan pekerjaan - lembur menambah kapasitas
// peran: satu jam lembur seorang pekerja penuh waktu menambah 1/8 hari-orang
// pada hari itu. Batas sahnya menurut PP 35/2021 Pasal 26:
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

// SustainedOvertimeHours adalah jam lembur per hari kerja yang tetap sah bila
// dilakukan setiap hari dalam lima hari kerja seminggu.
func SustainedOvertimeHours() float64 {
	return math.Min(model.OvertimeMaxDaily, model.OvertimeMaxWeekly/5)
}

// LevelledOvertime menghitung berapa hari jadwal levelling bisa dipotong dengan
// lembur sah, dan berapa biayanya.
func LevelledOvertime(acts []model.Activity, o level.OptimizeOptions, rates map[model.Role]float64) (OvertimeCurve, error) {
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
		for d := 0; d < H; d++ {
			if math.Abs(baseGrid[r][d]-opts.Capacity[r]) < 1e-9 {
				row[d] = out.HoursPerDay / model.RegularHoursPerDay * out.Persons[r]
			}
		}
		extra[r] = row
	}
	gridWith := func(on map[model.Role][]bool) map[model.Role][]float64 {
		g := make(map[model.Role][]float64, len(baseGrid))
		for r, row := range baseGrid {
			nr := append([]float64(nil), row...)
			if flags, ok := on[r]; ok {
				for d := range nr {
					if flags[d] {
						nr[d] += extra[r][d]
					}
				}
			}
			g[r] = nr
		}
		return g
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

	optimize := func(g map[model.Role][]float64, samples int) (level.Optimized, error) {
		q := o
		q.Options = opts
		q.CapGrid = g
		q.Samples = samples
		return level.Optimize(acts, q)
	}
	none, err := optimize(baseGrid, o.Samples)
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
	full, err := optimize(gridWith(on), o.Samples)
	if err != nil {
		return OvertimeCurve{}, err
	}
	out.MinDuration, out.Bound, out.MinProven = full.Best.Duration, full.Bound, full.Proven
	if out.MinDuration >= out.Levelled {
		return out, nil
	}

	order := full.Best.Order
	run := func(g map[model.Role][]float64) (level.Result, error) {
		q := opts
		q.Lite, q.CapGrid, q.Order, q.Rule = true, g, order, ""
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
			g := gridWith(on)
			res, err := run(g)
			if err != nil {
				return OvertimeCurve{}, err
			}
			if res.Duration <= T {
				continue
			}
			alt, err := optimize(g, 60)
			if err != nil {
				return OvertimeCurve{}, err
			}
			if alt.Best.Duration <= T {
				order = alt.Best.Order
				continue
			}
			on[r] = saved
		}
		// Tahap 2: lepas lembur hari demi hari dari belakang dengan urutan tetap.
		for _, r := range out.Eligible {
			for d := T - 1; d >= 0; d-- {
				if !on[r][d] {
					continue
				}
				on[r][d] = false
				res, err := run(gridWith(on))
				if err != nil {
					return OvertimeCurve{}, err
				}
				if res.Duration > T {
					on[r][d] = true
				}
			}
		}
		g := gridWith(on)
		sched, err := run(g)
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
// jadwal di atas kapasitas normal. Lembur dibagi rata ke orang dalam peran
// itu - pembagian termurah, karena jam pertama paling murah.
func overtimePoint(sched level.Result, baseGrid map[model.Role][]float64, c OvertimeCurve, opts level.Options) OvertimePoint {
	pt := OvertimePoint{Duration: sched.Duration, Hours: map[model.Role]float64{}, Days: map[model.Role]int{}, Schedule: sched}
	for _, r := range c.Eligible {
		n := c.Persons[r]
		week := map[string]float64{}
		for d := 0; d < sched.Duration && d < len(sched.Usage[r]); d++ {
			over := sched.Usage[r][d] - baseGrid[r][d]
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
