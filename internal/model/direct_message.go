package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// DirectConversation 上游 relay/offline store 用的双人会话元数据（非 mesh-proxy 主模型替代物）。
type DirectConversation struct {
	ID               uint64    `gorm:"primaryKey"`
	ConversationID   uuid.UUID `gorm:"column:conversation_id;type:uuid;uniqueIndex:idx_dm_conv_uuid;not null"`
	UserLowID        uint64    `gorm:"not null;uniqueIndex:idx_dm_user_pair"`
	UserHighID       uint64    `gorm:"not null;uniqueIndex:idx_dm_user_pair"`
	LastMessageSeq   uint64    `gorm:"not null;default:0"`
	LastMessageAt    time.Time `gorm:"not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (DirectConversation) TableName() string { return "direct_conversations" }

// DirectMessage 上游私聊消息行（持久化后投递，at-least-once）。
type DirectMessage struct {
	ID               uint64         `gorm:"primaryKey"`
	MessageID        uuid.UUID      `gorm:"column:message_id;type:uuid;uniqueIndex:idx_dm_msg_uuid;not null"`
	ConversationID   uuid.UUID      `gorm:"column:conversation_id;type:uuid;not null;index:idx_dm_msg_conv_seq"`
	SenderUserID     uint64         `gorm:"not null;index"`
	RecipientUserID  uint64         `gorm:"not null;index"`
	ClientMsgID      string         `gorm:"size:128;not null"`
	ContentType      string         `gorm:"size:32;not null"`
	PayloadJSON      datatypes.JSON `gorm:"type:jsonb;not null"`
	Status           string         `gorm:"size:32;not null"` // pending_ack | acked
	Seq              uint64         `gorm:"not null;index:idx_dm_msg_conv_seq"`
	RecipientAckedAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (DirectMessage) TableName() string { return "direct_messages" }

const (
	DMMessageStatusPendingAck = "pending_ack"
	DMMessageStatusAcked      = "acked"
)
