// Package quality mengimplementasikan Seven Basic Tools of Quality yang
// dibahas pada Pertemuan 9 MPPL, dalam bentuk yang bisa dihitung.
//
// Yang membedakan alat mutu dari grafik biasa adalah aturan keputusannya.
// Control chart bukan sekadar garis naik-turun: ia punya batas kendali yang
// dihitung dari data itu sendiri, dan aturan yang memberi tahu kapan sebuah
// proses sudah "di luar kendali" - bukan sekadar kelihatan jelek.
package quality

import (
	"math"
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// ---------------------------------------------------------------- Pareto

// ParetoItem adalah satu batang pada diagram Pareto.
type ParetoItem struct {
	Label      model.Text
	Value      float64
	Share      float64 // porsi terhadap total
	Cumulative float64 // porsi kumulatif
	InVitalFew bool    // masuk kelompok penyebab utama (kumulatif <= 80%)
}

// Pareto adalah hasil analisis Pareto.
type Pareto struct {
	Items []ParetoItem
	Total float64
	// VitalFewCount adalah jumlah kategori yang menyumbang 80% pertama.
	VitalFewCount int
	// VitalFewShare adalah porsi kategori tersebut terhadap seluruh kategori,
	// dipakai untuk menguji apakah aturan 80/20 benar-benar berlaku di sini.
	VitalFewShare float64
}

// BuildPareto membangun diagram Pareto dari pasangan label dan nilai.
func BuildPareto(labels []model.Text, values []float64) Pareto {
	type pair struct {
		label model.Text
		value float64
	}
	pairs := make([]pair, 0, len(values))
	var total float64
	for i := range values {
		pairs = append(pairs, pair{labels[i], values[i]})
		total += values[i]
	}
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].value > pairs[j].value })

	p := Pareto{Total: total}
	var running float64
	reachedEighty := false
	for _, pr := range pairs {
		running += pr.value
		item := ParetoItem{Label: pr.label, Value: pr.value}
		if total > 0 {
			item.Share = pr.value / total
			item.Cumulative = running / total
		}
		if !reachedEighty {
			item.InVitalFew = true
			p.VitalFewCount++
			if item.Cumulative >= 0.8 {
				reachedEighty = true
			}
		}
		p.Items = append(p.Items, item)
	}
	if len(p.Items) > 0 {
		p.VitalFewShare = float64(p.VitalFewCount) / float64(len(p.Items))
	}
	return p
}

// DefectPareto membangun diagram Pareto atas cacat berbobot keparahan.
func DefectPareto(defects []model.DefectRecord) Pareto {
	labels := make([]model.Text, len(defects))
	values := make([]float64, len(defects))
	for i, d := range defects {
		labels[i] = d.Module
		values[i] = float64(d.Weighted())
	}
	return BuildPareto(labels, values)
}

// ---------------------------------------------------------- Control chart

// Violation adalah satu pelanggaran aturan kendali.
type Violation struct {
	Rule  int
	Index int
	Text  model.Text
}

// ControlChart adalah peta kendali X-bar beserta hasil uji aturan.
type ControlChart struct {
	Values     []float64
	CenterLine float64
	UCL        float64
	LCL        float64
	Sigma      float64
	// Spec adalah batas spesifikasi dari pelanggan (di sini: standar mutu pada
	// Project Charter). Batas kendali berasal dari proses, batas spesifikasi
	// berasal dari kebutuhan - dua hal yang sering tertukar.
	Spec       float64
	HasSpec    bool
	Violations []Violation
	// Cpk adalah indeks kemampuan proses terhadap batas spesifikasi atas.
	Cpk       float64
	HasCpk    bool
	InControl bool
}

