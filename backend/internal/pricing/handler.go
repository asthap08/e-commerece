package pricing

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandlerWithService(svc *Service) *Handler { return &Handler{svc: svc} }

func fail(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Error(w, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, ErrExists):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "coupon not found")
	default:
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var c Coupon
	if !httpx.Decode(w, r, &c) {
		return
	}
	c.Active = true
	created, err := h.svc.Create(r.Context(), c)
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) SetActive(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active bool `json:"active"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.svc.SetActive(r.Context(), chi.URLParam(r, "id"), in.Active); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Validate previews a discount; it returns only what a shopper needs, not the
// whole coupon record (usage counters etc.). `total` is in paise.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	total, _ := strconv.ParseInt(r.URL.Query().Get("total"), 10, 64)
	c, discount, err := h.svc.Validate(r.Context(), r.URL.Query().Get("code"), total)
	if err != nil {
		fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"code": c.Code, "discount_type": c.DiscountType, "value": c.Value, "discount_minor": discount})
}
