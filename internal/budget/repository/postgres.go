// Package repository = penyimpanan budget di PostgreSQL.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"financial-tracker/internal/budget/domain"
	"financial-tracker/platform/postgres"
)

// Postgres menyimpan budget di tabel budgets.
type Postgres struct{ db *pgxpool.Pool }

func NewPostgres(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }

const selectBudget = `SELECT id, owner_id, category, period, limit_amount, created_at, updated_at FROM budgets`

func scanBudget(row pgx.CollectableRow) (domain.Budget, error) {
	var b domain.Budget
	var period string
	err := row.Scan(&b.ID, &b.OwnerID, &b.Category, &period, &b.Limit, &b.CreatedAt, &b.UpdatedAt)
	b.Period = domain.Period(period)
	return b, err
}

func (p *Postgres) List(ctx context.Context, ownerID string) ([]domain.Budget, error) {
	rows, err := p.db.Query(ctx, selectBudget+` WHERE owner_id = $1 ORDER BY category`, ownerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanBudget)
}

func (p *Postgres) Get(ctx context.Context, ownerID, id string) (domain.Budget, error) {
	if uuid.Validate(id) != nil {
		return domain.Budget{}, domain.ErrNotFound
	}
	rows, err := p.db.Query(ctx, selectBudget+` WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return domain.Budget{}, err
	}
	b, err := pgx.CollectExactlyOneRow(rows, scanBudget)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Budget{}, domain.ErrNotFound
	}
	return b, err
}

func (p *Postgres) Create(ctx context.Context, b domain.Budget) error {
	_, err := p.db.Exec(ctx, `INSERT INTO budgets
		(id, owner_id, category, period, limit_amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		b.ID, b.OwnerID, b.Category, string(b.Period), b.Limit, b.CreatedAt, b.UpdatedAt)
	// UNIQUE (owner_id, category) — juga aman kalau dua request datang bersamaan.
	if postgres.IsUniqueViolation(err) {
		return domain.DuplicateError{Category: b.Category}
	}
	return err
}

func (p *Postgres) Update(ctx context.Context, b domain.Budget) error {
	tag, err := p.db.Exec(ctx, `UPDATE budgets SET period = $3, limit_amount = $4, updated_at = $5
		WHERE id = $1 AND owner_id = $2`,
		b.ID, b.OwnerID, string(b.Period), b.Limit, b.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (p *Postgres) Delete(ctx context.Context, ownerID, id string) error {
	if uuid.Validate(id) != nil {
		return domain.ErrNotFound
	}
	tag, err := p.db.Exec(ctx, `DELETE FROM budgets WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
