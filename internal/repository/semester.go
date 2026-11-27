// Tahap 2: data semester dan periode KRS.

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"krs-online/internal/model"
)

type SemesterRepo struct {
	DB DBTX
}

const kolomSemester = `id, academic_year, term, is_active, krs_start, krs_end,
	approval_start, approval_end, change_start, change_end`

func scanSemester(sc interface{ Scan(...any) error }) (model.Semester, error) {
	var s model.Semester
	var ks, ke, as, ae, cs, ce sql.NullTime
	err := sc.Scan(&s.ID, &s.AcademicYear, &s.Term, &s.IsActive, &ks, &ke, &as, &ae, &cs, &ce)
	s.KRSStart, s.KRSEnd = waktu(ks), waktu(ke)
	s.ApprovalStart, s.ApprovalEnd = waktu(as), waktu(ae)
	s.ChangeStart, s.ChangeEnd = waktu(cs), waktu(ce)
	return s, err
}

func (r *SemesterRepo) Aktif(ctx context.Context) (model.Semester, error) {
	s, err := scanSemester(r.DB.QueryRowContext(ctx,
		"SELECT "+kolomSemester+" FROM semesters WHERE is_active = TRUE ORDER BY id DESC LIMIT 1"))
	return s, notFound(err)
}

func (r *SemesterRepo) CariByID(ctx context.Context, id int) (model.Semester, error) {
	s, err := scanSemester(r.DB.QueryRowContext(ctx,
		"SELECT "+kolomSemester+" FROM semesters WHERE id = ?", id))
	return s, notFound(err)
}

// Cari dipakai impor CSV nilai: tahun ajaran + term -> semester.
func (r *SemesterRepo) Cari(ctx context.Context, tahun, term string) (model.Semester, error) {
	s, err := scanSemester(r.DB.QueryRowContext(ctx,
		"SELECT "+kolomSemester+" FROM semesters WHERE academic_year = ? AND term = ?", tahun, term))
	return s, notFound(err)
}

func (r *SemesterRepo) Semua(ctx context.Context) ([]model.Semester, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT "+kolomSemester+" FROM semesters ORDER BY academic_year, term")
	if err != nil {
		return nil, fmt.Errorf("daftar semester: %w", err)
	}
	defer rows.Close()
	var hasil []model.Semester
	for rows.Next() {
		s, err := scanSemester(rows)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, s)
	}
	return hasil, rows.Err()
}

func (r *SemesterRepo) Tambah(ctx context.Context, s model.Semester) (int, error) {
	res, err := r.DB.ExecContext(ctx,
		"INSERT INTO semesters (academic_year, term, is_active) VALUES (?, ?, FALSE)",
		s.AcademicYear, s.Term)
	if errDuplikat(err) {
		return 0, model.ErrDuplikat
	}
	if err != nil {
		return 0, fmt.Errorf("tambah semester: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *SemesterRepo) UbahPeriode(ctx context.Context, s model.Semester) error {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE semesters SET krs_start=?, krs_end=?, approval_start=?, approval_end=?,
		       change_start=?, change_end=? WHERE id=?`,
		nullWaktu(s.KRSStart), nullWaktu(s.KRSEnd),
		nullWaktu(s.ApprovalStart), nullWaktu(s.ApprovalEnd),
		nullWaktu(s.ChangeStart), nullWaktu(s.ChangeEnd), s.ID)
	if err != nil {
		return fmt.Errorf("ubah periode semester %d: %w", s.ID, err)
	}
	_, err = res.RowsAffected()
	return err
}

// JadikanAktif: hanya satu semester yang aktif (dipanggil di dalam transaksi).
func (r *SemesterRepo) JadikanAktif(ctx context.Context, id int) error {
	if _, err := r.DB.ExecContext(ctx, "UPDATE semesters SET is_active = FALSE WHERE is_active = TRUE"); err != nil {
		return err
	}
	res, err := r.DB.ExecContext(ctx, "UPDATE semesters SET is_active = TRUE WHERE id = ?", id)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}
