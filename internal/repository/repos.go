package repository

import "database/sql"

// Repos mengumpulkan semua repository. Dependency dioper lewat struct,
// bukan variabel global.
type Repos struct {
	db *sql.DB // nil kalau Repos ini sedang berada di dalam transaksi

	User     *UserRepo
	Session  *SessionRepo
	Lecturer *LecturerRepo
	Notif    *NotifRepo
}

func bangun(q DBTX) *Repos {
	return &Repos{
		User:     &UserRepo{DB: q},
		Session:  &SessionRepo{DB: q},
		Lecturer: &LecturerRepo{DB: q},
		Notif:    &NotifRepo{DB: q},
	}
}