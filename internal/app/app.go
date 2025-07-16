package app

import (
	"context"
	"errors"
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
	codedb "github.com/vetrovegor/kushfinds-backend/internal/code/db"
	codeservice "github.com/vetrovegor/kushfinds-backend/internal/code/service"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	countrydb "github.com/vetrovegor/kushfinds-backend/internal/location/country/db"
	countryservice "github.com/vetrovegor/kushfinds-backend/internal/location/country/service"
	countryhandler "github.com/vetrovegor/kushfinds-backend/internal/location/country/handler"
	statedb "github.com/vetrovegor/kushfinds-backend/internal/location/state/db"
	stateservice "github.com/vetrovegor/kushfinds-backend/internal/location/state/service"
	regiondb "github.com/vetrovegor/kushfinds-backend/internal/location/region/db"
	regionservice "github.com/vetrovegor/kushfinds-backend/internal/location/region/service"
	userdb "github.com/vetrovegor/kushfinds-backend/internal/user/db"
	userhandler "github.com/vetrovegor/kushfinds-backend/internal/user/handler"
	userservice "github.com/vetrovegor/kushfinds-backend/internal/user/service"
	pgclient "github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	pgtx "github.com/vetrovegor/kushfinds-backend/pkg/transactor/postgresql"
	"go.uber.org/zap"

	"github.com/swaggo/http-swagger/v2"
	_ "github.com/vetrovegor/kushfinds-backend/docs"
)

type App struct {
	HTTPServer *http.Server
}

func New(log *zap.Logger, cfg config.Config) *App {
	// TODO: возможно стоит вынести в main.go, чтобы не создавать доп инстанс в тестах
	pgClient, err := pgclient.New(
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
			AllowedOrigins:   cfg.AllowedOrigins,
			AllowCredentials: cfg.AllowCredentials,
			AllowedMethods:   cfg.AllowedMethods,
			AllowedHeaders:   cfg.AllowedHeaders,
		}),
		middleware.Recoverer,
	)

	router.Get("/swagger/*", httpSwagger.Handler())

	router.Route("/api", func(r chi.Router) {
		r.Get("/ping", PingHandler)

		// TODO: рефакторить (посмотреть в сторону google/wire)
		authRepository := authdb.New(pgClient, log)

		userRepository := userdb.New(pgClient, log)

		countryRepository := countrydb.New(pgClient, log)

		countryService := countryservice.New(countryRepository, log)

		stateRepository := statedb.New(pgClient, log)

		stateService := stateservice.New(stateRepository, log)

		regionRepository := regiondb.New(pgClient, log)

		regionService := regionservice.New(regionRepository, log)

		userService := userservice.New(
			userRepository,
			countryService,
			stateService,
			regionService,
			log,
		)

		codeRepository := codedb.New(pgClient, log)

		codeService := codeservice.New(codeRepository, log)

		tokenManager := jwtauth.NewManager(cfg.JWT)

		mailManager := auth.NewMailManager(cfg.SMTP)

		passwordManager := password.New(log)

		txManager := pgtx.New(pgClient)

		authService := authservice.New(
			authRepository,
			userService,
			codeService,
			tokenManager,
			mailManager,
			passwordManager,
			txManager,
			log,
		)

		authMiddleware := jwtauth.NewMiddleware(log, tokenManager)

		authHandler := authhandler.New(authService, authMiddleware, log)

		log.Info("register auth handlers")

		authHandler.Register(r)

		userHandler := userhandler.New(userService, authMiddleware, log)

		log.Info("register user handlers")

		userHandler.Register(r)

		countryHandler := countryhandler.New(countryService, log)

		log.Info("register country handlers")

		countryHandler.Register(r)
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
	if err := a.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic("failed to start server")
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.HTTPServer.Shutdown(ctx)
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
