package models

import "time"

// Item represents a product available for purchase.
type Item struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

// LineItem is a line item in an order.
type LineItem struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
	Price    int `json:"price"` // Price at time of adding
}

// Order represents a completed purchase.
type Order struct {
	ID     int        `json:"id"`
	UserID int        `json:"user_id"`
	Items  []LineItem `json:"items"`
	Total  int        `json:"total"`
	Status string     `json:"status"` // pending, paid, failed
}

type Cart struct {
	ID     string     `json:"id"`
	UserID int        `json:"user_id"`
	Items  []LineItem `json:"items"`
}
