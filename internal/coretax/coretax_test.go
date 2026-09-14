package coretax

import (
	"strings"
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

func TestMirrorComparesBothProjects(t *testing.T) {
	d := Compute()
	rows := Mirror(d, model.BAC(), 85)
	if len(rows) != 4 {
		t.Fatalf("tabel cermin punya %d baris, mau 4", len(rows))
	}
	if !strings.Contains(rows[0].Ratio, "ribu kali") {
		t.Errorf("rasio nilai proyek %q seharusnya dalam ribuan kali", rows[0].Ratio)
	}
	for _, r := range rows {
		if r.Label.ID == "" || r.Label.EN == "" || r.Insight.ID == "" || r.Insight.EN == "" || r.Ratio == "" {
			t.Errorf("baris cermin %q tidak lengkap", r.Label.ID)
		}
	}
}

func TestNumberFormattingHelpers(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{{47.83, "47,8x"}, {12, "12x"}, {90250, "~90,3 ribu kali"}, {1000, "~1 ribu kali"}} {
		if got := formatRatio(c.in); got != c.want {
			t.Errorf("formatRatio(%v) = %q, mau %q", c.in, got, c.want)
		}
	}
	if trimFloat(3.04) != "3" || trimFloat(2.56) != "2,6" {
		t.Errorf("trimFloat salah: %q, %q", trimFloat(3.04), trimFloat(2.56))
	}
	if formatInt(0) != "0" || formatInt(-45) != "-45" || formatInt(1234) != "1234" {
		t.Error("formatInt salah pada nol, negatif, atau ribuan")
	}
}
