// Tahap 5: rekap peserta kelas dan daftar tunggu (modul G).

package service

import (
	"context"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

type Rekap struct {
	Semester                             model.Semester
	Status                               model.RekapStatus
	Kelas                                []model.Class
	Promosi                              []model.PromotionLog
	TotalTerisi, TotalKuota, TotalTunggu int
}

func (s *AdminService) Rekap(ctx context.Context, semesterID int, q string, hanyaPenuh bool) (Rekap, error) {
	var rk Rekap
	var err error
	if rk.Semester, err = s.R.Semester.CariByID(ctx, semesterID); err != nil {
		return rk, err
	}
	if rk.Status, err = s.R.KRS.RekapStatus(ctx, semesterID); err != nil {
		return rk, err
	}
	kelas, err := s.R.Class.DaftarSemester(ctx, semesterID, q)
	if err != nil {
		return rk, err
	}
	for _, k := range kelas {
		if hanyaPenuh && !k.Penuh() {
			continue
		}
		rk.Kelas = append(rk.Kelas, k)
		rk.TotalTerisi += k.Enrolled
		rk.TotalKuota += k.Quota
		rk.TotalTunggu += k.Waitlist
	}
	rk.Promosi, err = s.R.Waitlist.RiwayatPromosi(ctx, semesterID, 10)
	return rk, err
}

type RekapKelas struct {
	Kelas   model.Class
	Peserta []repository.Peserta
	Antrean []model.WaitlistEntry
}

func (s *AdminService) RekapKelas(ctx context.Context, classID int) (RekapKelas, error) {
	var rk RekapKelas
	var err error
	if rk.Kelas, err = s.R.Class.CariByID(ctx, classID); err != nil {
		return rk, err
	}
	if rk.Peserta, err = s.R.Class.Peserta(ctx, classID); err != nil {
		return rk, err
	}
	rk.Antrean, err = s.R.Waitlist.AntreanKelas(ctx, classID)
	return rk, err
}
