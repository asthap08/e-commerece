package orders

import (
	"context"
	"errors"
	"strings"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// Admin-driven fulfilment transitions.
var adminTransitions = map[string][]string{
	"pending":        {"processing", "cancelled"},
	"paid":           {"processing", "cancelled"},
	"processing":     {"shipped", "cancelled"},
	"shipped":        {"delivered"},
	"payment_failed": {"cancelled"},
}

// Customers may only cancel an order that has not been paid for or packed:
// COD/pending orders, or a failed card payment.
var customerTransitions = map[string][]string{
	"pending":        {"cancelled"},
	"payment_failed": {"cancelled"},
}

func (s *Service) Get(ctx context.Context, id string) (Order, error) { return s.repo.GetByID(ctx, id) }
func (s *Service) ListByUser(ctx context.Context, userID string, limit, offset int) ([]Order, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}
func (s *Service) ListAll(ctx context.Context, f ListFilter) ([]Order, int, error) {
	return s.repo.ListAll(ctx, f)
}
func (s *Service) Track(ctx context.Context, id, email string) (TrackedOrder, error) {
	return s.repo.Track(ctx, strings.TrimSpace(id), email)
}

// CancelOwn lets a customer cancel their own pending order.
func (s *Service) CancelOwn(ctx context.Context, id, userID string) error {
	return s.repo.SetStatus(ctx, id, "cancelled", userID, customerTransitions)
}

func (s *Service) UpdateFulfillmentStatus(ctx context.Context, id, status string) error {
	return s.repo.SetStatus(ctx, id, status, "", adminTransitions)
}

var ErrInvalidTracking = errors.New("tracking number is required (max 64 characters) and carrier max 64")

func (s *Service) UpdateTracking(ctx context.Context, orderID, number, carrier string) error {
	number, carrier = strings.TrimSpace(number), strings.TrimSpace(carrier)
	if number == "" || len(number) > 64 || len(carrier) > 64 {
		return ErrInvalidTracking
	}
	return s.repo.UpdateTracking(ctx, orderID, number, carrier)
}
