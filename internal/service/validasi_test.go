package service

import (
	"strings"
	"testing"

	"krs-online/internal/model"
)

// Aturan bisnis diuji tanpa server dan tanpa database (materi pertemuan 10).
// Jalankan: go test ./internal/service/

// ID mata kuliah mengikuti seed.sql
const (
	tif201   = 1 // Sistem Basis Data (kurikulum lama)
	tif302   = 2 // Pemrograman Internet (kurikulum lama)
	ifAlgo   = 4
	ifBD     = 6
	ifWeb    = 7
	ifMob    = 8
	ifBDL    = 9
	ifJarkom = 10
)

func dataDasar() DataValidasi {
	return DataValidasi{
		Terbaik: map[int]float64{},
		Setara: map[int][]int{
			ifBD: {tif201}, tif201: {ifBD},
			ifWeb: {tif302}, tif302: {ifWeb},
		},
		Prasyarat: map[int][]model.PrereqGroup{
			ifBDL: {
				{GroupNo: 1, Items: []model.PrereqItem{{PrereqCourseID: ifBD, Name: "Basis Data", MinPoint: 2, MinLetter: "C"}}},
				{GroupNo: 2, Items: []model.PrereqItem{
					{PrereqCourseID: ifWeb, Name: "Pemrograman Web", MinPoint: 2, MinLetter: "C"},
					{PrereqCourseID: ifMob, Name: "Pemrograman Mobile", MinPoint: 2, MinLetter: "C"},
				}},
			},
		},
		NamaMK:   map[int]string{tif201: "TIF201 Sistem Basis Data", ifBD: "IF301 Basis Data"},
		Menunggu: map[int]bool{},
		MaxSKS:   24,
	}
}

func kelasBDL() model.Class {
	return model.Class{ID: 1, CourseID: ifBDL, CourseName: "Basis Data Lanjut", Code: "A", Credits: 3,
		Schedules: []model.Schedule{{Day: 1, Start: "08:00", End: "10:30"}}}
}

func TestPrasyaratGabungan(t *testing.T) {
	d := dataDasar()
	d.Terbaik[ifBD] = 3 // grup 1 terpenuhi, grup 2 belum
	if kode, pesan := d.Periksa(kelasBDL()); kode != CekPrasyarat || !strings.Contains(pesan, "Pemrograman Web min C atau Pemrograman Mobile min C") {
		t.Fatalf("harus gagal prasyarat grup 2, dapat %s: %s", kode, pesan)
	}
	d.Terbaik[ifMob] = 2 // salah satu butir OR cukup
	if kode, pesan := d.Periksa(kelasBDL()); kode != CekOK {
		t.Fatalf("harus lolos, dapat %s: %s", kode, pesan)
	}
}

func TestNilaiMinimal(t *testing.T) {
	d := dataDasar()
	d.Terbaik[ifBD] = 1 // D, di bawah min C
	d.Terbaik[ifWeb] = 4
	if kode, _ := d.Periksa(kelasBDL()); kode != CekPrasyarat {
		t.Fatalf("nilai D tidak memenuhi min C, dapat %s", kode)
	}
}

func TestKesetaraanKurikulum(t *testing.T) {
	// seperti Dimas: lulus MK kurikulum lama yang setara (BR-2)
	d := dataDasar()
	d.Terbaik[tif201] = 3
	d.Terbaik[tif302] = 3
	if kode, pesan := d.Periksa(kelasBDL()); kode != CekOK {
		t.Fatalf("MK setara harus memenuhi prasyarat, dapat %s: %s", kode, pesan)
	}
	// MK baru yang setara dengan MK lama yang sudah lulus dianggap sudah lulus
	bd := model.Class{CourseID: ifBD, CourseName: "Basis Data", Credits: 3}
	if kode, _ := d.Periksa(bd); kode != CekLulus {
		t.Fatalf("Basis Data harus dianggap lulus lewat TIF201, dapat %s", kode)
	}
}

func TestMengulangBolehJikaNilaiD(t *testing.T) {
	d := dataDasar()
	d.Terbaik[ifJarkom] = 1 // D boleh diulang
	jk := model.Class{CourseID: ifJarkom, CourseName: "Jaringan Komputer", Credits: 3}
	if kode, _ := d.Periksa(jk); kode != CekOK {
		t.Fatalf("nilai D boleh diulang, dapat %s", kode)
	}
	d.Terbaik[ifJarkom] = 2 // C = lulus
	if kode, _ := d.Periksa(jk); kode != CekLulus {
		t.Fatalf("nilai C sudah lulus, dapat %s", kode)
	}
}

func TestBentrokJadwal(t *testing.T) {
	d := dataDasar()
	d.Terbaik[ifBD], d.Terbaik[ifWeb] = 3, 3
	d.Items = []model.KRSItem{{ID: 9, CourseID: 12, Status: model.ItemAktif, Class: model.Class{
		CourseName: "Kecerdasan Buatan", Code: "A", Credits: 3,
		Schedules: []model.Schedule{{Day: 1, Start: "09:00", End: "11:30"}}}}}
	kode, pesan := d.Periksa(kelasBDL())
	if kode != CekBentrok || !strings.Contains(pesan, "Senin 09:00–11:30") {
		t.Fatalf("harus bentrok, dapat %s: %s", kode, pesan)
	}
	// BR-4: jam berurutan bukan bentrok
	d.Items[0].Class.Schedules[0] = model.Schedule{Day: 1, Start: "10:30", End: "13:00"}
	if kode, _ := d.Periksa(kelasBDL()); kode != CekOK {
		t.Fatalf("10:30 setelah 10:30 bukan bentrok, dapat %s", kode)
	}
}

func TestBatasSKS(t *testing.T) {
	d := dataDasar()
	d.Terbaik[ifBD], d.Terbaik[ifWeb] = 3, 3
	d.MaxSKS = 15
	d.Items = []model.KRSItem{{ID: 1, CourseID: 50, Status: model.ItemAktif, Class: model.Class{Credits: 13}}}
	if kode, _ := d.Periksa(kelasBDL()); kode != CekSKS {
		t.Fatalf("13+3 > 15 harus ditolak, dapat %s", kode)
	}
	d.Items[0].Class.Credits = 12
	if kode, _ := d.Periksa(kelasBDL()); kode != CekOK {
		t.Fatalf("12+3 = 15 boleh, dapat %s", kode)
	}
}

func TestSatuKelasPerMK(t *testing.T) {
	d := dataDasar()
	d.Terbaik[ifBD], d.Terbaik[ifWeb] = 3, 3
	d.Items = []model.KRSItem{{ID: 1, CourseID: ifBDL, Status: model.ItemAktif, Class: model.Class{CourseName: "Basis Data Lanjut", Code: "B"}}}
	if kode, _ := d.Periksa(kelasBDL()); kode != CekDiKRS {
		t.Fatalf("BR-10 harus menolak kelas kedua untuk MK sama, dapat %s", kode)
	}
	// item yang sudah dibatalkan tidak dihitung
	d.Items[0].Status = model.ItemDibatalkan
	if kode, _ := d.Periksa(kelasBDL()); kode != CekOK {
		t.Fatalf("item dibatalkan tidak dihitung, dapat %s", kode)
	}
}
