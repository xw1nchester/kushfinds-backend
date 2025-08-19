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
	_ "github.com/xw1nchester/kushfinds-backend/docs"
	"github.com/xw1nchester/kushfinds-backend/internal/auth"
	authdb "github.com/xw1nchester/kushfinds-backend/internal/auth/db"
	authhandler "github.com/xw1nchester/kushfinds-backend/internal/auth/handler"
	"github.com/xw1nchester/kushfinds-backend/internal/auth/jwt"
	jwtmiddleware "github.com/xw1nchester/kushfinds-backend/internal/auth/jwt/middleware"
	"github.com/xw1nchester/kushfinds-backend/internal/auth/password"
	authservice "github.com/xw1nchester/kushfinds-backend/internal/auth/service"
	codedb "github.com/xw1nchester/kushfinds-backend/internal/code/db"
	codeservice "github.com/xw1nchester/kushfinds-backend/internal/code/service"
	"github.com/xw1nchester/kushfinds-backend/internal/config"
	industrydb "github.com/xw1nchester/kushfinds-backend/internal/industry/db"
	industryhandler "github.com/xw1nchester/kushfinds-backend/internal/industry/handler"
	industryservice "github.com/xw1nchester/kushfinds-backend/internal/industry/service"
	countrydb "github.com/xw1nchester/kushfinds-backend/internal/location/country/db"
	countryhandler "github.com/xw1nchester/kushfinds-backend/internal/location/country/handler"
	countryservice "github.com/xw1nchester/kushfinds-backend/internal/location/country/service"
	regiondb "github.com/xw1nchester/kushfinds-backend/internal/location/region/db"
	regionservice "github.com/xw1nchester/kushfinds-backend/internal/location/region/service"
	statedb "github.com/xw1nchester/kushfinds-backend/internal/location/state/db"
	statehandler "github.com/xw1nchester/kushfinds-backend/internal/location/state/handler"
	stateservice "github.com/xw1nchester/kushfinds-backend/internal/location/state/service"
	branddb "github.com/xw1nchester/kushfinds-backend/internal/market/brand/db"
	brandhandler "github.com/xw1nchester/kushfinds-backend/internal/market/brand/handler"
	brandservice "github.com/xw1nchester/kushfinds-backend/internal/market/brand/service"
	marketsectiondb "github.com/xw1nchester/kushfinds-backend/internal/market/section/db"
	marketsectionhandler "github.com/xw1nchester/kushfinds-backend/internal/market/section/handler"
	marketsectionservice "github.com/xw1nchester/kushfinds-backend/internal/market/section/service"
	storedb "github.com/xw1nchester/kushfinds-backend/internal/market/store/db"
	storehandler "github.com/xw1nchester/kushfinds-backend/internal/market/store/handler"
	storeservice "github.com/xw1nchester/kushfinds-backend/internal/market/store/service"
	uploadhandler "github.com/xw1nchester/kushfinds-backend/internal/upload/handler"
	uploadservice "github.com/xw1nchester/kushfinds-backend/internal/upload/service"
	userdb "github.com/xw1nchester/kushfinds-backend/internal/user/db"
	userhandler "github.com/xw1nchester/kushfinds-backend/internal/user/handler"
	userservice "github.com/xw1nchester/kushfinds-backend/internal/user/service"
	minioclient "github.com/xw1nchester/kushfinds-backend/pkg/client/minio"
	pgclient "github.com/xw1nchester/kushfinds-backend/pkg/client/postgresql"
	pgtx "github.com/xw1nchester/kushfinds-backend/pkg/transactor/postgresql"
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
		Endpoint:        cfg.Minio.Endpoint,
		AccessKeyID:     cfg.Minio.AccessKeyID,
		SecretAccessKey: cfg.Minio.SecretAccessKey,
		UseSSL:          cfg.Minio.UseSSL,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	router := chi.NewRouter()

	router.Use(
		LoggingMiddleware(log),
		cors.Handler(cors.Options{
			AllowedOrigins:   cfg.HTTPServer.AllowedOrigins,
			AllowCredentials: cfg.HTTPServer.AllowCredentials,
			AllowedMethods:   cfg.HTTPServer.AllowedMethods,
			AllowedHeaders:   cfg.HTTPServer.AllowedHeaders,
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

		authMiddleware := jwtmiddleware.NewMiddleware(log, tokenManager)

		industryRepository := industrydb.New(pgClient, log)

		industryService := industryservice.New(industryRepository, log)

		marketSectionRepository := marketsectiondb.New(pgClient, log)

		marketSectionService := marketsectionservice.New(marketSectionRepository, log)

		brandRepository := branddb.New(pgClient, log)

		brandService := brandservice.New(
			brandRepository,
			userService,
			countryService,
			stateService,
			marketSectionService,
			log,
		)

		storeRepository := storedb.New(pgClient, log)

		storeService := storeservice.New(
			storeRepository,
			userService,
			brandService,
			countryService,
			stateService,
			regionService,
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

		marketSectionHandler := marketsectionhandler.New(marketSectionService, log)

		log.Info("register market section handlers")

		marketSectionHandler.Register(r)

		brandHandler := brandhandler.New(
			brandService,
			authMiddleware,
			cfg.HTTPServer.StaticURL,
			log,
		)

		log.Info("register brand handlers")

		brandHandler.Register(r)

		storeHandler := storehandler.New(
			storeService,
			authMiddleware,
			cfg.HTTPServer.StaticURL,
			log,
		)

		log.Info("register store handlers")

		storeHandler.Register(r)
	})

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
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
