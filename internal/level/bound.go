package level

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// Bound adalah batas bawah durasi proyek: tidak ada jadwal yang menghormati
// kapasitas - dibuat dengan metode apa pun - yang bisa selesai lebih cepat.
//
// Tiga lapis argumen, masing-masing sah sendiri:
//
//	CPM        jaringan dengan durasi nominal, sumber daya tak terbatas
//	Solo       setiap aktivitas dikerjakan sendirian, tanpa berbagi orang,
//	           tetapi pada kapasitas nyata perannya (paruh waktu, ujian)
//	Energetik  sekelompok aktivitas yang memakai peran yang sama harus
//	           mendapat hari-orang yang cukup dari peran itu di dalam jendela
//	           waktu yang mungkin baginya
//
// Argumen energetik (Baptiste, Le Pape & Nuijten, 2001) memakai fakta yang
// tidak bisa ditawar model isi pekerjaan: hari-orang yang dibutuhkan setiap
// aktivitas tetap alokasi x durasi, berapa pun lajunya.
type Bound struct {
	CPM       int
	Solo      int
	Energetic int
	// Constructive adalah batas terbaik dari tiga lapis di atas; Value bisa
	// lebih tinggi bila pembuktian destruktif menyangkal tenggat-tenggat
	// berikutnya. Refuted mencatat tenggat yang terbukti mustahil.
	Constructive int
	Refuted      []int
	Value        int
	// Role adalah peran yang argumen energetiknya menentukan Value; kosong
	// bila Value berasal dari lapis Solo atau CPM.
	Role model.Role
}

const inf = math.MaxInt32 / 4

