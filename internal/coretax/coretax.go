// Package coretax menurunkan metrik analitis dari fakta terverifikasi tentang
// proyek Coretax DJP.
//
// Pemisahan tanggung jawab dijaga ketat:
//
//	model.CoretaxFacts    - fakta, masing-masing punya URL sumber
//	coretax.Derived       - aritmetika murni atas fakta di atas
//	coretax.Scenarios     - andaian, diberi label skenario, bukan fakta
//
// Setiap angka pada Derived bisa ditelusuri balik ke konstanta terverifikasi
// lewat rumus yang tertulis di komentar fungsinya. Tidak ada angka yang muncul
// begitu saja.
package coretax

import (
	"math"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Derived adalah metrik hasil hitungan atas fakta terverifikasi.
type Derived struct {
	// TotalContract = nilai kontrak utama + kontrak Owner's Agent.
	TotalContract float64
	// AwardToGoLiveDays = jumlah hari kalender dari penetapan pemenang tender
	// (1 Des 2020) sampai go-live (1 Jan 2025).
	AwardToGoLiveDays   int
	AwardToGoLiveMonths float64
	AwardToGoLiveYears  float64
	// PerpresToGoLiveYears = rentang dari Perpres 40/2018 sampai go-live.
	PerpresToGoLiveYears float64
	// BurnPerMonth = TotalContract / AwardToGoLiveMonths.
	BurnPerMonth float64
	// BurnPerPersonMonth = BurnPerMonth / jumlah tim PSIAP.
	BurnPerPersonMonth float64
	// ExposureRatio = potensi penerimaan hilang sebulan / TotalContract.
	// Inilah angka tunggal paling penting dari seluruh studi kasus ini.
	ExposureRatio float64
	// LossPerDayJanuary = potensi hilang Januari / 31 hari.
	LossPerDayJanuary float64
	// DaysOfLossEqualToProject = berapa hari kerugian setara seluruh biaya proyek.
	DaysOfLossEqualToProject float64
	// ImpliedAnnualTarget = penerimaan kuartal I / porsi target yang dicapai.
	ImpliedAnnualTarget float64
	// RemediationSlipDays = pengunduran tenggat perbaikan dari Mei ke 31 Juli.
	RemediationSlipDays int
	// StabilisationDays = hari dari go-live sampai evaluasi satu tahun.
	StabilisationDays int
}

// Compute menurunkan seluruh metrik dari konstanta terverifikasi.
func Compute() Derived {
	var d Derived

	d.TotalContract = model.CoretaxContractValue + model.CoretaxConsultantValue

	award := workcal.MustParseISO("2020-12-01")
	goLive := workcal.MustParseISO("2025-01-01")
	perpres := workcal.MustParseISO("2018-01-01")
	oneYear := workcal.MustParseISO("2026-01-20")

	d.AwardToGoLiveDays = int(goLive.Sub(award).Hours() / 24)
	d.AwardToGoLiveMonths = float64(d.AwardToGoLiveDays) / 30.4375
	d.AwardToGoLiveYears = float64(d.AwardToGoLiveDays) / 365.25
	d.PerpresToGoLiveYears = goLive.Sub(perpres).Hours() / 24 / 365.25

	d.BurnPerMonth = d.TotalContract / d.AwardToGoLiveMonths
	d.BurnPerPersonMonth = d.BurnPerMonth / float64(model.CoretaxPSIAPHeadcount)

	d.ExposureRatio = model.CoretaxLostRevenueJan2025 / d.TotalContract
	d.LossPerDayJanuary = model.CoretaxLostRevenueJan2025 / 31
	d.DaysOfLossEqualToProject = d.TotalContract / d.LossPerDayJanuary

	// Kuartal I 2025 tercatat Rp 322,6 T atau 14,7% dari target setahun.
	d.ImpliedAnnualTarget = model.CoretaxQ1Revenue / 0.147

	may := workcal.MustParseISO("2025-05-31")
	july := workcal.MustParseISO("2025-07-31")
	d.RemediationSlipDays = int(july.Sub(may).Hours() / 24)
	d.StabilisationDays = int(oneYear.Sub(goLive).Hours() / 24)

	return d
}

// Scenario adalah andaian tandingan - bukan fakta, dan ditandai demikian di
// seluruh tampilan. Gunanya menunjukkan bentuk keputusan yang bisa diambil,
// bukan mengklaim tahu apa yang akan terjadi.
type Scenario struct {
	Key        string
	Name       model.Text
	Assumption model.Text
	// ExtraCost adalah tambahan biaya strategi transisi terhadap biaya proyek.
	ExtraCostShare float64
	ExtraCost      float64
	// FailureProb adalah peluang kegagalan cutover yang diandaikan.
	FailureProb float64
	// ExposureShare adalah porsi paparan Rp 64 triliun yang tetap menimpa
	// meskipun strategi ini dipakai.
	ExposureShare float64
	// EMV = FailureProb * ExposureShare * paparan, ditambah ExtraCost.
	ExpectedLoss float64
	TotalCost    float64
	Note         model.Text
}

// Scenarios membandingkan empat strategi transisi dengan kerangka Expected
// Monetary Value.
//
// Peluang kegagalan dan porsi paparan di sini adalah ANDAIAN untuk menunjukkan
// cara kerja perhitungannya. Yang tidak berupa andaian adalah dua hal: biaya
// proyek Rp 1,34 triliun dan paparan Rp 64 triliun sebulan - keduanya berasal
// dari sumber publik. Selama paparan berada dua kali lipat orde di atas biaya
// proyek, urutan peringkat strateginya tidak berubah walaupun angka andaiannya
// digeser cukup jauh. Itulah inti argumennya.
func Scenarios(d Derived) []Scenario {
	exposure := model.CoretaxLostRevenueJan2025
	base := d.TotalContract

	defs := []Scenario{
		{
			Key: "bigbang", ExtraCostShare: 0, FailureProb: 0.35, ExposureShare: 1.0,
			Name:       model.Text{ID: "Cutover serentak (yang ditempuh)", EN: "Big-bang cutover (the path taken)"},
			Assumption: model.Text{ID: "Seluruh wajib pajak pindah pada satu tanggal, sistem lama dimatikan", EN: "All taxpayers move on one date; the old system is switched off"},
			Note:       model.Text{ID: "Tanpa biaya transisi tambahan, tetapi seluruh paparan ditanggung sekaligus", EN: "No extra transition cost, but the full exposure lands at once"},
		},
		{
			Key: "paralel", ExtraCostShare: 0.12, FailureProb: 0.35, ExposureShare: 0.10,
			Name:       model.Text{ID: "Jalan paralel enam bulan", EN: "Six-month parallel run"},
			Assumption: model.Text{ID: "Sistem lama tetap hidup dan menerima transaksi selama enam bulan pertama", EN: "The old system stays live and accepts transactions for the first six months"},
			Note:       model.Text{ID: "Paling mahal di muka, tetapi menyisakan jalur mundur yang benar-benar bisa dipakai", EN: "Costliest up front, but leaves a fallback that actually works"},
		},
		{
			Key: "bertahap", ExtraCostShare: 0.08, FailureProb: 0.35, ExposureShare: 0.25,
			Name:       model.Text{ID: "Bertahap per segmen wajib pajak", EN: "Phased by taxpayer segment"},
			Assumption: model.Text{ID: "Wajib pajak besar lebih dulu, orang pribadi menyusul setelah stabil", EN: "Large taxpayers first, individuals once the system is stable"},
			Note:       model.Text{ID: "Kegagalan tetap mungkin, tetapi hanya menimpa sebagian populasi pada satu waktu", EN: "Failure is still possible but hits only part of the population at a time"},
		},
		{
			Key: "pilot", ExtraCostShare: 0.05, FailureProb: 0.35, ExposureShare: 0.05,
			Name:       model.Text{ID: "Pilot satu kantor wilayah", EN: "Pilot in one regional office"},
			Assumption: model.Text{ID: "Satu kanwil memakai sistem penuh selama satu musim pelaporan sebelum nasional", EN: "One regional office runs the full system for a reporting season before national rollout"},
			Note:       model.Text{ID: "Termurah untuk menemukan kelas cacat yang sama, tetapi menambah waktu ke jalur kritis", EN: "Cheapest way to surface the same defect classes, but adds time to the critical path"},
		},
	}

	out := make([]Scenario, 0, len(defs))
	for _, s := range defs {
		s.ExtraCost = base * s.ExtraCostShare
		s.ExpectedLoss = s.FailureProb * s.ExposureShare * exposure
		s.TotalCost = s.ExtraCost + s.ExpectedLoss
		out = append(out, s)
	}
	return out
}

// BreakEvenFailureProb menghitung pada peluang kegagalan berapa strategi
// transisi berbiaya extraShare mulai lebih murah daripada cutover serentak.
//
// Cutover serentak: EMV = p * paparan
// Strategi lain   : EMV = biaya*extraShare + p * sisaPaparan * paparan
// Impas ketika    : p * paparan = biaya*extraShare + p * sisaPaparan * paparan
//
//	p = (biaya * extraShare) / (paparan * (1 - sisaPaparan))
func BreakEvenFailureProb(base, exposure, extraShare, residualShare float64) float64 {
	denom := exposure * (1 - residualShare)
	if denom <= 0 {
		return math.NaN()
	}
	return (base * extraShare) / denom
}

// MirrorMetric membandingkan satu metrik Coretax dengan padanannya pada proyek
// SIATS, supaya skala dan kesamaan polanya terlihat berdampingan.
type MirrorMetric struct {
	Label   model.Text
	Coretax string
	SIATS   string
	Ratio   string
	Insight model.Text
}

// Mirror membangun tabel perbandingan dua proyek.
func Mirror(d Derived, siatsBudget, siatsDurationDays float64) []MirrorMetric {
	scale := d.TotalContract / siatsBudget
	return []MirrorMetric{
		{
			Label:   model.Text{ID: "Nilai proyek", EN: "Project value"},
			Coretax: "Rp 1,338 triliun", SIATS: "Rp 14,83 juta",
			Ratio:   formatRatio(scale),
			Insight: model.Text{ID: "Selisih skala sekitar sembilan puluh ribu kali, tetapi daftar penyebab kegagalannya sama persis", EN: "Roughly ninety thousand times apart in scale, yet the failure causes are identical"},
		},
		{
			Label:   model.Text{ID: "Rentang kontrak sampai go-live", EN: "Award to go-live"},
			Coretax: "1.492 hari (4,1 tahun)", SIATS: "85 hari kerja (17 minggu)",
			Ratio:   "17,6x",
			Insight: model.Text{ID: "Keduanya menaruh pertemuan pertama pengguna dengan sistem tepat di hari sistem itu wajib dipakai", EN: "Both put the users' first encounter with the system on the very day it becomes mandatory"},
		},
		{
			Label:   model.Text{ID: "Strategi transisi", EN: "Transition strategy"},
			Coretax: "Serentak, sistem lama dimatikan", SIATS: "Serentak, tanpa periode paralel",
			Ratio:   "identik",
			Insight: model.Text{ID: "Tidak satu pun menyediakan jalur mundur yang teruji", EN: "Neither provides a rehearsed fallback"},
		},
		{
			Label:   model.Text{ID: "Nilai yang dipertaruhkan di luar biaya proyek", EN: "Value at stake beyond project cost"},
			Coretax: "Rp 64 triliun penerimaan sebulan", SIATS: "Peringkat akreditasi program studi",
			Ratio:   "47,8x biaya proyek",
			Insight: model.Text{ID: "Cadangan pada kedua proyek dihitung terhadap biaya proyek, bukan terhadap nilai yang dipertaruhkan", EN: "Both sized reserves against project cost rather than against the value at stake"},
		},
	}
}

func formatRatio(v float64) string {
	switch {
	case v >= 1000:
		return "~" + trimFloat(v/1000) + " ribu kali"
	default:
		return trimFloat(v) + "x"
	}
}

func trimFloat(v float64) string {
	r := math.Round(v*10) / 10
	s := ""
	if r == math.Trunc(r) {
		s = formatInt(int64(r))
	} else {
		s = formatInt(int64(math.Trunc(r))) + "," + formatInt(int64(math.Round((r-math.Trunc(r))*10)))
	}
	return s
}

func formatInt(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf []byte
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
