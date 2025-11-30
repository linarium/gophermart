package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"gophermart/internal/model"
)

type AuthService struct {
	db *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Register(ctx context.Context, login, password string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	query := `INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id, login, created_at`
	row := s.db.QueryRowContext(ctx, query, login, hash)

	var user model.User
	if err := row.Scan(&user.ID, &user.Login, &user.CreatedAt); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, errors.New("login already exists")
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	user.PasswordHash = hash

	return &user, nil
}

func (s *AuthService) Authenticate(ctx context.Context, login, password string) (*model.User, error) {
	query := `SELECT id, login, password_hash, created_at FROM users WHERE login = $1`
	row := s.db.QueryRowContext(ctx, query, login)

	var user model.User
	if err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid login or password")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		return nil, errors.New("invalid login or password")
	}

	return &user, nil
}

func GenerateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
