package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/swaggo/http-swagger/v2"
	_ "github.com/vetrovegor/kushfinds-backend/docs"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	authdb "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	marketsectiondb "github.com/vetrovegor/kushfinds-backend/internal/market/section/db"
	marketsectionservice "github.com/vetrovegor/kushfinds-backend/internal/market/section/service"
	marketsectionhandler "github.com/vetrovegor/kushfinds-backend/internal/market/section/handler"
	branddb "github.com/vetrovegor/kushfinds-backend/internal/market/brand/db"
	brandservice "github.com/vetrovegor/kushfinds-backend/internal/market/brand/service"
	brandhandler "github.com/vetrovegor/kushfinds-backend/internal/market/brand/handler"
	authhandler "github.com/vetrovegor/kushfinds-backend/internal/auth/handler"
	"github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/auth/password"
	authservice "github.com/vetrovegor/kushfinds-backend/internal/auth/service"
	codedb "github.com/vetrovegor/kushfinds-backend/internal/code/db"
	codeservice "github.com/vetrovegor/kushfinds-backend/internal/code/service"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	countrydb "github.com/vetrovegor/kushfinds-backend/internal/location/country/db"
	industrydb "github.com/vetrovegor/kushfinds-backend/internal/industry/db"
	countryhandler "github.com/vetrovegor/kushfinds-backend/internal/location/country/handler"
	countryservice "github.com/vetrovegor/kushfinds-backend/internal/location/country/service"
	regiondb "github.com/vetrovegor/kushfinds-backend/internal/location/region/db"
	regionservice "github.com/vetrovegor/kushfinds-backend/internal/location/region/service"
	statedb "github.com/vetrovegor/kushfinds-backend/internal/location/state/db"
	statehandler "github.com/vetrovegor/kushfinds-backend/internal/location/state/handler"
	industryhandler "github.com/vetrovegor/kushfinds-backend/internal/industry/handler"
	stateservice "github.com/vetrovegor/kushfinds-backend/internal/location/state/service"
	userdb "github.com/vetrovegor/kushfinds-backend/internal/user/db"
	userhandler "github.com/vetrovegor/kushfinds-backend/internal/user/handler"
	uploadhandler "github.com/vetrovegor/kushfinds-backend/internal/upload/handler"
	uploadservice "github.com/vetrovegor/kushfinds-backend/internal/upload/service"
	industryservice "github.com/vetrovegor/kushfinds-backend/internal/industry/service"
	userservice "github.com/vetrovegor/kushfinds-backend/internal/user/service"
	minioclient "github.com/vetrovegor/kushfinds-backend/pkg/client/minio"
	pgclient "github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	pgtx "github.com/vetrovegor/kushfinds-backend/pkg/transactor/postgresql"
	"go.uber.org/zap"
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

	minioClient, err := minioclient.New(minioclient.Config{
		Endpoint: cfg.Minio.Endpoint,
		AccessKeyID: cfg.Minio.AccessKeyID,
		SecretAccessKey: cfg.Minio.SecretAccessKey,
		UseSSL: cfg.Minio.UseSSL,
	})
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

		regionRepository := regiondb.New(pgClient, log)

		regionService := regionservice.New(regionRepository, log)

		stateRepository := statedb.New(pgClient, log)

		stateService := stateservice.New(stateRepository, regionService, log)

		countryRepository := countrydb.New(pgClient, log)

		countryService := countryservice.New(countryRepository, stateService, log)

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

		industryRepository := industrydb.New(pgClient, log)

		industryService := industryservice.New(industryRepository, log)

		brandRepository := branddb.New(pgClient, log)

		brandService := brandservice.New(
			brandRepository,
			countryService,
			stateService,
			log,
		)

		authHandler := authhandler.New(authService, authMiddleware, log)

		log.Info("register auth handlers")

		authHandler.Register(r)

		userHandler := userhandler.New(userService, authMiddleware, log)

		log.Info("register user handlers")

		userHandler.Register(r)

		countryHandler := countryhandler.New(countryService, log)

		log.Info("register country handlers")

		countryHandler.Register(r)

		stateHandler := statehandler.New(stateService, log)

		log.Info("register state handlers")

		stateHandler.Register(r)

		industryHandler := industryhandler.New(industryService, log)

		log.Info("register industry handlers")

		industryHandler.Register(r)

		uploadService := uploadservice.New(minioClient, log)

		uploadHandler := uploadhandler.New(uploadService, authMiddleware, log)

		log.Info("register upload handlers")

		uploadHandler.Register(r)

		marketSectionRepository := marketsectiondb.New(pgClient, log)

		marketSectionService := marketsectionservice.New(marketSectionRepository, log)

		marketSectionHandler := marketsectionhandler.New(marketSectionService, log)

		log.Info("register market section handlers")

		marketSectionHandler.Register(r)

		brandHandler := brandhandler.New(
			brandService, 
			authMiddleware, 
			log,
		)

		log.Info("register brand handlers")

		brandHandler.Register(r)
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
