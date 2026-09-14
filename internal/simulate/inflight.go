package simulate

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Prakiraan berjalan (in-flight forecast) menjalankan simulasi terpadu dari
// TANGGAL DATA, bukan dari hari pertama proyek.
//
// Pandangan perencanaan menjawab "kalau proyek ini dimulai, bagaimana
// hasilnya?". Setelah proyek berjalan, pertanyaannya berubah: "dengan apa yang
// sudah terjadi, bagaimana sisanya?". Tiga hal membedakannya:
//
//  1. Realisasi dikunci. Aktivitas yang sudah selesai memakai tanggal dan
//     biaya aktualnya; aktivitas yang sedang berjalan tidak mungkin berdurasi
//     kurang dari hari yang sudah dilaluinya, jadi durasinya diambil dari
//     sebaran bersyarat F(x | x > e).
//  2. Estimasi belajar dari realisasi lewat bobot kredibilitas Buhlmann:
//     faktor = Z x teramati + (1 - Z) x rencana, dengan Z = n / (n + k).
//     n adalah jumlah bukti, k bobot keyakinan awal. Bukti sedikit berarti
//     rencana masih dominan; bukti banyak berarti data yang bicara.
//  3. Yang sudah lewat ditutup: risiko berstatus "terjadi" atau "tertutup"
//     sudah tercermin di realisasi, dan putaran rework yang pemeriksaannya
//     sudah selesai tidak bisa berulang lagi. Risiko yang masih terbuka tetap
//     bisa terjadi; dampak jadwalnya dipindah ke pekerjaan yang belum selesai.

// Status aktivitas pada tanggal data.
const (
	NotStarted = iota
	InProgress
	Completed
)

// DefaultPriorWeight adalah bobot keyakinan awal k pada kredibilitas Buhlmann,
// dalam satuan "aktivitas". 10 berarti rencana dihargai setara sepuluh
// aktivitas yang sudah terbukti. ASUMSI, dinyatakan di halaman Metode.
const DefaultPriorWeight = 10.0

// ActState adalah status satu aktivitas pada tanggal data.
type ActState struct {
	Kind    int
	Start   int     // hari mulai aktual (Completed/InProgress)
	Finish  int     // hari selesai aktual, eksklusif (Completed)
	Elapsed float64 // hari yang sudah dilalui (InProgress)
}

// InFlight adalah keadaan proyek pada tanggal data yang siap disimulasikan.
type InFlight struct {
	StatusDay float64
	Now       int // hari kerja pertama yang belum lewat

	State      []ActState
	Completed  []string
	InProgress []string
	NotStarted []string

	// Residual adalah jaringan untuk dijadwalkan ulang: aktivitas selesai
	// menjadi simpul tanpa durasi dan tanpa tim yang dirilis pada hari
	// selesainya; sisanya dirilis paling awal pada Now.
	Residual []model.Activity
	Release  []int

	// Kalibrasi durasi dari aktivitas yang sudah selesai.
	Evidence       int
	PriorWeight    float64
	Credibility    float64 // Z
	ObservedRatio  float64 // jumlah durasi aktual / jumlah rerata PERT
	DurationFactor float64

	// Kalibrasi biaya tenaga kerja per hari.
	ObservedCostRatio float64 // biaya tenaga kerja aktual / (tarif harian x durasi aktual)
	CostFactor        float64

	// Kalibrasi kapasitas saat ujian dari jendela yang sudah lewat.
	ExamEvidence    int
	ExamObserved    float64 // laju saat ujian / laju di luar ujian, teramati
	ExamCredibility float64
	ExamFactor      float64 // pengganti model.ExamCapacityFactor

	ACToDate float64
	// ClosedRisk memuat risiko berstatus "terjadi" atau "tertutup" pada
	// register. Status register adalah otoritasnya: risiko terbuka tetap bisa
	// terjadi walaupun paket kerja tempat sebabnya lahir sudah selesai.
	ClosedRisk map[string]bool
	// Exposed memetakan risiko terbuka ke aktivitas yang menanggung hari
	// tambahannya. Bila aktivitas asal sudah selesai, dampaknya dipindahkan ke
	// aktivitas belum-selesai pertama pada paket kerja yang sama atau sesudahnya.
	Exposed    map[string]int
	ClosedLoop map[string]bool

	fixedCost    float64 // AC + biaya non-sewa aktivitas yang belum mulai
	timeCostPaid float64 // nilai rencana biaya sewa yang sudah ada di AC
}

