package notifications

import (
	"encoding/json"
	"net/http"
)

type Handler struct{ svc *Service }

func NewHandler() *Handler { return &Handler{} }

func NewHandlerWithService(svc *Service, log interface{}) *Handler { return &Handler{svc: svc} }

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	var n Notification
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h.svc.Send(n)
	w.WriteHeader(http.StatusAccepted)
}
