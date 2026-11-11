// Package usecase = alur aplikasi wishlist.
package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
	"financial-tracker/internal/wishlist/domain"
)

// Repository = penyimpanan wishlist.
type Repository interface {
	List(ctx context.Context, ownerID string) ([]domain.Item, error)
	Create(ctx context.Context, it domain.Item) error
	Delete(ctx context.Context, ownerID, id string) error
}

// Usecase mengelola wishlist.
type Usecase struct {
	repo  Repository
	clock clock.Clock
	newID func() string
}

func New(repo Repository, c clock.Clock) *Usecase {
	return &Usecase{repo: repo, clock: c, newID: uuid.NewString}
}

// List = semua item, termurah dulu.
func (uc *Usecase) List(ctx context.Context, ownerID string) ([]domain.Item, error) {
	return uc.repo.List(ctx, ownerID)
}

func (uc *Usecase) Create(ctx context.Context, ownerID string, p domain.Params) (domain.Item, error) {
	it, err := domain.New(uc.newID(), ownerID, p, uc.clock.Now())
	if err != nil {
		return domain.Item{}, err
	}
	if err := uc.repo.Create(ctx, it); err != nil {
		return domain.Item{}, err
	}
	return it, nil
}

func (uc *Usecase) Delete(ctx context.Context, ownerID, id string) error {
	err := uc.repo.Delete(ctx, ownerID, id)
	if errors.Is(err, domain.ErrNotFound) {
		return apperror.NotFound(err)
	}
	return err
}
