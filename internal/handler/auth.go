package handler

import (
	"errors"
	"log"
	"net/http"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

// AuthForm = data untuk halaman login (email yang diketik + pesan error).
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
		log.Printf("login: %v", err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
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

// Beranda SEMENTARA: membaca cookie langsung. Di langkah 7 tugas ini
// diambil alih middleware, dan fungsi ini diganti.
func (h *AuthHandler) Beranda(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	u, err := h.Svc.UserDariToken(r.Context(), c.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.View.Tampil(w, r, http.StatusOK, "akun/profil.html",
		Halaman{Title: "Akun saya", Menu: "profil", User: u})
}