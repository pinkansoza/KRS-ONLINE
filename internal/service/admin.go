package service

import (
	"krs-online/internal/repository"
)

// AdminService: master data & rekap untuk Admin Prodi.
type AdminService struct {
	R *repository.Repos
}