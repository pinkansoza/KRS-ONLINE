// Titik sambung handler: fungsi kecil yang isinya berubah ketika fitur di
// tahap berikutnya sudah ada. Tahap 5 mengganti file ini dengan versi lengkap.

package handler

import (
	"net/http"

	"krs-online/internal/model"
)

// ringkasanDashboard menambahkan ringkasan rekap KRS ke dashboard admin.
// Tahap 2: rekap belum dibuat, jadi belum ada yang ditambahkan (diisi di tahap 5).
func (h *AdminHandler) ringkasanDashboard(r *http.Request, semua []model.Semester, data map[string]any) error {
	return nil
}
