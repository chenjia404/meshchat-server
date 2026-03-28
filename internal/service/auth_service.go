package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"meshchat-server/internal/auth"
	"meshchat-server/internal/model"
	"meshchat-server/internal/redisx"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AuthService struct {
	users        *repo.UserRepo
	redis        *redis.Client
	jwt          *auth.JWTManager
	verifier     auth.Verifier
	challengeTTL time.Duration
}

type challengeRecord struct {
	PeerID    string    `json:"peer_id"`
	Challenge string    `json:"challenge"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewAuthService(users *repo.UserRepo, redis *redis.Client, jwt *auth.JWTManager, verifier auth.Verifier, challengeTTL time.Duration) *AuthService {
	return &AuthService{
		users:        users,
		redis:        redis,
		jwt:          jwt,
		verifier:     verifier,
		challengeTTL: challengeTTL,
	}
}

func (s *AuthService) RequestChallenge(ctx context.Context, peerID string) (*ChallengeResponse, error) {
	if strings.TrimSpace(peerID) == "" {
		return nil, apperrors.New(400, "invalid_peer_id", "peer_id is required")
	}

	challengeID := uuid.NewString()
	expiresAt := time.Now().UTC().Add(s.challengeTTL)
	challenge := fmt.Sprintf("meshchat login\nchallenge_id=%s\npeer_id=%s\nexpires_at=%s", challengeID, peerID, expiresAt.Format(time.RFC3339))
	record := challengeRecord{
		PeerID:    peerID,
		Challenge: challenge,
		ExpiresAt: expiresAt,
	}

	raw, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	if err := s.redis.Set(ctx, redisx.ChallengeKey(challengeID), raw, s.challengeTTL).Err(); err != nil {
		return nil, err
	}

	return &ChallengeResponse{
		ChallengeID: challengeID,
		Challenge:   challenge,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, peerID, challengeID, signature, publicKey string) (*LoginResponse, error) {
	raw, err := s.redis.Get(ctx, redisx.ChallengeKey(challengeID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, apperrors.New(401, "challenge_not_found", "challenge is missing or expired")
		}
		return nil, err
	}

	var record challengeRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	if record.PeerID != peerID || record.ExpiresAt.Before(time.Now().UTC()) {
		return nil, apperrors.New(401, "challenge_expired", "challenge is missing or expired")
	}

	if err := s.verifier.Verify(peerID, publicKey, record.Challenge, signature); err != nil {
		return nil, apperrors.New(401, "invalid_signature", err.Error())
	}

	user, err := s.users.GetByPeerID(ctx, peerID)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}

		user = &model.ServerUser{
			PeerID:         peerID,
			PublicKey:      publicKey,
			Username:       buildBootstrapUsername(peerID),
			DisplayName:    buildBootstrapUsername(peerID),
			ProfileVersion: 1,
			Status:         "active",
		}
		if err := s.users.Create(ctx, user); err != nil {
			if duplicateConstraintError(err) {
				user.Username = fmt.Sprintf("%s_%s", user.Username, uuid.NewString()[:8])
				user.DisplayName = user.Username
				if err := s.users.Create(ctx, user); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	token, err := s.jwt.IssueToken(user.ID)
	if err != nil {
		return nil, err
	}

	_ = s.redis.Del(ctx, redisx.ChallengeKey(challengeID)).Err()

	return &LoginResponse{
		Token: token,
		User:  toPublicUser(*user),
	}, nil
}

func buildBootstrapUsername(peerID string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		default:
			return -1
		}
	}, peerID)
	if len(safe) > 16 {
		safe = safe[len(safe)-16:]
	}
	if safe == "" {
		safe = uuid.NewString()[:12]
	}
	return "user_" + safe
}
