package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	authdb "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	authhandler "github.com/vetrovegor/kushfinds-backend/internal/auth/handler"
	"github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	codedb "github.com/vetrovegor/kushfinds-backend/internal/code/db"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	userdb "github.com/vetrovegor/kushfinds-backend/internal/user/db"
	pgclient "github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	pgtx "github.com/vetrovegor/kushfinds-backend/pkg/transactor/postgresql"
	"go.uber.org/zap"

	"github.com/swaggo/http-swagger/v2"
	_ "github.com/vetrovegor/kushfinds-backend/docs"
)

//	@title			Kushfinds API
//	@version		1.0
//	@description	API Server for Kushfinds application

//	@host		localhost:8080
//	@BasePath	/api

// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
func main() {
	cfg := config.MustLoad()

	log, _ := zap.NewDevelopment()
	defer log.Sync()

	pgClient, err := pgclient.NewClient(context.TODO(), cfg.PostgreSQL)
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

		// TODO: рефакторить
		authRepository := authdb.NewRepository(pgClient, log)

		userRepository := userdb.NewRepository(pgClient, log)

		userService := user.NewService(userRepository, log)

		codeRepository := codedb.NewRepository(pgClient, log)

		codeService := code.NewService(codeRepository, log)

		tokenManager := jwtauth.NewTokenManager(cfg.JWT)

		mailManager := auth.NewMailManager(cfg.SMTP)

		txManager := pgtx.NewPgManager(pgClient)

		authService := auth.NewService(
			authRepository,
			userService,
			codeService,
			tokenManager,
			mailManager,
			txManager,
			log,
		)

		authMiddleware := jwtauth.NewAuthMiddleware(log, tokenManager)

		authHandler := authhandler.NewHandler(authService, authMiddleware, log)

		log.Info("register auth handlers")

		authHandler.Register(r)

		userHandler := user.NewHandler(userService, authMiddleware, log)

		log.Info("register user handlers")

		userHandler.Register(r)
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
