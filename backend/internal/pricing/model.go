package pricing

import "time"

type Coupon struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	DiscountType  string     `json:"discount_type"` // percentage | fixed (fixed is in paise)
	Value         int64      `json:"value"`
	Active        bool       `json:"active"`
	MinOrderMinor int64      `json:"min_order_minor"`
	MaxUses       *int       `json:"max_uses,omitempty"`
	UsedCount     int        `json:"used_count"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// Discount computes the discount in minor units for a subtotal (never more than the subtotal).
func Discount(c Coupon, subtotal int64) int64 {
	var d int64
	switch c.DiscountType {
	case "percentage":
		d = subtotal * c.Value / 100
	case "fixed":
		d = c.Value
	}
	if d > subtotal {
		d = subtotal
	}
	if d < 0 {
		d = 0
	}
	return d
}

// Usable reports why a coupon cannot be applied ("" when it can).
func Usable(c Coupon, subtotal int64, now time.Time) string {
	switch {
	case !c.Active:
		return "this coupon is not active"
	case c.ExpiresAt != nil && now.After(*c.ExpiresAt):
		return "this coupon has expired"
	case c.MaxUses != nil && c.UsedCount >= *c.MaxUses:
		return "this coupon has reached its usage limit"
	case subtotal < c.MinOrderMinor:
		return "your order does not reach the minimum amount for this coupon"
	}
	return ""
}
