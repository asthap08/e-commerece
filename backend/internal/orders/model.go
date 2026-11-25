package orders

import (
	"encoding/json"
	"time"
)

type Order struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	Status          string          `json:"status"`
	Total           int64           `json:"total_minor"`
	Currency        string          `json:"currency"`
	Items           []OrderItem     `json:"items"`
	ItemCount       int             `json:"item_count,omitempty"`
	Created         time.Time       `json:"created_at"`
	PaymentMethod   string          `json:"payment_method"`
	PaymentStatus   string          `json:"payment_status,omitempty"`
	ShippingMethod  string          `json:"shipping_method"`
	ShippingCost    int64           `json:"shipping_cost_minor"`
	Discount        int64           `json:"discount_minor"`
	CouponCode      string          `json:"coupon_code,omitempty"`
	ShippingAddress json.RawMessage `json:"shipping_address"`
	CustomerName    string          `json:"customer_name"`
	CustomerEmail   string          `json:"customer_email"`
	CustomerPhone   string          `json:"customer_phone"`
	TrackingNumber  string          `json:"tracking_number"`
	TrackingCarrier string          `json:"tracking_carrier"`
}

type OrderItem struct {
	ID         string `json:"id"`
	OrderID    string `json:"order_id"`
	VariantID  string `json:"variant_id"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	PriceMinor int64  `json:"price_minor"`
	Quantity   int    `json:"quantity"`
	Image      string `json:"image,omitempty"`
}

// TrackedOrder is the limited view returned to guests who know the order id
// and the e-mail it was placed with. No address or phone number is exposed.
type TrackedOrder struct {
	ID              string      `json:"id"`
	Status          string      `json:"status"`
	Total           int64       `json:"total_minor"`
	Created         time.Time   `json:"created_at"`
	PaymentMethod   string      `json:"payment_method"`
	ShippingMethod  string      `json:"shipping_method"`
	TrackingNumber  string      `json:"tracking_number"`
	TrackingCarrier string      `json:"tracking_carrier"`
	Items           []OrderItem `json:"items"`
}

type ListFilter struct {
	Status string
	Query  string
	Limit  int
	Offset int
}
