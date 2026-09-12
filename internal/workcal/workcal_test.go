package workcal_test

import (
	"testing"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

func TestWeekendsAreSkipped(t *testing.T) {
	// 2025-10-20 adalah Senin. Lima hari kerja pertama harus Senin sampai
	// Jumat; hari keenam melompat ke Senin berikutnya.
	c := workcal.MustNew("2025-10-20", 10)
	want := []string{
		"2025-10-20", "2025-10-21", "2025-10-22", "2025-10-23", "2025-10-24",
		"2025-10-27", "2025-10-28", "2025-10-29", "2025-10-30", "2025-10-31",
	}
	for i, w := range want {
		if got := c.ISOAt(i); got != w {
			t.Errorf("hari kerja ke-%d = %s, mau %s", i, got, w)
		}
	}
}

func TestHolidaysAreSkipped(t *testing.T) {
	c := workcal.MustNew("2025-12-22", 10)
	// 25 dan 26 Desember adalah hari libur; 27-28 akhir pekan.
	// Jadi setelah 24 Des, hari kerja berikutnya adalah 29 Des.
	for i := 0; i < c.Len(); i++ {
		iso := c.ISOAt(i)
		if iso == "2025-12-25" || iso == "2025-12-26" {
			t.Errorf("hari libur %s ikut terhitung sebagai hari kerja", iso)
		}
	}
	if got := c.ISOAt(2); got != "2025-12-24" {
		t.Errorf("hari kerja ke-2 = %s, mau 2025-12-24", got)
	}
	if got := c.ISOAt(3); got != "2025-12-29" {
		t.Errorf("hari kerja ke-3 = %s, mau 2025-12-29 (melompati libur dan akhir pekan)", got)
	}
}

// TestCharterFinishDateIsLaterThanNaiveCalculation menjaga temuan yang
// ditampilkan di halaman Piagam: 17 minggu kalender polos dan 85 hari kerja
// adalah dua tanggal yang berbeda.
func TestCharterFinishDateIsLaterThanNaiveCalculation(t *testing.T) {
	c := workcal.MustNew(model.ProjectCharter.StartDate, 200)
	finish := c.ISOAt(84) // hari kerja ke-85
	if finish <= model.ProjectCharter.TargetFinish {
		t.Errorf("hari kerja ke-85 jatuh pada %s, tidak lebih lambat dari target piagam %s; "+
			"temuan soal hari libur di halaman Piagam jadi tidak berlaku",
			finish, model.ProjectCharter.TargetFinish)
	}
	if finish != "2026-02-20" {
		t.Errorf("hari kerja ke-85 = %s, mau 2026-02-20", finish)
	}
}

func TestIndexRoundTrip(t *testing.T) {
	c := workcal.MustNew("2025-10-20", 120)
	for i := 0; i < c.Len(); i++ {
		if got := c.IndexOf(c.ISOAt(i)); got != i {
			t.Fatalf("bolak-balik indeks gagal di %d: dapat %d", i, got)
		}
	}
}

func TestIndexOfNonWorkingDayRoundsForward(t *testing.T) {
	c := workcal.MustNew("2025-10-20", 60)
	// 2025-10-25 adalah Sabtu; harus dibulatkan ke Senin 27 Oktober.
	sat := c.IndexOf("2025-10-25")
	mon := c.IndexOf("2025-10-27")
	if sat != mon {
		t.Errorf("Sabtu memetakan ke indeks %d, Senin ke %d; Sabtu harus maju ke Senin", sat, mon)
	}
}

func TestFractionalIndexIsMonotonic(t *testing.T) {
	c := workcal.MustNew("2025-10-20", 200)
	prev := -1.0
	for _, iso := range []string{
		"2025-10-20", "2025-10-24", "2025-10-25", "2025-10-27",
		"2025-12-19", "2025-12-25", "2026-01-05", "2026-02-20",
	} {
		got := c.FractionalIndexOf(iso)
		if got < prev {
			t.Errorf("indeks pecahan turun pada %s: %v setelah %v", iso, got, prev)
		}
		prev = got
	}
}

func TestOutOfRangeDatesAreClamped(t *testing.T) {
	c := workcal.MustNew("2025-10-20", 50)
	if got := c.IndexOf("2024-01-01"); got != -1 {
		t.Errorf("tanggal sebelum proyek = %d, mau -1", got)
	}
	if got := c.IndexOf("2030-01-01"); got != c.Len() {
		t.Errorf("tanggal setelah horizon = %d, mau %d", got, c.Len())
	}
	// Date harus ter-clamp, bukan panik.
	if c.Date(-5).IsZero() || c.Date(9999).IsZero() {
		t.Error("Date harus meng-clamp indeks di luar rentang")
	}
}

func TestFormatDateBilingual(t *testing.T) {
	if got := workcal.FormatDate("2025-10-20", "id"); got != "20 Okt 2025" {
		t.Errorf("format id = %q, mau \"20 Okt 2025\"", got)
	}
	if got := workcal.FormatDate("2025-10-20", "en"); got != "20 Oct 2025" {
		t.Errorf("format en = %q, mau \"20 Oct 2025\"", got)
	}
	// Bulan yang namanya berbeda antar bahasa: Agustus vs August.
	if got := workcal.FormatDate("2025-08-01", "id"); got != "1 Agu 2025" {
		t.Errorf("format id Agustus = %q", got)
	}
}

func TestInvalidInputIsRejected(t *testing.T) {
	if _, err := workcal.New("bukan-tanggal", 10); err == nil {
		t.Error("tanggal tidak sah seharusnya menghasilkan galat")
	}
	if _, err := workcal.New("2025-10-20", 0); err == nil {
		t.Error("horizon nol seharusnya menghasilkan galat")
	}
}

func TestHolidaysBetween(t *testing.T) {
	c := workcal.MustNew(model.ProjectCharter.StartDate, 200)
	got := c.HolidaysBetween(0, 84)
	if len(got) != len(workcal.Holidays) {
		t.Errorf("hari libur di dalam rentang proyek = %d, mau %d", len(got), len(workcal.Holidays))
	}
	// Rentang sempit di awal proyek tidak boleh memuat libur Desember.
	if n := len(c.HolidaysBetween(0, 5)); n != 0 {
		t.Errorf("minggu pertama seharusnya tanpa hari libur, dapat %d", n)
	}
}
