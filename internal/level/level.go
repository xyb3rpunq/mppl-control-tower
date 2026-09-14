// Package level menyusun jadwal yang benar-benar bisa dijalankan dengan
// kapasitas orang yang tersedia (resource-constrained scheduling).
//
// CPM menjawab "sepagi apa pekerjaan ini boleh mulai kalau sumber daya tak
// terbatas". Paket resource sudah menunjukkan jawaban itu melanggar kapasitas
// pada 26 hari-peran. Paket ini menjawab pertanyaan berikutnya: kalau
// kapasitas dihormati, kapan proyeknya selesai?
//
// Metodenya Serial Schedule Generation Scheme (SGS). Run menjalankan satu
// kali SGS dengan satu aturan prioritas atau satu daftar aktivitas. Optimize
// (optimize.go) mencari jadwal terbaik dari banyak aturan, sampel acak
// berbias, dan justifikasi maju-mundur; LowerBound (bound.go) menghitung batas
// bawah yang TIDAK mungkin dikalahkan jadwal mana pun. Selisih keduanya adalah
// celah optimalitas: bila nol, jadwal terbaik terbukti optimal untuk model
// kerja ini, walaupun RCPSP secara umum NP-hard.
//
// Model kerjanya berbasis isi pekerjaan, bukan kalender: sebuah aktivitas
// berdurasi d hari dengan alokasi a butuh a x d hari-orang. Pada hari ketika
// orangnya hanya tersedia sebagian - karena berbagi dengan aktivitas lain,
// karena paruh waktu, atau karena ujian - pekerjaan tetap berjalan, hanya
// lebih lambat. Hari-orang yang dibutuhkan tidak bertambah; kalendernya yang
// memanjang.
package level

