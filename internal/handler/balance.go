package handler

import (
	"encoding/json"
	"net/http"

	"gophermart/internal/mw"
	"gophermart/internal/service"
)

func GetBalanceHandler(balanceSvc *service.BalanceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := r.Context().Value(mw.UserCtxKey).(string)

		balance, err := balanceSvc.Get(r.Context(), userID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(balance); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
		}
	}
}
