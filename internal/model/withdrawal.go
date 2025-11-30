package model

import "time"

type Withdrawal struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	OrderNumber string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
