// Package resource menghitung pembebanan sumber daya manusia terhadap jadwal.
//
// CPM menganggap sumber daya tak terbatas: kalau dua aktivitas boleh paralel,
// CPM menjadwalkannya paralel - tak peduli keduanya butuh orang yang sama.
// Paket ini memeriksa asumsi itu terhadap kapasitas nyata dan menunjukkan
// kapan seseorang dijadwalkan mengerjakan lebih dari yang mungkin dikerjakan.
package resource

import (
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// DayLoad adalah beban satu peran pada satu hari kerja.
type DayLoad struct {
	Day      int
	Load     float64
	Capacity float64
	Over     bool
	// Activities adalah aktivitas yang membebani peran itu pada hari tersebut.
	Activities []string
}

// RoleLoad adalah profil pembebanan satu peran sepanjang proyek.
type RoleLoad struct {
	Role        model.Role
	Days        []DayLoad
	PeakLoad    float64
	PeakDay     int
	TotalDays   float64 // total hari-orang yang dibebankan
	OverDays    int     // jumlah hari dengan beban melebihi kapasitas
	Utilisation float64 // total beban dibagi total kapasitas sepanjang proyek
	FirstDay    int
	LastDay     int
	Capacity    float64
}

// Profile adalah hasil analisis pembebanan seluruh tim.
type Profile struct {
	Roles           []RoleLoad
	Horizon         int
	TotalPersonDays float64
	// Conflicts adalah hari-hari dengan over-alokasi, diurutkan dari yang
	// kelebihannya paling besar - inilah daftar yang harus ditangani lebih dulu.
	Conflicts []Conflict
	// Headcount adalah jumlah peran yang aktif per hari kerja, dipakai untuk
	// kurva kebutuhan tim.
	Headcount []float64
}

// Conflict adalah satu kejadian over-alokasi.
type Conflict struct {
	Role       model.Role
	Day        int
	Load       float64
	Capacity   float64
	Excess     float64
	Activities []string
}

// Analyse menghitung profil pembebanan berdasarkan jadwal awal (early start).
func Analyse(activities []model.Activity, plan schedule.Result, capacity map[model.Role]float64) Profile {
	horizon := plan.Duration
	acc := map[model.Role][]DayLoad{}
	ensure := func(r model.Role) []DayLoad {
		if _, ok := acc[r]; !ok {
			days := make([]DayLoad, horizon)
			for d := range days {
				days[d] = DayLoad{Day: d, Capacity: capacity[r]}
			}
			acc[r] = days
		}
		return acc[r]
	}

	for _, a := range activities {
		task, ok := plan.Tasks[a.ID]
		if !ok || task.Duration == 0 {
			continue
		}
		for _, slot := range a.Team {
			days := ensure(slot.Role)
			for d := task.StartX; d < task.FinishX && d < horizon; d++ {
				days[d].Load += slot.Alloc
				days[d].Activities = append(days[d].Activities, a.ID)
			}
		}
	}

	p := Profile{Horizon: horizon, Headcount: make([]float64, horizon)}
	var roles []model.Role
	for r := range acc {
		roles = append(roles, r)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })

	for _, r := range roles {
		days := acc[r]
		rl := RoleLoad{Role: r, Days: days, Capacity: capacity[r], FirstDay: -1, LastDay: -1}
		var totalCapacity float64
		for i := range days {
			d := &days[i]
			d.Over = d.Load > d.Capacity+1e-9
			rl.TotalDays += d.Load
			totalCapacity += d.Capacity
			if d.Load > 0 {
				p.Headcount[i]++
				if rl.FirstDay < 0 {
					rl.FirstDay = i
				}
				rl.LastDay = i
			}
			if d.Load > rl.PeakLoad {
				rl.PeakLoad = d.Load
				rl.PeakDay = i
			}
			if d.Over {
				rl.OverDays++
				p.Conflicts = append(p.Conflicts, Conflict{
					Role: r, Day: i, Load: d.Load, Capacity: d.Capacity,
					Excess: d.Load - d.Capacity, Activities: append([]string(nil), d.Activities...),
				})
			}
		}
		if totalCapacity > 0 {
			rl.Utilisation = rl.TotalDays / totalCapacity
		}
		p.TotalPersonDays += rl.TotalDays
		p.Roles = append(p.Roles, rl)
	}

	sort.Slice(p.Conflicts, func(i, j int) bool {
		if p.Conflicts[i].Excess != p.Conflicts[j].Excess {
			return p.Conflicts[i].Excess > p.Conflicts[j].Excess
		}
		return p.Conflicts[i].Day < p.Conflicts[j].Day
	})
	return p
}

// PeakHeadcount mengembalikan jumlah peran aktif terbanyak dalam satu hari,
// beserta hari kerjanya.
func (p Profile) PeakHeadcount() (peak float64, day int) {
	for i, h := range p.Headcount {
		if h > peak {
			peak = h
			day = i
		}
	}
	return peak, day
}

// Smoothness mengukur seberapa rata kurva kebutuhan tim, memakai rata-rata
// selisih absolut antar hari dibagi puncaknya. Nilai mendekati nol berarti tim
// bekerja stabil; nilai besar berarti jadwalnya bergerigi - tanda bahwa
// resource levelling belum dilakukan dan orang menganggur lalu kewalahan
// bergantian.
func (p Profile) Smoothness() float64 {
	if len(p.Headcount) < 2 {
		return 0
	}
	peak, _ := p.PeakHeadcount()
	if peak == 0 {
		return 0
	}
	var sum float64
	for i := 1; i < len(p.Headcount); i++ {
		d := p.Headcount[i] - p.Headcount[i-1]
		if d < 0 {
			d = -d
		}
		sum += d
	}
	return sum / float64(len(p.Headcount)-1) / peak
}
