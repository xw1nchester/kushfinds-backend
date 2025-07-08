package tests

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"github.com/vetrovegor/kushfinds-backend/internal/app"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	pgclient "github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	"go.uber.org/zap"
)

type APITestSuite struct {
	suite.Suite
	cfg *config.Config
	dbClient *pgxpool.Pool
	logger   *zap.Logger
	baseUrl  string
	app      *app.App
}

func TestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, &APITestSuite{})
}

func (s *APITestSuite) SetupSuite() {
	cfg := config.MustLoadByPath("../config/test.yml")

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

	s.Require().NoError(err)

	log, _ := zap.NewDevelopment()
	defer log.Sync()

	app := app.NewApp(log, *cfg)

	s.cfg = cfg
	s.dbClient = pgClient
	s.logger = log
	s.baseUrl = fmt.Sprintf("http://localhost%s/api", cfg.HTTPServer.Address)
	s.app = app

	go func() {
		app.MustRun()
	}()

	log.Info("server started", zap.String("addr", cfg.Address))

	time.Sleep(500 * time.Millisecond)
}

func (s *APITestSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.app.Shutdown(ctx)
	s.Require().NoError(err)
}

func (s *APITestSuite) SetupTest() {
	s.applyMigrations(true)
}

func (s *APITestSuite) TearDownTest() {
	s.applyMigrations(false)
}

func (s *APITestSuite) applyMigrations(isUp bool) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		s.cfg.PostgreSQL.Username,
		s.cfg.PostgreSQL.Password,
		s.cfg.PostgreSQL.Host,
		s.cfg.PostgreSQL.Port,
		s.cfg.PostgreSQL.Database,
	)
	migrationsPath := "../migrations"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	s.Require().NoError(err)

	var migrationErr error

	if isUp {
		migrationErr = m.Up()
	} else {
		migrationErr = m.Down()
	}

	s.Require().NoError(migrationErr)
}