package model_test

import (
	"math"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// TestBudgetLayersReconcile menjaga invarian anggaran: estimasi bottom-up
// ditambah dua lapis cadangan harus persis sama dengan pagu Project Charter.
// Kalau tarif atau alokasi tim diubah, uji inilah yang memberi tahu bahwa
// rekonsiliasinya pecah - bukan pembaca yang menemukannya di halaman jadi.
func TestBudgetLayersReconcile(t *testing.T) {
	bac := model.BAC()
	baseline := model.CostBaseline()
	mr := model.ManagementReserve()

	if math.Abs(baseline-(bac+model.ContingencyReserve)) > 1e-6 {
		t.Errorf("cost baseline %.2f bukan BAC + kontinjensi", baseline)
	}
	if math.Abs(baseline+mr-model.TotalAuthorised) > 1e-6 {
		t.Errorf("baseline %.2f + cadangan manajemen %.2f tidak sama dengan pagu %.2f",
			baseline, mr, model.TotalAuthorised)
	}
	if mr < 0 {
		t.Errorf("cadangan manajemen negatif (%.2f): estimasi bottom-up sudah melampaui pagu "+
			"sebelum proyek dimulai", mr)
	}
	if bac <= 0 {
		t.Fatal("BAC harus positif")
	}
}

// TestActivityBudgetsSumToBAC memeriksa aturan 100% secara aritmetis.
func TestActivityBudgetsSumToBAC(t *testing.T) {
	var sum float64
	for _, a := range model.Activities {
		sum += a.Budget(model.RateCard)
	}
	if math.Abs(sum-model.BAC()) > 1e-6 {
		t.Errorf("jumlah anggaran aktivitas %.2f tidak sama dengan BAC %.2f", sum, model.BAC())
	}
}

// TestEveryActivityHasAParentPackage adalah sisi struktural aturan 100%:
// tidak boleh ada pekerjaan yang menggantung di luar WBS.
func TestEveryActivityHasAParentPackage(t *testing.T) {
	known := map[string]bool{}
	for _, ph := range model.WBSPhases {
		for _, pk := range ph.Packages {
			known[pk.Code] = true
		}
	}
	for _, a := range model.Activities {
		if !known[a.WBS] {
			t.Errorf("aktivitas %s menunjuk paket kerja tak dikenal %q", a.ID, a.WBS)
		}
		if model.PhaseOf(a.WBS).Code == "" {
			t.Errorf("aktivitas %s tidak punya fase induk", a.ID)
		}
	}
}

// TestEveryPackageHasActivities memeriksa sisi sebaliknya: tidak boleh ada
// paket kerja yang dideklarasikan tetapi tidak pernah dikerjakan.
func TestEveryPackageHasActivities(t *testing.T) {
	used := map[string]int{}
	for _, a := range model.Activities {
		used[a.WBS]++
	}
	for _, ph := range model.WBSPhases {
		for _, pk := range ph.Packages {
			if used[pk.Code] == 0 {
				t.Errorf("paket kerja %s (%s) tidak punya satu pun aktivitas", pk.Code, pk.Name.ID)
			}
		}
	}
}

func TestThreePointEstimatesAreOrdered(t *testing.T) {
	for _, a := range model.Activities {
		if a.Milestone {
			continue
		}
		if a.Optimistic <= 0 || a.Pessimistic <= 0 {
			t.Errorf("%s: estimasi tiga titik belum diisi (O=%d, P=%d)", a.ID, a.Optimistic, a.Pessimistic)
			continue
		}
		if !(a.Optimistic <= a.Duration && a.Duration <= a.Pessimistic) {
			t.Errorf("%s: urutan O <= M <= P dilanggar (%d, %d, %d)",
				a.ID, a.Optimistic, a.Duration, a.Pessimistic)
		}
	}
}

func TestMilestonesCarryNoWork(t *testing.T) {
	for _, a := range model.Activities {
		if !a.Milestone {
			continue
		}
		if a.Duration != 0 {
			t.Errorf("milestone %s berdurasi %d", a.ID, a.Duration)
		}
		if len(a.Team) != 0 || len(a.Extras) != 0 {
			t.Errorf("milestone %s membawa sumber daya atau biaya; milestone adalah penanda, bukan pekerjaan", a.ID)
		}
		if a.Budget(model.RateCard) != 0 {
			t.Errorf("milestone %s punya anggaran bukan nol", a.ID)
		}
	}
}

func TestActualsAreConsistent(t *testing.T) {
	for _, a := range model.Activities {
		if !a.Actual.Started {
			if a.Actual.Cost != 0 || a.Actual.Duration != 0 {
				t.Errorf("%s: belum dimulai tetapi punya biaya/durasi aktual", a.ID)
			}
			continue
		}
		if a.Actual.Start < 0 {
			t.Errorf("%s: tanggal mulai aktual negatif", a.ID)
		}
		if !a.Milestone && a.Actual.Duration <= 0 {
			t.Errorf("%s: sudah dimulai tetapi durasi aktualnya %d", a.ID, a.Actual.Duration)
		}
		if !a.Milestone && a.Actual.Cost <= 0 {
			t.Errorf("%s: sudah dimulai tetapi biaya aktualnya %.2f", a.ID, a.Actual.Cost)
		}
	}
}

// TestRACIHasExactlyOneAccountable adalah aturan paling penting pada matriks
// RACI. Dua A dalam satu baris berarti tidak ada yang benar-benar bertanggung
// jawab; nol A berarti tidak ada yang menyetujui.
func TestRACIHasExactlyOneAccountable(t *testing.T) {
	for _, row := range model.RACI {
		a, r := 0, 0
		for _, role := range model.RACIRoles {
			switch row.Assignment[role] {
			case model.A:
				a++
			case model.R:
				r++
			}
		}
		if a != 1 {
			t.Errorf("fase %s: ada %d Accountable, harus tepat 1", row.Phase, a)
		}
		if r < 1 {
			t.Errorf("fase %s: tidak ada Responsible sama sekali", row.Phase)
		}
	}
}

func TestRACICoversEveryPhase(t *testing.T) {
	seen := map[string]bool{}
	for _, row := range model.RACI {
		seen[row.Phase] = true
	}
	for _, ph := range model.WBSPhases {
		if !seen[ph.Code] {
			t.Errorf("fase %s tidak punya baris RACI", ph.Code)
		}
	}
}

func TestRatesAndCapacityCoverEveryAssignedRole(t *testing.T) {
	for _, a := range model.Activities {
		for _, slot := range a.Team {
			if _, ok := model.RateCard[slot.Role]; !ok {
				t.Errorf("peran %s pada %s tidak punya tarif", slot.Role, a.ID)
			}
			if _, ok := model.Capacity[slot.Role]; !ok {
				t.Errorf("peran %s pada %s tidak punya kapasitas", slot.Role, a.ID)
			}
			if slot.Alloc <= 0 || slot.Alloc > 1 {
				t.Errorf("%s: alokasi %s = %.2f, harus di (0, 1]", a.ID, slot.Role, slot.Alloc)
			}
		}
	}
}

// indonesianFunctionWords adalah kata fungsi yang nyaris mustahil muncul di
// kalimat Inggris yang benar. Kata-kata inilah alat deteksi yang tepat.
//
// Mendeteksi kebocoran bahasa dengan membandingkan apakah teks ID dan EN
// identik TIDAK bekerja: istilah teknis seperti "Bug fixing & regression
// testing" atau "PMBOK - Determine Budget" memang sama persis di kedua bahasa,
// dan uji semacam itu hanya menghasilkan derau yang akhirnya dimatikan orang.
// Yang benar-benar menandakan teks lupa diterjemahkan adalah munculnya kata
// fungsi Indonesia - "yang", "dengan", "untuk" - di dalam kolom bahasa Inggris.
var indonesianFunctionWords = []string{
	"yang", "dan", "untuk", "dengan", "dari", "pada", "tidak", "adalah",
	"karena", "sehingga", "tetapi", "atau", "akan", "sudah", "belum",
	"harus", "bisa", "dapat", "lebih", "sangat", "juga", "agar", "oleh",
	"kalau", "hanya", "masih", "setiap", "seluruh", "itu", "ini",
}

// containsIndonesian melaporkan kata fungsi Indonesia pertama yang ditemukan
// sebagai kata utuh di dalam s.
func containsIndonesian(s string) string {
	lower := strings.ToLower(s)
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !(r >= 'a' && r <= 'z')
	})
	present := make(map[string]bool, len(fields))
	for _, f := range fields {
		present[f] = true
	}
	for _, w := range indonesianFunctionWords {
		if present[w] {
			return w
		}
	}
	return ""
}

