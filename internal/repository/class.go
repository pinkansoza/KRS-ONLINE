// Tahap 3: kelas paralel, jadwal, dan kuota.

package repository

import (
	"context"
	"fmt"
	"strings"

	"krs-online/internal/model"
)

type ClassRepo struct {
	DB DBTX
}

const kolomClass = `c.id, c.semester_id, c.course_id, c.class_code, co.code, co.name, co.credits,
	c.lecturer_id, l.name, c.quota, COALESCE(c.room_default, ''),
	(SELECT COUNT(*) FROM krs_items ki WHERE ki.class_id = c.id AND ki.status IN ` + statusKursi + `),
	(SELECT COUNT(*) FROM waitlist_entries w WHERE w.class_id = c.id AND w.status = 'MENUNGGU')`

const joinClass = ` FROM classes c
	JOIN courses co  ON co.id = c.course_id
	JOIN lecturers l ON l.id = c.lecturer_id `

// Status item yang memegang kursi (kursi "dipesan" sejak KRS draft).
const statusKursi = "('AKTIF','MENUNGGU_TAMBAH','MENUNGGU_BATAL')"

func scanClass(sc interface{ Scan(...any) error }) (model.Class, error) {
	var c model.Class
	err := sc.Scan(&c.ID, &c.SemesterID, &c.CourseID, &c.Code, &c.CourseCode, &c.CourseName, &c.Credits,
		&c.LecturerID, &c.LecturerName, &c.Quota, &c.Room, &c.Enrolled, &c.Waitlist)
	return c, err
}

// DaftarSemester mengembalikan semua kelas dalam satu semester, lengkap dengan jadwalnya.
func (r *ClassRepo) DaftarSemester(ctx context.Context, semesterID int, q string) ([]model.Class, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT "+kolomClass+joinClass+`
		WHERE c.semester_id = ? AND (co.code LIKE ? OR co.name LIKE ? OR l.name LIKE ?)
		ORDER BY co.code, c.class_code`, semesterID, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	if err != nil {
		return nil, fmt.Errorf("daftar kelas: %w", err)
	}
	var hasil []model.Class
	for rows.Next() {
		c, err := scanClass(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		hasil = append(hasil, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hasil, r.isiJadwal(ctx, hasil)
}

func (r *ClassRepo) CariByID(ctx context.Context, id int) (model.Class, error) {
	c, err := scanClass(r.DB.QueryRowContext(ctx, "SELECT "+kolomClass+joinClass+"WHERE c.id = ?", id))
	if err != nil {
		return c, notFound(err)
	}
	list := []model.Class{c}
	err = r.isiJadwal(ctx, list)
	return list[0], err
}

// Kunci mengunci baris kelas (SELECT ... FOR UPDATE) agar dua mahasiswa yang
// mengambil kursi terakhir bersamaan tidak membuat kelas overbooked.
// Wajib dipanggil di dalam transaksi.
func (r *ClassRepo) Kunci(ctx context.Context, id int) (model.Class, error) {
	var dummy int
	if err := r.DB.QueryRowContext(ctx, "SELECT id FROM classes WHERE id = ? FOR UPDATE", id).Scan(&dummy); err != nil {
		return model.Class{}, notFound(err)
	}
	return r.CariByID(ctx, id)
}

// isiJadwal mengambil jadwal untuk sekumpulan kelas dalam satu query.
func (r *ClassRepo) isiJadwal(ctx context.Context, kelas []model.Class) error {
	if len(kelas) == 0 {
		return nil
	}
	idx := map[int]int{}
	args := make([]any, len(kelas))
	for i, c := range kelas {
		idx[c.ID] = i
		args[i] = c.ID
	}
	// Placeholder dibuat sebanyak jumlah id: "?,?,?" (tetap berparameter, bukan string user)
	ph := strings.TrimSuffix(strings.Repeat("?,", len(kelas)), ",")
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, class_id, day_of_week, start_time, end_time, COALESCE(room, '')
		FROM class_schedules WHERE class_id IN (`+ph+`) ORDER BY day_of_week, start_time`, args...)
	if err != nil {
		return fmt.Errorf("jadwal kelas: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s model.Schedule
		if err := rows.Scan(&s.ID, &s.ClassID, &s.Day, &s.Start, &s.End, &s.Room); err != nil {
			return err
		}
		s.Start, s.End = jam(s.Start), jam(s.End)
		i := idx[s.ClassID]
		kelas[i].Schedules = append(kelas[i].Schedules, s)
	}
	return rows.Err()
}

func (r *ClassRepo) Tambah(ctx context.Context, c model.Class) (int, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO classes (semester_id, course_id, class_code, lecturer_id, quota, room_default)
		VALUES (?, ?, ?, ?, ?, ?)`, c.SemesterID, c.CourseID, c.Code, c.LecturerID, c.Quota, c.Room)
	if errDuplikat(err) {
		return 0, model.ErrDuplikat
	}
	if err != nil {
		return 0, fmt.Errorf("tambah kelas: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *ClassRepo) Ubah(ctx context.Context, c model.Class) error {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE classes SET class_code=?, lecturer_id=?, quota=?, room_default=? WHERE id=?`,
		c.Code, c.LecturerID, c.Quota, c.Room, c.ID)
	if errDuplikat(err) {
		return model.ErrDuplikat
	}
	if err != nil {
		return fmt.Errorf("ubah kelas %d: %w", c.ID, err)
	}
	_, err = res.RowsAffected()
	return err
}

// Hapus permanen (hard delete). Gagal jika kelas sudah dipakai KRS/daftar tunggu.
func (r *ClassRepo) Hapus(ctx context.Context, id int) error {
	res, err := r.DB.ExecContext(ctx, "DELETE FROM classes WHERE id = ?", id)
	if errDipakai(err) {
		return model.ErrDuplikat
	}
	if err != nil {
		return fmt.Errorf("hapus kelas %d: %w", id, err)
	}
	return pastikanAdaBaris(res)
}

func (r *ClassRepo) TambahJadwal(ctx context.Context, s model.Schedule) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO class_schedules (class_id, day_of_week, start_time, end_time, room) VALUES (?, ?, ?, ?, ?)`,
		s.ClassID, s.Day, s.Start, s.End, s.Room)
	return err
}

func (r *ClassRepo) HapusJadwal(ctx context.Context, classID, scheduleID int) error {
	res, err := r.DB.ExecContext(ctx, "DELETE FROM class_schedules WHERE id = ? AND class_id = ?", scheduleID, classID)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}
