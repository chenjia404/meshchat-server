package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"meshchat-server/internal/events"
	"meshchat-server/internal/model"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DMService 上游私聊 relay / offline store。
type DMService struct {
	dm   *repo.DMRepo
	users *repo.UserRepo
	pub   EventPublisher
}

func NewDMService(dm *repo.DMRepo, users *repo.UserRepo, pub EventPublisher) *DMService {
	return &DMService{dm: dm, users: users, pub: pub}
}

// CanAccessConversation WebSocket 订阅校验。
func (s *DMService) CanAccessConversation(ctx context.Context, userID uint64, conversationID string) bool {
	cid, err := uuid.Parse(strings.TrimSpace(conversationID))
	if err != nil {
		return false
	}
	conv, err := s.dm.GetConversationByID(ctx, cid)
	if err != nil {
		return false
	}
	return s.dm.IsParticipant(conv, userID)
}

func (s *DMService) otherUserID(conv *model.DirectConversation, self uint64) uint64 {
	if conv.UserLowID == self {
		return conv.UserHighID
	}
	return conv.UserLowID
}

func (s *DMService) toMessageView(ctx context.Context, m *model.DirectMessage) (*DMMessageView, error) {
	sender, err := s.users.GetByID(ctx, m.SenderUserID)
	if err != nil {
		return nil, err
	}
	recipient, err := s.users.GetByID(ctx, m.RecipientUserID)
	if err != nil {
		return nil, err
	}
	var payload any
	if len(m.PayloadJSON) > 0 {
		if err := json.Unmarshal(m.PayloadJSON, &payload); err != nil {
			payload = json.RawMessage(m.PayloadJSON)
		}
	}
	return &DMMessageView{
		MessageID:        m.MessageID.String(),
		ConversationID: m.ConversationID.String(),
		Seq:              m.Seq,
		ContentType:      m.ContentType,
		Payload:          payload,
		SenderUserID:     m.SenderUserID,
		RecipientUserID:  m.RecipientUserID,
		SenderPeerID:     sender.PeerID,
		RecipientPeerID: recipient.PeerID,
		ClientMsgID:      m.ClientMsgID,
		Status:           m.Status,
		RecipientAckedAt: m.RecipientAckedAt,
		CreatedAt:        m.CreatedAt.UTC(),
	}, nil
}

// ListConversations 当前用户参与的 DM 会话。
func (s *DMService) ListConversations(ctx context.Context, userID uint64) ([]DMConversationView, error) {
	rows, err := s.dm.ListConversationsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]DMConversationView, 0, len(rows))
	for i := range rows {
		otherID := s.otherUserID(&rows[i], userID)
		u, err := s.users.GetByID(ctx, otherID)
		if err != nil {
			return nil, err
		}
		out = append(out, DMConversationView{
			ConversationID: rows[i].ConversationID.String(),
			PeerID:         u.PeerID,
			LastMessageSeq: rows[i].LastMessageSeq,
			LastMessageAt:  rows[i].LastMessageAt.UTC(),
		})
	}
	return out, nil
}

// CreateConversation 按对端 peer_id 获取或创建会话。
func (s *DMService) CreateConversation(ctx context.Context, userID uint64, in CreateDMConversationInput) (*DMConversationView, error) {
	peerID := strings.TrimSpace(in.PeerID)
	if peerID == "" {
		return nil, apperrors.New(400, "invalid_payload", "peer_id is required")
	}
	peer, err := s.users.GetByPeerID(ctx, peerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "user_not_found", "peer not registered on server")
		}
		return nil, err
	}
	if peer.ID == userID {
		return nil, apperrors.New(400, "invalid_payload", "cannot chat with yourself")
	}
	conv, err := s.dm.GetOrCreateConversation(ctx, userID, peer.ID)
	if err != nil {
		return nil, err
	}
	u, err := s.users.GetByID(ctx, s.otherUserID(conv, userID))
	if err != nil {
		return nil, err
	}
	return &DMConversationView{
		ConversationID: conv.ConversationID.String(),
		PeerID:         u.PeerID,
		LastMessageSeq: conv.LastMessageSeq,
		LastMessageAt:  conv.LastMessageAt.UTC(),
	}, nil
}

func (s *DMService) requireParticipant(ctx context.Context, userID uint64, convUUID uuid.UUID) (*model.DirectConversation, error) {
	conv, err := s.dm.GetConversationByID(ctx, convUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "conversation_not_found", "conversation not found")
		}
		return nil, err
	}
	if !s.dm.IsParticipant(conv, userID) {
		return nil, apperrors.New(403, "forbidden", "not a participant")
	}
	return conv, nil
}

