package service

import (
	"testing"

	"meshchat-server/pkg/apperrors"
)

func TestValidateDMTextPayload(t *testing.T) {
	t.Parallel()
	_, err := validateDMTextPayload(map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	_, err = validateDMTextPayload(map[string]any{"text": ""})
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if !apperrors.Is(err, "invalid_payload") {
		t.Fatalf("expected invalid_payload, got %v", err)
	}
	_, err = validateDMTextPayload(nil)
	if err == nil {
		t.Fatal("expected error for nil payload")
	}
}
