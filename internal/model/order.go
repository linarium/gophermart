package model

import (
	"time"
)

type Order struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Number     string    `json:"number"`
	Status     string    `json:"status"` // NEW, PROCESSING, INVALID, PROCESSED
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}
