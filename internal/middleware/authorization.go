package middleware

import (
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"goapi/api"
	"goapi/internal/database"
)

var ErrorUnauthorized = errors.New("unauthorized. Invalid token or expired token")

// Authorize validates auth token against the repository for the username query param.
func Authorize(repo database.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username := r.URL.Query().Get("username")
			token := r.Header.Get("Authorization")
			if username == "" || token == "" {
				log.Error(ErrorUnauthorized.Error())
				api.RequestErrorHandler(w, r, ErrorUnauthorized)
				return
			}

			loginDetails, err := repo.GetLoginDetails(r.Context(), username)
			if err != nil {
				log.Error(err)
				if errors.Is(err, database.ErrUserNotFound) {
					api.RequestErrorHandler(w, r, err)
					return
				}
				api.RequestErrorHandler(w, r, err)
				return
			}

			if loginDetails.AuthToken != token {
				log.Error(ErrorUnauthorized.Error())
				api.RequestErrorHandler(w, r, ErrorUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
