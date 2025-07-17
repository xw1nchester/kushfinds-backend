package apperror

import (
	"errors"
	"net/http"
)

type handler func(w http.ResponseWriter, r *http.Request) error

func Middleware(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		err := h(w, r)

		var appErr *AppError
		if err != nil {
			if errors.As(err, &appErr) {
				if errors.Is(err, ErrNotFound) {
					w.WriteHeader(http.StatusNotFound)
				} else if errors.Is(err, ErrUnauthorized) {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusBadRequest)
				}

				w.Write(appErr.Marshal())

				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write(internalError().Marshal())
		}
	}
}
