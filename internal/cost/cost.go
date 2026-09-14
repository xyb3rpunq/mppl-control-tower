// Package cost memisahkan biaya yang bergantung waktu dari biaya yang tidak.
//
// Anggaran bottom-up proyek ini memuat dua jenis biaya non-tenaga-kerja.
// Pembelian sekali (hosting produksi setahun, pelatihan) tidak peduli proyek
// selesai kapan. Sewa dan langganan (server pengembangan, lisensi tools
// bulanan) berbeda: setiap hari proyek molor, tagihannya bertambah.
//
// Nilai pada anggaran adalah nilai untuk rentang RENCANA, sehingga pada
// jadwal CPM rencana biaya totalnya persis sama dengan BAC. Tarif hariannya
// diturunkan dari rentang itu:
//
//	tarif = nilai / (durasi proyek rencana - mulai aktivitas pembeli)
//	biaya = tarif x (selesai proyek - mulai aktivitas pembeli)
package cost

import (
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
)

// Rental adalah satu biaya sewa atau langganan.
type Rental struct {
	Index    int    // indeks aktivitas pembeli pada irisan aktivitas
	Activity string // kode aktivitas pembeli
	Label    model.Text
	Amount   float64 // nilai pada rentang rencana
	PlanSpan int     // rentang rencana, hari kerja
	Rate     float64 // rupiah per hari kerja
}

// Cost mengembalikan biaya sewa untuk proyek yang selesai pada finish bila
// aktivitas pembelinya mulai pada start.
func (r Rental) Cost(start, finish int) float64 {
	span := finish - start
	if span < 0 {
		span = 0
	}
	return r.Rate * float64(span)
}

// Rentals menurunkan tarif harian seluruh biaya sewa dari jadwal CPM rencana,
// dan mengembalikan jumlah biaya non-tenaga-kerja lain yang tetap.
func Rentals(acts []model.Activity) ([]Rental, float64, error) {
	plan, err := schedule.Compute(acts, schedule.Options{})
	if err != nil {
		return nil, 0, err
	}
	var out []Rental
	var fixed float64
	for i, a := range acts {
		for _, e := range a.Extras {
			if !e.TimeBased {
				fixed += e.Amount
				continue
			}
			span := plan.Duration - plan.Task(a.ID).StartX
			if span <= 0 {
				fixed += e.Amount
				continue
			}
			out = append(out, Rental{
				Index: i, Activity: a.ID, Label: e.Label, Amount: e.Amount,
				PlanSpan: span, Rate: e.Amount / float64(span),
			})
		}
	}
	return out, fixed, nil
}

// DailyRate mengembalikan jumlah tarif harian seluruh sewa: tambahan biaya
// untuk setiap hari keterlambatan setelah semua aktivitas pembeli dimulai.
func DailyRate(rs []Rental) float64 {
	var t float64
	for _, r := range rs {
		t += r.Rate
	}
	return t
}
