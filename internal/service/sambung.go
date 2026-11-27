// Titik sambung service: fungsi kecil yang isinya berubah ketika fitur di
// tahap berikutnya sudah ada. Tahap 5 mengganti file ini dengan versi lengkap.

package service

import (
	"context"

	"krs-online/internal/repository"
)

// kuotaBertambah dipanggil (di dalam transaksi) setelah admin menaikkan kuota kelas.
// Daftar tunggu belum ada, jadi tidak ada yang dikerjakan (diisi di tahap 5).
func (s *AdminService) kuotaBertambah(ctx context.Context, tx *repository.Repos, classID int) error {
	return nil
}

// kursiKosong dipanggil (di dalam transaksi) setelah mahasiswa melepas kursi.
// Tahap 4: kursi cukup dilepas. Tahap 5: kursi diberikan ke antrean terdepan.
func (s *KRSService) kursiKosong(ctx context.Context, tx *repository.Repos, classID int) error {
	return nil
}

// antreanMenunggu = course_id yang sedang diantrekan mahasiswa di semester ini.
// Tahap 4: daftar tunggu belum ada, jadi selalu kosong.
func antreanMenunggu(ctx context.Context, r *repository.Repos, studentID, semesterID int) (map[int]bool, error) {
	return map[int]bool{}, nil
}
