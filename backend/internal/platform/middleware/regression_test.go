package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

// Every new TCP connection has a new source port; the limiter must key on the IP only.
func TestRateLimiterIgnoresSourcePort(t *testing.T) {
	h := NewRateLimiter(3, time.Minute).Limit(ok)
	var last int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = "10.0.0." + "1:" + string(rune('0'+i)) + "5000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		last = rec.Code
		if last == 429 && rec.Header().Get("Retry-After") == "" {
			t.Fatal("429 without Retry-After seconds")
		}
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("limiter never engaged across ports, last status %d", last)
	}
}

func TestCacheControlOnlyForAnonymousCatalogue(t *testing.T) {
	h := CacheControl(60)(ok)
	cases := []struct {
		path, cookie string
		method       string
		public       bool
	}{
		{"/api/v1/products", "", "GET", true},
		{"/api/v1/products/abc/reviews", "", "GET", false},
		{"/api/v1/cart/", "", "GET", false},
		{"/api/v1/products", "omniretail_session", "GET", false},
		{"/api/v1/products", "", "POST", false},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		if c.cookie != "" {
			req.AddCookie(&http.Cookie{Name: c.cookie, Value: "x"})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		got := rec.Header().Get("Cache-Control")
		if c.public != (got == "public, max-age=60") {
			t.Errorf("%s %s cookie=%q: Cache-Control=%q", c.method, c.path, c.cookie, got)
		}
	}
}