import (
	"fmt"
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/resource"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

const eps = 1e-9

// DefaultMinStartRate adalah 0,2: sebuah aktivitas baru dianggap dimulai bila
// hari pertamanya setara paling sedikit satu hari kerja per minggu (1/5).
// Ambang ini bukan angka bebas - ia satuan kerja terkecil yang lazim dipakai
// saat merencanakan pekerjaan paruh waktu mahasiswa, dan sama dengan
// kapasitas DevOps paruh waktu selama ujian (0,5 x 0,4).
const DefaultMinStartRate = 0.2

// Options mengatur satu kali levelling.
type Options struct {
	Calendar *workcal.Calendar
	Capacity map[model.Role]float64
	// UseWindows menerapkan model.AvailabilityWindows (mis. periode ujian).
	UseWindows bool
	// MinStartRate adalah laju kerja minimum pada hari pertama. Tanpa ambang
	// ini, aktivitas bisa "dimulai" dengan satu persen kapasitas lalu tertahan
	// berhari-hari, yang di atas kertas tampak mulai tepat waktu padahal belum.
	// Bawaannya DefaultMinStartRate.
	MinStartRate float64
	// DurationOf memungkinkan simulasi menyuntikkan durasi acak.
	DurationOf func(model.Activity) int
	// Horizon adalah batas hari kerja yang dicari; lewat dari itu dianggap gagal.
	Horizon int
	// Lite melewatkan pencatatan rinci (aktivitas per hari, penyebab
	// perlambatan) yang tidak dibutuhkan simulasi. Hasil durasi dan
	// pemakaian kapasitasnya identik dengan mode lengkap.
	Lite bool
	// CapGrid adalah kapasitas per peran per hari yang sudah dihitung
	// sebelumnya lewat CapacityGrid. Simulasi yang memanggil levelling ribuan
	// kali mengisinya sekali saja; grid ini hanya dibaca, tidak pernah diubah.
	CapGrid map[model.Role][]float64
	// ExamFactor mengganti faktor kapasitas jendela ketersediaan bila > 0.
	ExamFactor float64
	// ReleaseOf memberi hari paling awal aktivitas boleh dimulai.
	ReleaseOf func(model.Activity) int
	// Rule adalah aturan prioritas SGS; kosong berarti RuleLST.
	Rule Rule
	// Order, bila diisi, menggantikan Rule: SGS selalu memilih aktivitas layak
	// yang paling awal muncul di daftar ini. Aktivitas yang tidak tercantum
	// diletakkan di belakang menurut aturan LST.
	Order []string

	// plan adalah CPM yang sudah dihitung untuk durasi dan tanggal rilis yang
	// sama. Search mengisinya sekali untuk ratusan langkah, karena CPM tidak
	// berubah selama yang berubah hanya urutan aktivitas.
	plan *schedule.Result
}

// Rule adalah aturan prioritas pemilihan aktivitas pada SGS.
type Rule string

// Aturan prioritas baku RCPSP (Kolisch & Hartmann, 1999). Semuanya statis:
// dihitung sekali dari CPM, bukan dari keadaan jadwal parsial.
const (
	RuleLST  Rule = "LST"  // latest start terkecil
	RuleLFT  Rule = "LFT"  // latest finish terkecil
	RuleMSLK Rule = "MSLK" // total float terkecil
	RuleGRPW Rule = "GRPW" // bobot posisi terbesar: durasi + durasi penerus langsung
	RuleMTS  Rule = "MTS"  // jumlah penerus (langsung dan tidak langsung) terbanyak
	RuleSPT  Rule = "SPT"  // durasi terpendek
)

// Rules adalah seluruh aturan prioritas yang dicoba Optimize.
var Rules = []Rule{RuleLST, RuleLFT, RuleMSLK, RuleGRPW, RuleMTS, RuleSPT}

// CapacityGrid menghitung kapasitas setiap peran pada setiap hari kerja.
func CapacityGrid(cal *workcal.Calendar, capacity map[model.Role]float64, useWindows bool, horizon int) map[model.Role][]float64 {
	return CapacityGridWith(cal, capacity, useWindows, horizon, 0)
}

// CapacityGridWith sama dengan CapacityGrid dengan faktor jendela pengganti.
func CapacityGridWith(cal *workcal.Calendar, capacity map[model.Role]float64, useWindows bool, horizon int, examFactor float64) map[model.Role][]float64 {
	if horizon > cal.Len() {
		horizon = cal.Len()
	}
	grid := make(map[model.Role][]float64, len(capacity))
	iso := make([]string, horizon)
	for k := range iso {
		iso[k] = cal.ISOAt(k)
	}
	for r, base := range capacity {
		row := make([]float64, horizon)
		for k := 0; k < horizon; k++ {
			if useWindows {
				row[k] = model.CapacityOnDateWith(r, iso[k], capacity, examFactor)
			} else {
				row[k] = base
			}
		}
		grid[r] = row
	}
	return grid
}

// Task adalah hasil levelling untuk satu aktivitas.
type Task struct {
	ID       string
	Duration int // isi pekerjaan nominal, dalam hari kerja penuh
	ES       int // early start menurut CPM, sebagai pembanding
	Ready    int // hari paling awal yang diizinkan pendahulu pada jadwal levelling
	Start    int
	Finish   int // eksklusif

	// Pemecahan keterlambatan: berapa hari terbawa dari pendahulu, berapa hari
	// menunggu sumber daya, dan berapa hari kalender memanjang saat dikerjakan.
	CarriedDays int // Ready - ES
	WaitDays    int // Start - Ready
	StretchDays int // (Finish - Start) - Duration

	// Penyebab perlambatan saat dikerjakan.
	HalfTime bool // alokasi melebihi kapasitas normal peran itu
	Window   bool // melewati jendela ketersediaan (mis. ujian)
	Shared   bool // berbagi peran dengan aktivitas lain pada hari yang sama
}

// Delay mengembalikan total pergeseran selesai terhadap CPM.
func (t Task) Delay() int { return (t.Finish) - (t.ES + t.Duration) }

// Result adalah jadwal hasil levelling.
type Result struct {
	Tasks       map[string]Task
	Order       []string // urutan penjadwalan
	Duration    int
	CPMDuration int
	Horizon     int
	Usage       map[model.Role][]float64
	Cap         map[model.Role][]float64
	OnDay       map[model.Role][][]string // aktivitas yang memakai peran itu per hari
	Roles       []model.Role
}

// Run menjalankan levelling atas sebuah jaringan.
func Run(acts []model.Activity, opts Options) (Result, error) {
	if opts.Calendar == nil {
		return Result{}, fmt.Errorf("level: kalender wajib diisi")
	}
	if opts.Capacity == nil {
		return Result{}, fmt.Errorf("level: kapasitas wajib diisi")
	}
	if opts.MinStartRate <= 0 {
		opts.MinStartRate = DefaultMinStartRate
	}
	if opts.Horizon <= 0 {
		opts.Horizon = 300
	}
	if opts.Horizon > opts.Calendar.Len() {
		opts.Horizon = opts.Calendar.Len()
	}
	durationOf := opts.DurationOf
	if durationOf == nil {
		durationOf = func(a model.Activity) int { return a.Duration }
	}

	var plan schedule.Result
	if opts.plan != nil {
		plan = *opts.plan
	} else {
		var err error
		if plan, err = schedule.Compute(acts, schedule.Options{DurationOf: durationOf, ReleaseOf: opts.ReleaseOf}); err != nil {
			return Result{}, err
		}
	}
	key := priorityKeys(acts, plan, opts, durationOf)
	byID := make(map[string]model.Activity, len(acts))
	for _, a := range acts {
		byID[a.ID] = a
	}

	H := opts.Horizon
	roleSet := map[model.Role]bool{}
	for _, a := range acts {
		for _, s := range a.Team {
			roleSet[s.Role] = true
		}
	}
	var roles []model.Role
	for r := range roleSet {
		roles = append(roles, r)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })

	res := Result{
		Tasks:       make(map[string]Task, len(acts)),
		CPMDuration: plan.Duration,
		Horizon:     H,
		Usage:       make(map[model.Role][]float64, len(roles)),
		Cap:         make(map[model.Role][]float64, len(roles)),
		OnDay:       make(map[model.Role][][]string, len(roles)),
		Roles:       roles,
	}
	grid := opts.CapGrid
	if grid == nil {
		grid = CapacityGridWith(opts.Calendar, opts.Capacity, opts.UseWindows, H, opts.ExamFactor)
	}
	for _, r := range roles {
		res.Usage[r] = make([]float64, H)
		row := grid[r]
		if len(row) < H {
			return Result{}, fmt.Errorf("level: grid kapasitas peran %s lebih pendek dari horizon", r)
		}
		res.Cap[r] = row[:H]
		if !opts.Lite {
			res.OnDay[r] = make([][]string, H)
		}
	}

	scheduled := make(map[string]bool, len(acts))
	for len(scheduled) < len(acts) {
		// Pilih aktivitas layak dengan kunci prioritas terkecil.
		var pick string
		for _, a := range acts {
			if scheduled[a.ID] {
				continue
			}
			ready := true
			for _, p := range a.Pred {
				if !scheduled[p.ID] {
					ready = false
					break
				}
			}
			if !ready {
				continue
			}
			if pick == "" || key.less(a.ID, pick) {
				pick = a.ID
			}
		}
		if pick == "" {
			return Result{}, fmt.Errorf("level: tidak ada aktivitas yang layak dijadwalkan; jaringan tidak sah")
		}

		a := byID[pick]
		d := durationOf(a)
		cpmTask := plan.Task(pick)

		ready := 0
		if opts.ReleaseOf != nil {
			if r := opts.ReleaseOf(a); r > ready {
				ready = r
			}
		}
		for _, p := range a.Pred {
			pt := res.Tasks[p.ID]
			typ := p.Type
			if typ == "" {
				typ = "FS"
			}
			var c int
			switch typ {
			case "SS":
				c = pt.Start + p.Lag
			case "FF":
				c = pt.Finish + p.Lag - d
			case "SF":
				c = pt.Start + p.Lag - d
			default:
				c = pt.Finish + p.Lag
			}
			if c > ready {
				ready = c
			}
		}

		task := Task{ID: pick, Duration: d, ES: cpmTask.ES, Ready: ready}

		switch {
		case d == 0 || len(a.Team) == 0:
			task.Start = ready
			task.Finish = ready + d
		default:
			start := ready
			for start < H && rateAt(res, a, start) < opts.MinStartRate-eps {
				start++
			}
			if start >= H {
				return Result{}, fmt.Errorf("level: %s tidak menemukan hari mulai sebelum horizon %d", pick, H)
			}
			progress := 0.0
			k := start
			for progress < float64(d)-eps {
				if k >= H {
					return Result{}, fmt.Errorf("level: %s tidak selesai sebelum horizon %d", pick, H)
				}
				rate := rateAt(res, a, k)
				step := math.Min(rate, float64(d)-progress)
				if step > eps {
					for _, s := range a.Team {
						if !opts.Lite {
							base := opts.Capacity[s.Role]
							if s.Alloc > base+eps {
								task.HalfTime = true
							}
							if res.Cap[s.Role][k] < base-eps {
								task.Window = true
							}
							if res.Usage[s.Role][k] > eps {
								task.Shared = true
							}
							res.OnDay[s.Role][k] = append(res.OnDay[s.Role][k], pick)
						}
						res.Usage[s.Role][k] += s.Alloc * step
					}
					progress += step
				}
				k++
			}
			task.Start = start
			task.Finish = k
		}

		task.CarriedDays = task.Ready - task.ES
		task.WaitDays = task.Start - task.Ready
		task.StretchDays = (task.Finish - task.Start) - d
		res.Tasks[pick] = task
		res.Order = append(res.Order, pick)
		scheduled[pick] = true
		if task.Finish > res.Duration {
			res.Duration = task.Finish
		}
	}
	return res, nil
}

