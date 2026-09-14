package simulate

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Acceleration adalah opsi percepatan yang diputuskan pada tanggal data.
//
// Pada model isi pekerjaan, mempercepat berarti menambah hari-orang yang
// tersedia, bukan memotong pekerjaan. Dua caranya:
//
//   - Lembur: orang penuh waktu pada OvertimeRoles boleh bekerja
//     OvertimeHours jam tambahan per hari kerja, mulai From, kecuali pada hari
//     jendela ketersediaan (ujian). Lembur menambah kapasitas peran DAN
//     mempercepat satu pekerjaan: orang yang sama bekerja 1 + h/8 hari per
//     hari. Upahnya menurut PP 35/2021 dan hanya dibayar untuk jam yang
//     benar-benar terpakai.
//   - Menambah orang: Hire orang penuh waktu per peran mulai From. Orang baru
//     tidak ikut ujian kampus, dan selama RampDays hari kerja pertama bekerja
//     dengan laju RampFactor. Orang baru menambah kapasitas, tetapi tidak
//     mempercepat satu pekerjaan yang tak terbagi. Mereka dibayar tarif harian penuh sejak From
//     sampai aktivitas terakhir yang memakai perannya selesai.
type Acceleration struct {
	From          int
	OvertimeRoles []model.Role
	OvertimeHours float64
	Hire          map[model.Role]float64
	RampDays      int
	RampFactor    float64
}

// AccelGrids adalah kapasitas harian sebuah opsi percepatan.
type AccelGrids struct {
	// Normal adalah kapasitas tanpa lembur (termasuk orang baru).
	Normal map[model.Role][]float64
	// Max adalah Normal ditambah lembur yang diizinkan.
	Max map[model.Role][]float64
	// Persons adalah jumlah orang per peran untuk membagi jam lembur.
	Persons map[model.Role]float64
	// Rate adalah laju maksimum satu aktivitas per peran per hari (level.Options.RateCap).
	Rate map[model.Role][]float64
}

// Grids menghitung kapasitas harian opsi ini di atas kapasitas tim yang ada.
func (ac *Acceleration) Grids(cal *workcal.Calendar, capacity map[model.Role]float64, horizon int, examFactor float64) (AccelGrids, error) {
	if cal == nil || capacity == nil {
		return AccelGrids{}, fmt.Errorf("simulate: opsi percepatan butuh kalender dan kapasitas")
	}
	if horizon > cal.Len() {
		horizon = cal.Len()
	}
	if ac.OvertimeHours < 0 || ac.OvertimeHours > model.OvertimeMaxDaily || ac.OvertimeHours*5 > model.OvertimeMaxWeekly+1e-9 {
		return AccelGrids{}, fmt.Errorf("simulate: %v jam lembur per hari melanggar PP 35/2021", ac.OvertimeHours)
	}
	win := level.CapacityGridWith(cal, capacity, true, horizon, examFactor)
	plain := level.CapacityGridWith(cal, capacity, false, horizon, examFactor)
	g := AccelGrids{Normal: map[model.Role][]float64{}, Max: map[model.Role][]float64{}, Persons: map[model.Role]float64{}, Rate: map[model.Role][]float64{}}
	for r, row := range win {
		g.Normal[r] = append([]float64(nil), row...)
		if capacity[r] >= 1 {
			g.Persons[r] = math.Ceil(capacity[r])
		}
	}
	for r, n := range ac.Hire {
		if n <= 0 {
			continue
		}
		if _, ok := g.Normal[r]; !ok {
			return AccelGrids{}, fmt.Errorf("simulate: peran %s tidak ada di kapasitas tim", r)
		}
		g.Persons[r] += n
		for d := ac.From; d < horizon; d++ {
			f := 1.0
			if d < ac.From+ac.RampDays {
				f = ac.RampFactor
			}
			g.Normal[r][d] += n * f
		}
	}
	for r, row := range g.Normal {
		g.Max[r] = append([]float64(nil), row...)
	}
	for _, r := range ac.OvertimeRoles {
		n := g.Persons[r]
		if n <= 0 {
			continue // peran paruh waktu tanpa orang baru tidak diberi lembur
		}
		rate := make([]float64, horizon)
		for d := range rate {
			rate[d] = 1
		}
		for d := ac.From; d < horizon; d++ {
			if math.Abs(win[r][d]-plain[r][d]) < 1e-9 {
				g.Max[r][d] += n * ac.OvertimeHours / model.RegularHoursPerDay
				rate[d] = 1 + ac.OvertimeHours/model.RegularHoursPerDay
			}
		}
		g.Rate[r] = rate
	}
	return g, nil
}

// AccelPay adalah biaya tambahan opsi percepatan pada satu jadwal.
type AccelPay struct {
	Overtime      float64
	OvertimeHours float64
	Hire          float64
	HireDays      float64 // hari-orang orang baru yang dibayar
}

// Total mengembalikan seluruh biaya tambahan.
func (p AccelPay) Total() float64 { return p.Overtime + p.Hire }

// Pay menghitung upah lembur dari jam di atas kapasitas Normal atau di atas
// alokasi rencana (level.Result.Excess) - yang lebih besar - dan upah orang
// baru dari From sampai aktivitas terakhir perannya selesai.
func (ac *Acceleration) Pay(sched level.Result, acts []model.Activity, g AccelGrids, rates map[model.Role]float64) AccelPay {
	var p AccelPay
	for _, r := range ac.OvertimeRoles {
		n := g.Persons[r]
		if n <= 0 {
			continue
		}
		u := sched.Usage[r]
		for d := ac.From; d < sched.Duration && d < len(u); d++ {
			over := u[d] - g.Normal[r][d]
			if ex := sched.Excess[r]; d < len(ex) && ex[d] > over {
				over = ex[d]
			}
			if over <= 1e-9 {
				continue
			}
			h := over * model.RegularHoursPerDay
			p.OvertimeHours += h
			p.Overtime += n * model.OvertimeUnits(h/n) * rates[r] / model.RegularHoursPerDay
		}
	}
	roles := make([]model.Role, 0, len(ac.Hire))
	for r := range ac.Hire {
		roles = append(roles, r)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })
	for _, r := range roles {
		n := ac.Hire[r]
		if n <= 0 {
			continue
		}
		last := ac.From
		for _, a := range acts {
			t, ok := sched.Tasks[a.ID]
			if !ok || t.Finish <= ac.From {
				continue
			}
			for _, s := range a.Team {
				if s.Role == r && t.Finish > last {
					last = t.Finish
				}
			}
		}
		days := float64(last-ac.From) * n
		p.HireDays += days
		p.Hire += days * rates[r]
	}
	return p
}
