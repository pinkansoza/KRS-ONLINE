// Tahap 2: dashboard admin, semester, dan pengaturan periode KRS.

package handler

import (
	"fmt"
	"net/http"
	"strings"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

// semesterDipilih: ?semester= dari query string, default semester aktif.
func (h *AdminHandler) semesterDipilih(r *http.Request, semua []model.Semester) int {
	if id := angkaQuery(r, "semester", 0); id > 0 {
		return id
	}
	for _, s := range semua {
		if s.IsActive {
			return s.ID
		}
	}
	if len(semua) > 0 {
		return semua[len(semua)-1].ID
	}
	return 0
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	semua, err := h.Svc.Semester(ctx)
	if err != nil {
		gagal(w, err)
		return
	}
	data := map[string]any{"Semester": semua}
	if err := h.ringkasanDashboard(r, semua, data); err != nil { // titik sambung, lihat sambung.go
		gagal(w, err)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/dashboard.html", "Dashboard admin", "dashboard", data)
}

type SemesterForm struct {
	Judul    string
	Action   string
	Baru     bool
	Semester model.Semester
}

func (h *AdminHandler) FormSemesterBaru(w http.ResponseWriter, r *http.Request) {
	h.tampil(w, r, http.StatusOK, "admin/semester_form.html", "Semester baru", "dashboard",
		SemesterForm{Judul: "Tambah semester", Action: "/admin/semester", Baru: true, Semester: model.Semester{Term: "GANJIL"}})
}

func (h *AdminHandler) SimpanSemester(w http.ResponseWriter, r *http.Request) {
	s := model.Semester{AcademicYear: strings.TrimSpace(r.PostFormValue("tahun")), Term: r.PostFormValue("term")}
	if err := h.Svc.TambahSemester(r.Context(), s); err != nil {
		h.formGagal(w, r, err, "admin/semester_form.html", "Semester baru", "dashboard",
			SemesterForm{Judul: "Tambah semester", Action: "/admin/semester", Baru: true, Semester: s})
		return
	}
	redirect(w, r, "/admin", "dibuat")
}

func (h *AdminHandler) FormPeriode(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	s, err := h.Svc.SemesterByID(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/semester_form.html", "Atur periode", "dashboard",
		SemesterForm{Judul: "Atur periode " + s.Label(), Action: fmt.Sprintf("/admin/semester/%d/edit", id), Semester: s})
}

func (h *AdminHandler) UbahPeriode(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	s, err := h.Svc.SemesterByID(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	errs := service.FormError{}
	var okf bool
	if s.KRSStart, okf = waktuForm(r, "krs_start"); !okf {
		errs["krs"] = "Format tanggal tidak valid"
	}
	if s.KRSEnd, okf = waktuForm(r, "krs_end"); !okf {
		errs["krs"] = "Format tanggal tidak valid"
	}
	if s.ApprovalStart, okf = waktuForm(r, "approval_start"); !okf {
		errs["approval"] = "Format tanggal tidak valid"
	}
	if s.ApprovalEnd, okf = waktuForm(r, "approval_end"); !okf {
		errs["approval"] = "Format tanggal tidak valid"
	}
	if s.ChangeStart, okf = waktuForm(r, "change_start"); !okf {
		errs["change"] = "Format tanggal tidak valid"
	}
	if s.ChangeEnd, okf = waktuForm(r, "change_end"); !okf {
		errs["change"] = "Format tanggal tidak valid"
	}
	if len(errs) > 0 {
		err = errs
	} else {
		err = h.Svc.UbahPeriode(r.Context(), s)
	}
	if err != nil {
		h.formGagal(w, r, err, "admin/semester_form.html", "Atur periode", "dashboard",
			SemesterForm{Judul: "Atur periode " + s.Label(), Action: fmt.Sprintf("/admin/semester/%d/edit", id), Semester: s})
		return
	}
	redirect(w, r, "/admin", "diubah")
}

func (h *AdminHandler) AktifkanSemester(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	if err := h.Svc.AktifkanSemester(r.Context(), id); err != nil {
		gagal(w, err)
		return
	}
	redirect(w, r, "/admin", "aktif")
}
