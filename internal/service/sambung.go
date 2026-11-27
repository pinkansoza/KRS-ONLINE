// Titik sambung service (versi lengkap tahap 5). Fungsi-fungsi ini sudah ada
// sejak tahap 3 dan 4 dalam bentuk sederhana; di tahap 5 isinya disambungkan
// ke daftar tunggu.

package service

import (
	"context"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

// kuotaBertambah: kalau kuota dinaikkan, kursi baru langsung diisi dari daftar tunggu (FR-B3).
func (s *AdminService) kuotaBertambah(ctx context.Context, tx *repository.Repos, classID int) error {
	return s.KRS.promosikan(ctx, tx, classID)
}

// kursiKosong: kursi yang dilepas diberikan ke antrean terdepan yang masih valid (FR-E3, BR-8).
func (s *KRSService) kursiKosong(ctx context.Context, tx *repository.Repos, classID int) error {
	return s.promosikan(ctx, tx, classID)
}

// antreanMenunggu = course_id yang sedang diantrekan mahasiswa di semester ini.
// Dipakai validasi supaya mahasiswa tidak mengambil MK yang sedang ia antrekan.
func antreanMenunggu(ctx context.Context, r *repository.Repos, studentID, semesterID int) (map[int]bool, error) {
	menunggu := map[int]bool{}
	antre, err := r.Waitlist.MilikMahasiswa(ctx, studentID, semesterID)
	if err != nil {
		return nil, err
	}
	for _, w := range antre {
		if w.Status == model.TungguMenunggu {
			menunggu[w.Class.CourseID] = true
		}
	}
	return menunggu, nil
}
