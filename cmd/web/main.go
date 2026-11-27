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
	krsSvc := &service.KRSService{R: repo, SKSBaru: cfg.SKSBaru}
	adminSvc := &service.AdminService{R: repo, KRS: krsSvc} // tahap 5: KRS dipakai untuk promosi daftar tunggu

	// --- handler ---
	view := handler.NewRenderer("templates")
	view.Semester = func(ctx context.Context) string {
		s, err := repo.Semester.Aktif(ctx)
		if err != nil {
			return "-"
		}
		return s.Label()
	}
	authH := &handler.AuthHandler{Svc: authSvc, View: view, CookieSecure: cfg.CookieSecure}
	mhs := &handler.MahasiswaHandler{Svc: krsSvc, View: view}
	dosen := &handler.DosenHandler{Svc: krsSvc, View: view}
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
	privat.Handle("GET /mhs", M(mhs.Dashboard))
	privat.Handle("GET /mhs/katalog", M(mhs.Katalog))
	privat.Handle("GET /mhs/krs", M(mhs.KRS))
	privat.Handle("POST /mhs/krs/tambah", M(mhs.Tambah))
	privat.Handle("POST /mhs/krs/ajukan", M(mhs.Ajukan))
	privat.Handle("POST /mhs/krs/item/{id}/batal", M(mhs.Batal))
	privat.Handle("GET /mhs/tunggu", M(mhs.DaftarTunggu))
	privat.Handle("POST /mhs/tunggu", M(mhs.MasukTunggu))
	privat.Handle("POST /mhs/tunggu/{id}/keluar", M(mhs.KeluarTunggu))
	privat.Handle("GET /mhs/transkrip", M(mhs.Transkrip))

	// Dosen wali
	privat.Handle("GET /dosen", D(dosen.Daftar))
	privat.Handle("GET /dosen/krs/{id}", D(dosen.Detail))
	privat.Handle("POST /dosen/krs/{id}/setujui", D(dosen.Setujui))
	privat.Handle("POST /dosen/krs/{id}/tolak", D(dosen.Tolak))
	privat.Handle("POST /dosen/krs/{id}/item/{item}/putuskan", D(dosen.Putuskan))
	privat.Handle("GET /dosen/mahasiswa/{id}", D(dosen.Transkrip))

	// Admin prodi
	privat.Handle("GET /admin", A(admin.Dashboard))
	privat.Handle("GET /admin/semester/baru", A(admin.FormSemesterBaru))
	privat.Handle("POST /admin/semester", A(admin.SimpanSemester))
	privat.Handle("GET /admin/semester/{id}/edit", A(admin.FormPeriode))
	privat.Handle("POST /admin/semester/{id}/edit", A(admin.UbahPeriode))
	privat.Handle("POST /admin/semester/{id}/aktifkan", A(admin.AktifkanSemester))

	privat.Handle("GET /admin/mk", A(admin.DaftarMK))
	privat.Handle("GET /admin/mk/baru", A(admin.FormMKBaru))
	privat.Handle("POST /admin/mk", A(admin.SimpanMK))
	privat.Handle("GET /admin/mk/{id}", A(admin.DetailMK))
	privat.Handle("GET /admin/mk/{id}/edit", A(admin.FormEditMK))
	privat.Handle("POST /admin/mk/{id}/edit", A(admin.UbahMK))
	privat.Handle("POST /admin/mk/{id}/aktif", A(admin.AktifMK))
	privat.Handle("POST /admin/mk/{id}/grup", A(admin.TambahGrup))
	privat.Handle("POST /admin/mk/{id}/grup/{grup}/hapus", A(admin.HapusGrup))
	privat.Handle("POST /admin/mk/{id}/grup/{grup}/butir", A(admin.TambahButir))
	privat.Handle("POST /admin/mk/{id}/butir/{butir}/hapus", A(admin.HapusButir))
	privat.Handle("POST /admin/kurikulum", A(admin.TambahKurikulum))
	privat.Handle("GET /admin/kesetaraan", A(admin.Kesetaraan))
	privat.Handle("POST /admin/kesetaraan", A(admin.TambahKesetaraan))
	privat.Handle("POST /admin/kesetaraan/{id}/hapus", A(admin.HapusKesetaraan))

	privat.Handle("GET /admin/kelas", A(admin.DaftarKelas))
	privat.Handle("GET /admin/kelas/baru", A(admin.FormKelasBaru))
	privat.Handle("POST /admin/kelas", A(admin.SimpanKelas))
	privat.Handle("GET /admin/kelas/{id}", A(admin.DetailKelas))
	privat.Handle("GET /admin/kelas/{id}/edit", A(admin.FormEditKelas))
	privat.Handle("POST /admin/kelas/{id}/edit", A(admin.UbahKelas))
	privat.Handle("POST /admin/kelas/{id}/hapus", A(admin.HapusKelas))
	privat.Handle("POST /admin/kelas/{id}/jadwal", A(admin.TambahJadwal))
	privat.Handle("POST /admin/kelas/{id}/jadwal/{jadwal}/hapus", A(admin.HapusJadwal))

	privat.Handle("GET /admin/mahasiswa", A(admin.DaftarMhs))
	privat.Handle("GET /admin/mahasiswa/baru", A(admin.FormMhsBaru))
	privat.Handle("POST /admin/mahasiswa", A(admin.SimpanMhs))
	privat.Handle("GET /admin/mahasiswa/{id}/edit", A(admin.FormEditMhs))
	privat.Handle("POST /admin/mahasiswa/{id}/edit", A(admin.UbahMhs))

	privat.Handle("GET /admin/dosen", A(admin.DaftarDosen))
	privat.Handle("GET /admin/dosen/baru", A(admin.FormDosenBaru))
	privat.Handle("POST /admin/dosen", A(admin.SimpanDosen))

	privat.Handle("GET /admin/nilai", A(admin.Nilai))
	privat.Handle("POST /admin/nilai", A(admin.SimpanNilai))
	privat.Handle("POST /admin/nilai/{id}/hapus", A(admin.HapusNilai))
	privat.Handle("POST /admin/nilai/impor", A(admin.ImporNilai))

	privat.Handle("GET /admin/rekap", A(admin.Rekap))
	privat.Handle("GET /admin/rekap.json", A(admin.RekapJSON))
	privat.Handle("GET /admin/rekap/kelas/{id}", A(admin.RekapKelas))

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
