package inventory

import (
	"context"
	"fmt"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Get(ctx context.Context, variantID string) (Stock, error) {
	return s.repo.Get(ctx, variantID)
}
func (s *Service) Adjust(ctx context.Context, variantID string, delta int, reason string) error {
	return s.AdjustAs(ctx, "", variantID, delta, reason)
}

func (s *Service) AdjustAs(ctx context.Context, actorID, variantID string, delta int, reason string) error {
	if variantID == "" || reason == "" {
		return fmt.Errorf("variant ID and adjustment reason are required")
	}
	return s.repo.AdjustAs(ctx, actorID, variantID, delta, reason)
}
func (s *Service) Deduct(ctx context.Context, variantID string, qty int) error {
	if qty <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	return s.repo.Deduct(ctx, variantID, qty)
}
