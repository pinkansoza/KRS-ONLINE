// Tahap 5: daftar tunggu, log promosi, dan rekap status KRS.

package model

import (
	"time"
)

// Status antrean daftar tunggu.
const (
	TungguMenunggu     = "MENUNGGU"
	TungguDipromosikan = "DIPROMOSIKAN"
	TungguDilewati     = "DILEWATI"
	TungguKeluar       = "KELUAR"
)

type WaitlistEntry struct {
	ID        int
	ClassID   int
	StudentID int
	KRSID     int
	QueuedAt  time.Time
	Status    string
	Posisi    int
	Class     Class
	NIM       string
	Nama      string
}

type PromotionLog struct {
	ID        int
	ClassID   int
	ClassLbl  string
	NIM       string
	Nama      string
	Result    string
	Reason    string
	CreatedAt time.Time
}

// RekapStatus = FR-G3: jumlah mahasiswa per status KRS.
type RekapStatus struct {
	BelumMengisi int `json:"belum_mengisi"`
	Draft        int `json:"draft"`
	Diajukan     int `json:"diajukan"`
	Disetujui    int `json:"disetujui"`
	Total        int `json:"total_mahasiswa_aktif"`
}
