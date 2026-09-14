package site

import (
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/compress"
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

	a.Overtime = compress.OvertimeCurve{Levelled: 10, MinDuration: 10}
	if id, en := a.overtimeActionID(), a.overtimeActionEN(); !strings.Contains(id, "tidak memendekkan") || !strings.Contains(en, "does not shorten") {
		t.Errorf("saran tanpa rencana lembur keliru: %q / %q", id, en)
	}
	a.Overtime = compress.OvertimeCurve{Levelled: 10, MinDuration: 8, Points: []compress.OvertimePoint{{Duration: 9, Cost: 1000}, {Duration: 8, Cost: 2500}}}
	if id := a.overtimeActionID(); !strings.Contains(id, "2 hari") || !strings.Contains(id, "Rp 2.500") {
		t.Errorf("saran lembur tidak memakai rencana durasi minimum: %q", id)
	}
}
