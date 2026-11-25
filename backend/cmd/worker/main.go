// Command worker runs periodic housekeeping:
//   - cancels card orders that were never paid (and returns their stock),
//   - purges expired/revoked sessions and old idempotency keys,
//   - removes abandoned guest carts.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asthap08/ecommerce/internal/platform/config"
	"github.com/asthap08/ecommerce/internal/platform/db"
	"github.com/asthap08/ecommerce/internal/platform/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Unpaid card orders hold stock (checkout deducts it up front), so they are
// released after this long. Cash-on-delivery orders are intentionally excluded.
const unpaidCardOrderTTL = 2 * time.Hour

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logger.New(cfg.AppEnv)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("worker db connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("background worker started")

	run := func() {
		if n, err := cancelStaleCardOrders(ctx, pool); err != nil {
			log.Error("stale order cleanup failed", "err", err)
		} else if n > 0 {
			log.Info("cancelled unpaid card orders and restocked items", "orders", n)
		}
		for name, q := range map[string]string{
			"sessions":     `DELETE FROM user_sessions WHERE expires_at < now() - interval '1 day' OR (revoked_at IS NOT NULL AND revoked_at < now() - interval '1 day')`,
			"idempotency":  `DELETE FROM checkout_idempotency WHERE created_at < now() - interval '2 days'`,
			"reset tokens": `DELETE FROM password_resets WHERE expires_at < now() - interval '1 day'`,
			"guest carts":  `DELETE FROM carts WHERE user_id IS NULL AND updated_at < now() - interval '30 days'`,
		} {
			if tag, err := pool.Exec(ctx, q); err != nil {
				log.Error("cleanup failed", "job", name, "err", err)
			} else if tag.RowsAffected() > 0 {
				log.Info("cleanup", "job", name, "rows", tag.RowsAffected())
			}
		}
	}
	run()
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopped")
			return
		case <-t.C:
			run()
		}
	}
}

func cancelStaleCardOrders(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT o.id FROM orders o
		WHERE o.status IN ('pending','payment_failed') AND o.created_at < now() - make_interval(secs => $1)
		  AND EXISTS (SELECT 1 FROM payments p WHERE p.order_id=o.id AND p.provider='stripe')
		FOR UPDATE OF o SKIP LOCKED`, unpaidCardOrderTTL.Seconds())
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE orders SET status='cancelled', updated_at=now() WHERE id=$1`, id); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `UPDATE inventory i SET quantity=i.quantity+oi.quantity FROM order_items oi WHERE oi.order_id=$1 AND i.variant_id=oi.variant_id`, id); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='cancelled' WHERE order_id=$1 AND status IN ('pending','failed')`, id); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `UPDATE coupons SET used_count=GREATEST(used_count-1,0) WHERE upper(code)=(SELECT upper(coupon_code) FROM orders WHERE id=$1 AND coupon_code<>'')`, id); err != nil {
			return 0, err
		}
	}
	return len(ids), tx.Commit(ctx)
}
