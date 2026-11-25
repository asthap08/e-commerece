package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/asthap08/ecommerce/internal/platform/session"
)

type testSessionResolver struct{}

func (testSessionResolver) Resolve(_ context.Context, token string, _ time.Time) (session.Identity, error) {
	if token != "valid-session" {
		return session.Identity{}, session.ErrInvalidSession
	}
	return session.Identity{UserID: "user-123", Role: "customer"}, nil
}

func TestAuthRejectsMissingAndInvalidSessions(t *testing.T) {
	handler := Auth(testSessionResolver{})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called without a valid session cookie")
	}))
	for _, cookieValue := range []string{"", "invalid"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if cookieValue != "" {
			req.AddCookie(&http.Cookie{Name: session.CookieName, Value: cookieValue})
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Errorf("status for cookie %q = %d, want %d", cookieValue, res.Code, http.StatusUnauthorized)
		}
	}
}

func TestAuthAddsSessionIdentityToContext(t *testing.T) {
	handler := Auth(testSessionResolver{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := UserID(r.Context()); got != "user-123" {
			t.Errorf("user ID = %q, want user-123", got)
		}
		if got := UserRole(r.Context()); got != "customer" {
			t.Errorf("role = %q, want customer", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "valid-session"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestAuthDoesNotAcceptBearerTokens(t *testing.T) {
	handler := Auth(testSessionResolver{})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not accept a bearer token in place of a cookie")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-session")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}
