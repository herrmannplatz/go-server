package middleware

import (
	"context"
	"net/http"

	"github.com/herrmannplatz/go-server/pkg/auth"
	"github.com/herrmannplatz/go-server/pkg/util"
)

type authContextKey string

func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user_id, err := auth.ValidateRequestToken(r, secret)
			if err != nil {
				util.WriteJSONError(w, http.StatusUnauthorized, err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), authContextKey("user_id"), user_id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
