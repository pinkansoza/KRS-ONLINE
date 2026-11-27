// Tahap 2: semester, kurikulum, mata kuliah, prasyarat, kesetaraan, skala nilai, mahasiswa.

package model

import (
	"time"
)

// Semester beserta periode KRS. Waktu kosong (zero) berarti belum diatur.
type Semester struct {
	ID            int
	AcademicYear  string
	Term          string
	IsActive      bool
	KRSStart      time.Time
	KRSEnd        time.Time
	ApprovalStart time.Time
	ApprovalEnd   time.Time
	ChangeStart   time.Time
	ChangeEnd     time.Time
}

func (s Semester) Label() string {
	if s.ID == 0 {
		return "-"
	}
	t := "Ganjil"
	if s.Term == "GENAP" {
		t = "Genap"
	}
	return s.AcademicYear + " " + t
}

func dalam(mulai, selesai, now time.Time) bool {
	return !mulai.IsZero() && !selesai.IsZero() && !now.Before(mulai) && !now.After(selesai)
}

func (s Semester) MasaKRS(now time.Time) bool { return dalam(s.KRSStart, s.KRSEnd, now) }

func (s Semester) MasaPersetujuan(now time.Time) bool {
	return dalam(s.ApprovalStart, s.ApprovalEnd, now)
}

func (s Semester) MasaPerubahan(now time.Time) bool { return dalam(s.ChangeStart, s.ChangeEnd, now) }

type Curriculum struct {
	ID        int
	Name      string
	YearStart int
	IsActive  bool
}

type Course struct {
	ID                  int
	Code                string
	Name                string
	Credits             int
	CurriculumID        int
	CurriculumName      string
	RecommendedSemester int // 0 = tidak diisi
	IsActive            bool
}

func (c Course) Label() string { return c.Code + " " + c.Name }

type PrereqItem struct {
	ID             int
	GroupID        int
	PrereqCourseID int
	Code           string
	Name           string
	MinPoint       float64
	MinLetter      string
}

type PrereqGroup struct {
	ID       int
	CourseID int
	GroupNo  int
	Items    []PrereqItem
}

type Equivalence struct {
	ID      int
	OldID   int
	OldCode string
	OldName string
	NewID   int
	NewCode string
	NewName string
}

type GradeScale struct {
	Letter string
	Point  float64
}

type Student struct {
	ID             int
	UserID         int
	NIM            string
	Name           string
	Email          string
	EntryYear      int
	CurriculumID   int
	CurriculumName string
	AdvisorID      int
	AdvisorName    string
	AdvisorUserID  int
	Status         string
}
