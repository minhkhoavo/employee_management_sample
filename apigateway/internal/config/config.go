package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var DefaultEnvConfig *envConfig

type envConfig struct {
	// database config
	DB_HOST              string
	DB_PORT              int
	DB_USER              string
	DB_PASSWORD          string
	DB_NAME              string
	DB_SSL_MODE          string
	DB_CONN_MAX_LIFETIME time.Duration
	DB_MAX_IDLE_CONNS    int
	DB_MAX_OPEN_CONNS    int
	// logger config
	LOG_FILE_PATH string
}

func LoadEnvConfig() error {
	if err := godotenv.Load(); err != nil {
		return err
	}

	DefaultEnvConfig = &envConfig{
		DB_HOST:              getEnvString("DB_HOST", "localhost"),
		DB_PORT:              getEnvInt("DB_PORT", 5432),
		DB_USER:              getEnvString("DB_USER", "postgres"),
		DB_PASSWORD:          getEnvString("DB_PASSWORD", "postgres"),
		DB_NAME:              getEnvString("DB_NAME", "postgres"),
		DB_SSL_MODE:          getEnvString("DB_SSL_MODE", "disable"),
		DB_CONN_MAX_LIFETIME: getEnvDuration("DB_CONN_MAX_LIFETIME", 20*time.Minute),
		DB_MAX_IDLE_CONNS:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DB_MAX_OPEN_CONNS:    getEnvInt("DB_MAX_OPEN_CONNS", 100),
		LOG_FILE_PATH:        getEnvString("LOG_FILE_PATH", ""),
	}
	return nil
}

func getEnvString(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		if i, err := strconv.Atoi(val); err == nil {
			return time.Duration(i) * time.Second
		}
	}
	return fallback
}
