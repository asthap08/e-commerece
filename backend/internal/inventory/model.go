package inventory

type Stock struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
	Reserved  int    `json:"reserved"`
}
