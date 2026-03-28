package service

import (
	"context"

	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"gorm.io/gorm"
)

// ServerAdminService resolves server-level administrators from configured peer IDs.
type ServerAdminService struct {
	users          *repo.UserRepo
	allowedPeerIDs map[string]struct{}
}

func NewServerAdminService(users *repo.UserRepo, peerIDs []string) *ServerAdminService {
	allowed := make(map[string]struct{}, len(peerIDs))
	for _, peerID := range peerIDs {
		allowed[peerID] = struct{}{}
	}
	return &ServerAdminService{
		users:          users,
		allowedPeerIDs: allowed,
	}
}

func (s *ServerAdminService) IsServerAdmin(ctx context.Context, userID uint64) (bool, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, apperrors.New(404, "user_not_found", "user not found")
		}
		return false, err
	}

	_, ok := s.allowedPeerIDs[user.PeerID]
	return ok, nil
}

func (s *ServerAdminService) RequireServerAdmin(ctx context.Context, userID uint64) error {
	ok, err := s.IsServerAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.New(403, "server_admin_required", "server administrator permission is required")
	}
	return nil
}
