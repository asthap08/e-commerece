package addresses

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

var ErrNotFound = errors.New("address not found")

type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

var pincodeRe = regexp.MustCompile(`^[1-9][0-9]{5}$`)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, a Address) (Address, error) {
	a.Label, a.Line1, a.Line2 = strings.TrimSpace(a.Label), strings.TrimSpace(a.Line1), strings.TrimSpace(a.Line2)
	a.City, a.State, a.Pincode = strings.TrimSpace(a.City), strings.TrimSpace(a.State), strings.TrimSpace(a.Pincode)
	if a.Label == "" {
		a.Label = "Home"
	}
	if a.Country == "" {
		a.Country = "IN"
	}
	switch {
	case a.Line1 == "" || a.City == "" || a.State == "":
		return a, &ValidationError{"street address, city and state are required"}
	case a.Country == "IN" && !pincodeRe.MatchString(a.Pincode):
		return a, &ValidationError{"enter a valid 6-digit PIN code"}
	case len(a.Line1) > 200 || len(a.Line2) > 200 || len(a.City) > 100 || len(a.State) > 100 || len(a.Label) > 30:
		return a, &ValidationError{"one of the fields is too long"}
	}
	return s.repo.Create(ctx, a)
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]Address, error) {
	return s.repo.ListByUser(ctx, userID)
}
func (s *Service) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}
func (s *Service) SetDefault(ctx context.Context, id, userID string) error {
	return s.repo.SetDefault(ctx, id, userID)
}
