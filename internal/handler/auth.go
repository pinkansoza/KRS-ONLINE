package handler

import (
	"errors"
	"net/http"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

type AuthForm struct {
	Email string
	Error string
}

type AuthHandler struct {
	Svc          *service.AuthService
	View         *Renderer
	CookieSecure bool
}

func (h *AuthHandler) FormLogin(w http.ResponseWriter, r *http.Request) {
	// sudah login? langsung ke beranda
	if c, err := r.Cookie("session"); err == nil {
		if _, err := h.Svc.UserDariToken(r.Context(), c.Value); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	h.View.Tampil(w, r, http.StatusOK, "auth/login.html", Halaman{Title: "Masuk", Data: AuthForm{}})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	token, err := h.Svc.Login(r.Context(), email, r.PostFormValue("password"))
	if errors.Is(err, model.ErrLoginGagal) {
		// pesan umum agar tidak membocorkan email yang terdaftar
		h.View.Tampil(w, r, http.StatusUnauthorized, "auth/login.html", Halaman{Title: "Masuk",
			Data: AuthForm{Email: email, Error: "Email atau password salah."}})
		return
	}
	if err != nil {
		gagal(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 hari
		HttpOnly: true,             // tidak bisa dibaca JavaScript
		Secure:   h.CookieSecure,   // true di production (HTTPS)
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session"); err == nil {
		h.Svc.Logout(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	redirect(w, r, "/login", "keluar_app")
}

// Beranda mengarahkan sesuai users.role.
func (h *AuthHandler) Beranda(w http.ResponseWriter, r *http.Request) {
	switch user(r).Role {
	case model.RoleMahasiswa:
		http.Redirect(w, r, "/mhs", http.StatusSeeOther)
	case model.RoleDosen:
		http.Redirect(w, r, "/dosen", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func (h *AuthHandler) FormPassword(w http.ResponseWriter, r *http.Request) {
	h.View.Tampil(w, r, http.StatusOK, "akun/password.html", Halaman{Title: "Ganti password", Menu: "akun"})
}

func (h *AuthHandler) GantiPassword(w http.ResponseWriter, r *http.Request) {
	err := h.Svc.GantiPassword(r.Context(), user(r).ID,
		r.PostFormValue("lama"), r.PostFormValue("baru"), r.PostFormValue("ulang"))
	if err != nil {
		if pesan, fe, ok := pesanValidasi(err); ok {
			h.View.Tampil(w, r, http.StatusUnprocessableEntity, "akun/password.html",
				Halaman{Title: "Ganti password", Menu: "akun", Galat: pesan, Errors: fe})
			return
		}
		gagal(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	redirect(w, r, "/login", "password")
}

// Profil menampilkan akun yang sedang login: nama, email, dan peran.
func (h *AuthHandler) Profil(w http.ResponseWriter, r *http.Request) {
	h.View.Tampil(w, r, http.StatusOK, "akun/profil.html", Halaman{Title: "Akun saya", Menu: "profil"})
}

func (h *AuthHandler) Notifikasi(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.Svc.Notifikasi(r.Context(), user(r))
	if err != nil {
		gagal(w, err)
		return
	}
	h.View.Tampil(w, r, http.StatusOK, "notifikasi.html", Halaman{Title: "Notifikasi", Menu: "notif", Data: daftar})
}

func (h *AuthHandler) TandaiDibaca(w http.ResponseWriter, r *http.Request) {
	if err := h.Svc.TandaiDibaca(r.Context(), user(r)); err != nil {
		gagal(w, err)
		return
	}
	redirect(w, r, "/notifikasi", "dibaca")
}