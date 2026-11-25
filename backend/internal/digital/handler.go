package digital

import (
	"encoding/json"
	"net/http"
)

type Handler struct{ svc *Service }

func NewHandler() *Handler { return &Handler{} }

func NewHandlerWithService(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Grant(w http.ResponseWriter, r *http.Request) {
	var e Entitlement
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	created, err := h.svc.Grant(r.Context(), e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}
