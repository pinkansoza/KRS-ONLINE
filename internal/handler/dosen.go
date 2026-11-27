package handler

import (
	"fmt"
	"net/http"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

// DosenHandler: halaman untuk peran Dosen Wali (modul F).
type DosenHandler struct {
	Svc  *service.KRSService
	View *Renderer
}

var statusBimbingan = map[string]bool{"": true, "BELUM": true, "DRAFT": true, "DIAJUKAN": true, "DISETUJUI": true, "PERUBAHAN": true}

func (h *DosenHandler) Daftar(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if !statusBimbingan[status] {
		status = ""
	}
	d, err := h.Svc.Bimbingan(r.Context(), user(r), status)
	if err != nil {
		gagal(w, err)
		return
	}
	menu := "bimbingan"
	if status == "PERUBAHAN" {
		menu = "perubahan"
	}
	h.View.Tampil(w, r, http.StatusOK, "dosen/daftar.html", Halaman{Title: "KRS bimbingan", Menu: menu, Data: d})
}

func (h *DosenHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	h.tampilDetail(w, r, id, http.StatusOK, "", false)
}

func (h *DosenHandler) tampilDetail(w http.ResponseWriter, r *http.Request, id, status int, galat string, bukaTolak bool) {
	d, err := h.Svc.DetailBimbingan(r.Context(), user(r), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, r, status, "dosen/detail.html", Halaman{Title: "Detail KRS " + d.Mhs.Name, Menu: "bimbingan",
		Galat: galat, Data: map[string]any{"D": d, "BukaTolak": bukaTolak}})
}

func (h *DosenHandler) Setujui(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	if err := h.Svc.Setujui(r.Context(), user(r), id, r.PostFormValue("catatan")); err != nil {
		h.hasil(w, r, id, err, false)
		return
	}
	redirect(w, r, fmt.Sprintf("/dosen/krs/%d", id), "disetujui")
}

func (h *DosenHandler) Tolak(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	if err := h.Svc.Tolak(r.Context(), user(r), id, r.PostFormValue("catatan")); err != nil {
		h.hasil(w, r, id, err, true)
		return
	}
	redirect(w, r, fmt.Sprintf("/dosen/krs/%d", id), "ditolak")
}

func (h *DosenHandler) Putuskan(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	item, ok := idDariPath(w, r, "item")
	if !ok {
		return
	}
	setuju := r.PostFormValue("aksi") == "setuju"
	if err := h.Svc.PutuskanItem(r.Context(), user(r), id, item, setuju, r.PostFormValue("catatan")); err != nil {
		h.hasil(w, r, id, err, false)
		return
	}
	redirect(w, r, fmt.Sprintf("/dosen/krs/%d", id), "diputuskan")
}

func (h *DosenHandler) hasil(w http.ResponseWriter, r *http.Request, id int, err error, bukaTolak bool) {
	if pesan, _, ok := pesanValidasi(err); ok {
		h.tampilDetail(w, r, id, http.StatusUnprocessableEntity, pesan, bukaTolak)
		return
	}
	gagal(w, err)
}

// Transkrip mahasiswa bimbingan (dosen hanya boleh melihat bimbingannya).
func (h *DosenHandler) Transkrip(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	t, err := h.Svc.Transkrip(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	if t.Mhs.AdvisorID != user(r).LecturerID {
		gagal(w, model.ErrTerlarang)
		return
	}
	h.View.Tampil(w, r, http.StatusOK, "mhs/transkrip.html", Halaman{Title: "Transkrip " + t.Mhs.Name,
		Menu: "bimbingan", Data: map[string]any{"T": t, "Tab": r.URL.Query().Get("tab"),
			"Base": fmt.Sprintf("/dosen/mahasiswa/%d", id)}})
}
