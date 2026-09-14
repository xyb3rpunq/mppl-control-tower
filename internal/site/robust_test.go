package site

import (
	"math"
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
)

// TestValueBandsByHand: selubung atas tiga garis yang bisa dihitung tangan,
// termasuk opsi yang tidak pernah terbaik.
func TestValueBandsByHand(t *testing.T) {
	// tanpa (0, 0); lembur (8 hari, 1.132.859); tambah (4, 2.235.840); keduanya (10, 2.806.461)
	days := []float64{0, 8, 4, 10}
	extra := []float64{0, 1132859, 2235840, 2806461}
	b := valueBands(days, extra, []bool{true, true, true, true})
	if len(b) != 3 || b[0].Option != 0 || b[1].Option != 1 || b[2].Option != 3 {
		t.Fatalf("pita %+v, mau tanpa -> lembur -> keduanya", b)
	}
	if math.Abs(b[1].From-1132859.0/8) > 1e-6 || math.Abs(b[2].From-(2806461.0-1132859)/2) > 1e-6 || !b[2].Open() || b[0].From != 0 {
		t.Errorf("batas pita %+v", b)
	}
	for k := 1; k < len(b); k++ {
		if b[k].From != b[k-1].To {
			t.Errorf("pita %d tidak bersambung: %v lalu %v", k, b[k-1].To, b[k].From)
		}
	}
	// Opsi yang tidak eligible diabaikan; tanpa opsi lain, tanpa percepatan menang di mana-mana.
	only := valueBands(days, extra, []bool{true, false, false, false})
	if len(only) != 1 || only[0].Option != 0 || !only[0].Open() {
		t.Errorf("hanya tanpa percepatan: %+v", only)
	}
	if valueBands(days, extra, []bool{false, false, false, false}) != nil {
		t.Error("tanpa opsi eligible, tidak ada pita")
	}
	// Opsi yang lebih cepat DAN lebih murah menang sejak nilai nol.
	free := valueBands([]float64{0, 3}, []float64{0, -100}, []bool{true, true})
	if len(free) != 1 || free[0].Option != 1 {
		t.Errorf("opsi lebih cepat dan lebih murah harus terbaik sejak nol: %+v", free)
	}
	// Seri dipecah ke anggaran tambahan yang lebih kecil.
	tie := valueBands([]float64{0, 5, 5}, []float64{0, 500, 400}, []bool{true, true, true})
	if len(tie) != 2 || tie[1].Option != 2 {
		t.Errorf("seri harus dimenangkan opsi yang lebih murah: %+v", tie)
	}
}

func TestPricingAndComparisons(t *testing.T) {
	pts := []simulate.FrontierPoint{
		{Duration: 121, Budget: 100, Feasible: true},
		{Duration: 113, Budget: 180, Feasible: true},
		{Duration: 117, Budget: 120, Feasible: true},
		{Duration: 111, Budget: 400, Feasible: true},
		{},
	}
	days, extra, price, cheapest := pricing(pts, []bool{false, false, true, false, false})
	if days[1] != 8 || extra[1] != 80 || price[1] != 10 || days[4] != 0 {
		t.Errorf("hari %v, tambahan %v, harga %v", days, extra, price)
	}
	// Opsi 2 paling murah per hari (20/4 = 5 < 80/8 = 10) tetapi berasumsi, jadi opsi 1 yang termurah.
	if price[2] != 5 || cheapest != 1 {
		t.Errorf("opsi termurah %d, mau 1", cheapest)
	}
	if _, _, _, c := pricing([]simulate.FrontierPoint{{}}, []bool{false}); c != -1 {
		t.Error("tanpa titik dasar yang layak, tidak ada opsi termurah")
	}
	if !sameInts([]int{1, 2}, []int{1, 2}) || sameInts([]int{1}, []int{1, 2}) || sameInts([]int{1, 3}, []int{1, 2}) {
		t.Error("sameInts salah")
	}
	if p := jcl70(simulate.IntegratedResult{}); p.Feasible {
		t.Error("hasil kosong tidak punya titik JCL 70%")
	}
}

func TestDecisionTextsFollowBands(t *testing.T) {
	d := &Decision{
		Cheapest: 1, Fastest: 2,
		Options: []AccelOption{
			{Key: "tanpa", Name: model.Text{ID: "Tanpa percepatan", EN: "No acceleration"}, FloorProven: true},
			{Key: "lembur", Name: model.Text{ID: "Lembur sah BE", EN: "Legal overtime BE"}, JCL70: simulate.FrontierPoint{Duration: 113, Budget: 2000}, FloorProven: true},
			{Key: "tambah", Name: model.Text{ID: "Tambah satu BE", EN: "Add one BE"}, FloorProven: false},
		},
		Bands:  []ValueBand{{Option: 0, From: 0, To: 100}, {Option: 1, From: 100, To: math.Inf(1)}},
		Robust: &Robustness{BandsSame: 0.9},
		Scenarios: []Scenario{
			{Key: "dasar", Same: true, Cheapest: 1},
			{Key: "x", Same: true}, {Key: "y", Same: false},
		},
	}
	id, en := d.BandRuleID(), d.BandRuleEN()
	for _, want := range []string{"di bawah Rp 100", "Rp 100 ke atas", "lembur sah BE (113 hari kerja", "tidak pernah terbaik: tambah satu BE", "90,0% ulangan bootstrap dan 1 dari 2 skenario"} {
		if !strings.Contains(id, want) {
			t.Errorf("aturan ID tidak memuat %q: %s", want, id)
		}
	}
	if !strings.Contains(en, "below IDR 100") || !strings.Contains(en, "and above") || !strings.Contains(en, "Never the best option: add one BE") {
		t.Errorf("aturan EN keliru: %s", en)
	}
	if d.AllFloorsProven() {
		t.Error("ada lantai yang belum terbukti")
	}
	if got := d.BandOption(d.Bands[1]); got.Key != "lembur" {
		t.Errorf("BandOption %s", got.Key)
	}
	d.Bands = []ValueBand{{Option: 0, From: 0, To: math.Inf(1)}, {Option: 1, From: 5, To: 9}}
	if !strings.Contains(d.BandRuleID(), "berapa pun nilainya") || !strings.Contains(d.BandRuleID(), "Rp 5 sampai Rp 9") {
		t.Errorf("rentang terbuka dan tertutup: %s", d.BandRuleID())
	}
	if (&Decision{}).BandRuleID() != "" {
		t.Error("tanpa pita, aturan kosong")
	}
	if b := (&Decision{}).ScenarioBase(); b.Cheapest != -1 || (&Decision{}).AssumptionScenarios() != nil || (Scenario{}).Iterations() != 0 {
		t.Error("tanpa skenario, pembanding kosong")
	}
}
