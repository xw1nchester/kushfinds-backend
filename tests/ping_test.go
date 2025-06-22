package tests

import (
	"net/url"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

func TestPing(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
	}
	e := httpexpect.Default(t, u.String())

	e.GET("/api/ping").
		Expect().
		Status(200) 
}