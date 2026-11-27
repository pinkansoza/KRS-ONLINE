package repository

import (
	"context"
	"fmt"

	"krs-online/internal/model"
)

type LecturerRepo struct {
	DB DBTX
}

func (r *LecturerRepo) Semua(ctx context.Context) ([]model.Lecturer, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT l.id, COALESCE(l.user_id, 0), COALESCE(l.nip, ''), l.name, COALESCE(u.email, '') "+
			"FROM lecturers l LEFT JOIN users u ON u.id = l.user_id ORDER BY l.name")
	if err != nil {
		return nil, fmt.Errorf("daftar dosen: %w", err)
	}
	defer rows.Close()
	var hasil []model.Lecturer
	for rows.Next() {
		var l model.Lecturer
		if err := rows.Scan(&l.ID, &l.UserID, &l.NIP, &l.Name, &l.Email); err != nil {
			return nil, err
		}
		hasil = append(hasil, l)
	}
	return hasil, rows.Err()
}

func (r *LecturerRepo) Daftar(ctx context.Context, q string) ([]model.Lecturer, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT l.id, COALESCE(l.user_id, 0), COALESCE(l.nip, ''), l.name, COALESCE(u.email, '')
		FROM lecturers l LEFT JOIN users u ON u.id = l.user_id
		WHERE l.name LIKE ? OR l.nip LIKE ? OR COALESCE(u.email, '') LIKE ?
		ORDER BY l.name`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	if err != nil {
		return nil, fmt.Errorf("daftar dosen: %w", err)
	}
	defer rows.Close()
	var hasil []model.Lecturer
	for rows.Next() {
		var l model.Lecturer
		if err := rows.Scan(&l.ID, &l.UserID, &l.NIP, &l.Name, &l.Email); err != nil {
			return nil, err
		}
		hasil = append(hasil, l)
	}
	return hasil, rows.Err()
}

func (r *LecturerRepo) CariByID(ctx context.Context, id int) (model.Lecturer, error) {
	var l model.Lecturer
	err := r.DB.QueryRowContext(ctx,
		"SELECT l.id, COALESCE(l.user_id, 0), COALESCE(l.nip, ''), l.name, COALESCE(u.email, '') "+
			"FROM lecturers l LEFT JOIN users u ON u.id = l.user_id WHERE l.id = ?", id,
	).Scan(&l.ID, &l.UserID, &l.NIP, &l.Name, &l.Email)
	return l, notFound(err)
}

func (r *LecturerRepo) Tambah(ctx context.Context, l model.Lecturer) (int, error) {
	res, err := r.DB.ExecContext(ctx,
		"INSERT INTO lecturers (user_id, nip, name) VALUES (?, ?, ?)", l.UserID, l.NIP, l.Name)
	if errDuplikat(err) {
		return 0, model.ErrDuplikat
	}
	if err != nil {
		return 0, fmt.Errorf("tambah dosen: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}