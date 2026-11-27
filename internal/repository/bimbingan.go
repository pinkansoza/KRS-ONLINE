// Tahap 5: query KRS untuk dosen wali dan rekap status KRS.

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"krs-online/internal/model"
)

// Bimbingan: KRS mahasiswa bimbingan seorang dosen wali pada satu semester.
// Mahasiswa yang belum membuat KRS tetap tampil (status kosong).
func (r *KRSRepo) Bimbingan(ctx context.Context, lecturerID, semesterID int, status string) ([]model.KRS, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT COALESCE(k.id, 0), s.id, COALESCE(k.status, ''), COALESCE(k.max_credits_snapshot, 0),
		       k.ips_reference, k.submitted_at, s.nim, s.name,
		       COALESCE((SELECT SUM(co.credits) FROM krs_items ki JOIN courses co ON co.id = ki.course_id
		                 WHERE ki.krs_id = k.id AND ki.status IN `+statusKursi+`), 0),
		       COALESCE((SELECT COUNT(*) FROM krs_items ki WHERE ki.krs_id = k.id AND ki.status IN `+statusKursi+`), 0),
		       COALESCE((SELECT COUNT(*) FROM krs_items ki WHERE ki.krs_id = k.id
		                 AND ki.status IN ('MENUNGGU_TAMBAH','MENUNGGU_BATAL')), 0)
		FROM students s
		LEFT JOIN krs k ON k.student_id = s.id AND k.semester_id = ?
		WHERE s.advisor_id = ? AND s.status = 'AKTIF'
		  AND (? = '' OR COALESCE(k.status, 'BELUM') = ?
		       OR (? = 'PERUBAHAN' AND EXISTS (SELECT 1 FROM krs_items ki WHERE ki.krs_id = k.id
		                 AND ki.status IN ('MENUNGGU_TAMBAH','MENUNGGU_BATAL'))))
		ORDER BY FIELD(COALESCE(k.status, ''), 'DIAJUKAN', 'DRAFT', 'DISETUJUI', ''), s.nim`,
		semesterID, lecturerID, status, status, status)
	if err != nil {
		return nil, fmt.Errorf("krs bimbingan: %w", err)
	}
	defer rows.Close()
	var hasil []model.KRS
	for rows.Next() {
		var k model.KRS
		var ips sql.NullFloat64
		var sub sql.NullTime
		if err := rows.Scan(&k.ID, &k.StudentID, &k.Status, &k.MaxCredits, &ips, &sub,
			&k.Student.NIM, &k.Student.Name, &k.TotalSKS, &k.JumlahKelas, &k.PermintaanCt); err != nil {
			return nil, err
		}
		k.IPSRef, k.AdaIPS, k.SubmittedAt = ips.Float64, ips.Valid, waktu(sub)
		hasil = append(hasil, k)
	}
	return hasil, rows.Err()
}

// RekapStatus = FR-G3, dihitung untuk mahasiswa aktif.
func (r *KRSRepo) RekapStatus(ctx context.Context, semesterID int) (model.RekapStatus, error) {
	var rs model.RekapStatus
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*),
		  COALESCE(SUM(k.id IS NULL OR (k.status = 'DRAFT' AND NOT EXISTS
		      (SELECT 1 FROM krs_items ki WHERE ki.krs_id = k.id AND ki.status IN `+statusKursi+`))), 0),
		  COALESCE(SUM(k.status = 'DRAFT' AND EXISTS
		      (SELECT 1 FROM krs_items ki WHERE ki.krs_id = k.id AND ki.status IN `+statusKursi+`)), 0),
		  COALESCE(SUM(k.status = 'DIAJUKAN'), 0),
		  COALESCE(SUM(k.status = 'DISETUJUI'), 0)
		FROM students s
		LEFT JOIN krs k ON k.student_id = s.id AND k.semester_id = ?
		WHERE s.status = 'AKTIF'`, semesterID,
	).Scan(&rs.Total, &rs.BelumMengisi, &rs.Draft, &rs.Diajukan, &rs.Disetujui)
	return rs, err
}
