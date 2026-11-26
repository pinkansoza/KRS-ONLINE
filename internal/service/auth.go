package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

// AuthService: login, logout, session, dan password.
type AuthService struct {
	Users    *repository.UserRepo
	Sessions *repository.SessionRepo
	Notif    *repository.NotifRepo
}

// validasiPassword: bcrypt hanya memakai 72 byte pertama.
func validasiPassword(p string) error {
	if len(p) < 8 || len(p) > 72 {
		return &ValidasiError{"Password 8 sampai 72 karakter"}
	}
	return nil
}

func normalEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return email, &ValidasiError{"Format email tidak valid"}
	}
	return email, nil
}

// HashPassword dipakai saat admin membuat akun (akun dibuat prodi, bukan daftar sendiri).
func HashPassword(p string) (string, error) {
	if err := validasiPassword(p); err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(h), err
}

// Login mengembalikan token session baru jika email dan password cocok.
// Pesan gagal dibuat umum supaya tidak membocorkan email mana yang terdaftar.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	id, hash, err := s.Users.HashByEmail(ctx, email)
	if errors.Is(err, model.ErrNotFound) {
		return "", model.ErrLoginGagal
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", model.ErrLoginGagal
	}
	return s.Sessions.Buat(ctx, id)
}

func (s *AuthService) UserDariToken(ctx context.Context, token string) (model.User, error) {
	return s.Sessions.UserDariToken(ctx, token)
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.Sessions.Hapus(ctx, token)
}

// GantiPassword: cek password lama, simpan hash baru, cabut semua session lama.
func (s *AuthService) GantiPassword(ctx context.Context, userID int, lama, baru, ulang string) error {
	errs := FormError{}
	if lama == "" {
		errs["lama"] = "Password lama wajib diisi"
	}
	if err := validasiPassword(baru); err != nil {
		errs["baru"] = err.Error()
	} else if baru != ulang {
		errs["ulang"] = "Konfirmasi password tidak sama"
	}
	if len(errs) > 0 {
		return errs
	}
	hash, err := s.Users.HashByID(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(lama)) != nil {
		return FormError{"lama": "Password lama salah"}
	}
	baruHash, err := HashPassword(baru)
	if err != nil {
		return err
	}
	if err := s.Users.UbahPassword(ctx, userID, baruHash); err != nil {
		return err
	}
	return s.Sessions.HapusMilikUser(ctx, userID)
}

func (s *AuthService) Notifikasi(ctx context.Context, u model.User) ([]model.Notification, error) {
	return s.Notif.Daftar(ctx, u.ID, 50)
}

func (s *AuthService) TandaiDibaca(ctx context.Context, u model.User) error {
	return s.Notif.TandaiDibaca(ctx, u.ID)
}