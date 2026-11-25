package returns

import (
	"context"
	"net/url"
	"strings"
)

type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Request(ctx context.Context, userID, orderID string, itemIDs []string, reason string) (ReturnRequest, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 5 || len(reason) > 1000 {
		return ReturnRequest{}, &ValidationError{"please describe the reason (5-1000 characters)"}
	}
	return s.repo.CreateForCustomer(ctx, userID, orderID, itemIDs, reason)
}

func (s *Service) Get(ctx context.Context, id string) (ReturnRequest, error) {
	return s.repo.Get(ctx, id)
}
func (s *Service) ListByUser(ctx context.Context, userID string) ([]ReturnRequest, error) {
	return s.repo.ListByUser(ctx, userID)
}
func (s *Service) ListByOrder(ctx context.Context, orderID string) ([]ReturnRequest, error) {
	return s.repo.ListByOrder(ctx, orderID)
}
func (s *Service) ListAll(ctx context.Context, status string) ([]ReturnRequest, error) {
	return s.repo.ListAll(ctx, status)
}

func (s *Service) SetStatus(ctx context.Context, id, to, note, labelURL, authCode string) error {
	switch to {
	case "approved", "received", "refunded", "rejected":
	default:
		return &ValidationError{"invalid return status"}
	}
	if labelURL != "" {
		u, err := url.Parse(labelURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
			return &ValidationError{"label URL must be an http(s) link"}
		}
	}
	if len(note) > 1000 || len(authCode) > 64 {
		return &ValidationError{"note or authorisation code is too long"}
	}
	return s.repo.SetStatus(ctx, id, to, strings.TrimSpace(note), labelURL, strings.TrimSpace(authCode))
}
