// Tahap 4: KRS, item KRS, dan log persetujuan.

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"krs-online/internal/model"
)

type KRSRepo struct {
	DB DBTX
}

const kolomKRS = `k.id, k.student_id, k.semester_id, k.status, COALESCE(k.max_credits_snapshot, 0),
	k.ips_reference, k.submitted_at, k.approved_at`

func scanKRS(sc interface{ Scan(...any) error }) (model.KRS, error) {
	var k model.KRS
	var ips sql.NullFloat64
	var sub, appr sql.NullTime
	err := sc.Scan(&k.ID, &k.StudentID, &k.SemesterID, &k.Status, &k.MaxCredits, &ips, &sub, &appr)
	k.IPSRef, k.AdaIPS = ips.Float64, ips.Valid
	k.SubmittedAt, k.ApprovedAt = waktu(sub), waktu(appr)
	return k, err
}

func (r *KRSRepo) CariMilik(ctx context.Context, studentID, semesterID int) (model.KRS, error) {
	k, err := scanKRS(r.DB.QueryRowContext(ctx,
		"SELECT "+kolomKRS+" FROM krs k WHERE k.student_id = ? AND k.semester_id = ?", studentID, semesterID))
	return k, notFound(err)
}

func (r *KRSRepo) CariByID(ctx context.Context, id int) (model.KRS, error) {
	k, err := scanKRS(r.DB.QueryRowContext(ctx, "SELECT "+kolomKRS+" FROM krs k WHERE k.id = ?", id))
	return k, notFound(err)
}

// Kunci baris KRS agar dua request dari mahasiswa yang sama (mis. klik ganda)
// tidak sama-sama lolos cek batas SKS.
func (r *KRSRepo) Kunci(ctx context.Context, id int) (model.KRS, error) {
	k, err := scanKRS(r.DB.QueryRowContext(ctx, "SELECT "+kolomKRS+" FROM krs k WHERE k.id = ? FOR UPDATE", id))
	return k, notFound(err)
}

