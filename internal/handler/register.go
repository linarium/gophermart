package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gophermart/internal/service"
)

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func RegisterHandler(authSvc *service.AuthService, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Login == "" || req.Password == "" {
			http.Error(w, "login and password required", http.StatusBadRequest)
			return
		}

		user, err := authSvc.Register(r.Context(), req.Login, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, errors.New("login already exists")):
				http.Error(w, "login already exists", http.StatusConflict)
			default:
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": user.ID,
			"exp":     jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		})

		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			http.Error(w, "token generation failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Authorization", "Bearer "+tokenString)
		w.WriteHeader(http.StatusOK)
	}
}
