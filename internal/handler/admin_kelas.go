// Tahap 3: halaman kelas, jadwal, dan kuota.

package handler

import (
	"fmt"
	"net/http"
	"strings"

	"krs-online/internal/model"
)

func (h *AdminHandler) DaftarKelas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	semua, err := h.Svc.Semester(ctx)
	if err != nil {
		gagal(w, err)
		return
	}
	semID := h.semesterDipilih(r, semua)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	kelas, err := h.Svc.Kelas(ctx, semID, q)
	if err != nil {
		gagal(w, err)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/kelas_list.html", "Kelas & jadwal", "kelas",
		map[string]any{"Kelas": kelas, "Semester": semua, "SemID": semID, "Q": q})
}

type KelasForm struct {
	Judul    string
	Action   string
	Baru     bool
	Kelas    model.Class
	MK       []model.Course
	Dosen    []model.Lecturer
	Semester []model.Semester
}

func (h *AdminHandler) formKelas(w http.ResponseWriter, r *http.Request, f KelasForm, err error) {
	ctx := r.Context()
	var e error
	if f.Dosen, e = h.Svc.Dosen(ctx); e != nil {
		gagal(w, e)
		return
	}
	if f.Baru {
		if f.MK, e = h.Svc.MataKuliahAktif(ctx); e != nil {
			gagal(w, e)
			return
		}
		if f.Semester, e = h.Svc.Semester(ctx); e != nil {
			gagal(w, e)
			return
		}
	}
	if err != nil {
		h.formGagal(w, r, err, "admin/kelas_form.html", f.Judul, "kelas", f)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/kelas_form.html", f.Judul, "kelas", f)
}

func kelasDariForm(r *http.Request) model.Class {
	return model.Class{
		SemesterID: angka(r.PostFormValue("semester_id")),
		CourseID:   angka(r.PostFormValue("course_id")),
		Code:       r.PostFormValue("class_code"),
		LecturerID: angka(r.PostFormValue("lecturer_id")),
		Quota:      angka(r.PostFormValue("quota")),
		Room:       strings.TrimSpace(r.PostFormValue("room")),
	}
}

func (h *AdminHandler) FormKelasBaru(w http.ResponseWriter, r *http.Request) {
	k := model.Class{Quota: 40, Code: "A", SemesterID: angkaQuery(r, "semester", 0)}
	h.formKelas(w, r, KelasForm{Judul: "Buka kelas", Action: "/admin/kelas", Baru: true, Kelas: k}, nil)
}

func (h *AdminHandler) SimpanKelas(w http.ResponseWriter, r *http.Request) {
	k := kelasDariForm(r)
	id, err := h.Svc.TambahKelas(r.Context(), k)
	if err != nil {
		h.formKelas(w, r, KelasForm{Judul: "Buka kelas", Action: "/admin/kelas", Baru: true, Kelas: k}, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/kelas/%d", id), "dibuat")
}

func (h *AdminHandler) DetailKelas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	h.tampilDetailKelas(w, r, id, http.StatusOK, "", nil)
}

func (h *AdminHandler) tampilDetailKelas(w http.ResponseWriter, r *http.Request, id, status int, galat string, fe map[string]string) {
	k, err := h.Svc.DetailKelas(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, r, status, "admin/kelas_detail.html", Halaman{Title: k.Label(), Menu: "kelas",
		Galat: galat, Errors: fe, Data: k})
}

func (h *AdminHandler) FormEditKelas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	k, err := h.Svc.DetailKelas(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.formKelas(w, r, KelasForm{Judul: "Ubah kelas " + k.Label(), Action: fmt.Sprintf("/admin/kelas/%d/edit", id), Kelas: k}, nil)
}

func (h *AdminHandler) UbahKelas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	lama, err := h.Svc.DetailKelas(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	k := kelasDariForm(r)
	k.ID, k.CourseID, k.SemesterID = id, lama.CourseID, lama.SemesterID
	k.CourseCode, k.CourseName, k.Enrolled = lama.CourseCode, lama.CourseName, lama.Enrolled
	if err := h.Svc.UbahKelas(r.Context(), k); err != nil {
		h.formKelas(w, r, KelasForm{Judul: "Ubah kelas " + lama.Label(), Action: fmt.Sprintf("/admin/kelas/%d/edit", id), Kelas: k}, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/kelas/%d", id), "diubah")
}

func (h *AdminHandler) HapusKelas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	k, err := h.Svc.DetailKelas(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	if err := h.Svc.HapusKelas(r.Context(), id); err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilDetailKelas(w, r, id, http.StatusUnprocessableEntity, pesan, nil)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/kelas?semester=%d", k.SemesterID), "dihapus")
}

func (h *AdminHandler) TambahJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	j := model.Schedule{ClassID: id, Day: angka(r.PostFormValue("day")),
		Start: r.PostFormValue("start"), End: r.PostFormValue("end"), Room: strings.TrimSpace(r.PostFormValue("room"))}
	if err := h.Svc.TambahJadwal(r.Context(), j); err != nil {
		if pesan, fe, ok := pesanValidasi(err); ok {
			h.tampilDetailKelas(w, r, id, http.StatusUnprocessableEntity, pesan, fe)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/kelas/%d", id), "dibuat")
}

func (h *AdminHandler) HapusJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	j, ok := idDariPath(w, r, "jadwal")
	if !ok {
		return
	}
	if err := h.Svc.HapusJadwal(r.Context(), id, j); err != nil {
		gagal(w, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/kelas/%d", id), "dihapus")
}
