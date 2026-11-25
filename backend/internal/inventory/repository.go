package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Get(ctx context.Context, variantID string) (Stock, error) {
	var s Stock
	err := r.pool.QueryRow(ctx, `SELECT v.id,COALESCE(i.quantity,0),COALESCE(i.reserved,0) FROM product_variants v LEFT JOIN inventory i ON i.variant_id=v.id WHERE v.id=$1`, variantID).Scan(&s.VariantID, &s.Quantity, &s.Reserved)
	return s, err
}

func (r *Repository) Adjust(ctx context.Context, variantID string, delta int, reason string) error {
	return r.AdjustAs(ctx, "", variantID, delta, reason)
}

func (r *Repository) AdjustAs(ctx context.Context, actorID, variantID string, delta int, reason string) error {
	if delta == 0 {
		return fmt.Errorf("inventory adjustment must not be zero")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `
		UPDATE inventory
		SET quantity = quantity + $2
		WHERE variant_id = $1 AND quantity + $2 >= reserved AND quantity + $2 >= 0
	`, variantID, delta)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		if delta < 0 {
			return fmt.Errorf("adjustment would reduce available stock below reserved inventory")
		}
		result, err = tx.Exec(ctx, `INSERT INTO inventory(variant_id,quantity,reserved) VALUES($1,$2,0) ON CONFLICT (variant_id) DO NOTHING`, variantID, delta)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("inventory changed concurrently; retry the adjustment")
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_id,action,entity,entity_id) VALUES(NULLIF($1,'')::uuid,$2,$3,$4)`,
		actorID, fmt.Sprintf("inventory adjustment: %s (%+d)", reason, delta), "inventory", variantID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Deduct(ctx context.Context, variantID string, qty int) error {
	if qty <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	res, err := r.pool.Exec(ctx, `UPDATE inventory SET quantity = quantity - $1 WHERE variant_id = $2 AND quantity - reserved >= $1`, qty, variantID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("insufficient inventory for variant %s", variantID)
	}
	return nil
}
