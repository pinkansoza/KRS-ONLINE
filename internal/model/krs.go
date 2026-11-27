// Tahap 4: KRS, item KRS, dan log persetujuan.

package model

import (
	"time"
)

// Status KRS.
const (
	KRSDraft     = "DRAFT"
	KRSDiajukan  = "DIAJUKAN"
	KRSDisetujui = "DISETUJUI"
)

// Status item KRS.
const (
	ItemAktif          = "AKTIF"
	ItemDibatalkan     = "DIBATALKAN"
	ItemMenungguTambah = "MENUNGGU_TAMBAH"
	ItemMenungguBatal  = "MENUNGGU_BATAL"
)

type KRSItem struct {
	ID       int
	KRSID    int
	ClassID  int
	CourseID int
	Status   string
	AddedAt  time.Time
	Class    Class
}

// Hidup = item yang memegang kursi dan dihitung SKS-nya.
func (i KRSItem) Hidup() bool {
	return i.Status == ItemAktif || i.Status == ItemMenungguTambah || i.Status == ItemMenungguBatal
}

type KRS struct {
	ID          int
	StudentID   int
	SemesterID  int
	Status      string
	MaxCredits  int
	IPSRef      float64
	AdaIPS      bool
	SubmittedAt time.Time
	ApprovedAt  time.Time
	Items       []KRSItem
	// diisi untuk tampilan dosen wali
	Student      Student
	TotalSKS     int
	JumlahKelas  int
	PermintaanCt int
}

type ApprovalLog struct {
	ID        int
	KRSID     int
	ItemID    int
	ItemLabel string
	ActorName string
	Action    string
	Note      string
	CreatedAt time.Time
}
