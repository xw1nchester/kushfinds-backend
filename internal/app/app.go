package app

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	authdb "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	authhandler "github.com/vetrovegor/kushfinds-backend/internal/auth/handler"
	"github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/auth/password"
	authservice "github.com/vetrovegor/kushfinds-backend/internal/auth/service"
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

type App struct {
	HTTPServer *http.Server
}

func NewApp(log *zap.Logger, cfg config.Config) *App {
	pgClient, err := pgclient.NewClient(
		context.TODO(),
		pgclient.Config{
			Username: cfg.PostgreSQL.Username,
			Password: cfg.PostgreSQL.Password,
			Host:     cfg.PostgreSQL.Host,
			Port:     cfg.PostgreSQL.Port,
			Database: cfg.PostgreSQL.Database,
		},
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	router := chi.NewRouter()

	router.Use(
		LoggingMiddleware(log),
		cors.Handler(cors.Options{
			AllowedOrigins: cfg.AllowedOrigins,
			AllowCredentials: true,
		}),
		middleware.Recoverer,
	)

	router.Get("/swagger/*", httpSwagger.Handler())

	router.Route("/api", func(r chi.Router) {
		r.Get("/ping", PingHandler)

		// TODO: рефакторить (посмотреть в сторону google/wire)
		authRepository := authdb.NewRepository(pgClient, log)

		userRepository := userdb.NewRepository(pgClient, log)

		userService := user.NewService(userRepository, log)

		codeRepository := codedb.NewRepository(pgClient, log)

		codeService := code.NewService(codeRepository, log)

		tokenManager := jwtauth.NewTokenManager(cfg.JWT)

		mailManager := auth.NewMailManager(cfg.SMTP)

		passwordManager := password.New(log)

		txManager := pgtx.NewPgManager(pgClient)

		authService := authservice.NewService(
			authRepository,
			userService,
			codeService,
			tokenManager,
			mailManager,
			passwordManager,
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

	return &App{
		HTTPServer: srv,
	}
}

func (a *App) MustRun() {
	if err := a.HTTPServer.ListenAndServe(); err != nil {
		panic("failed to start server")
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

// @Tags		other
// @Success	200		{string}	string
// @Failure	400,500	{object}	apperror.AppError
// @Router		/ping [get]
func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}
