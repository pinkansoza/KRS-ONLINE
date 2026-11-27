// Tahap 3: kelas paralel, jadwal, dan nilai.

package model

import (
	"fmt"
)

type Schedule struct {
	ID      int
	ClassID int
	Day     int    // 1 = Senin ... 7 = Minggu
	Start   string // "08:00"
	End     string // "10:30"
	Room    string
}

// Bentrok: dua sesi beririsan pada hari yang sama (BR-4).
// Jam berurutan (selesai 10:00, mulai 10:00) bukan bentrok.
func (a Schedule) Bentrok(b Schedule) bool {
	return a.Day == b.Day && a.Start < b.End && b.Start < a.End
}

type Class struct {
	ID           int
	SemesterID   int
	CourseID     int
	Code         string // kode kelas: A, B, ...
	CourseCode   string
	CourseName   string
	Credits      int
	LecturerID   int
	LecturerName string
	Quota        int
	Room         string
	Enrolled     int
	Waitlist     int
	Schedules    []Schedule
}

func (c Class) Penuh() bool { return c.Enrolled >= c.Quota }

func (c Class) Label() string {
	return fmt.Sprintf("%s %s kelas %s", c.CourseCode, c.CourseName, c.Code)
}

func (c Class) SisaKursi() int { return c.Quota - c.Enrolled }

type Grade struct {
	ID            int
	StudentID     int
	CourseID      int
	CourseCode    string
	CourseName    string
	Credits       int
	SemesterID    int
	SemesterLabel string
	Letter        string
	Point         float64
}