// PrepareInFlight membaca realisasi pada model.Activities sampai statusDay.
// acAt mengembalikan Actual Cost kumulatif pada hari t (dari mesin EVM),
// supaya prakiraan dan halaman Earned Value berangkat dari AC yang sama.
func PrepareInFlight(acts []model.Activity, cal *workcal.Calendar, statusDay float64, priorWeight float64, acAt func(float64) float64) (*InFlight, error) {
	if cal == nil {
		return nil, fmt.Errorf("simulate: prakiraan berjalan butuh kalender")
	}
	if priorWeight <= 0 {
		priorWeight = DefaultPriorWeight
	}
	fl := &InFlight{
		StatusDay:   statusDay,
		Now:         int(math.Ceil(statusDay - 1e-9)),
		State:       make([]ActState, len(acts)),
		Residual:    make([]model.Activity, len(acts)),
		Release:     make([]int, len(acts)),
		PriorWeight: priorWeight,
		ClosedRisk:  map[string]bool{},
		ClosedLoop:  map[string]bool{},
	}
	sampler := NewSampler(acts)

	var sumActual, sumMean, sumLabourActual, sumLabourPlanRate float64
	for i, a := range acts {
		st := ActState{Kind: NotStarted}
		act := a.Actual
		if act.Started && float64(act.Start) < statusDay+1e-9 {
			end := act.Start + act.Duration
			switch {
			case a.Milestone || float64(end) <= statusDay+1e-9:
				st = ActState{Kind: Completed, Start: act.Start, Finish: end}
			default:
				st = ActState{Kind: InProgress, Start: act.Start, Elapsed: statusDay - float64(act.Start)}
			}
		}
		fl.State[i] = st

		r := a
		switch st.Kind {
		case Completed:
			fl.Completed = append(fl.Completed, a.ID)
			r.Duration, r.Optimistic, r.Pessimistic = 0, 0, 0
			r.Team = nil
			fl.Release[i] = st.Finish
			if !a.Milestone {
				sumActual += float64(act.Duration)
				sumMean += sampler.PERTMean(i)
				fl.Evidence++
				var rate float64
				for _, s := range a.Team {
					rate += model.RateCard[s.Role] * s.Alloc
				}
				sumLabourActual += act.Cost - a.ExtraCost()
				sumLabourPlanRate += rate * float64(act.Duration)
			}
		case InProgress:
			fl.InProgress = append(fl.InProgress, a.ID)
			fl.Release[i] = fl.Now
		default:
			fl.NotStarted = append(fl.NotStarted, a.ID)
			fl.Release[i] = fl.Now
		}
		fl.Residual[i] = r
	}

	if sumMean > 0 {
		fl.ObservedRatio = sumActual / sumMean
	} else {
		fl.ObservedRatio = 1
	}
	fl.Credibility = float64(fl.Evidence) / (float64(fl.Evidence) + priorWeight)
	fl.DurationFactor = fl.Credibility*fl.ObservedRatio + (1 - fl.Credibility)
	if sumLabourPlanRate > 0 {
		fl.ObservedCostRatio = sumLabourActual / sumLabourPlanRate
	} else {
		fl.ObservedCostRatio = 1
	}
	fl.CostFactor = fl.Credibility*fl.ObservedCostRatio + (1 - fl.Credibility)

	fl.calibrateExam(acts, cal, sampler, priorWeight)

	if acAt != nil {
		fl.ACToDate = acAt(statusDay)
	}
	fl.fixedCost = fl.ACToDate
	for i, a := range acts {
		if fl.State[i].Kind != NotStarted {
			for _, e := range a.Extras {
				if e.TimeBased {
					fl.timeCostPaid += e.Amount
				}
			}
			continue
		}
		for _, e := range a.Extras {
			if !e.TimeBased {
				fl.fixedCost += e.Amount
			}
		}
	}

	fl.Exposed = exposure(acts)
	for _, r := range model.Risks {
		if r.Status == "terjadi" || r.Status == "tertutup" {
			fl.ClosedRisk[r.ID] = true
			continue
		}
		if fl.State[fl.Exposed[r.ID]].Kind != Completed {
			continue
		}
		best := -1
		for i, a := range acts {
			if a.Milestone || fl.State[i].Kind == Completed || a.WBS < r.WBS {
				continue
			}
			if best < 0 || a.WBS < acts[best].WBS || (a.WBS == acts[best].WBS && a.Duration > acts[best].Duration) {
				best = i
			}
		}
		if best >= 0 {
			fl.Exposed[r.ID] = best
		} else {
			fl.ClosedRisk[r.ID] = true
		}
	}
	idx := make(map[string]int, len(acts))
	for i, a := range acts {
		idx[a.ID] = i
	}
	for _, l := range model.ReworkLoops {
		if i, ok := idx[l.Check]; ok && fl.State[i].Kind == Completed {
			fl.ClosedLoop[l.ID] = true
		}
	}
	sort.Strings(fl.Completed)
	sort.Strings(fl.InProgress)
	sort.Strings(fl.NotStarted)
	return fl, nil
}

