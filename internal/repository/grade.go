package repository

import (
	"context"
	"fmt"

	"krs-online/internal/model"
)

type GradeRepo struct {
	DB DBTX
}

// Riwayat mengembalikan SEMUA percobaan nilai mahasiswa (termasuk yang diulang).
func (r *GradeRepo) Riwayat(ctx context.Context, studentID int) ([]model.Grade, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT g.id, g.student_id, g.course_id, co.code, co.name, co.credits,
		       g.semester_id, CONCAT(se.academic_year, ' ', IF(se.term = 'GANJIL', 'Ganjil', 'Genap')),
		       g.grade_letter, g.grade_point
		FROM grades g
		JOIN courses co   ON co.id = g.course_id
		JOIN semesters se ON se.id = g.semester_id
		WHERE g.student_id = ?
		ORDER BY se.academic_year, se.term, co.code`, studentID)
	if err != nil {
		return nil, fmt.Errorf("riwayat nilai: %w", err)
	}
	defer rows.Close()
	var hasil []model.Grade
	for rows.Next() {
		var g model.Grade
		if err := rows.Scan(&g.ID, &g.StudentID, &g.CourseID, &g.CourseCode, &g.CourseName, &g.Credits,
			&g.SemesterID, &g.SemesterLabel, &g.Letter, &g.Point); err != nil {
			return nil, err
		}
		hasil = append(hasil, g)
	}
	return hasil, rows.Err()
}

// NilaiTerbaik membaca view v_best_grades: course_id -> bobot terbaik (BR-3).
func (r *GradeRepo) NilaiTerbaik(ctx context.Context, studentID int) (map[int]float64, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT course_id, best_grade_point FROM v_best_grades WHERE student_id = ?", studentID)
	if err != nil {
		return nil, fmt.Errorf("nilai terbaik: %w", err)
	}
	defer rows.Close()
	hasil := map[int]float64{}
	for rows.Next() {
		var id int
		var p float64
		if err := rows.Scan(&id, &p); err != nil {
			return nil, err
		}
		hasil[id] = p
	}
	return hasil, rows.Err()
}

// Simpan memakai upsert: satu nilai per mahasiswa per MK per semester.
func (r *GradeRepo) Simpan(ctx context.Context, g model.Grade) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO grades (student_id, course_id, semester_id, grade_letter, grade_point)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE grade_letter = VALUES(grade_letter), grade_point = VALUES(grade_point)`,
		g.StudentID, g.CourseID, g.SemesterID, g.Letter, g.Point)
	if err != nil {
		return fmt.Errorf("simpan nilai: %w", err)
	}
	return nil
}

func (r *GradeRepo) Hapus(ctx context.Context, studentID, id int) error {
	res, err := r.DB.ExecContext(ctx, "DELETE FROM grades WHERE id = ? AND student_id = ?", id, studentID)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}
