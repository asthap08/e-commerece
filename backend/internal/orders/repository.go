package orders

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
	ErrOrderNotFound           = errors.New("order not found")
	ErrForbidden               = errors.New("you cannot change this order")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const orderCols = `o.id,COALESCE(o.user_id::text,''),o.status,o.total_minor,o.currency,o.created_at,
	COALESCE(o.payment_method,''),COALESCE(o.shipping_method,''),COALESCE(o.shipping_cost_minor,0),o.discount_minor,o.coupon_code,
	COALESCE(o.shipping_address::text,'null'),COALESCE(o.customer_name,''),COALESCE(o.customer_email,''),COALESCE(o.customer_phone,''),
	COALESCE(o.tracking_number,''),COALESCE(o.tracking_carrier,''),
	COALESCE((SELECT status FROM payments WHERE order_id=o.id ORDER BY created_at DESC LIMIT 1),'')`

func scanOrder(row pgx.Row, o *Order) error {
	var addr string
	err := row.Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.Currency, &o.Created, &o.PaymentMethod, &o.ShippingMethod,
		&o.ShippingCost, &o.Discount, &o.CouponCode, &addr, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
		&o.TrackingNumber, &o.TrackingCarrier, &o.PaymentStatus)
	o.ShippingAddress = json.RawMessage(addr)
	return err
}

const itemsJSON = `COALESCE((SELECT json_agg(json_build_object('id',oi.id,'order_id',oi.order_id,'variant_id',oi.variant_id,'sku',oi.sku,'name',oi.name,
	'price_minor',oi.price_minor,'quantity',oi.quantity,'image',COALESCE((SELECT url FROM product_images pi JOIN product_variants pv ON pv.product_id=pi.product_id WHERE pv.id=oi.variant_id ORDER BY pi.position LIMIT 1),'')) ORDER BY oi.id)
	FROM order_items oi WHERE oi.order_id=o.id),'[]'::json)`

func (r *Repository) GetByID(ctx context.Context, id string) (Order, error) {
	var o Order
	var items []byte
	var addr0 string
	err := r.pool.QueryRow(ctx, `SELECT `+orderCols+`, `+itemsJSON+` FROM orders o WHERE o.id::text=$1`, id).Scan(
		&o.ID, &o.UserID, &o.Status, &o.Total, &o.Currency, &o.Created, &o.PaymentMethod, &o.ShippingMethod,
		&o.ShippingCost, &o.Discount, &o.CouponCode, &addr0, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
		&o.TrackingNumber, &o.TrackingCarrier, &o.PaymentStatus, &items)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrOrderNotFound
	}
	if err != nil {
		return o, err
	}
	o.ShippingAddress = json.RawMessage(addr0)
	_ = json.Unmarshal(items, &o.Items)
	return o, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]Order, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+orderCols+`, `+itemsJSON+` FROM orders o WHERE o.user_id=$1 ORDER BY o.created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Order, 0)
	for rows.Next() {
		var o Order
		var items []byte
		var addr string
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.Currency, &o.Created, &o.PaymentMethod, &o.ShippingMethod,
			&o.ShippingCost, &o.Discount, &o.CouponCode, &addr, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
			&o.TrackingNumber, &o.TrackingCarrier, &o.PaymentStatus, &items); err != nil {
			return nil, err
		}
		o.ShippingAddress = json.RawMessage(addr)
		_ = json.Unmarshal(items, &o.Items)
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListAll is the admin listing with optional status / free-text filter.
func (r *Repository) ListAll(ctx context.Context, f ListFilter) ([]Order, int, error) {
	where, args := []string{"true"}, []any{}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, "o.status=$"+itoa(len(args)))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)+"%")
		n := itoa(len(args))
		where = append(where, "(o.id::text ILIKE $"+n+" OR o.customer_email ILIKE $"+n+" OR o.customer_name ILIKE $"+n+" OR o.tracking_number ILIKE $"+n+")")
	}
	cond := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM orders o WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx, `SELECT `+orderCols+`, (SELECT count(*) FROM order_items WHERE order_id=o.id) FROM orders o WHERE `+cond+
		` ORDER BY o.created_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]Order, 0)
	for rows.Next() {
		var o Order
		var addr string
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.Currency, &o.Created, &o.PaymentMethod, &o.ShippingMethod,
			&o.ShippingCost, &o.Discount, &o.CouponCode, &addr, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
			&o.TrackingNumber, &o.TrackingCarrier, &o.PaymentStatus, &o.ItemCount); err != nil {
			return nil, 0, err
		}
		o.ShippingAddress = json.RawMessage(addr)
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func itoa(n int) string { return strconv.Itoa(n) }

// Track returns the guest-safe view when id and e-mail both match.
func (r *Repository) Track(ctx context.Context, id, email string) (TrackedOrder, error) {
	o, err := r.GetByID(ctx, id)
	if err != nil || !strings.EqualFold(strings.TrimSpace(o.CustomerEmail), strings.TrimSpace(email)) || o.CustomerEmail == "" {
		return TrackedOrder{}, ErrOrderNotFound
	}
	return TrackedOrder{ID: o.ID, Status: o.Status, Total: o.Total, Created: o.Created, PaymentMethod: o.PaymentMethod,
		ShippingMethod: o.ShippingMethod, TrackingNumber: o.TrackingNumber, TrackingCarrier: o.TrackingCarrier, Items: o.Items}, nil
}

// SetStatus moves an order to a new status inside one transaction: the row is
// locked, the transition is validated, and stock is returned when an order is
// cancelled (checkout deducts stock up front, so cancelling must give it back).
// actorID restricts customers to their own orders; admins pass actorID="".
func (r *Repository) SetStatus(ctx context.Context, id, to, actorID string, allowed map[string][]string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, owner string
	err = tx.QueryRow(ctx, `SELECT status, COALESCE(user_id::text,'') FROM orders WHERE id::text=$1 FOR UPDATE`, id).Scan(&status, &owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return err
	}
	if actorID != "" && owner != actorID {
		return ErrOrderNotFound // do not reveal that the order exists
	}
	ok := false
	for _, s := range allowed[status] {
		if s == to {
			ok = true
		}
	}
	if !ok {
		return ErrInvalidStatusTransition
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status=$2, updated_at=now() WHERE id::text=$1`, id, to); err != nil {
		return err
	}
	if to == "cancelled" {
		if _, err := tx.Exec(ctx, `UPDATE inventory i SET quantity=i.quantity+oi.quantity FROM order_items oi WHERE oi.order_id::text=$1 AND i.variant_id=oi.variant_id`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='cancelled' WHERE order_id::text=$1 AND status IN ('pending','failed')`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE coupons SET used_count=GREATEST(used_count-1,0) WHERE upper(code)=(SELECT upper(coupon_code) FROM orders WHERE id::text=$1 AND coupon_code<>'')`, id); err != nil {
			return err
		}
	}
	if to == "delivered" {
		// cash-on-delivery is settled on delivery
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='completed' WHERE order_id::text=$1 AND provider='manual' AND status='pending'`, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpdateTracking(ctx context.Context, orderID, trackingNumber, carrier string) error {
	res, err := r.pool.Exec(ctx, `UPDATE orders SET tracking_number=$2, tracking_carrier=$3, updated_at=now() WHERE id::text=$1`, orderID, trackingNumber, carrier)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}