// calibrateExam membandingkan laju kerja aktivitas yang beririsan dengan
// jendela ujian yang sudah lewat terhadap laju aktivitas lain. Laju sebuah
// aktivitas adalah rerata PERT-nya dibagi durasi aktualnya: 1 berarti sesuai
// estimasi, di bawah 1 berarti lebih lambat.
func (fl *InFlight) calibrateExam(acts []model.Activity, cal *workcal.Calendar, sampler *Sampler, k float64) {
	type win struct{ from, to int }
	var past []win
	for _, w := range model.AvailabilityWindows {
		to := cal.IndexOf(w.To)
		if w.Factor >= 1 || float64(to) > fl.StatusDay {
			continue
		}
		from := cal.IndexOf(w.From)
		// IndexOf membulatkan tanggal non-kerja ke hari kerja berikutnya;
		// batas atas eksklusif adalah hari kerja setelah tanggal akhir.
		if cal.ISOAt(to) != w.To {
			to--
		}
		past = append(past, win{from, to + 1})
	}
	fl.ExamObserved = 1
	if len(past) == 0 {
		fl.ExamFactor = model.ExamCapacityFactor
		return
	}
	var inW, inDays, outW, outDays float64
	for i, a := range acts {
		st := fl.State[i]
		if st.Kind != Completed || a.Milestone || a.Actual.Duration <= 0 {
			continue
		}
		rate := sampler.PERTMean(i) / float64(a.Actual.Duration)
		overlap := 0
		for _, w := range past {
			lo, hi := st.Start, st.Finish
			if w.from > lo {
				lo = w.from
			}
			if w.to < hi {
				hi = w.to
			}
			if hi > lo {
				overlap += hi - lo
			}
		}
		if overlap > 0 {
			fl.ExamEvidence++
			inW += rate * float64(overlap)
			inDays += float64(overlap)
			outW += rate * float64(a.Actual.Duration-overlap)
			outDays += float64(a.Actual.Duration - overlap)
			continue
		}
		outW += rate * float64(a.Actual.Duration)
		outDays += float64(a.Actual.Duration)
	}
	if inDays > 0 && outDays > 0 && outW > 0 {
		fl.ExamObserved = (inW / inDays) / (outW / outDays)
	}
	obs := math.Min(1, fl.ExamObserved)
	fl.ExamCredibility = float64(fl.ExamEvidence) / (float64(fl.ExamEvidence) + k)
	fl.ExamFactor = fl.ExamCredibility*obs + (1-fl.ExamCredibility)*model.ExamCapacityFactor
}

// duration mengubah sampel satu aktivitas menjadi durasi pada jaringan sisa.
func (fl *InFlight) duration(s *Sampler, i int, drawn int) int {
	st := fl.State[i]
	a := s.acts[i]
	switch st.Kind {
	case Completed:
		return 0
	case InProgress:
		if a.Pessimistic <= a.Optimistic {
			rem := float64(a.Duration) - st.Elapsed
			return int(math.Max(1, math.Round(rem*fl.DurationFactor)))
		}
		// Durasi bersyarat: u' = F(e) + u (1 - F(e)).
		fe := s.CDF(i, st.Elapsed)
		u := fe + s.U(i)*(1-fe)
		if u > 1-1e-12 {
			u = 1 - 1e-12
		}
		total := s.Value(i, u, "pert")
		rem := (total - st.Elapsed) * fl.DurationFactor
		return int(math.Max(1, math.Round(rem)))
	default:
		if a.Milestone {
			return 0
		}
		if a.Pessimistic <= a.Optimistic {
			return int(math.Max(1, math.Round(float64(a.Duration)*fl.DurationFactor)))
		}
		v := s.Value(i, s.U(i), "pert") * fl.DurationFactor
		return int(math.Max(1, math.Round(v)))
	}
}

// laborFactor mengembalikan pengali biaya tenaga kerja per hari sisa.
func (fl *InFlight) laborFactor(i int) float64 {
	if fl.State[i].Kind == Completed {
		return 0
	}
	return fl.CostFactor
}
