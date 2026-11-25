package pricing

import (
	"testing"
	"time"
)

func TestDiscountAndUsable(t *testing.T) {
	if d := Discount(Coupon{DiscountType: "percentage", Value: 10}, 49900); d != 4990 {
		t.Fatalf("percentage discount = %d", d)
	}
	if d := Discount(Coupon{DiscountType: "fixed", Value: 100000}, 49900); d != 49900 {
		t.Fatalf("fixed discount must be capped at subtotal, got %d", d)
	}
	past := time.Now().Add(-time.Hour)
	one := 1
	for name, c := range map[string]Coupon{
		"inactive": {Active: false}, "expired": {Active: true, ExpiresAt: &past},
		"used up": {Active: true, MaxUses: &one, UsedCount: 1}, "below minimum": {Active: true, MinOrderMinor: 1_000_000},
	} {
		if Usable(c, 5000, time.Now()) == "" {
			t.Errorf("%s coupon should be rejected", name)
		}
	}
	if Usable(Coupon{Active: true}, 5000, time.Now()) != "" {
		t.Error("plain active coupon should be usable")
	}
}
