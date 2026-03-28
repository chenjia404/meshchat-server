package service

import (
	"context"
	"strings"

	"meshchat-server/internal/ipfs"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"gorm.io/gorm"
)

type ProfileService struct {
	users *repo.UserRepo
	ipfs  ipfs.Client
}

func NewProfileService(users *repo.UserRepo, ipfs ipfs.Client) *ProfileService {
	return &ProfileService{users: users, ipfs: ipfs}
}

func (s *ProfileService) GetProfile(ctx context.Context, userID uint64) (*PublicUser, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "user_not_found", "user not found")
		}
		return nil, err
	}
	publicUser := toPublicUser(*user)
	return &publicUser, nil
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID uint64, input UpdateProfileInput) (*PublicUser, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "user_not_found", "user not found")
		}
		return nil, err
	}

	if err := validateProfileFieldLengths(input.Username, input.DisplayName, input.Bio); err != nil {
		return nil, err
	}
	if input.AvatarCID != "" {
		if err := s.ipfs.ValidateCID(input.AvatarCID); err != nil {
			return nil, payloadValidationError("avatar_cid is invalid")
		}
	}

	user.Username = strings.TrimSpace(input.Username)
	user.DisplayName = strings.TrimSpace(input.DisplayName)
	user.AvatarCID = strings.TrimSpace(input.AvatarCID)
	user.Bio = strings.TrimSpace(input.Bio)
	if input.Status != "" {
		user.Status = strings.TrimSpace(input.Status)
	}
	user.ProfileVersion++

	if err := s.users.UpdateProfile(ctx, user); err != nil {
		if duplicateConstraintError(err) {
			return nil, apperrors.New(409, "username_taken", "username already exists")
		}
		return nil, err
	}

	publicUser := toPublicUser(*user)
	return &publicUser, nil
}
