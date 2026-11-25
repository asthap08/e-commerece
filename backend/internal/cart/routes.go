package cart

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/cart", func(r chi.Router) {
		r.Use(authenticate)
		r.Get("/", h.Get)
		r.Post("/items", h.Add)
		r.Put("/items/{variant_id}", h.Update)
		r.Delete("/items/{variant_id}", h.Remove)
		r.Post("/merge", h.Merge)
	})
}
