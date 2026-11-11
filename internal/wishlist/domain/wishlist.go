// Package domain = aturan wishlist (tanpa HTTP/SQL).
//
// Wishlist = barang yang ingin dibeli. Dipakai frontend sebagai satuan
// konversi: "uang yang kamu simpan = 2× sepatu lari".
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"financial-tracker/internal/shared/validator"
)

// MaxPrice = batas atas harga (Rp 1 triliun) — penjaga salah ketik.
const MaxPrice int64 = 1_000_000_000_000

const maxNameLen = 80

// ErrNotFound = item tidak ada (atau bukan milik owner ini).
var ErrNotFound = errors.New("Wishlist item not found")

// Item = satu barang di wishlist.
type Item struct {
	ID        string
	OwnerID   string
	Name      string
	Price     int64
	CreatedAt time.Time
}

// Params = input membuat item.
type Params struct {
	Name  string
	Price int64
}

// New memvalidasi p lalu membuat Item.
func New(id, ownerID string, p Params, now time.Time) (Item, error) {
	p.Name = strings.TrimSpace(p.Name)
	v := validator.New()
	v.Check(p.Name != "", "name", "is required")
	v.Check(utf8.RuneCountInString(p.Name) <= maxNameLen, "name", "is too long")
	v.Check(p.Price > 0, "price", "must be greater than 0")
	v.Check(p.Price <= MaxPrice, "price", "is too large")
	if err := v.Err(); err != nil {
		return Item{}, err
	}
	return Item{ID: id, OwnerID: ownerID, Name: p.Name, Price: p.Price, CreatedAt: now}, nil
}