// LowerBound menghitung batas bawah durasi levelling untuk sebuah jaringan.
func LowerBound(acts []model.Activity, opts Options) (Bound, error) {
	if opts.Calendar == nil || opts.Capacity == nil {
		return Bound{}, fmt.Errorf("level: kalender dan kapasitas wajib diisi")
	}
	durationOf := opts.DurationOf
	if durationOf == nil {
		durationOf = func(a model.Activity) int { return a.Duration }
	}
	H := opts.Horizon
	if H <= 0 {
		H = 300
	}
	if H > opts.Calendar.Len() {
		H = opts.Calendar.Len()
	}
	grid := opts.CapGrid
	if grid == nil {
		grid = CapacityGridWith(opts.Calendar, opts.Capacity, opts.UseWindows, H, opts.ExamFactor)
	}

	plan, err := schedule.Compute(acts, schedule.Options{DurationOf: durationOf, ReleaseOf: opts.ReleaseOf})
	if err != nil {
		return Bound{}, err
	}
	order := plan.Order
	n := len(order)
	idx := make(map[string]int, n)
	for i, id := range order {
		idx[id] = i
	}
	byID := make(map[string]model.Activity, n)
	for _, a := range acts {
		byID[a.ID] = a
	}
	act := make([]model.Activity, n)
	dur := make([]int, n)
	for i, id := range order {
		act[i] = byID[id]
		dur[i] = durationOf(act[i])
	}

	// Prefix kapasitas per peran untuk menjawab "kapan W hari-orang
	// terkumpul sejak hari h" dengan pencarian biner.
	prefix := make(map[model.Role][]float64, len(grid))
	for role, row := range grid {
		p := make([]float64, H+1)
		for k := 0; k < H && k < len(row); k++ {
			p[k+1] = p[k] + row[k]
		}
		prefix[role] = p
	}
	supply := func(role model.Role, h int, w float64) int {
		p := prefix[role]
		if p == nil || h >= H {
			return inf
		}
		if h < 0 {
			h = 0
		}
		target := p[h] + w - 1e-9
		if p[H] < target {
			return inf
		}
		return sort.SearchFloat64s(p[h:], target) + h
	}

	// peak adalah kapasitas harian tertinggi setiap peran di horizon. Penalaran
	// yang tidak bertanggal wajib memakainya, bukan kapasitas dasar: jendela
	// ujian hanya menurunkan kapasitas, tetapi lembur dan orang baru
	// menaikkannya, dan batas yang memakai kapasitas dasar pada grid seperti
	// itu melampaui jadwal yang benar-benar bisa dibuat.
	peak := make(map[model.Role]float64, len(grid))
	for role, row := range grid {
		for k := 0; k < H && k < len(row); k++ {
			if row[k] > peak[role] {
				peak[role] = row[k]
			}
		}
	}
	// peakRate adalah RateCap tertinggi setiap peran, untuk alasan yang sama.
	peakRate := make(map[model.Role]float64, len(opts.RateCap))
	for role, row := range opts.RateCap {
		for k := 0; k < H && k < len(row); k++ {
			if row[k] > peakRate[role] {
				peakRate[role] = row[k]
			}
		}
	}

	// Durasi solo tanpa jendela: batas bawah waktu yang tidak bergantung
	// tanggal, dipakai untuk ekor (tail) dan jarak antar-aktivitas.
	soloDur := make([]int, n)
	for i, a := range act {
		// Laju tertinggi yang mungkin: RateCap tertinggi peran timnya yang
		// paling rendah, lalu dibatasi kapasitas harian tertinggi.
		rate, first := 1.0, true
		for _, s := range a.Team {
			if s.Alloc <= 0 {
				continue
			}
			if v := math.Max(1, peakRate[s.Role]); first || v < rate {
				rate, first = v, false
			}
		}
		for _, s := range a.Team {
			if s.Alloc <= 0 {
				continue
			}
			if r := peak[s.Role] / s.Alloc; r < rate {
				rate = r
			}
		}
		if dur[i] == 0 || len(a.Team) == 0 || rate <= 0 {
			soloDur[i] = dur[i]
			continue
		}
		soloDur[i] = int(math.Ceil(float64(dur[i])/rate - 1e-9))
	}

	// gap[i][j] = panjang jalur terpanjang dari selesainya i ke mulainya j
	// (durasi solo simpul di antaranya + lag), -1 bila j bukan keturunan i.
	gap := make([][]int, n)
	for i := range gap {
		gap[i] = make([]int, n)
		for j := range gap[i] {
			gap[i][j] = -1
		}
	}
	fsOnly := true
	for j := 0; j < n; j++ {
		for _, p := range act[j].Pred {
			if p.Type != "" && p.Type != "FS" {
				fsOnly = false
				continue
			}
			i := idx[p.ID]
			lag := p.Lag
			if lag < 0 {
				lag = 0
			}
			if lag > gap[i][j] {
				gap[i][j] = lag
			}
			for a := 0; a < n; a++ {
				if gap[a][i] >= 0 {
					if v := gap[a][i] + soloDur[i] + lag; v > gap[a][j] {
						gap[a][j] = v
					}
				}
			}
		}
	}

	users := map[model.Role][]use{}
	for i, a := range act {
		for _, s := range a.Team {
			if s.Alloc > 0 && dur[i] > 0 {
				users[s.Role] = append(users[s.Role], use{i, s.Alloc})
			}
		}
	}
	roles := make([]model.Role, 0, len(users))
	for r := range users {
		roles = append(roles, r)
	}
	sort.Slice(roles, func(a, b int) bool { return roles[a] < roles[b] })

	release := make([]int, n)
	if opts.ReleaseOf != nil {
		for i, a := range act {
			release[i] = opts.ReleaseOf(a)
		}
	}
	head := make([]int, n)   // batas bawah hari mulai
	finish := make([]int, n) // batas bawah hari selesai (eksklusif)
	tail := make([]int, n)   // batas bawah hari kerja setelah selesai

	// soloRate adalah laju tercepat aktivitas i pada hari k bila ia sendirian.
	soloRate := func(i, k int) float64 {
		rate := RateLimit(act[i], k, opts.RateCap)
		for _, s := range act[i].Team {
			if s.Alloc <= 0 {
				continue
			}
			c := 0.0
			if row := grid[s.Role]; k >= 0 && k < len(row) {
				c = row[k]
			}
			if r := c / s.Alloc; r < rate {
				rate = r
			}
		}
		return rate
	}
	minRate := opts.MinStartRate
	if minRate <= 0 {
		minRate = DefaultMinStartRate
	}
	// startable memajukan h ke hari pertama yang laju solonya memenuhi aturan
	// laju mulai minimum - aturan yang sama dengan Run. Laju saat berbagi
	// orang tidak mungkin melebihi laju solo, jadi batas ini tetap sah.
	startable := func(i, h int) int {
		if dur[i] == 0 || len(act[i].Team) == 0 {
			return h
		}
		for h < H && soloRate(i, h) < minRate-1e-9 {
			h++
		}
		return h
	}
	soloFinish := func(i, start int) int {
		if dur[i] == 0 || len(act[i].Team) == 0 {
			return start + dur[i]
		}
		need := float64(dur[i])
		for k := start; k < H; k++ {
			need -= soloRate(i, k)
			if need <= 1e-9 {
				return k + 1
			}
		}
		return inf
	}

	// propagate menjalankan penalaran maju-mundur sampai titik tetap dan
	// mengembalikan batas bawah max(selesai + ekor).
	propagate := func(energetic bool) int {
		for i := range head {
			head[i], finish[i], tail[i] = 0, 0, 0
		}
		for iter := 0; iter < 60; iter++ {
			changed := false
			for j := 0; j < n; j++ {
				h := release[j]
				for _, p := range act[j].Pred {
					// Relasi selain FS tidak dipakai menaikkan batas; nol selalu sah.
					if p.Type == "" || p.Type == "FS" {
						if c := finish[idx[p.ID]] + p.Lag; c > h {
							h = c
						}
					}
				}
				if energetic && fsOnly {
					for _, role := range roles {
						if v := ancestorEnergy(users[role], j, head, gap, dur, role, supply); v > h {
							h = v
						}
					}
				}
				h = startable(j, h)
				if h > head[j] {
					head[j] = h
					changed = true
				}
				if f := soloFinish(j, head[j]); f > finish[j] {
					finish[j] = f
					changed = true
				}
			}
			for i := n - 1; i >= 0; i-- {
				t := 0
				for j := i + 1; j < n; j++ {
					for _, p := range act[j].Pred {
						if p.ID != order[i] || (p.Type != "" && p.Type != "FS") {
							continue
						}
						lag := p.Lag
						if lag < 0 {
							lag = 0
						}
						if v := lag + soloDur[j] + tail[j]; v > t {
							t = v
						}
					}
				}
				if energetic && fsOnly {
					for _, role := range roles {
						base := peak[role]
						if base <= 0 {
							continue
						}
						var cand []use
						for _, u := range users[role] {
							if gap[i][u.i] >= 0 {
								cand = append(cand, u)
							}
						}
						if v := descendantEnergy(cand, i, gap, tail, dur, base); v > t {
							t = v
						}
					}
				}
				if t > tail[i] {
					tail[i] = t
					changed = true
				}
			}
			if !changed {
				break
			}
		}
		v := 0
		for i := 0; i < n; i++ {
			if x := finish[i] + tail[i]; x > v {
				v = x
			}
		}
		return v
	}

	b := Bound{CPM: plan.Duration, Solo: propagate(false)}
	b.Energetic = propagate(true)

	// Energetik global: sekelompok aktivitas peran r yang semuanya mulai
	// paling awal h dan menyisakan ekor paling sedikit q.
	for _, role := range roles {
		us := users[role]
		for _, pivot := range us {
			h := head[pivot.i]
			var sub []use
			for _, u := range us {
				if head[u.i] >= h {
					sub = append(sub, u)
				}
			}
			sort.Slice(sub, func(a, c int) bool { return tail[sub[a].i] > tail[sub[c].i] })
			var w float64
			for k, u := range sub {
				w += u.alloc * float64(dur[u.i])
				if k+1 < len(sub) && tail[sub[k+1].i] == tail[u.i] {
					continue
				}
				if v := supply(role, h, w); v < inf && v+tail[u.i] > b.Energetic {
					b.Energetic = v + tail[u.i]
					b.Role = role
				}
			}
		}
	}
	// CPM memakai durasi pada laju 1. Bila lembur membuat laju bisa melebihi 1,
	// durasi CPM bukan lagi batas bawah; batas solo menggantikannya.
	b.Value = b.CPM
	for _, r := range peakRate {
		if r > 1+1e-9 {
			b.Value = 0
			break
		}
	}
	if b.Solo > b.Value {
		b.Value = b.Solo
	}
	if b.Energetic > b.Value {
		b.Value = b.Energetic
	} else {
		b.Role = ""
	}

	// Pembuktian destruktif (Klein & Scholl, 1999): anggap proyek harus
	// selesai pada E. Bila tenggat itu memaksa suatu aktivitas mulai sebelum
	// batas mulainya, atau memaksa suatu peran menyediakan hari-orang lebih
	// banyak daripada kapasitasnya di sebuah jendela waktu, maka E mustahil
	// dan batas bawah naik menjadi E + 1. Berbeda dari ekor, tenggat bertanggal
	// sehingga jendela ujian ikut dihitung di kedua arah.
	b.Constructive = b.Value
	if fsOnly {
		deadline := make([]int, n)
		latest := make([]int, n)
		for E := b.Value; E < H; E++ {
			if !refute(E, n, order, act, dur, idx, head, deadline, latest, gap, users, roles, prefix, soloRate, minRate) {
				break
			}
			b.Value = E + 1
			b.Refuted = append(b.Refuted, E)
		}
	}
	return b, nil
}

