package shipping

type Shipment struct {
	ID             string `json:"id"`
	OrderID        string `json:"order_id"`
	Carrier        string `json:"carrier"`
	TrackingNumber string `json:"tracking_number"`
	Status         string `json:"status"`
}
