// Package postgres = koneksi PostgreSQL (pgx) + migrasi skema sederhana.
//
// Neon: pakai connection string pooled (host -pooler) untuk server dan
// direct untuk migrasi. pgx memakai prepared statement level protokol, yang
// didukung PgBouncer milik Neon.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect membuka connection pool dan memastikan database bisa dihubungi.
// Timeout ping agak longgar karena compute Neon bisa sedang "tidur"
// (scale-to-zero) dan butuh beberapa detik untuk bangun.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL tidak valid: %w", err)
	}
	cfg.MaxConns = 10
	// Lepas koneksi idle sebelum Neon/PgBouncer menutupnya duluan.
	cfg.MaxConnIdleTime = 4 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("tidak bisa terhubung ke database: %w", err)
	}
	return pool, nil
}

// IsUniqueViolation true jika err berasal dari constraint UNIQUE (23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Migrate menjalankan file *.sql yang belum pernah dijalankan, urut nama file.
// Versi yang sudah jalan dicatat di schema_migrations (idempotent); setiap
// file jalan di dalam transaksi. Mengembalikan file yang baru dijalankan.
func Migrate(ctx context.Context, pool *pgxpool.Pool, files fs.FS) ([]string, error) {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text        PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return nil, fmt.Errorf("membuat schema_migrations: %w", err)
	}

	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	var applied []string
	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
		).Scan(&exists); err != nil {
			return applied, err
		}
		if exists {
			continue
		}
		sqlBytes, err := fs.ReadFile(files, name)
		if err != nil {
			return applied, err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return applied, err
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return applied, fmt.Errorf("migrasi %s gagal: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return applied, err
		}
		if err := tx.Commit(ctx); err != nil {
			return applied, err
		}
		applied = append(applied, name)
	}
	return applied, nil
}
