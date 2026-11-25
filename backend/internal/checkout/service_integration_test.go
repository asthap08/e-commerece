package checkout

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/asthap08/ecommerce/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCheckoutIsAtomicAndIdempotent(t *testing.T) {
	databaseURL := os.Getenv("CHECKOUT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CHECKOUT_TEST_DATABASE_URL to run PostgreSQL checkout integration tests")
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
	defer adminPool.Close()
	if err := adminPool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	schema := fmt.Sprintf("checkout_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	}()
	testConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	testConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatalf("rerun migrations: %v", err)
	}

	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES('checkout@example.test','test') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(name,slug,status) VALUES('Checkout product','checkout-product','active') RETURNING id`).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	const firstVariantID = "00000000-0000-0000-0000-000000000001"
	const secondVariantID = "00000000-0000-0000-0000-000000000002"
	if _, err := pool.Exec(ctx, `INSERT INTO product_variants(id,product_id,sku,price_minor,currency) VALUES($1,$2,'CHECKOUT-1',25000,'INR'),($3,$2,'CHECKOUT-2',12000,'INR')`, firstVariantID, productID, secondVariantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO inventory(variant_id,quantity,reserved) VALUES($1,10,0),($2,0,0)`, firstVariantID, secondVariantID); err != nil {
		t.Fatal(err)
	}
	var cartID string
	if err := pool.QueryRow(ctx, `INSERT INTO carts(user_id) VALUES($1) RETURNING id`, userID).Scan(&cartID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO cart_items(cart_id,variant_id,quantity) VALUES($1,$2,2)`, cartID, firstVariantID); err != nil {
		t.Fatal(err)
	}

	service := NewService(pool)
	req := testCheckoutRequest(userID, "checkout-idempotency-key-001")
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION reject_payment_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'injected payment failure';
		END;
		$$
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE TRIGGER reject_checkout_payment BEFORE INSERT ON payments FOR EACH ROW EXECUTE FUNCTION reject_payment_insert()`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Checkout(ctx, req); err == nil {
		t.Fatal("expected injected payment write to fail")
	}
	assertCheckoutState(t, ctx, pool, userID, cartID, firstVariantID, 10, 1, 0)

	if _, err := pool.Exec(ctx, `DROP TRIGGER reject_checkout_payment ON payments`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DROP FUNCTION reject_payment_insert()`); err != nil {
		t.Fatal(err)
	}
	firstResponse, err := service.Checkout(ctx, req)
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if firstResponse.TotalMinor != 50000 || firstResponse.Currency != "INR" || firstResponse.OrderID == "" || firstResponse.PaymentID == "" {
		t.Fatalf("checkout response = %#v", firstResponse)
	}
	secondResponse, err := service.Checkout(ctx, req)
	if err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	if secondResponse != firstResponse {
		t.Fatalf("retry response = %#v, want original %#v", secondResponse, firstResponse)
	}
	differentRequest := req
	differentRequest.ShippingMethod = "express"
	if _, err := service.Checkout(ctx, differentRequest); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("reused key with changed request error = %v, want idempotency conflict", err)
	}
	assertCheckoutState(t, ctx, pool, userID, cartID, firstVariantID, 8, 0, 1)

	if _, err := pool.Exec(ctx, `
		INSERT INTO cart_items(cart_id,variant_id,quantity) VALUES($1,$2,2),($1,$3,1)
	`, cartID, firstVariantID, secondVariantID); err != nil {
		t.Fatal(err)
	}
	stockFailureRequest := testCheckoutRequest(userID, "checkout-stock-failure-key-001")
	if _, err := service.Checkout(ctx, stockFailureRequest); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("insufficient-stock checkout error = %v, want insufficient inventory", err)
	}
	assertCheckoutState(t, ctx, pool, userID, cartID, firstVariantID, 8, 2, 1)
	var secondVariantStock int
	if err := pool.QueryRow(ctx, `SELECT quantity FROM inventory WHERE variant_id=$1`, secondVariantID).Scan(&secondVariantStock); err != nil {
		t.Fatal(err)
	}
	if secondVariantStock != 0 {
		t.Fatalf("second variant stock = %d, want 0", secondVariantStock)
	}
}

func testCheckoutRequest(userID, key string) Request {
	return Request{
		UserID:          userID,
		IdempotencyKey:  key,
		PaymentProvider: "manual",
		PaymentMethod:   "cod",
		Currency:        "INR",
		ShippingMethod:  "standard",
		ShippingAddress: `{"address":"12 Demo Road","city":"Mumbai","state":"Maharashtra","pincode":"400001"}`,
		CustomerName:    "Demo Customer",
		CustomerEmail:   "checkout@example.test",
		CustomerPhone:   "+919999999999",
	}
}

func assertCheckoutState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, cartID, variantID string, wantStock, wantCartItems, wantOrders int) {
	t.Helper()
	var gotStock, gotCartItems, gotOrders, gotPayments, idempotencyRows int
	if err := pool.QueryRow(ctx, `SELECT quantity FROM inventory WHERE variant_id=$1`, variantID).Scan(&gotStock); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cart_items WHERE cart_id=$1`, cartID).Scan(&gotCartItems); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE user_id=$1`, userID).Scan(&gotOrders); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments`).Scan(&gotPayments); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM checkout_idempotency WHERE user_id=$1`, userID).Scan(&idempotencyRows); err != nil {
		t.Fatal(err)
	}
	if gotStock != wantStock || gotCartItems != wantCartItems || gotOrders != wantOrders || gotPayments != wantOrders || idempotencyRows != wantOrders {
		t.Fatalf("checkout state stock/cart/orders/payments/idempotency = %d/%d/%d/%d/%d; want %d/%d/%d/%d/%d",
			gotStock, gotCartItems, gotOrders, gotPayments, idempotencyRows,
			wantStock, wantCartItems, wantOrders, wantOrders, wantOrders)
	}
}
