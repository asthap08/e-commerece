package cart

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/asthap08/ecommerce/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCartRepositoryConcurrentCreateAndMerge(t *testing.T) {
	databaseURL := os.Getenv("CART_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CART_TEST_DATABASE_URL to run PostgreSQL cart integration tests")
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

	schema := fmt.Sprintf("cart_test_%d", time.Now().UnixNano())
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
	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer testPool.Close()

	if err := migrations.Run(ctx, testPool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	repo := NewRepository(testPool)

	var userID string
	if err := testPool.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES('cart@example.test','test') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	var productID string
	if err := testPool.QueryRow(ctx, `INSERT INTO products(name,slug,status) VALUES('Cart Test','cart-test','active') RETURNING id`).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	var firstVariantID, secondVariantID string
	if err := testPool.QueryRow(ctx, `INSERT INTO product_variants(product_id,sku,price_minor) VALUES($1,'CART-1',100) RETURNING id`, productID).Scan(&firstVariantID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO product_variants(product_id,sku,price_minor) VALUES($1,'CART-2',200) RETURNING id`, productID).Scan(&secondVariantID); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, `INSERT INTO product_images(product_id,url,position) VALUES($1,'https://images.example.test/cart-product.webp',0)`, productID); err != nil {
		t.Fatal(err)
	}

	const workers = 12
	var wait sync.WaitGroup
	ids := make(chan string, workers)
	errs := make(chan error, workers)
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			id, err := repo.GetOrCreateCart(ctx, userID)
			if err != nil {
				errs <- err
				return
			}
			ids <- id
		}()
	}
	wait.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent cart creation: %v", err)
	}
	var userCartID string
	for id := range ids {
		if userCartID == "" {
			userCartID = id
		} else if id != userCartID {
			t.Fatalf("concurrent cart creation returned IDs %q and %q", userCartID, id)
		}
	}
	if userCartID == "" {
		t.Fatal("concurrent cart creation returned no cart")
	}

	if err := repo.AddItem(ctx, userCartID, firstVariantID, 2); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddItem(ctx, userCartID, firstVariantID, 3); err != nil {
		t.Fatal(err)
	}
	sessionCartID, err := repo.GetOrCreateCartBySession(ctx, "guest-session")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AddItem(ctx, sessionCartID, firstVariantID, 4); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddItem(ctx, sessionCartID, secondVariantID, 1); err != nil {
		t.Fatal(err)
	}

	if err := repo.MergeSessionCart(ctx, "guest-session", userID); err != nil {
		t.Fatal(err)
	}
	if err := repo.MergeSessionCart(ctx, "guest-session", userID); err != nil {
		t.Fatal(err)
	}
	items, err := repo.ListItems(ctx, userCartID)
	if err != nil {
		t.Fatal(err)
	}
	quantities := make(map[string]int, len(items))
	for _, item := range items {
		quantities[item.VariantID] = item.Quantity
		if item.Name != "Cart Test" || item.ImageURL != "https://images.example.test/cart-product.webp" {
			t.Errorf("cart item product details = %#v", item)
		}
		if item.VariantID == firstVariantID && item.PriceMinor != 100 {
			t.Errorf("first variant price = %d, want 100", item.PriceMinor)
		}
		if item.VariantID == secondVariantID && item.PriceMinor != 200 {
			t.Errorf("second variant price = %d, want 200", item.PriceMinor)
		}
	}
	if quantities[firstVariantID] != 9 || quantities[secondVariantID] != 1 || len(items) != 2 {
		t.Fatalf("merged items = %#v, want quantities %s=9 and %s=1", quantities, firstVariantID, secondVariantID)
	}

	var remainingSessionCarts int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM carts WHERE session_id='guest-session'`).Scan(&remainingSessionCarts); err != nil {
		t.Fatal(err)
	}
	if remainingSessionCarts != 0 {
		t.Fatalf("remaining guest carts = %d, want 0", remainingSessionCarts)
	}

	var legacyUserID string
	if err := testPool.QueryRow(ctx, `INSERT INTO users(email,password_hash) VALUES('legacy-cart@example.test','test') RETURNING id`).Scan(&legacyUserID); err != nil {
		t.Fatal(err)
	}
	for _, index := range []string{
		"idx_carts_user_id_unique",
		"idx_carts_session_id_unique",
		"idx_cart_items_cart_variant_unique",
	} {
		if _, err := testPool.Exec(ctx, "DROP INDEX "+index); err != nil {
			t.Fatal(err)
		}
	}
	var legacyCartIDs [2]string
	for i := range legacyCartIDs {
		if err := testPool.QueryRow(ctx, `INSERT INTO carts(user_id) VALUES($1) RETURNING id`, legacyUserID).Scan(&legacyCartIDs[i]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := testPool.Exec(ctx, `
		INSERT INTO cart_items(cart_id, variant_id, quantity) VALUES
			($1, $2, 1),
			($1, $2, 2),
			($3, $2, 3)
	`, legacyCartIDs[0], firstVariantID, legacyCartIDs[1]); err != nil {
		t.Fatal(err)
	}
	var legacySessionCartIDs [2]string
	for i := range legacySessionCartIDs {
		if err := testPool.QueryRow(ctx, `INSERT INTO carts(session_id) VALUES('legacy-session') RETURNING id`).Scan(&legacySessionCartIDs[i]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := testPool.Exec(ctx, `
		INSERT INTO cart_items(cart_id, variant_id, quantity) VALUES
			($1, $2, 2),
			($3, $2, 3)
	`, legacySessionCartIDs[0], secondVariantID, legacySessionCartIDs[1]); err != nil {
		t.Fatal(err)
	}

	// Simulate a database created by the old runner (which had no version table):
	// forgetting migration 0005 makes the versioned runner apply its de-duplication again.
	if _, err := testPool.Exec(ctx, `DELETE FROM schema_migrations WHERE version LIKE '0005%'`); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(ctx, testPool); err != nil {
		t.Fatalf("rerun migrations with legacy duplicate carts: %v", err)
	}
	var legacyUserCartCount int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM carts WHERE user_id=$1`, legacyUserID).Scan(&legacyUserCartCount); err != nil {
		t.Fatal(err)
	}
	if legacyUserCartCount != 1 {
		t.Fatalf("legacy user cart count = %d, want 1", legacyUserCartCount)
	}
	var legacyUserQuantity int
	if err := testPool.QueryRow(ctx, `
		SELECT quantity FROM cart_items
		WHERE cart_id = (SELECT id FROM carts WHERE user_id=$1)
		  AND variant_id=$2
	`, legacyUserID, firstVariantID).Scan(&legacyUserQuantity); err != nil {
		t.Fatal(err)
	}
	if legacyUserQuantity != 6 {
		t.Fatalf("legacy user cart quantity = %d, want 6", legacyUserQuantity)
	}
	var legacySessionCartCount, legacySessionQuantity int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM carts WHERE session_id='legacy-session'`).Scan(&legacySessionCartCount); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `
		SELECT quantity FROM cart_items
		WHERE cart_id = (SELECT id FROM carts WHERE session_id='legacy-session')
		  AND variant_id=$1
	`, secondVariantID).Scan(&legacySessionQuantity); err != nil {
		t.Fatal(err)
	}
	if legacySessionCartCount != 1 || legacySessionQuantity != 5 {
		t.Fatalf("legacy session cart count/quantity = %d/%d, want 1/5", legacySessionCartCount, legacySessionQuantity)
	}
	if err := migrations.Run(ctx, testPool); err != nil {
		t.Fatalf("rerun idempotent migrations: %v", err)
	}
}
