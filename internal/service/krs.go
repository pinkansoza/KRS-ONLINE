// Tahap 4: pengisian KRS mahasiswa (modul D).

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

// KelasPenuhError dikembalikan saat semua validasi lolos tetapi kuota habis,
// supaya handler bisa menawarkan daftar tunggu (FR-E1).
type KelasPenuhError struct {
	Kelas model.Class
}

func (e *KelasPenuhError) Error() string { return "Kelas " + e.Kelas.Label() + " sudah penuh" }

// mode pengubahan KRS
const (
	modeIsi  = "ISI"  // masa pengisian, KRS draft: langsung AKTIF
	modeUbah = "UBAH" // masa perubahan, KRS disetujui: butuh persetujuan dosen
)

// modeKRS menentukan apakah KRS boleh diubah sekarang (BR-9, BR-11, FR-F5).
func modeKRS(sem model.Semester, k model.KRS, now time.Time) (string, error) {
	switch k.Status {
	case model.KRSDraft:
		if sem.MasaKRS(now) {
			return modeIsi, nil
		}
		return "", salah("Periode pengisian KRS sedang tidak dibuka.")
	case model.KRSDiajukan:
		return "", salah("KRS sedang menunggu persetujuan dosen wali, belum bisa diubah.")
	case model.KRSDisetujui:
		if sem.MasaPerubahan(now) {
			return modeUbah, nil
		}
		return "", salah("KRS sudah disetujui dan terkunci. Perubahan hanya bisa diajukan pada masa perubahan KRS.")
	}
	return "", salah("Status KRS tidak dikenal.")
}

// Konteks = data dasar mahasiswa yang sedang login untuk semester aktif.
type Konteks struct {
	Semester model.Semester
	Mhs      model.Student
	KRS      model.KRS
	Batas    BatasSKS
}

// pastikanKRS mengambil (atau membuat) KRS draft mahasiswa di semester aktif,
// sekaligus menyimpan snapshot batas SKS dan IPS acuannya.
func (s *KRSService) pastikanKRS(ctx context.Context, u model.User) (Konteks, error) {
	var k Konteks
	if !u.IsMahasiswa() || u.StudentID == 0 {
		return k, model.ErrTerlarang
	}
	var err error
	if k.Semester, err = s.R.Semester.Aktif(ctx); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return k, salah("Belum ada semester aktif. Hubungi admin prodi.")
		}
		return k, err
	}
	if k.Mhs, err = s.R.Student.CariByID(ctx, u.StudentID); err != nil {
		return k, err
	}
	if k.Batas, err = hitungBatas(ctx, s.R, k.Mhs.ID, k.Semester.ID, s.SKSBaru); err != nil {
		return k, err
	}
	k.KRS, err = s.R.KRS.CariMilik(ctx, k.Mhs.ID, k.Semester.ID)
	if errors.Is(err, model.ErrNotFound) {
		if k.Mhs.Status != "AKTIF" {
			return k, salah("Status mahasiswa %s, tidak dapat mengisi KRS.", k.Mhs.Status)
		}
		if err = s.R.KRS.Buat(ctx, k.Mhs.ID, k.Semester.ID, k.Batas.Maks, k.Batas.IPS, k.Batas.AdaIPS); err != nil {
			return k, err
		}
		k.KRS, err = s.R.KRS.CariMilik(ctx, k.Mhs.ID, k.Semester.ID)
	}
	if err != nil {
		return k, err
	}
	if k.KRS.MaxCredits == 0 { // snapshot belum ada
		k.KRS.MaxCredits, k.KRS.IPSRef, k.KRS.AdaIPS = k.Batas.Maks, k.Batas.IPS, k.Batas.AdaIPS
		if err := s.R.KRS.SetSnapshot(ctx, k.KRS.ID, k.Batas.Maks, k.Batas.IPS, k.Batas.AdaIPS); err != nil {
			return k, err
		}
	}
	return k, nil
}

// muat mengumpulkan DataValidasi untuk satu mahasiswa & KRS.
// r bisa Repos biasa atau Repos di dalam transaksi.
func muat(ctx context.Context, r *repository.Repos, studentID int, krs model.KRS) (DataValidasi, error) {
	d := DataValidasi{Setara: map[int][]int{}, NamaMK: map[int]string{}, Menunggu: map[int]bool{}, MaxSKS: krs.MaxCredits}
	var err error
	if d.Terbaik, err = r.Grade.NilaiTerbaik(ctx, studentID); err != nil {
		return d, err
	}
	setara, err := r.Course.Kesetaraan(ctx)
	if err != nil {
		return d, err
	}
	for _, e := range setara { // berlaku dua arah
		d.Setara[e.NewID] = append(d.Setara[e.NewID], e.OldID)
		d.Setara[e.OldID] = append(d.Setara[e.OldID], e.NewID)
		d.NamaMK[e.OldID] = e.OldCode + " " + e.OldName
		d.NamaMK[e.NewID] = e.NewCode + " " + e.NewName
	}
	if d.Prasyarat, err = r.Course.SemuaPrasyarat(ctx); err != nil {
		return d, err
	}
	if d.Items, err = r.KRS.Items(ctx, krs.ID, false); err != nil {
		return d, err
	}
	if d.Menunggu, err = antreanMenunggu(ctx, r, studentID, krs.SemesterID); err != nil { // titik sambung
		return d, err
	}
	skala, err := r.Course.SkalaNilai(ctx)
	if err != nil {
		return d, err
	}
	d.LabelSkala = labelSkala(skala)
	return d, nil
}

