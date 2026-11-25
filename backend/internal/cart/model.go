package cart

type Cart struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
	Items     []CartItem `json:"items"`
}

type CartItem struct {
	ID         string `json:"id"`
	CartID     string `json:"cart_id"`
	VariantID  string `json:"variant_id"`
	Quantity   int    `json:"quantity"`
	Name       string `json:"name"`
	PriceMinor int64  `json:"price_minor"`
	ImageURL   string `json:"image_url,omitempty"`
	ProductID  string `json:"product_id"`
	SKU        string `json:"sku"`
	Available  int    `json:"available"`
}
