// Package repository = penyimpanan wishlist di PostgreSQL.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"financial-tracker/internal/wishlist/domain"
)

// Postgres menyimpan wishlist di tabel wishlist_items.
type Postgres struct{ db *pgxpool.Pool }

func NewPostgres(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }

func scanItem(row pgx.CollectableRow) (domain.Item, error) {
	var it domain.Item
	err := row.Scan(&it.ID, &it.OwnerID, &it.Name, &it.Price, &it.CreatedAt)
	return it, err
}

func (p *Postgres) List(ctx context.Context, ownerID string) ([]domain.Item, error) {
	rows, err := p.db.Query(ctx, `SELECT id, owner_id, name, price, created_at
		FROM wishlist_items WHERE owner_id = $1 ORDER BY price, created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanItem)
}

func (p *Postgres) Create(ctx context.Context, it domain.Item) error {
	_, err := p.db.Exec(ctx, `INSERT INTO wishlist_items (id, owner_id, name, price, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		it.ID, it.OwnerID, it.Name, it.Price, it.CreatedAt)
	return err
}

func (p *Postgres) Delete(ctx context.Context, ownerID, id string) error {
	if uuid.Validate(id) != nil {
		return domain.ErrNotFound
	}
	tag, err := p.db.Exec(ctx, `DELETE FROM wishlist_items WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
