package reviews

import (
	"errors"
	"net/http"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create takes the author from the session. The old handler decoded user_id
// from the request body, so reviews could be posted as anyone (or failed with a
// uuid error when it was empty - which is what the UI sent).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Rating int    `json:"rating"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	rev, err := h.svc.Create(r.Context(), Review{ProductID: chi.URLParam(r, "productId"), UserID: middleware.UserID(r.Context()), Rating: in.Rating, Title: in.Title, Body: in.Body})
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusBadRequest, ve.Message)
	case errors.Is(err, ErrDuplicate):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrProductNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "your review could not be saved")
	default:
		httpx.JSON(w, http.StatusCreated, rev)
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	limit, offset := httpx.Page(r, 20, 50)
	items, err := h.svc.ListByProduct(r.Context(), productID, limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "reviews could not be loaded")
		return
	}
	avg, count, _ := h.svc.GetAverageRating(r.Context(), productID)
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items, "meta": map[string]any{"average_rating": avg, "review_count": count}})
}
