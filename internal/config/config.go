package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecret            string
}

func New() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "server address and port")
	flag.StringVar(&cfg.DatabaseURI, "d", "postgres://postgres:postgres@localhost:5432/gophermart?sslmode=disable", "database URI")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "http://localhost:8081", "accrual system address")
	flag.StringVar(&cfg.JWTSecret, "s", "super-secret-jwt-key", "jwt signing key")
	cfg.JWTSecret = getEnv("JWT_SECRET", cfg.JWTSecret)
	flag.Parse()

	cfg.RunAddress = getEnv("RUN_ADDRESS", cfg.RunAddress)
	cfg.DatabaseURI = getEnv("DATABASE_URI", cfg.DatabaseURI)
	cfg.AccrualSystemAddress = getEnv("ACCRUAL_SYSTEM_ADDRESS", cfg.AccrualSystemAddress)
	cfg.JWTSecret = getEnv("JWT_SECRET", "super-secret-jwt-key")

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
