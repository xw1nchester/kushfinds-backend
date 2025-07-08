package tests

import (
	"fmt"
	"io"
	"net/http"
)

func (s *APITestSuite) TestPing() {
	response, err := http.Get(fmt.Sprintf("%s/ping", s.baseUrl))
	s.NoError(err)

	byteBody, err := io.ReadAll(response.Body)
	s.NoError(err)

	response.Body.Close()

	s.Equal(http.StatusOK, response.StatusCode)
	s.Equal("pong", string(byteBody))
}
