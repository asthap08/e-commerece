package shipping

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Create(ctx context.Context, s Shipment) (Shipment, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO shipments(order_id,carrier,tracking_number,status) VALUES($1,$2,$3,$4) RETURNING id`, s.OrderID, s.Carrier, s.TrackingNumber, s.Status).Scan(&s.ID)
	return s, err
}
