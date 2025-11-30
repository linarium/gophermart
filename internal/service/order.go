package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gophermart/internal/model"
)

var (
	ErrOrderAlreadyExistsByUser  = errors.New("order already uploaded by this user")
	ErrOrderAlreadyExistsByOther = errors.New("order already uploaded by another user")
)

type OrderService struct {
	db *sql.DB
}

func NewOrderService(db *sql.DB) *OrderService {
	return &OrderService{db: db}
}

func (s *OrderService) Create(ctx context.Context, userID, number string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existingUserID string
	err = tx.QueryRowContext(ctx, `SELECT user_id FROM orders WHERE number = $1`, number).Scan(&existingUserID)
	if err == nil {
		if existingUserID == userID {
			return ErrOrderAlreadyExistsByUser
		}
		return ErrOrderAlreadyExistsByOther
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check order: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO orders (user_id, number, status, uploaded_at) VALUES ($1, $2, $3, $4)`,
		userID, number, "NEW", time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (s *OrderService) ListByUser(ctx context.Context, userID string) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		var accrual sql.NullFloat64
		if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &accrual, &o.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		if accrual.Valid {
			o.Accrual = accrual.Float64
		}
		orders = append(orders, o)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return orders, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, number, status string, accrual *float64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var query string
	if accrual != nil {
		query = `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`
		_, err = tx.ExecContext(ctx, query, status, *accrual, number)
	} else {
		query = `UPDATE orders SET status = $1 WHERE number = $2`
		_, err = tx.ExecContext(ctx, query, status, number)
	}
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if status == "PROCESSED" && accrual != nil {
		var userID string
		err = tx.QueryRowContext(ctx, `SELECT user_id FROM orders WHERE number = $1`, number).Scan(&userID)
		if err != nil {
			return fmt.Errorf("get user_id: %w", err)
		}

		_, err = tx.ExecContext(ctx, `UPDATE users SET current_balance = COALESCE(current_balance, 0) + $1 WHERE id = $2`, *accrual, userID)
		if err != nil {
			return fmt.Errorf("update balance: %w", err)
		}
	}

	return tx.Commit()
}

func (s *OrderService) GetUnprocessed(ctx context.Context, limit int) ([]model.Order, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, number, status, accrual, uploaded_at
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query unprocessed: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		var accrual sql.NullFloat64
		if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &accrual, &o.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		if accrual.Valid {
			o.Accrual = accrual.Float64
		}
		orders = append(orders, o)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return orders, nil
}

type Order struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func (o Order) MarshalJSON() ([]byte, error) {
	type Alias Order
	return json.Marshal(&struct {
		UploadedAt string `json:"uploaded_at"`
		*Alias
	}{
		UploadedAt: o.UploadedAt.Format(time.RFC3339),
		Alias:      (*Alias)(&o),
	})
}
