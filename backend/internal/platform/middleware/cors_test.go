package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSAllowsConfiguredOriginOnly(t *testing.T) {
	handler := CORS([]string{"https://shop.example"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	for _, tc := range []struct {
		origin string
		want   string
	}{
		{origin: "https://shop.example", want: "https://shop.example"},
		{origin: "https://untrusted.example", want: ""},
	} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", tc.origin)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if got := res.Header().Get("Access-Control-Allow-Origin"); got != tc.want {
			t.Errorf("origin %q got allow-origin %q, want %q", tc.origin, got, tc.want)
		}
	}
}

func TestCORSPermitsPatchPreflightForConfiguredOrigin(t *testing.T) {
	handler := CORS([]string{"https://admin.example"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("preflight request should not reach the handler")
		}),
	)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/orders/order-1/status", nil)
	req.Header.Set("Origin", "https://admin.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", res.Code, http.StatusNoContent)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example" {
		t.Fatalf("allow-origin = %q, want configured admin origin", got)
	}
	if methods := res.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "PATCH") {
		t.Fatalf("allow-methods = %q, want PATCH", methods)
	}
	if headers := res.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(headers, "Idempotency-Key") {
		t.Fatalf("allow-headers = %q, want Idempotency-Key", headers)
	}
	if got := res.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow-credentials = %q, want true", got)
	}
}

func TestCSRFRejectsUntrustedOrMissingOriginForSessionRequests(t *testing.T) {
	handler := CSRF([]string{"https://shop.example"}, "omniretail_session")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	for _, origin := range []string{"", "https://evil.example"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", nil)
		req.AddCookie(&http.Cookie{Name: "omniretail_session", Value: "session"})
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusForbidden {
			t.Errorf("origin %q status = %d, want %d", origin, res.Code, http.StatusForbidden)
		}
	}
}

func TestCSRFAllowsConfiguredOriginAndUnauthenticatedCLIAuth(t *testing.T) {
	handler := CSRF([]string{"https://shop.example"}, "omniretail_session")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	configured := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", nil)
	configured.AddCookie(&http.Cookie{Name: "omniretail_session", Value: "session"})
	configured.Header.Set("Origin", "https://shop.example")
	configuredResponse := httptest.NewRecorder()
	handler.ServeHTTP(configuredResponse, configured)
	if configuredResponse.Code != http.StatusNoContent {
		t.Fatalf("configured-origin status = %d, want %d", configuredResponse.Code, http.StatusNoContent)
	}

	cliLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	cliResponse := httptest.NewRecorder()
	handler.ServeHTTP(cliResponse, cliLogin)
	if cliResponse.Code != http.StatusNoContent {
		t.Fatalf("originless CLI login status = %d, want %d", cliResponse.Code, http.StatusNoContent)
	}
}

func TestCSRFAllowsOriginlessStripeWebhook(t *testing.T) {
	handler := CSRF([]string{"https://shop.example"}, "omniretail_session")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("webhook status = %d, want %d", res.Code, http.StatusNoContent)
	}
}
