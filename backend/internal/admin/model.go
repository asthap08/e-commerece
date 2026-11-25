package admin

type Dashboard struct {
	OrdersTotal   int64 `json:"orders_total"`
	OrdersPending int64 `json:"orders_pending"`
	ProductsLive  int64 `json:"products_live"`
	LowStockItems int64 `json:"low_stock_items"`
	RevenueMinor  int64 `json:"revenue_minor"`
}

type InventoryItem struct {
	VariantID  string `json:"variant_id"`
	ProductID  string `json:"product_id"`
	Product    string `json:"product"`
	SKU        string `json:"sku"`
	PriceMinor int64  `json:"price_minor"`
	Quantity   int    `json:"quantity"`
	Reserved   int    `json:"reserved"`
}
