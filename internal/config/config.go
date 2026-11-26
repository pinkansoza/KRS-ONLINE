package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr         string
	DBDSN        string
	CookieSecure bool
	SKSBaru      int
}

func Load() (Config, error) {
	_ = godotenv.Load() // baca .env jika ada

	cfg := Config{
		Addr:         getEnv("ADDR", ":8080"),
		DBDSN:        os.Getenv("DB_DSN"),
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
		SKSBaru:      20,
	}
	if v, err := strconv.Atoi(os.Getenv("SKS_MAHASISWA_BARU")); err == nil && v > 0 {
		cfg.SKSBaru = v
	}
	if cfg.DBDSN == "" {
		return cfg, errors.New("DB_DSN belum di-set (cek file .env)")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}