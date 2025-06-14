package code

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/vetrovegor/kushfinds-backend/internal/code/db"
	"go.uber.org/zap"
)

const (
	VerifyCodeType           = "verification"
	RecoveryPasswordCodeType = "recovery_password"
	ChangePasswordCodeType   = "change_password"
)

var (
	ErrInternal        = errors.New("internal error when working with confirmation codes")
	ErrCodeAlreadySent = errors.New("code has already been sent")
	ErrCodeNotFound    = errors.New("code not found")
)

type Service interface {
	GenerateVerify(ctx context.Context, userID int) (string, error)
	GenerateRecoveryPassword(ctx context.Context, userID int) (string, error)
	GenerateChangePassword(ctx context.Context, userID int) (string, error)
	ValidateVerify(ctx context.Context, code string, userID int) error
}

type service struct {
	repository db.Repository
	logger     *zap.Logger
}

func NewService(repository db.Repository, logger *zap.Logger) Service {
	return &service{
		repository: repository,
		logger:     logger,
	}
}

func (s service) generateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		s.logger.Error("error when confirmation code", zap.Error(err))
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s service) generate(ctx context.Context, codeType string, userID int) (string, error) {
	if err := s.repository.CheckRecentlyCodeExists(ctx, codeType, userID); err != nil {
		if errors.Is(err, db.ErrCodeAlreadySent) {
			return "", ErrCodeAlreadySent
		}

		s.logger.Info("error when check recently code exists", zap.Error(err))

		return "", ErrInternal
	}

	code, err := s.generateVerificationCode()
	if err != nil {
		return "", ErrInternal
	}

	if err := s.repository.Create(ctx, code, VerifyCodeType, userID, time.Now().Add(1*time.Minute), time.Now().Add(5*time.Minute)); err != nil {
		s.logger.Info("error when code creation", zap.Error(err))

		return "", ErrInternal
	}

	return code, nil
}

func (s service) GenerateVerify(ctx context.Context, userID int) (string, error) {
	return s.generate(ctx, VerifyCodeType, userID)
}

func (s service) GenerateRecoveryPassword(ctx context.Context, userID int) (string, error) {
	return s.generate(ctx, RecoveryPasswordCodeType, userID)
}

func (s service) GenerateChangePassword(ctx context.Context, userID int) (string, error) {
	return s.generate(ctx, ChangePasswordCodeType, userID)
}

func (s service) validate(ctx context.Context, code string, codeType string, userID int) error {
	err := s.repository.CheckNotExpiryCodeExists(ctx, code, codeType, userID)
	if err != nil {
		if errors.Is(err, db.ErrCodeNotFound) {
			return ErrCodeNotFound
		}

		s.logger.Info("error when check not expiry code exists", zap.Error(err))

		return err
	}

	return nil
}

func (s service) ValidateVerify(ctx context.Context, code string, userID int) error {
	return s.validate(ctx, code, VerifyCodeType, userID)
}
