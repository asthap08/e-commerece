package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                                                      string
	HTTPAddr                                                    string
	DatabaseURL                                                 string
	RedisURL                                                    string
	StripeSecret                                                string
	StripeWebhook                                               string
	CORSOrigins                                                 []string
	CheckoutEnabled                                             bool
	SessionCookieSecure                                         bool
	TrustProxy                                                  bool
	SMTPHost, SMTPPort, SMTPUser, SMTPPass, MailFrom, PublicURL string
	RateLimitPerMinute                                          int
	AuthRateLimit                                               int
}

func Load() (Config, error) {
	appEnv := getenv("APP_ENV", "development")
	if appEnv != "production" {
		loadDotEnv()
		appEnv = getenv("APP_ENV", "development")
	}
	if appEnv != "development" && appEnv != "test" && appEnv != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development, test, or production")
	}
	checkoutEnabled := false
	if raw := strings.TrimSpace(os.Getenv("CHECKOUT_ENABLED")); raw != "" {
		var err error
		checkoutEnabled, err = strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("CHECKOUT_ENABLED must be true or false")
		}
	}
	c := Config{
		TrustProxy: strings.EqualFold(os.Getenv("TRUST_PROXY"), "true"),
		SMTPHost:   os.Getenv("SMTP_HOST"), SMTPPort: os.Getenv("SMTP_PORT"), SMTPUser: os.Getenv("SMTP_USER"), SMTPPass: os.Getenv("SMTP_PASS"),
		MailFrom: os.Getenv("MAIL_FROM"), PublicURL: strings.TrimRight(os.Getenv("PUBLIC_URL"), "/"),
		RateLimitPerMinute:  atoiEnv("RATE_LIMIT_PER_MINUTE", 300),
		AuthRateLimit:       atoiEnv("AUTH_RATE_LIMIT_PER_MINUTE", 20),
		AppEnv:              appEnv,
		HTTPAddr:            getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            os.Getenv("REDIS_URL"),
		StripeSecret:        os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhook:       os.Getenv("STRIPE_WEBHOOK_SECRET"),
		CheckoutEnabled:     checkoutEnabled,
		SessionCookieSecure: appEnv == "production",
	}
	origins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	for _, origin := range origins {
		if origin = strings.TrimSpace(origin); origin != "" {
			if origin == "*" {
				return Config{}, fmt.Errorf("wildcard CORS origins are not allowed")
			}
			c.CORSOrigins = append(c.CORSOrigins, origin)
		}
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.CORSOrigins) == 0 && c.AppEnv != "production" {
		c.CORSOrigins = []string{"http://localhost:3001", "http://localhost:5173"}
	}
	if c.AppEnv == "production" && len(c.CORSOrigins) == 0 {
		return Config{}, fmt.Errorf("CORS_ORIGINS is required in production")
	}
	if c.AppEnv == "production" && c.StripeSecret != "" && c.StripeWebhook == "" {
		return c, fmt.Errorf("STRIPE_WEBHOOK_SECRET is required when Stripe is enabled in production")
	}
	if c.AppEnv == "production" && c.CheckoutEnabled {
		// Taking real orders needs a secure cookie context: every allowed browser
		// origin must be https, otherwise Secure session cookies are never sent back.
		for _, o := range c.CORSOrigins {
			if !strings.HasPrefix(o, "https://") {
				return c, fmt.Errorf("checkout in production requires https CORS_ORIGINS (got %q)", o)
			}
		}
		if (c.StripeSecret == "") != (c.StripeWebhook == "") {
			return c, fmt.Errorf("set both STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET, or neither (cash on delivery only)")
		}
	}
	return c, nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func atoiEnv(k string, d int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(k))); err == nil && n > 0 {
		return n
	}
	return d
}
