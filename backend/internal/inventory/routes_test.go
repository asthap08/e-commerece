package inventory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/asthap08/ecommerce/internal/platform/session"
	"github.com/go-chi/chi/v5"
)

type inventorySessionResolver struct{}

func (inventorySessionResolver) Resolve(_ context.Context, token string, _ time.Time) (session.Identity, error) {
	switch token {
	case "customer-session":
		return session.Identity{UserID: "user-1", Role: "customer"}, nil
	case "admin-session":
		return session.Identity{UserID: "admin-1", Role: "admin"}, nil
	default:
		return session.Identity{}, session.ErrInvalidSession
	}
}

func TestInventoryGetRequiresAdmin(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, NewHandler(), middleware.Auth(inventorySessionResolver{}))

	tests := []struct {
		name       string
		session    string
		wantStatus int
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized},
		{name: "customer", session: "customer-session", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/?variant_id=variant-1", nil)
			if test.session != "" {
				request.AddCookie(&http.Cookie{Name: session.CookieName, Value: test.session})
			}

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("GET inventory status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}
