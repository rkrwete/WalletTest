package config

import (
	"net"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	App AppConfig
	DB  DBConfig
}

type AppConfig struct {
	Port 			string
	MigrationsPath 	string
}

type DBConfig struct {
	User     string
	Password string
	NameDB   string
	Host     string
	Port     string
}

func (dbc DBConfig) DSN() string {
	url := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbc.User, dbc.Password),
		Host:     net.JoinHostPort(dbc.Host, dbc.Port),
		Path:     dbc.NameDB,
		RawQuery: "sslmode=disable",
	}

	return url.String()
}

func Load() *Config {
	_ = godotenv.Load("config.env")

	cfg := &Config{
		App: AppConfig{
			Port: getStr("APP_PORT", "80"),
			MigrationsPath: getStr("MIGRATIONS_PATH", "migrations"),
		},
		DB: DBConfig{
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			NameDB:   os.Getenv("POSTGRES_DB"),
			Host:     getStr("DB_HOST", "localhosh"),
			Port:     getStr("DB_PORT", "5432"),
		},
	}

	return cfg
}

func getStr(key, def string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}

	return def
}
