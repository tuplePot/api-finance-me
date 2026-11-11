// Package usecase = alur aplikasi habit.
package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"financial-tracker/internal/habit/domain"
	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
)

// Repository = penyimpanan habit.
type Repository interface {
	List(ctx context.Context, ownerID string) ([]domain.Habit, error)
	Get(ctx context.Context, ownerID, id string) (domain.Habit, error)
	Create(ctx context.Context, h domain.Habit) error
	Update(ctx context.Context, h domain.Habit) error
	Delete(ctx context.Context, ownerID, id string) error
}

// Usecase mengelola habit.
type Usecase struct {
	repo  Repository
	clock clock.Clock
	newID func() string
}

func New(repo Repository, c clock.Clock) *Usecase {
	return &Usecase{repo: repo, clock: c, newID: uuid.NewString}
}

// List = semua habit, urut waktu dibuat.
func (uc *Usecase) List(ctx context.Context, ownerID string) ([]domain.Habit, error) {
	return uc.repo.List(ctx, ownerID)
}

func (uc *Usecase) Create(ctx context.Context, ownerID string, p domain.Params) (domain.Habit, error) {
	h, err := domain.New(uc.newID(), ownerID, p, uc.clock.Now())
	if err != nil {
		return domain.Habit{}, err
	}
	if err := uc.repo.Create(ctx, h); err != nil {
		return domain.Habit{}, wrap(err)
	}
	return h, nil
}

func (uc *Usecase) Update(ctx context.Context, ownerID, id string, patch domain.Patch) (domain.Habit, error) {
	h, err := uc.repo.Get(ctx, ownerID, id)
	if err != nil {
		return domain.Habit{}, wrap(err)
	}
	if err := h.Apply(patch, uc.clock.Now()); err != nil {
		return domain.Habit{}, err
	}
	if err := uc.repo.Update(ctx, h); err != nil {
		return domain.Habit{}, wrap(err)
	}
	return h, nil
}

func (uc *Usecase) Delete(ctx context.Context, ownerID, id string) error {
	return wrap(uc.repo.Delete(ctx, ownerID, id))
}

// wrap memberi jenis HTTP pada error domain.
func wrap(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return apperror.NotFound(err)
	}
	return err
}
