// Package site merakit seluruh model data dan hasil hitungan menjadi struktur
// yang siap dipakai templat, lalu menuliskannya sebagai situs statis.
//
// Semua perhitungan dijalankan SEKALI di sini dan dibagikan ke seluruh
// halaman. Halaman biaya dan halaman risiko harus melihat angka BAC yang sama
// persis; cara paling pasti menjamin itu bukan disiplin penulis templat,
// melainkan menghitungnya sekali lalu meneruskan hasilnya.
package site

import (
	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/coretax"
	"github.com/xyb3rpunq/mppl-control-tower/internal/cost"
	"github.com/xyb3rpunq/mppl-control-tower/internal/evm"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/quality"
	"github.com/xyb3rpunq/mppl-control-tower/internal/resource"
	"github.com/xyb3rpunq/mppl-control-tower/internal/risk"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Analysis memuat seluruh hasil hitungan proyek pada satu tanggal data.
type Analysis struct {
	Calendar  *workcal.Calendar
	Plan      schedule.Result
	PERT      schedule.PERTStats
	EVM       *evm.Engine
	Snapshot  evm.Snapshot
	Curves    evm.Curves
	Phases    []evm.PhaseRow
	Rows      []evm.ActivityRow
	Sim       simulate.Result
	Risk      risk.Register
	Resources resource.Profile
	Pareto    quality.Pareto
	Control   quality.ControlChart
	COQ       quality.COQSummary
	Metrics   []quality.MetricStatus
	Coretax   coretax.Derived
	Scenarios []coretax.Scenario

	StatusDate  string
	StatusDay   float64
	BAC         float64
	Baseline    float64
	MgmtReserve float64

	// Levelling sumber daya.
	Level        level.Result
	LevelOpt     level.Optimized // pencarian jadwal terbaik + batas bawah
	LevelWhy     level.Breakdown
	LevelProfile resource.Profile
	CriticalRole model.Role // peran yang paling banyak membuat pekerjaan menunggu
	RoleWait     map[model.Role]int

	// Kompresi jadwal.
	Crash      compress.Curve
	FastTracks []compress.FastTrack
	FastViable []compress.FastTrack
	FastAllDur int // durasi bila seluruh kandidat layak diterapkan

	// Simulasi terpadu.
	Ladder   []simulate.IntegratedResult
	RhoSweep []RhoPoint
	Frontier []simulate.FrontierPoint
	Density  simulate.Grid
	JCL70    simulate.FrontierPoint // titik frontier JCL 70% dengan tenggat terpendek

	// Penutupan celah: crashing eksak, biaya bergantung waktu, risiko
	// bergerombol, putaran GERT, dan prakiraan dari tanggal data.
	Exact     compress.TradeOff
	Rentals   []cost.Rental
	RiskSweep []RiskPoint
	GERT      []GERTRow

	InFlight         *simulate.InFlight
	Forecast         simulate.IntegratedResult // dari tanggal data, estimasi terkalibrasi
	ForecastPrior    simulate.IntegratedResult // dari tanggal data, tanpa belajar dari realisasi
	ForecastFrontier []simulate.FrontierPoint
	ForecastJCL70    simulate.FrontierPoint
	// IEACt adalah prakiraan durasi Earned Schedule: AT + (PD - ES) / SPI(t).
	IEACt float64

	Findings []Finding
}

// RhoPoint adalah hasil simulasi lapisan korelasi untuk satu nilai rho.
type RhoPoint struct {
	Rho      float64
	P80      float64
	StdDev   float64
	Realised float64
	OnTime   float64
}

// Final mengembalikan lapisan simulasi terpadu paling realistis.
func (a *Analysis) Final() simulate.IntegratedResult { return a.Ladder[len(a.Ladder)-1] }

// LadderNames adalah label lapisan simulasi terpadu.
var LadderNames = []model.Text{
	{ID: "Independen", EN: "Independent"},
	{ID: "+ Korelasi peran", EN: "+ Role correlation"},
	{ID: "+ Risiko bergerombol", EN: "+ Clustered risks"},
	{ID: "+ Putaran rework", EN: "+ Rework loops"},
	{ID: "+ Kapasitas & ujian", EN: "+ Capacity & exams"},
}

// FinishISO mengembalikan tanggal hari kerja terakhir untuk durasi tertentu.
func (a *Analysis) FinishISO(days int) string { return a.Calendar.ISOAt(days - 1) }

// Finding adalah satu temuan yang muncul dari angka, bukan dari opini.
// Setiap temuan menyebutkan metrik pemicunya supaya bisa diperiksa ulang.
type Finding struct {
	Key      string
	Severity string // "kritis", "tinggi", "sedang", "baik"
	Title    model.Text
	Detail   model.Text
	Metric   model.Text
	Action   model.Text
	Route    string
}

// Build menjalankan seluruh analisis untuk sebuah tanggal data.
func Build(statusDate string) (*Analysis, error) {
	cal, err := workcal.New(model.ProjectCharter.StartDate, 400)
	if err != nil {
		return nil, err
	}
	plan, err := schedule.Compute(model.Activities, schedule.Options{})
	if err != nil {
		return nil, err
	}
	pert, err := schedule.AnalysePERT(model.Activities, schedule.Options{})
	if err != nil {
		return nil, err
	}
	sim, err := simulate.Run(model.Activities, plan, simulate.Defaults())
	if err != nil {
		return nil, err
	}

	engine := evm.New(model.Activities, plan, model.RateCard)
	statusDay := cal.FractionalIndexOf(statusDate)

	a := &Analysis{
		Calendar:    cal,
		Plan:        plan,
		PERT:        pert,
		EVM:         engine,
		Snapshot:    engine.At(statusDay),
		Curves:      engine.BuildCurves(statusDay),
		Phases:      engine.ByPhase(statusDay),
		Rows:        engine.ByActivity(statusDay),
		Sim:         sim,
		Resources:   resource.Analyse(model.Activities, plan, model.Capacity),
		Pareto:      quality.DefectPareto(model.Defects),
		Metrics:     quality.EvaluateMetrics(model.QualityMetrics),
		Coretax:     coretax.Compute(),
		StatusDate:  statusDate,
		StatusDay:   statusDay,
		BAC:         model.BAC(),
		Baseline:    model.CostBaseline(),
		MgmtReserve: model.ManagementReserve(),
	}
	a.Risk = risk.Analyse(model.Risks, a.BAC, model.ContingencyReserve)
	a.COQ = quality.SummariseCOQ(model.CostOfQuality, a.BAC)
	a.Scenarios = coretax.Scenarios(a.Coretax)

	means := make([]float64, 0, len(model.ResponseSamples))
	ranges := make([]float64, 0, len(model.ResponseSamples))
	for _, s := range model.ResponseSamples {
		means = append(means, s.Mean)
		ranges = append(ranges, s.Range)
	}
	a.Control = quality.BuildControlChart(means, ranges, 5, 3.0, true)

	if err := buildUpgrade(a); err != nil {
		return nil, err
	}

	a.Findings = deriveFindings(a)
	return a, nil
}

// FinishDatePlan mengembalikan tanggal selesai menurut jadwal rencana.
func (a *Analysis) FinishDatePlan() string {
	return a.Calendar.ISOAt(a.Plan.ProjectFinish - 1)
}

// FinishDateAt mengembalikan tanggal selesai untuk sejumlah hari kerja.
func (a *Analysis) FinishDateAt(days float64) string {
	return a.Calendar.ISOAt(int(days) - 1)
}

// HolidaysInPlan mengembalikan hari libur yang jatuh di dalam rentang proyek.
func (a *Analysis) HolidaysInPlan() []workcal.Holiday {
	return a.Calendar.HolidaysBetween(0, a.Plan.ProjectFinish-1)
}

// CriticalCount mengembalikan jumlah simpul kritis dan total simpul.
func (a *Analysis) CriticalCount() (int, int) {
	crit := 0
	for _, t := range a.Plan.Tasks {
		if t.Critical {
			crit++
		}
	}
	return crit, len(a.Plan.Tasks)
}

// Health mengembalikan status kesehatan proyek berdasarkan SPI dan CPI.
// Ambangnya mengikuti praktik umum: di bawah 0,95 sudah perlu perhatian,
// di bawah 0,90 perlu tindakan, di bawah 0,85 kritis.
func (a *Analysis) Health() string {
	worst := a.Snapshot.SPI
	if a.Snapshot.CPI < worst {
		worst = a.Snapshot.CPI
	}
	switch {
	case worst >= 0.95:
		return "good"
	case worst >= 0.90:
		return "watch"
	case worst >= 0.85:
		return "alert"
	default:
		return "severe"
	}
}

// deriveFindings menurunkan temuan dari angka. Tidak ada temuan yang ditulis
// tetap; semuanya lahir dari perbandingan metrik terhadap ambang, sehingga
// kalau datanya diperbaiki, temuannya ikut hilang dengan sendirinya.
func deriveFindings(a *Analysis) []Finding {
	var out []Finding
	s := a.Snapshot

	if s.EACTypical > model.TotalAuthorised {
		gap := s.EACTypical - model.TotalAuthorised
		out = append(out, Finding{
			Key: "eac-melewati-pagu", Severity: "kritis", Route: "/biaya/",
			Title: model.Text{
				ID: "Proyeksi biaya akhir melewati pagu yang disetujui",
				EN: "Forecast cost at completion exceeds the authorised budget",
			},
			Detail: model.Text{
				ID: "Dengan CPI " + fmtIdx(s.CPI) + ", proyeksi biaya akhir adalah " + fmtRp(s.EACTypical) + " terhadap pagu " + fmtRp(model.TotalAuthorised) + ". Seluruh cadangan kontinjensi dan cadangan manajemen habis terpakai, dan masih kurang " + fmtRp(gap) + ".",
				EN: "At a CPI of " + fmtIdx(s.CPI) + ", forecast cost at completion is " + fmtRp(s.EACTypical) + " against an authorised " + fmtRp(model.TotalAuthorised) + ". Both reserves are consumed and it is still " + fmtRp(gap) + " short.",
			},
			Metric: model.Text{ID: "EAC = BAC / CPI", EN: "EAC = BAC / CPI"},
			Action: model.Text{
				ID: "Ajukan perubahan anggaran ke sponsor sekarang, atau pangkas lingkup senilai minimal " + fmtRp(gap) + " lewat proses change control.",
				EN: "Raise a budget change with the sponsor now, or cut at least " + fmtRp(gap) + " of scope through change control.",
			},
		})
	}

	if a.Risk.ReserveGap > 0 {
		out = append(out, Finding{
			Key: "cadangan-kurang", Severity: "kritis", Route: "/risiko/",
			Title: model.Text{
				ID: "Cadangan kontinjensi jauh di bawah paparan risiko residual",
				EN: "Contingency reserve falls far short of residual risk exposure",
			},
			Detail: model.Text{
				ID: "Expected Monetary Value seluruh risiko setelah mitigasi adalah " + fmtRp(a.Risk.TotalResidualEMV) + ", sementara cadangan kontinjensi hanya " + fmtRp(model.ContingencyReserve) + " - menutup " + fmtPct(a.Risk.ReserveCoverage) + " paparan.",
				EN: "Expected Monetary Value of all risks after mitigation is " + fmtRp(a.Risk.TotalResidualEMV) + " while the contingency reserve is only " + fmtRp(model.ContingencyReserve) + ", covering " + fmtPct(a.Risk.ReserveCoverage) + " of the exposure.",
			},
			Metric: model.Text{ID: "EMV residual vs cadangan kontinjensi", EN: "residual EMV vs contingency reserve"},
			Action: model.Text{
				ID: "Naikkan cadangan menjadi sekitar " + fmtRp(a.Risk.TotalResidualEMV) + ", atau perkuat mitigasi pada risiko ber-EMV tertinggi sampai paparan turun ke tingkat yang tertutup.",
				EN: "Raise the reserve to about " + fmtRp(a.Risk.TotalResidualEMV) + ", or strengthen mitigation on the highest-EMV risks until exposure drops to a covered level.",
			},
		})
	}

	if a.Sim.OnTimeProb < 0.5 {
		out = append(out, Finding{
			Key: "jadwal-optimistis", Severity: "kritis", Route: "/pert/",
			Title: model.Text{
				ID: "Komitmen 17 minggu nyaris mustahil dipenuhi",
				EN: "The 17-week commitment is close to unattainable",
			},
			Detail: model.Text{
				ID: "Dari " + fmtInt(float64(a.Sim.Iterations)) + " iterasi Monte Carlo, hanya " + fmtPct(a.Sim.OnTimeProb) + " yang selesai dalam " + fmtInt(float64(a.Plan.Duration)) + " hari kerja. Durasi P80 adalah " + fmtInt(a.Sim.P80) + " hari kerja.",
				EN: "Across " + fmtInt(float64(a.Sim.Iterations)) + " Monte Carlo iterations, only " + fmtPct(a.Sim.OnTimeProb) + " finish within " + fmtInt(float64(a.Plan.Duration)) + " working days. The P80 duration is " + fmtInt(a.Sim.P80) + " working days.",
			},
			Metric: model.Text{ID: "P(durasi <= rencana) dari simulasi Monte Carlo", EN: "P(duration <= plan) from the Monte Carlo simulation"},
			Action: model.Text{
				ID: "Sampaikan komitmen P80 (" + fmtInt(a.Sim.P80) + " hari kerja) kepada sponsor, bukan estimasi titik tunggal. Jadwal titik tunggal adalah janji berpeluang " + fmtPct(a.Sim.OnTimeProb) + ".",
				EN: "Commit to the P80 figure (" + fmtInt(a.Sim.P80) + " working days) with the sponsor rather than a single-point estimate. A single-point schedule is a promise with " + fmtPct(a.Sim.OnTimeProb) + " odds.",
			},
		})
	}

	if len(a.Resources.Conflicts) > 0 {
		c := a.Resources.Conflicts[0]
		out = append(out, Finding{
			Key: "over-alokasi", Severity: "tinggi", Route: "/organisasi/",
			Title: model.Text{
				ID: "Jadwal menugaskan satu orang pada dua pekerjaan sekaligus",
				EN: "The schedule assigns one person two jobs at once",
			},
			Detail: model.Text{
				ID: "Ada " + fmtInt(float64(len(a.Resources.Conflicts))) + " hari-peran dengan beban melebihi kapasitas. Yang terberat: peran " + string(c.Role) + " dibebani " + fmtNum(c.Load) + " hari-orang terhadap kapasitas " + fmtNum(c.Capacity) + ".",
				EN: "There are " + fmtInt(float64(len(a.Resources.Conflicts))) + " role-days loaded beyond capacity. The worst: role " + string(c.Role) + " carries " + fmtNum(c.Load) + " person-days against a capacity of " + fmtNum(c.Capacity) + ".",
			},
			Metric: model.Text{ID: "beban harian vs kapasitas peran", EN: "daily load vs role capacity"},
			Action: model.Text{
				ID: "Lakukan resource levelling dengan menggeser aktivitas yang masih punya float, atau tambah orang pada peran yang kelebihan beban.",
				EN: "Level resources by shifting activities that still hold float, or add people to the overloaded roles.",
			},
		})
	}

	if !a.Control.InControl {
		out = append(out, Finding{
			Key: "proses-bergeser", Severity: "tinggi", Route: "/kualitas/",
			Title: model.Text{
				ID: "Waktu respons bergeser sistematis, bukan berfluktuasi acak",
				EN: "Response time is drifting systematically, not fluctuating randomly",
			},
			Detail: model.Text{
				ID: "Peta kendali menandai " + fmtInt(float64(len(a.Control.Violations))) + " pelanggaran aturan. Nilai masih di bawah batas spesifikasi 3 detik, tetapi arahnya konsisten memburuk - masalah akan muncul sebelum bebannya mencapai skala produksi.",
				EN: "The control chart flags " + fmtInt(float64(len(a.Control.Violations))) + " rule violations. Values remain under the 3-second specification limit, but the direction is consistently worsening - trouble will arrive before load reaches production scale.",
			},
			Metric: model.Text{ID: "aturan Nelson pada peta kendali X-bar", EN: "Nelson rules on the X-bar control chart"},
			Action: model.Text{
				ID: "Telusuri penyebab khususnya sekarang, saat datanya masih kecil. Indeks pada kueri agregasi dasbor adalah tersangka pertama.",
				EN: "Chase the special cause now, while the dataset is still small. Indexing on the dashboard aggregation query is the first suspect.",
			},
		})
	}

	if a.COQ.Ratio < 1 {
		out = append(out, Finding{
			Key: "biaya-mutu", Severity: "sedang", Route: "/kualitas/",
			Title: model.Text{
				ID: "Biaya kegagalan melebihi biaya pencegahan",
				EN: "Failure cost exceeds prevention cost",
			},
			Detail: model.Text{
				ID: "Biaya kesesuaian " + fmtRp(a.COQ.Conformance) + " berbanding biaya ketidaksesuaian " + fmtRp(a.COQ.Nonconformance) + ", rasio " + fmtNum(a.COQ.Ratio) + ". Proyek membayar akibat lebih banyak daripada mencegah sebab.",
				EN: "Conformance cost " + fmtRp(a.COQ.Conformance) + " against non-conformance cost " + fmtRp(a.COQ.Nonconformance) + ", a ratio of " + fmtNum(a.COQ.Ratio) + ". The project pays for consequences more than it pays to prevent causes.",
			},
			Metric: model.Text{ID: "biaya kesesuaian / biaya ketidaksesuaian", EN: "conformance cost / non-conformance cost"},
			Action: model.Text{
				ID: "Pindahkan sebagian anggaran pengujian ke muka: gerbang mutu otomatis di pipeline CI lebih murah daripada regression testing manual berulang.",
				EN: "Move part of the testing budget forward: an automated CI quality gate costs less than repeated manual regression testing.",
			},
		})
	}

	if s.SPIt < 1 {
		out = append(out, Finding{
			Key: "earned-schedule", Severity: "tinggi", Route: "/biaya/",
			Title: model.Text{
				ID: "Earned Schedule menunjukkan keterlambatan dalam satuan waktu",
				EN: "Earned Schedule shows the delay in time units",
			},
			Detail: model.Text{
				ID: "SPI berbasis rupiah " + fmtIdx(s.SPI) + " sudah memberi sinyal, tetapi Earned Schedule menerjemahkannya menjadi angka yang bisa dipakai: proyek tertinggal " + fmtNum(-s.SVt) + " hari kerja dari rencana.",
				EN: "The cost-based SPI of " + fmtIdx(s.SPI) + " already signals trouble, but Earned Schedule turns it into something usable: the project is " + fmtNum(-s.SVt) + " working days behind plan.",
			},
			Metric: model.Text{ID: "SV(t) = ES - AT", EN: "SV(t) = ES - AT"},
			Action: model.Text{
				ID: "Pakai SV(t) dalam laporan ke sponsor. Keterlambatan dalam hari lebih mudah dipahami dan tidak menyesatkan di akhir proyek seperti SPI.",
				EN: "Report SV(t) to the sponsor. A delay in days is easier to grasp and does not mislead near project end the way SPI does.",
			},
		})
	}

	out = append(out, upgradeFindings(a)...)
	return append(out, closureFindings(a)...)
}
