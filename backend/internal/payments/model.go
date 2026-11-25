package payments

type Payment struct {
	ID          string `json:"id"`
	OrderID     string `json:"order_id"`
	Provider    string `json:"provider"`
	ProviderRef string `json:"provider_ref"`
	Status      string `json:"status"`
	AmountMinor int64  `json:"amount_minor"`
	Method      string `json:"method"` // "card", "cod", "upi", "wallet", "netbanking"
}
