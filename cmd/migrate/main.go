// Command migrate menjalankan migrasi skema PostgreSQL secara manual.
//
//	go run ./cmd/migrate
//
// Memakai DATABASE_URL_UNPOOLED (koneksi direct Neon), fallback ke
// DATABASE_URL. Server juga otomatis migrasi saat start (AUTO_MIGRATE=true).
package main

import (
	"context"
	"fmt"
	"os"

	"financial-tracker/migrations"
	"financial-tracker/platform/config"
	"financial-tracker/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrasi gagal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := postgres.Connect(ctx, cfg.MigrationURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := postgres.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		fmt.Println("database sudah up to date")
		return nil
	}
	for _, f := range applied {
		fmt.Println("applied:", f)
	}
	return nil
}
