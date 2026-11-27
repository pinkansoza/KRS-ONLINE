// Tahap 2: uji cek prasyarat siklik tanpa database.

package service

import (
	"testing"
)

// Jalankan: go test ./internal/service/

func TestPrasyaratSiklik(t *testing.T) {
	graf := map[int][]int{5: {4}, 9: {6}, 6: {5}} // 9 -> 6 -> 5 -> 4
	if !AdaSiklus(graf, 4, 9) {
		t.Fatal("4 butuh 9 seharusnya siklik tidak langsung")
	}
	if AdaSiklus(graf, 9, 7) {
		t.Fatal("9 butuh 7 bukan siklik")
	}
	if !AdaSiklus(graf, 3, 3) {
		t.Fatal("MK tidak boleh menjadi prasyarat dirinya sendiri")
	}
}
