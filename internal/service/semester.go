// Tahap 2: semester dan periode KRS (FR-B4).

package service

import (
	"context"
	"errors"
	"regexp"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

func (s *AdminService) Semester(ctx context.Context) ([]model.Semester, error) {
	return s.R.Semester.Semua(ctx)
}

func (s *AdminService) SemesterByID(ctx context.Context, id int) (model.Semester, error) {
	return s.R.Semester.CariByID(ctx, id)
}

var polaTahun = regexp.MustCompile(`^\d{4}/\d{4}$`)

func (s *AdminService) TambahSemester(ctx context.Context, sem model.Semester) error {
	errs := FormError{}
	if !polaTahun.MatchString(sem.AcademicYear) {
		errs["tahun"] = "Format tahun ajaran: 2026/2027"
	}
	if sem.Term != "GANJIL" && sem.Term != "GENAP" {
		errs["term"] = "Pilih Ganjil atau Genap"
	}
	if len(errs) > 0 {
		return errs
	}
	_, err := s.R.Semester.Tambah(ctx, sem)
	if errors.Is(err, model.ErrDuplikat) {
		return FormError{"tahun": "Semester ini sudah ada"}
	}
	return err
}

func (s *AdminService) UbahPeriode(ctx context.Context, sem model.Semester) error {
	errs := FormError{}
	pasang := []struct {
		field          string
		mulai, selesai int64
		mz, sz         bool
	}{
		{"krs", sem.KRSStart.Unix(), sem.KRSEnd.Unix(), sem.KRSStart.IsZero(), sem.KRSEnd.IsZero()},
		{"approval", sem.ApprovalStart.Unix(), sem.ApprovalEnd.Unix(), sem.ApprovalStart.IsZero(), sem.ApprovalEnd.IsZero()},
		{"change", sem.ChangeStart.Unix(), sem.ChangeEnd.Unix(), sem.ChangeStart.IsZero(), sem.ChangeEnd.IsZero()},
	}
	for _, p := range pasang {
		if p.mz != p.sz {
			errs[p.field] = "Isi tanggal mulai dan selesai sekaligus (atau kosongkan keduanya)"
		} else if !p.mz && p.selesai < p.mulai {
			errs[p.field] = "Tanggal selesai harus setelah tanggal mulai"
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return s.R.Semester.UbahPeriode(ctx, sem)
}

func (s *AdminService) AktifkanSemester(ctx context.Context, id int) error {
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		return tx.Semester.JadikanAktif(ctx, id)
	})
}