// ListMessages 支持 before_seq（历史）或 after_seq（补拉），二者互斥优先 after_seq。
func (s *DMService) ListMessages(ctx context.Context, userID uint64, conversationID string, beforeSeq, afterSeq uint64, limit int) ([]DMMessageView, error) {
	cid, err := uuid.Parse(strings.TrimSpace(conversationID))
	if err != nil {
		return nil, apperrors.New(400, "invalid_payload", "invalid conversation_id")
	}
	if _, err := s.requireParticipant(ctx, userID, cid); err != nil {
		return nil, err
	}
	var rows []model.DirectMessage
	if afterSeq > 0 {
		rows, err = s.dm.ListMessagesAfter(ctx, cid, afterSeq, limit)
	} else {
		rows, err = s.dm.ListMessagesBefore(ctx, cid, beforeSeq, limit)
	}
	if err != nil {
		return nil, err
	}
	out := make([]DMMessageView, 0, len(rows))
	for i := range rows {
		v, err := s.toMessageView(ctx, &rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

func validateDMTextPayload(payload any) ([]byte, error) {
	if payload == nil {
		return nil, apperrors.New(400, "invalid_payload", "payload is required")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, apperrors.New(400, "invalid_payload", "payload must be json")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, apperrors.New(400, "invalid_payload", "payload must be object")
	}
	text, _ := m["text"].(string)
	if strings.TrimSpace(text) == "" {
		return nil, apperrors.New(400, "invalid_payload", "text is required for text messages")
	}
	return raw, nil
}

// SendMessage 持久化后投递；client_msg_id 幂等。
func (s *DMService) SendMessage(ctx context.Context, userID uint64, conversationID string, in SendDMMessageInput) (*DMMessageView, error) {
	cid, err := uuid.Parse(strings.TrimSpace(conversationID))
	if err != nil {
		return nil, apperrors.New(400, "invalid_payload", "invalid conversation_id")
	}
	conv, err := s.requireParticipant(ctx, userID, cid)
	if err != nil {
		return nil, err
	}
	clientMsgID := strings.TrimSpace(in.ClientMsgID)
	if clientMsgID == "" {
		return nil, apperrors.New(400, "invalid_payload", "client_msg_id is required")
	}
	ct := strings.TrimSpace(in.ContentType)
	if ct != model.MessageContentTypeText {
		return nil, apperrors.New(400, "invalid_payload", "only content_type=text is supported in this release")
	}
	payloadJSON, err := validateDMTextPayload(in.Payload)
	if err != nil {
		return nil, err
	}

	if existing, err := s.dm.FindByClientMsgID(ctx, cid, userID, clientMsgID); err == nil {
		return s.toMessageView(ctx, existing)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	recipientID := s.otherUserID(conv, userID)
	msg := &model.DirectMessage{
		MessageID:       uuid.New(),
		ConversationID:  cid,
		SenderUserID:    userID,
		RecipientUserID: recipientID,
		ClientMsgID:     clientMsgID,
		ContentType:     ct,
		PayloadJSON:     datatypes.JSON(payloadJSON),
		Status:          model.DMMessageStatusPendingAck,
	}

	if err := s.dm.CreateMessage(ctx, conv, msg); err != nil {
		if existing, e2 := s.dm.FindByClientMsgID(ctx, cid, userID, clientMsgID); e2 == nil {
			return s.toMessageView(ctx, existing)
		}
		return nil, err
	}

	view, err := s.toMessageView(ctx, msg)
	if err != nil {
		return nil, err
	}

	_ = s.pub.Publish(ctx, events.Envelope{
		Type:           events.EventDMMessageCreated,
		ConversationID: cid.String(),
		MessageID:      msg.MessageID.String(),
		At:             time.Now().UTC(),
	})

	return view, nil
}

// AckMessage 接收方 ACK，幂等。
func (s *DMService) AckMessage(ctx context.Context, userID uint64, messageID string) (*DMMessageView, error) {
	mid, err := uuid.Parse(strings.TrimSpace(messageID))
	if err != nil {
		return nil, apperrors.New(400, "invalid_payload", "invalid message_id")
	}
	m, err := s.dm.GetMessageByUUID(ctx, mid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "message_not_found", "message not found")
		}
		return nil, err
	}
	if _, err := s.requireParticipant(ctx, userID, m.ConversationID); err != nil {
		return nil, err
	}

	updated, err := s.dm.AckMessage(ctx, mid, userID)
	if err != nil {
		if errors.Is(err, repo.ErrDMNotRecipientForAck) {
			return nil, apperrors.New(403, "forbidden", "only recipient can ack")
		}
		return nil, err
	}

	view, err := s.toMessageView(ctx, updated)
	if err != nil {
		return nil, err
	}

	_ = s.pub.Publish(ctx, events.Envelope{
		Type:           events.EventDMMessageAcked,
		ConversationID: m.ConversationID.String(),
		MessageID:      m.MessageID.String(),
		UserID:         userID,
		At:             time.Now().UTC(),
	})

	return view, nil
}

// BuildRealtimeDMEvent 为指定用户构造 WS 负载（Redis 消费或本地广播）。
func (s *DMService) BuildRealtimeDMEvent(ctx context.Context, viewerID uint64, env events.Envelope) (*RealtimeEnvelope, error) {
	cid, err := uuid.Parse(strings.TrimSpace(env.ConversationID))
	if err != nil {
		return nil, err
	}
	if _, err := s.requireParticipant(ctx, viewerID, cid); err != nil {
		return nil, err
	}
	switch env.Type {
	case events.EventDMMessageCreated:
		mid, err := uuid.Parse(strings.TrimSpace(env.MessageID))
		if err != nil {
			return nil, err
		}
		m, err := s.dm.GetMessageByUUID(ctx, mid)
		if err != nil {
			return nil, err
		}
		v, err := s.toMessageView(ctx, m)
		if err != nil {
			return nil, err
		}
		return &RealtimeEnvelope{
			Type: events.EventDMMessageCreated,
			Data: map[string]any{
				"conversation_id": cid.String(),
				"message":         v,
			},
		}, nil
	case events.EventDMMessageAcked:
		mid, err := uuid.Parse(strings.TrimSpace(env.MessageID))
		if err != nil {
			return nil, err
		}
		m, err := s.dm.GetMessageByUUID(ctx, mid)
		if err != nil {
			return nil, err
		}
		v, err := s.toMessageView(ctx, m)
		if err != nil {
			return nil, err
		}
		return &RealtimeEnvelope{
			Type: events.EventDMMessageAcked,
			Data: map[string]any{
				"conversation_id": cid.String(),
				"message":         v,
			},
		}, nil
	default:
		return nil, nil
	}
}
