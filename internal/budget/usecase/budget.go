// Package usecase = alur aplikasi budget.
package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"financial-tracker/internal/budget/domain"
	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
)

// Repository = penyimpanan budget.
type Repository interface {
	List(ctx context.Context, ownerID string) ([]domain.Budget, error)
	Get(ctx context.Context, ownerID, id string) (domain.Budget, error)
	// Create mengembalikan domain.DuplicateError jika kategori sudah ada.
	Create(ctx context.Context, b domain.Budget) error
	Update(ctx context.Context, b domain.Budget) error
	Delete(ctx context.Context, ownerID, id string) error
}

// Usecase mengelola budget.
type Usecase struct {
	repo  Repository
	clock clock.Clock
	newID func() string
}

func New(repo Repository, c clock.Clock) *Usecase {
	return &Usecase{repo: repo, clock: c, newID: uuid.NewString}
}

func (uc *Usecase) List(ctx context.Context, ownerID string) ([]domain.Budget, error) {
	return uc.repo.List(ctx, ownerID)
}

func (uc *Usecase) Create(ctx context.Context, ownerID string, p domain.Params) (domain.Budget, error) {
	b, err := domain.New(uc.newID(), ownerID, p, uc.clock.Now())
	if err != nil {
		return domain.Budget{}, err
	}
	if err := uc.repo.Create(ctx, b); err != nil {
		return domain.Budget{}, wrap(err)
	}
	return b, nil
}

func (uc *Usecase) Update(ctx context.Context, ownerID, id string, period domain.Period, limit int64) (domain.Budget, error) {
	b, err := uc.repo.Get(ctx, ownerID, id)
	if err != nil {
		return domain.Budget{}, wrap(err)
	}
	if err := b.Update(period, limit, uc.clock.Now()); err != nil {
		return domain.Budget{}, err
	}
	if err := uc.repo.Update(ctx, b); err != nil {
		return domain.Budget{}, wrap(err)
	}
	return b, nil
}

func (uc *Usecase) Delete(ctx context.Context, ownerID, id string) error {
	return wrap(uc.repo.Delete(ctx, ownerID, id))
}

// wrap memberi jenis HTTP pada error domain.
func wrap(err error) error {
	var dup domain.DuplicateError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return apperror.NotFound(err)
	case errors.As(err, &dup):
		return apperror.Conflict(err)
	default:
		return err
	}
}
