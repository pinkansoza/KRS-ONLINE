// Tahap 5: daftar tunggu dan promosi otomatis FIFO (modul E).

package service

import (
	"context"
	"errors"
	"fmt"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

// MasukTunggu: hanya untuk kelas yang penuh dan lolos validasi lainnya (BR-7).
func (s *KRSService) MasukTunggu(ctx context.Context, u model.User, classID int) (string, error) {
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
		if _, err := modeKRS(k.Semester, krs, s.now()); err != nil {
			return err
		}
		kelas, err := tx.Class.Kunci(ctx, classID)
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
		if !kelas.Penuh() {
			return salah("Kelas %s masih memiliki kursi kosong, silakan langsung tambah.", kelas.Label())
		}
		if err := tx.Waitlist.Tambah(ctx, classID, k.Mhs.ID, krs.ID); err != nil {
			if errors.Is(err, model.ErrDuplikat) {
				return salah("Anda sudah berada di daftar tunggu kelas ini.")
			}
			return err
		}
		posisi := kelas.Waitlist + 1
		pesan = fmt.Sprintf("Anda masuk daftar tunggu %s (antrean %d).", kelas.Label(), posisi)
		return tx.Notif.Kirim(ctx, u.ID, pesan)
	})
	return pesan, err
}

func (s *KRSService) KeluarTunggu(ctx context.Context, u model.User, entryID int) error {
	w, err := s.R.Waitlist.CariByID(ctx, entryID)
	if err != nil {
		return err
	}
	if w.StudentID != u.StudentID {
		return model.ErrTerlarang
	}
	if err := s.R.Waitlist.SetStatus(ctx, w.ID, model.TungguMenunggu, model.TungguKeluar); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return salah("Antrean ini sudah tidak aktif.")
		}
		return err
	}
	return nil
}

// promosikan mengisi kursi kosong dari daftar tunggu secara FIFO (FR-E3, BR-8).
// Antrean teratas divalidasi ulang (bentrok & batas SKS); kalau gagal ditandai
// DILEWATI (bukan dihapus diam-diam) lalu lanjut ke antrean berikutnya.
// Wajib dipanggil di dalam transaksi.
func (s *KRSService) promosikan(ctx context.Context, tx *repository.Repos, classID int) error {
	for {
		kelas, err := tx.Class.Kunci(ctx, classID)
		if err != nil {
			return err
		}
		if kelas.Penuh() {
			return nil
		}
		w, err := tx.Waitlist.Terdepan(ctx, classID)
		if errors.Is(err, model.ErrNotFound) {
			return nil // antrean kosong
		}
		if err != nil {
			return err
		}
		krs, err := tx.KRS.Kunci(ctx, w.KRSID)
		if err != nil {
			return err
		}
		mhs, err := tx.Student.CariByID(ctx, w.StudentID)
		if err != nil {
			return err
		}
		data, err := muat(ctx, tx, mhs.ID, krs)
		if err != nil {
			return err
		}
		delete(data.Menunggu, kelas.CourseID) // antrean ini sendiri

		kode, alasan := data.Periksa(kelas)
		if kode != CekOK {
			if err := tx.Waitlist.SetStatus(ctx, w.ID, model.TungguMenunggu, model.TungguDilewati); err != nil {
				return err
			}
			if err := tx.Waitlist.LogPromosi(ctx, w.ID, classID, mhs.ID, "SKIPPED", alasan); err != nil {
				return err
			}
			if err := tx.Notif.Kirim(ctx, mhs.UserID, fmt.Sprintf(
				"Kursi %s tersedia, tetapi antrean Anda dilewati: %s", kelas.Label(), alasan)); err != nil {
				return err
			}
			continue
		}

		status := model.ItemAktif
		if krs.Status == model.KRSDisetujui {
			status = model.ItemMenungguTambah // KRS terkunci: tetap butuh persetujuan dosen wali
		}
		itemID, err := tx.KRS.TambahItem(ctx, krs.ID, kelas, status)
		if err != nil {
			return err
		}
		if err := tx.Waitlist.SetStatus(ctx, w.ID, model.TungguMenunggu, model.TungguDipromosikan); err != nil {
			return err
		}
		if err := tx.Waitlist.KeluarkanMK(ctx, mhs.ID, kelas.CourseID, kelas.SemesterID); err != nil {
			return err
		}
		if err := tx.Waitlist.LogPromosi(ctx, w.ID, classID, mhs.ID, "PROMOTED", ""); err != nil {
			return err
		}
		pesan := fmt.Sprintf("Selamat! Anda mendapat kursi %s dari daftar tunggu.", kelas.Label())
		if status == model.ItemMenungguTambah {
			pesan += " Menunggu persetujuan dosen wali."
			if err := tx.KRS.TambahLog(ctx, krs.ID, itemID, mhs.UserID, "REQUEST_ADD", "Promosi daftar tunggu"); err != nil {
				return err
			}
		}
		if krs.Status != model.KRSDraft { // dosen wali perlu tahu KRS bimbingannya berubah
			if err := tx.Notif.Kirim(ctx, mhs.AdvisorUserID, fmt.Sprintf(
				"%s mendapat kursi %s dari daftar tunggu.", mhs.Name, kelas.Label())); err != nil {
				return err
			}
		}
		if err := tx.Notif.Kirim(ctx, mhs.UserID, pesan); err != nil {
			return err
		}
	}
}

type DaftarTungguMhs struct {
	Konteks
	Antrean []model.WaitlistEntry
}

func (s *KRSService) DaftarTunggu(ctx context.Context, u model.User) (DaftarTungguMhs, error) {
	var d DaftarTungguMhs
	var err error
	if d.Konteks, err = s.pastikanKRS(ctx, u); err != nil {
		return d, err
	}
	d.Antrean, err = s.R.Waitlist.MilikMahasiswa(ctx, d.Mhs.ID, d.Semester.ID)
	return d, err
}

// DashboardLengkap = dashboard tahap 4 ditambah daftar tunggu milik mahasiswa.
type DashboardLengkap struct {
	DashboardMhs
	Tunggu []model.WaitlistEntry
}

func (s *KRSService) DashboardLengkap(ctx context.Context, u model.User) (DashboardLengkap, error) {
	var d DashboardLengkap
	var err error
	if d.DashboardMhs, err = s.Dashboard(ctx, u); err != nil {
		return d, err
	}
	d.Tunggu, err = s.R.Waitlist.MilikMahasiswa(ctx, d.Mhs.ID, d.Semester.ID)
	return d, err
}
