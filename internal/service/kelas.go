// Tahap 3: kelas, jadwal, dan kuota (modul B).

package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

func (s *AdminService) Kelas(ctx context.Context, semesterID int, q string) ([]model.Class, error) {
	return s.R.Class.DaftarSemester(ctx, semesterID, q)
}

func (s *AdminService) DetailKelas(ctx context.Context, id int) (model.Class, error) {
	return s.R.Class.CariByID(ctx, id)
}

var polaKodeKelas = regexp.MustCompile(`^[A-Z0-9]{1,5}$`)

func validasiKelas(c model.Class) FormError {
	errs := FormError{}
	if !polaKodeKelas.MatchString(c.Code) {
		errs["code"] = "Kode kelas 1–5 huruf/angka, mis. A"
	}
	if c.LecturerID == 0 {
		errs["lecturer"] = "Pilih dosen pengampu"
	}
	if c.Quota < 1 || c.Quota > 500 {
		errs["quota"] = "Kuota 1 sampai 500"
	}
	if utf8.RuneCountInString(c.Room) > 30 {
		errs["room"] = "Ruang maksimal 30 karakter"
	}
	return errs
}

func (s *AdminService) TambahKelas(ctx context.Context, c model.Class) (int, error) {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	errs := validasiKelas(c)
	if c.CourseID == 0 {
		errs["course"] = "Pilih mata kuliah"
	}
	if c.SemesterID == 0 {
		errs["semester"] = "Pilih semester"
	}
	if len(errs) > 0 {
		return 0, errs
	}
	id, err := s.R.Class.Tambah(ctx, c)
	if errors.Is(err, model.ErrDuplikat) {
		return 0, FormError{"code": "Kode kelas sudah dipakai untuk MK ini di semester ini"}
	}
	return id, err
}

// UbahKelas: kuota tidak boleh diturunkan di bawah jumlah peserta (FR-B3).
// Kalau kuota dinaikkan, kursi baru langsung diisi dari daftar tunggu.
func (s *AdminService) UbahKelas(ctx context.Context, c model.Class) error {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	if errs := validasiKelas(c); len(errs) > 0 {
		return errs
	}
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		lama, err := tx.Class.Kunci(ctx, c.ID)
		if err != nil {
			return err
		}
		if c.Quota < lama.Enrolled {
			return FormError{"quota": fmt.Sprintf("Kuota tidak boleh di bawah jumlah peserta saat ini (%d)", lama.Enrolled)}
		}
		if err := tx.Class.Ubah(ctx, c); err != nil {
			if errors.Is(err, model.ErrDuplikat) {
				return FormError{"code": "Kode kelas sudah dipakai untuk MK ini di semester ini"}
			}
			return err
		}
		if c.Quota > lama.Quota {
			return s.kuotaBertambah(ctx, tx, c.ID) // titik sambung, lihat sambung.go
		}
		return nil
	})
}

func (s *AdminService) HapusKelas(ctx context.Context, id int) error {
	err := s.R.Class.Hapus(ctx, id)
	if errors.Is(err, model.ErrDuplikat) {
		return salah("Kelas sudah dipakai di KRS atau daftar tunggu, tidak bisa dihapus. Turunkan saja kuotanya.")
	}
	return err
}

var polaJam = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (s *AdminService) TambahJadwal(ctx context.Context, j model.Schedule) error {
	errs := FormError{}
	if j.Day < 1 || j.Day > 7 {
		errs["day"] = "Pilih hari"
	}
	if !polaJam.MatchString(j.Start) {
		errs["start"] = "Format jam HH:MM"
	}
	if !polaJam.MatchString(j.End) {
		errs["end"] = "Format jam HH:MM"
	} else if j.End <= j.Start {
		errs["end"] = "Jam selesai harus setelah jam mulai"
	}
	if len(errs) > 0 {
		return errs
	}
	if _, err := s.R.Class.CariByID(ctx, j.ClassID); err != nil {
		return err
	}
	return s.R.Class.TambahJadwal(ctx, j)
}

func (s *AdminService) HapusJadwal(ctx context.Context, classID, id int) error {
	return s.R.Class.HapusJadwal(ctx, classID, id)
}
