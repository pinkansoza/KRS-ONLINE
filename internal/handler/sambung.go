// Titik sambung handler: fungsi kecil yang isinya berubah ketika fitur di
// tahap berikutnya sudah ada. Tahap 5 mengganti file ini dengan versi lengkap.

package handler

import (
	"net/http"

	"krs-online/internal/model"
)

// ringkasanDashboard menambahkan ringkasan rekap KRS ke dashboard admin.
// Rekap belum dibuat, jadi belum ada yang ditambahkan (diisi di tahap 5).
func (h *AdminHandler) ringkasanDashboard(r *http.Request, semua []model.Semester, data map[string]any) error {
	return nil
}

// dataDashboard mengambil data dashboard mahasiswa.
// Tahap 4: data KRS saja. Tahap 5 menambahkan daftar tunggu.
func (h *MahasiswaHandler) dataDashboard(r *http.Request) (any, error) {
	return h.Svc.Dashboard(r.Context(), user(r))
}

// kelasPenuh dipanggil saat kelas lolos semua validasi tetapi kuotanya habis.
// Tahap 4: cukup tampilkan pesan. Tahap 5 menawarkan daftar tunggu.
func (h *MahasiswaHandler) kelasPenuh(w http.ResponseWriter, r *http.Request, k *model.Class) {
	h.tampilKatalog(w, r, http.StatusConflict, "", 0, "", false, "Kelas "+k.Label()+" sudah penuh.", nil)
}
