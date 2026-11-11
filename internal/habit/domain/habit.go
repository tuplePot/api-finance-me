// Package domain = aturan "habit yang tidak kamu punya" (tanpa HTTP/SQL).
//
// Habit hanya menyimpan biaya per periode + tanggal mulai dihitung. Uang yang
// tersimpan dihitung client (cost per hari × jumlah hari sejak Since), jadi
// tidak ada angka yang harus di-update tiap hari.
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"financial-tracker/internal/shared/validator"
)

// Frequency = biaya habit dihitung per apa.
type Frequency string

const (
	Daily   Frequency = "daily"
	Weekly  Frequency = "weekly"
	Monthly Frequency = "monthly"
)

func (f Frequency) Valid() bool { return f == Daily || f == Weekly || f == Monthly }

// MaxCost = batas atas biaya (Rp 1 triliun) — penjaga salah ketik.
const MaxCost int64 = 1_000_000_000_000

const (
	maxNameLen = 60
	maxIconLen = 80
)

// minSince = tanggal paling awal yang masuk akal untuk mulai menghitung.
var minSince = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

// ErrNotFound = habit tidak ada (atau bukan milik owner ini).
var ErrNotFound = errors.New("Habit not found")

// Habit = sesuatu yang tidak kamu beli, mis. rokok Rp 35.000 per hari.
type Habit struct {
	ID        string
	OwnerID   string
	Name      string
	Icon      string // nama Iconify, mis. "lucide:cigarette"
	Cost      int64
	Frequency Frequency
	Since     time.Time // tanggal saja (00:00 UTC), inklusif
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Params = input membuat habit.
type Params struct {
	Name      string
	Icon      string
	Cost      int64
	Frequency Frequency
	Since     time.Time
}

// New memvalidasi p lalu membuat Habit.
func New(id, ownerID string, p Params, now time.Time) (Habit, error) {
	p, err := validate(p)
	if err != nil {
		return Habit{}, err
	}
	return Habit{
		ID:        id,
		OwnerID:   ownerID,
		Name:      p.Name,
		Icon:      p.Icon,
		Cost:      p.Cost,
		Frequency: p.Frequency,
		Since:     p.Since,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Patch = perubahan sebagian. Field nil = tidak diubah.
type Patch struct {
	Name      *string
	Icon      *string
	Cost      *int64
	Frequency *Frequency
	Since     *time.Time
}

// Apply menerapkan patch lalu memvalidasi ulang. h tidak berubah jika gagal.
func (h *Habit) Apply(patch Patch, now time.Time) error {
	p := Params{Name: h.Name, Icon: h.Icon, Cost: h.Cost, Frequency: h.Frequency, Since: h.Since}
	if patch.Name != nil {
		p.Name = *patch.Name
	}
	if patch.Icon != nil {
		p.Icon = *patch.Icon
	}
	if patch.Cost != nil {
		p.Cost = *patch.Cost
	}
	if patch.Frequency != nil {
		p.Frequency = *patch.Frequency
	}
	if patch.Since != nil {
		p.Since = *patch.Since
	}
	p, err := validate(p)
	if err != nil {
		return err
	}
	h.Name, h.Icon, h.Cost, h.Frequency, h.Since = p.Name, p.Icon, p.Cost, p.Frequency, p.Since
	h.UpdatedAt = now
	return nil
}

func validate(p Params) (Params, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.Icon = strings.TrimSpace(p.Icon)
	v := validator.New()
	v.Check(p.Name != "", "name", "is required")
	v.Check(utf8.RuneCountInString(p.Name) <= maxNameLen, "name", "is too long")
	v.Check(p.Icon != "", "icon", "is required")
	v.Check(utf8.RuneCountInString(p.Icon) <= maxIconLen, "icon", "is too long")
	v.Check(p.Cost > 0, "cost", "must be greater than 0")
	v.Check(p.Cost <= MaxCost, "cost", "is too large")
	v.Check(p.Frequency.Valid(), "frequency", "must be daily, weekly or monthly")
	v.Check(!p.Since.Before(minSince), "since", "is too far in the past")
	return p, v.Err()
}
