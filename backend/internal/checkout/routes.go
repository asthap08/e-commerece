package checkout

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Get("/api/v1/checkout", h.Index)
	r.With(authenticate).Post("/api/v1/checkout", h.Checkout)
}
