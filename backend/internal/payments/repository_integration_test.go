package payments

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/asthap08/ecommerce/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStripeWebhookPaymentLifecycle(t *testing.T) {
	pool := newPaymentTestPool(t)
	ctx := context.Background()
	repo := NewRepository(pool)
	payment := seedStripePayment(t, ctx, pool, "pending", 2)

	intent := StripeEventIntent{
		ID:             payment.ProviderRef,
		Amount:         payment.AmountMinor,
		AmountReceived: payment.AmountMinor,
		Currency:       "inr",
		OrderID:        payment.OrderID,
		PaymentID:      payment.ID,
	}
	mismatch := intent
	mismatch.Amount++
	if err := repo.ProcessStripeEvent(ctx, "evt_bad_amount", "payment_intent.succeeded", mismatch); err == nil {
		t.Fatal("expected mismatched amount to be rejected")
	}

	if err := repo.ProcessStripeEvent(ctx, "evt_success", "payment_intent.succeeded", intent); err != nil {
		t.Fatalf("process successful event: %v", err)
	}
	if err := repo.ProcessStripeEvent(ctx, "evt_success", "payment_intent.succeeded", intent); err != nil {
		t.Fatalf("process duplicate event: %v", err)
	}
	assertStripeState(t, ctx, pool, payment, "completed", "paid", 8)

	lateFailure := intent
	if err := repo.ProcessStripeEvent(ctx, "evt_late_failure", "payment_intent.payment_failed", lateFailure); err != nil {
		t.Fatalf("process late failure event: %v", err)
	}
	assertStripeState(t, ctx, pool, payment, "completed", "paid", 8)

	retryPayment := seedStripePayment(t, ctx, pool, "pending", 1)
	retryIntent := StripeEventIntent{
		ID: retryPayment.ProviderRef, Amount: retryPayment.AmountMinor, Currency: "inr",
		OrderID: retryPayment.OrderID, PaymentID: retryPayment.ID,
	}
	if err := repo.ProcessStripeEvent(ctx, "evt_failure", "payment_intent.payment_failed", retryIntent); err != nil {
		t.Fatalf("process failure event: %v", err)
	}
	retryIntent.AmountReceived = retryPayment.AmountMinor
	if err := repo.ProcessStripeEvent(ctx, "evt_retry_success", "payment_intent.succeeded", retryIntent); err != nil {
		t.Fatalf("process retry success event: %v", err)
	}
	assertStripeState(t, ctx, pool, retryPayment, "completed", "paid", 9)

	cancelPayment := seedStripePayment(t, ctx, pool, "pending", 3)
	cancelIntent := StripeEventIntent{
		ID: cancelPayment.ProviderRef, Amount: cancelPayment.AmountMinor, Currency: "inr",
		OrderID: cancelPayment.OrderID, PaymentID: cancelPayment.ID,
	}
	if err := repo.ProcessStripeEvent(ctx, "evt_cancelled", "payment_intent.canceled", cancelIntent); err != nil {
		t.Fatalf("process canceled event: %v", err)
	}
	assertStripeState(t, ctx, pool, cancelPayment, "cancelled", "cancelled", 10)
}

func newPaymentTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("CHECKOUT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CHECKOUT_TEST_DATABASE_URL to run PostgreSQL payment integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { adminPool.Close() })
	if err := adminPool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	schema := fmt.Sprintf("payment_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	testConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	testConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	})
	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("rerun migrations: %v", err)
	}
	return pool
}

func seedStripePayment(t *testing.T, ctx context.Context, pool *pgxpool.Pool, status string, quantity int) Payment {
	t.Helper()
	var userID, productID, variantID, orderID, paymentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users(email,password_hash) VALUES($1,'test') RETURNING id
	`, fmt.Sprintf("payment-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(name,slug,status) VALUES('Payment product',$1,'active') RETURNING id
	`, fmt.Sprintf("payment-%d", time.Now().UnixNano())).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO product_variants(product_id,sku,price_minor,currency)
		VALUES($1,$2,10000,'INR') RETURNING id
	`, productID, fmt.Sprintf("PAY-%d", time.Now().UnixNano())).Scan(&variantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO inventory(variant_id,quantity,reserved) VALUES($1,$2,0)`, variantID, 10-quantity); err != nil {
		t.Fatal(err)
	}
	total := int64(quantity) * 10000
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders(user_id,status,total_minor,currency,payment_method)
		VALUES($1,'pending',$2,'INR','card') RETURNING id
	`, userID, total).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO order_items(order_id,variant_id,sku,name,price_minor,quantity)
		VALUES($1,$2,$3,'Payment product',10000,$4)
	`, orderID, variantID, fmt.Sprintf("PAY-%d", time.Now().UnixNano()), quantity); err != nil {
		t.Fatal(err)
	}
	providerRef := fmt.Sprintf("pi_%d", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `
		INSERT INTO payments(order_id,provider,provider_ref,status,amount_minor,method)
		VALUES($1,'stripe',$2,$3,$4,'card') RETURNING id
	`, orderID, providerRef, status, total).Scan(&paymentID); err != nil {
		t.Fatal(err)
	}
	return Payment{
		ID: paymentID, OrderID: orderID, Provider: "stripe",
		ProviderRef: providerRef,
		Status:      status, AmountMinor: total, Method: "card",
	}
}

func assertStripeState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, payment Payment, paymentStatus, orderStatus string, wantStock int) {
	t.Helper()
	var gotPaymentStatus, gotOrderStatus string
	var gotStock int
	err := pool.QueryRow(ctx, `
		SELECT p.status,o.status,i.quantity
		FROM payments p
		JOIN orders o ON o.id=p.order_id
		JOIN order_items oi ON oi.order_id=o.id
		JOIN inventory i ON i.variant_id=oi.variant_id
		WHERE p.id=$1
	`, payment.ID).Scan(&gotPaymentStatus, &gotOrderStatus, &gotStock)
	if err != nil {
		t.Fatal(err)
	}
	if gotPaymentStatus != paymentStatus || gotOrderStatus != orderStatus || gotStock != wantStock {
		t.Fatalf("payment/order/stock = %q/%q/%d, want %q/%q/%d",
			gotPaymentStatus, gotOrderStatus, gotStock, paymentStatus, orderStatus, wantStock)
	}
}
