package catalog

import "time"

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ImageURL    string    `json:"image_url,omitempty"`
	Images      []string  `json:"images,omitempty"`
	Category    string    `json:"category"`
	Rating      float64   `json:"rating"`
	ReviewCount int       `json:"review_count"`
	Variants    []Variant `json:"variants,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Variant struct {
	ID             string `json:"id"`
	ProductID      string `json:"product_id"`
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	PriceMinor     int64  `json:"price_minor"`
	CompareAtMinor *int64 `json:"compare_at_minor,omitempty"`
	Currency       string `json:"currency"`
	InStock        bool   `json:"in_stock"`
	Available      int    `json:"available"`
}

// ListFilter describes a catalogue query. Prices are in minor units (paise).
type ListFilter struct {
	Query              string
	Category           string
	MinPrice, MaxPrice int64
	Sort               string // newest | price_asc | price_desc | rating
	Limit, Offset      int
}

type Category struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