// BuildControlChart membangun peta kendali X-bar dari deret nilai dan rentang
// subgrup. Batas kendali dihitung dengan metode rentang rata-rata:
//
//	UCL = x-bar-bar + A2 * R-bar
//	LCL = x-bar-bar - A2 * R-bar
//
// A2 bergantung pada ukuran subgrup; untuk n = 5, A2 = 0,577 (tabel kendali
// mutu standar). Sigma proses ditaksir dari R-bar / d2 dengan d2 = 2,326.
func BuildControlChart(values, ranges []float64, subgroupSize int, spec float64, hasSpec bool) ControlChart {
	c := ControlChart{Values: values, Spec: spec, HasSpec: hasSpec}
	if len(values) == 0 {
		return c
	}
	var sum, rsum float64
	for i, v := range values {
		sum += v
		if i < len(ranges) {
			rsum += ranges[i]
		}
	}
	c.CenterLine = sum / float64(len(values))
	rbar := rsum / float64(len(ranges))

	a2, d2 := constantsFor(subgroupSize)
	c.UCL = c.CenterLine + a2*rbar
	c.LCL = c.CenterLine - a2*rbar
	if c.LCL < 0 {
		c.LCL = 0
	}
	if d2 > 0 {
		c.Sigma = rbar / d2
	}

	// Aturan Nelson harus diuji terhadap batas yang SAMA dengan yang digambar.
	// Pada peta X-bar, batas kendali adalah tiga sigma dari rata-rata subgrup
	// (sigma dibagi akar n), bukan tiga sigma dari pengukuran individual.
	// Memakai sigma individual akan membuat titik yang jelas-jelas di luar
	// garis merah pada grafik tetap lolos dari pemeriksaan aturan.
	sigmaSubgroup := (c.UCL - c.CenterLine) / 3
	c.Violations = nelsonRules(values, c.CenterLine, sigmaSubgroup)
	c.InControl = len(c.Violations) == 0

	if hasSpec && c.Sigma > 0 {
		// Cpk satu sisi: hanya batas atas yang relevan untuk waktu respons -
		// respons yang terlalu cepat bukan masalah.
		c.Cpk = (spec - c.CenterLine) / (3 * c.Sigma)
		c.HasCpk = true
	}
	return c
}

// constantsFor mengembalikan konstanta A2 dan d2 untuk ukuran subgrup.
func constantsFor(n int) (a2, d2 float64) {
	table := map[int][2]float64{
		2: {1.880, 1.128}, 3: {1.023, 1.693}, 4: {0.729, 2.059},
		5: {0.577, 2.326}, 6: {0.483, 2.534}, 7: {0.419, 2.704},
		8: {0.373, 2.847}, 9: {0.337, 2.970}, 10: {0.308, 3.078},
	}
	if v, ok := table[n]; ok {
		return v[0], v[1]
	}
	return 0.577, 2.326
}

// nelsonRules menerapkan empat aturan Nelson yang paling sering dipakai.
//
// Aturan 1 dan 2 adalah yang disebut eksplisit di Pertemuan 9 (satu titik di
// luar batas, dan tujuh titik berurutan di sisi yang sama - "rule of seven").
// Aturan 3 dan 5 ditambahkan karena keduanya menangkap pergeseran bertahap,
// persis pola yang muncul pada waktu respons sistem yang memburuk perlahan
// seiring bertambahnya data.
func nelsonRules(values []float64, center, sigma float64) []Violation {
	var out []Violation
	if sigma <= 0 {
		return out
	}
	ucl := center + 3*sigma
	lcl := center - 3*sigma

	// Aturan 1: satu titik di luar 3 sigma.
	for i, v := range values {
		if v > ucl || v < lcl {
			out = append(out, Violation{Rule: 1, Index: i, Text: model.Text{
				ID: "Satu titik berada di luar batas kendali 3 sigma",
				EN: "A single point lies beyond the 3-sigma control limit",
			}})
		}
	}
	// Aturan 2: tujuh titik berurutan di sisi yang sama terhadap garis tengah.
	run, dir := 0, 0
	for i, v := range values {
		d := 0
		if v > center {
			d = 1
		} else if v < center {
			d = -1
		}
		if d != 0 && d == dir {
			run++
		} else {
			run, dir = 1, d
		}
		if run == 7 {
			out = append(out, Violation{Rule: 2, Index: i, Text: model.Text{
				ID: "Tujuh titik berurutan berada di sisi yang sama terhadap garis tengah",
				EN: "Seven consecutive points fall on the same side of the centre line",
			}})
		}
	}
	// Aturan 3: enam titik berurutan yang terus menaik atau menurun.
	trend := 0
	for i := 1; i < len(values); i++ {
		switch {
		case values[i] > values[i-1]:
			if trend > 0 {
				trend++
			} else {
				trend = 1
			}
		case values[i] < values[i-1]:
			if trend < 0 {
				trend--
			} else {
				trend = -1
			}
		default:
			trend = 0
		}
		if trend >= 5 || trend <= -5 {
			out = append(out, Violation{Rule: 3, Index: i, Text: model.Text{
				ID: "Enam titik berurutan bergerak searah - proses sedang bergeser, bukan berfluktuasi",
				EN: "Six consecutive points move in one direction - the process is drifting, not fluctuating",
			}})
			trend = 0
		}
	}
	// Aturan 5: dua dari tiga titik berurutan berada di luar 2 sigma pada
	// sisi yang sama.
	for i := 2; i < len(values); i++ {
		hi, lo := 0, 0
		for _, v := range values[i-2 : i+1] {
			if v > center+2*sigma {
				hi++
			}
			if v < center-2*sigma {
				lo++
			}
		}
		if hi >= 2 || lo >= 2 {
			out = append(out, Violation{Rule: 5, Index: i, Text: model.Text{
				ID: "Dua dari tiga titik berurutan melewati 2 sigma di sisi yang sama",
				EN: "Two of three consecutive points exceed 2 sigma on the same side",
			}})
		}
	}
	return out
}

