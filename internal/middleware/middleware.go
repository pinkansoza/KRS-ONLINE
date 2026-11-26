package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"time"

	"krs-online/internal/model"
	"krs-online/internal/service"
)

// Logging mencatat method, path, status, dan lama proses setiap request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mulai := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, sw.status, time.Since(mulai))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Recover mencegah satu panic mematikan seluruh server.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("panic %s %s: %v", r.Method, r.URL.Path, v)
				http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Keamanan menambahkan header keamanan dasar dan membatasi ukuran body.
func Keamanan(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:")
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // maks 2 MB
		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Autentikasi & otorisasi

type ctxKey struct{} // tipe sendiri agar tidak bentrok dengan key lain

type Auth struct {
	Svc *service.AuthService
}

// RequireLogin: semua route privat wajib login. User yang login
// dititipkan di context request.
func (a *Auth) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		u, err := a.Svc.UserDariToken(r.Context(), c.Value)
		if errors.Is(err, model.ErrNotFound) {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Printf("cek session: %v", err)
			http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
			return
		}
		// halaman privat jangan disimpan di cache browser (tombol back setelah logout)
		w.Header().Set("Cache-Control", "no-store")
		ctx := context.WithValue(r.Context(), ctxKey{}, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CSRF: setiap POST di area privat wajib membawa token milik session-nya.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			u, _ := UserDari(r.Context())
			token := r.PostFormValue("csrf_token")
			if u.CSRF == "" || subtle.ConstantTimeCompare([]byte(token), []byte(u.CSRF)) != 1 {
				http.Error(w, "Token formulir tidak valid. Muat ulang halaman lalu coba lagi.", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Peran membatasi route hanya untuk peran tertentu: 403 kalau sudah login
// tetapi tidak berhak.
func Peran(role string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserDari(r.Context())
		if !ok || u.Role != role {
			http.Error(w, "Anda tidak berhak membuka halaman ini", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func UserDari(ctx context.Context) (model.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(model.User)
	return u, ok
}