package risk_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/risk"
)

func TestProbabilityLevelBoundaries(t *testing.T) {
	cases := map[float64]risk.Level{
		0.05: 1, 0.09: 1,
		0.10: 2, 0.24: 2,
		0.25: 3, 0.44: 3,
		0.45: 4, 0.64: 4,
		0.65: 5, 1.00: 5,
	}
	for p, want := range cases {
		if got := risk.ProbabilityLevel(p); got != want {
			t.Errorf("ProbabilityLevel(%v) = %d, mau %d", p, got, want)
		}
	}
}

func TestImpactLevelScalesWithBAC(t *testing.T) {
	bac := 10_000_000.0
	cases := map[float64]risk.Level{
		100_000:   1, // 1%
		400_000:   2, // 4%
		800_000:   3, // 8%
		1_500_000: 4, // 15%
		3_000_000: 5, // 30%
	}
	for impact, want := range cases {
		if got := risk.ImpactLevel(impact, bac); got != want {
			t.Errorf("ImpactLevel(%v, %v) = %d, mau %d", impact, bac, got, want)
		}
	}
	// Dampak yang sama pada proyek sepuluh kali lebih besar harus turun tingkat.
	small := risk.ImpactLevel(1_500_000, 10_000_000)
	large := risk.ImpactLevel(1_500_000, 100_000_000)
	if large >= small {
		t.Errorf("dampak yang sama pada proyek lebih besar seharusnya bertingkat lebih rendah (%d vs %d)", large, small)
	}
}

func TestSeverityBands(t *testing.T) {
	cases := map[int]risk.Severity{
		1: risk.SeverityLow, 4: risk.SeverityLow,
		5: risk.SeverityModerate, 9: risk.SeverityModerate,
		10: risk.SeverityHigh, 15: risk.SeverityHigh,
		16: risk.SeverityExtreme, 25: risk.SeverityExtreme,
	}
	for score, want := range cases {
		if got := risk.SeverityOf(score); got != want {
			t.Errorf("SeverityOf(%d) = %q, mau %q", score, got, want)
		}
	}
}

func TestEMVArithmetic(t *testing.T) {
	r := model.Risk{
		Probability: 0.4, Impact: 2_000_000,
		ResidualProb: 0.2, ResidualImpact: 1_000_000,
	}
	if got := r.EMV(); math.Abs(got-800_000) > 1e-9 {
		t.Errorf("EMV = %v, mau 800.000", got)
	}
	if got := r.ResidualEMV(); math.Abs(got-200_000) > 1e-9 {
		t.Errorf("EMV residual = %v, mau 200.000", got)
	}
}

func TestRegisterTotalsMatchSumOfParts(t *testing.T) {
	bac := model.BAC()
	reg := risk.Analyse(model.Risks, bac, model.ContingencyReserve)

	var emv, residual float64
	for _, r := range model.Risks {
		emv += r.EMV()
		residual += r.ResidualEMV()
	}
	if math.Abs(reg.TotalEMV-emv) > 1e-6 {
		t.Errorf("total EMV = %v, mau %v", reg.TotalEMV, emv)
	}
	if math.Abs(reg.TotalResidualEMV-residual) > 1e-6 {
		t.Errorf("total EMV residual = %v, mau %v", reg.TotalResidualEMV, residual)
	}
	if math.Abs(reg.ReserveGap-(residual-model.ContingencyReserve)) > 1e-6 {
		t.Errorf("kekurangan cadangan tidak konsisten")
	}
	if reg.TotalResidualEMV >= reg.TotalEMV {
		t.Error("mitigasi seharusnya menurunkan total EMV")
	}
}

