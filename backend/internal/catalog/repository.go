package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("slug or sku already exists")
	ErrNotLive   = errors.New("active products require at least one variant")
	ErrBadParent = errors.New("product does not exist")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func mapErr(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrBadParent
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

const productSelect = `
SELECT p.id,p.name,p.slug,COALESCE(p.description,''),p.status,p.category,p.created_at,p.updated_at,
       COALESCE(image.url,''), COALESCE(images.urls,'[]'::json),
       COALESCE(rv.avg,0), COALESCE(rv.cnt,0),
       COALESCE(variants.items,'[]'::json)
FROM products p
LEFT JOIN LATERAL (SELECT url FROM product_images WHERE product_id=p.id ORDER BY position,id LIMIT 1) image ON true
LEFT JOIN LATERAL (SELECT json_agg(url ORDER BY position,id) AS urls FROM product_images WHERE product_id=p.id) images ON true
LEFT JOIN LATERAL (SELECT round(avg(rating)::numeric,1)::float8 AS avg, count(*)::int AS cnt FROM reviews WHERE product_id=p.id) rv ON true
LEFT JOIN LATERAL (
  SELECT json_agg(json_build_object(
    'id',v.id,'product_id',v.product_id,'sku',v.sku,'name',v.name,'price_minor',v.price_minor,
    'compare_at_minor',v.compare_at_minor,'currency',v.currency,
    'in_stock',COALESCE(i.quantity-i.reserved,0)>0,'available',GREATEST(COALESCE(i.quantity-i.reserved,0),0)
  ) ORDER BY v.price_minor, v.created_at) AS items
  FROM product_variants v LEFT JOIN inventory i ON i.variant_id=v.id WHERE v.product_id=p.id
) variants ON true `

func scanProduct(row pgx.Row) (Product, error) {
	var p Product
	var imgs, vars []byte
	if err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.Category, &p.CreatedAt, &p.UpdatedAt,
		&p.ImageURL, &imgs, &p.Rating, &p.ReviewCount, &vars); err != nil {
		return p, err
	}
	_ = json.Unmarshal(imgs, &p.Images)
	if err := json.Unmarshal(vars, &p.Variants); err != nil {
		return p, err
	}
	return p, nil
}

// escapeLike makes user text match literally in ILIKE (%, _ and \ are wildcards).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// List returns active products plus the total number of matches.
func (r *Repository) List(ctx context.Context, f ListFilter) ([]Product, int, error) {
	where := []string{"p.status='active'"}
	var args []any
	arg := func(v any) string { args = append(args, v); return "$" + strconv.Itoa(len(args)) }
	if f.Category != "" && f.Category != "all" {
		where = append(where, "p.category="+arg(f.Category))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		a := arg("%" + escapeLike(q) + "%")
		where = append(where, "(p.name ILIKE "+a+" OR p.description ILIKE "+a+" OR p.category ILIKE "+a+")")
	}
	priceExpr := "(SELECT min(price_minor) FROM product_variants WHERE product_id=p.id)"
	if f.MinPrice > 0 {
		where = append(where, priceExpr+" >= "+arg(f.MinPrice))
	}
	if f.MaxPrice > 0 {
		where = append(where, priceExpr+" <= "+arg(f.MaxPrice))
	}
	cond := " WHERE " + strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM products p"+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := "p.created_at DESC"
	switch f.Sort {
	case "price_asc":
		order = priceExpr + " ASC NULLS LAST, p.created_at DESC"
	case "price_desc":
		order = priceExpr + " DESC NULLS LAST, p.created_at DESC"
	case "rating":
		order = "COALESCE(rv.avg,0) DESC, COALESCE(rv.cnt,0) DESC"
	}
	q := productSelect + cond + " ORDER BY " + order + " LIMIT " + arg(f.Limit) + " OFFSET " + arg(f.Offset)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]Product, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// Get returns one product with variants and images. publicOnly hides non-active products.
func (r *Repository) Get(ctx context.Context, idOrSlug string, publicOnly bool) (Product, error) {
	cond := " WHERE (p.slug=$1 OR p.id::text=$1)"
	if publicOnly {
		cond += " AND p.status='active'"
	}
	p, err := scanProduct(r.pool.QueryRow(ctx, productSelect+cond, idOrSlug))
	return p, mapErr(err)
}

func (r *Repository) Categories(ctx context.Context) ([]Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT category, count(*)::int FROM products WHERE status='active' GROUP BY category ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.Name, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) setImage(ctx context.Context, tx pgx.Tx, productID, url string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM product_images WHERE product_id=$1`, productID); err != nil {
		return err
	}
	if url == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO product_images(product_id,url,position) VALUES($1,$2,0)`, productID, url)
	return err
}

func (r *Repository) Create(ctx context.Context, p Product) (Product, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `INSERT INTO products(name,slug,description,status,category) VALUES($1,$2,$3,$4,$5) RETURNING id,created_at,updated_at`,
		p.Name, p.Slug, p.Description, p.Status, p.Category).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return p, mapErr(err)
	}
	if err := r.setImage(ctx, tx, p.ID, p.ImageURL); err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}

type Update struct {
	Name, Description, Category *string
	ImageURL                    *string
}

func (r *Repository) UpdateProduct(ctx context.Context, id string, u Update) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE products SET name=COALESCE($2,name), description=COALESCE($3,description), category=COALESCE($4,category), updated_at=now() WHERE id=$1`,
		id, u.Name, u.Description, u.Category)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if u.ImageURL != nil {
		if err := r.setImage(ctx, tx, id, *u.ImageURL); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	res, err := r.pool.Exec(ctx, `UPDATE products SET status=$1, updated_at=now() WHERE id=$2 AND ($1 <> 'active' OR EXISTS (SELECT 1 FROM product_variants WHERE product_id=products.id))`, status, id)
	if err != nil {
		return mapErr(err)
	}
	if res.RowsAffected() == 0 {
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1)`, id).Scan(&exists)
		if exists {
			return ErrNotLive
		}
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListVariants(ctx context.Context, productID string) ([]Variant, error) {
	p, err := r.Get(ctx, productID, true)
	if err != nil {
		return nil, err
	}
	return p.Variants, nil
}

// CreateVariant inserts a variant and its (zero) inventory row in one step so
// stock can be tracked immediately.
func (r *Repository) CreateVariant(ctx context.Context, v Variant) (Variant, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return v, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `INSERT INTO product_variants(product_id,sku,name,price_minor,compare_at_minor,currency) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		v.ProductID, v.SKU, v.Name, v.PriceMinor, v.CompareAtMinor, v.Currency).Scan(&v.ID); err != nil {
		return v, mapErr(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO inventory(variant_id,quantity,reserved) VALUES($1,0,0) ON CONFLICT DO NOTHING`, v.ID); err != nil {
		return v, err
	}
	return v, tx.Commit(ctx)
}

func (r *Repository) UpdateVariant(ctx context.Context, id string, name *string, price, compareAt *int64, clearCompare bool) error {
	res, err := r.pool.Exec(ctx, `UPDATE product_variants SET name=COALESCE($2,name), price_minor=COALESCE($3,price_minor),
		compare_at_minor = CASE WHEN $5 THEN NULL ELSE COALESCE($4,compare_at_minor) END WHERE id=$1`, id, name, price, compareAt, clearCompare)
	if err != nil {
		return mapErr(err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
