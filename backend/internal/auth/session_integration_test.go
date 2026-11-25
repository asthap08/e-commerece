package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/asthap08/ecommerce/internal/platform/session"
	"github.com/asthap08/ecommerce/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSessionCookieRegistrationLoginLogoutAndExpiration(t *testing.T) {
	pool := newAuthSessionTestPool(t)
	ctx := context.Background()
	store := session.NewStore(pool)
	handler := NewHandler(NewService(NewRepository(pool), nil), store, true, http.SameSiteNoneMode)
	email := fmt.Sprintf("session-%d@example.test", time.Now().UnixNano())
	body := `{"email":"` + email + `","password":"Correct-Horse9battery"}`

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", registerResponse.Code, registerResponse.Body)
	}
	registerCookie := assertSessionCookie(t, registerResponse)
	if strings.Contains(registerResponse.Body.String(), registerCookie.Value) {
		t.Fatal("registration response exposed the opaque session token")
	}
	hash := sha256.Sum256([]byte(registerCookie.Value))
	var storedHash string
	if err := pool.QueryRow(ctx, `SELECT token_hash FROM user_sessions WHERE token_hash=$1`, hex.EncodeToString(hash[:])).Scan(&storedHash); err != nil {
		t.Fatalf("session token hash was not persisted: %v", err)
	}
	if storedHash == registerCookie.Value {
		t.Fatal("session token was stored in plaintext")
	}

	meHandler := middleware.Auth(store)(http.HandlerFunc(handler.Me))
	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRequest.AddCookie(registerCookie)
	meResponse := httptest.NewRecorder()
	meHandler.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("session introspection status = %d, body = %s", meResponse.Code, meResponse.Body)
	}
	var identity map[string]string
	if err := json.Unmarshal(meResponse.Body.Bytes(), &identity); err != nil {
		t.Fatal(err)
	}
	if identity["role"] != "customer" || identity["id"] == "" {
		t.Fatalf("session identity = %#v", identity)
	}

	logoutHandler := middleware.Auth(store)(http.HandlerFunc(handler.Logout))
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.AddCookie(registerCookie)
	logoutResponse := httptest.NewRecorder()
	logoutHandler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", logoutResponse.Code, http.StatusNoContent)
	}
	logoutCookies := logoutResponse.Result().Cookies()
	if len(logoutCookies) != 1 {
		t.Fatalf("logout cookies = %d, want 1", len(logoutCookies))
	}
	if logoutCookie := logoutCookies[0]; logoutCookie.MaxAge >= 0 {
		t.Fatalf("logout cookie MaxAge = %d, want expired", logoutCookie.MaxAge)
	}
	if _, err := store.Resolve(ctx, registerCookie.Value, time.Now()); err == nil {
		t.Fatal("revoked session still resolves")
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginResponse.Code, loginResponse.Body)
	}
	loginCookie := assertSessionCookie(t, loginResponse)
	if _, err := store.Resolve(ctx, loginCookie.Value, time.Now()); err != nil {
		t.Fatalf("login session does not resolve: %v", err)
	}
	if _, err := store.Resolve(ctx, loginCookie.Value, time.Now().Add(session.Lifetime)); err == nil {
		t.Fatal("expired session still resolves")
	}
}

func assertSessionCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name != session.CookieName {
			continue
		}
		if !cookie.HttpOnly || !cookie.Secure || cookie.Path != "/" ||
			cookie.SameSite != http.SameSiteNoneMode || cookie.Value == "" {
			t.Fatalf("session cookie flags are unsafe or incomplete: %#v", cookie)
		}
		return cookie
	}
	t.Fatal("response did not set a session cookie")
	return nil
}

func newAuthSessionTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("CHECKOUT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CHECKOUT_TEST_DATABASE_URL to run PostgreSQL session integration tests")
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

	schema := fmt.Sprintf("auth_session_test_%d", time.Now().UnixNano())
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
