// Package domain = aturan budget per kategori pengeluaran (tanpa HTTP/SQL).
//
// Budget hanya menyimpan batasnya. Berapa yang sudah terpakai dihitung dari
// transaksi di rentang periode (lihat frontend budgetUsage), jadi tidak ada
// angka "spent" yang bisa selisih dengan data transaksi.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"financial-tracker/internal/shared/validator"
)

// Period = kapan budget di-reset.
type Period string

const (
	Weekly  Period = "weekly"  // Senin–Minggu
	Monthly Period = "monthly" // bulan kalender
)

func (p Period) Valid() bool { return p == Weekly || p == Monthly }

// MaxLimit = batas atas budget (Rp 1 triliun) — penjaga salah ketik.
const MaxLimit int64 = 1_000_000_000_000

const maxCategoryLen = 50

// ErrNotFound = budget tidak ada (atau bukan milik owner ini).
var ErrNotFound = errors.New("Budget not found")

// DuplicateError = kategori ini sudah punya budget.
type DuplicateError struct{ Category string }

func (e DuplicateError) Error() string {
	return fmt.Sprintf("%s already has a budget", e.Category)
}

// Budget = batas pengeluaran satu kategori per periode.
type Budget struct {
	ID        string
	OwnerID   string
	Category  string
	Period    Period
	Limit     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Params = input membuat budget.
type Params struct {
	Category string
	Period   Period
	Limit    int64
}

// New memvalidasi p lalu membuat Budget.
func New(id, ownerID string, p Params, now time.Time) (Budget, error) {
	p.Category = strings.TrimSpace(p.Category)
	v := validator.New()
	v.Check(p.Category != "", "category", "is required")
	v.Check(utf8.RuneCountInString(p.Category) <= maxCategoryLen, "category", "is too long")
	checkLimits(v, p.Period, p.Limit)
	if err := v.Err(); err != nil {
		return Budget{}, err
	}
	return Budget{
		ID:        id,
		OwnerID:   ownerID,
		Category:  p.Category,
		Period:    p.Period,
		Limit:     p.Limit,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update mengganti periode & batas. Kategori tidak bisa diganti — hapus lalu
// buat baru kalau perlu.
func (b *Budget) Update(period Period, limit int64, now time.Time) error {
	v := validator.New()
	checkLimits(v, period, limit)
	if err := v.Err(); err != nil {
		return err
	}
	b.Period, b.Limit, b.UpdatedAt = period, limit, now
	return nil
}

func checkLimits(v *validator.Validator, period Period, limit int64) {
	v.Check(period.Valid(), "period", "must be weekly or monthly")
	v.Check(limit > 0, "limit", "must be greater than 0")
	v.Check(limit <= MaxLimit, "limit", "is too large")
}
