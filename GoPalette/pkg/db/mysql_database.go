package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/go-sql-driver/mysql"
)

const createDatabaseTimeout = 10 * time.Second

// EnsureMySQLDatabase creates the database named in the DSN before GORM opens it.
func EnsureMySQLDatabase(source string) error {
	cfg, err := mysql.ParseDSN(source)
	if err != nil {
		return fmt.Errorf("parse mysql dsn: %w", err)
	}
	if cfg.DBName == "" {
		return nil
	}
	if !isSafeMySQLIdentifier(cfg.DBName) {
		return fmt.Errorf("invalid mysql database name %q", cfg.DBName)
	}

	dbName := cfg.DBName
	cfg.DBName = ""
	dsn := cfg.FormatDSN()

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql connection: %w", err)
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), createDatabaseTimeout)
	defer cancel()

	query := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		strings.ReplaceAll(dbName, "`", "``"),
	)
	if _, err := sqlDB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create mysql database %q: %w", dbName, err)
	}
	return nil
}

func isSafeMySQLIdentifier(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
}
