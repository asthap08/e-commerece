package catalog

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not found")
	case errors.Is(err, ErrConflict):
		httpx.Error(w, http.StatusConflict, "a product or SKU with that slug/SKU already exists")
	case errors.Is(err, ErrBadParent):
		httpx.Error(w, http.StatusNotFound, "product not found")
	case errors.Is(err, ErrNotLive):
		httpx.Error(w, http.StatusConflict, "add at least one variant before publishing")
	default:
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

// rupeesToMinor accepts a whole-rupee value from the query string.
func minor(v string) int64 {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n * 100
}

// ListProducts supports q, category, min_price/max_price (in rupees), sort, limit, offset.
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, offset := httpx.Page(r, 20, 100)
	items, total, err := h.svc.List(r.Context(), ListFilter{
		Query: q.Get("q"), Category: q.Get("category"), Sort: q.Get("sort"),
		MinPrice: minor(q.Get("min_price")), MaxPrice: minor(q.Get("max_price")),
		Limit: limit, Offset: offset,
	})
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items, "meta": map[string]int{"limit": limit, "offset": offset, "total": total}})
}

// SearchProducts is kept for API compatibility: it is List with a required q.
func (h *Handler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("q") == "" {
		httpx.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	h.ListProducts(w, r)
}

func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Categories(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) AdminGetProduct(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.GetAny(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p Product
	if !httpx.Decode(w, r, &p) {
		return
	}
	created, err := h.svc.Create(r.Context(), p)
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Category    *string `json:"category"`
		ImageURL    *string `json:"image_url"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), Update{Name: in.Name, Description: in.Description, Category: in.Category, ImageURL: in.ImageURL}); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.svc.UpdateStatus(r.Context(), chi.URLParam(r, "id"), in.Status); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListVariants(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListVariants(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) CreateVariant(w http.ResponseWriter, r *http.Request) {
	var v Variant
	if !httpx.Decode(w, r, &v) {
		return
	}
	v.ProductID = chi.URLParam(r, "id")
	created, err := h.svc.CreateVariant(r.Context(), v)
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func (h *Handler) UpdateVariant(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name           *string `json:"name"`
		PriceMinor     *int64  `json:"price_minor"`
		CompareAtMinor *int64  `json:"compare_at_minor"`
		ClearCompareAt bool    `json:"clear_compare_at"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.svc.UpdateVariant(r.Context(), chi.URLParam(r, "variantId"), in.Name, in.PriceMinor, in.CompareAtMinor, in.ClearCompareAt); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
