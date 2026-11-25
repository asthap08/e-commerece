package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPaymentNotFound = errors.New("payment not found")
var ErrInvalidStripeEvent = errors.New("invalid Stripe payment event")

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) GetOrderPayment(ctx context.Context, userID, orderID string) (Payment, string, error) {
	var payment Payment
	var currency string
	err := r.pool.QueryRow(ctx, `
		SELECT p.id::text, p.order_id::text, p.provider, COALESCE(p.provider_ref, ''),
		       p.status, p.amount_minor, p.method, o.currency
		FROM payments p
		JOIN orders o ON o.id=p.order_id
		WHERE p.order_id=$1 AND o.user_id=$2 AND p.provider='stripe' AND p.method='card'
		  AND p.status IN ('pending','failed') AND o.status IN ('pending','payment_failed')
	`, orderID, userID).Scan(&payment.ID, &payment.OrderID, &payment.Provider, &payment.ProviderRef,
		&payment.Status, &payment.AmountMinor, &payment.Method, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, "", ErrPaymentNotFound
	}
	return payment, currency, err
}

func (r *Repository) SaveProviderRef(ctx context.Context, paymentID, providerRef string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE payments
		SET provider_ref=$2
		WHERE id=$1 AND provider='stripe' AND status IN ('pending','failed')
		  AND (provider_ref IS NULL OR provider_ref='' OR provider_ref=$2)
	`, paymentID, providerRef)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrPaymentNotFound
	}
	return nil
}

type StripeEventIntent struct {
	ID             string
	Amount         int64
	AmountReceived int64
	Currency       string
	OrderID        string
	PaymentID      string
}

func (r *Repository) ProcessStripeEvent(ctx context.Context, eventID, eventType string, intent StripeEventIntent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted, err := tx.Exec(ctx, `
		INSERT INTO payment_webhook_events(provider,event_id,event_type)
		VALUES('stripe',$1,$2)
		ON CONFLICT (provider,event_id) DO NOTHING
	`, eventID, eventType)
	if err != nil {
		return err
	}
	if inserted.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}

	switch eventType {
	case "payment_intent.succeeded", "payment_intent.payment_failed", "payment_intent.canceled":
	default:
		return tx.Commit(ctx)
	}

	var payment Payment
	err = tx.QueryRow(ctx, `
		SELECT id::text, order_id::text, provider, COALESCE(provider_ref,''),
		       status, amount_minor, method
		FROM payments
		WHERE provider='stripe' AND provider_ref=$1
		FOR UPDATE
	`, intent.ID).Scan(&payment.ID, &payment.OrderID, &payment.Provider, &payment.ProviderRef,
		&payment.Status, &payment.AmountMinor, &payment.Method)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidStripeEvent
		}
		return err
	}
	if intent.OrderID != payment.OrderID || intent.PaymentID != payment.ID ||
		intent.Amount != payment.AmountMinor || intent.Currency != "inr" {
		return fmt.Errorf("%w: event does not match the stored payment", ErrInvalidStripeEvent)
	}
	if eventType == "payment_intent.succeeded" && intent.AmountReceived != payment.AmountMinor {
		return fmt.Errorf("%w: received amount does not match the stored payment", ErrInvalidStripeEvent)
	}

	var orderStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1 FOR UPDATE`, payment.OrderID).Scan(&orderStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidStripeEvent
		}
		return err
	}

	switch eventType {
	case "payment_intent.succeeded":
		if payment.Status == "completed" && orderStatus == "paid" {
			break
		}
		if payment.Status != "pending" && payment.Status != "failed" {
			return fmt.Errorf("%w: payment cannot transition to completed", ErrInvalidStripeEvent)
		}
		if orderStatus != "pending" && orderStatus != "payment_failed" {
			return fmt.Errorf("%w: order cannot transition to paid", ErrInvalidStripeEvent)
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='completed' WHERE id=$1`, payment.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET status='paid' WHERE id=$1`, payment.OrderID); err != nil {
			return err
		}
	case "payment_intent.payment_failed":
		if payment.Status == "completed" {
			break
		}
		if payment.Status != "pending" && payment.Status != "failed" {
			return fmt.Errorf("%w: payment cannot transition to failed", ErrInvalidStripeEvent)
		}
		if orderStatus != "pending" && orderStatus != "payment_failed" {
			return fmt.Errorf("%w: order cannot transition to payment_failed", ErrInvalidStripeEvent)
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='failed' WHERE id=$1`, payment.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET status='payment_failed' WHERE id=$1`, payment.OrderID); err != nil {
			return err
		}
	case "payment_intent.canceled":
		if payment.Status == "completed" {
			break
		}
		if payment.Status != "pending" && payment.Status != "failed" {
			return fmt.Errorf("%w: payment cannot transition to canceled", ErrInvalidStripeEvent)
		}
		if orderStatus != "pending" && orderStatus != "payment_failed" {
			return fmt.Errorf("%w: order cannot transition to canceled", ErrInvalidStripeEvent)
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='cancelled' WHERE id=$1`, payment.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET status='cancelled' WHERE id=$1`, payment.OrderID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory i SET quantity=i.quantity+oi.quantity
			FROM order_items oi
			WHERE oi.order_id=$1 AND i.variant_id=oi.variant_id
		`, payment.OrderID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
