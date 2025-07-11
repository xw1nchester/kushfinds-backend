package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"time"

	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	authservice "github.com/vetrovegor/kushfinds-backend/internal/auth/service"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	userdb "github.com/vetrovegor/kushfinds-backend/internal/user/db"
)

const (
	JSONContentType = "application/json"
)

func (s *APITestSuite) TestRegister() {
	require := s.Require()
	ctx := context.Background()

	registerEmailURL := fmt.Sprintf("%s/auth/register/email", s.baseUrl)

	email := "test@mail.ru"

	// регистрация
	emailPayload := fmt.Sprintf(`{"email":"%s"}`, email)
	response, err := http.Post(registerEmailURL, JSONContentType, bytes.NewBufferString(emailPayload))
	require.NoError(err)

	require.Equal(http.StatusOK, response.StatusCode)

	// проверка что пользователь появился в бд
	var createdUser userdb.User
	s.dbClient.QueryRow(ctx, "SELECT id, email, is_verified FROM users WHERE email=$1", email).
		Scan(&createdUser.ID, &createdUser.Email, &createdUser.IsVerified)

	require.NotNil(createdUser)
	require.Equal(email, createdUser.Email)
	require.False(createdUser.IsVerified)

	// попытка регистрации с существующим email
	response, err = http.Post(registerEmailURL, "application/json", bytes.NewBufferString(emailPayload))
	require.NoError(err)

	appErr, err := decodeResponseBody[apperror.AppError](response)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, response.StatusCode)
	require.Equal(authservice.ErrEmailAlreadyExists.Error(), appErr.Message)

	// проверка что код появился в бд
	var createdCodeValue string
	s.dbClient.QueryRow(
		ctx,
		"SELECT code FROM codes WHERE type='verification' AND user_id=$1 AND retry_date>NOW()",
		createdUser.ID,
	).Scan(&createdCodeValue)

	require.NotEmpty(createdCodeValue)
}

func (s *APITestSuite) TestVerifyResend() {
	require := s.Require()
	ctx := context.Background()
	resendVerifyURL := fmt.Sprintf("%s/auth/verify/resend", s.baseUrl)

	email := "user1@mail.ru"
	emailPayload := fmt.Sprintf(`{"email":"%s"}`, email)

	// повторная отправка кода верификации
	response, err := http.Post(resendVerifyURL, JSONContentType, bytes.NewBufferString(emailPayload))
	require.NoError(err)
	require.Equal(http.StatusOK, response.StatusCode)

	// проверка что код появился в бд
	var createdCodeID int
	var createdCodeValue string
	s.dbClient.QueryRow(
		ctx,
		"SELECT codes.id, codes.code FROM codes JOIN users ON users.id = codes.user_id WHERE users.email = $1;",
		email,
	).Scan(&createdCodeID, &createdCodeValue)

	require.NotEmpty(createdCodeID)
	require.NotEmpty(createdCodeValue)

	// попытка отправить код повторно, когда это еще сделать нельзя
	response, err = http.Post(resendVerifyURL, JSONContentType, bytes.NewBufferString(emailPayload))
	require.NoError(err)

	appErr, err := decodeResponseBody[apperror.AppError](response)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, response.StatusCode)
	require.Equal(authservice.ErrCodeAlreadySent.Error(), appErr.Message)

	// установка retry_date < текущей даты
	_, err = s.dbClient.Exec(
		ctx, 
		"UPDATE codes SET retry_date=$1 WHERE id=$2", 
		time.Now().Add(-time.Second), 
		createdCodeID,
	)
	require.NoError(err)

	// попытка отправить код повторно, когда это сделать можно
	response, err = http.Post(resendVerifyURL, JSONContentType, bytes.NewBufferString(emailPayload))
	require.NoError(err)

	require.Equal(http.StatusOK, response.StatusCode)

	// получение нового кода
	s.dbClient.QueryRow(
		ctx,
		"SELECT codes.code FROM codes JOIN users ON users.id = codes.user_id WHERE users.email = $1;",
		email,
	).Scan(&createdCodeValue)

	require.NotEmpty(createdCodeValue)
}

