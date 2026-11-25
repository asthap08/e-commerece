package returns

import "time"

type Item struct {
	ID         string `json:"id"` // order_items.id
	Name       string `json:"name"`
	Quantity   int    `json:"quantity"`
	PriceMinor int64  `json:"price_minor"`
}

type ReturnRequest struct {
	ID                  string     `json:"id"`
	OrderID             string     `json:"order_id"`
	UserID              string     `json:"user_id,omitempty"`
	Status              string     `json:"status"` // requested, approved, received, refunded, rejected
	Reason              string     `json:"reason"`
	RefundAmountMinor   int64      `json:"refund_amount_minor"`
	RefundMethod        string     `json:"refund_method"`
	ReturnItems         []Item     `json:"return_items"`
	ReturnLabelURL      string     `json:"return_label_url"`
	ReturnAuthorization string     `json:"return_authorization"`
	AdminNote           string     `json:"admin_note"`
	ProcessedAt         *time.Time `json:"processed_at"`
	CreatedAt           time.Time  `json:"created_at"`
	CustomerEmail       string     `json:"customer_email,omitempty"`
}
