package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/test-task-front/wms/internal/auth"
	"github.com/test-task-front/wms/internal/http/handlers"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userID"
	ContextKeyEmail  contextKey = "email"
)

func WithAuth(jwtManager *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				handlers.WriteUnauthorized(w)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				handlers.WriteUnauthorized(w)
				return
			}

			claims, err := jwtManager.ParseToken(parts[1])
			if err != nil {
				handlers.WriteUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.Subject)
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
