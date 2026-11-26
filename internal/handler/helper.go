package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"krs-online/internal/middleware"
	"krs-online/internal/model"
	"krs-online/internal/service"
)

// gagal menerjemahkan error dari service menjadi status HTTP.
// Detail error database hanya ditulis ke log, tidak ditampilkan ke user.
func gagal(w http.ResponseWriter, err error) {
	var ve *service.ValidasiError
	var fe service.FormError
	switch {
	case errors.As(err, &ve):
		http.Error(w, ve.Pesan, http.StatusBadRequest)
	case errors.As(err, &fe):
		http.Error(w, fe.Error(), http.StatusBadRequest)
	case errors.Is(err, model.ErrNotFound):
		http.Error(w, "Data tidak ditemukan", http.StatusNotFound)
	case errors.Is(err, model.ErrTerlarang):
		http.Error(w, "Anda tidak berhak mengakses data ini", http.StatusForbidden)
	default:
		log.Printf("error: %v", err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
	}
}

// pesanValidasi mengembalikan pesan untuk ditampilkan di halaman bila err
// adalah kesalahan input; ok=false berarti error lain (404/403/500).
func pesanValidasi(err error) (string, map[string]string, bool) {
	var ve *service.ValidasiError
	var fe service.FormError
	switch {
	case errors.As(err, &ve):
		return ve.Pesan, nil, true
	case errors.As(err, &fe):
		return "Periksa kembali isian yang ditandai.", fe, true
	}
	return "", nil, false
}

// idDariPath mengambil {id} dari URL (r.PathValue, pola route Go 1.22).
func idDariPath(w http.ResponseWriter, r *http.Request, nama string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(nama))
	if err != nil || id < 1 {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// angka: input form selalu string, ubah ke int (0 kalau kosong/salah).
func angka(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// angkaQuery membaca query string dengan nilai default.
func angkaQuery(r *http.Request, key string, def int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || n < 0 {
		return def
	}
	return n
}

// waktuForm membaca <input type="datetime-local"> (zona waktu lokal server).
func waktuForm(r *http.Request, key string) (time.Time, bool) {
	v := strings.TrimSpace(r.PostFormValue(key))
	if v == "" {
		return time.Time{}, true
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", v, time.Local)
	return t, err == nil
}

func user(r *http.Request) model.User {
	u, _ := middleware.UserDari(r.Context())
	return u
}

// redirect = pola Post/Redirect/Get dengan kode pesan (303 See Other).
func redirect(w http.ResponseWriter, r *http.Request, path, pesan string) {
	if pesan != "" {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		path += sep + "pesan=" + pesan
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}