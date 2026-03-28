package middleware

import (
	"encoding/json"
	"net/http"

	"meshchat-server/pkg/apperrors"
)

func writeMiddlewareError(w http.ResponseWriter, err error) {
	publicErr := apperrors.Public(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apperrors.HTTPStatus(err))
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": publicErr,
	})
}
