// Tahap 2: halaman data mahasiswa.

package handler

import (
	"fmt"
	"net/http"
	"strings"

	"krs-online/internal/model"
)

func (h *AdminHandler) DaftarMhs(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	daftar, err := h.Svc.Mahasiswa(r.Context(), q)
	if err != nil {
		gagal(w, err)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/mhs_list.html", "Mahasiswa", "mhs", map[string]any{"Mhs": daftar, "Q": q})
}

type MhsForm struct {
	Judul     string
	Action    string
	Baru      bool
	Mhs       model.Student
	Kurikulum []model.Curriculum
	Dosen     []model.Lecturer
}

func (h *AdminHandler) formMhs(w http.ResponseWriter, r *http.Request, f MhsForm, err error) {
	var e error
	if f.Kurikulum, e = h.Svc.Kurikulum(r.Context()); e != nil {
		gagal(w, e)
		return
	}
	if f.Dosen, e = h.Svc.Dosen(r.Context()); e != nil {
		gagal(w, e)
		return
	}
	if err != nil {
		h.formGagal(w, r, err, "admin/mhs_form.html", f.Judul, "mhs", f)
		return
	}
	h.tampil(w, r, http.StatusOK, "admin/mhs_form.html", f.Judul, "mhs", f)
}

func mhsDariForm(r *http.Request) model.Student {
	return model.Student{
		NIM:          r.PostFormValue("nim"),
		Name:         r.PostFormValue("name"),
		Email:        r.PostFormValue("email"),
		EntryYear:    angka(r.PostFormValue("entry_year")),
		CurriculumID: angka(r.PostFormValue("curriculum_id")),
		AdvisorID:    angka(r.PostFormValue("advisor_id")),
		Status:       r.PostFormValue("status"),
	}
}

func (h *AdminHandler) FormMhsBaru(w http.ResponseWriter, r *http.Request) {
	h.formMhs(w, r, MhsForm{Judul: "Tambah mahasiswa", Action: "/admin/mahasiswa", Baru: true,
		Mhs: model.Student{Status: "AKTIF", EntryYear: 2026}}, nil)
}

func (h *AdminHandler) SimpanMhs(w http.ResponseWriter, r *http.Request) {
	m := mhsDariForm(r)
	if err := h.Svc.TambahMahasiswa(r.Context(), m, r.PostFormValue("password")); err != nil {
		h.formMhs(w, r, MhsForm{Judul: "Tambah mahasiswa", Action: "/admin/mahasiswa", Baru: true, Mhs: m}, err)
		return
	}
	redirect(w, r, "/admin/mahasiswa", "dibuat")
}

func (h *AdminHandler) FormEditMhs(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	m, err := h.Svc.DetailMahasiswa(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.formMhs(w, r, MhsForm{Judul: "Ubah mahasiswa", Action: fmt.Sprintf("/admin/mahasiswa/%d/edit", id), Mhs: m}, nil)
}

func (h *AdminHandler) UbahMhs(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	lama, err := h.Svc.DetailMahasiswa(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	m := mhsDariForm(r)
	m.ID, m.NIM = id, lama.NIM
	if err := h.Svc.UbahMahasiswa(r.Context(), m, r.PostFormValue("password")); err != nil {
		h.formMhs(w, r, MhsForm{Judul: "Ubah mahasiswa", Action: fmt.Sprintf("/admin/mahasiswa/%d/edit", id), Mhs: m}, err)
		return
	}
	redirect(w, r, "/admin/mahasiswa", "diubah")
}
