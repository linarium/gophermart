package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type BalanceService struct {
	db *sql.DB
}

func NewBalanceService(db *sql.DB) *BalanceService {
	return &BalanceService{db: db}
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func (s *BalanceService) Get(ctx context.Context, userID string) (*Balance, error) {
	var b Balance
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(current_balance, 0), COALESCE(withdrawn, 0) FROM users WHERE id = $1`,
		userID,
	).Scan(&b.Current, &b.Withdrawn)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return &b, nil
}
