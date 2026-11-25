package cart

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/asthap08/ecommerce/internal/platform/httpx"

	"github.com/asthap08/ecommerce/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler() *Handler { return &Handler{} }

func NewHandlerWithService(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	c, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var in struct {
		VariantID string `json:"variant_id"`
		Quantity  int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Add(r.Context(), middleware.UserID(r.Context()), in.VariantID, in.Quantity)
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	variantID := chi.URLParam(r, "variant_id")
	var in struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Update(r.Context(), middleware.UserID(r.Context()), variantID, in.Quantity)
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	variantID := chi.URLParam(r, "variant_id")
	c, err := h.svc.Remove(r.Context(), middleware.UserID(r.Context()), variantID)
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Merge(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Merge(r.Context(), in.SessionID, middleware.UserID(r.Context()))
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		httpx.Error(w, http.StatusBadRequest, "choose an item and a quantity between 1 and 99")
	case errors.Is(err, ErrUnavailable):
		httpx.Error(w, http.StatusConflict, ErrUnavailable.Error())
	case errors.Is(err, ErrStock):
		httpx.Error(w, http.StatusConflict, ErrStock.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "the cart could not be updated")
	}
}
