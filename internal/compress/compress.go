// Package compress mengimplementasikan dua teknik kompresi jadwal yang
// disebut Modul 4 MPPL: crashing (membeli waktu dengan uang) dan
// fast-tracking (membeli waktu dengan risiko).
//
// Keduanya menjawab pertanyaan yang pasti muncul setelah halaman simulasi
// menunjukkan jadwal 17 minggu mustahil: "kalau tanggalnya tidak boleh
// mundur, apa yang harus dibayar?"
package compress

import (
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// Step adalah satu langkah crashing: satu hari proyek yang dibeli.
type Step struct {
	Duration   int      // durasi proyek SESUDAH langkah ini
	Crashed    []string // aktivitas yang dipotong sehari pada langkah ini
	StepCost   float64  // biaya langkah ini
	TotalCost  float64  // biaya crashing kumulatif
	Marginal   float64  // sama dengan StepCost; dinamai terpisah agar maknanya jelas
	CriticalAt int      // jumlah simpul kritis sebelum langkah
}

// Curve adalah kurva waktu-biaya hasil crashing.
type Curve struct {
	NormalDuration int
	MinDuration    int
	Steps          []Step
	// Exhausted menjelaskan mengapa crashing berhenti.
	Exhausted model.Text
}

// CostToSave mengembalikan biaya crashing kumulatif untuk memotong n hari.
// Mengembalikan false bila n melebihi kemampuan kompresi jaringan.
func (c Curve) CostToSave(n int) (float64, bool) {
	if n <= 0 {
		return 0, true
	}
	if n > len(c.Steps) {
		return 0, false
	}
	return c.Steps[n-1].TotalCost, true
}

// Crash menjalankan crashing serakah: pada setiap langkah, cari cara termurah
// untuk memotong durasi proyek tepat satu hari, lalu terapkan.
//
// Kalau hanya ada satu jalur kritis, cukup memotong satu aktivitas kritis
// termurah. Tetapi jaringan ini punya jalur kritis paralel di fase analisis
// (A03/A04 dan A05/A06); memotong salah satunya saja tidak memendekkan proyek
// sama sekali. Karena itu pencarian dilanjutkan ke pasangan, lalu ke tiga
// aktivitas sekaligus. Jaringan 40 simpul cukup kecil untuk pencarian ini,
// dan hasilnya jujur - tidak ada hari yang "dibeli" tetapi tidak benar-benar
// memendekkan proyek.
//
// Serakah per hari tidak menjamin kurva optimum global (pemecahan eksaknya
// adalah pemrograman linear). Untuk jaringan dengan slope linear dan satu
// langkah per hari, selisihnya kecil, dan heuristik ini dinyatakan terbuka.
func Crash(acts []model.Activity, rates map[model.Role]float64) (Curve, error) {
	dur := make(map[string]int, len(acts))
	plans := make(map[string]model.CrashPlan, len(acts))
	for _, a := range acts {
		dur[a.ID] = a.Duration
		plans[a.ID] = a.Crash(rates)
	}
	durOf := func(a model.Activity) int { return dur[a.ID] }

	base, err := schedule.Compute(acts, schedule.Options{DurationOf: durOf})
	if err != nil {
		return Curve{}, err
	}
	curve := Curve{NormalDuration: base.Duration, MinDuration: base.Duration}
	var total float64

	for guard := 0; guard < 200; guard++ {
		cur, err := schedule.Compute(acts, schedule.Options{DurationOf: durOf})
		if err != nil {
			return Curve{}, err
		}

		var cand []string
		critical := 0
		for _, a := range acts {
			t := cur.Task(a.ID)
			if !t.Critical {
				continue
			}
			critical++
			p := plans[a.ID]
			if p.Allowed && dur[a.ID] > p.CrashDur {
				cand = append(cand, a.ID)
			}
		}
		sort.Slice(cand, func(i, j int) bool {
			ci, cj := plans[cand[i]].SlopePerDay, plans[cand[j]].SlopePerDay
			if ci != cj {
				return ci < cj
			}
			return cand[i] < cand[j]
		})

		try := func(ids []string) (bool, error) {
			for _, id := range ids {
				dur[id]--
			}
			r, err := schedule.Compute(acts, schedule.Options{DurationOf: durOf})
			for _, id := range ids {
				dur[id]++
			}
			if err != nil {
				return false, err
			}
			return r.Duration < cur.Duration, nil
		}
		cost := func(ids []string) float64 {
			var c float64
			for _, id := range ids {
				c += plans[id].SlopePerDay
			}
			return c
		}

		var best []string
		bestCost := math.Inf(1)
		consider := func(ids []string) error {
			c := cost(ids)
			if c >= bestCost {
				return nil
			}
			ok, err := try(ids)
			if err != nil {
				return err
			}
			if ok {
				best = append([]string(nil), ids...)
				bestCost = c
			}
			return nil
		}

		for _, id := range cand {
			if err := consider([]string{id}); err != nil {
				return Curve{}, err
			}
		}
		if best == nil {
			for i := 0; i < len(cand); i++ {
				for j := i + 1; j < len(cand); j++ {
					if err := consider([]string{cand[i], cand[j]}); err != nil {
						return Curve{}, err
					}
				}
			}
		}
		if best == nil {
			for i := 0; i < len(cand); i++ {
				for j := i + 1; j < len(cand); j++ {
					for k := j + 1; k < len(cand); k++ {
						if err := consider([]string{cand[i], cand[j], cand[k]}); err != nil {
							return Curve{}, err
						}
					}
				}
			}
		}
		if best == nil {
			if len(cand) == 0 {
				curve.Exhausted = model.Text{
					ID: "Seluruh aktivitas kritis sudah di batas durasi crash atau memang tidak bisa dipercepat dengan uang.",
					EN: "Every critical activity is already at its crash limit or cannot be accelerated with money.",
				}
			} else {
				curve.Exhausted = model.Text{
					ID: "Masih ada aktivitas kritis yang bisa dipotong, tetapi jalur kritis paralel lain sudah tidak bisa ikut dipendekkan - memotong lebih jauh tidak mengubah tanggal selesai.",
					EN: "Some critical activities could still be cut, but a parallel critical path can no longer shrink - cutting further would not move the finish date.",
				}
			}
			break
		}

		for _, id := range best {
			dur[id]--
		}
		total += bestCost
		sort.Strings(best)
		curve.Steps = append(curve.Steps, Step{
			Duration: cur.Duration - 1, Crashed: best,
			StepCost: bestCost, TotalCost: total, Marginal: bestCost,
			CriticalAt: critical,
		})
		curve.MinDuration = cur.Duration - 1
	}
	return curve, nil
}

// FastTrack adalah satu kandidat tumpang tindih: aktivitas penerus dimulai
// sebelum pendahulunya selesai.
type FastTrack struct {
	Pred, Succ     string
	OverlapDays    int
	DaysSaved      int
	ReworkProb     float64
	ExpectedRework float64 // rupiah
	CostPerDay     float64 // rework harapan per hari yang dihemat
	SameWBSPhase   bool
	// SameResource bernilai true bila pendahulu dan penerus dikerjakan peran
	// dominan yang sama. Fast-tracking semacam itu tidak mungkin secara fisik:
	// satu orang tidak bisa mengerjakan dua pekerjaan tumpang tindih, jadi
	// "penghematan" CPM-nya hanya ilusi yang akan hilang saat levelling.
	SameResource bool
}

// Viable melaporkan apakah kandidat benar-benar bisa dijalankan.
func (f FastTrack) Viable() bool { return f.DaysSaved > 0 && !f.SameResource }

// ReworkProbability adalah peluang pekerjaan tumpang tindih harus diulang
// sebagian, sebagai ASUMSI tunggal: 30%. Nilai ini terlihat di halaman
// Metode, dan urutan kandidatnya tidak bergantung padanya karena semua
// kandidat dikalikan angka yang sama.
const ReworkProbability = 0.30

// FastTrackCandidates menilai setiap relasi FS di jalur kritis sebagai
// kandidat fast-tracking: relasinya diganti SS dengan lag setengah durasi
// pendahulu (tumpang tindih 50%), lalu CPM dihitung ulang.
//
// Rework harapan dihitung sebagai peluang rework x hari tumpang tindih x
// biaya tenaga kerja harian penerus - pekerjaan yang dikerjakan di atas
// masukan yang belum final, dan sebagian harus diulang.
//
// Relasi yang melintasi milestone persetujuan (M1, M2, M3, M4) sengaja tidak
// dijadikan kandidat: tumpang tindih melewati gerbang persetujuan bukan
// fast-tracking, melainkan melompati kendali proyek.
func FastTrackCandidates(acts []model.Activity, rates map[model.Role]float64) ([]FastTrack, error) {
	base, err := schedule.Compute(acts, schedule.Options{})
	if err != nil {
		return nil, err
	}
	byID := model.ActivityByID()
	for _, a := range acts {
		byID[a.ID] = a
	}

	var out []FastTrack
	for ai, a := range acts {
		if a.Milestone || !base.Task(a.ID).Critical {
			continue
		}
		for pi, p := range a.Pred {
			pred := byID[p.ID]
			if pred.Milestone || (p.Type != "" && p.Type != "FS") || !base.Task(p.ID).Critical {
				continue
			}
			overlap := (pred.Duration + 1) / 2
			if overlap < 1 {
				continue
			}
			mod := make([]model.Activity, len(acts))
			copy(mod, acts)
			preds := append([]model.Predecessor(nil), a.Pred...)
			preds[pi] = model.Predecessor{ID: p.ID, Type: "SS", Lag: pred.Duration - overlap}
			mod[ai].Pred = preds

			r, err := schedule.Compute(mod, schedule.Options{})
			if err != nil {
				return nil, err
			}
			saved := base.Duration - r.Duration
			daily := a.LabourCost(rates) / float64(a.Duration)
			ft := FastTrack{
				Pred: p.ID, Succ: a.ID, OverlapDays: overlap, DaysSaved: saved,
				ReworkProb:     ReworkProbability,
				ExpectedRework: ReworkProbability * float64(overlap) * daily,
				SameWBSPhase:   model.PhaseOf(pred.WBS).Code == model.PhaseOf(a.WBS).Code,
				SameResource:   pred.DominantRole() != "" && pred.DominantRole() == a.DominantRole(),
			}
			if saved > 0 {
				ft.CostPerDay = ft.ExpectedRework / float64(saved)
			}
			out = append(out, ft)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Viable() != out[j].Viable() {
			return out[i].Viable()
		}
		if out[i].DaysSaved != out[j].DaysSaved {
			return out[i].DaysSaved > out[j].DaysSaved
		}
		if out[i].CostPerDay != out[j].CostPerDay {
			return out[i].CostPerDay < out[j].CostPerDay
		}
		return out[i].Succ < out[j].Succ
	})
	return out, nil
}

// ViableCandidates menyaring kandidat yang benar-benar bisa dijalankan.
func ViableCandidates(all []FastTrack) []FastTrack {
	var out []FastTrack
	for _, f := range all {
		if f.Viable() {
			out = append(out, f)
		}
	}
	return out
}

// ApplyFastTracks menerapkan sekumpulan kandidat sekaligus dan mengembalikan
// durasi proyek hasilnya. Penghematan kandidat tidak bisa dijumlahkan begitu
// saja: memendekkan satu jalur bisa membuat jalur lain menjadi kritis.
func ApplyFastTracks(acts []model.Activity, picks []FastTrack) (int, error) {
	mod := make([]model.Activity, len(acts))
	copy(mod, acts)
	idx := map[string]int{}
	for i, a := range mod {
		idx[a.ID] = i
	}
	byID := model.ActivityByID()
	for _, ft := range picks {
		i, ok := idx[ft.Succ]
		if !ok {
			continue
		}
		preds := append([]model.Predecessor(nil), mod[i].Pred...)
		for k, p := range preds {
			if p.ID == ft.Pred {
				preds[k] = model.Predecessor{ID: p.ID, Type: "SS", Lag: byID[p.ID].Duration - ft.OverlapDays}
			}
		}
		mod[i].Pred = preds
	}
	r, err := schedule.Compute(mod, schedule.Options{})
	if err != nil {
		return 0, err
	}
	return r.Duration, nil
}
