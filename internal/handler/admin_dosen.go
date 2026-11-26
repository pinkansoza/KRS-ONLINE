package handler

import (
	"net/http"
	"strings"

	"krs-online/internal/model"
)

func (h *AdminHandler) DaftarDosen(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	daftar, err := h.Svc.DaftarDosen(r.Context(), q)
	if err != nil {
		gagal(w, err)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/dosen_list.html", "Dosen", "dosen-admin",
		map[string]any{"Dosen": daftar, "Q": q})
}

type DosenForm struct {
	Dosen model.Lecturer
}

func (h *AdminHandler) FormDosenBaru(w http.ResponseWriter, r *http.Request) {
	h.tampil(w, r, http.StatusOK, "admin/dosen_form.html", "Tambah dosen", "dosen-admin",
		DosenForm{})
}

func (h *AdminHandler) SimpanDosen(w http.ResponseWriter, r *http.Request) {
	d := model.Lecturer{
		NIP:   r.PostFormValue("nip"),
		Name:  r.PostFormValue("name"),
		Email: r.PostFormValue("email"),
	}
	if err := h.Svc.TambahDosen(r.Context(), d, r.PostFormValue("password")); err != nil {
		if pesan, fe, ok := pesanValidasi(err); ok {
			h.View.Tampil(w, r, http.StatusUnprocessableEntity, "admin/dosen_form.html",
				Halaman{Title: "Tambah dosen", Menu: "dosen-admin", Galat: pesan, Errors: fe,
					Data: DosenForm{Dosen: d}})
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, "/admin/dosen", "dibuat")
}