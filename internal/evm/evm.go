// Package evm mengimplementasikan Earned Value Management.
//
// Tiga besaran pokok, semuanya dalam rupiah dan dievaluasi pada satu tanggal
// data (data date):
//
//	PV (Planned Value)  - nilai pekerjaan yang SEHARUSNYA sudah selesai
//	EV (Earned Value)   - nilai pekerjaan yang NYATANYA sudah selesai
//	AC (Actual Cost)    - uang yang NYATANYA sudah dikeluarkan
//
// Kemajuan di dalam satu aktivitas diinterpolasi linear terhadap durasinya.
// Ini adalah aturan "percent complete" paling sederhana dan paling bisa
// diaudit; aturan lain (0/100, 50/50, milestone weighting) akan memberi angka
// berbeda dan harus disepakati di awal proyek, bukan dipilih belakangan.
package evm

import (
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// Snapshot adalah potret Earned Value pada satu tanggal data.
type Snapshot struct {
	AtDay float64 // indeks hari kerja (boleh pecahan)

	PV  float64
	EV  float64
	AC  float64
	BAC float64

	SV float64 // Schedule Variance = EV - PV
	CV float64 // Cost Variance     = EV - AC

	SPI float64 // Schedule Performance Index = EV / PV
	CPI float64 // Cost Performance Index     = EV / AC

	// Earned Schedule: titik waktu pada kurva PV yang nilainya sama dengan EV
	// sekarang. Lebih jujur daripada SPI berbasis rupiah, karena SPI selalu
	// kembali ke 1,0 di akhir proyek walaupun proyeknya telat berbulan-bulan.
	ES   float64
	SVt  float64 // Schedule Variance (waktu) = ES - AtDay, dalam hari kerja
	SPIt float64 // SPI berbasis waktu = ES / AtDay

	// Perkiraan biaya akhir. Tiga rumus PMBOK dengan asumsi berbeda.
	EACOptimistic  float64 // BAC - EV + AC : varians dianggap tidak terulang
	EACTypical     float64 // BAC / CPI     : varians dianggap terus berlanjut
	EACPessimistic float64 // AC + (BAC-EV)/(CPI*SPI) : telat DAN boros berlanjut

	ETC  float64 // Estimate to Complete berdasarkan EACTypical
	VAC  float64 // Variance at Completion = BAC - EACTypical
	TCPI float64 // efisiensi yang harus dicapai agar tetap di dalam BAC

	PercentComplete float64 // EV / BAC
	PercentSpent    float64 // AC / BAC
	PercentElapsed  float64 // AtDay / durasi rencana
}

// Curves adalah deret waktu PV/EV/AC untuk menggambar kurva-S.
type Curves struct {
	Days []float64
	PV   []float64
	EV   []float64
	AC   []float64
	// EV dan AC berhenti di tanggal data - proyeksi ke depan digambar
	// terpisah supaya tidak tertukar dengan kenyataan.
	CutoffIndex int
	Forecast    []float64 // proyeksi AC memakai EACTypical
}

// Engine menyimpan data yang dipakai berulang kali agar pergeseran tanggal
// data tidak perlu menghitung ulang CPM setiap saat.
type Engine struct {
	activities []model.Activity
	plan       schedule.Result
	rates      map[model.Role]float64
	budgets    map[string]float64
	bac        float64
	planDays   int
}

// New membangun engine EVM untuk sebuah jaringan dan jadwal rencana.
func New(activities []model.Activity, plan schedule.Result, rates map[model.Role]float64) *Engine {
	budgets := make(map[string]float64, len(activities))
	var bac float64
	for _, a := range activities {
		b := a.Budget(rates)
		budgets[a.ID] = b
		bac += b
	}
	return &Engine{
		activities: activities,
		plan:       plan,
		rates:      rates,
		budgets:    budgets,
		bac:        bac,
		planDays:   plan.Duration,
	}
}

// BAC mengembalikan Budget at Completion.
func (e *Engine) BAC() float64 { return e.bac }

// PlanDays mengembalikan durasi rencana dalam hari kerja.
func (e *Engine) PlanDays() int { return e.planDays }

// plannedProgress mengembalikan porsi aktivitas yang seharusnya rampung pada
// hari ke-t menurut jadwal rencana.
func (e *Engine) plannedProgress(a model.Activity, t float64) float64 {
	task, ok := e.plan.Tasks[a.ID]
	if !ok {
		return 0
	}
	if a.Milestone || task.Duration == 0 {
		if t >= float64(task.StartX) {
			return 1
		}
		return 0
	}
	return clamp01((t - float64(task.StartX)) / float64(task.Duration))
}

// actualProgress mengembalikan porsi aktivitas yang nyatanya rampung pada
// hari ke-t menurut realisasi lapangan.
func (e *Engine) actualProgress(a model.Activity, t float64) float64 {
	if !a.Actual.Started {
		return 0
	}
	if a.Milestone || a.Actual.Duration == 0 {
		if t >= float64(a.Actual.Start) {
			return 1
		}
		return 0
	}
	return clamp01((t - float64(a.Actual.Start)) / float64(a.Actual.Duration))
}

// PVAt menghitung Planned Value kumulatif pada hari ke-t.
func (e *Engine) PVAt(t float64) float64 {
	var pv float64
	for _, a := range e.activities {
		pv += e.budgets[a.ID] * e.plannedProgress(a, t)
	}
	return pv
}

// EVAt menghitung Earned Value kumulatif pada hari ke-t.
func (e *Engine) EVAt(t float64) float64 {
	var ev float64
	for _, a := range e.activities {
		ev += e.budgets[a.ID] * e.actualProgress(a, t)
	}
	return ev
}

// ACAt menghitung Actual Cost kumulatif pada hari ke-t.
func (e *Engine) ACAt(t float64) float64 {
	var ac float64
	for _, a := range e.activities {
		ac += a.Actual.Cost * e.actualProgress(a, t)
	}
	return ac
}

// earnedSchedule mencari titik waktu pada kurva PV yang nilainya sama dengan
// ev, lewat pencarian biner lalu interpolasi linear di antara dua hari.
func (e *Engine) earnedSchedule(ev float64) float64 {
	if ev <= 0 {
		return 0
	}
	total := e.PVAt(float64(e.planDays))
	if ev >= total {
		return float64(e.planDays)
	}
	lo, hi := 0, e.planDays
	for lo < hi {
		mid := (lo + hi) / 2
		if e.PVAt(float64(mid)) < ev {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return 0
	}
	prev := e.PVAt(float64(lo - 1))
	cur := e.PVAt(float64(lo))
	if cur == prev {
		return float64(lo)
	}
	return float64(lo-1) + (ev-prev)/(cur-prev)
}

// At menghitung seluruh metrik Earned Value pada hari ke-t.
func (e *Engine) At(t float64) Snapshot {
	pv := e.PVAt(t)
	ev := e.EVAt(t)
	ac := e.ACAt(t)

	s := Snapshot{
		AtDay: t,
		PV:    pv, EV: ev, AC: ac, BAC: e.bac,
		SV: ev - pv,
		CV: ev - ac,
	}
	s.SPI = safeDiv(ev, pv)
	s.CPI = safeDiv(ev, ac)

	s.ES = e.earnedSchedule(ev)
	s.SVt = s.ES - t
	s.SPIt = safeDiv(s.ES, t)

	s.EACOptimistic = e.bac - ev + ac
	if s.CPI > 0 {
		s.EACTypical = e.bac / s.CPI
	} else {
		s.EACTypical = e.bac
	}
	if s.CPI > 0 && s.SPI > 0 {
		s.EACPessimistic = ac + (e.bac-ev)/(s.CPI*s.SPI)
	} else {
		s.EACPessimistic = s.EACTypical
	}

	s.ETC = s.EACTypical - ac
	s.VAC = e.bac - s.EACTypical
	// TCPI dihitung terhadap BAC: seberapa efisien sisa pekerjaan harus
	// dikerjakan agar proyek tetap mendarat di anggaran semula.
	if denom := e.bac - ac; denom != 0 {
		s.TCPI = (e.bac - ev) / denom
	}

	s.PercentComplete = safeDiv(ev, e.bac)
	s.PercentSpent = safeDiv(ac, e.bac)
	s.PercentElapsed = safeDiv(t, float64(e.planDays))
	return s
}

// BuildCurves membuat deret PV/EV/AC per hari kerja untuk kurva-S.
func (e *Engine) BuildCurves(statusDay float64) Curves {
	n := e.planDays
	c := Curves{CutoffIndex: int(math.Floor(statusDay))}
	snap := e.At(statusDay)
	for d := 0; d <= n; d++ {
		t := float64(d)
		c.Days = append(c.Days, t)
		c.PV = append(c.PV, e.PVAt(t))
		if t <= statusDay {
			c.EV = append(c.EV, e.EVAt(t))
			c.AC = append(c.AC, e.ACAt(t))
		}
	}
	// Proyeksi biaya: dari titik (statusDay, AC) menuju (durasi, EACTypical),
	// mengikuti bentuk sisa kurva PV agar proyeksinya tidak berupa garis lurus
	// yang menyesatkan.
	pvNow := e.PVAt(statusDay)
	pvEnd := e.PVAt(float64(n))
	acNow := snap.AC
	remaining := snap.EACTypical - acNow
	for d := 0; d <= n; d++ {
		t := float64(d)
		if t < statusDay {
			c.Forecast = append(c.Forecast, math.NaN())
			continue
		}
		var share float64
		if pvEnd > pvNow {
			share = (e.PVAt(t) - pvNow) / (pvEnd - pvNow)
		} else if t >= float64(n) {
			share = 1
		}
		c.Forecast = append(c.Forecast, acNow+remaining*clamp01(share))
	}
	return c
}

// PhaseRow adalah ringkasan Earned Value untuk satu fase WBS.
type PhaseRow struct {
	Code    string
	Name    model.Text
	PV      float64
	EV      float64
	AC      float64
	SV      float64
	CV      float64
	SPI     float64
	CPI     float64
	Budget  float64
	Percent float64
}

// ByPhase memecah metrik Earned Value per fase WBS level 1.
func (e *Engine) ByPhase(t float64) []PhaseRow {
	agg := map[string]*PhaseRow{}
	var codes []string
	for _, a := range e.activities {
		phase := model.PhaseOf(a.WBS)
		if phase.Code == "" {
			continue
		}
		row, ok := agg[phase.Code]
		if !ok {
			row = &PhaseRow{Code: phase.Code, Name: phase.Name}
			agg[phase.Code] = row
			codes = append(codes, phase.Code)
		}
		b := e.budgets[a.ID]
		row.Budget += b
		row.PV += b * e.plannedProgress(a, t)
		row.EV += b * e.actualProgress(a, t)
		row.AC += a.Actual.Cost * e.actualProgress(a, t)
	}
	sort.Strings(codes)
	out := make([]PhaseRow, 0, len(codes))
	for _, code := range codes {
		r := agg[code]
		r.SV = r.EV - r.PV
		r.CV = r.EV - r.AC
		r.SPI = safeDiv(r.EV, r.PV)
		r.CPI = safeDiv(r.EV, r.AC)
		r.Percent = safeDiv(r.EV, r.Budget)
		out = append(out, *r)
	}
	return out
}

// ActivityRow adalah baris rinci Earned Value untuk satu aktivitas.
type ActivityRow struct {
	ID              string
	Name            model.Text
	WBS             string
	Budget          float64
	PV              float64
	EV              float64
	AC              float64
	PlannedProgress float64
	ActualProgress  float64
	CV              float64
	SV              float64
	CPI             float64
	Status          string // "selesai", "berjalan", "belum", "telat"
}

// ByActivity memberi rincian Earned Value per aktivitas pada hari ke-t.
func (e *Engine) ByActivity(t float64) []ActivityRow {
	rows := make([]ActivityRow, 0, len(e.activities))
	for _, a := range e.activities {
		b := e.budgets[a.ID]
		pp := e.plannedProgress(a, t)
		ap := e.actualProgress(a, t)
		row := ActivityRow{
			ID: a.ID, Name: a.Name, WBS: a.WBS, Budget: b,
			PV: b * pp, EV: b * ap, AC: a.Actual.Cost * ap,
			PlannedProgress: pp, ActualProgress: ap,
		}
		row.CV = row.EV - row.AC
		row.SV = row.EV - row.PV
		row.CPI = safeDiv(row.EV, row.AC)
		switch {
		case ap >= 1:
			row.Status = "selesai"
		case ap > 0:
			row.Status = "berjalan"
		case pp > 0:
			row.Status = "telat"
		default:
			row.Status = "belum"
		}
		rows = append(rows, row)
	}
	return rows
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
