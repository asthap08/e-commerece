package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asthap08/ecommerce/internal/addresses"
	"github.com/asthap08/ecommerce/internal/admin"
	"github.com/asthap08/ecommerce/internal/audit"
	"github.com/asthap08/ecommerce/internal/auth"
	"github.com/asthap08/ecommerce/internal/cart"
	"github.com/asthap08/ecommerce/internal/catalog"
	"github.com/asthap08/ecommerce/internal/checkout"
	"github.com/asthap08/ecommerce/internal/digital"
	"github.com/asthap08/ecommerce/internal/health"
	"github.com/asthap08/ecommerce/internal/inventory"
	"github.com/asthap08/ecommerce/internal/mailer"
	"github.com/asthap08/ecommerce/internal/notifications"
	"github.com/asthap08/ecommerce/internal/orders"
	"github.com/asthap08/ecommerce/internal/payments"
	"github.com/asthap08/ecommerce/internal/platform/config"
	"github.com/asthap08/ecommerce/internal/platform/db"
	"github.com/asthap08/ecommerce/internal/platform/httpserver"
	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/logger"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/asthap08/ecommerce/internal/platform/session"
	"github.com/asthap08/ecommerce/internal/pricing"
	"github.com/asthap08/ecommerce/internal/returns"
	"github.com/asthap08/ecommerce/internal/reviews"
	"github.com/asthap08/ecommerce/internal/shipping"
	"github.com/asthap08/ecommerce/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log := logger.New(cfg.AppEnv)
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(signalCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	if err := migrations.Run(signalCtx, pool); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}
	log.Info("database schema migrations applied successfully")

	router := httpserver.NewRouter(cfg.TrustProxy)
	router.Use(middleware.CORS(cfg.CORSOrigins))
	router.Use(middleware.Compression)
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.NewRateLimiter(cfg.RateLimitPerMinute, time.Minute).Limit)
	router.Use(middleware.NewRateLimiter(cfg.AuthRateLimit, time.Minute).OnlyPaths("/api/v1/auth/login", "/api/v1/auth/register", "/api/v1/auth/forgot", "/api/v1/auth/reset").Limit)
	router.Use(middleware.CSRF(cfg.CORSOrigins, session.CookieName))
	router.Use(middleware.CacheControl(60))
	router.Use(maxBody)
	router.Use(middleware.Logging(log))

	router.Get("/health", health.New(pool).ServeHTTP)
	router.Get("/api/v1/health", health.New(pool).ServeHTTP)

	// Audit
	auditRepo := audit.NewRepository(pool)
	auditSvc := audit.NewService(auditRepo)

	// Auth
	authRepo := auth.NewRepository(pool)
	sessionStore := session.NewStore(pool)
	authenticate := middleware.Auth(sessionStore)
	sameSite := http.SameSiteLaxMode
	if cfg.SessionCookieSecure {
		sameSite = http.SameSiteNoneMode
	}
	mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.MailFrom, log)
	publicURL := cfg.PublicURL
	if publicURL == "" && len(cfg.CORSOrigins) > 0 {
		publicURL = cfg.CORSOrigins[0]
	}
	authHandler := auth.NewHandler(auth.NewService(authRepo, auditSvc).WithMail(mail, publicURL, log), sessionStore, cfg.SessionCookieSecure, sameSite)
	auth.RegisterRoutes(router, authHandler, authenticate)

	// Catalog
	catalogRepo := catalog.NewRepository(pool)
	catalogSvc := catalog.NewService(catalogRepo)
	catalog.RegisterRoutes(router, catalog.NewHandler(catalogSvc), authenticate)

	// Cart
	cartRepo := cart.NewRepository(pool)
	cartSvc := cart.NewService(cartRepo)
	cart.RegisterRoutes(router, cart.NewHandlerWithService(cartSvc), authenticate)

	// Digital
	digitalRepo := digital.NewRepository(pool)
	digital.RegisterRoutes(router, digital.NewHandlerWithService(digital.NewService(digitalRepo)), authenticate)

	// Inventory
	inventoryRepo := inventory.NewRepository(pool)
	inventorySvc := inventory.NewService(inventoryRepo)
	inventory.RegisterRoutes(router, inventory.NewHandlerWithService(inventorySvc), authenticate)

	// Notifications
	notifications.RegisterRoutes(router, notifications.NewHandlerWithService(notifications.NewService(log), nil), authenticate)

	// Orders (must be created before payments so payments can update order status)
	ordersRepo := orders.NewRepository(pool)
	ordersSvc := orders.NewService(ordersRepo)

	orders.RegisterRoutes(router, orders.NewHandlerWithService(ordersSvc), authenticate)
	admin.RegisterRoutes(router, admin.NewHandler(pool, ordersSvc), authenticate)

	// Payments
	paymentsRepo := payments.NewRepository(pool)
	paymentsSvc := payments.NewService(paymentsRepo, cfg.StripeSecret, cfg.StripeWebhook)
	payments.RegisterRoutes(router, payments.NewHandlerWithService(paymentsSvc), authenticate)

	// Pricing
	pricingRepo := pricing.NewRepository(pool)
	pricing.RegisterRoutes(router, pricing.NewHandlerWithService(pricing.NewService(pricingRepo)), authenticate)

	// Reviews
	reviewsRepo := reviews.NewRepository(pool)
	reviews.RegisterRoutes(router, reviews.NewHandler(reviews.NewService(reviewsRepo)), authenticate)

	// Addresses
	addressesRepo := addresses.NewRepository(pool)
	addresses.RegisterRoutes(router, addresses.NewHandler(addresses.NewService(addressesRepo)), authenticate)

	// Returns
	returnsRepo := returns.NewRepository(pool)
	returns.RegisterRoutes(router, returns.NewHandlerWithService(returns.NewService(returnsRepo)), authenticate)

	// Shipping
	shippingRepo := shipping.NewRepository(pool)
	shipping.RegisterRoutes(router, shipping.NewHandlerWithService(shipping.NewService(shippingRepo)), authenticate)

	// Checkout writes inventory, order, payment, cart, and idempotency state atomically.
	checkoutSvc := checkout.NewService(pool, cfg.StripeSecret != "" && cfg.StripeWebhook != "")
	checkoutHandler := checkout.NewHandlerWithService(checkoutSvc, cfg.CheckoutEnabled)
	checkoutHandler.OnPlaced(func(email, name, orderID string, totalMinor int64) {
		go func() {
			body := fmt.Sprintf("Hi %s,\n\nThanks for your order! Order ID: %s\nTotal: Rs %.2f\n\nTrack it any time: %s/track-order?id=%s\n", name, orderID, float64(totalMinor)/100, publicURL, orderID)
			if err := mail.Send(email, "Your order is confirmed", body); err != nil {
				log.Error("order confirmation e-mail failed", "order", orderID, "err", err)
			}
		}()
	})
	checkout.RegisterRoutes(router, checkoutHandler, authenticate)

	srv := httpserver.New(cfg.HTTPAddr, router)
	log.Info("starting api", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	select {
	case err := <-serverErr:
		if err == nil {
			return errors.New("HTTP server stopped unexpectedly")
		}
		return fmt.Errorf("HTTP server failed: %w", err)
	case <-signalCtx.Done():
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shut down HTTP server: %w", err)
	}
	log.Info("server stopped")
	if err := <-serverErr; err != nil {
		return fmt.Errorf("HTTP server stopped with error: %w", err)
	}
	return nil
}

// maxBody caps every request body so a client cannot stream unbounded JSON.
func maxBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxBody)
		next.ServeHTTP(w, r)
	})
}