type DashboardMhs struct {
	Konteks
	Items    []model.KRSItem
	TotalSKS int
	Notif    []model.Notification
	IPK      float64
	Aturan   []repository.AturanSKS
	MasaKRS  bool
	MasaUbah bool
	Langkah  int // 1 isi, 2 diajukan, 3 disetujui, 4 masa perubahan
}

func (s *KRSService) Dashboard(ctx context.Context, u model.User) (DashboardMhs, error) {
	var d DashboardMhs
	var err error
	if d.Konteks, err = s.pastikanKRS(ctx, u); err != nil {
		return d, err
	}
	if d.Items, err = s.R.KRS.Items(ctx, d.KRS.ID, false); err != nil {
		return d, err
	}
	for _, it := range d.Items {
		d.TotalSKS += it.Class.Credits
	}
	if d.Notif, err = s.R.Notif.Daftar(ctx, u.ID, 5); err != nil {
		return d, err
	}
	riwayat, err := s.R.Grade.Riwayat(ctx, d.Mhs.ID)
	if err != nil {
		return d, err
	}
	d.IPK = RingkasNilai(riwayat, d.Semester.ID).IPK
	if d.Aturan, err = s.R.Course.AturanSKS(ctx); err != nil {
		return d, err
	}
	now := s.now()
	d.MasaKRS, d.MasaUbah = d.Semester.MasaKRS(now), d.Semester.MasaPerubahan(now)
	switch {
	case d.KRS.Status == model.KRSDisetujui && d.MasaUbah:
		d.Langkah = 4
	case d.KRS.Status == model.KRSDisetujui:
		d.Langkah = 3
	case d.KRS.Status == model.KRSDiajukan:
		d.Langkah = 2
	default:
		d.Langkah = 1
	}
	return d, nil
}

// BarisKatalog = satu kelas di katalog beserta status untuk mahasiswa (FR-D1).
type BarisKatalog struct {
	Kelas      model.Class
	Kode       string
	Pesan      string
	BisaTambah bool
	BisaTunggu bool
}

type Katalog struct {
	Konteks
	Baris    []BarisKatalog
	TotalSKS int
	Mode     string
	ModeErr  string
	Q        string
	Hari     int
	Status   string
	Sembunyi bool
}

// Katalog menampilkan kelas semester aktif dengan filter dari query string.
func (s *KRSService) Katalog(ctx context.Context, u model.User, q string, hari int, status string, sembunyiLulus bool) (Katalog, error) {
	kt := Katalog{Q: q, Hari: hari, Status: status, Sembunyi: sembunyiLulus}
	var err error
	if kt.Konteks, err = s.pastikanKRS(ctx, u); err != nil {
		return kt, err
	}
	kt.Mode, err = modeKRS(kt.Semester, kt.KRS, s.now())
	if err != nil {
		kt.ModeErr = err.Error()
	}
	data, err := muat(ctx, s.R, kt.Mhs.ID, kt.KRS)
	if err != nil {
		return kt, err
	}
	kt.TotalSKS = data.TotalSKS()
	kelas, err := s.R.Class.DaftarSemester(ctx, kt.Semester.ID, q)
	if err != nil {
		return kt, err
	}
	for _, k := range kelas {
		if hari > 0 && !adaHari(k, hari) {
			continue
		}
		b := BarisKatalog{Kelas: k}
		b.Kode, b.Pesan = data.Periksa(k)
		if b.Kode == CekOK {
			if p, _ := data.Efektif(k.CourseID); p >= 0 {
				b.Pesan = "Mengulang (nilai terbaik sebelumnya " + data.LabelSkala(p) + ")."
			}
		}
		if b.Kode == CekOK && k.Penuh() {
			b.Kode, b.Pesan = "PENUH", "Kelas penuh, tersedia daftar tunggu."
		}
		if sembunyiLulus && b.Kode == CekLulus {
			continue
		}
		if status != "" && !cocokStatus(status, b.Kode) {
			continue
		}
		b.BisaTambah = kt.Mode != "" && b.Kode == CekOK
		b.BisaTunggu = kt.Mode != "" && b.Kode == "PENUH"
		kt.Baris = append(kt.Baris, b)
	}
	return kt, nil
}

