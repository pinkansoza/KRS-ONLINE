// Tahap 5: daftar peserta satu kelas untuk rekap.

package repository

import (
	"context"
	"fmt"
)

// Peserta = mahasiswa yang memegang kursi di kelas (rekap FR-G1).
type Peserta struct {
	NIM, Nama, Status string
	KRSStatus         string
}

func (r *ClassRepo) Peserta(ctx context.Context, classID int) ([]Peserta, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT s.nim, s.name, ki.status, k.status
		FROM krs_items ki
		JOIN krs k      ON k.id = ki.krs_id
		JOIN students s ON s.id = k.student_id
		WHERE ki.class_id = ? AND ki.status IN `+statusKursi+`
		ORDER BY s.nim`, classID)
	if err != nil {
		return nil, fmt.Errorf("peserta kelas: %w", err)
	}
	defer rows.Close()
	var hasil []Peserta
	for rows.Next() {
		var p Peserta
		if err := rows.Scan(&p.NIM, &p.Nama, &p.Status, &p.KRSStatus); err != nil {
			return nil, err
		}
		hasil = append(hasil, p)
	}
	return hasil, rows.Err()
}
