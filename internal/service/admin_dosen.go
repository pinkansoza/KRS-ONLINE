package service

import (
	"context"
	"errors"
	"strings"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

func (s *AdminService) Dosen(ctx context.Context) ([]model.Lecturer, error) {
	return s.R.Lecturer.Semua(ctx)
}

func (s *AdminService) DaftarDosen(ctx context.Context, q string) ([]model.Lecturer, error) {
	return s.R.Lecturer.Daftar(ctx, q)
}

func validasiDosen(d model.Lecturer) FormError {
	errs := FormError{}
	wajib(errs, "nip", d.NIP, "NIP", 20)
	wajib(errs, "name", d.Name, "Nama", 150)
	return errs
}

func (s *AdminService) TambahDosen(ctx context.Context, d model.Lecturer, password string) error {
	d.NIP = strings.TrimSpace(d.NIP)
	d.Name = strings.TrimSpace(d.Name)
	errs := validasiDosen(d)
	email, err := normalEmail(d.Email)
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
		uid, err := tx.User.Tambah(ctx, email, hash, model.RoleDosen)
		if errors.Is(err, model.ErrEmailTerpakai) {
			return FormError{"email": "Email sudah terdaftar"}
		}
		if err != nil {
			return err
		}
		d.UserID = uid
		if _, err := tx.Lecturer.Tambah(ctx, d); errors.Is(err, model.ErrDuplikat) {
			return FormError{"nip": "NIP sudah terdaftar"}
		} else {
			return err
		}
	})
}