func TestCategoryTotalsSumToWhole(t *testing.T) {
	reg := risk.Analyse(model.Risks, model.BAC(), model.ContingencyReserve)
	var emv, residual float64
	count := 0
	for _, c := range reg.ByCategory {
		emv += c.EMV
		residual += c.ResidualEMV
		count += c.Count
	}
	if count != len(model.Risks) {
		t.Errorf("jumlah risiko per kategori = %d, mau %d", count, len(model.Risks))
	}
	if math.Abs(emv-reg.TotalEMV) > 1e-6 || math.Abs(residual-reg.TotalResidualEMV) > 1e-6 {
		t.Error("agregat kategori tidak sama dengan total register")
	}
	// Kategori harus terurut dari paparan residual terbesar.
	for i := 1; i < len(reg.ByCategory); i++ {
		if reg.ByCategory[i].ResidualEMV > reg.ByCategory[i-1].ResidualEMV {
			t.Errorf("kategori tidak terurut menurun di posisi %d", i)
		}
	}
}

func TestEveryRiskAppearsInBothMatrices(t *testing.T) {
	reg := risk.Analyse(model.Risks, model.BAC(), model.ContingencyReserve)
	countIn := func(m [5][5][]string) int {
		n := 0
		for _, row := range m {
			for _, cell := range row {
				n += len(cell)
			}
		}
		return n
	}
	if got := countIn(reg.Matrix); got != len(model.Risks) {
		t.Errorf("peta inheren memuat %d risiko, mau %d", got, len(model.Risks))
	}
	if got := countIn(reg.ResidualMatrix); got != len(model.Risks) {
		t.Errorf("peta residual memuat %d risiko, mau %d", got, len(model.Risks))
	}
}

func TestMitigationNeverWorsensScore(t *testing.T) {
	reg := risk.Analyse(model.Risks, model.BAC(), model.ContingencyReserve)
	for _, s := range reg.Risks {
		if s.ResidualScore > s.Score {
			t.Errorf("%s: skor residual %d lebih buruk daripada inheren %d", s.ID, s.ResidualScore, s.Score)
		}
		if s.Reduction < 0 || s.Reduction > 1 {
			t.Errorf("%s: penurunan EMV %v di luar [0,1]", s.ID, s.Reduction)
		}
	}
}

func TestSortedByScoreIsDescending(t *testing.T) {
	reg := risk.Analyse(model.Risks, model.BAC(), model.ContingencyReserve)
	sorted := reg.SortedByScore()
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Score > sorted[i-1].Score {
			t.Errorf("urutan skor tidak menurun di posisi %d", i)
		}
	}
	top := reg.TopByEMV
	for i := 1; i < len(top); i++ {
		if top[i].ResidualEMV > top[i-1].ResidualEMV {
			t.Errorf("urutan EMV residual tidak menurun di posisi %d", i)
		}
	}
}

// TestReserveShortfallIsReal menjaga temuan utama halaman Risiko. Kalau
// seseorang menaikkan cadangan sampai cukup, temuannya harus ikut hilang -
// dan uji ini yang mengingatkan bahwa teksnya perlu diperbarui.
func TestReserveShortfallIsReal(t *testing.T) {
	reg := risk.Analyse(model.Risks, model.BAC(), model.ContingencyReserve)
	if reg.ReserveGap <= 0 {
		t.Errorf("cadangan sudah mencukupi (kelebihan %.2f); temuan kekurangan cadangan "+
			"di halaman Risiko tidak lagi berlaku dan teksnya harus diperbarui", -reg.ReserveGap)
	}
	if reg.ReserveCoverage <= 0 || reg.ReserveCoverage >= 1 {
		t.Errorf("cakupan cadangan %v seharusnya di antara 0 dan 1", reg.ReserveCoverage)
	}
}

func TestScheduleExposureIsPositive(t *testing.T) {
	reg := risk.Analyse(model.Risks, model.BAC(), model.ContingencyReserve)
	if reg.ScheduleExposure <= 0 {
		t.Error("paparan jadwal harapan harus positif ketika ada risiko berdampak jadwal")
	}
}

func TestEmptyRegisterIsSafe(t *testing.T) {
	reg := risk.Analyse(nil, 1000, 100)
	if reg.TotalEMV != 0 || reg.TotalResidualEMV != 0 {
		t.Error("register kosong harus bertotal nol")
	}
	if reg.ReserveCoverage != 1 {
		t.Errorf("tanpa risiko, cakupan cadangan = %v, mau 1", reg.ReserveCoverage)
	}
}