func adaHari(k model.Class, hari int) bool {
	for _, j := range k.Schedules {
		if j.Day == hari {
			return true
		}
	}
	return false
}

func cocokStatus(filter, kode string) bool {
	switch filter {
	case "bisa":
		return kode == CekOK
	case "penuh":
		return kode == "PENUH"
	case "tidak":
		return kode == CekPrasyarat || kode == CekBentrok || kode == CekSKS
	case "dikrs":
		return kode == CekDiKRS || kode == CekMenunggu
	}
	return true
}

// Tambah menambah kelas ke KRS. Urutan validasi sesuai rancangan bagian 5:
// periode -> duplikat -> prasyarat -> bentrok -> SKS -> (transaksi) kuota.
func (s *KRSService) Tambah(ctx context.Context, u model.User, classID int) (string, error) {
	k, err := s.pastikanKRS(ctx, u)
	if err != nil {
		return "", err
	}
	var pesan string
	err = s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		krs, err := tx.KRS.Kunci(ctx, k.KRS.ID) // baca ulang status di dalam transaksi
		if err != nil {
			return err
		}
		mode, err := modeKRS(k.Semester, krs, s.now())
		if err != nil {
			return err
		}
		kelas, err := tx.Class.CariByID(ctx, classID)
		if err != nil || kelas.SemesterID != k.Semester.ID {
			return model.ErrNotFound
		}
		data, err := muat(ctx, tx, k.Mhs.ID, krs)
		if err != nil {
			return err
		}
		if kode, p := data.Periksa(kelas); kode != CekOK {
			return &ValidasiError{Pesan: p}
		}
		// kunci baris kelas, lalu hitung kursi terisi (BR-6)
		if kelas, err = tx.Class.Kunci(ctx, classID); err != nil {
			return err
		}
		if kelas.Penuh() {
			return &KelasPenuhError{Kelas: kelas}
		}
		status := model.ItemAktif
		if mode == modeUbah {
			status = model.ItemMenungguTambah
		}
		itemID, err := tx.KRS.TambahItem(ctx, krs.ID, kelas, status)
		if errors.Is(err, model.ErrDuplikat) {
			return salah("Mata kuliah %s sudah ada di KRS.", kelas.CourseName)
		}
		if err != nil {
			return err
		}
		total := data.TotalSKS() + kelas.Credits
		if mode == modeUbah {
			if err := tx.KRS.TambahLog(ctx, krs.ID, itemID, u.ID, "REQUEST_ADD", ""); err != nil {
				return err
			}
			if err := tx.Notif.Kirim(ctx, k.Mhs.AdvisorUserID,
				fmt.Sprintf("%s mengajukan tambah kelas %s.", k.Mhs.Name, kelas.Label())); err != nil {
				return err
			}
			pesan = fmt.Sprintf("Permintaan tambah %s dikirim ke dosen wali. Kursi sudah dipesan.", kelas.Label())
		} else {
			pesan = fmt.Sprintf("%s masuk ke KRS (%d SKS). Total sekarang %d / %d SKS.",
				kelas.Label(), kelas.Credits, total, krs.MaxCredits)
		}
		return nil
	})
	return pesan, err
}

// Batal menangani tombol batal pada item KRS:
//   - KRS draft (masa isi)       : item dihapus dari KRS, kursi dilepas (FR-D4, BR-8)
//   - masa perubahan, item aktif : diajukan pembatalan ke dosen wali (FR-F4)
//   - permintaan yang menunggu   : permintaan ditarik kembali
func (s *KRSService) Batal(ctx context.Context, u model.User, itemID int) (string, error) {
	k, err := s.pastikanKRS(ctx, u)
	if err != nil {
		return "", err
	}
	var pesan string
	err = s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		krs, err := tx.KRS.Kunci(ctx, k.KRS.ID)
		if err != nil {
			return err
		}
		item, err := tx.KRS.Item(ctx, itemID)
		if err != nil || item.KRSID != krs.ID {
			return model.ErrNotFound
		}
		mode, err := modeKRS(k.Semester, krs, s.now())
		if err != nil {
			return err
		}
		kelas, err := tx.Class.CariByID(ctx, item.ClassID)
		if err != nil {
			return err
		}
		switch {
		case mode == modeIsi && item.Status == model.ItemAktif:
			if err := tx.KRS.SetStatusItem(ctx, item.ID, model.ItemAktif, model.ItemDibatalkan); err != nil {
				return err
			}
			pesan = kelas.Label() + " dihapus dari KRS."
			return s.kursiKosong(ctx, tx, item.ClassID) // titik sambung, lihat sambung.go

		case mode == modeUbah && item.Status == model.ItemMenungguTambah:
			if err := tx.KRS.SetStatusItem(ctx, item.ID, item.Status, model.ItemDibatalkan); err != nil {
				return err
			}
			pesan = "Permintaan tambah " + kelas.Label() + " ditarik."
			return s.kursiKosong(ctx, tx, item.ClassID)

		case mode == modeUbah && item.Status == model.ItemAktif:
			if err := tx.KRS.SetStatusItem(ctx, item.ID, model.ItemAktif, model.ItemMenungguBatal); err != nil {
				return err
			}
			if err := tx.KRS.TambahLog(ctx, krs.ID, item.ID, u.ID, "REQUEST_DROP", ""); err != nil {
				return err
			}
			pesan = "Permintaan batal " + kelas.Label() + " dikirim ke dosen wali."
			return tx.Notif.Kirim(ctx, k.Mhs.AdvisorUserID,
				fmt.Sprintf("%s mengajukan batal kelas %s.", k.Mhs.Name, kelas.Label()))

		case mode == modeUbah && item.Status == model.ItemMenungguBatal:
			pesan = "Permintaan batal " + kelas.Label() + " ditarik."
			return tx.KRS.SetStatusItem(ctx, item.ID, model.ItemMenungguBatal, model.ItemAktif)
		}
		return salah("Item ini tidak dapat diubah pada status sekarang.")
	})
	return pesan, err
}

