// Titik sambung service: fungsi kecil yang isinya berubah ketika fitur di
// tahap berikutnya sudah ada. Tahap 5 mengganti file ini dengan versi lengkap.

package service

import (
	"context"

	"krs-online/internal/repository"
)

// kuotaBertambah dipanggil (di dalam transaksi) setelah admin menaikkan kuota kelas.
// Tahap 3: daftar tunggu belum ada, jadi tidak ada yang dikerjakan.
// Tahap 5: kursi baru langsung diisi dari daftar tunggu.
func (s *AdminService) kuotaBertambah(ctx context.Context, tx *repository.Repos, classID int) error {
	return nil
}
