package quality_test

import (
	"math"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/quality"
)

func TestParetoOrdersDescendingAndAccumulates(t *testing.T) {
	labels := []model.Text{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	values := []float64{10, 50, 30, 10}
	p := quality.BuildPareto(labels, values)

	if p.Total != 100 {
		t.Errorf("total = %v, mau 100", p.Total)
	}
	for i := 1; i < len(p.Items); i++ {
		if p.Items[i].Value > p.Items[i-1].Value {
			t.Fatalf("Pareto tidak terurut menurun di posisi %d", i)
		}
	}
	if p.Items[0].Value != 50 || math.Abs(p.Items[0].Share-0.5) > 1e-9 {
		t.Errorf("batang teratas = %v (%v), mau 50 (0,5)", p.Items[0].Value, p.Items[0].Share)
	}
	if last := p.Items[len(p.Items)-1].Cumulative; math.Abs(last-1) > 1e-9 {
		t.Errorf("kumulatif terakhir = %v, mau 1", last)
	}
	// 50 + 30 = 80% tepat, jadi vital few berhenti di dua kategori.
	if p.VitalFewCount != 2 {
		t.Errorf("vital few = %d kategori, mau 2", p.VitalFewCount)
	}
}

// TestWeightedDefectsOutrankRawCounts membuktikan alasan pembobotan ada:
// modul dengan sedikit cacat kritis harus mengalahkan modul dengan banyak
// cacat kosmetik.
func TestWeightedDefectsOutrankRawCounts(t *testing.T) {
	fewCritical := model.DefectRecord{Critical: 3, Major: 0, Minor: 0} // bobot 15
	manyMinor := model.DefectRecord{Critical: 0, Major: 0, Minor: 12}  // bobot 12

	if manyMinor.Total() <= fewCritical.Total() {
		t.Fatal("prasyarat uji salah: modul kosmetik harus punya lebih banyak cacat mentah")
	}
	if fewCritical.Weighted() <= manyMinor.Weighted() {
		t.Errorf("pembobotan gagal: 3 cacat kritis (%d) tidak mengalahkan 12 cacat minor (%d)",
			fewCritical.Weighted(), manyMinor.Weighted())
	}
}

func TestControlChartLimitsAndCapability(t *testing.T) {
	// Proses stabil di sekitar 2,0 dengan rentang subgrup tetap 0,5.
	values := []float64{2.0, 2.0, 2.0, 2.0, 2.0, 2.0}
	ranges := []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	cc := quality.BuildControlChart(values, ranges, 5, 3.0, true)

	if math.Abs(cc.CenterLine-2.0) > 1e-9 {
		t.Errorf("garis tengah = %v, mau 2,0", cc.CenterLine)
	}
	// UCL = CL + A2 x R-bar = 2 + 0,577 x 0,5 = 2,2885
	if math.Abs(cc.UCL-2.2885) > 1e-4 {
		t.Errorf("UCL = %v, mau 2,2885", cc.UCL)
	}
	if math.Abs(cc.LCL-1.7115) > 1e-4 {
		t.Errorf("LCL = %v, mau 1,7115", cc.LCL)
	}
	// sigma = R-bar / d2 = 0,5 / 2,326
	if math.Abs(cc.Sigma-0.5/2.326) > 1e-9 {
		t.Errorf("sigma = %v", cc.Sigma)
	}
	if !cc.InControl {
		t.Error("proses datar seharusnya terkendali")
	}
	if !cc.HasCpk || cc.Cpk <= 0 {
		t.Error("Cpk harus dihitung ketika batas spesifikasi diberikan")
	}
}

// TestNelsonRulesUseTheDrawnLimits menjaga koreksi penting: aturan harus diuji
// terhadap batas yang SAMA dengan yang digambar. Kalau aturan memakai sigma
// individual sementara grafiknya memakai batas subgrup, titik yang jelas-jelas
// di luar garis merah akan lolos tanpa terdeteksi.
func TestNelsonRulesUseTheDrawnLimits(t *testing.T) {
	values := []float64{2.0, 2.0, 2.0, 2.0, 2.0, 3.5} // titik terakhir jauh di atas UCL
	ranges := []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	cc := quality.BuildControlChart(values, ranges, 5, 9.0, true)

	if cc.InControl {
		t.Fatal("titik di luar UCL seharusnya membuat proses tidak terkendali")
	}
	found := false
	for _, v := range cc.Violations {
		if v.Rule == 1 && v.Index == 5 {
			found = true
		}
	}
	if !found {
		t.Errorf("aturan 1 tidak menangkap titik %v yang berada di atas UCL %v", values[5], cc.UCL)
	}
}

func TestNelsonRuleTwoDetectsRunOfSeven(t *testing.T) {
	values := make([]float64, 14)
	ranges := make([]float64, 14)
	for i := range values {
		ranges[i] = 0.5
		if i < 7 {
			values[i] = 1.9
		} else {
			values[i] = 2.1
		}
	}
	cc := quality.BuildControlChart(values, ranges, 5, 0, false)
	found := false
	for _, v := range cc.Violations {
		if v.Rule == 2 {
			found = true
		}
	}
	if !found {
		t.Error("aturan 2 tidak menangkap tujuh titik berurutan di satu sisi garis tengah")
	}
}

func TestNelsonRuleThreeDetectsDrift(t *testing.T) {
	values := []float64{1.0, 1.2, 1.4, 1.6, 1.8, 2.0, 2.2, 2.4}
	ranges := []float64{0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3}
	cc := quality.BuildControlChart(values, ranges, 5, 0, false)
	found := false
	for _, v := range cc.Violations {
		if v.Rule == 3 {
			found = true
		}
	}
	if !found {
		t.Error("aturan 3 tidak menangkap enam titik berurutan yang terus menaik")
	}
}

// TestRealResponseDataIsOutOfControl menjaga temuan yang ditampilkan di
// halaman Mutu: waktu respons proyek ini sedang bergeser sistematis meskipun
// belum satu pun nilai melewati batas spesifikasi tiga detik.
func TestRealResponseDataIsOutOfControl(t *testing.T) {
	var means, ranges []float64
	for _, s := range model.ResponseSamples {
		means = append(means, s.Mean)
		ranges = append(ranges, s.Range)
	}
	cc := quality.BuildControlChart(means, ranges, 5, 3.0, true)

	if cc.InControl {
		t.Error("data respons nyata seharusnya TIDAK terkendali - temuan di halaman Mutu bergantung padanya")
	}
	for i, v := range means {
		if v > cc.Spec {
			t.Errorf("sampel minggu %d (%v detik) melewati batas spesifikasi; "+
				"argumen \"memenuhi spesifikasi tetapi tidak terkendali\" jadi gugur", i+1, v)
		}
	}
	if cc.Cpk < 1.0 {
		t.Errorf("Cpk = %v; argumen di halaman Mutu mengandaikan proses masih tampak mampu", cc.Cpk)
	}
}

func TestCOQSummaryArithmetic(t *testing.T) {
	items := []model.COQItem{
		{Category: model.COQPrevention, Amount: 100},
		{Category: model.COQAppraisal, Amount: 200},
		{Category: model.COQInternalFailure, Amount: 400},
		{Category: model.COQExternalFailure, Amount: 500},
	}
	s := quality.SummariseCOQ(items, 10000)

	if s.Conformance != 300 {
		t.Errorf("biaya kesesuaian = %v, mau 300", s.Conformance)
	}
	if s.Nonconformance != 900 {
		t.Errorf("biaya ketidaksesuaian = %v, mau 900", s.Nonconformance)
	}
	if s.Total != 1200 {
		t.Errorf("total = %v, mau 1200", s.Total)
	}
	if math.Abs(s.Ratio-300.0/900.0) > 1e-9 {
		t.Errorf("rasio = %v, mau 0,3333", s.Ratio)
	}
	if math.Abs(s.ShareOfBudget-0.12) > 1e-9 {
		t.Errorf("porsi terhadap BAC = %v, mau 0,12", s.ShareOfBudget)
	}
}

func TestMetricEvaluationHandlesBothDirections(t *testing.T) {
	metrics := []model.QualityMetric{
		{Name: model.Text{ID: "makin tinggi makin baik"}, Target: 80, Actual: 90, HigherIsBetter: true},
		{Name: model.Text{ID: "makin tinggi makin buruk"}, Target: 3, Actual: 2, HigherIsBetter: false},
		{Name: model.Text{ID: "gagal, arah naik"}, Target: 80, Actual: 41, HigherIsBetter: true},
		{Name: model.Text{ID: "gagal, arah turun"}, Target: 0, Actual: 11, HigherIsBetter: false},
	}
	got := quality.EvaluateMetrics(metrics)
	want := []bool{true, true, false, false}
	for i, w := range want {
		if got[i].Met != w {
			t.Errorf("metrik %d: Met = %v, mau %v", i, got[i].Met, w)
		}
		if got[i].Gap < 0 {
			t.Errorf("metrik %d: jarak negatif %v", i, got[i].Gap)
		}
	}
	if got[2].Gap != 39 {
		t.Errorf("jarak cakupan uji = %v, mau 39", got[2].Gap)
	}
}

func TestDefectAggregatesOnRealData(t *testing.T) {
	if quality.DefectDensity(model.Defects) <= 0 {
		t.Error("kerapatan cacat harus positif")
	}
	if quality.TotalRework(model.Defects) <= 0 {
		t.Error("total jam rework harus positif")
	}
	p := quality.DefectPareto(model.Defects)
	if len(p.Items) != len(model.Defects) {
		t.Errorf("Pareto memuat %d kategori, mau %d", len(p.Items), len(model.Defects))
	}
	if p.VitalFewCount == 0 || p.VitalFewCount > len(p.Items) {
		t.Errorf("vital few tidak masuk akal: %d dari %d", p.VitalFewCount, len(p.Items))
	}
}

func TestEmptyInputsDoNotPanic(t *testing.T) {
	cc := quality.BuildControlChart(nil, nil, 5, 3, true)
	if len(cc.Violations) != 0 {
		t.Error("data kosong tidak boleh menghasilkan pelanggaran")
	}
	p := quality.BuildPareto(nil, nil)
	if len(p.Items) != 0 || p.Total != 0 {
		t.Error("Pareto kosong harus tetap kosong")
	}
	s := quality.SummariseCOQ(nil, 0)
	if s.Total != 0 {
		t.Error("ringkasan COQ kosong harus nol")
	}
}
