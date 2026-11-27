// Tahap 2: halaman mata kuliah, prasyarat, kurikulum, dan kesetaraan.

package handler

import (
	"fmt"
	"net/http"
	"strings"

	"krs-online/internal/model"
)

func (h *AdminHandler) DaftarMK(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	kur := angkaQuery(r, "kurikulum", 0)
	daftar, err := h.Svc.MataKuliah(ctx, q, kur)
	if err != nil {
		gagal(w, err)
		return
	}
	kurikulum, err := h.Svc.Kurikulum(ctx)
	if err != nil {
		gagal(w, err)
		return
	}
	pras, err := h.Svc.R.Course.SemuaPrasyarat(ctx)
	if err != nil {
		gagal(w, err)
		return
	}
	jml := map[int]int{}
	for id, g := range pras {
		jml[id] = len(g)
	}
	h.tampil(w, r, http.StatusOK, "admin/mk_list.html", "Mata kuliah", "mk",
		map[string]any{"MK": daftar, "Kurikulum": kurikulum, "Q": q, "Kur": kur, "JmlGrup": jml})
}

type MKForm struct {
	Judul     string
	Action    string
	MK        model.Course
	Kurikulum []model.Curriculum
}

func (h *AdminHandler) formMK(w http.ResponseWriter, r *http.Request, status int, f MKForm, err error) {
	kur, e := h.Svc.Kurikulum(r.Context())
	if e != nil {
		gagal(w, e)
		return
	}
	f.Kurikulum = kur
	if err != nil {
		h.formGagal(w, r, err, "admin/mk_form.html", f.Judul, "mk", f)
		return
	}
	h.tampil(w, r, status, "admin/mk_form.html", f.Judul, "mk", f)
}

func mkDariForm(r *http.Request) model.Course {
	return model.Course{
		Code:                r.PostFormValue("code"),
		Name:                r.PostFormValue("name"),
		Credits:             angka(r.PostFormValue("credits")),
		CurriculumID:        angka(r.PostFormValue("curriculum_id")),
		RecommendedSemester: angka(r.PostFormValue("recommended_semester")),
		IsActive:            r.PostFormValue("is_active") == "1",
	}
}

func (h *AdminHandler) FormMKBaru(w http.ResponseWriter, r *http.Request) {
	h.formMK(w, r, http.StatusOK, MKForm{Judul: "Tambah mata kuliah", Action: "/admin/mk",
		MK: model.Course{IsActive: true, Credits: 3}}, nil)
}

