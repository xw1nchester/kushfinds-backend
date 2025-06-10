package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	db "github.com/vetrovegor/kushfinds-backend/internal/user/db/postgresql"
	"github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad()

	log, _ := zap.NewDevelopment()
	defer log.Sync()

	pgClient, err := postgresql.NewClient(context.TODO(), cfg.PostgreSQL)
	if err != nil {
		log.Fatal(err.Error())
	}

	router := chi.NewRouter()

	router.Use(LoggingMiddleware(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.SetHeader("Content-Type", "application/json"))

	router.Route("/api", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})

		authRepository := db.NewRepository(pgClient, log)

		authService := auth.NewService(authRepository, cfg.JWT, log)

		authHandler := auth.NewHandler(authService, log)

		log.Info("register auth handlers")

		authHandler.Register(r)
	})

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	log.Info("starting server", zap.String("addr", cfg.Address))

	if err := srv.ListenAndServe(); err != nil {
		log.Error("Failed to start server")
	}
}

func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			logger.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote", r.RemoteAddr),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}
