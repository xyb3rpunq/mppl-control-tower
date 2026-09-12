package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Format angka mengikuti kaidah Indonesia (titik sebagai pemisah ribuan, koma
// sebagai pemisah desimal) untuk bahasa Indonesia, dan kaidah Inggris untuk
// bahasa Inggris. Salah tanda baca pada angka rupiah bukan perkara kosmetik -
// "Rp 1,338" dan "Rp 1.338" berbeda seribu kali lipat.

// Thousands menyisipkan pemisah ribuan pada bagian bulat sebuah angka.
func Thousands(n int64, sep string) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var out []string
	for len(s) > 3 {
		out = append([]string{s[len(s)-3:]}, out...)
		s = s[:len(s)-3]
	}
	out = append([]string{s}, out...)
	res := strings.Join(out, sep)
	if neg {
		res = "-" + res
	}
	return res
}

// Num memformat bilangan dengan sejumlah angka desimal sesuai bahasa.
func Num(v float64, decimals int, lang string) string {
	sepK, sepD := ".", ","
	if lang == "en" {
		sepK, sepD = ",", "."
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	pow := math.Pow(10, float64(decimals))
	rounded := math.Round(v*pow) / pow
	intPart := int64(math.Floor(rounded))
	out := Thousands(intPart, sepK)
	if decimals > 0 {
		frac := rounded - float64(intPart)
		fs := strconv.FormatFloat(frac, 'f', decimals, 64)
		out += sepD + fs[2:]
	}
	if neg {
		out = "-" + out
	}
	return out
}

// Rp memformat rupiah penuh, mis. "Rp 14.832.000".
func Rp(v float64, lang string) string {
	prefix := "Rp "
	if lang == "en" {
		prefix = "IDR "
	}
	if v < 0 {
		return "-" + prefix + Num(-v, 0, lang)
	}
	return prefix + Num(v, 0, lang)
}

// RpShort memformat rupiah dalam satuan ringkas: ribu, juta, miliar, triliun.
// Dipakai di kartu KPI dan label grafik yang ruangnya sempit.
func RpShort(v float64, lang string) string {
	prefix := "Rp "
	if lang == "en" {
		prefix = "IDR "
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	units := [][2]interface{}{
		{1e12, "triliun"}, {1e9, "miliar"}, {1e6, "juta"}, {1e3, "ribu"},
	}
	if lang == "en" {
		units = [][2]interface{}{
			{1e12, "trillion"}, {1e9, "billion"}, {1e6, "million"}, {1e3, "thousand"},
		}
	}
	for _, u := range units {
		div := u[0].(float64)
		if v >= div {
			return sign + prefix + Num(v/div, 2, lang) + " " + u[1].(string)
		}
	}
	return sign + prefix + Num(v, 0, lang)
}

// Pct memformat rasio 0..1 sebagai persentase.
func Pct(v float64, decimals int, lang string) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	return Num(v*100, decimals, lang) + "%"
}

// PctPoints memformat angka yang sudah dalam satuan persen.
func PctPoints(v float64, decimals int, lang string) string {
	return Num(v, decimals, lang) + "%"
}

// Index memformat indeks kinerja seperti SPI atau CPI dengan tiga desimal.
func Index(v float64, lang string) string { return Num(v, 3, lang) }

// Signed memberi tanda plus eksplisit pada angka positif - penting untuk
// varians, karena "SV 534.286" dan "SV -534.286" bermakna berlawanan dan mata
// gampang melewatkan tanda minus tunggal.
func Signed(v float64, lang string) string {
	s := Rp(math.Abs(v), lang)
	if v > 0 {
		return "+" + s
	}
	if v < 0 {
		return "-" + s
	}
	return s
}

// SignedNum memberi tanda plus eksplisit pada bilangan biasa.
func SignedNum(v float64, decimals int, lang string) string {
	s := Num(math.Abs(v), decimals, lang)
	if v > 0 {
		return "+" + s
	}
	if v < 0 {
		return "-" + s
	}
	return s
}

// Ratio memformat perbandingan seperti "47,8x".
func Ratio(v float64, lang string) string { return Num(v, 1, lang) + "x" }

// Ordinal mengubah bilangan menjadi label hari kerja.
func WorkdayLabel(d int, lang string) string {
	if lang == "en" {
		return fmt.Sprintf("day %d", d)
	}
	return fmt.Sprintf("hari ke-%d", d)
}

// Weeks mengubah hari kerja menjadi minggu dengan satu desimal.
func Weeks(days float64, lang string) string {
	return Num(days/5, 1, lang)
}
