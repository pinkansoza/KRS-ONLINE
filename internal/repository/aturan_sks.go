// Tahap 3: tabel aturan batas SKS dari IPS (BR-5).

package repository

import (
	"context"
	"database/sql"

	"krs-online/internal/model"
)

// BatasSKS mencari aturan sks_limit_rules untuk IPS tertentu.
func (r *CourseRepo) BatasSKS(ctx context.Context, ips float64) (int, error) {
	var maks int
	err := r.DB.QueryRowContext(ctx,
		"SELECT max_credits FROM sks_limit_rules WHERE ? BETWEEN min_ips AND max_ips ORDER BY min_ips DESC LIMIT 1",
		ips).Scan(&maks)
	if err == sql.ErrNoRows {
		return 0, model.ErrNotFound
	}
	return maks, err
}

type AturanSKS struct {
	MinIPS, MaxIPS float64
	MaxSKS         int
}

func (r *CourseRepo) AturanSKS(ctx context.Context) ([]AturanSKS, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT min_ips, max_ips, max_credits FROM sks_limit_rules ORDER BY min_ips DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hasil []AturanSKS
	for rows.Next() {
		var a AturanSKS
		if err := rows.Scan(&a.MinIPS, &a.MaxIPS, &a.MaxSKS); err != nil {
			return nil, err
		}
		hasil = append(hasil, a)
	}
	return hasil, rows.Err()
}
