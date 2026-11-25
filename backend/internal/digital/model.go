package digital

type Entitlement struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	OrderID     string `json:"order_id"`
	VariantID   string `json:"variant_id"`
	DownloadURL string `json:"download_url"`
}
