// Tahap 2: admin mengelola data & akun mahasiswa.

package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

func (s *AdminService) Mahasiswa(ctx context.Context, q string) ([]model.Student, error) {
	return s.R.Student.Daftar(ctx, q)
}

func (s *AdminService) DetailMahasiswa(ctx context.Context, id int) (model.Student, error) {
	return s.R.Student.CariByID(ctx, id)
}

var polaNIM = regexp.MustCompile(`^\d{8,20}$`)

func validasiMhs(m model.Student) FormError {
	errs := FormError{}
	wajib(errs, "name", m.Name, "Nama", 150)
	if m.EntryYear < 2000 || m.EntryYear > 2100 {
		errs["entry"] = "Tahun masuk tidak valid"
	}
	if m.CurriculumID == 0 {
		errs["curriculum"] = "Pilih kurikulum"
	}
	switch m.Status {
	case "AKTIF", "CUTI", "LULUS", "NONAKTIF":
	default:
		errs["status"] = "Status tidak valid"
	}
	return errs
}

// TambahMahasiswa membuat akun login (password di-hash bcrypt) dan data
// mahasiswa dalam satu transaksi.
func (s *AdminService) TambahMahasiswa(ctx context.Context, m model.Student, password string) error {
	m.Name, m.NIM = strings.TrimSpace(m.Name), strings.TrimSpace(m.NIM)
	errs := validasiMhs(m)
	if !polaNIM.MatchString(m.NIM) {
		errs["nim"] = "NIM berupa 8–20 digit angka"
	}
	email, err := normalEmail(m.Email)
	if err != nil {
		errs["email"] = err.Error()
	}
	hash, err := HashPassword(password)
	if err != nil {
		errs["password"] = err.Error()
	}
	if len(errs) > 0 {
		return errs
	}
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		uid, err := tx.User.Tambah(ctx, email, hash, model.RoleMahasiswa)
		if errors.Is(err, model.ErrEmailTerpakai) {
			return FormError{"email": "Email sudah terdaftar"}
		}
		if err != nil {
			return err
		}
		m.UserID = uid
		if _, err := tx.Student.Tambah(ctx, m); errors.Is(err, model.ErrDuplikat) {
			return FormError{"nim": "NIM sudah terdaftar"}
		} else {
			return err
		}
	})
}

func (s *AdminService) UbahMahasiswa(ctx context.Context, m model.Student, passwordBaru string) error {
	m.Name = strings.TrimSpace(m.Name)
	errs := validasiMhs(m)
	email, err := normalEmail(m.Email)
	if err != nil {
		errs["email"] = err.Error()
	}
	var hash string
	if passwordBaru != "" {
		if hash, err = HashPassword(passwordBaru); err != nil {
			errs["password"] = err.Error()
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		lama, err := tx.Student.CariByID(ctx, m.ID)
		if err != nil {
			return err
		}
		if err := tx.Student.Ubah(ctx, m); err != nil {
			return err
		}
		if err := tx.User.UbahEmail(ctx, lama.UserID, email); errors.Is(err, model.ErrEmailTerpakai) {
			return FormError{"email": "Email sudah terdaftar"}
		} else if err != nil {
			return err
		}
		if err := tx.User.SetAktif(ctx, lama.UserID, m.Status != "NONAKTIF" && m.Status != "LULUS"); err != nil {
			return err
		}
		if hash != "" {
			if err := tx.User.UbahPassword(ctx, lama.UserID, hash); err != nil {
				return err
			}
			return tx.Session.HapusMilikUser(ctx, lama.UserID)
		}
		return nil
	})
}
