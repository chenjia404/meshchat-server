package model

import (
	"time"

	"gorm.io/datatypes"
)

// PublicChannel stores the canonical server-side profile/head for a public channel.
type PublicChannel struct {
	ID                      uint64         `gorm:"primaryKey"`
	ChannelID               string         `gorm:"size:320;uniqueIndex;not null"`
	OwnerUserID             uint64         `gorm:"not null;index"`
	OwnerUser               ServerUser     `gorm:"foreignKey:OwnerUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	OwnerPeerID             string         `gorm:"size:255;not null;index"`
	OwnerVersion            int64          `gorm:"not null;default:1"`
	Name                    string         `gorm:"size:256;not null"`
	AvatarJSON              datatypes.JSON `gorm:"type:jsonb;not null"`
	Bio                     string         `gorm:"type:text;not null;default:''"`
	MessageRetentionMinutes int            `gorm:"not null;default:0"`
	ProfileVersion          int64          `gorm:"not null;default:1"`
	LastMessageID           int64          `gorm:"not null;default:0"`
	LastSeq                 int64          `gorm:"not null;default:0"`
	CreatedAtUnix           int64          `gorm:"not null"`
	UpdatedAtUnix           int64          `gorm:"not null"`
	HeadUpdatedAtUnix       int64          `gorm:"not null;default:0"`
	ProfileSignature        string         `gorm:"type:text;not null;default:''"`
	HeadSignature           string         `gorm:"type:text;not null;default:''"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type PublicChannelMessage struct {
	ID            uint64         `gorm:"primaryKey"`
	ChannelDBID   uint64         `gorm:"not null;uniqueIndex:idx_public_channel_message_idem;uniqueIndex:idx_public_channel_message_seq;index"`
	MessageID     int64          `gorm:"not null;uniqueIndex:idx_public_channel_message_idem"`
	Version       int64          `gorm:"not null;default:1"`
	Seq           int64          `gorm:"not null;uniqueIndex:idx_public_channel_message_seq"`
	OwnerVersion  int64          `gorm:"not null;default:1"`
	CreatorPeerID string         `gorm:"size:255;not null"`
	AuthorPeerID  string         `gorm:"size:255;not null"`
	CreatedAtUnix int64          `gorm:"not null"`
	UpdatedAtUnix int64          `gorm:"not null"`
	IsDeleted     bool           `gorm:"not null;default:false"`
	MessageType   string         `gorm:"size:32;not null"`
	ContentJSON   datatypes.JSON `gorm:"type:jsonb;not null"`
	Signature     string         `gorm:"type:text;not null;default:''"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PublicChannelChange struct {
	ID             uint64     `gorm:"primaryKey"`
	ChannelDBID    uint64     `gorm:"not null;uniqueIndex:idx_public_channel_change_seq;index"`
	Seq            int64      `gorm:"not null;uniqueIndex:idx_public_channel_change_seq"`
	ChangeType     string     `gorm:"size:32;not null"`
	MessageID      *int64
	Version        *int64
	IsDeleted      *bool
	ProfileVersion *int64
	CreatedAtUnix  int64      `gorm:"not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PublicChannelSubscription struct {
	ID                    uint64    `gorm:"primaryKey"`
	ChannelDBID           uint64    `gorm:"not null;uniqueIndex:idx_public_channel_subscription_user;index"`
	UserID                uint64    `gorm:"not null;uniqueIndex:idx_public_channel_subscription_user;index"`
	LastSeenSeq           int64     `gorm:"not null;default:0"`
	LastSyncedSeq         int64     `gorm:"not null;default:0"`
	LatestLoadedMessageID int64     `gorm:"not null;default:0"`
	OldestLoadedMessageID int64     `gorm:"not null;default:0"`
	UnreadCount           int       `gorm:"not null;default:0"`
	Subscribed            bool      `gorm:"not null;default:true"`
	UpdatedAtUnix         int64     `gorm:"not null;default:0"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
