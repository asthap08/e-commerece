package pricing

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

var codeRe = regexp.MustCompile(`^[A-Z0-9_-]{3,32}$`)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, c Coupon) (Coupon, error) {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	switch {
	case !codeRe.MatchString(c.Code):
		return c, &ValidationError{"code must be 3-32 letters, numbers, - or _"}
	case c.DiscountType != "percentage" && c.DiscountType != "fixed":
		return c, &ValidationError{"discount_type must be percentage or fixed"}
	case c.Value <= 0 || (c.DiscountType == "percentage" && c.Value > 100):
		return c, &ValidationError{"percentage must be 1-100; fixed amounts must be positive (paise)"}
	case c.MinOrderMinor < 0 || (c.MaxUses != nil && *c.MaxUses <= 0):
		return c, &ValidationError{"minimum order and usage limit must be positive"}
	case c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()):
		return c, &ValidationError{"expiry must be in the future"}
	}
	return s.repo.Create(ctx, c)
}

func (s *Service) List(ctx context.Context) ([]Coupon, error) { return s.repo.List(ctx) }
func (s *Service) SetActive(ctx context.Context, id string, a bool) error {
	return s.repo.SetActive(ctx, id, a)
}

// Validate returns the coupon and the discount it gives on subtotalMinor.
func (s *Service) Validate(ctx context.Context, code string, subtotalMinor int64) (Coupon, int64, error) {
	c, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c, 0, &ValidationError{"that coupon code is not valid"}
		}
		return c, 0, err
	}
	if why := Usable(c, subtotalMinor, time.Now()); why != "" {
		return c, 0, &ValidationError{why}
	}
	return c, Discount(c, subtotalMinor), nil
}
