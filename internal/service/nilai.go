// Tahap 3: aturan nilai: IPS, nilai terbaik, IPK, dan batas SKS (BR-3, BR-5).

package service

import (
	"context"
	"math"
	"sort"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

// BatasLulus: nilai minimal agar MK dianggap sudah lulus dan tidak boleh diambil
// lagi. Nilai D dan E boleh diulang (keputusan tim, sesuai catatan wireframe).
const BatasLulus = 2.00

// IPSSemester = ringkasan nilai satu semester (FR-C4).
type IPSSemester struct {
	SemesterID int
	Label      string
	SKS        int
	IPS        float64
	Nilai      []model.Grade
}

// BarisTranskrip = nilai efektif (terbaik) per mata kuliah (FR-C3, BR-3).
type BarisTranskrip struct {
	model.Grade
	Mengulang  bool
	Percobaan  []model.Grade
	Keterangan string
}

type Transkrip struct {
	Baris         []BarisTranskrip
	PerSemester   []IPSSemester
	IPK           float64
	TotalSKS      int
	SKSLulus      int
	TotalBobot    float64
	IPSTerakhir   float64
	LabelTerakhir string
	AdaIPS        bool
}

func bulat2(x float64) float64 { return math.Round(x*100) / 100 }

// RingkasNilai menghitung IPS per semester, nilai terbaik per MK, dan IPK.
// Murni tanpa database. riwayat sudah terurut dari semester paling lama.
// kecualiSemester: semester yang tidak dihitung sebagai "IPS terakhir" (semester aktif).
func RingkasNilai(riwayat []model.Grade, kecualiSemester int) Transkrip {
	var t Transkrip

	// IPS per semester (semua percobaan di semester itu dihitung)
	for _, g := range riwayat {
		n := len(t.PerSemester)
		if n == 0 || t.PerSemester[n-1].SemesterID != g.SemesterID {
			t.PerSemester = append(t.PerSemester, IPSSemester{SemesterID: g.SemesterID, Label: g.SemesterLabel})
			n++
		}
		s := &t.PerSemester[n-1]
		s.Nilai = append(s.Nilai, g)
	}
	for i := range t.PerSemester {
		s := &t.PerSemester[i]
		bobot := 0.0
		for _, g := range s.Nilai {
			s.SKS += g.Credits
			bobot += float64(g.Credits) * g.Point
		}
		if s.SKS > 0 {
			s.IPS = bulat2(bobot / float64(s.SKS))
		}
		if s.SemesterID != kecualiSemester {
			t.IPSTerakhir, t.LabelTerakhir, t.AdaIPS = s.IPS, s.Label, true
		}
	}

	// nilai terbaik per MK
	perMK := map[int]*BarisTranskrip{}
	var urutan []int
	for _, g := range riwayat {
		b, ada := perMK[g.CourseID]
		if !ada {
			b = &BarisTranskrip{Grade: g}
			perMK[g.CourseID] = b
			urutan = append(urutan, g.CourseID)
		}
		b.Percobaan = append(b.Percobaan, g)
		if g.Point > b.Point {
			b.Grade = g
		}
	}
	for _, id := range urutan {
		b := perMK[id]
		if len(b.Percobaan) > 1 {
			b.Mengulang = true
			b.Keterangan = "Mengulang; nilai terbaik dipakai"
		}
		t.TotalSKS += b.Credits
		t.TotalBobot += float64(b.Credits) * b.Point
		if b.Point >= BatasLulus {
			t.SKSLulus += b.Credits
		}
		t.Baris = append(t.Baris, *b)
	}
	sort.SliceStable(t.Baris, func(i, j int) bool { return t.Baris[i].CourseCode < t.Baris[j].CourseCode })
	if t.TotalSKS > 0 {
		t.IPK = bulat2(t.TotalBobot / float64(t.TotalSKS))
	}
	return t
}

// BatasSKS = hasil penentuan batas SKS dari IPS semester sebelumnya (BR-5, FR-D6).
type BatasSKS struct {
	IPS    float64
	AdaIPS bool
	Label  string
	Maks   int
}

func hitungBatas(ctx context.Context, r *repository.Repos, studentID, semesterAktif, sksBaru int) (BatasSKS, error) {
	riwayat, err := r.Grade.Riwayat(ctx, studentID)
	if err != nil {
		return BatasSKS{}, err
	}
	t := RingkasNilai(riwayat, semesterAktif)
	if !t.AdaIPS {
		// mahasiswa baru: belum punya IPS, pakai aturan default
		return BatasSKS{Maks: sksBaru, Label: "mahasiswa baru"}, nil
	}
	maks, err := r.Course.BatasSKS(ctx, t.IPSTerakhir)
	if err != nil {
		return BatasSKS{}, err
	}
	return BatasSKS{IPS: t.IPSTerakhir, AdaIPS: true, Label: t.LabelTerakhir, Maks: maks}, nil
}

// labelSkala mengubah bobot menjadi huruf: 2.00 -> "C".
func labelSkala(skala []model.GradeScale) func(float64) string {
	return func(p float64) string {
		for _, s := range skala { // urut dari bobot tertinggi
			if p >= s.Point {
				return s.Letter
			}
		}
		return "E"
	}
}
