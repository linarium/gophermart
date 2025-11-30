package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"gophermart/internal/mw"
	"gophermart/internal/service"
)

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func WithdrawHandler(withdrawalSvc *service.WithdrawalService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := r.Context().Value(mw.UserCtxKey).(string)

		var req withdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Order == "" || req.Sum <= 0 {
			http.Error(w, "invalid order or sum", http.StatusUnprocessableEntity)
			return
		}

		if !validateLuhn(req.Order) {
			http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
			return
		}

		if err := withdrawalSvc.Create(r.Context(), userID, req.Order, req.Sum); err != nil {
			switch {
			case errors.Is(err, errors.New("insufficient funds")):
				http.Error(w, "insufficient funds", http.StatusPaymentRequired)
			default:
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func ListWithdrawalsHandler(withdrawalSvc *service.WithdrawalService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := r.Context().Value(mw.UserCtxKey).(string)

		withdrawals, err := withdrawalSvc.ListByUser(r.Context(), userID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if len(withdrawals) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
		}
	}
}
