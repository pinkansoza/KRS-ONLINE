// Package handler: urusan HTTP. Membaca r, memanggil service, lalu menulis
// response / redirect / render template. Tidak ada SQL di sini.
package handler

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"krs-online/internal/middleware"
	"krs-online/internal/model"
	"krs-online/internal/service"
)

var bulan = [...]string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// tanggal memformat waktu: "29 Okt 2026, 20:16" (layout Go: 02 Jan 2006 15:04).
func tanggal(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return fmt.Sprintf("%d %s %d, %s", t.Day(), bulan[t.Month()], t.Year(), t.Format("15:04"))
}

func tanggalPendek(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return fmt.Sprintf("%d %s %d", t.Day(), bulan[t.Month()], t.Year())
}

// tglInput untuk <input type="datetime-local">.
func tglInput(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04")
}

// koma: 3.5 -> "3,50" (format Indonesia).
func koma(f float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", f), ".", ",", 1)
}

func jadwal(s []model.Schedule) string {
	if len(s) == 0 {
		return "belum dijadwalkan"
	}
	parts := make([]string, len(s))
	for i, j := range s {
		parts[i] = fmt.Sprintf("%s %s–%s", service.NamaHari(j.Day), j.Start, j.End)
		if j.Room != "" {
			parts[i] += " · " + j.Room
		}
	}
	return strings.Join(parts, "; ")
}

// badge memilih kelas CSS untuk status (warna mengikuti keterangan wireframe).
func badge(status string) string {
	switch status {
	case "OK", "AKTIF", "DISETUJUI", "PROMOTED", "DIPROMOSIKAN", "APPROVE", "APPROVE_ADD", "APPROVE_DROP", "BERJALAN":
		return "hijau"
	case "PRASYARAT", "BENTROK", "SKS", "DIBATALKAN", "REJECT", "REJECT_ADD", "REJECT_DROP", "DILEWATI", "SKIPPED", "NONAKTIF":
		return "merah"
	case "PENUH", "MENUNGGU", "MENUNGGU_TAMBAH", "MENUNGGU_BATAL", "REQUEST_ADD", "REQUEST_DROP", "PERUBAHAN":
		return "kuning"
	case "DIAJUKAN", "DI_KRS", "SUBMIT", "CUTI":
		return "biru"
	}
	return "abu"
}

var labelStatus = map[string]string{
	"OK": "Dapat diambil", "PENUH": "Penuh · daftar tunggu", "LULUS": "Sudah lulus",
	"PRASYARAT": "Prasyarat belum terpenuhi", "BENTROK": "Bentrok jadwal", "SKS": "Melebihi batas SKS",
	"DI_KRS": "Sudah di KRS", "MENUNGGU": "Menunggu", "MENUNGGU_TAMBAH": "Menunggu tambah",
	"MENUNGGU_BATAL": "Menunggu batal", "AKTIF": "Aktif", "DIBATALKAN": "Dibatalkan",
	"DRAFT": "Draft", "DIAJUKAN": "Diajukan", "DISETUJUI": "Disetujui", "": "Belum mengisi",
	"DIPROMOSIKAN": "Dipromosikan", "DILEWATI": "Dilewati", "KELUAR": "Keluar",
	"SUBMIT": "Diajukan", "APPROVE": "Disetujui", "REJECT": "Ditolak / revisi",
	"REQUEST_ADD": "Minta tambah", "REQUEST_DROP": "Minta batal", "APPROVE_ADD": "Tambah disetujui",
	"REJECT_ADD": "Tambah ditolak", "APPROVE_DROP": "Batal disetujui", "REJECT_DROP": "Batal ditolak",
	"PROMOTED": "Dipromosikan", "SKIPPED": "Dilewati",
}

func label(s string) string {
	if l, ok := labelStatus[s]; ok {
		return l
	}
	return s
}

func addOne(n int) int { return n + 1 }

func persen(a, b int) int {
	if b <= 0 {
		return 0
	}
	p := (a*100/b + 5) / 10 * 10 // dibulatkan ke kelipatan 10 untuk kelas CSS .p0 ... .p100
	if p > 100 {
		p = 100
	}
	return p
}