func (h *AdminHandler) SimpanMK(w http.ResponseWriter, r *http.Request) {
	c := mkDariForm(r)
	id, err := h.Svc.SimpanMK(r.Context(), c)
	if err != nil {
		h.formMK(w, r, 0, MKForm{Judul: "Tambah mata kuliah", Action: "/admin/mk", MK: c}, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/mk/%d", id), "dibuat")
}

func (h *AdminHandler) FormEditMK(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	c, err := h.Svc.DetailMK(r.Context(), id)
	if err != nil {
		gagal(w, err)
		return
	}
	h.formMK(w, r, http.StatusOK, MKForm{Judul: "Ubah mata kuliah", Action: fmt.Sprintf("/admin/mk/%d/edit", id), MK: c}, nil)
}

func (h *AdminHandler) UbahMK(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	c := mkDariForm(r)
	c.ID = id
	if _, err := h.Svc.SimpanMK(r.Context(), c); err != nil {
		h.formMK(w, r, 0, MKForm{Judul: "Ubah mata kuliah", Action: fmt.Sprintf("/admin/mk/%d/edit", id), MK: c}, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/mk/%d", id), "diubah")
}

func (h *AdminHandler) AktifMK(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	if err := h.Svc.SetAktifMK(r.Context(), id, r.PostFormValue("aktif") == "1"); err != nil {
		gagal(w, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/mk/%d", id), "diubah")
}

// DetailMK menampilkan data MK + pengelolaan grup prasyarat (FR-A2).
func (h *AdminHandler) DetailMK(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	h.tampilDetailMK(w, r, id, http.StatusOK, "")
}

func (h *AdminHandler) tampilDetailMK(w http.ResponseWriter, r *http.Request, id, status int, galat string) {
	ctx := r.Context()
	c, err := h.Svc.DetailMK(ctx, id)
	if err != nil {
		gagal(w, err)
		return
	}
	grup, err := h.Svc.PrasyaratMK(ctx, id)
	if err != nil {
		gagal(w, err)
		return
	}
	semua, err := h.Svc.MataKuliah(ctx, "", 0)
	if err != nil {
		gagal(w, err)
		return
	}
	skala, err := h.Svc.SkalaNilai(ctx)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, r, status, "admin/mk_detail.html", Halaman{Title: c.Label(), Menu: "mk", Galat: galat,
		Data: map[string]any{"MK": c, "Grup": grup, "Semua": semua, "Skala": skala}})
}

func (h *AdminHandler) aksiMK(w http.ResponseWriter, r *http.Request, id int, err error) {
	if err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilDetailMK(w, r, id, http.StatusUnprocessableEntity, pesan)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/mk/%d", id), "diubah")
}

func (h *AdminHandler) TambahGrup(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	h.aksiMK(w, r, id, h.Svc.TambahGrup(r.Context(), id))
}

func (h *AdminHandler) HapusGrup(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	g, ok := idDariPath(w, r, "grup")
	if !ok {
		return
	}
	h.aksiMK(w, r, id, h.Svc.HapusGrup(r.Context(), id, g))
}

func (h *AdminHandler) TambahButir(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	g, ok := idDariPath(w, r, "grup")
	if !ok {
		return
	}
	h.aksiMK(w, r, id, h.Svc.TambahButir(r.Context(), id, g,
		angka(r.PostFormValue("prereq_course_id")), r.PostFormValue("min_letter")))
}

func (h *AdminHandler) HapusButir(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	b, ok := idDariPath(w, r, "butir")
	if !ok {
		return
	}
	h.aksiMK(w, r, id, h.Svc.HapusButir(r.Context(), b))
}

func (h *AdminHandler) TambahKurikulum(w http.ResponseWriter, r *http.Request) {
	c := model.Curriculum{Name: strings.TrimSpace(r.PostFormValue("name")),
		YearStart: angka(r.PostFormValue("year_start")), IsActive: r.PostFormValue("is_active") == "1"}
	if err := h.Svc.TambahKurikulum(r.Context(), c); err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			http.Error(w, pesan, http.StatusUnprocessableEntity)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, "/admin/mk", "dibuat")
}

func (h *AdminHandler) Kesetaraan(w http.ResponseWriter, r *http.Request) {
	h.tampilKesetaraan(w, r, http.StatusOK, "")
}

func (h *AdminHandler) tampilKesetaraan(w http.ResponseWriter, r *http.Request, status int, galat string) {
	ctx := r.Context()
	daftar, err := h.Svc.Kesetaraan(ctx)
	if err != nil {
		gagal(w, err)
		return
	}
	mk, err := h.Svc.MataKuliah(ctx, "", 0)
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, r, status, "admin/kesetaraan.html", Halaman{Title: "Kesetaraan kurikulum", Menu: "setara",
		Galat: galat, Data: map[string]any{"Daftar": daftar, "MK": mk}})
}

func (h *AdminHandler) TambahKesetaraan(w http.ResponseWriter, r *http.Request) {
	err := h.Svc.TambahKesetaraan(r.Context(), angka(r.PostFormValue("old_course_id")), angka(r.PostFormValue("new_course_id")))
	if err != nil {
		if pesan, _, ok := pesanValidasi(err); ok {
			h.tampilKesetaraan(w, r, http.StatusUnprocessableEntity, pesan)
			return
		}
		gagal(w, err)
		return
	}
	redirect(w, r, "/admin/kesetaraan", "dibuat")
}

func (h *AdminHandler) HapusKesetaraan(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r, "id")
	if !ok {
		return
	}
	if err := h.Svc.HapusKesetaraan(r.Context(), id); err != nil {
		gagal(w, err)
		return
	}
	redirect(w, r, "/admin/kesetaraan", "dihapus")
}
