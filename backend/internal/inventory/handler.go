package inventory

import (
	"encoding/json"
	"net/http"

	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/asthap08/ecommerce/internal/platform/middleware"
)

type Handler struct{ svc *Service }

func NewHandler() *Handler { return &Handler{} }

func NewHandlerWithService(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	variantID := r.URL.Query().Get("variant_id")
	s, err := h.svc.Get(r.Context(), variantID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "inventory record not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s)
}

func (h *Handler) Adjust(w http.ResponseWriter, r *http.Request) {
	var in struct {
		VariantID string `json:"variant_id"`
		Delta     int    `json:"delta"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.VariantID == "" || in.Delta == 0 || in.Reason == "" {
		httpx.Error(w, http.StatusBadRequest, "variant_id, non-zero delta, and reason are required")
		return
	}
	if err := h.svc.AdjustAs(r.Context(), middleware.UserID(r.Context()), in.VariantID, in.Delta, in.Reason); err != nil {
		httpx.Error(w, http.StatusConflict, "adjustment failed: stock cannot go below reserved/zero, or the variant does not exist")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
