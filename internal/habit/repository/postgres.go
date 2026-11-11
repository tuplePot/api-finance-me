// Package repository = penyimpanan habit di PostgreSQL.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"financial-tracker/internal/habit/domain"
)

// Postgres menyimpan habit di tabel habits.
type Postgres struct{ db *pgxpool.Pool }

func NewPostgres(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }

const selectHabit = `SELECT id, owner_id, name, icon, cost, frequency, since, created_at, updated_at FROM habits`

func scanHabit(row pgx.CollectableRow) (domain.Habit, error) {
	var h domain.Habit
	var freq string
	err := row.Scan(&h.ID, &h.OwnerID, &h.Name, &h.Icon, &h.Cost, &freq, &h.Since, &h.CreatedAt, &h.UpdatedAt)
	h.Frequency = domain.Frequency(freq)
	return h, err
}

func (p *Postgres) List(ctx context.Context, ownerID string) ([]domain.Habit, error) {
	rows, err := p.db.Query(ctx, selectHabit+` WHERE owner_id = $1 ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanHabit)
}

func (p *Postgres) Get(ctx context.Context, ownerID, id string) (domain.Habit, error) {
	if uuid.Validate(id) != nil {
		return domain.Habit{}, domain.ErrNotFound
	}
	rows, err := p.db.Query(ctx, selectHabit+` WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return domain.Habit{}, err
	}
	h, err := pgx.CollectExactlyOneRow(rows, scanHabit)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Habit{}, domain.ErrNotFound
	}
	return h, err
}

func (p *Postgres) Create(ctx context.Context, h domain.Habit) error {
	_, err := p.db.Exec(ctx, `INSERT INTO habits
		(id, owner_id, name, icon, cost, frequency, since, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		h.ID, h.OwnerID, h.Name, h.Icon, h.Cost, string(h.Frequency), h.Since, h.CreatedAt, h.UpdatedAt)
	return err
}

func (p *Postgres) Update(ctx context.Context, h domain.Habit) error {
	tag, err := p.db.Exec(ctx, `UPDATE habits
		SET name = $3, icon = $4, cost = $5, frequency = $6, since = $7, updated_at = $8
		WHERE id = $1 AND owner_id = $2`,
		h.ID, h.OwnerID, h.Name, h.Icon, h.Cost, string(h.Frequency), h.Since, h.UpdatedAt)
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
	tag, err := p.db.Exec(ctx, `DELETE FROM habits WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
