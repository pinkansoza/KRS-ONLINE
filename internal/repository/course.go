// Tahap 2: mata kuliah, kurikulum, grup prasyarat, kesetaraan, skala nilai.

package repository

import (
	"context"
	"fmt"

	"krs-online/internal/model"
)

type CourseRepo struct {
	DB DBTX
}

const kolomCourse = `co.id, co.code, co.name, co.credits, co.curriculum_id, cu.name,
	COALESCE(co.recommended_semester, 0), co.is_active`

func scanCourse(sc interface{ Scan(...any) error }) (model.Course, error) {
	var c model.Course
	err := sc.Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.CurriculumID, &c.CurriculumName,
		&c.RecommendedSemester, &c.IsActive)
	return c, err
}

// Daftar mata kuliah dengan filter kata kunci dan kurikulum (0 = semua).
func (r *CourseRepo) Daftar(ctx context.Context, q string, kurikulum int, hanyaAktif bool) ([]model.Course, error) {
	query := "SELECT " + kolomCourse + ` FROM courses co JOIN curricula cu ON cu.id = co.curriculum_id
		WHERE (co.code LIKE ? OR co.name LIKE ?) AND (? = 0 OR co.curriculum_id = ?)`
	if hanyaAktif {
		query += " AND co.is_active = TRUE"
	}
	query += " ORDER BY cu.year_start DESC, co.code"
	rows, err := r.DB.QueryContext(ctx, query, "%"+q+"%", "%"+q+"%", kurikulum, kurikulum)
	if err != nil {
		return nil, fmt.Errorf("daftar mata kuliah: %w", err)
	}
	defer rows.Close()
	var hasil []model.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, c)
	}
	return hasil, rows.Err()
}

func (r *CourseRepo) CariByID(ctx context.Context, id int) (model.Course, error) {
	c, err := scanCourse(r.DB.QueryRowContext(ctx, "SELECT "+kolomCourse+
		" FROM courses co JOIN curricula cu ON cu.id = co.curriculum_id WHERE co.id = ?", id))
	return c, notFound(err)
}

func (r *CourseRepo) CariByKode(ctx context.Context, kode string) (model.Course, error) {
	c, err := scanCourse(r.DB.QueryRowContext(ctx, "SELECT "+kolomCourse+
		" FROM courses co JOIN curricula cu ON cu.id = co.curriculum_id WHERE co.code = ?", kode))
	return c, notFound(err)
}

func (r *CourseRepo) Tambah(ctx context.Context, c model.Course) (int, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO courses (code, name, credits, curriculum_id, recommended_semester, is_active)
		VALUES (?, ?, ?, ?, ?, ?)`,
		c.Code, c.Name, c.Credits, c.CurriculumID, nullInt(c.RecommendedSemester), c.IsActive)
	if errDuplikat(err) {
		return 0, model.ErrDuplikat
	}
	if err != nil {
		return 0, fmt.Errorf("tambah mata kuliah: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *CourseRepo) Ubah(ctx context.Context, c model.Course) error {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE courses SET code=?, name=?, credits=?, curriculum_id=?, recommended_semester=?, is_active=?
		WHERE id=?`,
		c.Code, c.Name, c.Credits, c.CurriculumID, nullInt(c.RecommendedSemester), c.IsActive, c.ID)
	if errDuplikat(err) {
		return model.ErrDuplikat
	}
	if err != nil {
		return fmt.Errorf("ubah mata kuliah %d: %w", c.ID, err)
	}
	_, err = res.RowsAffected()
	return err
}

// Nonaktifkan = soft delete dengan kolom boolean (materi CRUD bagian 2).
// Mata kuliah tidak dihapus permanen karena dipakai riwayat nilai.
func (r *CourseRepo) SetAktif(ctx context.Context, id int, aktif bool) error {
	res, err := r.DB.ExecContext(ctx, "UPDATE courses SET is_active = ? WHERE id = ?", aktif, id)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}

func (r *CourseRepo) Kurikulum(ctx context.Context) ([]model.Curriculum, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id, name, year_start, is_active FROM curricula ORDER BY year_start DESC")
	if err != nil {
		return nil, fmt.Errorf("daftar kurikulum: %w", err)
	}
	defer rows.Close()
	var hasil []model.Curriculum
	for rows.Next() {
		var c model.Curriculum
		if err := rows.Scan(&c.ID, &c.Name, &c.YearStart, &c.IsActive); err != nil {
			return nil, err
		}
		hasil = append(hasil, c)
	}
	return hasil, rows.Err()
}

func (r *CourseRepo) TambahKurikulum(ctx context.Context, c model.Curriculum) error {
	_, err := r.DB.ExecContext(ctx, "INSERT INTO curricula (name, year_start, is_active) VALUES (?, ?, ?)",
		c.Name, c.YearStart, c.IsActive)
	if errDuplikat(err) {
		return model.ErrDuplikat
	}
	return err
}

