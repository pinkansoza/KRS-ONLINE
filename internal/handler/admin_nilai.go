// Tahap 3: halaman input dan impor nilai (FR-C1, FR-C2).

package handler

import (
	"fmt"
	"net/http"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

type NilaiView struct {
	Mhs      []model.Student
	Dipilih  model.Student
	Riwayat  []model.Grade
	MK       []model.Course
	Semester []model.Semester
	Skala    []model.GradeScale
	Form     model.Grade
	Impor    *service.HasilImpor
}

func (h *AdminHandler) Nilai(w http.ResponseWriter, r *http.Request) {
	h.tampilNilai(w, r, http.StatusOK, angkaQuery(r, "mahasiswa", 0), model.Grade{}, nil, nil)
}

func (h *AdminHandler) tampilNilai(w http.ResponseWriter, r *http.Request, status, studentID int, form model.Grade, impor *service.HasilImpor, err error) {
	ctx := r.Context()
	v := NilaiView{Form: form, Impor: impor}
	var e error
	if v.Mhs, e = h.Svc.Mahasiswa(ctx, ""); e != nil {
		gagal(w, e)
		return
	}
	if v.MK, e = h.Svc.MataKuliah(ctx, "", 0); e != nil {
		gagal(w, e)
		return
	}
	if v.Semester, e = h.Svc.Semester(ctx); e != nil {
		gagal(w, e)
		return
	}
	if v.Skala, e = h.Svc.SkalaNilai(ctx); e != nil {
		gagal(w, e)
		return
	}
	if studentID > 0 {
		if v.Dipilih, e = h.Svc.DetailMahasiswa(ctx, studentID); e != nil {
			gagal(w, e)
			return
		}
		if v.Riwayat, e = h.Svc.RiwayatNilai(ctx, studentID); e != nil {
			gagal(w, e)
			return
		}
	}
	if err != nil {
		h.formGagal(w, r, err, "admin/nilai.html", "Input nilai", "nilai", v)
		return
	}
	h.tampil(w, r, status, "admin/nilai.html", "Input nilai", "nilai", v)
}

func (h *AdminHandler) SimpanNilai(w http.ResponseWriter, r *http.Request) {
	g := model.Grade{
		StudentID:  angka(r.PostFormValue("student_id")),
		CourseID:   angka(r.PostFormValue("course_id")),
		SemesterID: angka(r.PostFormValue("semester_id")),
		Letter:     r.PostFormValue("letter"),
	}
	if err := h.Svc.SimpanNilai(r.Context(), g); err != nil {
		h.tampilNilai(w, r, 0, g.StudentID, g, nil, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/nilai?mahasiswa=%d", g.StudentID), "dibuat")
}

func (h *AdminHandler) HapusNilai(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	mhs := angka(r.PostFormValue("student_id"))
	if err := h.Svc.HapusNilai(r.Context(), mhs, id); err != nil {
		gagal(w, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/nilai?mahasiswa=%d", mhs), "dihapus")
}

// ImporNilai menerima upload file CSV (form multipart/form-data).
func (h *AdminHandler) ImporNilai(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		h.tampilNilai(w, r, 0, 0, model.Grade{}, nil, &service.ValidasiError{Pesan: "Pilih file CSV terlebih dahulu."})
		return
	}
	defer file.Close()
	hasil, err := h.Svc.ImporNilai(r.Context(), file)
	if err != nil {
		gagal(w, err)
		return
	}
	// hasil impor ditampilkan langsung (bukan redirect) agar daftar baris gagal terlihat
	h.tampilNilai(w, r, http.StatusOK, 0, model.Grade{}, &hasil, nil)
}
