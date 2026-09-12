//go:build js && wasm

// Command wasm mengekspos mesin hitung MPPL ke peramban lewat WebAssembly.
//
// Inilah alasan seluruh perhitungan ditulis di paket internal yang murni:
// paket yang sama dipakai generator situs statis di sisi server DAN dipakai
// di sini di dalam peramban. Tidak ada rumus yang ditulis dua kali, sehingga
// tidak mungkin ada versi JavaScript yang diam-diam berbeda dari versi Go.
//
// Bangun dengan:
//
//	GOOS=js GOARCH=wasm go build -o web/static/js/mppl.wasm ./cmd/wasm
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/xyb3rpunq/mppl-control-tower/internal/evm"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
	"github.com/xyb3rpunq/mppl-control-tower/internal/schedule"
	"github.com/xyb3rpunq/mppl-control-tower/internal/simulate"
	"github.com/xyb3rpunq/mppl-control-tower/internal/workcal"
)

var (
	cal    *workcal.Calendar
	plan   schedule.Result
	engine *evm.Engine
)

func main() {
	cal = workcal.MustNew(model.ProjectCharter.StartDate, 400)
	plan = schedule.MustCompute(model.Activities, schedule.Options{})
	engine = evm.New(model.Activities, plan, model.RateCard)

	js.Global().Set("mpplRecompute", js.FuncOf(recompute))
	js.Global().Set("mpplSimulate", js.FuncOf(runSimulation))
	js.Global().Set("mpplBounds", js.FuncOf(bounds))

	// Beri tahu halaman bahwa mesinnya siap; tanpa ini kendali interaktif
	// tetap tersembunyi dan halaman berperilaku seperti halaman statis biasa.
	if ready := js.Global().Get("mpplReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}

	// Blokir selamanya: program WebAssembly yang keluar dari main akan
	// melepaskan seluruh fungsi yang baru saja didaftarkan.
	select {}
}

// recompute mengembalikan seluruh metrik Earned Value pada tanggal data baru.
func recompute(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errPayload("tanggal data tidak diberikan")
	}
	iso := args[0].String()
	day := cal.FractionalIndexOf(iso)
	s := engine.At(day)

	out := map[string]any{
		"ok":         true,
		"statusDate": iso,
		"day":        day,
		"planDays":   plan.Duration,
		"pv":         s.PV,
		"ev":         s.EV,
		"ac":         s.AC,
		"bac":        s.BAC,
		"sv":         s.SV,
		"cv":         s.CV,
		"spi":        s.SPI,
		"cpi":        s.CPI,
		"es":         s.ES,
		"svt":        s.SVt,
		"spit":       s.SPIt,
		"eacOpt":     s.EACOptimistic,
		"eac":        s.EACTypical,
		"eacPes":     s.EACPessimistic,
		"etc":        s.ETC,
		"vac":        s.VAC,
		"tcpi":       s.TCPI,
		"complete":   s.PercentComplete,
		"spent":      s.PercentSpent,
		"elapsed":    s.PercentElapsed,
		"authorised": model.TotalAuthorised,
		"overCap":    s.EACTypical > model.TotalAuthorised,
	}

	// Kurva-S ikut dikirim supaya halaman bisa menggambar ulang grafiknya
	// tanpa permintaan kedua.
	curves := engine.BuildCurves(day)
	out["curves"] = map[string]any{
		"days":     curves.Days,
		"pv":       curves.PV,
		"ev":       curves.EV,
		"ac":       curves.AC,
		"cutoff":   curves.CutoffIndex,
		"forecast": nanToNull(curves.Forecast),
	}
	return toJS(out)
}

