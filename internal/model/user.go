package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Login        string    `json:"login"`
	PasswordHash []byte    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
