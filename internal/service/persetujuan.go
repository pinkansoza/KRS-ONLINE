package service

import (
	"context"
	"fmt"
	"strings"

	"krs-online/internal/model"
	"krs-online/internal/repository"
)

// Bagian ini untuk peran Dosen Wali (modul F).

type DaftarBimbingan struct {
	Semester   model.Semester
	KRS        []model.KRS
	Status     string
	Hitung     map[string]int
	MasaSetuju bool
}

func (s *KRSService) Bimbingan(ctx context.Context, u model.User, status string) (DaftarBimbingan, error) {
	var d DaftarBimbingan
	if !u.IsDosen() || u.LecturerID == 0 {
		return d, model.ErrTerlarang
	}
	var err error
	if d.Semester, err = s.R.Semester.Aktif(ctx); err != nil {
		return d, err
	}
	d.Status = status
	d.MasaSetuju = d.Semester.MasaPersetujuan(s.now())
	semua, err := s.R.KRS.Bimbingan(ctx, u.LecturerID, d.Semester.ID, "")
	if err != nil {
		return d, err
	}
	d.Hitung = map[string]int{"": len(semua)}
	for _, k := range semua {
		st := k.Status
		if st == "" {
			st = "BELUM"
		}
		d.Hitung[st]++
		if k.PermintaanCt > 0 {
			d.Hitung["PERUBAHAN"]++
		}
	}
	if status == "" {
		d.KRS = semua
		return d, nil
	}
	d.KRS, err = s.R.KRS.Bimbingan(ctx, u.LecturerID, d.Semester.ID, status)
	return d, err
}

// HasilItem = hasil validasi ulang sistem untuk satu item (ditampilkan ke dosen).
type HasilItem struct {
	model.KRSItem
	Valid bool
	Pesan string
}

type DetailBimbingan struct {
	Semester   model.Semester
	KRS        model.KRS
	Mhs        model.Student
	Items      []HasilItem
	Riwayat    []model.KRSItem
	TotalSKS   int
	SemuaValid bool
	IPK        float64
	Log        []model.ApprovalLog
	Antrean    []model.WaitlistEntry
	MasaSetuju bool
	Permintaan int
}

// ambilBimbingan memastikan KRS milik mahasiswa bimbingan dosen ini (otorisasi).
func (s *KRSService) ambilBimbingan(ctx context.Context, r *repository.Repos, u model.User, krsID int) (model.KRS, model.Student, error) {
	krs, err := r.KRS.CariByID(ctx, krsID)
	if err != nil {
		return krs, model.Student{}, err
	}
	mhs, err := r.Student.CariByID(ctx, krs.StudentID)
	if err != nil {
		return krs, mhs, err
	}
	if !u.IsDosen() || mhs.AdvisorID != u.LecturerID {
		return krs, mhs, model.ErrTerlarang
	}
	return krs, mhs, nil
}

func (s *KRSService) DetailBimbingan(ctx context.Context, u model.User, krsID int) (DetailBimbingan, error) {
	var d DetailBimbingan
	var err error
	if d.KRS, d.Mhs, err = s.ambilBimbingan(ctx, s.R, u, krsID); err != nil {
		return d, err
	}
	if d.Semester, err = s.R.Semester.CariByID(ctx, d.KRS.SemesterID); err != nil {
		return d, err
	}
	d.MasaSetuju = d.Semester.MasaPersetujuan(s.now())
	data, err := muat(ctx, s.R, d.Mhs.ID, d.KRS)
	if err != nil {
		return d, err
	}
	d.SemuaValid = true
	for _, it := range data.Items {
		cek := data.Tanpa(it.ID)
		cek.Menunggu = nil
		h := HasilItem{KRSItem: it, Valid: true}
		if kode, p := cek.Periksa(it.Class); kode != CekOK {
			h.Valid, h.Pesan, d.SemuaValid = false, p, false
		}
		if it.Status != model.ItemAktif {
			d.Permintaan++
		}
		d.Items = append(d.Items, h)
		d.TotalSKS += it.Class.Credits
	}
	semua, err := s.R.KRS.Items(ctx, d.KRS.ID, true)
	if err != nil {
		return d, err
	}
	for _, it := range semua {
		if !it.Hidup() {
			d.Riwayat = append(d.Riwayat, it)
		}
	}
	riwayat, err := s.R.Grade.Riwayat(ctx, d.Mhs.ID)
	if err != nil {
		return d, err
	}
	d.IPK = RingkasNilai(riwayat, d.Semester.ID).IPK
	if d.Log, err = s.R.KRS.Log(ctx, d.KRS.ID); err != nil {
		return d, err
	}
	d.Antrean, err = s.R.Waitlist.MilikMahasiswa(ctx, d.Mhs.ID, d.Semester.ID)
	return d, err
}

