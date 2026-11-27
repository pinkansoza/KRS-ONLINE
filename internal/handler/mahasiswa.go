// Tahap 3: halaman untuk peran Mahasiswa. Dimulai dengan transkrip; tahap 4 menambah KRS.

package handler

import (
	"net/http"

	"krs-online/internal/service"
)

// MahasiswaHandler: halaman untuk peran Mahasiswa (modul C, D, E).
type MahasiswaHandler struct {
	Svc  *service.KRSService
	View *Renderer
}

func (h *MahasiswaHandler) Transkrip(w http.ResponseWriter, r *http.Request) {
	t, err := h.Svc.Transkrip(r.Context(), user(r).StudentID)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, r, http.StatusOK, "mhs/transkrip.html", Halaman{Title: "Transkrip", Menu: "transkrip",
		Data: map[string]any{"T": t, "Tab": r.URL.Query().Get("tab"), "Base": "/mhs/transkrip"}})
}

// galatAtauGagal: kesalahan "bisnis" (mis. belum ada semester aktif) tetap
// ditampilkan di dalam layout supaya user tidak melihat halaman kosong.
func (h *MahasiswaHandler) galatAtauGagal(w http.ResponseWriter, r *http.Request, err error) {
	if pesan, _, ok := pesanValidasi(err); ok {
		h.View.Tampil(w, r, http.StatusUnprocessableEntity, "notifikasi.html",
			Halaman{Title: "Informasi", Galat: pesan})
		return
	}
	gagal(w, err)
}
