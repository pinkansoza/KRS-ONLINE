package repository

import (
	"context"
	"crypto/rand"
	"fmt"

	"krs-online/internal/model"
)

type UserRepo struct {
	DB DBTX
}

// kolomUser mengambil nama dari tabel students/lecturers sesuai peran.
const kolomUser = `u.id, u.email, u.role,
	COALESCE(s.name, l.name, 'Admin Prodi') AS nama,
	COALESCE(s.id, 0), COALESCE(l.id, 0)`

// Tambah membuat akun baru. Email UNIQUE dijamin oleh database.
func (r *UserRepo) Tambah(ctx context.Context, email, hash, role string) (int, error) {
	res, err := r.DB.ExecContext(ctx,
		"INSERT INTO users (email, password_hash, role) VALUES (?, ?, ?)", email, hash, role)
	if errDuplikat(err) {
		return 0, model.ErrEmailTerpakai
	}
	if err != nil {
		return 0, fmt.Errorf("tambah user: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// HashByEmail hanya dipakai saat login. Akun nonaktif dianggap tidak ada.
func (r *UserRepo) HashByEmail(ctx context.Context, email string) (int, string, error) {
	var id int
	var hash string
	err := r.DB.QueryRowContext(ctx,
		"SELECT id, password_hash FROM users WHERE email = ? AND is_active = TRUE", email,
	).Scan(&id, &hash)
	return id, hash, notFound(err)
}

func (r *UserRepo) HashByID(ctx context.Context, id int) (string, error) {
	var hash string
	err := r.DB.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = ?", id).Scan(&hash)
	return hash, notFound(err)
}

func (r *UserRepo) UbahPassword(ctx context.Context, id int, hash string) error {
	res, err := r.DB.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ?", hash, id)
	if err != nil {
		return fmt.Errorf("ubah password: %w", err)
	}
	return pastikanAdaBaris(res)
}

func (r *UserRepo) UbahEmail(ctx context.Context, id int, email string) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE users SET email = ? WHERE id = ?", email, id)
	if errDuplikat(err) {
		return model.ErrEmailTerpakai
	}
	return err
}

func (r *UserRepo) SetAktif(ctx context.Context, id int, aktif bool) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE users SET is_active = ? WHERE id = ?", aktif, id)
	return err
}

// ---------------------------------------------------------------------------

type SessionRepo struct {
	DB DBTX
}

// Buat membuat session baru (berlaku 7 hari) beserta token CSRF-nya.
func (r *SessionRepo) Buat(ctx context.Context, userID int) (string, error) {
	token := rand.Text() // crypto/rand, acak (bukan math/rand)
	csrf := rand.Text()
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, csrf_token, expires_at)
		VALUES (?, ?, ?, NOW() + INTERVAL 7 DAY)`, token, userID, csrf)
	return token, err
}

// UserDariToken: session kedaluwarsa otomatis tidak lolos karena expires_at > NOW().
func (r *SessionRepo) UserDariToken(ctx context.Context, token string) (model.User, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT `+kolomUser+`, ss.csrf_token,
		       (SELECT COUNT(*) FROM notifications n WHERE n.user_id = u.id AND n.is_read = FALSE)
		FROM sessions ss
		JOIN users u ON u.id = ss.user_id
		LEFT JOIN students  s ON s.user_id = u.id
		LEFT JOIN lecturers l ON l.user_id = u.id
		WHERE ss.token = ? AND ss.expires_at > NOW() AND u.is_active = TRUE`, token)
	var u model.User
	err := row.Scan(&u.ID, &u.Email, &u.Role, &u.Nama, &u.StudentID, &u.LecturerID, &u.CSRF, &u.NotifBaru)
	return u, notFound(err)
}

func (r *SessionRepo) Hapus(ctx context.Context, token string) error {
	_, err := r.DB.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	return err
}

// HapusMilikUser dipakai setelah ganti password: semua session lama dicabut.
func (r *SessionRepo) HapusMilikUser(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// ---------------------------------------------------------------------------

type NotifRepo struct {
	DB DBTX
}

func (r *NotifRepo) Kirim(ctx context.Context, userID int, pesan string) error {
	if userID == 0 {
		return nil // dosen tanpa akun, lewati
	}
	pesan = potong(pesan, 255)
	_, err := r.DB.ExecContext(ctx, "INSERT INTO notifications (user_id, message) VALUES (?, ?)", userID, pesan)
	return err
}

func (r *NotifRepo) Daftar(ctx context.Context, userID, batas int) ([]model.Notification, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, message, is_read, created_at FROM notifications
		WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`, userID, batas)
	if err != nil {
		return nil, fmt.Errorf("daftar notifikasi: %w", err)
	}
	defer rows.Close()
	var hasil []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		hasil = append(hasil, n)
	}
	return hasil, rows.Err()
}

func (r *NotifRepo) TandaiDibaca(ctx context.Context, userID int) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE notifications SET is_read = TRUE WHERE user_id = ?", userID)
	return err
}