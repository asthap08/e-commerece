package checkout

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckoutReturnsUnavailableWhenDisabled(t *testing.T) {
	handler := NewHandlerWithService(nil, false)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", nil)
	rec := httptest.NewRecorder()

	handler.Checkout(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