// ------------------------------------------------------- Cost of quality

// COQSummary adalah ringkasan biaya kualitas.
type COQSummary struct {
	Prevention      float64
	Appraisal       float64
	InternalFailure float64
	ExternalFailure float64
	Conformance     float64 // pencegahan + penilaian
	Nonconformance  float64 // kegagalan internal + eksternal
	Total           float64
	// Ratio adalah biaya kesesuaian dibagi biaya ketidaksesuaian. Di bawah 1
	// berarti proyek lebih banyak membayar akibat daripada mencegah sebab.
	Ratio float64
	// ShareOfBudget adalah total biaya kualitas terhadap BAC.
	ShareOfBudget float64
	Items         []model.COQItem
}

// SummariseCOQ meringkas seluruh pos biaya kualitas.
func SummariseCOQ(items []model.COQItem, bac float64) COQSummary {
	s := COQSummary{Items: items}
	for _, it := range items {
		switch it.Category {
		case model.COQPrevention:
			s.Prevention += it.Amount
		case model.COQAppraisal:
			s.Appraisal += it.Amount
		case model.COQInternalFailure:
			s.InternalFailure += it.Amount
		case model.COQExternalFailure:
			s.ExternalFailure += it.Amount
		}
	}
	s.Conformance = s.Prevention + s.Appraisal
	s.Nonconformance = s.InternalFailure + s.ExternalFailure
	s.Total = s.Conformance + s.Nonconformance
	if s.Nonconformance > 0 {
		s.Ratio = s.Conformance / s.Nonconformance
	}
	if bac > 0 {
		s.ShareOfBudget = s.Total / bac
	}
	return s
}

// ------------------------------------------------------------- Metrik

// MetricStatus adalah hasil evaluasi satu metrik mutu terhadap targetnya.
type MetricStatus struct {
	model.QualityMetric
	Met   bool
	Gap   float64 // jarak ke target, positif berarti belum tercapai
	Ratio float64
}

// EvaluateMetrics menilai seluruh metrik mutu terhadap targetnya.
func EvaluateMetrics(metrics []model.QualityMetric) []MetricStatus {
	out := make([]MetricStatus, 0, len(metrics))
	for _, m := range metrics {
		st := MetricStatus{QualityMetric: m}
		if m.HigherIsBetter {
			st.Met = m.Actual >= m.Target
			st.Gap = m.Target - m.Actual
			if m.Target != 0 {
				st.Ratio = m.Actual / m.Target
			}
		} else {
			st.Met = m.Actual <= m.Target
			st.Gap = m.Actual - m.Target
			if m.Actual != 0 {
				st.Ratio = m.Target / m.Actual
			}
		}
		if st.Gap < 0 {
			st.Gap = 0
		}
		out = append(out, st)
	}
	return out
}

// DefectDensity mengembalikan rata-rata cacat berbobot per modul - ukuran
// kasar yang berguna sebagai ambang pemicu pada risk register.
func DefectDensity(defects []model.DefectRecord) float64 {
	if len(defects) == 0 {
		return 0
	}
	var total float64
	for _, d := range defects {
		total += float64(d.Weighted())
	}
	return total / float64(len(defects))
}

// TotalRework mengembalikan total jam rework yang tercatat.
func TotalRework(defects []model.DefectRecord) float64 {
	var total float64
	for _, d := range defects {
		total += d.ReworkHours
	}
	return total
}

// Round membulatkan ke n angka desimal - dipakai templat agar tidak perlu
// memanggil fungsi format berulang kali.
func Round(v float64, n int) float64 {
	p := math.Pow(10, float64(n))
	return math.Round(v*p) / p
}
