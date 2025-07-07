package password

import (
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type manager struct {
	logger *zap.Logger
}

func New(logger *zap.Logger) *manager {
	return &manager{
		logger: logger,
	}
}

func (m *manager) GenerateHashFromPassword(password []byte) ([]byte, error) {
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		m.logger.Error("unexpected error when hashing password", zap.Error(err))
		return []byte{}, err
	}

	return passHash, nil
}

func (m *manager) CompareHashAndPassword(hashedPassword []byte, password []byte) error {
	err := bcrypt.CompareHashAndPassword(hashedPassword, password)
	if err != nil {
		m.logger.Error("unexpected error when compare passwords", zap.Error(err))
	}

	return err

}
