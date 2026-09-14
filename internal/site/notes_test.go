package site

import (
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Cabang teks yang tidak terpicu data proyek tetap harus benar: temuan dan
// contoh hitung mengikuti hasil perbandingan, bukan ditulis tetap.
func TestNotesFollowTheNumbers(t *testing.T) {
	a := &Analysis{Exact: compress.TradeOff{GreedyOptimal: false, MaxGreedyExcess: 4062.5}}
	if id, en := greedyNoteID(a), greedyNoteEN(a); !strings.Contains(id, "Rp 4.063") || !strings.Contains(en, "Rp 4.063") {
		t.Errorf("catatan serakah tidak optimal tidak memuat kelebihannya: %q / %q", id, en)
	}
	a.Exact.GreedyOptimal = true
	if id, en := greedyNoteID(a), greedyNoteEN(a); strings.Contains(id, "Rp") || !strings.Contains(en, "only the LP can prove it") {
		t.Errorf("catatan serakah optimal keliru: %q / %q", id, en)
	}

	if a.overtimeActionID() != "" || a.overtimeActionEN() != "" || a.budgetRequestID() != "" || a.budgetRequestEN() != "" {
		t.Error("tanpa paket keputusan, saran turunan harus kosong")
	}
	if !strings.Contains(a.commitmentID(), "Keputusan Sponsor") || !strings.Contains(a.commitmentEN(), "Sponsor Decisions") {
		t.Error("tanpa JCL 70% berjalan, komitmen merujuk ke halaman keputusan")
	}
	a.Decision = &Decision{Cheapest: -1, Fastest: -1, Options: []AccelOption{{Key: "tanpa", Floor: 50}}}
	if id, en := a.overtimeActionID(), a.overtimeActionEN(); !strings.Contains(id, "tidak ada opsi") || !strings.Contains(en, "no option") {
		t.Errorf("saran tanpa opsi yang mempercepat keliru: %q / %q", id, en)
	}
	if _, ok := a.Decision.CheapestOption(); ok {
		t.Error("tanpa opsi termurah, CheapestOption harus kosong")
	}
	if _, ok := a.Decision.FastestOption(); ok {
		t.Error("tanpa opsi tercepat, FastestOption harus kosong")
	}
	if _, ok := a.Decision.Option("tidak-ada"); ok {
		t.Error("kunci opsi yang tidak ada harus melapor tidak ditemukan")
	}
	if a.Decision.RampChangesNothing() {
		t.Error("tanpa opsi orang baru, masa adaptasi tidak bisa dinilai")
	}
	a.Decision.Options = append(a.Decision.Options, AccelOption{Key: "lembur", Name: model.Text{ID: "Lembur sah BE", EN: "Legal overtime BE"}, DaysEarlier: 2, PricePerDay: 2500})
	a.Decision.Cheapest = 1
	if id := a.overtimeActionID(); !strings.Contains(id, "lembur sah BE") || !strings.Contains(id, "Rp 2.500") {
		t.Errorf("saran lembur tidak memakai opsi termurah: %q", id)
	}
	if lowerFirst("BE, FE") != "BE, FE" || lowerFirst("Tambah satu") != "tambah satu" || lowerFirst("x") != "x" {
		t.Error("lowerFirst harus mempertahankan singkatan peran")
	}
}
