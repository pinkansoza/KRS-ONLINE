package handler

import (
	"net/http"

	"krs-online/internal/service"
)

// AdminHandler: halaman Admin Prodi.
type AdminHandler struct {
	Svc  *service.AdminService
	View *Renderer
}

func (h *AdminHandler) tampil(w http.ResponseWriter, r *http.Request, status int, file, title, menu string, data any) {
	h.View.Tampil(w, r, status, file, Halaman{Title: title, Menu: menu, Data: data})
}

// formGagal: kalau error validasi, form tampil lagi dengan isian lama (status 422).
func (h *AdminHandler) formGagal(w http.ResponseWriter, r *http.Request, err error, file, title, menu string, data any) {
	if pesan, fe, ok := pesanValidasi(err); ok {
		h.View.Tampil(w, r, http.StatusUnprocessableEntity, file,
			Halaman{Title: title, Menu: menu, Galat: pesan, Errors: fe, Data: data})
		return
	}
	gagal(w, err)
}