// Setujui KRS: hanya pada periode persetujuan dan bila semua item valid.
// Setelah disetujui KRS terkunci (BR-9).
func (s *KRSService) Setujui(ctx context.Context, u model.User, krsID int, catatan string) error {
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		krs, mhs, err := s.ambilBimbingan(ctx, tx, u, krsID)
		if err != nil {
			return err
		}
		if krs, err = tx.KRS.Kunci(ctx, krs.ID); err != nil {
			return err
		}
		if krs.Status != model.KRSDiajukan {
			return salah("Hanya KRS berstatus DIAJUKAN yang dapat disetujui.")
		}
		sem, err := tx.Semester.CariByID(ctx, krs.SemesterID)
		if err != nil {
			return err
		}
		if !sem.MasaPersetujuan(s.now()) {
			return salah("Di luar periode persetujuan dosen wali.")
		}
		data, err := muat(ctx, tx, mhs.ID, krs)
		if err != nil {
			return err
		}
		if msg := validasiUlang(data); msg != "" {
			return salah("KRS tidak valid, minta revisi: %s", msg)
		}
		if err := tx.KRS.SetStatus(ctx, krs.ID, model.KRSDiajukan, model.KRSDisetujui); err != nil {
			return err
		}
		if err := tx.KRS.TambahLog(ctx, krs.ID, 0, u.ID, "APPROVE", strings.TrimSpace(catatan)); err != nil {
			return err
		}
		return tx.Notif.Kirim(ctx, mhs.UserID, "KRS Anda disetujui oleh dosen wali dan sekarang terkunci.")
	})
}

// Tolak / minta revisi: catatan wajib, KRS kembali ke DRAFT (FR-F2).
// Item tetap di KRS supaya mahasiswa cukup merevisi bagian yang bermasalah.
func (s *KRSService) Tolak(ctx context.Context, u model.User, krsID int, catatan string) error {
	catatan = strings.TrimSpace(catatan)
	if catatan == "" {
		return salah("Catatan wajib diisi untuk penolakan.")
	}
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		krs, mhs, err := s.ambilBimbingan(ctx, tx, u, krsID)
		if err != nil {
			return err
		}
		if krs.Status != model.KRSDiajukan {
			return salah("Hanya KRS berstatus DIAJUKAN yang dapat ditolak.")
		}
		if err := tx.KRS.SetStatus(ctx, krs.ID, model.KRSDiajukan, model.KRSDraft); err != nil {
			return err
		}
		if err := tx.KRS.TambahLog(ctx, krs.ID, 0, u.ID, "REJECT", catatan); err != nil {
			return err
		}
		return tx.Notif.Kirim(ctx, mhs.UserID, "KRS dikembalikan untuk revisi: "+catatan)
	})
}

// PutuskanItem: persetujuan per permintaan pada masa perubahan KRS (FR-F4).
func (s *KRSService) PutuskanItem(ctx context.Context, u model.User, krsID, itemID int, setuju bool, catatan string) error {
	catatan = strings.TrimSpace(catatan)
	if !setuju && catatan == "" {
		return salah("Catatan wajib diisi untuk penolakan.")
	}
	return s.R.Transaksi(ctx, func(tx *repository.Repos) error {
		krs, mhs, err := s.ambilBimbingan(ctx, tx, u, krsID)
		if err != nil {
			return err
		}
		item, err := tx.KRS.Item(ctx, itemID)
		if err != nil || item.KRSID != krs.ID {
			return model.ErrNotFound
		}
		kelas, err := tx.Class.CariByID(ctx, item.ClassID)
		if err != nil {
			return err
		}
		var ke, aksi, pesan string
		lepasKursi := false
		switch {
		case item.Status == model.ItemMenungguTambah && setuju:
			ke, aksi, pesan = model.ItemAktif, "APPROVE_ADD", "Permintaan tambah %s disetujui."
		case item.Status == model.ItemMenungguTambah:
			ke, aksi, pesan, lepasKursi = model.ItemDibatalkan, "REJECT_ADD", "Permintaan tambah %s ditolak.", true
		case item.Status == model.ItemMenungguBatal && setuju:
			ke, aksi, pesan, lepasKursi = model.ItemDibatalkan, "APPROVE_DROP", "Permintaan batal %s disetujui.", true
		case item.Status == model.ItemMenungguBatal:
			ke, aksi, pesan = model.ItemAktif, "REJECT_DROP", "Permintaan batal %s ditolak."
		default:
			return salah("Item ini tidak sedang menunggu keputusan.")
		}
		if err := tx.KRS.SetStatusItem(ctx, item.ID, item.Status, ke); err != nil {
			return err
		}
		if err := tx.KRS.TambahLog(ctx, krs.ID, item.ID, u.ID, aksi, catatan); err != nil {
			return err
		}
		pesan = fmt.Sprintf(pesan, kelas.Label())
		if catatan != "" {
			pesan += " Catatan: " + catatan
		}
		if err := tx.Notif.Kirim(ctx, mhs.UserID, pesan); err != nil {
			return err
		}
		if lepasKursi {
			return s.promosikan(ctx, tx, kelas.ID)
		}
		return nil
	})
}
