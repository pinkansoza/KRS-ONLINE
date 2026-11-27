// Tahap 5: halaman rekap kelas & daftar tunggu, plus output JSON.

package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

func (h *AdminHandler) ambilRekap(w http.ResponseWriter, r *http.Request) (service.Rekap, []model.Semester, bool) {
	ctx := r.Context()
	semua, err := h.Svc.Semester(ctx)
	if err != nil {
		gagal(w, err)
		return service.Rekap{}, nil, false
	}
	rk, err := h.Svc.Rekap(ctx, h.semesterDipilih(r, semua), strings.TrimSpace(r.URL.Query().Get("q")), r.URL.Query().Get("penuh") == "1")
	if err != nil {
		gagal(w, err)
		return rk, nil, false
	}
	return rk, semua, true
}

func (h *AdminHandler) Rekap(w http.ResponseWriter, r *http.Request) {
	rk, semua, ok := h.ambilRekap(w, r)
	if !ok {
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/rekap.html", "Rekap kelas & daftar tunggu", "rekap", map[string]any{
		"R": rk, "Semester": semua, "Q": r.URL.Query().Get("q"), "Penuh": r.URL.Query().Get("penuh") == "1",
		"Query": r.URL.RawQuery})
}

// RekapJSON: output data JSON (materi menyajikan output) untuk dipakai aplikasi lain.
func (h *AdminHandler) RekapJSON(w http.ResponseWriter, r *http.Request) {
	rk, _, ok := h.ambilRekap(w, r)
	if !ok {
		return
	}
	type kelasJSON struct {
		ID       int    `json:"id"`
		Kode     string `json:"kode_mk"`
		Nama     string `json:"mata_kuliah"`
		Kelas    string `json:"kelas"`
		Dosen    string `json:"dosen"`
		Terisi   int    `json:"terisi"`
		Kuota    int    `json:"kuota"`
		Menunggu int    `json:"daftar_tunggu"`
	}
	out := struct {
		Semester string            `json:"semester"`
		Status   model.RekapStatus `json:"status_krs"`
		Kelas    []kelasJSON       `json:"kelas"`
	}{Semester: rk.Semester.Label(), Status: rk.Status, Kelas: []kelasJSON{}}
	for _, k := range rk.Kelas {
		out.Kelas = append(out.Kelas, kelasJSON{k.ID, k.CourseCode, k.CourseName, k.Code, k.LecturerName, k.Enrolled, k.Quota, k.Waitlist})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

func (h *AdminHandler) RekapKelas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	rk, err := h.Svc.RekapKelas(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/rekap_kelas.html", "Peserta "+rk.Kelas.Label(), "rekap", rk)
}