// TestBilingualTextIsComplete menangkap kebocoran bahasa: teks yang lupa
// diterjemahkan akan tampil dalam bahasa Indonesia di halaman berbahasa
// Inggris, dan itu hanya terlihat kalau seseorang benar-benar membacanya.
func TestBilingualTextIsComplete(t *testing.T) {
	check := func(where string, texts ...model.Text) {
		t.Helper()
		for _, tx := range texts {
			if strings.TrimSpace(tx.ID) == "" {
				t.Errorf("%s: teks bahasa Indonesia kosong", where)
			}
			if strings.TrimSpace(tx.EN) == "" {
				t.Errorf("%s: terjemahan bahasa Inggris kosong untuk %q", where, tx.ID)
				continue
			}
			if w := containsIndonesian(tx.EN); w != "" {
				t.Errorf("%s: kolom bahasa Inggris memuat kata Indonesia %q - teks belum diterjemahkan: %q",
					where, w, tx.EN)
			}
		}
	}

	for _, a := range model.Activities {
		check("aktivitas "+a.ID, a.Name)
	}
	for _, ph := range model.WBSPhases {
		check("fase "+ph.Code, ph.Name, ph.Milestone)
		for _, pk := range ph.Packages {
			check("paket "+pk.Code, pk.Name)
		}
	}
	for _, r := range model.Risks {
		check("risiko "+r.ID, r.Title, r.Category, r.Cause, r.Effect, r.Mitigation, r.Trigger)
	}
	for _, m := range model.Team {
		check("peran "+string(m.Role), m.Title)
		check("tanggung jawab "+string(m.Role), m.Responsibili...)
	}
	for _, s := range model.Stakeholders {
		check("pemangku kepentingan "+s.ID, s.Name, s.Need, s.Strategy)
	}
	for _, f := range model.Formulas {
		check("rumus "+f.Key, f.Name, f.Meaning, f.Reading, f.Pitfall, f.Source, f.Notation)
		for _, sym := range f.Symbols {
			check("simbol "+f.Key+"/"+sym.Sym, sym.Desc)
		}
	}
	for _, k := range model.KnowledgeAreas {
		check("area pengetahuan", k.Name, k.Content, k.Evidence)
	}
	for _, lc := range model.Lifecycle {
		check("siklus hidup "+lc.Key, lc.Name, lc.Goal, lc.Outputs, lc.Watch)
		check("kegiatan "+lc.Key, lc.Activities...)
	}
	for _, l := range model.CoretaxLessons {
		check("pelajaran coretax "+l.Key, l.Title, l.Finding, l.Practice, l.Mirror)
	}
	for _, f := range model.CoretaxFacts {
		check("fakta coretax "+f.Key, f.Label, f.Detail)
	}
	for _, d := range model.Defects {
		check("cacat", d.Module)
	}
	for _, q := range model.QualityMetrics {
		check("metrik mutu", q.Name, q.Source)
	}
	for _, fb := range model.Fishbones {
		check("fishbone "+fb.Key, fb.Effect, fb.RootCause)
		for _, br := range fb.Branches {
			check("cabang "+fb.Key, br.Category)
			check("sebab "+fb.Key, br.Causes...)
		}
	}
}

