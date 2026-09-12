// Package i18n menyediakan kamus teks antarmuka dwibahasa.
//
// Isi substantif proyek (nama aktivitas, risiko, peran) sudah dwibahasa lewat
// model.Text, jadi kamus ini hanya memuat teks kerangka: judul halaman, label
// tabel, dan kalimat penjelas. Pemisahan itu disengaja - isi proyek berubah
// ketika proyeknya berubah, sedangkan kerangka berubah ketika situsnya
// berubah, dan keduanya sebaiknya tidak saling mengunci.
package i18n

import "sort"

// Langs adalah bahasa yang didukung. Bahasa Indonesia adalah bahasa sumber.
var Langs = []string{"id", "en"}

// Dict memetakan kunci ke pasangan terjemahan.
type Dict map[string][2]string // [0] = id, [1] = en

var dict = Dict{
	// --- kerangka situs -----------------------------------------------
	"site.name":       {"Control Tower MPPL", "MPPL Control Tower"},
	"site.tagline":    {"Ruang kendali kuantitatif untuk proyek perangkat lunak", "A quantitative control room for software projects"},
	"site.skip":       {"Lompat ke konten utama", "Skip to main content"},
	"site.menu":       {"Menu", "Menu"},
	"site.lang":       {"Bahasa", "Language"},
	"site.theme":      {"Ganti tema", "Toggle theme"},
	"site.sourceCode": {"Kode sumber", "Source code"},
	"site.builtWith":  {"Dibangun dengan Go - mesin yang sama berjalan di server dan di peramban lewat WebAssembly", "Built with Go - the same engine runs on the server and in the browser via WebAssembly"},
	"site.dataDate":   {"Tanggal data", "Data date"},
	"site.updated":    {"Diperbarui", "Updated"},
	"site.print":      {"Cetak / simpan PDF", "Print / save as PDF"},
	"site.csv":        {"Unduh CSV", "Download CSV"},
	"site.offline":    {"Situs ini bekerja luring setelah kunjungan pertama", "This site works offline after the first visit"},

	// --- navigasi -----------------------------------------------------
	"nav.dashboard": {"Ruang Kendali", "Control Room"},
	"nav.charter":   {"Piagam & Lingkup", "Charter & Scope"},
	"nav.schedule":  {"Jadwal & Jalur Kritis", "Schedule & Critical Path"},
	"nav.risksched": {"PERT & Monte Carlo", "PERT & Monte Carlo"},
	"nav.cost":      {"Biaya & Earned Value", "Cost & Earned Value"},
	"nav.risk":      {"Manajemen Risiko", "Risk Management"},
	"nav.org":       {"Organisasi & Sumber Daya", "Organisation & Resources"},
	"nav.quality":   {"Manajemen Mutu", "Quality Management"},
	"nav.coretax":   {"Bedah Kasus: Coretax", "Case Study: Coretax"},
	"nav.formulas":  {"Referensi Rumus", "Formula Reference"},
	"nav.material":  {"Materi & Area Pengetahuan", "Course Material & Knowledge Areas"},
	"nav.method":    {"Metode & Sumber", "Method & Sources"},

	// --- istilah umum -------------------------------------------------
	"t.activity":    {"Aktivitas", "Activity"},
	"t.code":        {"Kode", "Code"},
	"t.name":        {"Nama", "Name"},
	"t.duration":    {"Durasi", "Duration"},
	"t.days":        {"hari kerja", "working days"},
	"t.day":         {"hari", "day"},
	"t.weeks":       {"minggu", "weeks"},
	"t.start":       {"Mulai", "Start"},
	"t.finish":      {"Selesai", "Finish"},
	"t.budget":      {"Anggaran", "Budget"},
	"t.actual":      {"Aktual", "Actual"},
	"t.plan":        {"Rencana", "Plan"},
	"t.variance":    {"Varians", "Variance"},
	"t.total":       {"Total", "Total"},
	"t.status":      {"Status", "Status"},
	"t.owner":       {"Pemilik", "Owner"},
	"t.category":    {"Kategori", "Category"},
	"t.target":      {"Target", "Target"},
	"t.phase":       {"Fase", "Phase"},
	"t.role":        {"Peran", "Role"},
	"t.critical":    {"Kritis", "Critical"},
	"t.float":       {"Float", "Float"},
	"t.totalFloat":  {"Total float", "Total float"},
	"t.freeFloat":   {"Free float", "Free float"},
	"t.milestone":   {"Milestone", "Milestone"},
	"t.probability": {"Peluang", "Probability"},
	"t.impact":      {"Dampak", "Impact"},
	"t.score":       {"Skor", "Score"},
	"t.response":    {"Respons", "Response"},
	"t.source":      {"Sumber", "Source"},
	"t.formula":     {"Rumus", "Formula"},
	"t.example":     {"Contoh dari proyek ini", "Worked example from this project"},
	"t.meaning":     {"Arti", "Meaning"},
	"t.reading":     {"Cara membaca", "How to read it"},
	"t.finding":     {"Temuan", "Finding"},
	"t.recommend":   {"Rekomendasi", "Recommendation"},
	"t.evidence":    {"Bukti", "Evidence"},
	"t.verified":    {"Terverifikasi", "Verified"},
	"t.derived":     {"Turunan", "Derived"},
	"t.scenario":    {"Skenario", "Scenario"},
	"t.assumption":  {"Asumsi", "Assumption"},
	"t.notStarted":  {"Belum mulai", "Not started"},
	"t.inProgress":  {"Berjalan", "In progress"},
	"t.done":        {"Selesai", "Complete"},
	"t.late":        {"Telat mulai", "Late start"},

	// --- status aktivitas (dari evm.ActivityRow.Status) -----------------
	"as.selesai":  {"selesai", "complete"},
	"as.berjalan": {"berjalan", "in progress"},
	"as.telat":    {"telat mulai", "late start"},
	"as.belum":    {"belum mulai", "not started"},

	// --- status risiko (dari model.Risk.Status) -------------------------
	"st.terbuka":   {"terbuka", "open"},
	"st.terpantau": {"terpantau", "monitored"},
	"st.terjadi":   {"terjadi", "materialised"},
	"st.tertutup":  {"tertutup", "closed"},

	// --- keparahan temuan (dari site.Finding.Severity) ------------------
	"sv.kritis": {"kritis", "critical"},
	"sv.tinggi": {"tinggi", "high"},
	"sv.sedang": {"sedang", "moderate"},
	"sv.baik":   {"baik", "good"},

	// --- keparahan risiko (dari risk.Severity) --------------------------
	"rs.rendah":  {"rendah", "low"},
	"rs.sedang":  {"sedang", "moderate"},
	"rs.tinggi":  {"tinggi", "high"},
	"rs.ekstrem": {"ekstrem", "extreme"},

	// --- strategi respons risiko (dari model.Response) ------------------
	"rp.avoid":    {"hindari", "avoid"},
	"rp.mitigate": {"kurangi", "mitigate"},
	"rp.transfer": {"alihkan", "transfer"},
	"rp.accept":   {"terima", "accept"},

	// --- jenis tonggak studi kasus (dari model.CoretaxMilestone.Kind) ---
	"ck.regulasi":    {"regulasi", "regulation"},
	"ck.pengadaan":   {"pengadaan", "procurement"},
	"ck.pelaksanaan": {"pelaksanaan", "delivery"},
	"ck.cutover":     {"cutover", "cutover"},
	"ck.dampak":      {"dampak", "impact"},
	"ck.remediasi":   {"remediasi", "remediation"},

	// --- status kesehatan ---------------------------------------------
	"h.good":   {"Sehat", "Healthy"},
	"h.watch":  {"Perlu diawasi", "Watch"},
	"h.alert":  {"Perlu tindakan", "Action needed"},
	"h.severe": {"Kritis", "Critical"},
}

// T mengambil teks untuk kunci dan bahasa. Kunci yang tidak dikenal
// dikembalikan apa adanya supaya kesalahan langsung terlihat di halaman,
// bukan diam-diam berubah jadi string kosong.
func T(lang, key string) string {
	v, ok := dict[key]
	if !ok {
		return "!" + key
	}
	if lang == "en" {
		return v[1]
	}
	return v[0]
}

// Keys mengembalikan seluruh kunci terurut - dipakai uji kelengkapan kamus.
func Keys() []string {
	out := make([]string, 0, len(dict))
	for k := range dict {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Lookup mengembalikan pasangan terjemahan mentah untuk sebuah kunci.
func Lookup(key string) ([2]string, bool) {
	v, ok := dict[key]
	return v, ok
}

// OtherLang mengembalikan bahasa pasangannya.
func OtherLang(lang string) string {
	if lang == "en" {
		return "id"
	}
	return "en"
}