// Buat KRS draft. ips < 0 berarti mahasiswa baru (tanpa IPS acuan).
func (r *KRSRepo) Buat(ctx context.Context, studentID, semesterID, maxSKS int, ips float64, adaIPS bool) error {
	var ipsArg any
	if adaIPS {
		ipsArg = ips
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO krs (student_id, semester_id, status, max_credits_snapshot, ips_reference)
		VALUES (?, ?, 'DRAFT', ?, ?)`, studentID, semesterID, maxSKS, ipsArg)
	if errDuplikat(err) {
		return nil // sudah dibuat request lain bersamaan
	}
	return err
}

func (r *KRSRepo) SetSnapshot(ctx context.Context, id, maxSKS int, ips float64, adaIPS bool) error {
	var ipsArg any
	if adaIPS {
		ipsArg = ips
	}
	_, err := r.DB.ExecContext(ctx,
		"UPDATE krs SET max_credits_snapshot = ?, ips_reference = ? WHERE id = ?", maxSKS, ipsArg, id)
	return err
}

// SetStatus mengubah status KRS. Tanggal diisi sesuai status tujuan.
func (r *KRSRepo) SetStatus(ctx context.Context, id int, dari, ke string) error {
	q := "UPDATE krs SET status = ?"
	switch ke {
	case model.KRSDiajukan:
		q += ", submitted_at = NOW()"
	case model.KRSDisetujui:
		q += ", approved_at = NOW()"
	case model.KRSDraft:
		q += ", submitted_at = NULL"
	}
	res, err := r.DB.ExecContext(ctx, q+" WHERE id = ? AND status = ?", ke, id, dari)
	if err != nil {
		return fmt.Errorf("ubah status krs %d: %w", id, err)
	}
	return pastikanAdaBaris(res)
}

// Items mengembalikan item KRS beserta info kelas & jadwal.
// semuaStatus=false hanya item yang masih memegang kursi.
func (r *KRSRepo) Items(ctx context.Context, krsID int, semuaStatus bool) ([]model.KRSItem, error) {
	q := `SELECT ki.id, ki.krs_id, ki.class_id, ki.course_id, ki.status, ki.added_at, ` + kolomClass +
		joinClass + ` JOIN krs_items ki ON ki.class_id = c.id WHERE ki.krs_id = ?`
	if !semuaStatus {
		q += " AND ki.status IN " + statusKursi
	}
	rows, err := r.DB.QueryContext(ctx, q+" ORDER BY co.code", krsID)
	if err != nil {
		return nil, fmt.Errorf("item krs: %w", err)
	}
	var hasil []model.KRSItem
	for rows.Next() {
		var it model.KRSItem
		c := &it.Class
		if err := rows.Scan(&it.ID, &it.KRSID, &it.ClassID, &it.CourseID, &it.Status, &it.AddedAt,
			&c.ID, &c.SemesterID, &c.CourseID, &c.Code, &c.CourseCode, &c.CourseName, &c.Credits,
			&c.LecturerID, &c.LecturerName, &c.Quota, &c.Room, &c.Enrolled, &c.Waitlist); err != nil {
			rows.Close()
			return nil, err
		}
		hasil = append(hasil, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// isi jadwal semua kelas sekaligus
	kelas := make([]model.Class, len(hasil))
	for i := range hasil {
		kelas[i] = hasil[i].Class
	}
	if err := (&ClassRepo{DB: r.DB}).isiJadwal(ctx, kelas); err != nil {
		return nil, err
	}
	for i := range hasil {
		hasil[i].Class = kelas[i]
	}
	return hasil, nil
}

func (r *KRSRepo) Item(ctx context.Context, itemID int) (model.KRSItem, error) {
	var it model.KRSItem
	err := r.DB.QueryRowContext(ctx,
		"SELECT id, krs_id, class_id, course_id, status, added_at FROM krs_items WHERE id = ? FOR UPDATE", itemID,
	).Scan(&it.ID, &it.KRSID, &it.ClassID, &it.CourseID, &it.Status, &it.AddedAt)
	return it, notFound(err)
}

func (r *KRSRepo) TambahItem(ctx context.Context, krsID int, kelas model.Class, status string) (int, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO krs_items (krs_id, class_id, course_id, semester_id, status) VALUES (?, ?, ?, ?, ?)`,
		krsID, kelas.ID, kelas.CourseID, kelas.SemesterID, status)
	if errDuplikat(err) {
		return 0, model.ErrDuplikat // BR-10: satu kelas per MK dalam satu KRS
	}
	if err != nil {
		return 0, fmt.Errorf("tambah item krs: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *KRSRepo) SetStatusItem(ctx context.Context, itemID int, dari, ke string) error {
	res, err := r.DB.ExecContext(ctx,
		"UPDATE krs_items SET status = ? WHERE id = ? AND status = ?", ke, itemID, dari)
	if err != nil {
		return fmt.Errorf("ubah status item %d: %w", itemID, err)
	}
	return pastikanAdaBaris(res)
}

func (r *KRSRepo) TambahLog(ctx context.Context, krsID, itemID, actorID int, aksi, catatan string) error {
	var note any
	if catatan != "" {
		note = catatan
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO krs_approval_logs (krs_id, krs_item_id, actor_id, action, note) VALUES (?, ?, ?, ?, ?)`,
		krsID, nullInt(itemID), actorID, aksi, note)
	return err
}

func (r *KRSRepo) Log(ctx context.Context, krsID int) ([]model.ApprovalLog, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT lg.id, lg.krs_id, COALESCE(lg.krs_item_id, 0),
		       COALESCE(CONCAT(co.code, ' ', co.name, ' kelas ', c.class_code), ''),
		       COALESCE(s.name, l.name, 'Admin Prodi'), lg.action, COALESCE(lg.note, ''), lg.created_at
		FROM krs_approval_logs lg
		JOIN users u ON u.id = lg.actor_id
		LEFT JOIN students  s ON s.user_id = u.id
		LEFT JOIN lecturers l ON l.user_id = u.id
		LEFT JOIN krs_items ki ON ki.id = lg.krs_item_id
		LEFT JOIN classes c    ON c.id = ki.class_id
		LEFT JOIN courses co   ON co.id = c.course_id
		WHERE lg.krs_id = ? ORDER BY lg.created_at DESC, lg.id DESC`, krsID)
	if err != nil {
		return nil, fmt.Errorf("log krs: %w", err)
	}
	defer rows.Close()
	var hasil []model.ApprovalLog
	for rows.Next() {
		var l model.ApprovalLog
		if err := rows.Scan(&l.ID, &l.KRSID, &l.ItemID, &l.ItemLabel, &l.ActorName, &l.Action, &l.Note, &l.CreatedAt); err != nil {
			return nil, err
		}
		hasil = append(hasil, l)
	}
	return hasil, rows.Err()
}
