package reviews

import (
	"context"
	"strings"
	"unicode/utf8"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func (s *Service) Create(ctx context.Context, rev Review) (Review, error) {
	rev.Title, rev.Body = strings.TrimSpace(rev.Title), strings.TrimSpace(rev.Body)
	switch {
	case rev.Rating < 1 || rev.Rating > 5:
		return rev, &ValidationError{Message: "rating must be between 1 and 5"}
	case utf8.RuneCountInString(rev.Title) > 120:
		return rev, &ValidationError{Message: "title is too long (max 120 characters)"}
	case utf8.RuneCountInString(rev.Body) > 2000:
		return rev, &ValidationError{Message: "review is too long (max 2000 characters)"}
	}
	return s.repo.Create(ctx, rev)
}

func (s *Service) ListByProduct(ctx context.Context, productID string, limit, offset int) ([]Review, error) {
	return s.repo.ListByProduct(ctx, productID, limit, offset)
}

func (s *Service) GetAverageRating(ctx context.Context, productID string) (float64, int, error) {
	return s.repo.GetAverageRating(ctx, productID)
}
