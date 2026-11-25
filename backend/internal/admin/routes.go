package admin

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(authenticate, middleware.AdminOnly)
		r.Get("/", h.Index)
		r.Get("/dashboard", h.Dashboard)
		r.Get("/orders", h.Orders)
		r.Get("/products", h.Products)
		r.Get("/inventory", h.Inventory)
	})
}
