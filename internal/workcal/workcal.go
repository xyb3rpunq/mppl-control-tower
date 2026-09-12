// Package workcal memodelkan kalender kerja proyek.
//
// Seluruh penjadwalan di aplikasi ini memakai satuan HARI KERJA (Senin-Jumat,
// dikurangi hari libur). Paket inilah satu-satunya yang tahu soal tanggal
// kalender; engine CPM bekerja murni dengan indeks hari kerja (0 = hari kerja
// pertama proyek) supaya bebas dari kerumitan zona waktu dan akhir pekan.
package workcal

import (
	"fmt"
	"time"
)

// Holiday adalah satu hari libur di jendela proyek.
type Holiday struct {
	Date   string // YYYY-MM-DD
	NameID string
	NameEN string
	// Asumsi menandai tanggal yang belum ditetapkan lewat SKB resmi saat
	// dokumen proyek disusun. Ditampilkan apa adanya di UI supaya pembaca
	// bisa mengoreksi tanpa menyentuh kode penjadwalan.
	Asumsi bool
}

// Holidays adalah kalender libur yang diasumsikan untuk Okt 2025 - Mar 2026.
var Holidays = []Holiday{
	{Date: "2025-12-25", NameID: "Hari Raya Natal", NameEN: "Christmas Day"},
	{Date: "2025-12-26", NameID: "Cuti Bersama Natal", NameEN: "Christmas collective leave", Asumsi: true},
	{Date: "2026-01-01", NameID: "Tahun Baru Masehi", NameEN: "New Year's Day"},
	{Date: "2026-01-16", NameID: "Isra Mikraj 1447 H", NameEN: "Isra Mi'raj 1447 H", Asumsi: true},
	{Date: "2026-02-17", NameID: "Tahun Baru Imlek 2577", NameEN: "Lunar New Year 2577", Asumsi: true},
}

// Calendar memetakan indeks hari kerja ke tanggal dan sebaliknya.
type Calendar struct {
	days     []time.Time
	byISO    map[string]int
	holidays map[string]Holiday
}

// ParseISO membaca "YYYY-MM-DD" sebagai waktu UTC tengah malam.
func ParseISO(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.UTC)
}

// MustParseISO seperti ParseISO tetapi panik bila formatnya salah. Dipakai
// untuk konstanta di dalam kode, bukan untuk masukan pengguna.
func MustParseISO(s string) time.Time {
	t, err := ParseISO(s)
	if err != nil {
		panic(fmt.Sprintf("workcal: tanggal tidak sah %q: %v", s, err))
	}
	return t
}

// ISO memformat waktu sebagai "YYYY-MM-DD".
func ISO(t time.Time) string { return t.Format("2006-01-02") }

// IsWeekend melaporkan apakah t jatuh pada Sabtu atau Minggu.
func IsWeekend(t time.Time) bool {
	d := t.Weekday()
	return d == time.Saturday || d == time.Sunday
}

