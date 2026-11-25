package cart

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) GetOrCreateCart(ctx context.Context, userID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO carts(user_id) VALUES($1)
		ON CONFLICT (user_id) WHERE user_id IS NOT NULL
		DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id
	`, userID).Scan(&id)
	return id, err
}

func (r *Repository) GetOrCreateCartBySession(ctx context.Context, sessionID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO carts(session_id) VALUES($1)
		ON CONFLICT (session_id) WHERE session_id IS NOT NULL
		DO UPDATE SET session_id = EXCLUDED.session_id
		RETURNING id
	`, sessionID).Scan(&id)
	return id, err
}

func (r *Repository) MergeSessionCart(ctx context.Context, sessionID, userID string) error {
	// Only anonymous session carts (user_id IS NULL) can be merged, so one user
	// cannot absorb another user's cart by guessing its id.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sessionCartID string
	err = tx.QueryRow(ctx, `SELECT id FROM carts WHERE session_id=$1 AND user_id IS NULL FOR UPDATE`, sessionID).Scan(&sessionCartID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}

	var userCartID string
	err = tx.QueryRow(ctx, `
		INSERT INTO carts(user_id) VALUES($1)
		ON CONFLICT (user_id) WHERE user_id IS NOT NULL
		DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id
	`, userID).Scan(&userCartID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO cart_items(cart_id, variant_id, quantity)
		SELECT $1, variant_id, quantity
		FROM cart_items
		WHERE cart_id = $2
		ON CONFLICT (cart_id, variant_id)
		DO UPDATE SET quantity = LEAST(99, cart_items.quantity + EXCLUDED.quantity)
	`, userCartID, sessionCartID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM carts WHERE id=$1`, sessionCartID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Availability reports whether a variant can be bought (its product is active)
// and how many units are free, plus what the cart already holds.
func (r *Repository) Availability(ctx context.Context, cartID, variantID string) (live bool, available, inCart int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT p.status='active', GREATEST(COALESCE(i.quantity-i.reserved,0),0),
		       COALESCE((SELECT quantity FROM cart_items WHERE cart_id=$1 AND variant_id=v.id),0)
		FROM product_variants v JOIN products p ON p.id=v.product_id
		LEFT JOIN inventory i ON i.variant_id=v.id WHERE v.id=$2`, cartID, variantID).Scan(&live, &available, &inCart)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, 0, nil
	}
	return
}

func (r *Repository) AddItem(ctx context.Context, cartID, variantID string, qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO cart_items(cart_id,variant_id,quantity) VALUES($1,$2,$3)
		ON CONFLICT (cart_id, variant_id)
		DO UPDATE SET quantity = LEAST(99, cart_items.quantity + EXCLUDED.quantity)
	`, cartID, variantID, qty)
	return err
}

func (r *Repository) ListItems(ctx context.Context, cartID string) ([]CartItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ci.id,ci.cart_id,ci.variant_id,ci.quantity,p.name,v.price_minor,
		       COALESCE(image.url,''),p.id,v.sku,GREATEST(COALESCE(inv.quantity-inv.reserved,0),0)
		FROM cart_items ci
		JOIN product_variants v ON v.id=ci.variant_id
		JOIN products p ON p.id=v.product_id
		LEFT JOIN inventory inv ON inv.variant_id=v.id
		LEFT JOIN LATERAL (
		    SELECT url FROM product_images
		    WHERE product_id=p.id
		    ORDER BY position,id
		    LIMIT 1
		) image ON true
		WHERE ci.cart_id=$1
		ORDER BY ci.id
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CartItem, 0)
	for rows.Next() {
		var ci CartItem
		if err := rows.Scan(&ci.ID, &ci.CartID, &ci.VariantID, &ci.Quantity, &ci.Name, &ci.PriceMinor, &ci.ImageURL, &ci.ProductID, &ci.SKU, &ci.Available); err != nil {
			return nil, err
		}
		out = append(out, ci)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateItem(ctx context.Context, cartID, variantID string, qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}
	result, err := r.pool.Exec(ctx, `UPDATE cart_items SET quantity=$1 WHERE cart_id=$2 AND variant_id=$3`, qty, cartID, variantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("cart item not found")
	}
	return nil
}

func (r *Repository) RemoveItem(ctx context.Context, cartID, variantID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id=$1 AND variant_id=$2`, cartID, variantID)
	return err
}

func (r *Repository) ClearCart(ctx context.Context, cartID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id=$1`, cartID)
	return err
}
