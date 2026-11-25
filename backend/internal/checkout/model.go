package checkout

type Request struct {
	IdempotencyKey  string `json:"-"`
	UserID          string `json:"user_id"`
	PaymentProvider string `json:"payment_provider"` // "stripe" or "manual"
	PaymentMethod   string `json:"payment_method"`   // "card", "cod", "upi", "wallet", "netbanking"
	Currency        string `json:"currency"`         // defaults to "INR"
	ShippingMethod  string `json:"shipping_method"`  // "standard", "express", "same_day"
	ShippingAddress string `json:"shipping_address"` // JSON string of address
	CustomerName    string `json:"customer_name"`
	CustomerEmail   string `json:"customer_email"`
	CustomerPhone   string `json:"customer_phone"`
	CouponCode      string `json:"coupon_code"`
}

type Response struct {
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	TotalMinor    int64  `json:"total_minor"`
	SubtotalMinor int64  `json:"subtotal_minor"`
	DiscountMinor int64  `json:"discount_minor"`
	ShippingMinor int64  `json:"shipping_minor"`
	Currency      string `json:"currency"`
	ClientSecret  string `json:"client_secret,omitempty"`
	PaymentID     string `json:"payment_id,omitempty"`
}
