// Package repository = penyimpanan challenge di PostgreSQL.
//
// Tabel: challenges dan challenge_logs.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"financial-tracker/internal/challenge/domain"
	"financial-tracker/internal/challenge/usecase"
)

// Postgres = repository challenge.
type Postgres struct{ db *pgxpool.Pool }

func NewPostgres(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }

// querier = bagian yang sama dari *pgxpool.Pool dan pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

const selectChallenge = `SELECT id, owner_id, title, temptation_name, temptation_price,
	goal_name, goal_price, target_amount, deadline, saved_amount,
	status, completed_at, created_at FROM challenges`

func scanChallenge(row pgx.CollectableRow) (domain.Challenge, error) {
	var c domain.Challenge
	var status string
	err := row.Scan(&c.ID, &c.OwnerID, &c.Title, &c.Temptation.Name, &c.Temptation.Price,
		&c.Goal.Name, &c.Goal.Price, &c.TargetAmount, &c.Deadline, &c.SavedAmount,
		&status, &c.CompletedAt, &c.CreatedAt)
	c.Status = domain.Status(status)
	c.Logs = []domain.Log{}
	return c, err
}

func (p *Postgres) SettleExpired(ctx context.Context, ownerID string, today time.Time) error {
	_, err := p.db.Exec(ctx, `UPDATE challenges SET status = 'failed'
		WHERE owner_id = $1 AND status = 'active' AND deadline < $2`, ownerID, today)
	return err
}

func (p *Postgres) List(ctx context.Context, ownerID string) ([]domain.Challenge, error) {
	rows, err := p.db.Query(ctx, selectChallenge+` WHERE owner_id = $1 ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, scanChallenge)
	if err != nil || len(list) == 0 {
		return list, err
	}
	return list, attachLogs(ctx, p.db, list)
}

// attachLogs mengisi Logs semua challenge dengan satu query (bukan N+1).
func attachLogs(ctx context.Context, q querier, list []domain.Challenge) error {
	ids := make([]string, len(list))
	index := make(map[string]int, len(list))
	for i, c := range list {
		ids[i] = c.ID
		index[c.ID] = i
	}
	rows, err := q.Query(ctx, `SELECT challenge_id, id, kind, amount, date, created_at
		FROM challenge_logs WHERE challenge_id = ANY($1)
		ORDER BY created_at DESC`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var challengeID, kind string
		var l domain.Log
		if err := rows.Scan(&challengeID, &l.ID, &kind, &l.Amount, &l.Date, &l.CreatedAt); err != nil {
			return err
		}
		l.Kind = domain.LogKind(kind)
		i := index[challengeID]
		list[i].Logs = append(list[i].Logs, l)
	}
	return rows.Err()
}

func (p *Postgres) Create(ctx context.Context, c domain.Challenge) error {
	_, err := p.db.Exec(ctx, `INSERT INTO challenges
		(id, owner_id, title, temptation_name, temptation_price, goal_name, goal_price,
		 target_amount, deadline, saved_amount, status, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		c.ID, c.OwnerID, c.Title, c.Temptation.Name, c.Temptation.Price, c.Goal.Name, c.Goal.Price,
		c.TargetAmount, c.Deadline, c.SavedAmount, string(c.Status), c.CompletedAt, c.CreatedAt)
	return err
}

func (p *Postgres) Delete(ctx context.Context, ownerID, id string) error {
	if uuid.Validate(id) != nil {
		return domain.ErrNotFound
	}
	// Logs ikut terhapus (CASCADE).
	tag, err := p.db.Exec(ctx, `DELETE FROM challenges WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (p *Postgres) Mutate(ctx context.Context, ownerID, id string,
	fn func(c *domain.Challenge) (usecase.Mutation, error),
) (domain.Challenge, error) {
	if uuid.Validate(id) != nil {
		return domain.Challenge{}, domain.ErrNotFound
	}
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return domain.Challenge{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op setelah Commit

	rows, err := tx.Query(ctx, selectChallenge+` WHERE id = $1 AND owner_id = $2 FOR UPDATE`, id, ownerID)
	if err != nil {
		return domain.Challenge{}, err
	}
	c, err := pgx.CollectExactlyOneRow(rows, scanChallenge)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Challenge{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Challenge{}, err
	}
	list := []domain.Challenge{c}
	if err := attachLogs(ctx, tx, list); err != nil {
		return domain.Challenge{}, err
	}
	c = list[0]

	m, err := fn(&c)
	if err != nil {
		return domain.Challenge{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE challenges SET saved_amount = $2, status = $3, completed_at = $4
		WHERE id = $1`, c.ID, c.SavedAmount, string(c.Status), c.CompletedAt); err != nil {
		return domain.Challenge{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO challenge_logs (id, challenge_id, kind, amount, date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		m.Log.ID, c.ID, string(m.Log.Kind), m.Log.Amount, m.Log.Date, m.Log.CreatedAt); err != nil {
		return domain.Challenge{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Challenge{}, err
	}
	return c, nil
}
