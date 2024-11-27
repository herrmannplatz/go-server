package middleware

import (
	"context"
	"net/http"

	"github.com/herrmannplatz/chirpy/pkg/auth"
	"github.com/herrmannplatz/chirpy/pkg/util"
)

type authContextKey string

func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user_id, err := auth.ValidateRequestToken(r, secret)
			if err != nil {
				response := struct {
					Error string `json:"error"`
				}{Error: err.Error()}
				util.SendJSON(w, http.StatusUnauthorized, response)
				return
			}

			ctx := context.WithValue(r.Context(), authContextKey("user_id"), user_id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
