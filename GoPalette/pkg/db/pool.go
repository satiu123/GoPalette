package db

import (
	"database/sql"
	"os"
	"strconv"
	"time"
)

const (
	defaultMaxOpenConns    = 80
	defaultMaxIdleConns    = 40
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 10 * time.Minute
)

func ConfigurePool(sqlDB *sql.DB) {
	if sqlDB == nil {
		return
	}
	maxOpen := envInt("MYSQL_MAX_OPEN_CONNS", defaultMaxOpenConns)
	maxIdle := envInt("MYSQL_MAX_IDLE_CONNS", defaultMaxIdleConns)
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(envDuration("MYSQL_CONN_MAX_LIFETIME", defaultConnMaxLifetime))
	sqlDB.SetConnMaxIdleTime(envDuration("MYSQL_CONN_MAX_IDLE_TIME", defaultConnMaxIdleTime))
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
