package repository

import (
	"context"
	"fmt"

	"krs-online/internal/model"
)

// WaitlistRepo: antrean FIFO berdasarkan queued_at (BR-7).
type WaitlistRepo struct {
	DB DBTX
}

func (r *WaitlistRepo) Tambah(ctx context.Context, classID, studentID, krsID int) error {
	_, err := r.DB.ExecContext(ctx,
		"INSERT INTO waitlist_entries (class_id, student_id, krs_id) VALUES (?, ?, ?)", classID, studentID, krsID)
	if errDuplikat(err) {
		return model.ErrDuplikat // unique (student_id, waiting_class_id)
	}
	return err
}

// Posisi = urutan antrean (1 = terdepan) untuk entri yang masih MENUNGGU.
const kolomPosisi = `(SELECT COUNT(*) FROM waitlist_entries w2
	WHERE w2.class_id = w.class_id AND w2.status = 'MENUNGGU'
	  AND (w2.queued_at < w.queued_at OR (w2.queued_at = w.queued_at AND w2.id <= w.id)))`

// MilikMahasiswa mengembalikan antrean seorang mahasiswa pada satu semester.
func (r *WaitlistRepo) MilikMahasiswa(ctx context.Context, studentID, semesterID int) ([]model.WaitlistEntry, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT w.id, w.class_id, w.student_id, w.krs_id, w.queued_at, w.status,
		       IF(w.status = 'MENUNGGU', `+kolomPosisi+`, 0),
		       c.class_code, c.course_id, co.code, co.name, co.credits, c.quota,
		       (SELECT COUNT(*) FROM krs_items ki WHERE ki.class_id = c.id AND ki.status IN `+statusKursi+`),
		       (SELECT COUNT(*) FROM waitlist_entries w3 WHERE w3.class_id = c.id AND w3.status = 'MENUNGGU')
		FROM waitlist_entries w
		JOIN classes c  ON c.id = w.class_id
		JOIN courses co ON co.id = c.course_id
		WHERE w.student_id = ? AND c.semester_id = ?
		ORDER BY w.status = 'MENUNGGU' DESC, w.queued_at DESC`, studentID, semesterID)
	if err != nil {
		return nil, fmt.Errorf("daftar tunggu mahasiswa: %w", err)
	}
	defer rows.Close()
	var hasil []model.WaitlistEntry
	for rows.Next() {
		var w model.WaitlistEntry
		c := &w.Class
		if err := rows.Scan(&w.ID, &w.ClassID, &w.StudentID, &w.KRSID, &w.QueuedAt, &w.Status, &w.Posisi,
			&c.Code, &c.CourseID, &c.CourseCode, &c.CourseName, &c.Credits, &c.Quota, &c.Enrolled, &c.Waitlist); err != nil {
			return nil, err
		}
		c.ID = w.ClassID
		hasil = append(hasil, w)
	}
	return hasil, rows.Err()
}

// Antrean sebuah kelas (rekap admin).
func (r *WaitlistRepo) AntreanKelas(ctx context.Context, classID int) ([]model.WaitlistEntry, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT w.id, w.class_id, w.student_id, w.krs_id, w.queued_at, w.status,
		       IF(w.status = 'MENUNGGU', `+kolomPosisi+`, 0), s.nim, s.name
		FROM waitlist_entries w
		JOIN students s ON s.id = w.student_id
		WHERE w.class_id = ?
		ORDER BY w.status = 'MENUNGGU' DESC, w.queued_at, w.id`, classID)
	if err != nil {
		return nil, fmt.Errorf("antrean kelas: %w", err)
	}
	defer rows.Close()
	var hasil []model.WaitlistEntry
	for rows.Next() {
		var w model.WaitlistEntry
		if err := rows.Scan(&w.ID, &w.ClassID, &w.StudentID, &w.KRSID, &w.QueuedAt, &w.Status, &w.Posisi,
			&w.NIM, &w.Nama); err != nil {
			return nil, err
		}
		hasil = append(hasil, w)
	}
	return hasil, rows.Err()
}

