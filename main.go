package krsonline
import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Application struct {
	DB           *sql.DB
	logger       *log.Logger
	models       *Models
	router       *http.ServeMux
	templateCache map[string]*template.Template
	expositions    map[string]*ExpositionService
}

type Models struct {
	DBModel DBModel
	User    UserModel
	Semester   SemesterModel
	Fakultas   FakultasModel
	Jurusan    JurusanModel
	Kelas      KelasModel
	Matakuliah MatakuliahModel
	
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("no .env file found")
	}

	// Create application
	app := &Application{}

	// Database
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgresql://postgres:postgres@localhost:5432/krs_online?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("error connecting to db:", err)
	}
	app.DB = db
	defer db.Close()

	// Logger
	app.logger = log.New(os.Stdout, "CRON: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Cache
	cache, err := newTemplateCache("./ui/html/*/*.page.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	app.templateCache = cache

	// Models
	app.models = &Models{
		DBModel:    DBModel{DB: db},
		User:       UserModel{DB: db},
		Semester:   SemesterModel{DB: db},
		Fakultas:   FakultasModel{DB: db},
		Jurusan:    JurusanModel{DB: db},
		Kelas:      KelasModel{DB: db},
		Matakuliah: MatakuliahModel{DB: db},
	}

	// Routes
	app.router = http.NewServeMux()
	app.router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	app.router.Handle("/api/", http.StripPrefix("/api/", http.HandlerFunc(app.routesAPI())))
	
	// Routes Web
	fileServer := http.FileServer(http.Dir("./static"))
	app.router.Handle("/static/", http.StripPrefix("/static/", fileServer))

	app.router.HandleFunc("/", app.home)
	app.router.HandleFunc("/login", app.handleLoginShow)
	app.router.HandleFunc("/api/login", app.handleLoginPost)
	app.router.HandleFunc("/logout", app.handleLogout)

	// Admin routes
	app.router.HandleFunc("/admin", app.adminDashboard)
	
	app.router.HandleFunc("/admin/pengguna", app.listPengguna)
	app.router.HandleFunc("/admin/pengguna/baru", app.tambahPengguna)
	app.router.HandleFunc("/admin/pengguna/simpan", app.simpanPengguna)
	app.router.HandleFunc("/admin/pengguna/edit", app.editPengguna)
	app.router.HandleFunc("/admin/pengguna/update", app.updatePengguna)
	app.router.HandleFunc("/admin/pengguna/hapus", app.hapusPengguna)

	app.router.HandleFunc("/admin/fakultas", app.listFakultas)
	app.router.HandleFunc("/admin/fakultas/baru", app.tambahFakultas)
	app.router.HandleFunc("/admin/fakultas/simpan", app.simpanFakultas)
	app.router.HandleFunc("/admin/fakultas/edit", app.editFakultas)
	app.router.HandleFunc("/admin/fakultas/update", app.updateFakultas)
	app.router.HandleFunc("/admin/fakultas/hapus", app.hapusFakultas)

	app.router.HandleFunc("/admin/jurusan", app.listJurusan)
	app.router.HandleFunc("/admin/jurusan/baru", app.tambahJurusan)
	app.router.HandleFunc("/admin/jurusan/simpan", app.simpanJurusan)
	app.router.HandleFunc("/admin/jurusan/edit", app.editJurusan)
	app.router.HandleFunc("/admin/jurusan/update", app.updateJurusan)
	app.router.HandleFunc("/admin/jurusan/hapus", app.hapusJurusan)

	app.router.HandleFunc("/admin/kelas", app.listKelas)
	app.router.HandleFunc("/admin/kelas/baru", app.tambahKelas)
	app.router.HandleFunc("/admin/kelas/simpan", app.simpanKelas)
	app.router.HandleFunc("/admin/kelas/edit", app.editKelas)
	app.router.HandleFunc("/admin/kelas/update", app.updateKelas)
	app.router.HandleFunc("/admin/kelas/hapus", app.hapusKelas)

	app.router.HandleFunc("/admin/matakuliah", app.listMatakuliah)
	app.router.HandleFunc("/admin/matakuliah/baru", app.tambahMatakuliah)
	app.router.HandleFunc("/admin/matakuliah/simpan", app.simpanMatakuliah)
	app.router.HandleFunc("/admin/matakuliah/edit", app.editMatakuliah)
	app.router.HandleFunc("/admin/matakuliah/update", app.updateMatakuliah)
	app.router.HandleFunc("/admin/matakuliah/hapus", app.hapusMatakuliah)

	// Mahasiswa routes
	app.router.HandleFunc("/mhs", app.mhsDashboard)
	
	app.router.HandleFunc("/mhs/krs", app.mhsKRS)
	app.router.HandleFunc("/mhs/krs/simpan", app.mhsKRSSimpan)

	app.router.HandleFunc("/mhs/riwayat-krs", app.mhsRiwayatKRS)
	
	app.router.HandleFunc("/mhs/profile", app.mhsProfile)
	app.router.HandleFunc("/mhs/profile/simpan", app.mhsProfileSimpan)

	// Dosen routes
	app.router.HandleFunc("/dosen", app.dosenDashboard)
	
	app.router.HandleFunc("/dosen/khs", app.dosenKHS)
	app.router.HandleFunc("/dosen/khs/simpan", app.dosenKHSSimpan)

	app.router.HandleFunc("/dosen/riwayat-khs", app.dosenRiwayatKHS)
	
	app.router.HandleFunc("/dosen/profile", app.dosenProfile)
	app.router.HandleFunc("/dosen/profile/simpan", app.dosenProfileSimpan)

	// Expositions (API REST)
	if cfgPath := os.Getenv("EXPOSITIONS_CONFIG"); cfgPath != "" {
