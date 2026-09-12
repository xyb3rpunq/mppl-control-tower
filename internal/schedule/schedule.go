// Package schedule mengimplementasikan Critical Path Method (CPM) di atas
// jaringan Precedence Diagramming Method (PDM / Activity-on-Node).
//
// Mendukung empat relasi PMBOK - FS, SS, FF, SF - beserta lag (positif) dan
// lead (lag negatif). Satuan waktu: hari kerja, bilangan bulat, indeks 0-based
// dari hari kerja pertama proyek.
//
// Konvensi tanggal yang dipakai di seluruh aplikasi:
//
//	ES = hari kerja pertama aktivitas dikerjakan (inklusif)
//	EF = hari kerja terakhir aktivitas dikerjakan (inklusif)
//
// sehingga EF = ES + durasi - 1, dan milestone (durasi 0) punya EF = ES - 1.
// Untuk aritmetika relasi dipakai batas eksklusif (finish = EF + 1) supaya
// milestone berdurasi nol tidak perlu diperlakukan khusus.
package schedule

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Task adalah hasil perhitungan CPM untuk satu aktivitas.
type Task struct {
	ID       string
	Duration int

	// Batas inklusif, untuk ditampilkan ke pengguna.
	ES, EF, LS, LF int

	// Batas eksklusif, untuk aritmetika lanjutan (Gantt, EVM).
	StartX, FinishX         int
	LateStartX, LateFinishX int

	TotalFloat int
	FreeFloat  int
	Critical   bool

	Successors   []string
	Predecessors []model.Predecessor
}

// Result adalah keluaran satu kali perhitungan CPM.
type Result struct {
	Tasks         map[string]Task
	Order         []string // urutan topologis
	Duration      int      // durasi proyek dalam hari kerja
	ProjectFinish int      // indeks hari kerja setelah aktivitas terakhir
	CriticalPath  []string
}

// Task mengambil hasil untuk satu aktivitas.
func (r Result) Task(id string) Task { return r.Tasks[id] }

// DurationFunc memungkinkan penyuntikan durasi lain tanpa menyalin data -
// dipakai Monte Carlo untuk menjalankan ribuan iterasi atas jaringan yang sama.
type DurationFunc func(model.Activity) int

// Options mengatur satu kali perhitungan CPM.
type Options struct {
	DurationOf   DurationFunc
	ProjectStart int
}

func normalise(p model.Predecessor) model.Predecessor {
	if p.Type == "" {
		p.Type = "FS"
	}
	return p
}

