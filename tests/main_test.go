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
	jwtauth "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
	pgclient "github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	"go.uber.org/zap"
)

type TokenManager interface {
	GenerateToken(userID int) (string, error)
}

type APITestSuite struct {
	suite.Suite
	cfg *config.Config
	dbClient *pgxpool.Pool
	logger   *zap.SugaredLogger
	baseUrl  string
	app      *app.App
	tokenManager TokenManager
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
	s.logger = log.Sugar()
	s.baseUrl = fmt.Sprintf("http://localhost%s/api", cfg.HTTPServer.Address)
	s.app = app
	s.tokenManager = jwtauth.NewTokenManager(cfg.JWT)

	go func() {
		app.MustRun()
	}()

	err = s.applyMigrations(true)
	s.Require().NoError(err)

	time.Sleep(500 * time.Millisecond)
}

func (s *APITestSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.Require().NoError(s.app.Shutdown(ctx))

	s.Require().NoError(s.applyMigrations(false))
}

func (s *APITestSuite) SetupTest() {
	s.Require().NoError(s.populateDb())
}

func (s *APITestSuite) TearDownTest() {
	s.Require().NoError(s.cleanupDb())
}

func (s *APITestSuite) applyMigrations(isUp bool) error {
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
		return err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	if err != nil {
		return err
	}

	if isUp {
		return m.Up()
	}

	return m.Down()
}

func (s *APITestSuite) populateDb() error {
	query := `
		INSERT INTO users (email, username, first_name, last_name, password_hash, is_verified) 
		VALUES 
		('user1@mail.ru', NULL, NULL, NULL, NULL, false),
		('user2@mail.ru', NULL, NULL, NULL, NULL, true),
		('user3@mail.ru', 'username', 'John', 'Doe', NULL, true);
	`

	_, err := s.dbClient.Exec(context.Background(), query)
	
	return err
}

func (s *APITestSuite) cleanupDb() error {
	query := `
		DELETE FROM users;
	`

	_, err := s.dbClient.Exec(context.Background(), query)
	
	return err
}