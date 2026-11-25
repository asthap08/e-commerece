package audit

import "context"

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Log(ctx context.Context, actorID, action, entity, entityID string) {
	_ = s.repo.Log(ctx, actorID, action, entity, entityID)
}
