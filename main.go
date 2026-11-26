// Program utama: hanya menyambung komponen (materi pertemuan 10).
// Urutan: config -> db -> repository -> service -> handler -> route -> server.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql" // driver mendaftarkan dirinya sebagai "mysql"

	"krs-online/internal/config"
	"krs-online/internal/handler"
	"krs-online/internal/middleware"
	"krs-online/internal/model"
	"krs-online/internal/repository"
	"krs-online/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := bukaDB(cfg.DBDSN)
	if err != nil {
		log.Fatalf("database tidak bisa dihubungi: %v", err)
	}
	defer db.Close()

	// --- repository & service ---
	repo := repository.New(db)
	authSvc := &service.AuthService{Users: repo.User, Sessions: repo.Session, Notif: repo.Notif}
	adminSvc := &service.AdminService{R: repo}

	// --- handler ---
	view := handler.NewRenderer("templates")
	authH := &handler.AuthHandler{Svc: authSvc, View: view, CookieSecure: cfg.CookieSecure}
	admin := &handler.AdminHandler{Svc: adminSvc, View: view}
	authMw := &middleware.Auth{Svc: authSvc}

	// pembatas peran
	M := func(h http.HandlerFunc) http.Handler { return middleware.Peran(model.RoleMahasiswa, h) }
	D := func(h http.HandlerFunc) http.Handler { return middleware.Peran(model.RoleDosen, h) }
	A := func(h http.HandlerFunc) http.Handler { return middleware.Peran(model.RoleAdmin, h) }

	// privat: semua route yang wajib login
	privat := http.NewServeMux()
	privat.HandleFunc("GET /{$}", authH.Beranda)
	privat.HandleFunc("POST /logout", authH.Logout)
	privat.HandleFunc("GET /akun", authH.Profil)
	privat.HandleFunc("GET /akun/password", authH.FormPassword)
	privat.HandleFunc("POST /akun/password", authH.GantiPassword)
	privat.HandleFunc("GET /notifikasi", authH.Notifikasi)
	privat.HandleFunc("POST /notifikasi/dibaca", authH.TandaiDibaca)

	// Mahasiswa
	privat.Handle("GET /mhs", M(authH.Profil)) // beranda sementara; dashboard mahasiswa dibuat di tahap 4

	// Dosen wali
	privat.Handle("GET /dosen", D(authH.Profil)) // beranda sementara; daftar bimbingan dibuat di tahap 5

	// Admin prodi
	privat.Handle("GET /admin", A(authH.Profil)) // beranda sementara; dashboard admin dibuat di tahap 2

	privat.Handle("GET /admin/dosen", A(admin.DaftarDosen))
	privat.Handle("GET /admin/dosen/baru", A(admin.FormDosenBaru))
	privat.Handle("POST /admin/dosen", A(admin.SimpanDosen))

	// publik: boleh diakses tanpa login
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", authH.FormLogin)
	mux.HandleFunc("POST /login", authH.Login)
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	mux.Handle("/", authMw.RequireLogin(middleware.CSRF(privat))) // sisanya wajib login

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           middleware.Logging(middleware.Recover(middleware.Keamanan(mux))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("server jalan di http://localhost%s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())
}

// bukaDB membuat satu connection pool untuk seluruh aplikasi dan
// mengecek koneksi saat startup (lebih baik gagal sekarang daripada saat user datang).
func bukaDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	log.Println("database terhubung")
	return db, nil
}
