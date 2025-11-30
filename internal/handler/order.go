package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"gophermart/internal/mw"
	"gophermart/internal/service"
)

func UploadOrderHandler(orderSvc *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := r.Context().Value(mw.UserCtxKey).(string)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		number, err := readOrderNumber(r)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if !validateLuhn(number) {
			http.Error(w, "invalid order number (failed Luhn check)", http.StatusUnprocessableEntity)
			return
		}

		err = orderSvc.Create(r.Context(), userID, number)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrOrderAlreadyExistsByUser):
				w.WriteHeader(http.StatusOK) // ← 200 — как в ТЗ
				return
			case errors.Is(err, service.ErrOrderAlreadyExistsByOther):
				http.Error(w, "order already uploaded by another user", http.StatusConflict) // ← 409
				return
			default:
				slog.Error("order create failed", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func readOrderNumber(r *http.Request) (string, error) {
	maxBody := http.MaxBytesReader(nil, r.Body, 1024)
	body, err := io.ReadAll(maxBody)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return "", errors.New("request body too large")
		}
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	number := strings.TrimSpace(string(body))
	if number == "" {
		return "", errors.New("empty order number")
	}

	if _, err := strconv.ParseUint(number, 10, 64); err != nil {
		return "", errors.New("order number must contain only digits")
	}

	return number, nil
}

func validateLuhn(s string) bool {
	if len(s) < 2 {
		return false
	}
	var sum int
	double := false
	for i := len(s) - 1; i >= 0; i-- {
		digit := int(s[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}

func ListOrdersHandler(orderSvc *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := r.Context().Value(mw.UserCtxKey).(string)

		orders, err := orderSvc.ListByUser(r.Context(), userID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(orders); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
		}
	}
}
