package model

import (
	"time"

	"gorm.io/datatypes"
)

// GroupMessage stores the message envelope and payload without storing any binary content.
type GroupMessage struct {
	ID                   uint64         `gorm:"primaryKey" json:"id"`
	GroupID              uint64         `gorm:"not null;uniqueIndex:idx_group_message_message_id,priority:1;uniqueIndex:idx_group_message_seq,priority:1;index:idx_group_created_at,priority:1;index:idx_group_sender_created_at,priority:1;index:idx_group_reply,priority:1" json:"-"`
	Group                Group          `gorm:"foreignKey:GroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	MessageID            string         `gorm:"type:text;not null;uniqueIndex:idx_group_message_message_id,priority:2" json:"message_id"`
	SenderUserID         uint64         `gorm:"not null;index:idx_group_sender_created_at,priority:2" json:"-"`
	Sender               ServerUser     `gorm:"foreignKey:SenderUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Seq                  uint64         `gorm:"not null;uniqueIndex:idx_group_message_seq,priority:2" json:"seq"`
	ContentType          string         `gorm:"size:32;not null" json:"content_type"`
	PayloadJSON          datatypes.JSON `gorm:"type:jsonb;not null" json:"payload_json"`
	ReplyToMessageID     *string        `gorm:"type:text;index:idx_group_reply,priority:2" json:"reply_to_message_id"`
	ForwardFromMessageID *string        `gorm:"type:text;index" json:"forward_from_message_id"`
	Status               string         `gorm:"size:32;not null;default:'normal'" json:"status"`
	EditCount            uint32         `gorm:"not null;default:0" json:"edit_count"`
	LastEditedAt         *time.Time     `json:"last_edited_at"`
	LastEditedByUserID   *uint64        `json:"-"`
	DeletedAt            *time.Time     `json:"deleted_at"`
	DeletedByUserID      *uint64        `json:"-"`
	DeleteReason         *string        `gorm:"size:64" json:"delete_reason"`
	Signature            string         `gorm:"type:text" json:"-"`
	CreatedAt            time.Time      `gorm:"index:idx_group_created_at,priority:2;index:idx_group_sender_created_at,priority:3,sort:desc" json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// GroupMessageEdit stores a full edit history for auditability.
type GroupMessageEdit struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	GroupID        uint64         `gorm:"not null;index" json:"-"`
	MessageID      string         `gorm:"type:text;not null;index" json:"message_id"`
	EditorUserID   uint64         `gorm:"not null;index" json:"-"`
	OldPayloadJSON datatypes.JSON `gorm:"type:jsonb;not null" json:"old_payload_json"`
	NewPayloadJSON datatypes.JSON `gorm:"type:jsonb;not null" json:"new_payload_json"`
	CreatedAt      time.Time      `json:"created_at"`
}
