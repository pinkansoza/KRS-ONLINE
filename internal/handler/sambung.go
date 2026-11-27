// Titik sambung handler (versi lengkap tahap 5). Fungsi-fungsi ini sudah ada
// sejak tahap 2 dan 4 dalam bentuk sederhana; di tahap 5 isinya disambungkan
// ke rekap dan daftar tunggu.

package handler

import (
	"net/http"

	"krs-online/internal/model"
)

// ringkasanDashboard menambahkan ringkasan rekap KRS semester terpilih
// ke dashboard admin (FR-G3).
func (h *AdminHandler) ringkasanDashboard(r *http.Request, semua []model.Semester, data map[string]any) error {
	if id := h.semesterDipilih(r, semua); id > 0 {
		rk, err := h.Svc.Rekap(r.Context(), id, "", false)
		if err != nil {
			return err
		}
		data["Rekap"] = rk
	}
	return nil
}

// dataDashboard: data KRS ditambah daftar tunggu milik mahasiswa.
func (h *MahasiswaHandler) dataDashboard(r *http.Request) (any, error) {
	return h.Svc.DashboardLengkap(r.Context(), user(r))
}

// kelasPenuh: katalog tampil lagi dengan tawaran masuk daftar tunggu (FR-E1).
func (h *MahasiswaHandler) kelasPenuh(w http.ResponseWriter, r *http.Request, k *model.Class) {
	h.tampilKatalog(w, r, http.StatusConflict, "", 0, "", false, "", k)
}
