// Package repository = penyimpanan transaksi di PostgreSQL.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"financial-tracker/internal/transaction/domain"
)

// Postgres menyimpan transaksi di tabel transactions.
type Postgres struct{ db *pgxpool.Pool }

func NewPostgres(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }

func (p *Postgres) List(ctx context.Context, ownerID string) ([]domain.Transaction, error) {
	rows, err := p.db.Query(ctx, `SELECT id, owner_id, type, amount, category, date,
		COALESCE(note, ''), created_at
		FROM transactions WHERE owner_id = $1
		ORDER BY date DESC, created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Transaction, error) {
		var t domain.Transaction
		var typ string
		err := row.Scan(&t.ID, &t.OwnerID, &typ, &t.Amount, &t.Category, &t.Date, &t.Note, &t.CreatedAt)
		t.Type = domain.Type(typ)
		return t, err
	})
}

func (p *Postgres) Create(ctx context.Context, t domain.Transaction) error {
	_, err := p.db.Exec(ctx, `INSERT INTO transactions
		(id, owner_id, type, amount, category, date, note, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8)`,
		t.ID, t.OwnerID, string(t.Type), t.Amount, t.Category, t.Date, t.Note, t.CreatedAt)
	return err
}

func (p *Postgres) Delete(ctx context.Context, ownerID, id string) error {
	// ID bukan UUID tidak mungkin ada — jangan kirim ke Postgres (akan error cast).
	if uuid.Validate(id) != nil {
		return domain.ErrNotFound
	}
	tag, err := p.db.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
