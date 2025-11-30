package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"gophermart/internal/config"
	"gophermart/internal/database"
	"gophermart/internal/handler"
	"gophermart/internal/mw"
	"gophermart/internal/service"
	"gophermart/internal/worker"
)

func main() {
	cfg := config.New()

	db, err := database.NewDB(cfg.DatabaseURI)
	if err != nil {
		slog.Error("failed to connect to DB", "error", err)
		os.Exit(1)
	}
	defer database.CloseDB(context.Background(), db)

	if err := database.InitSchema(db); err != nil {
		slog.Error("failed to init DB schema", "error", err)
		os.Exit(1)
	}

	// Services
	authSvc := service.NewAuthService(db)
	orderSvc := service.NewOrderService(db)
	balanceSvc := service.NewBalanceService(db)
	withdrawalSvc := service.NewWithdrawalService(db)
	accrualClient := service.NewAccrualClient(cfg.AccrualSystemAddress)

	// Worker
	accrualWorker := worker.NewAccrualWorker(orderSvc, accrualClient)

	// Router
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Post("/api/user/register", handler.RegisterHandler(authSvc, cfg.JWTSecret))
	r.Post("/api/user/login", handler.LoginHandler(authSvc, cfg.JWTSecret))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(mw.AuthMiddleware(cfg.JWTSecret))

		r.Post("/api/user/orders", handler.UploadOrderHandler(orderSvc))
		r.Get("/api/user/orders", handler.ListOrdersHandler(orderSvc))

		r.Get("/api/user/balance", handler.GetBalanceHandler(balanceSvc))
		r.Post("/api/user/balance/withdraw", handler.WithdrawHandler(withdrawalSvc))
		r.Get("/api/user/withdrawals", handler.ListWithdrawalsHandler(withdrawalSvc))
	})

	srv := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go accrualWorker.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	slog.Info("starting server", "addr", cfg.RunAddress)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
		}
	}()

	<-quit
	slog.Info("shutting down...")

	cancel() // stop worker
	ctxShut, cancelShut := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShut()

	if err := srv.Shutdown(ctxShut); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	slog.Info("server stopped")
}
