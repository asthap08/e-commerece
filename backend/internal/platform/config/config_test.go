package config

import (
	"strings"
	"testing"
)

func setProductionEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("CHECKOUT_ENABLED", "")
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/commerce")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("STRIPE_SECRET_KEY", "")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	t.Setenv("CORS_ORIGINS", "https://shop.example.com,https://admin.example.com")
}

func TestLoadRequiresWebhookSecretWhenStripeEnabledInProduction(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_example")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing Stripe webhook secret to fail")
	}
}

func TestLoadAcceptsProductionConfiguration(t *testing.T) {
	setProductionEnv(t)
	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRequiresCORSOriginsInProduction(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("CORS_ORIGINS", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing production CORS origins to fail")
	}
}

func TestProductionCheckoutRequiresHTTPSOrigins(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("CHECKOUT_ENABLED", "true")
	t.Setenv("CORS_ORIGINS", "http://shop.example.com")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("Load() error = %v, want https requirement", err)
	}
	t.Setenv("CORS_ORIGINS", "https://shop.example.com")
	if _, err := Load(); err != nil {
		t.Fatalf("production checkout with https origins should load: %v", err)
	}
	t.Setenv("STRIPE_SECRET_KEY", "sk_live_x")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("stripe key without webhook secret must be rejected")
	}
}

func TestLoadRejectsWildcardCORSOrigin(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("CORS_ORIGINS", "*")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "wildcard CORS origins") {
		t.Fatalf("Load() error = %v, want wildcard CORS rejection", err)
	}
}