// Terdepan mengambil antrean teratas yang masih MENUNGGU lalu menguncinya.
func (r *WaitlistRepo) Terdepan(ctx context.Context, classID int) (model.WaitlistEntry, error) {
	var w model.WaitlistEntry
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, class_id, student_id, krs_id, queued_at, status FROM waitlist_entries
		WHERE class_id = ? AND status = 'MENUNGGU'
		ORDER BY queued_at, id LIMIT 1 FOR UPDATE`, classID,
	).Scan(&w.ID, &w.ClassID, &w.StudentID, &w.KRSID, &w.QueuedAt, &w.Status)
	return w, notFound(err)
}

func (r *WaitlistRepo) CariByID(ctx context.Context, id int) (model.WaitlistEntry, error) {
	var w model.WaitlistEntry
	err := r.DB.QueryRowContext(ctx,
		"SELECT id, class_id, student_id, krs_id, queued_at, status FROM waitlist_entries WHERE id = ?", id,
	).Scan(&w.ID, &w.ClassID, &w.StudentID, &w.KRSID, &w.QueuedAt, &w.Status)
	return w, notFound(err)
}

func (r *WaitlistRepo) SetStatus(ctx context.Context, id int, dari, ke string) error {
	res, err := r.DB.ExecContext(ctx,
		"UPDATE waitlist_entries SET status = ? WHERE id = ? AND status = ?", ke, id, dari)
	if err != nil {
		return err
	}
	return pastikanAdaBaris(res)
}

// Menunggu: apakah mahasiswa sedang mengantre di salah satu kelas MK tertentu.
func (r *WaitlistRepo) Menunggu(ctx context.Context, studentID, courseID, semesterID int) (bool, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM waitlist_entries w JOIN classes c ON c.id = w.class_id
		WHERE w.student_id = ? AND c.course_id = ? AND c.semester_id = ? AND w.status = 'MENUNGGU'`,
		studentID, courseID, semesterID).Scan(&n)
	return n > 0, err
}

// KeluarkanMK: setelah mahasiswa mendapat kursi di satu kelas MK, antrean
// di kelas paralel lain untuk MK yang sama tidak relevan lagi.
func (r *WaitlistRepo) KeluarkanMK(ctx context.Context, studentID, courseID, semesterID int) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE waitlist_entries w JOIN classes c ON c.id = w.class_id
		SET w.status = 'KELUAR'
		WHERE w.student_id = ? AND c.course_id = ? AND c.semester_id = ? AND w.status = 'MENUNGGU'`,
		studentID, courseID, semesterID)
	return err
}

func (r *WaitlistRepo) LogPromosi(ctx context.Context, entryID, classID, studentID int, hasil, alasan string) error {
	var a any
	if alasan != "" {
		a = potong(alasan, 255)
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO waitlist_promotion_logs (waitlist_entry_id, class_id, student_id, result, reason)
		VALUES (?, ?, ?, ?, ?)`, entryID, classID, studentID, hasil, a)
	return err
}

func (r *WaitlistRepo) RiwayatPromosi(ctx context.Context, semesterID, batas int) ([]model.PromotionLog, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT p.id, p.class_id, CONCAT(co.code, ' ', co.name, ' ', c.class_code), s.nim, s.name,
		       p.result, COALESCE(p.reason, ''), p.created_at
		FROM waitlist_promotion_logs p
		JOIN classes c  ON c.id = p.class_id
		JOIN courses co ON co.id = c.course_id
		JOIN students s ON s.id = p.student_id
		WHERE c.semester_id = ? ORDER BY p.created_at DESC, p.id DESC LIMIT ?`, semesterID, batas)
	if err != nil {
		return nil, fmt.Errorf("riwayat promosi: %w", err)
	}
	defer rows.Close()
	var hasil []model.PromotionLog
	for rows.Next() {
		var p model.PromotionLog
		if err := rows.Scan(&p.ID, &p.ClassID, &p.ClassLbl, &p.NIM, &p.Nama, &p.Result, &p.Reason, &p.CreatedAt); err != nil {
			return nil, err
		}
		hasil = append(hasil, p)
	}
	return hasil, rows.Err()
}
