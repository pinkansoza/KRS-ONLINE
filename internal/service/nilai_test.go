// Tahap 3: uji perhitungan IPS dan nilai terbaik tanpa database.

package service

import (
	"testing"

	"krs-online/internal/model"
)

// Jalankan: go test ./internal/service/

func TestRingkasNilaiNilaiTerbaik(t *testing.T) {
	riwayat := []model.Grade{
		{CourseID: 6, CourseCode: "IF301", Credits: 3, SemesterID: 3, SemesterLabel: "S3", Letter: "E", Point: 0},
		{CourseID: 5, CourseCode: "IF201", Credits: 3, SemesterID: 3, SemesterLabel: "S3", Letter: "C", Point: 2},
		{CourseID: 6, CourseCode: "IF301", Credits: 3, SemesterID: 4, SemesterLabel: "S4", Letter: "B", Point: 3},
		{CourseID: 7, CourseCode: "IF401", Credits: 3, SemesterID: 4, SemesterLabel: "S4", Letter: "C", Point: 2},
	}
	tr := RingkasNilai(riwayat, 99)
	if tr.IPSTerakhir != 2.5 || tr.LabelTerakhir != "S4" {
		t.Fatalf("IPS S4 = (9+6)/6 = 2.50, dapat %.2f %s", tr.IPSTerakhir, tr.LabelTerakhir)
	}
	if tr.PerSemester[0].IPS != 1 {
		t.Fatalf("IPS S3 = 6/6 = 1.00, dapat %.2f", tr.PerSemester[0].IPS)
	}
	// IPK memakai nilai terbaik: (3*3 + 3*2 + 3*2) / 9 = 2.33
	if tr.IPK != 2.33 || tr.TotalSKS != 9 {
		t.Fatalf("IPK = 2.33 dari 9 SKS, dapat %.2f dari %d", tr.IPK, tr.TotalSKS)
	}
}
