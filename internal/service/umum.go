package service

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ValidasiError: error khusus supaya handler bisa membedakan salah input
// dari server error.
type ValidasiError struct {
	Pesan string
}

func (e *ValidasiError) Error() string { return e.Pesan }

func salah(format string, a ...any) error {
	return &ValidasiError{Pesan: fmt.Sprintf(format, a...)}
}

// FormError mengumpulkan pesan per field: semua field diperiksa dulu,
// baru dikembalikan sekaligus.
type FormError map[string]string

func (f FormError) Error() string {
	pesan := make([]string, 0, len(f))
	for _, p := range f {
		pesan = append(pesan, p)
	}
	return strings.Join(pesan, "; ")
}

func (f FormError) atauNil() error {
	if len(f) == 0 {
		return nil
	}
	return f
}

func wajib(errs FormError, field, nilai, label string, maks int) {
	switch {
	case strings.TrimSpace(nilai) == "":
		errs[field] = label + " wajib diisi"
	case utf8.RuneCountInString(nilai) > maks:
		errs[field] = fmt.Sprintf("%s maksimal %d karakter", label, maks)
	}
}

var namaHari = [...]string{"", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"}

func NamaHari(d int) string {
	if d >= 1 && d <= 7 {
		return namaHari[d]
	}
	return "?"
}