// runSimulation menjalankan Monte Carlo dengan jumlah iterasi dan benih yang
// ditentukan pengguna, lalu mengembalikan ringkasannya.
func runSimulation(_ js.Value, args []js.Value) any {
	cfg := simulate.Defaults()
	if len(args) > 0 && args[0].Type() == js.TypeNumber {
		cfg.Iterations = args[0].Int()
	}
	if len(args) > 1 && args[1].Type() == js.TypeNumber {
		cfg.Seed = uint32(args[1].Int())
	}
	if len(args) > 2 && args[2].Type() == js.TypeString {
		cfg.Distribution = args[2].String()
	}
	if cfg.Iterations < 100 {
		cfg.Iterations = 100
	}
	if cfg.Iterations > 200_000 {
		cfg.Iterations = 200_000
	}

	res, err := simulate.Run(model.Activities, plan, cfg)
	if err != nil {
		return errPayload(err.Error())
	}

	bins := make([]map[string]any, 0, len(res.Histogram))
	for _, b := range res.Histogram {
		bins = append(bins, map[string]any{
			"from": b.From, "to": b.To, "count": b.Count, "cum": b.Cumulative,
		})
	}
	sens := make([]map[string]any, 0, len(res.Sensitivity))
	for i, s := range res.Sensitivity {
		if i >= 14 {
			break
		}
		sens = append(sens, map[string]any{
			"id": s.ID, "name": s.Name.ID, "r": s.Correlation, "critical": s.CriticalRate,
		})
	}

	return toJS(map[string]any{
		"ok":            true,
		"iterations":    res.Iterations,
		"distribution":  cfg.Distribution,
		"seed":          cfg.Seed,
		"mean":          res.Mean,
		"stdDev":        res.StdDev,
		"min":           res.Min,
		"max":           res.Max,
		"p10":           res.P10,
		"p50":           res.P50,
		"p80":           res.P80,
		"p90":           res.P90,
		"p95":           res.P95,
		"deterministic": res.Deterministic,
		"onTime":        res.OnTimeProb,
		"p50Date":       cal.ISOAt(int(res.P50) - 1),
		"p80Date":       cal.ISOAt(int(res.P80) - 1),
		"p90Date":       cal.ISOAt(int(res.P90) - 1),
		"histogram":     bins,
		"sensitivity":   sens,
	})
}

// bounds memberi tahu halaman rentang tanggal data yang sah.
//
// Batas atasnya BUKAN akhir proyek melainkan hari terakhir yang masih punya
// data realisasi. Menggeser tanggal data melewati batas itu akan membuat EV
// berhenti tumbuh sementara PV terus naik, sehingga SPI anjlok bukan karena
// proyeknya melambat melainkan karena datanya habis. Membiarkan pengguna
// melakukannya tanpa peringatan sama saja menyajikan artefak sebagai temuan.
func bounds(_ js.Value, _ []js.Value) any {
	last := 0
	for _, a := range model.Activities {
		if !a.Actual.Started {
			continue
		}
		if end := a.Actual.Start + a.Actual.Duration; end > last {
			last = end
		}
	}
	if last >= cal.Len() {
		last = cal.Len() - 1
	}
	return toJS(map[string]any{
		"ok":             true,
		"start":          cal.ISOAt(0),
		"finish":         cal.ISOAt(plan.Duration - 1),
		"actualsThrough": cal.ISOAt(last),
		"actualsDay":     last,
		"planDays":       plan.Duration,
		"default":        model.DefaultStatusDate,
	})
}

// nanToNull mengganti NaN dengan null agar bisa diserialkan ke JSON - proyeksi
// biaya sebelum tanggal data sengaja bernilai NaN supaya garisnya terputus.
func nanToNull(v []float64) []any {
	out := make([]any, len(v))
	for i, x := range v {
		if x != x { // NaN
			out[i] = nil
			continue
		}
		out[i] = x
	}
	return out
}

func toJS(v map[string]any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return errPayload(err.Error())
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return errPayload(err.Error())
	}
	return js.ValueOf(generic)
}

func errPayload(msg string) any {
	return js.ValueOf(map[string]any{"ok": false, "error": msg})
}
