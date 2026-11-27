// Tahap 2: data mahasiswa.

package repository

import (
	"context"
	"fmt"

	"krs-online/internal/model"
)

type StudentRepo struct {
	DB DBTX
}

const kolomStudent = `s.id, s.user_id, s.nim, s.name, u.email, s.entry_year, s.curriculum_id, c.name,
	COALESCE(s.advisor_id, 0), COALESCE(l.name, '-'), COALESCE(l.user_id, 0), s.status`

const joinStudent = ` FROM students s
	JOIN users u     ON u.id = s.user_id
	JOIN curricula c ON c.id = s.curriculum_id
	LEFT JOIN lecturers l ON l.id = s.advisor_id `

func scanStudent(sc interface{ Scan(...any) error }) (model.Student, error) {
	var s model.Student
	err := sc.Scan(&s.ID, &s.UserID, &s.NIM, &s.Name, &s.Email, &s.EntryYear, &s.CurriculumID,
		&s.CurriculumName, &s.AdvisorID, &s.AdvisorName, &s.AdvisorUserID, &s.Status)
	return s, err
}

func (r *StudentRepo) CariByID(ctx context.Context, id int) (model.Student, error) {
	s, err := scanStudent(r.DB.QueryRowContext(ctx, "SELECT "+kolomStudent+joinStudent+"WHERE s.id = ?", id))
	return s, notFound(err)
}

func (r *StudentRepo) CariByNIM(ctx context.Context, nim string) (model.Student, error) {
	s, err := scanStudent(r.DB.QueryRowContext(ctx, "SELECT "+kolomStudent+joinStudent+"WHERE s.nim = ?", nim))
	return s, notFound(err)
}

// Daftar dengan pencarian nama/NIM (query string ?q=, materi input dari client).
func (r *StudentRepo) Daftar(ctx context.Context, q string) ([]model.Student, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT "+kolomStudent+joinStudent+"WHERE s.name LIKE ? OR s.nim LIKE ? ORDER BY s.nim",
		"%"+q+"%", "%"+q+"%")
	if err != nil {
		return nil, fmt.Errorf("daftar mahasiswa: %w", err)
	}
	defer rows.Close()
	var hasil []model.Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, s)
	}
	return hasil, rows.Err()
}

func (r *StudentRepo) Tambah(ctx context.Context, s model.Student) (int, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO students (user_id, nim, name, entry_year, curriculum_id, advisor_id, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.UserID, s.NIM, s.Name, s.EntryYear, s.CurriculumID, nullInt(s.AdvisorID), s.Status)
	if errDuplikat(err) {
		return 0, model.ErrDuplikat
	}
	if err != nil {
		return 0, fmt.Errorf("tambah mahasiswa: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *StudentRepo) Ubah(ctx context.Context, s model.Student) error {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE students SET name=?, entry_year=?, curriculum_id=?, advisor_id=?, status=? WHERE id=?`,
		s.Name, s.EntryYear, s.CurriculumID, nullInt(s.AdvisorID), s.Status, s.ID)
	if err != nil {
		return fmt.Errorf("ubah mahasiswa %d: %w", s.ID, err)
	}
	_, err = res.RowsAffected()
	return err
}

// JumlahAktif dipakai rekap status KRS.
func (r *StudentRepo) JumlahAktif(ctx context.Context) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM students WHERE status = 'AKTIF'").Scan(&n)
	return n, err
}
