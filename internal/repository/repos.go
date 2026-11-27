// Daftar repository yang dimiliki aplikasi. File ini bertambah di setiap tahap:
// tiap tahap yang membuat repository baru juga mendaftarkannya di sini.

package repository

import "database/sql"

// Repos mengumpulkan semua repository. Dependency dioper lewat struct,
// bukan variabel global.
type Repos struct {
	db *sql.DB // nil kalau Repos ini sedang berada di dalam transaksi

	User     *UserRepo
	Session  *SessionRepo
	Semester *SemesterRepo
	Course   *CourseRepo
	Student  *StudentRepo
	Lecturer *LecturerRepo
	Notif    *NotifRepo
}

func bangun(q DBTX) *Repos {
	return &Repos{
		User:     &UserRepo{DB: q},
		Session:  &SessionRepo{DB: q},
		Semester: &SemesterRepo{DB: q},
		Course:   &CourseRepo{DB: q},
		Student:  &StudentRepo{DB: q},
		Lecturer: &LecturerRepo{DB: q},
		Notif:    &NotifRepo{DB: q},
	}
}
