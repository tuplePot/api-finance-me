// Package domain = aturan transaksi pemasukan / pengeluaran (tanpa HTTP/SQL).
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"financial-tracker/internal/shared/validator"
)

// Type = jenis transaksi.
type Type string

const (
	Income  Type = "income"
	Expense Type = "expense"
)

// Valid true jika t salah satu jenis yang dikenal.
func (t Type) Valid() bool { return t == Income || t == Expense }

// MaxAmount = batas atas satu transaksi (Rp 1 triliun) — penjaga salah ketik.
const MaxAmount int64 = 1_000_000_000_000

const (
	maxCategoryLen = 50
	maxNoteLen     = 200
)

// ErrNotFound = transaksi tidak ada (atau bukan milik owner ini).
var ErrNotFound = errors.New("Transaction not found")

// Transaction = satu pemasukan / pengeluaran. Amount dalam Rupiah utuh.
type Transaction struct {
	ID        string
	OwnerID   string
	Type      Type
	Amount    int64
	Category  string
	Date      time.Time // tanggal saja (00:00 UTC)
	Note      string
	CreatedAt time.Time
}

// Params = input untuk membuat transaksi.
type Params struct {
	Type     Type
	Amount   int64
	Category string
	Date     time.Time
	Note     string
}

// New memvalidasi p lalu membuat Transaction.
func New(id, ownerID string, p Params, now time.Time) (Transaction, error) {
	p.Category = strings.TrimSpace(p.Category)
	p.Note = strings.TrimSpace(p.Note)

	v := validator.New()
	v.Check(p.Type.Valid(), "type", "must be income or expense")
	v.Check(p.Amount > 0, "amount", "must be greater than 0")
	v.Check(p.Amount <= MaxAmount, "amount", "is too large")
	v.Check(p.Category != "", "category", "is required")
	v.Check(utf8.RuneCountInString(p.Category) <= maxCategoryLen, "category", "is too long")
	v.Check(!p.Date.IsZero(), "date", "is required")
	v.Check(utf8.RuneCountInString(p.Note) <= maxNoteLen, "note", "must be 200 characters or fewer")
	if err := v.Err(); err != nil {
		return Transaction{}, err
	}

	return Transaction{
		ID:        id,
		OwnerID:   ownerID,
		Type:      p.Type,
		Amount:    p.Amount,
		Category:  p.Category,
		Date:      p.Date,
		Note:      p.Note,
		CreatedAt: now,
	}, nil
}