// refute melaporkan apakah tenggat E terbukti mustahil.
func refute(E, n int, order []string, act []model.Activity, dur []int, idx map[string]int,
	head, deadline, latest []int, gap [][]int, users map[model.Role][]use, roles []model.Role,
	prefix map[model.Role][]float64, soloRate func(i, k int) float64, minRate float64) bool {

	capBetween := func(role model.Role, from, to int) float64 {
		p := prefix[role]
		if p == nil || to <= from {
			return 0
		}
		if from < 0 {
			from = 0
		}
		if to > len(p)-1 {
			to = len(p) - 1
		}
		return p[to] - p[from]
	}
	// latestStartBefore mengembalikan hari t terbesar sehingga kapasitas
	// peran di [t, to) masih memuat w hari-orang.
	latestStartBefore := func(role model.Role, to int, w float64) int {
		p := prefix[role]
		if p == nil {
			return -inf
		}
		if to > len(p)-1 {
			to = len(p) - 1
		}
		lo, hi := 0, to
		if p[to]-p[0] < w-1e-9 {
			return -inf
		}
		for lo < hi {
			mid := (lo + hi + 1) / 2
			if p[to]-p[mid] >= w-1e-9 {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		return lo
	}

	succOf := make([][]model.Predecessor, n)
	for j := 0; j < n; j++ {
		for _, p := range act[j].Pred {
			i := idx[p.ID]
			succOf[i] = append(succOf[i], model.Predecessor{ID: order[j], Type: p.Type, Lag: p.Lag})
		}
	}

	for i := n - 1; i >= 0; i-- {
		d := E
		for _, s := range succOf[i] {
			if c := latest[idx[s.ID]] - s.Lag; c < d {
				d = c
			}
		}
		// Keturunan yang memakai peran sama harus muat di antara selesainya i
		// (plus jarak g) dan tenggat mereka.
		for _, role := range roles {
			type item struct {
				g, dl int
				w     float64
			}
			var items []item
			for _, u := range users[role] {
				if g := gap[i][u.i]; g >= 0 {
					items = append(items, item{g, deadline[u.i], u.alloc * float64(dur[u.i])})
				}
			}
			for _, pv := range items {
				var w float64
				minG := inf
				for _, it := range items {
					if it.dl <= pv.dl && it.g >= pv.g {
						w += it.w
						if it.g < minG {
							minG = it.g
						}
					}
				}
				if t := latestStartBefore(role, pv.dl, w); t-pv.g < d {
					d = t - pv.g
				}
			}
		}
		deadline[i] = d
		if dur[i] == 0 || len(act[i].Team) == 0 {
			latest[i] = d - dur[i]
		} else {
			need := float64(dur[i])
			k := d - 1
			for ; k >= 0; k-- {
				need -= soloRate(i, k)
				if need <= 1e-9 {
					break
				}
			}
			for k >= 0 && soloRate(i, k) < minRate-1e-9 {
				k--
			}
			latest[i] = k
			if k < 0 {
				return true
			}
		}
		if latest[i] < head[i] {
			return true
		}
	}

	// Energetik interval: aktivitas peran r dengan mulai >= h dan tenggat <= dl
	// harus muat di dalam kapasitas [h, dl).
	for _, role := range roles {
		us := users[role]
		for _, a := range us {
			h := head[a.i]
			for _, b := range us {
				dl := deadline[b.i]
				if dl <= h {
					continue
				}
				var w float64
				for _, u := range us {
					if head[u.i] >= h && deadline[u.i] <= dl {
						w += u.alloc * float64(dur[u.i])
					}
				}
				if capBetween(role, h, dl) < w-1e-9 {
					return true
				}
			}
		}
	}
	return false
}

// ancestorEnergy: seluruh leluhur i dari j yang memakai peran itu, mulai
// paling awal h dan berjarak paling sedikit g ke j, harus mendapat
// hari-orangnya di [h, mulai_j - g). Maka mulai_j >= supply(h, W) + g.
func ancestorEnergy(us []use, j int, head []int, gap [][]int, dur []int, role model.Role, supply func(model.Role, int, float64) int) int {
	type item struct {
		h, g int
		w    float64
	}
	var items []item
	for _, u := range us {
		if gap[u.i][j] < 0 {
			continue
		}
		items = append(items, item{head[u.i], gap[u.i][j], u.alloc * float64(dur[u.i])})
	}
	best := 0
	for _, pivot := range items {
		h := pivot.h
		var sub []item
		for _, it := range items {
			if it.h >= h {
				sub = append(sub, it)
			}
		}
		sort.Slice(sub, func(a, b int) bool { return sub[a].g > sub[b].g })
		var w float64
		for k, it := range sub {
			w += it.w
			if k+1 < len(sub) && sub[k+1].g == it.g {
				continue
			}
			if v := supply(role, h, w); v < inf && v+it.g > best {
				best = v + it.g
			}
		}
	}
	return best
}

// descendantEnergy: seluruh keturunan dari i yang memakai peran itu, berjarak
// paling sedikit g dari selesainya i dan menyisakan ekor paling sedikit q,
// butuh ceil(W / kapasitas dasar) hari. Maka ekor_i >= g + ceil(W/c) + q.
// Kapasitas dasar dipakai (bukan jendela ujian) karena ekor tidak bertanggal;
// jendela hanya bisa memperlambat, jadi batasnya tetap sah.
func descendantEnergy(us []use, i int, gap [][]int, tail []int, dur []int, base float64) int {
	best := 0
	for _, pivot := range us {
		g := gap[i][pivot.i]
		var sub []use
		for _, u := range us {
			if gap[i][u.i] >= g {
				sub = append(sub, u)
			}
		}
		sort.Slice(sub, func(a, b int) bool { return tail[sub[a].i] > tail[sub[b].i] })
		var w float64
		for k, u := range sub {
			w += u.alloc * float64(dur[u.i])
			if k+1 < len(sub) && tail[sub[k+1].i] == tail[u.i] {
				continue
			}
			days := int(math.Ceil(w/base - 1e-9))
			if v := g + days + tail[u.i]; v > best {
				best = v
			}
		}
	}
	return best
}

// use adalah pemakaian satu peran oleh satu aktivitas (indeks topologis).
type use struct {
	i     int
	alloc float64
}

// scheduleFor menghitung CPM dengan durasi dan tanggal rilis dari opsi.
func scheduleFor(acts []model.Activity, opts Options) (schedule.Result, error) {
	return schedule.Compute(acts, schedule.Options{DurationOf: opts.DurationOf, ReleaseOf: opts.ReleaseOf})
}
