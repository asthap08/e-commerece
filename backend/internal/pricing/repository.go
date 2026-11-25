package pricing

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("coupon not found")
	ErrExists   = errors.New("a coupon with that code already exists")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const cols = `id,code,discount_type,value,active,min_order_minor,max_uses,used_count,expires_at`

func scan(row pgx.Row) (Coupon, error) {
	var c Coupon
	err := row.Scan(&c.ID, &c.Code, &c.DiscountType, &c.Value, &c.Active, &c.MinOrderMinor, &c.MaxUses, &c.UsedCount, &c.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

func (r *Repository) Create(ctx context.Context, c Coupon) (Coupon, error) {
	out, err := scan(r.pool.QueryRow(ctx, `INSERT INTO coupons(code,discount_type,value,active,min_order_minor,max_uses,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+cols,
		strings.ToUpper(c.Code), c.DiscountType, c.Value, c.Active, c.MinOrderMinor, c.MaxUses, c.ExpiresAt))
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return out, ErrExists
	}
	return out, err
}

func (r *Repository) GetByCode(ctx context.Context, code string) (Coupon, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM coupons WHERE upper(code)=upper($1)`, strings.TrimSpace(code)))
}

func (r *Repository) List(ctx context.Context) ([]Coupon, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM coupons ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Coupon, 0)
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) SetActive(ctx context.Context, id string, active bool) error {
	res, err := r.pool.Exec(ctx, `UPDATE coupons SET active=$2 WHERE id::text=$1`, id, active)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
