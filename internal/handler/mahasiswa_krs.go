// Tahap 4: halaman dashboard, katalog, dan KRS Saya milik mahasiswa.

package handler

import (
	"errors"
	"net/http"
	"strings"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

func (h *MahasiswaHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.dataDashboard(r) // titik sambung, lihat sambung.go
	if err != nil {
		h.galatAtauGagal(w, r, err)
		return
	}
	h.View.Tampil(w, r, http.StatusOK, "mhs/dashboard.html", Halaman{Title: "Dashboard", Menu: "dashboard", Data: d})
}

// KatalogView menambah info kelas penuh yang sedang ditawarkan daftar tunggu.
type KatalogView struct {
	service.Katalog
	Penuh *model.Class
}

func (h *MahasiswaHandler) Katalog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	h.tampilKatalog(w, r, http.StatusOK, strings.TrimSpace(q.Get("q")), angkaQuery(r, "hari", 0),
		q.Get("status"), q.Get("lulus") == "1", "", nil)
}

func (h *MahasiswaHandler) tampilKatalog(w http.ResponseWriter, r *http.Request, status int,
	q string, hari int, filter string, sembunyi bool, galat string, penuh *model.Class) {
	kt, err := h.Svc.Katalog(r.Context(), user(r), q, hari, filter, sembunyi)
	if err != nil {
		h.galatAtauGagal(w, r, err)
		return
	}
	h.View.Tampil(w, r, status, "mhs/katalog.html", Halaman{Title: "Katalog kelas", Menu: "katalog",
		Galat: galat, Data: KatalogView{Katalog: kt, Penuh: penuh}})
}

// Tambah kelas ke KRS (FR-D2). Gagal validasi -> katalog tampil lagi dengan
// pesan spesifik (FR-D3). Kelas penuh -> tawarkan daftar tunggu.
func (h *MahasiswaHandler) Tambah(w http.ResponseWriter, r *http.Request) {
	classID := angka(r.PostFormValue("class_id"))
	_, err := h.Svc.Tambah(r.Context(), user(r), classID)
	var penuh *service.KelasPenuhError
	switch {
	case err == nil:
		kode := "ditambah"
		if r.PostFormValue("mode") == "UBAH" {
			kode = "diminta"
		}
		redirect(w, r, "/mhs/krs", kode)
	case errors.As(err, &penuh):
		h.kelasPenuh(w, r, &penuh.Kelas) // titik sambung, lihat sambung.go
	default:
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilKatalog(w, r, http.StatusUnprocessableEntity, "", 0, "", false, pesan, nil)
			return
		}
		gagal(w, err)
	}
}

func (h *MahasiswaHandler) KRS(w http.ResponseWriter, r *http.Request) {
	h.tampilKRS(w, r, http.StatusOK, "")
}

func (h *MahasiswaHandler) tampilKRS(w http.ResponseWriter, r *http.Request, status int, galat string) {
	d, err := h.Svc.KRSSaya(r.Context(), user(r))
	if err != nil {
		h.galatAtauGagal(w, r, err)
		return
	}
	h.View.Tampil(w, r, status, "mhs/krs.html", Halaman{Title: "KRS Saya", Menu: "krs", Galat: galat, Data: d})
}

func (h *MahasiswaHandler) Batal(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	if _, err := h.Svc.Batal(r.Context(), user(r), id); err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilKRS(w, r, http.StatusUnprocessableEntity, pesan)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, "/mhs/krs", "dibatalkan")
}

func (h *MahasiswaHandler) Ajukan(w http.ResponseWriter, r *http.Request) {
	if err := h.Svc.Ajukan(r.Context(), user(r)); err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilKRS(w, r, http.StatusUnprocessableEntity, pesan)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, "/mhs/krs", "diajukan")
}