func TestRiskProbabilitiesAreValid(t *testing.T) {
	for _, r := range model.Risks {
		if r.Probability <= 0 || r.Probability > 1 {
			t.Errorf("%s: peluang %.2f di luar (0,1]", r.ID, r.Probability)
		}
		if r.ResidualProb < 0 || r.ResidualProb > r.Probability {
			t.Errorf("%s: peluang residual %.2f harus di [0, %.2f]", r.ID, r.ResidualProb, r.Probability)
		}
		if r.ResidualImpact > r.Impact {
			t.Errorf("%s: dampak residual melebihi dampak inheren", r.ID)
		}
		if r.ResidualEMV() > r.EMV() {
			t.Errorf("%s: mitigasi menaikkan EMV, bukan menurunkannya", r.ID)
		}
		if r.Response == "" {
			t.Errorf("%s: tidak punya strategi respons", r.ID)
		}
	}
}

func TestFormulaKeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	groups := map[string]bool{}
	for _, g := range model.FormulaGroups {
		groups[g.Key] = true
	}
	for _, f := range model.Formulas {
		if seen[f.Key] {
			t.Errorf("kunci rumus ganda: %s", f.Key)
		}
		seen[f.Key] = true
		if !groups[f.Group] {
			t.Errorf("rumus %s masuk kelompok tak dikenal %q", f.Key, f.Group)
		}
		if strings.TrimSpace(f.Notation.ID) == "" || strings.TrimSpace(f.Notation.EN) == "" {
			t.Errorf("rumus %s tidak punya notasi lengkap dua bahasa", f.Key)
		}
		if len(f.Symbols) == 0 {
			t.Errorf("rumus %s tidak menjelaskan satu simbol pun", f.Key)
		}
	}
}

func TestCoretaxFactsAllCarrySources(t *testing.T) {
	for _, f := range model.CoretaxFacts {
		if len(f.SourceIdx) == 0 {
			t.Errorf("fakta %q tidak punya sumber - setiap angka pada bagian fakta wajib bersumber", f.Key)
		}
		for _, i := range f.SourceIdx {
			if i < 0 || i >= len(model.CoretaxSources) {
				t.Errorf("fakta %q menunjuk sumber di luar daftar (indeks %d)", f.Key, i)
			}
		}
	}
	for i, s := range model.CoretaxSources {
		if !strings.HasPrefix(s.URL, "https://") {
			t.Errorf("sumber %d bukan URL https: %q", i, s.URL)
		}
		if s.Publisher == "" || s.Date == "" {
			t.Errorf("sumber %d tidak lengkap", i)
		}
	}
	for _, m := range model.CoretaxTimeline {
		if len(m.SourceIdx) == 0 {
			t.Errorf("tonggak %s tidak punya sumber", m.Date)
		}
	}
}

func TestPackageNameFallsBackToCode(t *testing.T) {
	if got := model.PackageName("3.2"); !strings.Contains(got.ID, "Backend") {
		t.Errorf("paket 3.2 = %q", got.ID)
	}
	if got := model.PackageName("9.9"); got.ID != "9.9" || got.EN != "9.9" {
		t.Errorf("kode tak dikenal harus dikembalikan apa adanya, dapat %+v", got)
	}
}
