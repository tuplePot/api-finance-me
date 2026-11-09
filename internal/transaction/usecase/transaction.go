// Package usecase = alur aplikasi transaksi.
package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
	"financial-tracker/internal/transaction/domain"
)

// Repository = penyimpanan transaksi (didefinisikan oleh consumer).
type Repository interface {
	List(ctx context.Context, ownerID string) ([]domain.Transaction, error)
	Get(ctx context.Context, ownerID, id string) (domain.Transaction, error)
	Create(ctx context.Context, t domain.Transaction) error
	Update(ctx context.Context, t domain.Transaction) error
	Delete(ctx context.Context, ownerID, id string) error
}

// Usecase mengelola transaksi.
type Usecase struct {
	repo  Repository
	clock clock.Clock
	newID func() string
}

// New membuat usecase transaksi.
func New(repo Repository, c clock.Clock) *Usecase {
	return &Usecase{repo: repo, clock: c, newID: uuid.NewString}
}

// List = semua transaksi, terbaru dulu.
func (uc *Usecase) List(ctx context.Context, ownerID string) ([]domain.Transaction, error) {
	return uc.repo.List(ctx, ownerID)
}

// Create memvalidasi lalu menyimpan transaksi baru.
//
// Budget sengaja TIDAK memblokir di sini: frontend memperingatkan sebelum
// menyimpan, dan user boleh memilih "Save anyway".
func (uc *Usecase) Create(ctx context.Context, ownerID string, p domain.Params) (domain.Transaction, error) {
	t, err := domain.New(uc.newID(), ownerID, p, uc.clock.Now())
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := uc.repo.Create(ctx, t); err != nil {
		return domain.Transaction{}, err
	}
	return t, nil
}

// Update mengubah sebagian field transaksi. Sama seperti Create, budget tidak
// memblokir di sini.
func (uc *Usecase) Update(ctx context.Context, ownerID, id string, patch domain.Patch) (domain.Transaction, error) {
	t, err := uc.repo.Get(ctx, ownerID, id)
	if err != nil {
		return domain.Transaction{}, wrap(err)
	}
	if err := t.Apply(patch); err != nil {
		return domain.Transaction{}, err
	}
	if err := uc.repo.Update(ctx, t); err != nil {
		return domain.Transaction{}, wrap(err)
	}
	return t, nil
}

// Delete menghapus satu transaksi.
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
