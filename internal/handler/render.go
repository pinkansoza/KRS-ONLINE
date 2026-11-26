package handler

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"krs-online/internal/model"
)

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
	tpl map[string]*template.Template
}

// kerangka = file yang selalu ikut di-parse bersama setiap halaman.
var kerangka = map[string]bool{"layout.html": true, "header.html": true, "sidebar.html": true, "footer.html": true}

// NewRenderer mem-parse semua halaman sekali saat server menyala
// (template.Must: kalau ada salah tulis template, program langsung berhenti).
func NewRenderer(dir string) *Renderer {
	rd := &Renderer{tpl: make(map[string]*template.Template)}
	atas, _ := filepath.Glob(filepath.Join(dir, "*.html"))
	sub, _ := filepath.Glob(filepath.Join(dir, "*", "*.html"))
	for _, path := range append(atas, sub...) {
		file, _ := filepath.Rel(dir, path)
		file = filepath.ToSlash(file) // kunci selalu "auth/login.html", juga di Windows
		if kerangka[file] {
			continue
		}
		rd.tpl[file] = template.Must(template.New("layout").ParseFiles(
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
// halaman setengah jadi.
func (rd *Renderer) Tampil(w http.ResponseWriter, r *http.Request, status int, file string, h Halaman) {
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