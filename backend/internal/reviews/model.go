package reviews

import "time"

type Review struct {
	ID         string    `json:"id"`
	ProductID  string    `json:"product_id"`
	UserID     string    `json:"-"` // never exposed publicly
	AuthorName string    `json:"author"`
	Verified   bool      `json:"verified_purchase"`
	Rating     int       `json:"rating"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}