func (s *APITestSuite) TestVerify() {
	require := s.Require()
	ctx := context.Background()
	registerVerifyURL := fmt.Sprintf("%s/auth/register/verify", s.baseUrl)

	email := "user1@mail.ru"
	validCode := "067125"

	// отправка некорректного кода подтверждения
	response, err := http.Post(
		registerVerifyURL, 
		JSONContentType, 
		bytes.NewBufferString(fmt.Sprintf(`{"email":"%s","code":"%s"}`, email, "invalid")),
	)
	require.NoError(err)

	appErr, err := decodeResponseBody[apperror.AppError](response)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, response.StatusCode)
	require.Equal(authservice.ErrInvalidCode.Error(), appErr.Message)

	// вставка кода подтверждения
	_, err = s.dbClient.Exec(
		ctx, 
		"INSERT INTO codes (code, type, user_id, retry_date, expiry_date) VALUES ($1, 'verification', (SELECT id FROM users WHERE email = $2), $3, $4)",
		validCode,
		email,
		time.Now().Add(time.Minute),
		time.Now().Add(5 * time.Minute),
	)
	require.NoError(err)

	// отправка корректного кода подтверждения
	response, err = http.Post(
		registerVerifyURL, 
		JSONContentType, 
		bytes.NewBufferString(fmt.Sprintf(`{"email":"%s","code":"%s"}`, email, validCode)),
	)
	require.NoError(err)

	authResponse, err := decodeResponseBody[auth.AuthResponse](response)
	require.NoError(err)
	require.Equal(http.StatusOK, response.StatusCode)
	require.NotEmpty(authResponse.AccessToken)
	require.Equal(email, authResponse.User.Email)
	require.Nil(authResponse.User.Username)
	require.Nil(authResponse.User.FirstName)
	require.Nil(authResponse.User.LastName)
	require.Nil(authResponse.User.Avatar)
	require.True(authResponse.User.IsVerified)
}

func (s *APITestSuite) TestSaveProfile() {
	require := s.Require()
	ctx := context.Background()
	saveProfileURL := fmt.Sprintf("%s/auth/register/profile", s.baseUrl)

	busy := auth.ProfileRequest{
		Username: "username",
		FirstName: "John",
		LastName: "Doe",
	}

	var buf bytes.Buffer
	require.NoError(json.NewEncoder(&buf).Encode(busy))

	client := &http.Client{}

	// без токена
	req, err := http.NewRequest(http.MethodPatch, saveProfileURL, &buf)
	require.NoError(err)

	req.Header.Add("Content-Type", JSONContentType)

	response, err := client.Do(req)
	require.NoError(err)
	require.Equal(http.StatusUnauthorized, response.StatusCode)

	// с токеном и занятым username
	var userID int
	err = s.dbClient.QueryRow(
		ctx,
		"SELECT id from users WHERE email=$1",
		"user1@mail.ru",
	).Scan(&userID)
	require.NoError(err)
	require.NotEmpty(userID)

	accessToken, err := s.tokenManager.GenerateToken(userID)
	require.NoError(err)
	
	require.NoError(json.NewEncoder(&buf).Encode(busy))

	req, err = http.NewRequest(http.MethodPatch, saveProfileURL, &buf)
	require.NoError(err)

	req.Header.Add("Content-Type", JSONContentType)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	response, err = client.Do(req)
	require.NoError(err)

	appErr, err := decodeResponseBody[apperror.AppError](response)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, response.StatusCode)
	require.Equal(authservice.ErrUsernameAlreadyExists.Error(), appErr.Message)

	// с токеном + валидным username
	valid := auth.ProfileRequest{
		Username: "username1",
		FirstName: "John",
		LastName: "Doe",
	}

	require.NoError(json.NewEncoder(&buf).Encode(valid))
	req, err = http.NewRequest(http.MethodPatch, saveProfileURL, &buf)
	require.NoError(err)

	req.Header.Add("Content-Type", JSONContentType)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	response, err = client.Do(req)
	require.NoError(err)

	user, err := decodeResponseBody[user.UserResponse](response)
	require.NoError(err)
	require.Equal(http.StatusOK, response.StatusCode)
	require.Equal(userID, user.User.ID)
	require.Equal(valid.Username, *user.User.Username)
	require.Equal(valid.FirstName, *user.User.FirstName)
	require.Equal(valid.LastName, *user.User.LastName)

	// повторная отправка данных профиля
	require.NoError(json.NewEncoder(&buf).Encode(valid))
	req, err = http.NewRequest(http.MethodPatch, saveProfileURL, &buf)
	require.NoError(err)

	req.Header.Add("Content-Type", JSONContentType)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	response, err = client.Do(req)
	require.NoError(err)

	appErr, err = decodeResponseBody[apperror.AppError](response)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, response.StatusCode)
	require.Equal(authservice.ErrNicknameAlreadySet.Error(), appErr.Message)
}