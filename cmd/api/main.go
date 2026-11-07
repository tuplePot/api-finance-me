// Command api = server HTTP Financial Tracker.
//
// main.go adalah composition root: satu-satunya tempat yang mengenal semua
// package. Dependency dirakit dari dalam ke luar — repository → usecase →
// handler → router — lalu server dijalankan.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Data zona waktu ikut di-embed: Windows / container minimal sering tidak
	// punya tzdata, padahal "hari ini" dihitung di Asia/Jakarta.
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"

	budgethandler "financial-tracker/internal/budget/handler"
	budgetrepo "financial-tracker/internal/budget/repository"
	budgetuc "financial-tracker/internal/budget/usecase"
	challengehandler "financial-tracker/internal/challenge/handler"
	challengerepo "financial-tracker/internal/challenge/repository"
	challengeuc "financial-tracker/internal/challenge/usecase"
	"financial-tracker/internal/shared/clock"
	transactionhandler "financial-tracker/internal/transaction/handler"
	transactionrepo "financial-tracker/internal/transaction/repository"
	transactionuc "financial-tracker/internal/transaction/usecase"
	"financial-tracker/migrations"
	"financial-tracker/platform/config"
	"financial-tracker/platform/httpserver"
	"financial-tracker/platform/logger"
	"financial-tracker/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server berhenti dengan error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ---- 1. Migrasi (koneksi direct) ----
	if cfg.AutoMigrate {
		if err := migrate(ctx, cfg.MigrationURL, log); err != nil {
			return err
		}
	}

	// ---- 2. Database (koneksi pooled) ----
	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// ---- 3. Repository → usecase → handler ----
	clk := clock.New(cfg.Location)
	transactions := transactionhandler.New(transactionuc.New(transactionrepo.NewPostgres(pool), clk))
	budgets := budgethandler.New(budgetuc.New(budgetrepo.NewPostgres(pool), clk))
	challenges := challengehandler.New(challengeuc.New(challengerepo.NewPostgres(pool), clk))

	// ---- 4. Router ----
	r := httpserver.NewRouter(httpserver.Options{Logger: log, CORSOrigins: cfg.CORSOrigins})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			httpserver.Fail(w, http.StatusServiceUnavailable, "db_unavailable", "Database is not reachable")
			return
		}
		httpserver.OK(w, http.StatusOK, "OK", map[string]string{"status": "ok"})
	})
	r.Route("/api", func(api chi.Router) {
		api.Use(httpserver.APIKey(cfg.APIKey, cfg.OwnerID))
		transactions.Routes(api)
		budgets.Routes(api)
		challenges.Routes(api)
	})

	// ---- 5. Jalankan server + graceful shutdown ----
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("server berjalan", "addr", srv.Addr, "timezone", cfg.Location.String())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("mematikan server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func migrate(ctx context.Context, url string, log *slog.Logger) error {
	pool, err := postgres.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	applied, err := postgres.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		log.Info("migrasi database dijalankan", "files", applied)
	}
	return nil
}
