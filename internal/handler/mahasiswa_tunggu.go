// Tahap 5: halaman daftar tunggu mahasiswa.

package handler

import (
	"net/http"
)

func (h *MahasiswaHandler) MasukTunggu(w http.ResponseWriter, r *http.Request) {
	_, err := h.Svc.MasukTunggu(r.Context(), user(r), angka(r.PostFormValue("class_id")))
	if err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilKatalog(w, r, http.StatusUnprocessableEntity, "", 0, "", false, pesan, nil)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, "/mhs/tunggu", "tunggu")
}

func (h *MahasiswaHandler) KeluarTunggu(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	if err := h.Svc.KeluarTunggu(r.Context(), user(r), id); err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilTunggu(w, r, http.StatusUnprocessableEntity, pesan)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, "/mhs/tunggu", "keluar")
}

func (h *MahasiswaHandler) DaftarTunggu(w http.ResponseWriter, r *http.Request) {
	h.tampilTunggu(w, r, http.StatusOK, "")
}

func (h *MahasiswaHandler) tampilTunggu(w http.ResponseWriter, r *http.Request, status int, galat string) {
	d, err := h.Svc.DaftarTunggu(r.Context(), user(r))
	if err != nil {
		h.galatAtauGagal(w, r, err)
		return
	}
	h.View.Tampil(w, r, status, "mhs/tunggu.html", Halaman{Title: "Daftar tunggu", Menu: "tunggu", Galat: galat, Data: d})
}
