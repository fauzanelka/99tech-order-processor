package models

import "time"

// Order represents a trading order from the input file
type Order struct {
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
} 