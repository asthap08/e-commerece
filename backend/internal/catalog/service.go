package catalog

import (
	"context"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/asthap08/ecommerce/internal/platform/validation"
)

// ValidationError is a safe, user-facing message.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f ListFilter) ([]Product, int, error) {
	return s.repo.List(ctx, f)
}
func (s *Service) Get(ctx context.Context, id string) (Product, error) {
	return s.repo.Get(ctx, id, true)
}
func (s *Service) GetAny(ctx context.Context, id string) (Product, error) {
	return s.repo.Get(ctx, id, false)
}
func (s *Service) Categories(ctx context.Context) ([]Category, error) { return s.repo.Categories(ctx) }

func validImageURL(raw string) bool {
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && len(raw) <= 2000
}

func normCategory(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	if c == "" {
		return "general"
	}
	return c
}

func (s *Service) Create(ctx context.Context, p Product) (Product, error) {
	p.Name = validation.SanitizeString(p.Name)
	if p.Name == "" || utf8.RuneCountInString(p.Name) > 200 {
		return p, &ValidationError{"product name is required (max 200 characters)"}
	}
	if p.Slug == "" {
		p.Slug = Slugify(p.Name)
	}
	if err := validation.ValidateSlug(p.Slug); err != nil {
		return p, &ValidationError{"slug may only contain lowercase letters, numbers and hyphens"}
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	if p.Status != "draft" {
		return p, &ValidationError{"new products start as draft; add a variant and stock, then publish"}
	}
	if !validImageURL(p.ImageURL) {
		return p, &ValidationError{"image URL must be a valid http(s) address"}
	}
	p.Category = normCategory(p.Category)
	return s.repo.Create(ctx, p)
}

func (s *Service) Update(ctx context.Context, id string, u Update) error {
	if u.Name != nil {
		n := validation.SanitizeString(*u.Name)
		if n == "" || utf8.RuneCountInString(n) > 200 {
			return &ValidationError{"product name is required (max 200 characters)"}
		}
		u.Name = &n
	}
	if u.Category != nil {
		c := normCategory(*u.Category)
		u.Category = &c
	}
	if u.ImageURL != nil && !validImageURL(*u.ImageURL) {
		return &ValidationError{"image URL must be a valid http(s) address"}
	}
	return s.repo.UpdateProduct(ctx, id, u)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status string) error {
	switch status {
	case "draft", "active", "archived":
	default:
		return &ValidationError{"invalid product status"}
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *Service) ListVariants(ctx context.Context, productID string) ([]Variant, error) {
	return s.repo.ListVariants(ctx, productID)
}

func (s *Service) CreateVariant(ctx context.Context, v Variant) (Variant, error) {
	v.SKU = strings.TrimSpace(v.SKU)
	if v.SKU == "" || len(v.SKU) > 64 {
		return v, &ValidationError{"SKU is required (max 64 characters)"}
	}
	if v.PriceMinor <= 0 || v.PriceMinor > 1_000_000_000_00 {
		return v, &ValidationError{"price must be greater than zero"}
	}
	if v.CompareAtMinor != nil && *v.CompareAtMinor < v.PriceMinor {
		return v, &ValidationError{"compare-at price must not be lower than the price"}
	}
	if v.Currency == "" {
		v.Currency = "INR"
	}
	if v.Currency != "INR" {
		return v, &ValidationError{"only INR is supported"}
	}
	return s.repo.CreateVariant(ctx, v)
}

func (s *Service) UpdateVariant(ctx context.Context, id string, name *string, price, compareAt *int64, clearCompare bool) error {
	if price != nil && *price <= 0 {
		return &ValidationError{"price must be greater than zero"}
	}
	if price != nil && compareAt != nil && *compareAt < *price {
		return &ValidationError{"compare-at price must not be lower than the price"}
	}
	return s.repo.UpdateVariant(ctx, id, name, price, compareAt, clearCompare)
}

// Slugify turns a product name into a URL slug.
func Slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
