package middleware

import (
	"context"
	"net/http"
	"strings"

	"meshchat-server/internal/auth"
	"meshchat-server/pkg/apperrors"
)

type authContextKey string

const userContextKey authContextKey = "user_id"

func UserIDFromContext(ctx context.Context) (uint64, bool) {
	value, ok := ctx.Value(userContextKey).(uint64)
	return value, ok
}

func RequireAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(header, "Bearer ") {
				writeMiddlewareError(w, apperrors.New(http.StatusUnauthorized, "missing_token", "authorization token is required"))
				return
			}

			claims, err := jwtManager.ParseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err != nil {
				writeMiddlewareError(w, apperrors.New(http.StatusUnauthorized, "invalid_token", "authorization token is invalid"))
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
