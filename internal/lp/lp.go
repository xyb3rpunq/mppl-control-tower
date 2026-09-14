// Package lp memecahkan program linear kecil dengan metode simpleks dua fase.
//
// Masalah yang dipecahkan berbentuk
//
//	minimalkan  c . x
//	dengan      A x <= b,  x >= 0
//
// b boleh negatif; baris seperti itu dibalik tandanya dan diberi variabel
// artifisial pada fase 1. Pemilihan pivot memakai aturan Bland (indeks
// terkecil), yang lebih lambat dari aturan Dantzig tetapi terbukti tidak
// pernah berputar pada masalah degeneratif - dan masalah crashing proyek
// sangat degeneratif, karena banyak batasan jalur yang aktif bersamaan.
//
// Ukuran yang dituju puluhan variabel dan ratusan batasan: tableau padat
// sudah lebih dari cukup, dan kodenya bisa diaudit baris demi baris.
package lp

import "math"

// Status adalah hasil pemecahan.
type Status int

// Status pemecahan.
const (
	Optimal Status = iota
	Infeasible
	Unbounded
)

func (s Status) String() string {
	switch s {
	case Optimal:
		return "optimal"
	case Infeasible:
		return "tidak layak"
	default:
		return "tak terbatas"
	}
}

const tol = 1e-9

// Solution adalah hasil satu program linear.
type Solution struct {
	Status    Status
	X         []float64
	Objective float64
	Pivots    int
}

// Minimize memecahkan min c.x dengan A x <= b dan x >= 0.
func Minimize(c []float64, A [][]float64, b []float64) Solution {
	m, n := len(A), len(c)
	// Kolom: n struktural, m slack, lalu artifisial seperlunya.
	var artRows []int
	for i := 0; i < m; i++ {
		if b[i] < -tol {
			artRows = append(artRows, i)
		}
	}
	nArt := len(artRows)
	cols := n + m + nArt
	rhs := cols
	T := make([][]float64, m+1)
	for i := range T {
		T[i] = make([]float64, cols+1)
	}
	basis := make([]int, m)
	art := 0
	for i := 0; i < m; i++ {
		sign := 1.0
		if b[i] < -tol {
			sign = -1
		}
		for j := 0; j < n; j++ {
			T[i][j] = sign * A[i][j]
		}
		T[i][n+i] = sign
		T[i][rhs] = sign * b[i]
		if sign < 0 {
			T[i][n+m+art] = 1
			basis[i] = n + m + art
			art++
		} else {
			basis[i] = n + i
		}
	}
	sol := Solution{}

	// Fase 1: minimalkan jumlah artifisial.
	if nArt > 0 {
		obj := T[m]
		for j := range obj {
			obj[j] = 0
		}
		for k := 0; k < nArt; k++ {
			obj[n+m+k] = 1
		}
		priceOut(T, basis)
		if !run(T, basis, cols, &sol.Pivots) {
			return Solution{Status: Unbounded}
		}
		if -T[m][rhs] > 1e-7 {
			return Solution{Status: Infeasible, Pivots: sol.Pivots}
		}
		// Keluarkan artifisial yang masih basis pada level nol.
		for i := 0; i < m; i++ {
			if basis[i] < n+m {
				continue
			}
			for j := 0; j < n+m; j++ {
				if math.Abs(T[i][j]) > 1e-7 {
					pivot(T, basis, i, j)
					sol.Pivots++
					break
				}
			}
		}
	}

	// Fase 2: biaya asli; kolom artifisial dilarang masuk.
	obj := T[m]
	for j := range obj {
		obj[j] = 0
	}
	copy(obj, c)
	priceOut(T, basis)
	if !run(T, basis, n+m, &sol.Pivots) {
		return Solution{Status: Unbounded, Pivots: sol.Pivots}
	}
	sol.Status = Optimal
	sol.X = make([]float64, n)
	for i, bj := range basis {
		if bj < n {
			sol.X[bj] = T[i][rhs]
		}
	}
	sol.Objective = -T[m][rhs]
	return sol
}

// priceOut menghapus komponen variabel basis dari baris tujuan.
func priceOut(T [][]float64, basis []int) {
	m := len(basis)
	obj := T[m]
	for i, bj := range basis {
		if f := obj[bj]; f != 0 {
			row := T[i]
			for j := range obj {
				obj[j] -= f * row[j]
			}
		}
	}
}

// run menjalankan iterasi simpleks dengan aturan Bland pada kolom < allowed.
// Mengembalikan false bila masalah tak terbatas.
func run(T [][]float64, basis []int, allowed int, pivots *int) bool {
	m := len(basis)
	rhs := len(T[0]) - 1
	for guard := 0; guard < 50_000; guard++ {
		enter := -1
		for j := 0; j < allowed; j++ {
			if T[m][j] < -tol {
				enter = j
				break
			}
		}
		if enter < 0 {
			return true
		}
		leave := -1
		best := math.Inf(1)
		for i := 0; i < m; i++ {
			a := T[i][enter]
			if a <= tol {
				continue
			}
			ratio := T[i][rhs] / a
			if ratio < best-tol || (math.Abs(ratio-best) <= tol && basis[i] < basis[leave]) {
				best, leave = ratio, i
			}
		}
		if leave < 0 {
			return false
		}
		pivot(T, basis, leave, enter)
		*pivots++
	}
	return true
}

func pivot(T [][]float64, basis []int, r, c int) {
	row := T[r]
	p := row[c]
	for j := range row {
		row[j] /= p
	}
	for i := range T {
		if i == r {
			continue
		}
		if f := T[i][c]; f != 0 {
			ri := T[i]
			for j := range ri {
				ri[j] -= f * row[j]
			}
		}
	}
	basis[r] = c
}
