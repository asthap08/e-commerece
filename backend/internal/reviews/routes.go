package reviews

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/products/{productId}/reviews", func(r chi.Router) {
		r.Get("/", h.List)
		r.With(authenticate).Post("/", h.Create)
	})
}