// Ajukan: validasi ulang menyeluruh sebelum status berubah (FR-D5).
func (s *KRSService) Ajukan(ctx context.Context, u model.User) error {
	k, err := s.pastikanKRS(ctx, u)
	if err != nil {
		return err
	}
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		krs, err := tx.KRS.Kunci(ctx, k.KRS.ID)
		if err != nil {
			return err
		}
		mode, err := modeKRS(k.Semester, krs, s.now())
		if err != nil {
			return err
		}
		if mode != modeIsi {
			return salah("KRS hanya bisa diajukan saat masih draft.")
		}
		data, err := muat(ctx, tx, k.Mhs.ID, krs)
		if err != nil {
			return err
		}
		if len(data.Items) == 0 {
			return salah("KRS masih kosong. Tambahkan minimal satu kelas.")
		}
		if msg := validasiUlang(data); msg != "" {
			return &ValidasiError{Pesan: "KRS belum valid: " + msg}
		}
		if err := tx.KRS.SetStatus(ctx, krs.ID, model.KRSDraft, model.KRSDiajukan); err != nil {
			return err
		}
		if err := tx.KRS.TambahLog(ctx, krs.ID, 0, u.ID, "SUBMIT", ""); err != nil {
			return err
		}
		return tx.Notif.Kirim(ctx, k.Mhs.AdvisorUserID,
			fmt.Sprintf("%s (%s) mengajukan KRS %d SKS.", k.Mhs.Name, k.Mhs.NIM, data.TotalSKS()))
	})
}

// validasiUlang memeriksa ulang setiap item seolah-olah baru ditambahkan.
func validasiUlang(data DataValidasi) string {
	var masalah []string
	for _, it := range data.Items {
		d := data.Tanpa(it.ID)
		d.Menunggu = nil
		if kode, p := d.Periksa(it.Class); kode != CekOK {
			masalah = append(masalah, it.Class.CourseCode+" "+it.Class.CourseName+": "+p)
		}
	}
	return strings.Join(masalah, " | ")
}

type KRSSaya struct {
	Konteks
	Items    []model.KRSItem
	Riwayat  []model.KRSItem
	TotalSKS int
	Mode     string
	ModeErr  string
	Log      []model.ApprovalLog
	Masalah  string
}

func (s *KRSService) KRSSaya(ctx context.Context, u model.User) (KRSSaya, error) {
	var ks KRSSaya
	var err error
	if ks.Konteks, err = s.pastikanKRS(ctx, u); err != nil {
		return ks, err
	}
	ks.Mode, err = modeKRS(ks.Semester, ks.KRS, s.now())
	if err != nil {
		ks.ModeErr = err.Error()
	}
	semua, err := s.R.KRS.Items(ctx, ks.KRS.ID, true)
	if err != nil {
		return ks, err
	}
	for _, it := range semua {
		if it.Hidup() {
			ks.Items = append(ks.Items, it)
			ks.TotalSKS += it.Class.Credits
		} else {
			ks.Riwayat = append(ks.Riwayat, it)
		}
	}
	if ks.Log, err = s.R.KRS.Log(ctx, ks.KRS.ID); err != nil {
		return ks, err
	}
	if ks.KRS.Status == model.KRSDraft && len(ks.Items) > 0 {
		data, err := muat(ctx, s.R, ks.Mhs.ID, ks.KRS)
		if err != nil {
			return ks, err
		}
		ks.Masalah = validasiUlang(data)
	}
	return ks, nil
}