var funcs = template.FuncMap{
	"add_one": addOne, "tanggal": tanggal, "tanggalPendek": tanggalPendek, "tglInput": tglInput,
	"koma": koma, "jadwal": jadwal, "badge": badge, "label": label, "hari": service.NamaHari,
	"persen": persen, "lower": strings.ToLower,
	"kali":       func(a int, b float64) float64 { return float64(a) * b },
	"status_mhs": func() []string { return []string{"AKTIF", "CUTI", "LULUS", "NONAKTIF"} },
}

// Halaman = data umum untuk layout; isi khusus halaman ada di Data.
type Halaman struct {
	Title    string
	Menu     string
	User     model.User
	Semester string
	Flash    string
	Galat    string
	Errors   map[string]string
	Data     any
}

type Renderer struct {
	tpl      map[string]*template.Template
	Semester func(ctx context.Context) string // label semester aktif untuk header
}

// kerangka = file yang selalu ikut di-parse bersama setiap halaman.
var kerangka = map[string]bool{"layout.html": true, "header.html": true, "sidebar.html": true, "footer.html": true}

// NewRenderer mem-parse semua halaman sekali saat server menyala (template.Must:
// kalau ada salah tulis template, program langsung berhenti).
// Halaman = setiap file .html di folder templates (dan subfolder-nya) selain
// kerangka, jadi halaman baru di tahap berikutnya cukup ditambahkan filenya.
func NewRenderer(dir string) *Renderer {
	rd := &Renderer{tpl: make(map[string]*template.Template)}
	atas, _ := filepath.Glob(filepath.Join(dir, "*.html"))
	sub, _ := filepath.Glob(filepath.Join(dir, "*", "*.html"))
	for _, path := range append(atas, sub...) {
		file, _ := filepath.Rel(dir, path)
		file = filepath.ToSlash(file) // kunci selalu "mhs/katalog.html", juga di Windows
		if kerangka[file] {
			continue
		}
		rd.tpl[file] = template.Must(template.New("layout").Funcs(funcs).ParseFiles(
			filepath.Join(dir, "layout.html"),
			filepath.Join(dir, "header.html"),
			filepath.Join(dir, "sidebar.html"),
			filepath.Join(dir, "footer.html"),
			path,
		))
	}
	return rd
}

// Tampil me-render ke buffer dulu supaya error template tidak menghasilkan
// halaman setengah jadi. Urutan: header -> status -> body.
func (rd *Renderer) Tampil(w http.ResponseWriter, r *http.Request, status int, file string, h Halaman) {
	if u, ok := middleware.UserDari(r.Context()); ok {
		h.User = u
		if rd.Semester != nil {
			h.Semester = rd.Semester(r.Context())
		}
	}
	if h.Flash == "" {
		h.Flash = pesanFlash[r.URL.Query().Get("pesan")]
	}
	t, ok := rd.tpl[file]
	if !ok {
		log.Printf("template %s belum didaftarkan", file)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", h); err != nil {
		log.Printf("render %s: %v", file, err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

// pesanFlash: kode dari query string diterjemahkan lewat map, bukan teks
// bebas dari URL (materi CRUD bagian 2).
var pesanFlash = map[string]string{
	"dibuat":     "Data berhasil disimpan.",
	"diubah":     "Perubahan berhasil disimpan.",
	"dihapus":    "Data berhasil dihapus.",
	"ditambah":   "Kelas berhasil ditambahkan ke KRS.",
	"diminta":    "Permintaan tambah kelas dikirim ke dosen wali. Kursi sudah dipesan untuk Anda.",
	"dibatalkan": "Item KRS berhasil diperbarui.",
	"diajukan":   "KRS berhasil diajukan ke dosen wali.",
	"tunggu":     "Anda masuk daftar tunggu. Posisi antrean bisa dilihat di menu Daftar tunggu.",
	"keluar":     "Anda keluar dari daftar tunggu.",
	"disetujui":  "KRS disetujui dan sekarang terkunci.",
	"ditolak":    "KRS dikembalikan ke mahasiswa untuk direvisi.",
	"diputuskan": "Keputusan permintaan perubahan disimpan.",
	"password":   "Password berhasil diganti. Silakan masuk kembali.",
	"dibaca":     "Semua notifikasi ditandai sudah dibaca.",
	"aktif":      "Semester aktif berhasil diganti.",
	"keluar_app": "Anda sudah keluar.",
}
