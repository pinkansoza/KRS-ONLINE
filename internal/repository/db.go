package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"

	"krs-online/internal/model"
)

// DBTX dipenuhi oleh *sql.DB maupun *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func New(db *sql.DB) *Repos {
	r := bangun(db)
	r.db = db
	return r
}

// Transaksi menjalankan fn di dalam satu transaksi database.
// Jika fn mengembalikan error, semua perubahan dibatalkan (rollback).
func (r *Repos) Transaksi(ctx context.Context, fn func(tx *Repos) error) error {
	if r.db == nil { // sudah di dalam transaksi
		return fn(r)
	}
	var err error
	for coba := 0; coba < 3; coba++ {
		if err = r.sekali(ctx, fn); !errDeadlock(err) {
			return err
		}
	}
	return err
}

func (r *Repos) sekali(ctx context.Context, fn func(tx *Repos) error) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback() // aman dipanggil walau sudah commit

	if err := fn(bangun(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func errDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && (me.Number == 1213 || me.Number == 1205)
}

func pastikanAdaBaris(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// errDuplikat menerjemahkan error MySQL 1062 (duplicate entry).
func errDuplikat(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// errDipakai = error 1451, baris masih direferensikan foreign key.
func errDipakai(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1451
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrNotFound
	}
	return err
}

func waktu(n sql.NullTime) time.Time {
	if n.Valid {
		return n.Time
	}
	return time.Time{}
}

func nullWaktu(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullInt(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func intNull(n sql.NullInt64) int {
	if n.Valid {
		return int(n.Int64)
	}
	return 0
}

// potong memangkas teks agar muat di kolom VARCHAR tanpa merusak huruf UTF-8.
func potong(s string, maks int) string {
	if len(s) <= maks {
		return s
	}
	for maks > 0 && !utf8.RuneStart(s[maks]) {
		maks--
	}
	return s[:maks]
}

// jam memotong "08:00:00" menjadi "08:00".
func jam(s string) string {
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}