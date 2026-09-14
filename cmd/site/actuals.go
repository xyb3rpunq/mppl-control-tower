package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

// Realisasi proyek bisa diperbarui tanpa menyentuh kode: situs mengekspor
// data/realisasi.csv, tim mengisinya, lalu situs dibangun ulang dengan
// -realisasi. Kolomnya persis sama dengan model.Actual, sehingga ekspor lalu
// impor tanpa perubahan menghasilkan analisis yang identik.
//
//	id            kode aktivitas
//	dimulai       true bila aktivitas sudah dimulai
//	mulai         tanggal mulai aktual (ISO, hari kerja)
//	durasi_aktual hari kerja; untuk aktivitas yang belum selesai, perkiraan
//	              durasi totalnya - aktivitas dianggap selesai bila mulai +
//	              durasi tidak melewati tanggal data
//	biaya_aktual  rupiah untuk seluruh durasi aktual
const actualsHeader = "id,dimulai,mulai,durasi_aktual,biaya_aktual"

// actualsCSV menulis realisasi seluruh aktivitas.
func actualsCSV(acts []model.Activity, cal *workcal.Calendar) string {
	var sb strings.Builder
	sb.WriteString(actualsHeader + "\n")
	for _, a := range acts {
		act := a.Actual
		if !act.Started {
			sb.WriteString(a.ID + ",false,,,\n")
			continue
		}
		sb.WriteString(fmt.Sprintf("%s,true,%s,%d,%s\n", a.ID, cal.ISOAt(act.Start), act.Duration, strconv.FormatFloat(act.Cost, 'f', -1, 64)))
	}
	return sb.String()
}

// applyActualsCSV membaca realisasi dan menimpanya ke acts. Berkas harus
// memuat setiap aktivitas tepat sekali; baris yang tidak sah menghentikan
// impor tanpa mengubah apa pun, supaya analisis tidak diam-diam memakai
// campuran data lama dan baru.
func applyActualsCSV(r io.Reader, acts []model.Activity, cal *workcal.Calendar) error {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return fmt.Errorf("realisasi: %w", err)
	}
	if len(rows) == 0 || strings.Join(rows[0], ",") != actualsHeader {
		return fmt.Errorf("realisasi: kepala kolom harus %q", actualsHeader)
	}
	idx := make(map[string]int, len(acts))
	for i, a := range acts {
		idx[a.ID] = i
	}
	next := make([]model.Actual, len(acts))
	seen := make(map[string]bool, len(acts))
	for n, row := range rows[1:] {
		line := n + 2
		if len(row) != 5 {
			return fmt.Errorf("realisasi baris %d: butuh 5 kolom, ada %d", line, len(row))
		}
		id := strings.TrimSpace(row[0])
		i, ok := idx[id]
		if !ok {
			return fmt.Errorf("realisasi baris %d: aktivitas %q tidak dikenal", line, id)
		}
		if seen[id] {
			return fmt.Errorf("realisasi baris %d: aktivitas %s muncul dua kali", line, id)
		}
		seen[id] = true
		started, err := strconv.ParseBool(strings.TrimSpace(row[1]))
		if err != nil {
			return fmt.Errorf("realisasi baris %d: dimulai harus true atau false", line)
		}
		if !started {
			continue
		}
		iso := strings.TrimSpace(row[2])
		day := cal.IndexOf(iso)
		if day < 0 || cal.ISOAt(day) != iso {
			return fmt.Errorf("realisasi baris %d: %q bukan hari kerja pada kalender proyek", line, iso)
		}
		dur, err := strconv.Atoi(strings.TrimSpace(row[3]))
		if err != nil || dur < 0 || (dur == 0 && !acts[i].Milestone) {
			return fmt.Errorf("realisasi baris %d: durasi_aktual %q tidak sah untuk %s", line, row[3], id)
		}
		cost, err := strconv.ParseFloat(strings.TrimSpace(row[4]), 64)
		if err != nil || cost < 0 {
			return fmt.Errorf("realisasi baris %d: biaya_aktual %q tidak sah", line, row[4])
		}
		next[i] = model.Actual{Started: true, Start: day, Duration: dur, Cost: cost}
	}
	var missing []string
	for _, a := range acts {
		if !seen[a.ID] {
			missing = append(missing, a.ID)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("realisasi: aktivitas tanpa baris: %s", strings.Join(missing, ", "))
	}
	for i := range acts {
		acts[i].Actual = next[i]
	}
	return nil
}

// loadActuals menerapkan berkas realisasi ke model sebelum analisis.
func loadActuals(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cal, err := workcal.New(model.ProjectCharter.StartDate, 400)
	if err != nil {
		return err
	}
	return applyActualsCSV(f, model.Activities, cal)
}