// SemuaPrasyarat mengembalikan grup prasyarat untuk semua mata kuliah,
// dikelompokkan per course_id. Tabelnya kecil sehingga cukup satu query.
func (r *CourseRepo) SemuaPrasyarat(ctx context.Context) (map[int][]model.PrereqGroup, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT g.id, g.course_id, g.group_no,
		       COALESCE(i.id, 0), COALESCE(i.prereq_course_id, 0), COALESCE(c.code, ''), COALESCE(c.name, ''),
		       COALESCE(i.min_grade_point, 0),
		       COALESCE((SELECT gs.letter FROM grade_scale gs WHERE gs.grade_point <= i.min_grade_point
		                 ORDER BY gs.grade_point DESC LIMIT 1), '')
		FROM prereq_groups g
		LEFT JOIN prereq_items i ON i.group_id = g.id
		LEFT JOIN courses c      ON c.id = i.prereq_course_id
		ORDER BY g.course_id, g.group_no, c.code`)
	if err != nil {
		return nil, fmt.Errorf("daftar prasyarat: %w", err)
	}
	defer rows.Close()

	hasil := map[int][]model.PrereqGroup{}
	for rows.Next() {
		var g model.PrereqGroup
		var it model.PrereqItem
		if err := rows.Scan(&g.ID, &g.CourseID, &g.GroupNo, &it.ID, &it.PrereqCourseID,
			&it.Code, &it.Name, &it.MinPoint, &it.MinLetter); err != nil {
			return nil, err
		}
		grup := hasil[g.CourseID]
		if n := len(grup); n == 0 || grup[n-1].ID != g.ID {
			grup = append(grup, g)
		}
		if it.ID != 0 {
			it.GroupID = g.ID
			grup[len(grup)-1].Items = append(grup[len(grup)-1].Items, it)
		}
		hasil[g.CourseID] = grup
	}
	return hasil, rows.Err()
}

func (r *CourseRepo) TambahGrup(ctx context.Context, courseID int) (int, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO prereq_groups (course_id, group_no)
		SELECT ?, COALESCE(MAX(group_no), 0) + 1 FROM prereq_groups WHERE course_id = ?`, courseID, courseID)
	if err != nil {
		return 0, fmt.Errorf("tambah grup prasyarat: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *CourseRepo) HapusGrup(ctx context.Context, courseID, groupID int) error {
	res, err := r.DB.ExecContext(ctx, "DELETE FROM prereq_groups WHERE id = ? AND course_id = ?", groupID, courseID)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}

func (r *CourseRepo) GrupMilik(ctx context.Context, groupID int) (int, error) {
	var courseID int
	err := r.DB.QueryRowContext(ctx, "SELECT course_id FROM prereq_groups WHERE id = ?", groupID).Scan(&courseID)
	return courseID, notFound(err)
}

func (r *CourseRepo) TambahButir(ctx context.Context, groupID, prereqID int, minPoint float64) error {
	_, err := r.DB.ExecContext(ctx,
		"INSERT INTO prereq_items (group_id, prereq_course_id, min_grade_point) VALUES (?, ?, ?)",
		groupID, prereqID, minPoint)
	if errDuplikat(err) {
		return model.ErrDuplikat
	}
	return err
}

func (r *CourseRepo) HapusButir(ctx context.Context, itemID int) error {
	res, err := r.DB.ExecContext(ctx, "DELETE FROM prereq_items WHERE id = ?", itemID)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}

func (r *CourseRepo) Kesetaraan(ctx context.Context) ([]model.Equivalence, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT e.id, o.id, o.code, o.name, n.id, n.code, n.name
		FROM course_equivalences e
		JOIN courses o ON o.id = e.old_course_id
		JOIN courses n ON n.id = e.new_course_id
		ORDER BY n.code`)
	if err != nil {
		return nil, fmt.Errorf("daftar kesetaraan: %w", err)
	}
	defer rows.Close()
	var hasil []model.Equivalence
	for rows.Next() {
		var e model.Equivalence
		if err := rows.Scan(&e.ID, &e.OldID, &e.OldCode, &e.OldName, &e.NewID, &e.NewCode, &e.NewName); err != nil {
			return nil, err
		}
		hasil = append(hasil, e)
	}
	return hasil, rows.Err()
}

func (r *CourseRepo) TambahKesetaraan(ctx context.Context, lama, baru int) error {
	_, err := r.DB.ExecContext(ctx,
		"INSERT INTO course_equivalences (old_course_id, new_course_id) VALUES (?, ?)", lama, baru)
	if errDuplikat(err) {
		return model.ErrDuplikat
	}
	return err
}

func (r *CourseRepo) HapusKesetaraan(ctx context.Context, id int) error {
	res, err := r.DB.ExecContext(ctx, "DELETE FROM course_equivalences WHERE id = ?", id)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}

func (r *CourseRepo) SkalaNilai(ctx context.Context) ([]model.GradeScale, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT letter, grade_point FROM grade_scale ORDER BY grade_point DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hasil []model.GradeScale
	for rows.Next() {
		var g model.GradeScale
		if err := rows.Scan(&g.Letter, &g.Point); err != nil {
			return nil, err
		}
		hasil = append(hasil, g)
	}
	return hasil, rows.Err()
}
