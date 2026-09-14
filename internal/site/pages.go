package site

import (
	"github.com/xyb3rpunq/mppl-control-tower/internal/i18n"
	"github.com/xyb3rpunq/mppl-control-tower/internal/model"
)

// Page adalah satu rute pada situs.
type Page struct {
	Route    string // rute kanonis bahasa Indonesia, selalu diakhiri garis miring
	Template string
	NavKey   string // kunci i18n untuk label navigasi
	Icon     string
	Summary  model.Text
}

// Pages adalah seluruh rute situs, dalam urutan navigasi.
var Pages = []Page{
	{
		Route: "/", Template: "dashboard", NavKey: "nav.dashboard", Icon: "◉",
		Summary: model.Text{
			ID: "Delapan angka yang menentukan nasib proyek, plus temuan yang lahir langsung dari angka itu.",
			EN: "The eight numbers that decide a project's fate, plus findings derived straight from them.",
		},
	},
	{
		Route: "/piagam/", Template: "charter", NavKey: "nav.charter", Icon: "§",
		Summary: model.Text{
			ID: "Project Charter lengkap dan Work Breakdown Structure sampai 35 aktivitas, dengan pemeriksaan aturan 100%.",
			EN: "The full Project Charter and a Work Breakdown Structure down to 35 activities, with a 100%-rule check.",
		},
	},
	{
		Route: "/jadwal/", Template: "schedule", NavKey: "nav.schedule", Icon: "▤",
		Summary: model.Text{
			ID: "Gantt chart, diagram jaringan Activity-on-Node, dan tabel CPM lengkap dengan ES, EF, LS, LF, serta float.",
			EN: "Gantt chart, Activity-on-Node network diagram, and a full CPM table with ES, EF, LS, LF, and float.",
		},
	},
	{
		Route: "/optimasi/", Template: "optimize", NavKey: "nav.optimize", Icon: "⇄",
		Summary: model.Text{
			ID: "Jadwal yang benar-benar bisa dijalankan dengan kapasitas nyata, plus harga mempercepatnya: levelling sumber daya, crashing, dan fast-tracking.",
			EN: "The schedule that can actually be delivered with real capacity, plus the price of speeding it up: resource levelling, crashing, and fast-tracking.",
		},
	},
	{
		Route: "/pert/", Template: "pert", NavKey: "nav.risksched", Icon: "∿",
		Summary: model.Text{
			ID: "Estimasi tiga titik, peluang selesai tepat waktu menurut PERT, dan sepuluh ribu iterasi Monte Carlo.",
			EN: "Three-point estimates, PERT on-time probability, and ten thousand Monte Carlo iterations.",
		},
	},
	{
		Route: "/simulasi-terpadu/", Template: "integrated", NavKey: "nav.integrated", Icon: "⧉",
		Summary: model.Text{
			ID: "Monte Carlo yang menghitung korelasi peran, kejadian risiko, dan kapasitas sekaligus - lalu menjawab peluang tepat waktu DAN tepat anggaran lewat Joint Confidence Level.",
			EN: "A Monte Carlo that counts role correlation, risk events, and capacity together - then answers the odds of on time AND on budget through a Joint Confidence Level.",
		},
	},
	{
		Route: "/biaya/", Template: "cost", NavKey: "nav.cost", Icon: "₹",
		Summary: model.Text{
			ID: "Earned Value lengkap: PV, EV, AC, SPI, CPI, tiga varian EAC, TCPI, dan Earned Schedule.",
			EN: "Full Earned Value: PV, EV, AC, SPI, CPI, three EAC variants, TCPI, and Earned Schedule.",
		},
	},
	{
		Route: "/risiko/", Template: "risk", NavKey: "nav.risk", Icon: "⚠",
		Summary: model.Text{
			ID: "Risk register dua belas entri dengan EMV, peta panas 5x5 sebelum dan sesudah mitigasi, serta uji kecukupan cadangan.",
			EN: "A twelve-entry risk register with EMV, 5x5 heat maps before and after mitigation, and a reserve adequacy test.",
		},
	},
	{
		Route: "/organisasi/", Template: "org", NavKey: "nav.org", Icon: "⬡",
		Summary: model.Text{
			ID: "Bagan organisasi, matriks RACI tervalidasi, histogram pembebanan sumber daya, dan grid kuasa-kepentingan.",
			EN: "Organisation chart, validated RACI matrix, resource loading histogram, and power-interest grid.",
		},
	},
	{
		Route: "/kualitas/", Template: "quality", NavKey: "nav.quality", Icon: "◈",
		Summary: model.Text{
			ID: "Seven Basic Tools of Quality yang benar-benar dihitung: peta kendali beraturan Nelson, Pareto berbobot, fishbone, dan biaya kualitas.",
			EN: "The Seven Basic Tools of Quality, actually computed: a Nelson-rule control chart, weighted Pareto, fishbone, and cost of quality.",
		},
	},
	{
		Route: "/coretax/", Template: "coretax", NavKey: "nav.coretax", Icon: "◎",
		Summary: model.Text{
			ID: "Bedah kasus Coretax DJP dengan mesin yang sama: fakta bersumber, metrik turunan, dan perbandingan strategi transisi.",
			EN: "The Coretax DJP case study through the same engine: sourced facts, derived metrics, and a transition strategy comparison.",
		},
	},
	{
		Route: "/rumus/", Template: "formulas", NavKey: "nav.formulas", Icon: "Σ",
		Summary: model.Text{
			ID: "Setiap rumus yang dijalankan aplikasi ini, lengkap dengan arti simbol, cara membaca, jebakan umum, dan contoh hitung dari data hidup.",
			EN: "Every formula this application runs, with symbol meanings, how to read it, common traps, and a worked example from live data.",
		},
	},
	{
		Route: "/materi/", Template: "material", NavKey: "nav.material", Icon: "▦",
		Summary: model.Text{
			ID: "Sembilan area pengetahuan, empat tahap siklus hidup, dan peta dari tiap materi kuliah ke fitur yang membuktikannya.",
			EN: "Nine knowledge areas, four lifecycle phases, and a map from each piece of course material to the feature that proves it.",
		},
	},
	{
		Route: "/metode/", Template: "method", NavKey: "nav.method", Icon: "⌘",
		Summary: model.Text{
			ID: "Cara angka-angka di situs ini dihitung, dari mana datanya, apa yang diasumsikan, dan apa yang belum tertutup.",
			EN: "How the numbers here are computed, where the data comes from, what is assumed, and what remains uncovered.",
		},
	},
}

// PathFor mengembalikan jalur berkas keluaran untuk sebuah rute dan bahasa.
// Bahasa Indonesia menempati akar situs; bahasa Inggris di bawah /en/.
func PathFor(route, lang string) string {
	if lang == "en" {
		if route == "/" {
			return "/en/"
		}
		return "/en" + route
	}
	return route
}

// NavLabel mengembalikan label navigasi sebuah halaman.
func (p Page) NavLabel(lang string) string { return i18n.T(lang, p.NavKey) }
