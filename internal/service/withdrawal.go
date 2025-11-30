package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gophermart/internal/model"
)

type WithdrawalService struct {
	db *sql.DB
}

func NewWithdrawalService(db *sql.DB) *WithdrawalService {
	return &WithdrawalService{db: db}
}

func (s *WithdrawalService) Create(ctx context.Context, userID, orderNumber string, sum float64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var current float64
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(current_balance, 0) FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&current)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	if current < sum {
		return errors.New("insufficient funds")
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum, processed_at) VALUES ($1, $2, $3, $4)`,
		userID, orderNumber, sum, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert withdrawal: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET current_balance = current_balance - $1, withdrawn = withdrawn + $1 WHERE id = $2`,
		sum, userID,
	)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	return tx.Commit()
}

func (s *WithdrawalService) ListByUser(ctx context.Context, userID string) ([]model.Withdrawal, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return withdrawals, nil
}
