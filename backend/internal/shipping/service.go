package shipping

import "context"

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, sh Shipment) (Shipment, error) {
	if sh.Status == "" {
		sh.Status = "pending"
	}
	return s.repo.Create(ctx, sh)
}
