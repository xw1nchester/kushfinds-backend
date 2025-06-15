package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	authDB "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	codeDb "github.com/vetrovegor/kushfinds-backend/internal/code/db"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	userDB "github.com/vetrovegor/kushfinds-backend/internal/user/db"
	"github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	"go.uber.org/zap"

	_ "github.com/vetrovegor/kushfinds-backend/docs"
	"github.com/swaggo/http-swagger/v2"
)

//	@title			Kushfinds API
//	@version		1.0
//	@description	API Server for Kushfinds application

//	@host		localhost:8080
//	@BasePath	/api

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
func main() {
	cfg := config.MustLoad()

	log, _ := zap.NewDevelopment()
	defer log.Sync()

	pgClient, err := postgresql.NewClient(context.TODO(), cfg.PostgreSQL)
	if err != nil {
		log.Fatal(err.Error())
	}

	router := chi.NewRouter()

	router.Use(
		LoggingMiddleware(log),
		middleware.Recoverer,
	)

	router.Get("/swagger/*", httpSwagger.Handler())

	router.Route("/api", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})

		codeRepository := codeDb.NewRepository(pgClient, log)

		codeService := code.NewService(codeRepository, log)

		userRepository := userDB.NewRepository(pgClient, log)

		authRepository := authDB.NewRepository(pgClient, log)

		tokenManager := auth.NewTokenManager(cfg.JWT)

		mailManager := auth.NewMailManager(cfg.SMTP)

		authService := auth.NewService(userRepository, authRepository, codeService, tokenManager, mailManager, log)

		authMiddleware := auth.NewAuthMiddleware(log, cfg.JWT.Secret)

		authHandler := auth.NewHandler(authService, authMiddleware, log)

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
		log.Error("failed to start server")
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
				// zap.String("status", r.Response.Status),
				zap.String("remote", r.RemoteAddr),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}
