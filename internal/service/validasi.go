// Tahap 4: mesin validasi KRS: duplikat, prasyarat, kesetaraan, bentrok, batas SKS.

package service

import (
	"fmt"
	"strings"

	"krs-online/internal/model"
)

// Kode hasil pemeriksaan sebuah kelas untuk seorang mahasiswa.
const (
	CekOK        = "OK"
	CekDiKRS     = "DI_KRS"
	CekLulus     = "LULUS"
	CekPrasyarat = "PRASYARAT"
	CekBentrok   = "BENTROK"
	CekSKS       = "SKS"
	CekMenunggu  = "MENUNGGU"
)

// DataValidasi berisi semua data yang dibutuhkan untuk memvalidasi KRS seorang
// mahasiswa. Strukturnya murni (tanpa database) sehingga aturan bisnis bisa
// diuji tanpa server dan tanpa database (lihat validasi_test.go).
type DataValidasi struct {
	Terbaik    map[int]float64             // course_id -> bobot terbaik (BR-3)
	Setara     map[int][]int               // course_id -> daftar MK setara, dua arah (BR-2)
	Prasyarat  map[int][]model.PrereqGroup // course_id -> grup prasyarat (BR-1)
	Items      []model.KRSItem             // item KRS yang memegang kursi
	MaxSKS     int                         // BR-5
	NamaMK     map[int]string              // untuk pesan error
	Menunggu   map[int]bool                // course_id yang sedang diantrekan
	LabelSkala func(float64) string
}

// Efektif = bobot terbaik MK itu sendiri atau MK yang setara dengannya.
// Mengembalikan -1 kalau belum pernah diambil.
func (d DataValidasi) Efektif(courseID int) (float64, int) {
	terbaik, dari := -1.0, 0
	if p, ok := d.Terbaik[courseID]; ok {
		terbaik, dari = p, courseID
	}
	for _, s := range d.Setara[courseID] {
		if p, ok := d.Terbaik[s]; ok && p > terbaik {
			terbaik, dari = p, s
		}
	}
	return terbaik, dari
}

func (d DataValidasi) TotalSKS() int {
	total := 0
	for _, it := range d.Items {
		if it.Hidup() {
			total += it.Class.Credits
		}
	}
	return total
}

// Tanpa mengembalikan salinan data dengan satu item dikeluarkan.
// Dipakai saat validasi ulang menyeluruh sebelum KRS diajukan (FR-D5).
func (d DataValidasi) Tanpa(itemID int) DataValidasi {
	salinan := d
	salinan.Items = nil
	for _, it := range d.Items {
		if it.ID != itemID {
			salinan.Items = append(salinan.Items, it)
		}
	}
	return salinan
}

func (d DataValidasi) huruf(p float64) string {
	if d.LabelSkala != nil {
		return d.LabelSkala(p)
	}
	return fmt.Sprintf("%.2f", p)
}

// Periksa menjalankan validasi berurutan sesuai FR-D2:
// (1) belum lulus / belum di KRS, (2) prasyarat, (3) bentrok, (4) batas SKS.
// Kuota (5) diperiksa terpisah di dalam transaksi karena butuh penguncian baris.
func (d DataValidasi) Periksa(k model.Class) (kode, pesan string) {
	// (1a) BR-10: satu kelas per MK
	for _, it := range d.Items {
		if it.Hidup() && it.CourseID == k.CourseID {
			return CekDiKRS, fmt.Sprintf("Sudah mengambil %s kelas %s di KRS ini. Batalkan dulu untuk pindah kelas.",
				it.Class.CourseName, it.Class.Code)
		}
	}
	if d.Menunggu[k.CourseID] {
		return CekMenunggu, fmt.Sprintf("Anda sedang mengantre di daftar tunggu %s. Keluar dulu dari antrean untuk memilih kelas lain.", k.CourseName)
	}
	// (1b) sudah lulus (termasuk lewat MK setara)
	if p, dari := d.Efektif(k.CourseID); p >= BatasLulus {
		if dari == k.CourseID {
			return CekLulus, fmt.Sprintf("Mata kuliah sudah lulus dengan nilai %s.", d.huruf(p))
		}
		return CekLulus, fmt.Sprintf("Mata kuliah sudah lulus: %s setara dengan %s (nilai %s).",
			d.NamaMK[dari], k.CourseName, d.huruf(p))
	}
	// (2) prasyarat: semua grup (AND), tiap grup minimal satu butir (OR)
	var gagal []string
	for _, g := range d.Prasyarat[k.CourseID] {
		if len(g.Items) == 0 {
			continue
		}
		terpenuhi := false
		var opsi []string
		for _, it := range g.Items {
			if p, _ := d.Efektif(it.PrereqCourseID); p >= it.MinPoint {
				terpenuhi = true
				break
			}
			opsi = append(opsi, fmt.Sprintf("%s min %s", it.Name, it.MinLetter))
		}
		if !terpenuhi {
			gagal = append(gagal, strings.Join(opsi, " atau "))
		}
	}
	if len(gagal) > 0 {
		return CekPrasyarat, "Prasyarat belum terpenuhi: " + strings.Join(gagal, "; ")
	}
	// (3) BR-4 bentrok jadwal
	for _, it := range d.Items {
		if !it.Hidup() {
			continue
		}
		for _, a := range k.Schedules {
			for _, b := range it.Class.Schedules {
				if a.Bentrok(b) {
					return CekBentrok, fmt.Sprintf("Bentrok dengan %s kelas %s, %s %s–%s.",
						it.Class.CourseName, it.Class.Code, NamaHari(b.Day), b.Start, b.End)
				}
			}
		}
	}
	// (4) BR-5 batas SKS
	if total := d.TotalSKS() + k.Credits; total > d.MaxSKS {
		return CekSKS, fmt.Sprintf("Total SKS akan menjadi %d, melebihi batas %d SKS.", total, d.MaxSKS)
	}
	return CekOK, ""
}
