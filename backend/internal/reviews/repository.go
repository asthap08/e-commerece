package reviews

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDuplicate       = errors.New("you have already reviewed this product")
	ErrProductNotFound = errors.New("product not found")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Create(ctx context.Context, rev Review) (Review, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO reviews(product_id,user_id,rating,title,body)
		SELECT p.id,$2,$3,$4,$5 FROM products p WHERE p.id::text=$1 AND p.status='active' RETURNING id,product_id,created_at`,
		rev.ProductID, rev.UserID, rev.Rating, rev.Title, rev.Body).Scan(&rev.ID, &rev.ProductID, &rev.CreatedAt)
	var pg *pgconn.PgError
	switch {
	case errors.As(err, &pg) && pg.Code == "23505":
		return rev, ErrDuplicate
	case errors.As(err, &pg) && pg.Code == "22P02":
		return rev, ErrProductNotFound
	case errors.Is(err, pgx.ErrNoRows):
		return rev, ErrProductNotFound
	}
	return rev, err
}

// ListByProduct shows a masked author name (never the e-mail or user id) and
// flags reviewers who actually bought the product.
func (r *Repository) ListByProduct(ctx context.Context, productID string, limit, offset int) ([]Review, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rv.id,rv.product_id,rv.rating,COALESCE(rv.title,''),COALESCE(rv.body,''),rv.created_at,
		       COALESCE(NULLIF(split_part(u.full_name,' ',1),''), 'Customer'),
		       EXISTS(SELECT 1 FROM orders o JOIN order_items oi ON oi.order_id=o.id JOIN product_variants v ON v.id=oi.variant_id
		              WHERE o.user_id=rv.user_id AND v.product_id=rv.product_id AND o.status IN ('paid','processing','shipped','delivered'))
		FROM reviews rv JOIN users u ON u.id=rv.user_id WHERE rv.product_id::text=$1 ORDER BY rv.created_at DESC LIMIT $2 OFFSET $3`, productID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Review, 0)
	for rows.Next() {
		var rev Review
		if err := rows.Scan(&rev.ID, &rev.ProductID, &rev.Rating, &rev.Title, &rev.Body, &rev.CreatedAt, &rev.AuthorName, &rev.Verified); err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}

func (r *Repository) GetAverageRating(ctx context.Context, productID string) (float64, int, error) {
	var avg float64
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(round(AVG(rating)::numeric,1),0)::float8, COUNT(*) FROM reviews WHERE product_id::text=$1`, productID).Scan(&avg, &count)
	return avg, count, err
}
