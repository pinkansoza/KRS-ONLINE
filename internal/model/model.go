package model

import (
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("data tidak ditemukan")
	ErrLoginGagal    = errors.New("email atau password salah")
	ErrEmailTerpakai = errors.New("email sudah terdaftar")
	ErrDuplikat      = errors.New("data sudah ada")
	ErrTerlarang     = errors.New("tidak berhak mengakses data ini")
)

const (
	RoleMahasiswa = "MAHASISWA"
	RoleDosen     = "DOSEN_WALI"
	RoleAdmin     = "ADMIN"
)

type User struct {
	ID         int
	Email      string
	Role       string
	Nama       string
	StudentID  int // 0 kalau bukan mahasiswa
	LecturerID int // 0 kalau bukan dosen
	CSRF       string
	NotifBaru  int
}

func (u User) IsMahasiswa() bool { return u.Role == RoleMahasiswa }

func (u User) IsDosen() bool { return u.Role == RoleDosen }

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

type Lecturer struct {
	ID     int
	UserID int
	NIP    string
	Name   string
	Email  string
}

type Notification struct {
	ID        int
	Message   string
	IsRead    bool
	CreatedAt time.Time
}