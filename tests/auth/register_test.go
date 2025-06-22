package tests

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/gavv/httpexpect/v2"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
)

func TestRegisterEmail(t *testing.T) {
	validEmail := gofakeit.Email()

	testCases := []struct {
		name               string
		email              string
		expectedStatusCode int
	}{
		{
			name:               "Valid email",
			email:              validEmail,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Invalid email",
			email:              "Invalid",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Without email",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Empty email",
			email: "",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Duplicated email",
			email:              validEmail,
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := url.URL{
				Scheme: "http",
				Host:   "localhost:8080",
			}

			e := httpexpect.Default(t, u.String())

			e.POST("/api/auth/register/email").
				WithJSON(auth.EmailRequest{Email: tc.email}).
				Expect().Status(tc.expectedStatusCode)
		})
	}
}
