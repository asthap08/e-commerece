package cart

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalid     = errors.New("invalid cart request")
	ErrUnavailable = errors.New("this item is no longer available")
	ErrStock       = errors.New("not enough stock for that quantity")
)

const maxQty = 99

func (s *Service) check(ctx context.Context, cartID, variantID string, want int, replace bool, inCartQty *int) error {
	live, avail, inCart, err := s.repo.Availability(ctx, cartID, variantID)
	if err != nil {
		return err
	}
	if !live {
		return ErrUnavailable
	}
	total := want
	if !replace {
		total += inCart
	}
	if total > maxQty || total > avail {
		return ErrStock
	}
	return nil
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Add(ctx context.Context, userID, variantID string, qty int) (Cart, error) {
	if userID == "" || variantID == "" || qty <= 0 {
		return Cart{}, fmt.Errorf("%w: variant and a positive quantity are required", ErrInvalid)
	}
	cartID, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	if err := s.check(ctx, cartID, variantID, qty, false, nil); err != nil {
		return Cart{}, err
	}
	if err := s.repo.AddItem(ctx, cartID, variantID, qty); err != nil {
		return Cart{}, err
	}
	items, err := s.repo.ListItems(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	return Cart{ID: cartID, UserID: userID, Items: items}, nil
}

func (s *Service) Get(ctx context.Context, userID string) (Cart, error) {
	if userID == "" {
		return Cart{}, fmt.Errorf("user ID is required")
	}
	cartID, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	items, err := s.repo.ListItems(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	c := Cart{ID: cartID, UserID: userID, Items: items}
	return c, nil
}

// Update modifies the quantity of a cart item.
func (s *Service) Update(ctx context.Context, userID, variantID string, qty int) (Cart, error) {
	if userID == "" || variantID == "" || qty <= 0 {
		return Cart{}, fmt.Errorf("%w: variant and a positive quantity are required", ErrInvalid)
	}
	cartID, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	if err := s.check(ctx, cartID, variantID, qty, true, nil); err != nil {
		return Cart{}, err
	}
	if err := s.repo.UpdateItem(ctx, cartID, variantID, qty); err != nil {
		return Cart{}, err
	}
	items, err := s.repo.ListItems(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	return Cart{ID: cartID, UserID: userID, Items: items}, nil
}

// Remove deletes a cart item.
func (s *Service) Remove(ctx context.Context, userID, variantID string) (Cart, error) {
	if userID == "" || variantID == "" {
		return Cart{}, fmt.Errorf("user ID and variant ID are required")
	}
	cartID, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	if err := s.repo.RemoveItem(ctx, cartID, variantID); err != nil {
		return Cart{}, err
	}
	items, err := s.repo.ListItems(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	return Cart{ID: cartID, UserID: userID, Items: items}, nil
}

// Clear removes all items from the user's cart.
func (s *Service) Clear(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	cartID, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.ClearCart(ctx, cartID); err != nil {
		return err
	}
	return nil
}

// Merge transfers items from an anonymous session cart into an authenticated user's cart.
func (s *Service) Merge(ctx context.Context, sessionID, userID string) (Cart, error) {
	if userID == "" {
		return Cart{}, fmt.Errorf("user ID is required")
	}
	if sessionID == "" {
		return s.Get(ctx, userID)
	}
	if err := s.repo.MergeSessionCart(ctx, sessionID, userID); err != nil {
		return Cart{}, err
	}
	return s.Get(ctx, userID)
}
