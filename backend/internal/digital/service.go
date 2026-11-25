package digital

import "context"

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Grant(ctx context.Context, e Entitlement) (Entitlement, error) {
	return s.repo.Create(ctx, e)
}
