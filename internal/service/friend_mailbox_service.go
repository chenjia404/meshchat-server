package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"meshchat-server/internal/model"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type FriendMailboxService struct {
	mail  *repo.FriendMailboxRepo
	users *repo.UserRepo
}

func NewFriendMailboxService(mail *repo.FriendMailboxRepo, users *repo.UserRepo) *FriendMailboxService {
	return &FriendMailboxService{mail: mail, users: users}
}

// FriendMailboxView 与 mesh-proxy / 客户端 FriendRequestDto 字段对齐，便于 mesh-proxy 转发。
type FriendMailboxView struct {
	RequestID        string `json:"request_id"`
	FromPeerID       string `json:"from_peer_id"`
	ToPeerID         string `json:"to_peer_id"`
	State            string `json:"state"`
	IntroText        string `json:"intro_text"`
	Nickname         string `json:"nickname"`
	Bio              string `json:"bio,omitempty"`
	Avatar           string `json:"avatar,omitempty"` // 与 mesh-proxy 键名一致，值为 avatar_cid
	RetentionMinutes int    `json:"retention_minutes"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func toFriendMailboxView(r model.FriendMailboxRequest) FriendMailboxView {
	return FriendMailboxView{
		RequestID:        r.RequestID,
		FromPeerID:       r.FromPeerID,
		ToPeerID:         r.ToPeerID,
		State:            r.State,
		IntroText:        r.IntroText,
		Nickname:         r.Nickname,
		Bio:              r.Bio,
		Avatar:           r.AvatarCID,
		RetentionMinutes: 0,
		CreatedAt:        r.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (s *FriendMailboxService) List(ctx context.Context, userID uint64) ([]FriendMailboxView, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "user_not_found", "user not found")
		}
		return nil, err
	}
	peer := strings.TrimSpace(u.PeerID)
	if peer == "" {
		return nil, apperrors.New(400, "invalid_peer", "user has no peer_id")
	}
	rows, err := s.mail.ListForPeer(ctx, peer, 200)
	if err != nil {
		return nil, err
	}
	out := make([]FriendMailboxView, 0, len(rows))
	for _, r := range rows {
		out = append(out, toFriendMailboxView(r))
	}
	return out, nil
}

type CreateFriendMailboxInput struct {
	ToPeerID  string
	IntroText string
	Nickname  string
	Bio       string
	AvatarCID string
}

func (s *FriendMailboxService) Create(ctx context.Context, userID uint64, in CreateFriendMailboxInput) (*FriendMailboxView, error) {
	fromUser, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "user_not_found", "user not found")
		}
		return nil, err
	}
	fromPeer := strings.TrimSpace(fromUser.PeerID)
	if fromPeer == "" {
		return nil, apperrors.New(400, "invalid_peer", "user has no peer_id")
	}
	toPeer := strings.TrimSpace(in.ToPeerID)
	if toPeer == "" || toPeer == fromPeer {
		return nil, apperrors.New(400, "invalid_to_peer", "to_peer_id is invalid")
	}
	intro := strings.TrimSpace(in.IntroText)
	if len(intro) > 4096 {
		return nil, apperrors.New(400, "invalid_intro", "intro_text is too long")
	}
	toUser, err := s.users.GetByPeerID(ctx, toPeer)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "target_not_found", "target peer is not registered on this server")
		}
		return nil, err
	}
	if toUser.ID == userID {
		return nil, apperrors.New(400, "invalid_to_peer", "cannot send friend mailbox request to yourself")
	}

	nick := strings.TrimSpace(in.Nickname)
	bio := strings.TrimSpace(in.Bio)
	avatar := strings.TrimSpace(in.AvatarCID)
	if nick == "" {
		nick = strings.TrimSpace(fromUser.DisplayName)
	}
	if nick == "" {
		nick = fromUser.Username
	}
	if bio == "" {
		bio = strings.TrimSpace(fromUser.Bio)
	}
	if avatar == "" {
		avatar = strings.TrimSpace(fromUser.AvatarCID)
	}
	if err := validateFriendMailboxProfileFields(nick, bio, avatar); err != nil {
		return nil, err
	}

	row := &model.FriendMailboxRequest{
		RequestID:  uuid.NewString(),
		FromPeerID: fromPeer,
		ToPeerID:   toPeer,
		State:      model.FriendMailboxStatePending,
		IntroText:  intro,
		Nickname:   nick,
		Bio:        bio,
		AvatarCID:  avatar,
	}
	if err := s.mail.Create(ctx, row); err != nil {
		if isUniquePendingPairViolation(err) {
			return nil, apperrors.New(409, "mailbox_pending_exists", "a pending friend mailbox request already exists for this pair")
		}
		return nil, err
	}
	v := toFriendMailboxView(*row)
	return &v, nil
}

func validateFriendMailboxProfileFields(nickname, bio, avatarCID string) error {
	if len(nickname) > 128 {
		return apperrors.New(400, "invalid_nickname", "nickname is too long")
	}
	if len(bio) > 1024 {
		return apperrors.New(400, "invalid_bio", "bio is too long")
	}
	if len(avatarCID) > 255 {
		return apperrors.New(400, "invalid_avatar_cid", "avatar_cid is too long")
	}
	return nil
}

func isUniquePendingPairViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "idx_friend_mailbox_pending")
}

func (s *FriendMailboxService) Accept(ctx context.Context, userID uint64, requestID string) (*FriendMailboxView, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "user_not_found", "user not found")
		}
		return nil, err
	}
	peer := strings.TrimSpace(u.PeerID)
	row, err := s.mail.UpdateStateIfToPeer(ctx, strings.TrimSpace(requestID), peer, model.FriendMailboxStateAccepted)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "request_not_found", "friend mailbox request not found")
		}
		if errors.Is(err, repo.ErrNotMailboxRecipient) {
			return nil, apperrors.New(403, "forbidden", "only the recipient can accept this request")
		}
		if errors.Is(err, repo.ErrMailboxNotPending) {
			return nil, apperrors.New(409, "request_not_pending", "request is not pending")
		}
		return nil, err
	}
	v := toFriendMailboxView(*row)
	return &v, nil
}

func (s *FriendMailboxService) Reject(ctx context.Context, userID uint64, requestID string) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.New(404, "user_not_found", "user not found")
		}
		return err
	}
	peer := strings.TrimSpace(u.PeerID)
	_, err = s.mail.UpdateStateIfToPeer(ctx, strings.TrimSpace(requestID), peer, model.FriendMailboxStateRejected)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(404, "request_not_found", "friend mailbox request not found")
		}
		if errors.Is(err, repo.ErrNotMailboxRecipient) {
			return apperrors.New(403, "forbidden", "only the recipient can reject this request")
		}
		if errors.Is(err, repo.ErrMailboxNotPending) {
			return apperrors.New(409, "request_not_pending", "request is not pending")
		}
		return err
	}
	return nil
}
