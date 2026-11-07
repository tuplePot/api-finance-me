// Package usecase = alur aplikasi saving challenge.
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"financial-tracker/internal/challenge/domain"
	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
)

// Mutation = perubahan yang harus disimpan setelah satu Record.
type Mutation struct {
	Log domain.Log
}

// Repository = penyimpanan challenge.
type Repository interface {
	// SettleExpired menandai challenge aktif yang lewat deadline sebagai failed.
	SettleExpired(ctx context.Context, ownerID string, today time.Time) error
	List(ctx context.Context, ownerID string) ([]domain.Challenge, error)
	Create(ctx context.Context, c domain.Challenge) error
	Delete(ctx context.Context, ownerID, id string) error
	// Mutate mengunci baris challenge (SELECT ... FOR UPDATE) di dalam satu
	// transaksi, memanggil fn, lalu menyimpan Mutation-nya. fn error → rollback.
	// Kunci ini yang mencegah dua klik "I skipped it" bersamaan saling menimpa savedAmount.
	Mutate(ctx context.Context, ownerID, id string, fn func(c *domain.Challenge) (Mutation, error)) (domain.Challenge, error)
}

// Usecase mengelola challenge.
type Usecase struct {
	repo  Repository
	clock clock.Clock
	newID func() string
}

func New(repo Repository, c clock.Clock) *Usecase {
	return &Usecase{repo: repo, clock: c, newID: uuid.NewString}
}

// List = semua challenge (terbaru dulu), setelah yang kedaluwarsa ditutup.
func (uc *Usecase) List(ctx context.Context, ownerID string) ([]domain.Challenge, error) {
	if err := uc.repo.SettleExpired(ctx, ownerID, uc.clock.Today()); err != nil {
		return nil, err
	}
	return uc.repo.List(ctx, ownerID)
}

func (uc *Usecase) Create(ctx context.Context, ownerID string, p domain.Params) (domain.Challenge, error) {
	c, err := domain.New(uc.newID(), ownerID, p, uc.clock.Today(), uc.clock.Now())
	if err != nil {
		return domain.Challenge{}, err
	}
	if err := uc.repo.Create(ctx, c); err != nil {
		return domain.Challenge{}, err
	}
	return c, nil
}

func (uc *Usecase) Delete(ctx context.Context, ownerID, id string) error {
	return wrap(uc.repo.Delete(ctx, ownerID, id))
}

// Log mencatat skip / slip; challenge otomatis success saat target tercapai.
func (uc *Usecase) Log(ctx context.Context, ownerID, id string, kind domain.LogKind) (domain.Challenge, error) {
	if !kind.Valid() {
		return domain.Challenge{}, apperror.Invalid(errors.New("Kind must be skip or slip"),
			map[string]string{"kind": "must be skip or slip"})
	}
	today, now := uc.clock.Today(), uc.clock.Now()
	c, err := uc.repo.Mutate(ctx, ownerID, id, func(c *domain.Challenge) (Mutation, error) {
		log, _, err := c.Record(uc.newID(), kind, today, now)
		if err != nil {
			return Mutation{}, err
		}
		return Mutation{Log: log}, nil
	})
	return c, wrap(err)
}

func wrap(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return apperror.NotFound(err)
	case errors.Is(err, domain.ErrClosed):
		return apperror.Conflict(err)
	default:
		return err
	}
}
