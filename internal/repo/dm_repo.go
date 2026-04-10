package repo

import (
	"context"
	"errors"
	"time"

	"meshchat-server/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrDMNotRecipientForAck 非接收方尝试 ACK。
var ErrDMNotRecipientForAck = errors.New("dm: not recipient for ack")

type DMRepo struct {
	db *gorm.DB
}

func NewDMRepo(db *gorm.DB) *DMRepo {
	return &DMRepo{db: db}
}

func userPair(low, high uint64) (uint64, uint64) {
	if low > high {
		return high, low
	}
	return low, high
}

// GetConversationByID 按 conversation UUID 查找。
func (r *DMRepo) GetConversationByID(ctx context.Context, convUUID uuid.UUID) (*model.DirectConversation, error) {
	var c model.DirectConversation
	if err := r.db.WithContext(ctx).Where("conversation_id = ?", convUUID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// IsParticipant 判断 userID 是否属于该会话。
func (r *DMRepo) IsParticipant(c *model.DirectConversation, userID uint64) bool {
	if c == nil {
		return false
	}
	return c.UserLowID == userID || c.UserHighID == userID
}

// GetOrCreateConversation 按双方 user id 查找或创建会话。
func (r *DMRepo) GetOrCreateConversation(ctx context.Context, a, b uint64) (*model.DirectConversation, error) {
	low, high := userPair(a, b)
	var c model.DirectConversation
	err := r.db.WithContext(ctx).Where("user_low_id = ? AND user_high_id = ?", low, high).First(&c).Error
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := time.Now().UTC()
	c = model.DirectConversation{
		ConversationID: uuid.New(),
		UserLowID:      low,
		UserHighID:     high,
		LastMessageSeq: 0,
		LastMessageAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
		if err2 := r.db.WithContext(ctx).Where("user_low_id = ? AND user_high_id = ?", low, high).First(&c).Error; err2 == nil {
			return &c, nil
		}
		return nil, err
	}
	return &c, nil
}

// FindByClientMsgID 幂等键查询。
func (r *DMRepo) FindByClientMsgID(ctx context.Context, convUUID uuid.UUID, senderID uint64, clientMsgID string) (*model.DirectMessage, error) {
	var m model.DirectMessage
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND sender_user_id = ? AND client_msg_id = ?", convUUID, senderID, clientMsgID).
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateMessage 在事务内插入消息并推进会话序号。
func (r *DMRepo) CreateMessage(ctx context.Context, conv *model.DirectConversation, m *model.DirectMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked model.DirectConversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", conv.ID).First(&locked).Error; err != nil {
			return err
		}
		nextSeq := locked.LastMessageSeq + 1
		m.Seq = nextSeq
		now := time.Now().UTC()
		m.CreatedAt = now
		m.UpdatedAt = now
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		*conv = locked
		conv.LastMessageSeq = nextSeq
		conv.LastMessageAt = now
		conv.UpdatedAt = now
		return tx.Model(&locked).Updates(map[string]any{
			"last_message_seq": nextSeq,
			"last_message_at":  now,
			"updated_at":       now,
		}).Error
	})
}

// ListMessagesBefore 历史分页：seq < beforeSeq，按 seq 降序，最多 limit 条。
func (r *DMRepo) ListMessagesBefore(ctx context.Context, convUUID uuid.UUID, beforeSeq uint64, limit int) ([]model.DirectMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := r.db.WithContext(ctx).Where("conversation_id = ?", convUUID).Order("seq DESC").Limit(limit)
	if beforeSeq > 0 {
		q = q.Where("seq < ?", beforeSeq)
	}
	var rows []model.DirectMessage
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	// 返回时间正序，便于客户端展示
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}

// ListMessagesAfter 增量补拉：seq > afterSeq，按 seq 升序。
func (r *DMRepo) ListMessagesAfter(ctx context.Context, convUUID uuid.UUID, afterSeq uint64, limit int) ([]model.DirectMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []model.DirectMessage
	q := r.db.WithContext(ctx).Where("conversation_id = ? AND seq > ?", convUUID, afterSeq).Order("seq ASC").Limit(limit)
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetMessageByUUID 按 message_id UUID 查找。
func (r *DMRepo) GetMessageByUUID(ctx context.Context, id uuid.UUID) (*model.DirectMessage, error) {
	var m model.DirectMessage
	if err := r.db.WithContext(ctx).Where("message_id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// AckMessage 接收方幂等 ACK。
func (r *DMRepo) AckMessage(ctx context.Context, msgUUID uuid.UUID, recipientUserID uint64) (*model.DirectMessage, error) {
	var out *model.DirectMessage
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m model.DirectMessage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("message_id = ?", msgUUID).First(&m).Error; err != nil {
			return err
		}
		if m.RecipientUserID != recipientUserID {
			return ErrDMNotRecipientForAck
		}
		now := time.Now().UTC()
		if m.Status == model.DMMessageStatusAcked && m.RecipientAckedAt != nil {
			out = &m
			return nil
		}
		m.Status = model.DMMessageStatusAcked
		m.RecipientAckedAt = &now
		m.UpdatedAt = now
		if err := tx.Model(&m).Updates(map[string]any{
			"status":               m.Status,
			"recipient_acked_at":   m.RecipientAckedAt,
			"updated_at":           m.UpdatedAt,
		}).Error; err != nil {
			return err
		}
		out = &m
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListConversationsForUser 返回用户参与的所有 DM 会话。
func (r *DMRepo) ListConversationsForUser(ctx context.Context, userID uint64) ([]model.DirectConversation, error) {
	var rows []model.DirectConversation
	err := r.db.WithContext(ctx).
		Where("user_low_id = ? OR user_high_id = ?", userID, userID).
		Order("last_message_at DESC").
		Find(&rows).Error
	return rows, err
}
