// Tahap 3: layanan untuk mahasiswa. Dimulai dengan transkrip; tahap 4 menambah pengisian KRS.

package service

import (
	"context"
	"errors"
	"time"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

// KRSService memuat aturan bisnis pengisian KRS, daftar tunggu, dan
// persetujuan dosen wali. Service tidak tahu http.Request dan tidak tahu SQL.
type KRSService struct {
	R        *repository.Repos
	SKSBaru  int              // batas SKS mahasiswa baru (belum punya IPS)
	Sekarang func() time.Time // bisa diganti saat testing
}

func (s *KRSService) now() time.Time {
	if s.Sekarang != nil {
		return s.Sekarang()
	}
	return time.Now()
}

type TranskripMhs struct {
	Mhs model.Student
	Transkrip
}

func (s *KRSService) Transkrip(ctx context.Context, studentID int) (TranskripMhs, error) {
	var t TranskripMhs
	var err error
	if t.Mhs, err = s.R.Student.CariByID(ctx, studentID); err != nil {
		return t, err
	}
	riwayat, err := s.R.Grade.Riwayat(ctx, studentID)
	if err != nil {
		return t, err
	}
	sem, err := s.R.Semester.Aktif(ctx)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return t, err
	}
	t.Transkrip = RingkasNilai(riwayat, sem.ID)
	return t, nil
}
