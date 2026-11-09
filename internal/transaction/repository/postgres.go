// Package repository = penyimpanan transaksi di PostgreSQL.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"financial-tracker/internal/transaction/domain"
)

// Postgres menyimpan transaksi di tabel transactions.
type Postgres struct{ db *pgxpool.Pool }

func NewPostgres(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }

const selectTransaction = `SELECT id, owner_id, type, amount, category, date,
	COALESCE(note, ''), created_at FROM transactions`

func scanTransaction(row pgx.CollectableRow) (domain.Transaction, error) {
	var t domain.Transaction
	var typ string
	err := row.Scan(&t.ID, &t.OwnerID, &typ, &t.Amount, &t.Category, &t.Date, &t.Note, &t.CreatedAt)
	t.Type = domain.Type(typ)
	return t, err
}

func (p *Postgres) List(ctx context.Context, ownerID string) ([]domain.Transaction, error) {
	rows, err := p.db.Query(ctx, selectTransaction+` WHERE owner_id = $1
		ORDER BY date DESC, created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanTransaction)
}

func (p *Postgres) Get(ctx context.Context, ownerID, id string) (domain.Transaction, error) {
	if uuid.Validate(id) != nil {
		return domain.Transaction{}, domain.ErrNotFound
	}
	rows, err := p.db.Query(ctx, selectTransaction+` WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return domain.Transaction{}, err
	}
	t, err := pgx.CollectExactlyOneRow(rows, scanTransaction)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, domain.ErrNotFound
	}
	return t, err
}

func (p *Postgres) Update(ctx context.Context, t domain.Transaction) error {
	tag, err := p.db.Exec(ctx, `UPDATE transactions
		SET type = $3, amount = $4, category = $5, date = $6, note = NULLIF($7, '')
		WHERE id = $1 AND owner_id = $2`,
		t.ID, t.OwnerID, string(t.Type), t.Amount, t.Category, t.Date, t.Note)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
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
