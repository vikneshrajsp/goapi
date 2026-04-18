package middleware

import (
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"goapi/api"
	"goapi/internal/tools"
)

var ErrorUnauthorized = errors.New("unauthorized. Invalid token or expired token")

func AuthorizeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		token := r.Header.Get("Authorization")
		if username == "" || token == "" {
			log.Error(ErrorUnauthorized.Error())
			api.RequestErrorHandler(w, r, ErrorUnauthorized)
			return
		}

		database, error := tools.NewDatabase()
		if error != nil {
			log.Error(error)
			api.RequestErrorHandler(w, r, error)
			return
		}

		loginDetails, error := database.GetLoginDetails(username)
		if error != nil {
			log.Error(error)
			api.RequestErrorHandler(w, r, error)
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
