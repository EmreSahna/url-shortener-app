package middleware

import (
	"context"
	"net/http"
)

func BearerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "userID", r.Header.Get("Authorization"))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
