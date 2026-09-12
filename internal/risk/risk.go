// Package risk menghitung analisis risiko kuantitatif.
//
// Yang membedakan analisis kuantitatif dari sekadar daftar risiko berwarna
// adalah satu pertanyaan: berapa rupiah cadangan yang harus disiapkan? Nilainya
// adalah jumlah Expected Monetary Value dari risiko RESIDUAL - risiko setelah
// respons dijalankan. Memakai risiko inheren akan melebih-lebihkan cadangan
// dan mengunci anggaran yang sebenarnya masih bisa dipakai.
package risk

import (
	"sort"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Level adalah tingkat 1..5 untuk sumbu matriks probabilitas-dampak.
type Level int

// ProbabilityLevel memetakan peluang 0..1 ke tingkat 1..5.
func ProbabilityLevel(p float64) Level {
	switch {
	case p < 0.10:
		return 1
	case p < 0.25:
		return 2
	case p < 0.45:
		return 3
	case p < 0.65:
		return 4
	default:
		return 5
	}
}

// ImpactLevel memetakan dampak rupiah ke tingkat 1..5. Ambangnya relatif
// terhadap BAC supaya matriks ikut menyesuaikan bila anggaran proyek berubah.
func ImpactLevel(impact, bac float64) Level {
	if bac <= 0 {
		return 3
	}
	share := impact / bac
	switch {
	case share < 0.02:
		return 1
	case share < 0.05:
		return 2
	case share < 0.10:
		return 3
	case share < 0.20:
		return 4
	default:
		return 5
	}
}

// Severity adalah kategori keparahan hasil perkalian tingkat.
type Severity string

// Kategori keparahan yang dipakai untuk pewarnaan peta panas.
const (
	SeverityLow      Severity = "rendah"
	SeverityModerate Severity = "sedang"
	SeverityHigh     Severity = "tinggi"
	SeverityExtreme  Severity = "ekstrem"
)

// SeverityOf mengelompokkan skor 1..25 menjadi empat kategori.
func SeverityOf(score int) Severity {
	switch {
	case score <= 4:
		return SeverityLow
	case score <= 9:
		return SeverityModerate
	case score <= 15:
		return SeverityHigh
	default:
		return SeverityExtreme
	}
}

// Scored adalah satu risiko yang sudah dinilai.
type Scored struct {
	model.Risk
	ProbLevel        Level
	ImpactLev        Level
	Score            int // ProbLevel * ImpactLev, 1..25
	Severity         Severity
	ResidualProbLev  Level
	ResidualImpLev   Level
	ResidualScore    int
	ResidualSeverity Severity
	EMV              float64
	ResidualEMV      float64
	// Reduction adalah porsi EMV yang dihapus oleh respons; inilah ukuran
	// "seberapa berguna rencana mitigasi ini", bukan sekadar ada atau tidak.
	Reduction float64
}

// Register adalah hasil analisis seluruh risk register.
type Register struct {
	Risks []Scored

	TotalEMV         float64
	TotalResidualEMV float64
	// ReserveGap adalah selisih antara EMV residual dan cadangan kontinjensi
	// yang tersedia. Positif berarti cadangan kurang.
	ReserveGap        float64
	ReserveCoverage   float64 // porsi EMV residual yang tertutup cadangan
	ScheduleExposure  float64 // harapan hari kerja tambahan dari seluruh risiko
	ByCategory        []CategoryRow
	Matrix            [5][5][]string // [tingkat dampak-1][tingkat peluang-1] -> kode risiko
	ResidualMatrix    [5][5][]string
	TopByEMV          []Scored
	SeverityBreakdown map[Severity]int
}

// CategoryRow adalah agregasi risiko per kategori.
type CategoryRow struct {
	Category    model.Text
	Count       int
	EMV         float64
	ResidualEMV float64
}

// Analyse menilai seluruh risk register terhadap sebuah BAC dan cadangan.
func Analyse(risks []model.Risk, bac, contingency float64) Register {
	reg := Register{SeverityBreakdown: map[Severity]int{}}
	catIndex := map[string]int{}

	for _, r := range risks {
		s := Scored{
			Risk:            r,
			ProbLevel:       ProbabilityLevel(r.Probability),
			ImpactLev:       ImpactLevel(r.Impact, bac),
			ResidualProbLev: ProbabilityLevel(r.ResidualProb),
			ResidualImpLev:  ImpactLevel(r.ResidualImpact, bac),
			EMV:             r.EMV(),
			ResidualEMV:     r.ResidualEMV(),
		}
		s.Score = int(s.ProbLevel) * int(s.ImpactLev)
		s.Severity = SeverityOf(s.Score)
		s.ResidualScore = int(s.ResidualProbLev) * int(s.ResidualImpLev)
		s.ResidualSeverity = SeverityOf(s.ResidualScore)
		if s.EMV > 0 {
			s.Reduction = (s.EMV - s.ResidualEMV) / s.EMV
		}

		reg.Risks = append(reg.Risks, s)
		reg.TotalEMV += s.EMV
		reg.TotalResidualEMV += s.ResidualEMV
		reg.ScheduleExposure += r.ResidualProb * float64(r.ScheduleImpact)
		reg.SeverityBreakdown[s.Severity]++

		reg.Matrix[int(s.ImpactLev)-1][int(s.ProbLevel)-1] =
			append(reg.Matrix[int(s.ImpactLev)-1][int(s.ProbLevel)-1], r.ID)
		reg.ResidualMatrix[int(s.ResidualImpLev)-1][int(s.ResidualProbLev)-1] =
			append(reg.ResidualMatrix[int(s.ResidualImpLev)-1][int(s.ResidualProbLev)-1], r.ID)

		key := r.Category.ID
		if i, ok := catIndex[key]; ok {
			reg.ByCategory[i].Count++
			reg.ByCategory[i].EMV += s.EMV
			reg.ByCategory[i].ResidualEMV += s.ResidualEMV
		} else {
			catIndex[key] = len(reg.ByCategory)
			reg.ByCategory = append(reg.ByCategory, CategoryRow{
				Category: r.Category, Count: 1, EMV: s.EMV, ResidualEMV: s.ResidualEMV,
			})
		}
	}

	reg.ReserveGap = reg.TotalResidualEMV - contingency
	if reg.TotalResidualEMV > 0 {
		reg.ReserveCoverage = contingency / reg.TotalResidualEMV
	} else {
		reg.ReserveCoverage = 1
	}

	sort.Slice(reg.ByCategory, func(i, j int) bool {
		return reg.ByCategory[i].ResidualEMV > reg.ByCategory[j].ResidualEMV
	})

	top := append([]Scored(nil), reg.Risks...)
	sort.Slice(top, func(i, j int) bool { return top[i].ResidualEMV > top[j].ResidualEMV })
	reg.TopByEMV = top

	return reg
}

// SortedByScore mengembalikan risiko terurut dari skor inheren tertinggi.
func (r Register) SortedByScore() []Scored {
	out := append([]Scored(nil), r.Risks...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].EMV > out[j].EMV
	})
	return out
}
