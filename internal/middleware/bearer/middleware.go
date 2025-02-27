package bearer

import (
	"context"
	"net/http"
	"strings"

	"github.com/emresahna/url-shortener-app/internal/models"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		at := r.Header.Get("Authorization")
		if at == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(at, "Bearer ")
		if len(parts) != 2 {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), models.TokenKey, parts[1])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
