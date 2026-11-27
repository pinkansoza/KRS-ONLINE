package service

import (
	"krs-online/internal/repository"
)

// AdminService: master data & rekap untuk Admin Prodi (modul A, B, C, G).
type AdminService struct {
	R   *repository.Repos
	KRS *KRSService // dipakai untuk promosi daftar tunggu saat kuota dinaikkan
}
