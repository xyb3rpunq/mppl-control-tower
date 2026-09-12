package site

import "github.com/xyb3rpunq/mppl-control-tower/internal/render"

// Pembantu format untuk teks temuan. Temuan disusun sekali dan dipakai di
// kedua bahasa, jadi angkanya diformat dengan kaidah Indonesia - format angka
// bukan bagian yang berbeda antar bahasa di sini, karena rupiah tetap rupiah.

func fmtRp(v float64) string  { return render.Rp(v, "id") }
func fmtIdx(v float64) string { return render.Num(v, 3, "id") }
func fmtPct(v float64) string { return render.Pct(v, 1, "id") }
func fmtInt(v float64) string { return render.Num(v, 0, "id") }
func fmtNum(v float64) string { return render.Num(v, 2, "id") }
