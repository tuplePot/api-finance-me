// Package domain = aturan saving challenge (tanpa HTTP/SQL).
//
// Contoh: "Skip the yogurt" — yogurt Rp 70.000 (temptation) senilai 2× makan
// malam Rp 35.000 (goal). Setiap "skip" menambah savedAmount sebesar harga
// temptation; "slip" (tetap beli) dicatat jujur tapi tidak menambah apa-apa.
// Saat savedAmount mencapai targetAmount sebelum deadline → status success.
// Lewat deadline → failed.
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"financial-tracker/internal/shared/validator"
)

// Status = keadaan challenge.
type Status string

const (
	Active  Status = "active"
	Success Status = "success"
	Failed  Status = "failed"
)

// LogKind = jenis catatan harian.
type LogKind string

const (
	Skip LogKind = "skip" // berhasil menahan diri → harga temptation dihitung tersimpan
	Slip LogKind = "slip" // tetap beli → tidak menyimpan apa-apa
)

func (k LogKind) Valid() bool { return k == Skip || k == Slip }

const (
	maxTitleLen       = 80
	maxNameLen        = 50
	maxPrice    int64 = 1_000_000_000_000
)

var (
	ErrNotFound = errors.New("Challenge not found")
	// ErrClosed = challenge sudah success / failed, tidak bisa dicatat lagi.
	ErrClosed = errors.New("This challenge is already closed")
)

// Item = barang dengan harga, mis. {Yogurt, 70000}.
type Item struct {
	Name  string
	Price int64
}

// Log = satu catatan skip / slip.
type Log struct {
	ID        string
	Kind      LogKind
	Amount    int64 // yang tersimpan oleh catatan ini (0 untuk slip)
	Date      time.Time
	CreatedAt time.Time
}

// Challenge = satu saving challenge beserta catatannya (terbaru dulu).
type Challenge struct {
	ID           string
	OwnerID      string
	Title        string
	Temptation   Item
	Goal         Item
	TargetAmount int64
	Deadline     time.Time // inklusif
	SavedAmount  int64
	Logs         []Log
	Status       Status
	CompletedAt  *time.Time
	CreatedAt    time.Time
}

// Params = input membuat challenge.
type Params struct {
	Title        string
	Temptation   Item
	Goal         Item
	TargetAmount int64
	Deadline     time.Time
}

// New memvalidasi p lalu membuat challenge aktif. Deadline tidak boleh di
// masa lalu (today = tanggal hari ini di zona waktu pengguna).
func New(id, ownerID string, p Params, today, now time.Time) (Challenge, error) {
	p.Title = strings.TrimSpace(p.Title)
	p.Temptation.Name = strings.TrimSpace(p.Temptation.Name)
	p.Goal.Name = strings.TrimSpace(p.Goal.Name)

	v := validator.New()
	v.Check(p.Title != "", "title", "is required")
	v.Check(utf8.RuneCountInString(p.Title) <= maxTitleLen, "title", "is too long")
	checkItem(v, "temptation", p.Temptation)
	checkItem(v, "goal", p.Goal)
	v.Check(p.TargetAmount > 0, "targetAmount", "must be greater than 0")
	v.Check(p.TargetAmount <= maxPrice, "targetAmount", "is too large")
	v.Check(!p.Deadline.IsZero(), "deadline", "is required")
	v.Check(!p.Deadline.Before(today), "deadline", "can't be in the past")
	if err := v.Err(); err != nil {
		return Challenge{}, err
	}

	return Challenge{
		ID:           id,
		OwnerID:      ownerID,
		Title:        p.Title,
		Temptation:   p.Temptation,
		Goal:         p.Goal,
		TargetAmount: p.TargetAmount,
		Deadline:     p.Deadline,
		Logs:         []Log{},
		Status:       Active,
		CreatedAt:    now,
	}, nil
}

func checkItem(v *validator.Validator, field string, it Item) {
	v.Check(it.Name != "", field+".name", "is required")
	v.Check(utf8.RuneCountInString(it.Name) <= maxNameLen, field+".name", "is too long")
	v.Check(it.Price > 0, field+".price", "must be greater than 0")
	v.Check(it.Price <= maxPrice, field+".price", "is too large")
}

// Settle menandai challenge aktif yang sudah lewat deadline sebagai failed.
// Mengembalikan true jika status berubah.
func (c *Challenge) Settle(today time.Time) bool {
	if c.Status == Active && c.Deadline.Before(today) {
		c.Status = Failed
		return true
	}
	return false
}

// Record mencatat skip / slip. Jika catatan ini membuat target tercapai,
// challenge menjadi success dan completed = true.
func (c *Challenge) Record(logID string, kind LogKind, today, now time.Time) (log Log, completed bool, err error) {
	if !kind.Valid() {
		return Log{}, false, errors.New("kind must be skip or slip")
	}
	c.Settle(today)
	if c.Status != Active {
		return Log{}, false, ErrClosed
	}

	var amount int64
	if kind == Skip {
		amount = c.Temptation.Price
	}
	log = Log{ID: logID, Kind: kind, Amount: amount, Date: today, CreatedAt: now}
	c.Logs = append([]Log{log}, c.Logs...)
	c.SavedAmount += amount

	if c.SavedAmount >= c.TargetAmount {
		c.Status = Success
		c.CompletedAt = &now
		completed = true
	}
	return log, completed, nil
}