// keys adalah kunci prioritas setiap aktivitas untuk satu kali SGS.
type keys struct {
	primary   map[string]float64
	secondary map[string]float64
}

// less membandingkan dua aktivitas: kunci utama, lalu kunci kedua (early
// start CPM), lalu kode aktivitas agar urutannya deterministik.
func (k keys) less(a, b string) bool {
	if k.primary[a] != k.primary[b] {
		return k.primary[a] < k.primary[b]
	}
	if k.secondary[a] != k.secondary[b] {
		return k.secondary[a] < k.secondary[b]
	}
	return a < b
}

// priorityKeys menurunkan kunci prioritas dari aturan atau daftar aktivitas.
func priorityKeys(acts []model.Activity, plan schedule.Result, opts Options, durationOf func(model.Activity) int) keys {
	k := keys{primary: make(map[string]float64, len(acts)), secondary: make(map[string]float64, len(acts))}
	for _, a := range acts {
		k.secondary[a.ID] = float64(plan.Task(a.ID).ES)
	}
	if len(opts.Order) > 0 {
		pos := make(map[string]int, len(opts.Order))
		for i, id := range opts.Order {
			if _, seen := pos[id]; !seen {
				pos[id] = i
			}
		}
		// Aktivitas di luar daftar menyusul di belakang, diurutkan LST.
		for _, a := range acts {
			if i, ok := pos[a.ID]; ok {
				k.primary[a.ID] = float64(i)
			} else {
				k.primary[a.ID] = float64(len(opts.Order)) + float64(plan.Task(a.ID).LS)/1e6
			}
		}
		return k
	}
	rule := opts.Rule
	if rule == "" {
		rule = RuleLST
	}
	var succCount map[string]int
	if rule == RuleMTS {
		succCount = transitiveSuccessors(plan)
	}
	for _, a := range acts {
		t := plan.Task(a.ID)
		var v float64
		switch rule {
		case RuleLFT:
			v = float64(t.LateFinishX)
		case RuleMSLK:
			v = float64(t.TotalFloat)
		case RuleGRPW:
			w := durationOf(a)
			for _, s := range t.Successors {
				w += plan.Task(s).Duration
			}
			v = -float64(w)
		case RuleMTS:
			v = -float64(succCount[a.ID])
		case RuleSPT:
			v = float64(durationOf(a))
		default:
			v = float64(t.LS)
		}
		k.primary[a.ID] = v
	}
	return k
}