// New membangun kalender yang dimulai pada hari kerja pertama pada atau
// sesudah startISO, sepanjang horizon hari kerja.
func New(startISO string, horizon int) (*Calendar, error) {
	start, err := ParseISO(startISO)
	if err != nil {
		return nil, err
	}
	if horizon <= 0 {
		return nil, fmt.Errorf("workcal: horizon harus positif, dapat %d", horizon)
	}

	hol := make(map[string]Holiday, len(Holidays))
	for _, h := range Holidays {
		hol[h.Date] = h
	}

	c := &Calendar{
		byISO:    make(map[string]int, horizon),
		holidays: hol,
	}

	cur := start
	for len(c.days) < horizon {
		iso := ISO(cur)
		if _, isHoliday := hol[iso]; !IsWeekend(cur) && !isHoliday {
			c.byISO[iso] = len(c.days)
			c.days = append(c.days, cur)
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return c, nil
}

// MustNew seperti New tetapi panik bila gagal.
func MustNew(startISO string, horizon int) *Calendar {
	c, err := New(startISO, horizon)
	if err != nil {
		panic(err)
	}
	return c
}

// Len mengembalikan jumlah hari kerja yang dipetakan.
func (c *Calendar) Len() int { return len(c.days) }

// Start mengembalikan hari kerja pertama.
func (c *Calendar) Start() time.Time { return c.days[0] }

// Date mengembalikan tanggal untuk indeks hari kerja, di-clamp ke horizon.
func (c *Calendar) Date(index int) time.Time {
	if index < 0 {
		index = 0
	}
	if index >= len(c.days) {
		index = len(c.days) - 1
	}
	return c.days[index]
}

// ISOAt mengembalikan tanggal ISO untuk indeks hari kerja.
func (c *Calendar) ISOAt(index int) string { return ISO(c.Date(index)) }

// IndexOf mengembalikan indeks hari kerja untuk sebuah tanggal ISO. Tanggal
// non-kerja dibulatkan ke hari kerja berikutnya. Tanggal sebelum proyek mulai
// menghasilkan -1; sesudah horizon menghasilkan Len().
func (c *Calendar) IndexOf(iso string) int {
	if i, ok := c.byISO[iso]; ok {
		return i
	}
	t, err := ParseISO(iso)
	if err != nil {
		return -1
	}
	if t.Before(c.days[0]) {
		return -1
	}
	if t.After(c.days[len(c.days)-1]) {
		return len(c.days)
	}
	for probe, guard := t, 0; guard < 30; guard++ {
		if i, ok := c.byISO[ISO(probe)]; ok {
			return i
		}
		probe = probe.AddDate(0, 0, 1)
	}
	return len(c.days)
}

// FractionalIndexOf memberi posisi pecahan pada sumbu hari kerja. Dipakai saat
// tanggal data jatuh di akhir pekan: perhitungan Earned Value tetap butuh
// angka kontinu untuk interpolasi.
func (c *Calendar) FractionalIndexOf(iso string) float64 {
	t, err := ParseISO(iso)
	if err != nil {
		return 0
	}
	if !t.After(c.days[0]) {
		return 0
	}
	last := c.days[len(c.days)-1]
	if !t.Before(last) {
		return float64(len(c.days) - 1)
	}
	lo, hi := 0, len(c.days)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if !c.days[mid].After(t) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	next := lo + 1
	if next >= len(c.days) {
		return float64(lo)
	}
	span := c.days[next].Sub(c.days[lo]).Hours() / 24
	off := t.Sub(c.days[lo]).Hours() / 24
	if span <= 0 {
		return float64(lo)
	}
	frac := off / span
	if frac > 1 {
		frac = 1
	}
	return float64(lo) + frac
}

// HolidaysBetween mengembalikan hari libur yang jatuh di antara dua indeks
// hari kerja (inklusif), berguna untuk menjelaskan mengapa suatu fase molor
// di kalender meskipun jumlah hari kerjanya tetap.
func (c *Calendar) HolidaysBetween(fromIdx, toIdx int) []Holiday {
	if fromIdx < 0 {
		fromIdx = 0
	}
	if toIdx >= len(c.days) {
		toIdx = len(c.days) - 1
	}
	var out []Holiday
	from, to := c.days[fromIdx], c.days[toIdx]
	for _, h := range Holidays {
		t, err := ParseISO(h.Date)
		if err != nil {
			continue
		}
		if !t.Before(from) && !t.After(to) {
			out = append(out, h)
		}
	}
	return out
}

var monthsID = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
var monthsEN = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// MonthShort mengembalikan nama bulan pendek dalam bahasa lang ("id"/"en").
func MonthShort(m time.Month, lang string) string {
	if lang == "en" {
		return monthsEN[int(m)-1]
	}
	return monthsID[int(m)-1]
}

// FormatDate memformat tanggal ISO jadi "20 Okt 2025" / "20 Oct 2025".
func FormatDate(iso, lang string) string {
	t, err := ParseISO(iso)
	if err != nil {
		return iso
	}
	return fmt.Sprintf("%d %s %d", t.Day(), MonthShort(t.Month(), lang), t.Year())
}

// FormatDateShort memformat tanggal ISO jadi "20 Okt".
func FormatDateShort(iso, lang string) string {
	t, err := ParseISO(iso)
	if err != nil {
		return iso
	}
	return fmt.Sprintf("%d %s", t.Day(), MonthShort(t.Month(), lang))
}
