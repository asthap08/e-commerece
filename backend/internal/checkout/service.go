package checkout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/asthap08/ecommerce/internal/pricing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidRequest      = errors.New("invalid checkout request")
	ErrCartEmpty           = errors.New("cart is empty")
	ErrInsufficientStock   = errors.New("insufficient inventory")
	ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")
	ErrCoupon              = errors.New("coupon cannot be applied")
)

// ShippingCosts defines delivery pricing in minor currency units (INR paise).
var pincodeRe = regexp.MustCompile(`^[1-9][0-9]{5}$`)

var ShippingCosts = map[string]int64{
	"standard": 0,
	"express":  14900,
	"same_day": 29900,
}

type Service struct {
	pool          *pgxpool.Pool
	stripeEnabled bool
}

func NewService(pool *pgxpool.Pool, stripeEnabled ...bool) *Service {
	service := &Service{pool: pool}
	if len(stripeEnabled) > 0 {
		service.stripeEnabled = stripeEnabled[0]
	}
	return service
}

func (s *Service) Checkout(ctx context.Context, req Request) (Response, error) {
	if err := validateRequest(&req); err != nil {
		return Response{}, err
	}
	if req.PaymentProvider == "stripe" && !s.stripeEnabled {
		return Response{}, fmt.Errorf("%w: card payments are not configured", ErrInvalidRequest)
	}
	requestHash, err := hashRequest(req)
	if err != nil {
		return Response{}, fmt.Errorf("hash checkout request: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("begin checkout: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted, err := tx.Exec(ctx, `
		INSERT INTO checkout_idempotency(user_id, idempotency_key, request_hash)
		VALUES($1, $2, $3)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
	`, req.UserID, req.IdempotencyKey, requestHash)
	if err != nil {
		return Response{}, fmt.Errorf("reserve checkout idempotency key: %w", err)
	}
	if inserted.RowsAffected() == 0 {
		var existingHash string
		var responseJSON []byte
		err := tx.QueryRow(ctx, `
			SELECT request_hash, response
			FROM checkout_idempotency
			WHERE user_id=$1 AND idempotency_key=$2
		`, req.UserID, req.IdempotencyKey).Scan(&existingHash, &responseJSON)
		if err != nil {
			return Response{}, fmt.Errorf("load previous checkout result: %w", err)
		}
		if existingHash != requestHash {
			return Response{}, ErrIdempotencyConflict
		}
		if len(responseJSON) == 0 {
			return Response{}, errors.New("previous checkout result is incomplete")
		}
		var response Response
		if err := json.Unmarshal(responseJSON, &response); err != nil {
			return Response{}, fmt.Errorf("decode previous checkout result: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return Response{}, fmt.Errorf("finish idempotent checkout lookup: %w", err)
		}
		return response, nil
	}

	var cartID string
	err = tx.QueryRow(ctx, `SELECT id FROM carts WHERE user_id=$1 FOR UPDATE`, req.UserID).Scan(&cartID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{}, ErrCartEmpty
	}
	if err != nil {
		return Response{}, fmt.Errorf("lock customer cart: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT v.id, v.sku, v.price_minor, v.currency, p.name, p.status, ci.quantity
		FROM cart_items ci
		JOIN product_variants v ON v.id=ci.variant_id
		JOIN products p ON p.id=v.product_id
		WHERE ci.cart_id=$1
		ORDER BY v.id
		FOR UPDATE OF ci
	`, cartID)
	if err != nil {
		return Response{}, fmt.Errorf("load checkout cart: %w", err)
	}
	type orderLine struct {
		variantID string
		sku       string
		name      string
		price     int64
		quantity  int
	}
	lines := make([]orderLine, 0)
	for rows.Next() {
		var line orderLine
		var currency string
		var productStatus string
		if err := rows.Scan(&line.variantID, &line.sku, &line.price, &currency, &line.name, &productStatus, &line.quantity); err != nil {
			rows.Close()
			return Response{}, fmt.Errorf("read checkout cart: %w", err)
		}
		if productStatus != "active" {
			rows.Close()
			return Response{}, fmt.Errorf("%w: cart contains an unavailable product", ErrInvalidRequest)
		}
		if line.quantity <= 0 || line.price <= 0 || currency != "INR" {
			rows.Close()
			return Response{}, fmt.Errorf("%w: cart contains an invalid quantity, price, or currency", ErrInvalidRequest)
		}
		if int64(line.quantity) > math.MaxInt64/line.price {
			rows.Close()
			return Response{}, fmt.Errorf("%w: checkout total is too large", ErrInvalidRequest)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Response{}, fmt.Errorf("iterate checkout cart: %w", err)
	}
	rows.Close()
	if len(lines) == 0 {
		return Response{}, ErrCartEmpty
	}

	var subtotal int64
	for _, line := range lines {
		lineTotal := line.price * int64(line.quantity)
		if subtotal > math.MaxInt64-lineTotal {
			return Response{}, fmt.Errorf("%w: checkout total is too large", ErrInvalidRequest)
		}
		subtotal += lineTotal
		result, err := tx.Exec(ctx, `
			UPDATE inventory
			SET quantity=quantity-$1
			WHERE variant_id=$2 AND quantity-reserved >= $1
		`, line.quantity, line.variantID)
		if err != nil {
			return Response{}, fmt.Errorf("deduct inventory: %w", err)
		}
		if result.RowsAffected() != 1 {
			return Response{}, fmt.Errorf("%w for variant %s", ErrInsufficientStock, line.variantID)
		}
	}
	var discount int64
	if req.CouponCode != "" {
		var c pricing.Coupon
		err := tx.QueryRow(ctx, `SELECT id,code,discount_type,value,active,min_order_minor,max_uses,used_count,expires_at FROM coupons WHERE upper(code)=upper($1) FOR UPDATE`, req.CouponCode).
			Scan(&c.ID, &c.Code, &c.DiscountType, &c.Value, &c.Active, &c.MinOrderMinor, &c.MaxUses, &c.UsedCount, &c.ExpiresAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return Response{}, fmt.Errorf("%w: that coupon code is not valid", ErrCoupon)
		}
		if err != nil {
			return Response{}, fmt.Errorf("load coupon: %w", err)
		}
		if why := pricing.Usable(c, subtotal, time.Now()); why != "" {
			return Response{}, fmt.Errorf("%w: %s", ErrCoupon, why)
		}
		discount = pricing.Discount(c, subtotal)
		req.CouponCode = c.Code
		if _, err := tx.Exec(ctx, `UPDATE coupons SET used_count=used_count+1 WHERE id=$1`, c.ID); err != nil {
			return Response{}, fmt.Errorf("count coupon use: %w", err)
		}
	}
	shippingCost := ShippingCosts[req.ShippingMethod]
	if subtotal > math.MaxInt64-shippingCost {
		return Response{}, fmt.Errorf("%w: checkout total is too large", ErrInvalidRequest)
	}
	total := subtotal - discount + shippingCost

	var orderID string
	err = tx.QueryRow(ctx, `
		INSERT INTO orders(
			user_id,status,total_minor,currency,payment_method,shipping_method,
			shipping_cost_minor,shipping_address,customer_name,customer_email,customer_phone,discount_minor,coupon_code
		) VALUES($1,'pending',$2,'INR',$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11)
		RETURNING id
	`, req.UserID, total, req.PaymentMethod, req.ShippingMethod, shippingCost,
		req.ShippingAddress, strings.TrimSpace(req.CustomerName), strings.ToLower(strings.TrimSpace(req.CustomerEmail)),
		strings.TrimSpace(req.CustomerPhone), discount, req.CouponCode).Scan(&orderID)
	if err != nil {
		return Response{}, fmt.Errorf("create order: %w", err)
	}
	for _, line := range lines {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items(order_id,variant_id,sku,name,price_minor,quantity)
			VALUES($1,$2,$3,$4,$5,$6)
		`, orderID, line.variantID, line.sku, line.name, line.price, line.quantity); err != nil {
			return Response{}, fmt.Errorf("create order item: %w", err)
		}
	}
	var paymentID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO payments(order_id,provider,status,amount_minor,method)
		VALUES($1,$2,'pending',$3,$4)
		RETURNING id
	`, orderID, req.PaymentProvider, total, req.PaymentMethod).Scan(&paymentID); err != nil {
		return Response{}, fmt.Errorf("create payment record: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id=$1`, cartID); err != nil {
		return Response{}, fmt.Errorf("clear checked-out cart: %w", err)
	}

	response := Response{
		OrderID:       orderID,
		Status:        "pending",
		TotalMinor:    total,
		SubtotalMinor: subtotal,
		DiscountMinor: discount,
		ShippingMinor: shippingCost,
		Currency:      "INR",
		PaymentID:     paymentID,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return Response{}, fmt.Errorf("encode checkout response: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE checkout_idempotency
		SET response=$3::jsonb
		WHERE user_id=$1 AND idempotency_key=$2
	`, req.UserID, req.IdempotencyKey, responseJSON); err != nil {
		return Response{}, fmt.Errorf("save checkout idempotency result: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Response{}, fmt.Errorf("commit checkout: %w", err)
	}
	return response, nil
}

func validateRequest(req *Request) error {
	if req.UserID == "" {
		return fmt.Errorf("%w: user is required", ErrInvalidRequest)
	}
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if len(req.IdempotencyKey) < 16 || len(req.IdempotencyKey) > 128 {
		return fmt.Errorf("%w: idempotency key must be 16 to 128 characters", ErrInvalidRequest)
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}
	if req.Currency != "INR" {
		return fmt.Errorf("%w: only INR is supported", ErrInvalidRequest)
	}
	if req.PaymentProvider == "" {
		req.PaymentProvider = "manual"
	}
	if req.PaymentMethod == "" {
		if req.PaymentProvider == "stripe" {
			req.PaymentMethod = "card"
		} else {
			req.PaymentMethod = "cod"
		}
	}
	if (req.PaymentProvider == "manual" && req.PaymentMethod != "cod") ||
		(req.PaymentProvider == "stripe" && req.PaymentMethod != "card") ||
		(req.PaymentProvider != "manual" && req.PaymentProvider != "stripe") {
		return fmt.Errorf("%w: payment method is not enabled", ErrInvalidRequest)
	}
	if req.ShippingMethod == "" {
		req.ShippingMethod = "standard"
	}
	if _, ok := ShippingCosts[req.ShippingMethod]; !ok {
		return fmt.Errorf("%w: invalid shipping method", ErrInvalidRequest)
	}
	var address map[string]any
	if err := json.Unmarshal([]byte(req.ShippingAddress), &address); err != nil || len(address) == 0 {
		return fmt.Errorf("%w: a valid shipping address is required", ErrInvalidRequest)
	}
	if pin, _ := address["pincode"].(string); !pincodeRe.MatchString(strings.TrimSpace(pin)) {
		return fmt.Errorf("%w: enter a valid 6-digit PIN code", ErrInvalidRequest)
	}
	for _, field := range []string{"address", "city", "state", "pincode"} {
		value, ok := address[field].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: shipping address %s is required", ErrInvalidRequest, field)
		}
	}
	name := strings.TrimSpace(req.CustomerName)
	if name == "" || len(name) > 200 {
		return fmt.Errorf("%w: customer name is required", ErrInvalidRequest)
	}
	email := strings.TrimSpace(strings.ToLower(req.CustomerEmail))
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return fmt.Errorf("%w: a valid customer email is required", ErrInvalidRequest)
	}
	req.CustomerName = name
	req.CustomerEmail = email
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.CouponCode = strings.ToUpper(strings.TrimSpace(req.CouponCode))
	if len(req.CustomerPhone) < 7 || len(req.CustomerPhone) > 32 {
		return fmt.Errorf("%w: a valid customer phone number is required", ErrInvalidRequest)
	}
	return nil
}

func hashRequest(req Request) (string, error) {
	value := struct {
		UserID          string `json:"user_id"`
		PaymentProvider string `json:"payment_provider"`
		PaymentMethod   string `json:"payment_method"`
		Currency        string `json:"currency"`
		ShippingMethod  string `json:"shipping_method"`
		ShippingAddress string `json:"shipping_address"`
		CustomerName    string `json:"customer_name"`
		CustomerEmail   string `json:"customer_email"`
		CustomerPhone   string `json:"customer_phone"`
		CouponCode      string `json:"coupon_code"`
	}{
		UserID: req.UserID, PaymentProvider: req.PaymentProvider, PaymentMethod: req.PaymentMethod,
		Currency: req.Currency, ShippingMethod: req.ShippingMethod, ShippingAddress: req.ShippingAddress,
		CustomerName: req.CustomerName, CustomerEmail: req.CustomerEmail, CustomerPhone: req.CustomerPhone, CouponCode: req.CouponCode,
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