// transitiveSuccessors menghitung jumlah penerus langsung dan tidak langsung.
func transitiveSuccessors(plan schedule.Result) map[string]int {
	reach := make(map[string]map[string]bool, len(plan.Order))
	for i := len(plan.Order) - 1; i >= 0; i-- {
		id := plan.Order[i]
		set := map[string]bool{}
		for _, s := range plan.Task(id).Successors {
			set[s] = true
			for x := range reach[s] {
				set[x] = true
			}
		}
		reach[id] = set
	}
	out := make(map[string]int, len(reach))
	for id, set := range reach {
		out[id] = len(set)
	}
	return out
}

// rateAt menghitung laju kerja aktivitas pada hari k: porsi hari penuh yang
// bisa dikerjakan mengingat sisa kapasitas setiap peran yang dibutuhkannya.
// Peran yang paling sempit menentukan laju seluruh aktivitas - sama seperti
// dalam kenyataan, pekerjaan integrasi tidak bisa jalan lebih cepat dari
// orang tersibuk yang terlibat.
func rateAt(res Result, a model.Activity, k int) float64 {
	rate := 1.0
	for _, s := range a.Team {
		if s.Alloc <= 0 {
			continue
		}
		free := res.Cap[s.Role][k] - res.Usage[s.Role][k]
		if free < 0 {
			free = 0
		}
		if r := free / s.Alloc; r < rate {
			rate = r
		}
	}
	return rate
}

