package middleware

import (
	"context"
	"net/http"
	"strings"

	"meshchat-server/internal/auth"
	"meshchat-server/pkg/apperrors"
)

type adminAuthContextKey string

const adminUsernameContextKey adminAuthContextKey = "admin_username"

func AdminUsernameFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(adminUsernameContextKey).(string)
	return value, ok
}

func RequireAdminAuth(jwtManager *auth.AdminJWTManager) func(http.Handler) http.Handler {
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

			ctx := context.WithValue(r.Context(), adminUsernameContextKey, claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
