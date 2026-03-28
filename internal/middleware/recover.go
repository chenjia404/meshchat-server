package middleware

import (
	"log/slog"
	"net/http"

	"meshchat-server/pkg/apperrors"
)

func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("request panicked", slog.Any("panic", recovered))
					writeMiddlewareError(w, apperrors.New(http.StatusInternalServerError, "panic", "internal server error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