// Profile mengubah hasil levelling menjadi profil pembebanan agar histogram
// yang sama bisa menggambar kondisi sebelum dan sesudah levelling.
func (r Result) Profile() resource.Profile {
	h := r.Duration
	if h > r.Horizon {
		h = r.Horizon
	}
	p := resource.Profile{Horizon: h, Headcount: make([]float64, h)}
	for _, role := range r.Roles {
		rl := resource.RoleLoad{Role: role, FirstDay: -1, LastDay: -1}
		var capSum float64
		for k := 0; k < h; k++ {
			load := r.Usage[role][k]
			c := r.Cap[role][k]
			day := resource.DayLoad{
				Day: k, Load: load, Capacity: c,
				Over:       load > c+1e-6,
				Activities: onDay(r, role, k),
			}
			rl.Days = append(rl.Days, day)
			rl.TotalDays += load
			capSum += c
			if load > eps {
				p.Headcount[k]++
				if rl.FirstDay < 0 {
					rl.FirstDay = k
				}
				rl.LastDay = k
			}
			if load > rl.PeakLoad {
				rl.PeakLoad = load
				rl.PeakDay = k
			}
			if day.Over {
				rl.OverDays++
			}
			if c > rl.Capacity {
				rl.Capacity = c
			}
		}
		if capSum > 0 {
			rl.Utilisation = rl.TotalDays / capSum
		}
		p.TotalPersonDays += rl.TotalDays
		p.Roles = append(p.Roles, rl)
	}
	return p
}

func onDay(r Result, role model.Role, k int) []string {
	days := r.OnDay[role]
	if k >= len(days) {
		return nil
	}
	return append([]string(nil), days[k]...)
}

// Moved mengembalikan aktivitas yang selesai lebih lambat daripada CPM,
// diurutkan dari yang paling jauh bergeser.
func (r Result) Moved() []Task {
	var out []Task
	for _, t := range r.Tasks {
		if t.Delay() > 0 {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Delay() != out[j].Delay() {
			return out[i].Delay() > out[j].Delay()
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Breakdown memecah tambahan durasi proyek menjadi dua penyebab dengan
// menjalankan ulang levelling secara bertahap:
//
//  1. kapasitas normal saja, tanpa jendela ketersediaan
//  2. ditambah jendela ketersediaan (ujian)
//
// Selisih antar-tahap adalah kontribusi masing-masing penyebab. Pemecahan
// bertahap bergantung pada urutan, dan urutan di sini sengaja dimulai dari
// yang struktural (kapasitas) ke yang musiman (ujian). Setiap tahap memakai
// jadwal terbaik dari Optimize, bukan SGS polos, supaya selisihnya tidak
// tercemar kelemahan satu aturan prioritas.
type Breakdown struct {
	CPM          int
	CapacityOnly int
	WithWindows  int
	// Stage adalah hasil Optimize setiap tahap: [0] kapasitas, [1] + jendela.
	Stage [2]Optimized
}

// Explain menjalankan Optimize untuk kedua tahap Breakdown.
func Explain(acts []model.Activity, o OptimizeOptions) (Breakdown, error) {
	plan, err := schedule.Compute(acts, schedule.Options{DurationOf: o.DurationOf, ReleaseOf: o.ReleaseOf})
	if err != nil {
		return Breakdown{}, err
	}
	b := Breakdown{CPM: plan.Duration}
	for i, win := range []bool{false, true} {
		c := o
		c.UseWindows = win
		c.CapGrid = nil
		if b.Stage[i], err = Optimize(acts, c); err != nil {
			return Breakdown{}, err
		}
	}
	b.CapacityOnly = b.Stage[0].Best.Duration
	b.WithWindows = b.Stage[1].Best.Duration
	return b, nil
}
