package tests

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	userdb "github.com/vetrovegor/kushfinds-backend/internal/user/db"
	_ "go.uber.org/zap"
)

func (s *APITestSuite) TestAuth() {
	ctx := context.Background()
	contentType := "application/json"
	email := "test@mail.ru"
	verificationCodeType := "verification"

	// регистрация
	registerEmailURL := fmt.Sprintf("%s/auth/register/email", s.baseUrl)
	emailBody := bytes.NewBufferString(fmt.Sprintf(`{"email":"%s"}`, email))
	response, err := http.Post(registerEmailURL, contentType, emailBody)
	s.NoError(err)

	s.Equal(http.StatusOK, response.StatusCode)

	// проверка что пользователь появился в бд
	var createdUser userdb.User
	s.dbClient.QueryRow(ctx, "SELECT id, email FROM users WHERE email=$1", email).
		Scan(&createdUser.ID, &createdUser.Email)

	s.NotNil(createdUser)
	s.Equal(email, createdUser.Email)

	// попытка регистрации с существующим email
	response, err = http.Post(registerEmailURL, "application/json", emailBody)
	s.NoError(err)

	s.Equal(http.StatusBadRequest, response.StatusCode)

	// проверка что код появился в бд
	var createdCodeID int
	var createdCodeValue string
	s.dbClient.QueryRow(
		ctx,
		"SELECT id, code FROM codes WHERE type=$1 AND user_id=$2 AND retry_date>NOW()",
		verificationCodeType,
		createdUser.ID,
	).Scan(&createdCodeID, &createdCodeValue)

	s.NotEmpty(createdCodeValue)

	// попытка отправить код повторно, когда это еще сделать нельзя
	resendVerifyURL := fmt.Sprintf("%s/auth/verify/resend", s.baseUrl)
	response, err = http.Post(resendVerifyURL, contentType, emailBody)
	s.NoError(err)

	s.Equal(http.StatusBadRequest, response.StatusCode)

	// установка retry_date < текущей даты
	_, err = s.dbClient.Exec(
		ctx, 
		"UPDATE codes SET retry_date=$1 WHERE id=$2", 
		time.Now().Add(-time.Hour), 
		createdCodeID,
	)
	s.NoError(err)

	// попытка отправить код повторно, когда это сделать можно
	// response, err = http.Post(resendVerifyURL, contentType, emailBody)
	// s.NoError(err)

	// s.Equal(http.StatusOK, response.StatusCode)
}
