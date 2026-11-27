// Tahap 2: mata kuliah, prasyarat gabungan, dan kesetaraan kurikulum (modul A).

package service

import (
	"context"
	"errors"
	"strings"

	"krs-online/internal/model"
)

func (s *AdminService) MataKuliah(ctx context.Context, q string, kurikulum int) ([]model.Course, error) {
	return s.R.Course.Daftar(ctx, q, kurikulum, false)
}

func (s *AdminService) MataKuliahAktif(ctx context.Context) ([]model.Course, error) {
	return s.R.Course.Daftar(ctx, "", 0, true)
}

func (s *AdminService) DetailMK(ctx context.Context, id int) (model.Course, error) {
	return s.R.Course.CariByID(ctx, id)
}

func (s *AdminService) Kurikulum(ctx context.Context) ([]model.Curriculum, error) {
	return s.R.Course.Kurikulum(ctx)
}

func (s *AdminService) TambahKurikulum(ctx context.Context, c model.Curriculum) error {
	errs := FormError{}
	wajib(errs, "nama", c.Name, "Nama kurikulum", 100)
	if c.YearStart < 1990 || c.YearStart > 2100 {
		errs["tahun"] = "Tahun berlaku tidak valid"
	}
	if len(errs) > 0 {
		return errs
	}
	if err := s.R.Course.TambahKurikulum(ctx, c); errors.Is(err, model.ErrDuplikat) {
		return FormError{"nama": "Nama kurikulum sudah ada"}
	} else {
		return err
	}
}

func validasiMK(c model.Course) error {
	errs := FormError{}
	wajib(errs, "code", c.Code, "Kode", 20)
	wajib(errs, "name", c.Name, "Nama mata kuliah", 150)
	if c.Credits < 1 || c.Credits > 10 {
		errs["credits"] = "SKS harus 1 sampai 10"
	}
	if c.CurriculumID == 0 {
		errs["curriculum"] = "Pilih kurikulum"
	}
	if c.RecommendedSemester < 0 || c.RecommendedSemester > 14 {
		errs["semester"] = "Semester anjuran 1 sampai 14 (atau kosong)"
	}
	return errs.atauNil()
}

func (s *AdminService) SimpanMK(ctx context.Context, c model.Course) (int, error) {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	c.Name = strings.TrimSpace(c.Name)
	if err := validasiMK(c); err != nil {
		return 0, err
	}
	var err error
	if c.ID == 0 {
		c.ID, err = s.R.Course.Tambah(ctx, c)
	} else {
		err = s.R.Course.Ubah(ctx, c)
	}
	if errors.Is(err, model.ErrDuplikat) {
		return 0, FormError{"code": "Kode mata kuliah sudah dipakai"}
	}
	return c.ID, err
}

func (s *AdminService) SetAktifMK(ctx context.Context, id int, aktif bool) error {
	return s.R.Course.SetAktif(ctx, id, aktif)
}

func (s *AdminService) PrasyaratMK(ctx context.Context, courseID int) ([]model.PrereqGroup, error) {
	semua, err := s.R.Course.SemuaPrasyarat(ctx)
	return semua[courseID], err
}

func (s *AdminService) TambahGrup(ctx context.Context, courseID int) error {
	if _, err := s.R.Course.CariByID(ctx, courseID); err != nil {
		return err
	}
	_, err := s.R.Course.TambahGrup(ctx, courseID)
	return err
}

func (s *AdminService) HapusGrup(ctx context.Context, courseID, groupID int) error {
	return s.R.Course.HapusGrup(ctx, courseID, groupID)
}

// TambahButir menambah butir OR ke sebuah grup, dengan cek prasyarat siklik (FR-A3).
func (s *AdminService) TambahButir(ctx context.Context, courseID, groupID, prereqID int, minHuruf string) error {
	pemilik, err := s.R.Course.GrupMilik(ctx, groupID)
	if err != nil || pemilik != courseID {
		return model.ErrNotFound
	}
	if prereqID == 0 {
		return salah("Pilih mata kuliah prasyarat.")
	}
	skala, err := s.R.Course.SkalaNilai(ctx)
	if err != nil {
		return err
	}
	minPoint := -1.0
	for _, g := range skala {
		if g.Letter == minHuruf {
			minPoint = g.Point
		}
	}
	if minPoint < 0 {
		return salah("Nilai minimal tidak valid.")
	}
	semua, err := s.R.Course.SemuaPrasyarat(ctx)
	if err != nil {
		return err
	}
	graf := map[int][]int{}
	for mk, grup := range semua {
		for _, g := range grup {
			for _, it := range g.Items {
				graf[mk] = append(graf[mk], it.PrereqCourseID)
			}
		}
	}
	if AdaSiklus(graf, courseID, prereqID) {
		p, _ := s.R.Course.CariByID(ctx, prereqID)
		return salah("Ditolak: prasyarat siklik. %s (langsung atau tidak langsung) sudah membutuhkan mata kuliah ini.", p.Label())
	}
	err = s.R.Course.TambahButir(ctx, groupID, prereqID, minPoint)
	if errors.Is(err, model.ErrDuplikat) {
		return salah("Mata kuliah tersebut sudah ada di grup ini.")
	}
	return err
}

func (s *AdminService) HapusButir(ctx context.Context, itemID int) error {
	return s.R.Course.HapusButir(ctx, itemID)
}

func (s *AdminService) Kesetaraan(ctx context.Context) ([]model.Equivalence, error) {
	return s.R.Course.Kesetaraan(ctx)
}

func (s *AdminService) TambahKesetaraan(ctx context.Context, lama, baru int) error {
	if lama == 0 || baru == 0 {
		return salah("Pilih MK lama dan MK baru.")
	}
	if lama == baru {
		return salah("MK lama dan MK baru tidak boleh sama.")
	}
	err := s.R.Course.TambahKesetaraan(ctx, lama, baru)
	if errors.Is(err, model.ErrDuplikat) {
		return salah("Pasangan kesetaraan ini sudah ada.")
	}
	return err
}

func (s *AdminService) HapusKesetaraan(ctx context.Context, id int) error {
	return s.R.Course.HapusKesetaraan(ctx, id)
}

func (s *AdminService) SkalaNilai(ctx context.Context) ([]model.GradeScale, error) {
	return s.R.Course.SkalaNilai(ctx)
}

// AdaSiklus memeriksa apakah menambah prasyarat "mk membutuhkan butuh" akan
// membuat prasyarat siklik, termasuk siklik tidak langsung (FR-A3).
// graf: course_id -> daftar course_id yang menjadi prasyaratnya.
func AdaSiklus(graf map[int][]int, mk, butuh int) bool {
	if mk == butuh {
		return true
	}
	// telusuri dari "butuh": kalau bisa sampai ke "mk", berarti siklik
	dikunjungi := map[int]bool{}
	tumpukan := []int{butuh}
	for len(tumpukan) > 0 {
		n := tumpukan[len(tumpukan)-1]
		tumpukan = tumpukan[:len(tumpukan)-1]
		if n == mk {
			return true
		}
		if dikunjungi[n] {
			continue
		}
		dikunjungi[n] = true
		tumpukan = append(tumpukan, graf[n]...)
	}
	return false
}
