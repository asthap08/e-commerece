package digital

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Create(ctx context.Context, e Entitlement) (Entitlement, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO digital_entitlements(user_id,order_id,variant_id,download_url) VALUES($1,$2,$3,$4) RETURNING id`, e.UserID, e.OrderID, e.VariantID, e.DownloadURL).Scan(&e.ID)
	return e, err
}
