// Package httpserver = router Chi + middleware global + helper response JSON.
package httpserver

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"financial-tracker/internal/shared/owner"
)

// Options = pengaturan router.
type Options struct {
	Logger      *slog.Logger
	CORSOrigins []string
}

// NewRouter membuat router dengan middleware: request id, real ip, logging,
// recover panic, CORS.
func NewRouter(opts Options) chi.Router {
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)
	r.Use(cors(opts.CORSOrigins))

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		Fail(w, http.StatusNotFound, "not_found", "Route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		Fail(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	})
	return r
}

// APIKey menolak request tanpa header X-API-Key yang benar, lalu menandai
// request sebagai milik ownerID. Perbandingan constant-time supaya key tidak
// bisa ditebak lewat waktu respons.
func APIKey(key, ownerID string) func(http.Handler) http.Handler {
	want := []byte(key)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := []byte(r.Header.Get("X-API-Key"))
			if subtle.ConstantTimeCompare(got, want) != 1 {
				Fail(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API key")
				return
			}
			next.ServeHTTP(w, r.WithContext(owner.With(r.Context(), ownerID)))
		})
	}
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}

// cors = middleware CORS sederhana. "*" = semua origin (development).
func cors(origins []string) func(http.Handler) http.Handler {
	allowAll := false
	allowed := map[string]bool{}
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "*" {
			allowAll = true
		}
		allowed[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || allowed[origin]) {
				h := w.Header()
				if allowAll {
					h.Set("Access-Control-Allow-Origin", "*")
				} else {
					h.Set("Access-Control-Allow-Origin", origin)
					h.Add("Vary", "Origin")
				}
				h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
				h.Set("Access-Control-Max-Age", "600")
			}
			// Preflight dijawab di sini, sebelum middleware API key.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