// TopoSort mengurutkan aktivitas secara topologis. Mengembalikan galat bila
// ada siklus - jaringan proyek yang sah tidak boleh punya loop (lihat Modul 4:
// GERT dipakai untuk kasus berulang, CPM tidak).
func TopoSort(activities []model.Activity) (order []string, succ map[string][]string, byID map[string]model.Activity, err error) {
	byID = make(map[string]model.Activity, len(activities))
	indeg := make(map[string]int, len(activities))
	succ = make(map[string][]string, len(activities))
	for _, a := range activities {
		if _, dup := byID[a.ID]; dup {
			return nil, nil, nil, fmt.Errorf("schedule: kode aktivitas ganda %q", a.ID)
		}
		byID[a.ID] = a
		indeg[a.ID] = 0
	}
	for _, a := range activities {
		for _, p := range a.Pred {
			if _, ok := byID[p.ID]; !ok {
				return nil, nil, nil, fmt.Errorf("schedule: aktivitas %q merujuk predecessor tak dikenal %q", a.ID, p.ID)
			}
			succ[p.ID] = append(succ[p.ID], a.ID)
			indeg[a.ID]++
		}
	}

	var queue []string
	for _, a := range activities {
		if indeg[a.ID] == 0 {
			queue = append(queue, a.ID)
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, s := range succ[id] {
			indeg[s]--
			if indeg[s] == 0 {
				queue = append(queue, s)
			}
		}
	}
	if len(order) != len(activities) {
		seen := make(map[string]bool, len(order))
		for _, id := range order {
			seen[id] = true
		}
		var stuck []string
		for _, a := range activities {
			if !seen[a.ID] {
				stuck = append(stuck, a.ID)
			}
		}
		sort.Strings(stuck)
		return nil, nil, nil, fmt.Errorf("schedule: siklus terdeteksi pada jaringan: %v", stuck)
	}
	return order, succ, byID, nil
}

// Compute menjalankan forward pass, backward pass, dan perhitungan float.
func Compute(activities []model.Activity, opts Options) (Result, error) {
	durationOf := opts.DurationOf
	if durationOf == nil {
		durationOf = func(a model.Activity) int { return a.Duration }
	}
	order, succ, byID, err := TopoSort(activities)
	if err != nil {
		return Result{}, err
	}

	es := make(map[string]int, len(order))
	ef := make(map[string]int, len(order))

	// --- Forward pass -----------------------------------------------------
	for _, id := range order {
		act := byID[id]
		dur := durationOf(act)
		earliest := opts.ProjectStart
		for _, raw := range act.Pred {
			p := normalise(raw)
			pAct := byID[p.ID]
			pStart := es[p.ID]
			pFinish := pStart + durationOf(pAct) // batas eksklusif
			var candidate int
			switch p.Type {
			case "SS":
				candidate = pStart + p.Lag
			case "FF":
				candidate = pFinish + p.Lag - dur
			case "SF":
				candidate = pStart + p.Lag - dur
			default: // FS
				candidate = pFinish + p.Lag
			}
			if candidate > earliest {
				earliest = candidate
			}
		}
		es[id] = earliest
		ef[id] = earliest + dur
	}

	projectFinish := opts.ProjectStart
	for _, id := range order {
		if ef[id] > projectFinish {
			projectFinish = ef[id]
		}
	}

	// --- Backward pass ----------------------------------------------------
	lf := make(map[string]int, len(order))
	ls := make(map[string]int, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		act := byID[id]
		dur := durationOf(act)
		successors := succ[id]

		latest := projectFinish
		if len(successors) > 0 {
			latest = math.MaxInt32
		}
		for _, sid := range successors {
			sAct := byID[sid]
			sDur := durationOf(sAct)
			rel := findRel(sAct, id)
			sStart := ls[sid]
			sFinish := sStart + sDur
			var candidate int
			switch rel.Type {
			case "SS":
				candidate = sStart - rel.Lag + dur
			case "FF":
				candidate = sFinish - rel.Lag
			case "SF":
				candidate = sFinish - rel.Lag + dur
			default: // FS
				candidate = sStart - rel.Lag
			}
			if candidate < latest {
				latest = candidate
			}
		}
		lf[id] = latest
		ls[id] = latest - dur
	}

	// --- Float & jalur kritis --------------------------------------------
	tasks := make(map[string]Task, len(order))
	for _, id := range order {
		act := byID[id]
		dur := durationOf(act)
		totalFloat := ls[id] - es[id]

		// Free float: seberapa lama aktivitas boleh molor tanpa menggeser ES
		// penerus mana pun. Tanpa penerus, dibatasi akhir proyek.
		successors := succ[id]
		freeFloat := projectFinish - ef[id]
		if len(successors) > 0 {
			freeFloat = math.MaxInt32
			for _, sid := range successors {
				sAct := byID[sid]
				rel := findRel(sAct, id)
				sEs := es[sid]
				var slack int
				switch rel.Type {
				case "SS":
					slack = sEs - rel.Lag - es[id]
				case "FF":
					slack = sEs + durationOf(sAct) - rel.Lag - ef[id]
				case "SF":
					slack = sEs + durationOf(sAct) - rel.Lag - es[id]
				default: // FS
					slack = sEs - rel.Lag - ef[id]
				}
				if slack < freeFloat {
					freeFloat = slack
				}
			}
		}
		if freeFloat < 0 {
			freeFloat = 0
		}

		preds := make([]model.Predecessor, 0, len(act.Pred))
		for _, p := range act.Pred {
			preds = append(preds, normalise(p))
		}
		sortedSucc := append([]string(nil), successors...)
		sort.Strings(sortedSucc)

		tasks[id] = Task{
			ID:           id,
			Duration:     dur,
			ES:           es[id],
			EF:           ef[id] - 1,
			LS:           ls[id],
			LF:           lf[id] - 1,
			StartX:       es[id],
			FinishX:      ef[id],
			LateStartX:   ls[id],
			LateFinishX:  lf[id],
			TotalFloat:   totalFloat,
			FreeFloat:    freeFloat,
			Critical:     totalFloat == 0,
			Successors:   sortedSucc,
			Predecessors: preds,
		}
	}

	return Result{
		Tasks:         tasks,
		Order:         order,
		Duration:      projectFinish - opts.ProjectStart,
		ProjectFinish: projectFinish,
		CriticalPath:  longestCriticalChain(tasks, order),
	}, nil
}

// MustCompute seperti Compute tetapi panik bila jaringan tidak sah. Dipakai
// untuk data statis yang sudah dijaga oleh uji.
func MustCompute(activities []model.Activity, opts Options) Result {
	r, err := Compute(activities, opts)
	if err != nil {
		panic(err)
	}
	return r
}

func findRel(a model.Activity, predID string) model.Predecessor {
	for _, p := range a.Pred {
		if p.ID == predID {
			return normalise(p)
		}
	}
	return model.Predecessor{ID: predID, Type: "FS"}
}

// longestCriticalChain mengembalikan rantai kritis terpanjang yang benar-benar
// tersambung. Sekadar memfilter TotalFloat == 0 bisa memunculkan aktivitas
// kritis yang tidak nyambung ketika ada beberapa jalur kritis paralel; di sini
// ditelusuri satu rantai utuh dari awal sampai akhir proyek.
func longestCriticalChain(tasks map[string]Task, order []string) []string {
	type best struct {
		length int
		chain  []string
	}
	memo := make(map[string]best, len(order))
	var bestID string

	for _, id := range order {
		t := tasks[id]
		if !t.Critical {
			continue
		}
		chain := []string{id}
		longest := 0
		for _, p := range t.Predecessors {
			if prev, ok := memo[p.ID]; ok && prev.length > longest {
				longest = prev.length
				chain = append(append([]string(nil), prev.chain...), id)
			}
		}
		memo[id] = best{length: longest + t.Duration, chain: chain}
		if bestID == "" || memo[id].length > memo[bestID].length {
			bestID = id
		}
	}
	if bestID == "" {
		return nil
	}
	return memo[bestID].chain
}
