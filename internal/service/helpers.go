package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"meshchat-server/internal/model"
	"meshchat-server/pkg/apperrors"

	"gorm.io/datatypes"
)

func toPublicUser(user model.ServerUser) PublicUser {
	return PublicUser{
		ID:             user.ID,
		PeerID:         user.PeerID,
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		AvatarCID:      user.AvatarCID,
		Bio:            user.Bio,
		ProfileVersion: user.ProfileVersion,
		Status:         user.Status,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

func isVisibleByTTL(ttlSeconds int64, createdAt time.Time, now time.Time) bool {
	if ttlSeconds <= 0 {
		return true
	}
	return createdAt.After(now.Add(-time.Duration(ttlSeconds)*time.Second)) || createdAt.Equal(now.Add(-time.Duration(ttlSeconds)*time.Second))
}

func hasPermission(perms, perm int64) bool {
	return perms&perm == perm
}

func decodeStrict[T any](raw any, out *T) error {
	bytesValue, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(bytesValue))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

func encodeJSON(value any) (datatypes.JSON, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(raw), nil
}

func decodeJSONPayload(raw datatypes.JSON) (map[string]any, error) {
	var payload map[string]any
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func validateProfileFieldLengths(displayName, bio string) error {
	if len(displayName) > 128 || len(bio) > 1024 {
		return apperrors.New(400, "invalid_profile", "profile field is too long")
	}
	return nil
}

func payloadValidationError(detail string) error {
	return apperrors.New(400, "invalid_payload", detail)
}

func validateMediaCommon(cidValue, mimeType string, size int64) error {
	if cidValue == "" || mimeType == "" || size <= 0 {
		return payloadValidationError("cid, mime_type and size are required")
	}
	return nil
}

func errRecordNotVisible(kind string) error {
	return apperrors.New(400, "reference_not_visible", fmt.Sprintf("%s is deleted, expired or not visible", kind))
}

func sanitizeDeletedNotice(message *model.GroupMessage) string {
	if message.Status == model.MessageStatusDeleted {
		return "原消息已删除"
	}
	return "原消息已过期"
}

func duplicateConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique")
}

func require(cond bool, err error) error {
	if cond {
		return nil
	}
	return err
}

func combineErrors(base error, extra error) error {
	if base == nil {
		return extra
	}
	if extra == nil {
		return base
	}
	return errors.Join(base, extra)
}
