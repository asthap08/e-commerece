package catalog

import (
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, authenticate middleware.Authenticator) {
	r.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", h.ListProducts)
		r.Get("/search", h.SearchProducts)
		r.Get("/categories", h.Categories)
		r.Get("/{id}", h.GetProduct)
		r.Get("/{id}/variants", h.ListVariants)
		r.Group(func(r chi.Router) {
			r.Use(authenticate, middleware.AdminOnly)
			r.Post("/", h.CreateProduct)
			r.Get("/{id}/admin", h.AdminGetProduct)
			r.Patch("/{id}", h.UpdateProduct)
			r.Patch("/{id}/status", h.UpdateProductStatus)
			r.Post("/{id}/variants", h.CreateVariant)
			r.Patch("/{id}/variants/{variantId}", h.UpdateVariant)
		})
	})
}
