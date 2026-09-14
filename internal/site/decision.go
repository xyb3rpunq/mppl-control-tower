package site

import (
	"math"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/level"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// Keputusan sponsor pada tanggal data.
//
// Temuan situs ini lahir dari halaman yang berbeda dan menyebut komitmen yang
// berbeda: P80 PERT, JCL 70% perencanaan, JCL 70% berjalan. Semuanya benar untuk
// pertanyaannya masing-masing, tetapi sponsor hanya boleh menerima satu. Paket
// keputusan menetapkan komitmen yang berlaku - prakiraan berjalan, satu-satunya
// yang memakai realisasi - dan menghitung ulang setiap opsi percepatan dari
// tanggal data, hari keputusan itu benar-benar diambil. Rencana lembur dari
// hari pertama proyek tidak bisa dibeli lagi bila sebagian harinya sudah lewat.

// AccelOption adalah satu opsi percepatan yang dinilai dari tanggal data.
type AccelOption struct {
	Key   string
	Name  model.Text
	Accel *simulate.Acceleration // nil: tanpa percepatan
	// Assumption diisi bila opsi memuat asumsi tanpa data.
	Assumption model.Text

	// Lantai deterministik: durasi paling mungkin, jadwal levelling terbaik
	// dan batas bawahnya pada kapasitas opsi ini.
	Floor       int
	FloorBound  int
	FloorProven bool

	Sim   simulate.IntegratedResult
	JCL70 simulate.FrontierPoint

	// Dibanding tanpa percepatan, pada titik JCL 70%:
	DaysEarlier float64
	ExtraBudget float64
	PricePerDay float64 // ExtraBudget / DaysEarlier; nol bila tidak lebih cepat
}

// Decision adalah paket keputusan sponsor.
type Decision struct {
	From     int
	Overtime compress.OvertimeCurve // lembur sah dari tanggal data
	Options  []AccelOption
	// Cheapest adalah opsi tanpa asumsi tambahan yang memajukan JCL 70% dengan
	// harga per hari terendah; Fastest opsi dengan JCL 70% paling awal. -1 bila
	// tidak ada.
	Cheapest, Fastest int
	// BudgetRequest adalah anggaran JCL 70% berjalan dikurangi pagu piagam.
	BudgetRequest float64
	// PlanMissed adalah hari-peran lembur rencana perencanaan (dari hari
	// pertama) yang jatuh sebelum tanggal data, dengan tanggal pertamanya.
	PlanMissed      int
	PlanMissedFirst string

	// Bands adalah pita nilai satu hari lebih cepat atas opsi tanpa asumsi.
	Bands []ValueBand
	// Robust adalah sebaran angka keputusan menurut bootstrap berpasangan.
	Robust *Robustness
	// ScenarioOptions adalah indeks opsi yang dijalankan ulang pada setiap
	// skenario asumsi; Scenarios hasilnya.
	ScenarioOptions []int
	Scenarios       []Scenario
}

// BandOption mengembalikan opsi sebuah pita.
func (d *Decision) BandOption(b ValueBand) AccelOption { return d.Options[b.Option] }

// NeverBest mengembalikan opsi tanpa asumsi yang tidak pernah terbaik pada
// nilai satu hari berapa pun.
func (d *Decision) NeverBest() []AccelOption {
	on := map[int]bool{}
	for _, b := range d.Bands {
		on[b.Option] = true
	}
	var out []AccelOption
	for i, o := range d.Options {
		if o.Assumption.ID == "" && !on[i] {
			out = append(out, o)
		}
	}
	return out
}

// ScenariosSame menghitung skenario asumsi (tanpa pembanding) yang
// rekomendasinya sama dengan pembanding.
func (d *Decision) ScenariosSame() int {
	n := 0
	for _, s := range d.AssumptionScenarios() {
		if s.Same {
			n++
		}
	}
	return n
}

// AssumptionScenarios mengembalikan skenario selain pembanding asumsi dasar.
func (d *Decision) AssumptionScenarios() []Scenario {
	if len(d.Scenarios) < 2 {
		return nil
	}
	return d.Scenarios[1:]
}

// ScenarioBase mengembalikan skenario pembanding asumsi dasar.
func (d *Decision) ScenarioBase() Scenario {
	if len(d.Scenarios) == 0 {
		return Scenario{Cheapest: -1}
	}
	return d.Scenarios[0]
}

// Option mengembalikan opsi berdasarkan kunci.
func (d *Decision) Option(key string) (AccelOption, bool) {
	for _, o := range d.Options {
		if o.Key == key {
			return o, true
		}
	}
	return AccelOption{}, false
}

// prepareDecision menyiapkan opsi percepatan dari tanggal data. Opsi lembur
// memakai peran yang benar-benar lembur pada rencana termurah durasi minimum -
// diturunkan dari angka, bukan dipilih - dan opsi orang baru memakai peran
// kritis jadwal levelling.
func prepareDecision(a *Analysis, fl *simulate.InFlight) (*Decision, error) {
	d := &Decision{From: fl.Now, Cheapest: -1, Fastest: -1}
	det := fl.Deterministic(model.Activities, a.Calendar, model.Capacity)
	var err error
	d.Overtime, err = compress.LevelledOvertime(fl.Residual, level.OptimizeOptions{Options: det}, model.RateCard, fl.Now)
	if err != nil {
		return nil, err
	}
	h := compress.SustainedOvertimeHours()
	role := a.CriticalRole
	d.Options = append(d.Options, AccelOption{Key: "tanpa", Name: model.Text{ID: "Tanpa percepatan", EN: "No acceleration"}})
	var roles []model.Role
	if n := len(d.Overtime.Points); n > 0 {
		roles = d.Overtime.Points[n-1].Roles()
	}
	if len(roles) > 0 {
		names := roleNames(roles)
		d.Options = append(d.Options, AccelOption{
			Key:   "lembur",
			Name:  model.Text{ID: "Lembur sah " + names, EN: "Legal overtime " + names},
			Accel: &simulate.Acceleration{From: fl.Now, OvertimeRoles: roles, OvertimeHours: h},
		})
	}
	hire := map[model.Role]float64{role: 1}
	d.Options = append(d.Options, AccelOption{
		Key:   "tambah",
		Name:  model.Text{ID: "Tambah satu " + string(role) + " penuh waktu", EN: "Add one full-time " + string(role)},
		Accel: &simulate.Acceleration{From: fl.Now, Hire: hire},
	})
	d.Options = append(d.Options, AccelOption{
		Key:        "tambah-adaptasi",
		Name:       model.Text{ID: "Tambah satu " + string(role) + ", dengan masa adaptasi", EN: "Add one " + string(role) + ", with ramp-up"},
		Accel:      &simulate.Acceleration{From: fl.Now, Hire: hire, RampDays: RampDays, RampFactor: RampFactor},
		Assumption: model.Text{ID: "Asumsi tanpa data: 10 hari kerja pertama pada setengah laju.", EN: "Assumed without data: the first 10 working days at half speed."},
	})
	defer func() {
		for i, o := range d.Options {
			if o.Assumption.ID == "" {
				d.ScenarioOptions = append(d.ScenarioOptions, i)
			}
		}
		d.Scenarios = scenarios()
	}()
	if len(roles) > 0 {
		d.Options = append(d.Options, AccelOption{
			Key:   "tambah-lembur",
			Name:  model.Text{ID: "Tambah satu " + string(role) + " + lembur sah", EN: "Add one " + string(role) + " + legal overtime"},
			Accel: &simulate.Acceleration{From: fl.Now, Hire: hire, OvertimeRoles: roles, OvertimeHours: h},
		})
	}
	return d, nil
}

// Masa adaptasi orang baru pada opsi kepekaan. Tidak ada data untuk
// mengukurnya; dua minggu kerja pada setengah laju adalah asumsi terbuka.
const (
	RampDays   = 10
	RampFactor = 0.5
)

// decisionJobs mengembalikan pekerjaan paralel untuk setiap opsi: lantai
// deterministik dan simulasi dari tanggal data. Opsi tanpa percepatan memakai
// prakiraan berjalan yang sudah dijalankan.
func decisionJobs(a *Analysis, fl *simulate.InFlight, fc simulate.IntegratedConfig) []func() error {
	d := a.Decision
	det := fl.Deterministic(model.Activities, a.Calendar, model.Capacity)
	var jobs []func() error
	for i := range d.Options {
		opt := &d.Options[i]
		jobs = append(jobs, func() error {
			q := det
			if opt.Accel != nil {
				g, err := opt.Accel.Grids(a.Calendar, model.Capacity, 300, fl.ExamFactor)
				if err != nil {
					return err
				}
				q.CapGrid, q.RateCap = g.Max, g.Rate
			}
			best, err := level.Optimize(fl.Residual, level.OptimizeOptions{Options: q})
			if err != nil {
				return err
			}
			opt.Floor, opt.FloorBound, opt.FloorProven = best.Best.Duration, best.Bound.Value, best.Proven
			if opt.Accel == nil {
				return nil
			}
			c := fc
			c.Accel = opt.Accel
			opt.Sim, err = simulate.RunIntegrated(model.Activities, c)
			return err
		})
	}
	for si := range d.Scenarios {
		sc := &d.Scenarios[si]
		sc.Sims = make([]simulate.IntegratedResult, len(d.ScenarioOptions))
		for k, oi := range d.ScenarioOptions {
			k, accel := k, d.Options[oi].Accel
			jobs = append(jobs, func() (err error) {
				c := fc
				c.Iterations, c.ExactLevel = ScenarioIterations, false
				sc.Config(&c)
				c.Accel = accel
				sc.Sims[k], err = simulate.RunIntegrated(model.Activities, c)
				return err
			})
		}
	}
	return jobs
}

// finishDecision menurunkan komitmen, harga per hari, dan rekomendasi.
func finishDecision(a *Analysis) error {
	d := a.Decision
	if d == nil {
		return nil
	}
	for i := range d.Options {
		o := &d.Options[i]
		if o.Accel == nil {
			o.Sim, o.JCL70 = a.Forecast, a.ForecastJCL70
			continue
		}
		o.JCL70 = jcl70(o.Sim)
	}
	base := d.Options[0].JCL70
	d.BudgetRequest = base.Budget - model.TotalAuthorised
	for i := range d.Options {
		o := &d.Options[i]
		if i == 0 || !o.JCL70.Feasible || !base.Feasible {
			continue
		}
		o.DaysEarlier = base.Duration - o.JCL70.Duration
		o.ExtraBudget = o.JCL70.Budget - base.Budget
		if o.DaysEarlier > 0 {
			o.PricePerDay = o.ExtraBudget / o.DaysEarlier
			if o.Assumption.ID == "" && (d.Cheapest < 0 || o.PricePerDay < d.Options[d.Cheapest].PricePerDay) {
				d.Cheapest = i
			}
		}
		if o.DaysEarlier > 0 && (d.Fastest < 0 || o.JCL70.Duration < d.Options[d.Fastest].JCL70.Duration ||
			(o.JCL70.Duration == d.Options[d.Fastest].JCL70.Duration && o.JCL70.Budget < d.Options[d.Fastest].JCL70.Budget)) {
			d.Fastest = i
		}
	}

	// Pita nilai, bootstrap berpasangan, dan skenario asumsi.
	days := make([]float64, len(d.Options))
	extra := make([]float64, len(d.Options))
	eligible := make([]bool, len(d.Options))
	for i, o := range d.Options {
		days[i], extra[i], eligible[i] = o.DaysEarlier, o.ExtraBudget, o.Assumption.ID == "" && (i == 0 || o.JCL70.Feasible)
	}
	d.Bands = valueBands(days, extra, eligible)
	var err error
	if d.Robust, err = robustness(d); err != nil {
		return err
	}
	for si := range d.Scenarios {
		if si == 0 {
			finishScenario(d, &d.Scenarios[0], d.Cheapest, d.Bands)
			continue
		}
		finishScenario(d, &d.Scenarios[si], d.Scenarios[0].Cheapest, d.Scenarios[0].Bands)
	}

	// Rencana lembur perencanaan: berapa hari-peran lemburnya sudah lewat.
	ot := a.Overtime
	if p, ok := ot.PointAt(ot.MinDuration); ok {
		base := level.CapacityGridWith(a.Calendar, model.Capacity, true, len(p.Schedule.Usage[model.RoleBE]), 0)
		for day := 0; day < d.From && day < p.Duration; day++ {
			for _, r := range ot.Eligible {
				u, ex := p.Schedule.Usage[r], p.Schedule.Excess[r]
				over := 0.0
				if day < len(u) {
					over = u[day] - base[r][day]
				}
				if day < len(ex) && ex[day] > over {
					over = ex[day]
				}
				if over > 1e-9 {
					if d.PlanMissed == 0 {
						d.PlanMissedFirst = a.Calendar.ISOAt(day)
					}
					d.PlanMissed++
				}
			}
		}
	}
	return nil
}

// jcl70 mengambil titik frontier JCL 70% pertama yang layak.
func jcl70(r simulate.IntegratedResult) simulate.FrontierPoint {
	if len(r.Durations) == 0 {
		return simulate.FrontierPoint{}
	}
	var ds []float64
	for d := math.Floor(r.DurP50); d <= r.Durations[len(r.Durations)-1]; d++ {
		ds = append(ds, d)
	}
	for _, p := range r.Frontier(0.7, ds) {
		if p.Feasible {
			return p
		}
	}
	return simulate.FrontierPoint{}
}

// roleNames menulis daftar peran, mis. "BE, FE, TL".
func roleNames(roles []model.Role) string {
	parts := make([]string, len(roles))
	for i, r := range roles {
		parts[i] = string(r)
	}
	return strings.Join(parts, ", ")
}

// Ramp mengembalikan masa adaptasi opsi kepekaan, dalam hari kerja.
func (d *Decision) Ramp() int { return RampDays }

// AllFloorsProven melaporkan apakah lantai setiap opsi terbukti optimal.
func (d *Decision) AllFloorsProven() bool {
	for _, o := range d.Options {
		if !o.FloorProven {
			return false
		}
	}
	return true
}

// RampChangesNothing melaporkan apakah masa adaptasi tidak mengubah titik
// JCL 70% opsi orang baru - teks halaman hanya menyebutnya bila benar.
func (d *Decision) RampChangesNothing() bool {
	h, ok1 := d.Option("tambah")
	r, ok2 := d.Option("tambah-adaptasi")
	return ok1 && ok2 && h.JCL70 == r.JCL70
}

// CheapestOption dan FastestOption mengembalikan opsi rekomendasi, bila ada.
func (d *Decision) CheapestOption() (AccelOption, bool) {
	if d.Cheapest < 0 {
		return AccelOption{}, false
	}
	return d.Options[d.Cheapest], true
}

func (d *Decision) FastestOption() (AccelOption, bool) {
	if d.Fastest < 0 {
		return AccelOption{}, false
	}
	return d.Options[d.Fastest], true
}

// commitmentKeys adalah temuan yang tindakannya diselesaikan halaman Keputusan
// Sponsor: komitmen, anggaran, dan percepatan.
var commitmentKeys = map[string]bool{
	"jadwal-optimistis": true, "jcl-rendah": true, "prakiraan-berjalan": true, "eac-melewati-pagu": true,
	"cadangan-kurang": true, "jadwal-tak-terjalankan": true, "percepatan-tanggal-data": true,
	"crashing-hampir-impas": true,
}

// lowerFirst mengecilkan huruf pertama nama opsi di tengah kalimat, kecuali
// singkatan peran yang ditulis kapital (BE, FE, ...).
func lowerFirst(s string) string {
	if len(s) < 2 || strings.ToUpper(s[:2]) == s[:2] {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// OtherActions mengembalikan temuan di luar komitmen dan anggaran, dalam urutan
// temuan.
func (a *Analysis) OtherActions() []Finding {
	var out []Finding
	for _, f := range a.Findings {
		if !commitmentKeys[f.Key] {
			out = append(out, f)
		}
	}
	return out
}

// ProofSummary mengembalikan porsi iterasi terbukti optimal yang paling rendah
// di antara opsi, dan celah terbesar iterasi yang belum terbukti.
func (d *Decision) ProofSummary() (minShare float64, maxGap int) {
	minShare = 1
	for _, o := range d.Options {
		if o.Sim.Proof.Iterations == 0 {
			continue
		}
		if s := o.Sim.Proof.ProvenShare(); s < minShare {
			minShare = s
		}
		if o.Sim.Proof.MaxGap > maxGap {
			maxGap = o.Sim.Proof.MaxGap
		}
	}
	return minShare, maxGap
}

// MinProvenShare dan MaxProofGap membuka ProofSummary untuk templat.
func (d *Decision) MinProvenShare() float64 { s, _ := d.ProofSummary(); return s }

func (d *Decision) MaxProofGap() int { _, g := d.ProofSummary(); return g }
