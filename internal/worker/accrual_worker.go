package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gophermart/internal/service"
)

type AccrualWorker struct {
	orderSvc    *service.OrderService
	accrualSvc  *service.AccrualClient
	interval    time.Duration
	batchSize   int
	stopChannel chan struct{}
}

func NewAccrualWorker(orderSvc *service.OrderService, accrualSvc *service.AccrualClient) *AccrualWorker {
	return &AccrualWorker{
		orderSvc:    orderSvc,
		accrualSvc:  accrualSvc,
		interval:    10 * time.Second,
		batchSize:   5,
		stopChannel: make(chan struct{}),
	}
}

func (w *AccrualWorker) Start(ctx context.Context) {
	slog.Info("starting accrual worker")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("accrual worker stopped")
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("batch processing failed", "error", err)
			}
		}
	}
}

func (w *AccrualWorker) processBatch(ctx context.Context) error {
	orders, err := w.orderSvc.GetUnprocessed(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("get unprocessed orders: %w", err)
	}

	for _, order := range orders {
		resp, err := w.accrualSvc.GetOrder(ctx, order.Number)
		if err != nil {
			if err.Error() == "rate limit exceeded" {
				slog.Warn("rate limited, skipping", "order", order.Number)
				continue
			}
			if err.Error() == "order not registered" {
				continue
			}
			slog.Error("failed to check accrual", "order", order.Number, "error", err)
			continue
		}

		status := resp.Status
		var accrual *float64
		if status == "PROCESSED" || status == "INVALID" {
			switch status {
			case "PROCESSED":
				if resp.Accrual > 0 {
					accrual = &resp.Accrual
					status = "PROCESSED"
				} else {
					status = "INVALID"
				}
			case "INVALID":
				status = "INVALID"
			case "REGISTERED", "PROCESSING":
				status = "PROCESSING"
			}
		} else if status == "REGISTERED" || status == "PROCESSING" {
			status = "PROCESSING"
		}

		if err := w.orderSvc.UpdateStatus(ctx, order.Number, status, accrual); err != nil {
			slog.Error("failed to update order status", "order", order.Number, "error", err)
		} else {
			slog.Info("order updated", "number", order.Number, "status", status, "accrual", accrual)
		}
	}

	return nil
}
