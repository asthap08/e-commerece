package addresses

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Create stores an address. The first address a user saves becomes their
// default; asking for a new default demotes the old one (one default per user).
func (r *Repository) Create(ctx context.Context, a Address) (Address, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return a, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM addresses WHERE user_id=$1`, a.UserID).Scan(&count); err != nil {
		return a, err
	}
	if count == 0 {
		a.IsDefault = true
	}
	if a.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE addresses SET is_default=false WHERE user_id=$1`, a.UserID); err != nil {
			return a, err
		}
	}
	err = tx.QueryRow(ctx, `INSERT INTO addresses(user_id,label,line1,line2,city,state,pincode,country,is_default) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,created_at`, a.UserID, a.Label, a.Line1, a.Line2, a.City, a.State, a.Pincode, a.Country, a.IsDefault).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		return a, err
	}
	return a, tx.Commit(ctx)
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]Address, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,user_id,label,line1,COALESCE(line2,''),city,state,pincode,country,is_default,created_at FROM addresses WHERE user_id=$1 ORDER BY is_default DESC, created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Address, 0)
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.UserID, &a.Label, &a.Line1, &a.Line2, &a.City, &a.State, &a.Pincode, &a.Country, &a.IsDefault, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.pool.Exec(ctx, `DELETE FROM addresses WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	// keep one default if the default was removed
	_, err = r.pool.Exec(ctx, `UPDATE addresses SET is_default=true WHERE id=(SELECT id FROM addresses WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1) AND NOT EXISTS (SELECT 1 FROM addresses WHERE user_id=$1 AND is_default)`, userID)
	return err
}

func (r *Repository) SetDefault(ctx context.Context, id, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var owned bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM addresses WHERE id=$1 AND user_id=$2)`, id, userID).Scan(&owned); err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE addresses SET is_default=false WHERE user_id=$1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE addresses SET is_default=true WHERE id=$1 AND user_id=$2`, id, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
