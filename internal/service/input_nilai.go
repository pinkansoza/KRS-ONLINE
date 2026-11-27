// Tahap 3: admin mencatat riwayat nilai, satu per satu atau impor CSV (modul C).

package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"krs-online/internal/model"
)

func (s *AdminService) RiwayatNilai(ctx context.Context, studentID int) ([]model.Grade, error) {
	return s.R.Grade.Riwayat(ctx, studentID)
}

func (s *AdminService) bobot(ctx context.Context, huruf string) (float64, error) {
	skala, err := s.R.Course.SkalaNilai(ctx)
	if err != nil {
		return 0, err
	}
	for _, g := range skala {
		if g.Letter == huruf {
			return g.Point, nil
		}
	}
	return 0, salah("Nilai huruf %q tidak dikenal.", huruf)
}

// SimpanNilai input nilai per mahasiswa per MK per semester (FR-C1).
func (s *AdminService) SimpanNilai(ctx context.Context, g model.Grade) error {
	errs := FormError{}
	if g.StudentID == 0 {
		errs["student"] = "Pilih mahasiswa"
	}
	if g.CourseID == 0 {
		errs["course"] = "Pilih mata kuliah"
	}
	if g.SemesterID == 0 {
		errs["semester"] = "Pilih semester"
	}
	p, err := s.bobot(ctx, g.Letter)
	if err != nil {
		var ve *ValidasiError
		if !errors.As(err, &ve) {
			return err
		}
		errs["letter"] = "Pilih nilai huruf"
	}
	if len(errs) > 0 {
		return errs
	}
	g.Point = p
	return s.R.Grade.Simpan(ctx, g)
}

func (s *AdminService) HapusNilai(ctx context.Context, studentID, id int) error {
	return s.R.Grade.Hapus(ctx, studentID, id)
}

type HasilImpor struct {
	Berhasil int
	Gagal    []string
}

// ImporNilai membaca CSV: nim,kode_mk,tahun_ajaran,term,nilai (FR-C2).
// Baris yang gagal dilaporkan per baris, baris lain tetap disimpan.
func (s *AdminService) ImporNilai(ctx context.Context, r io.Reader) (HasilImpor, error) {
	var h HasilImpor
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	baris := 0
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		baris++
		if err != nil {
			h.Gagal = append(h.Gagal, fmt.Sprintf("Baris %d: format CSV rusak", baris))
			continue
		}
		if baris == 1 && strings.EqualFold(strings.TrimSpace(rec[0]), "nim") {
			continue // header
		}
		if len(rec) < 5 {
			h.Gagal = append(h.Gagal, fmt.Sprintf("Baris %d: butuh 5 kolom", baris))
			continue
		}
		for i := range rec {
			rec[i] = strings.TrimSpace(rec[i])
		}
		mhs, err := s.R.Student.CariByNIM(ctx, rec[0])
		if err != nil {
			h.Gagal = append(h.Gagal, fmt.Sprintf("Baris %d: NIM %s tidak ditemukan", baris, rec[0]))
			continue
		}
		mk, err := s.R.Course.CariByKode(ctx, strings.ToUpper(rec[1]))
		if err != nil {
			h.Gagal = append(h.Gagal, fmt.Sprintf("Baris %d: kode MK %s tidak ditemukan", baris, rec[1]))
			continue
		}
		sem, err := s.R.Semester.Cari(ctx, rec[2], strings.ToUpper(rec[3]))
		if err != nil {
			h.Gagal = append(h.Gagal, fmt.Sprintf("Baris %d: semester %s %s tidak ditemukan", baris, rec[2], rec[3]))
			continue
		}
		p, err := s.bobot(ctx, strings.ToUpper(rec[4]))
		if err != nil {
			h.Gagal = append(h.Gagal, fmt.Sprintf("Baris %d: nilai %s tidak dikenal", baris, rec[4]))
			continue
		}
		if err := s.R.Grade.Simpan(ctx, model.Grade{StudentID: mhs.ID, CourseID: mk.ID, SemesterID: sem.ID,
			Letter: strings.ToUpper(rec[4]), Point: p}); err != nil {
			return h, err
		}
		h.Berhasil++
	}
	return h, nil
}
