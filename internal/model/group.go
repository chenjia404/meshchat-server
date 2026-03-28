package model

import (
	"time"

	"github.com/google/uuid"
)

// Group stores mutable room-level settings. TTL and slow mode are evaluated dynamically.
type Group struct {
	ID                     uint64     `gorm:"primaryKey" json:"id"`
	GroupID                uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"group_id"`
	Title                  string     `gorm:"size:256;not null" json:"title"`
	About                  string     `gorm:"size:2048" json:"about"`
	AvatarCID              string     `gorm:"column:avatar_cid;size:255" json:"avatar_cid"`
	OwnerUserID            uint64     `gorm:"not null;index" json:"owner_user_id"`
	OwnerUser              ServerUser `gorm:"foreignKey:OwnerUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	MemberListVisibility   string     `gorm:"size:32;not null;default:'visible'" json:"member_list_visibility"`
	JoinMode               string     `gorm:"size:32;not null;default:'invite_only'" json:"join_mode"`
	DefaultPermissions     int64      `gorm:"not null;default:0" json:"default_permissions"`
	MessageTTLSeconds      int64      `gorm:"not null;default:0" json:"message_ttl_seconds"`
	MessageRetractSeconds  int64      `gorm:"not null;default:0" json:"message_retract_seconds"`
	MessageCooldownSeconds int64      `gorm:"not null;default:0" json:"message_cooldown_seconds"`
	LastMessageSeq         uint64     `gorm:"not null;default:0" json:"last_message_seq"`
	SettingsVersion        uint64     `gorm:"not null;default:1" json:"settings_version"`
	Status                 string     `gorm:"size:32;not null;default:'active'" json:"status"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// GroupMember stores membership and permission overrides.
type GroupMember struct {
	ID               uint64     `gorm:"primaryKey" json:"id"`
	GroupID          uint64     `gorm:"not null;uniqueIndex:idx_group_member_user;index" json:"-"`
	Group            Group      `gorm:"foreignKey:GroupID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	UserID           uint64     `gorm:"not null;uniqueIndex:idx_group_member_user;index" json:"user_id"`
	User             ServerUser `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Role             string     `gorm:"size:32;not null" json:"role"`
	Status           string     `gorm:"size:32;not null" json:"status"`
	Title            string     `gorm:"size:128" json:"title"`
	JoinedAt         time.Time  `gorm:"not null" json:"joined_at"`
	MutedUntil       *time.Time `json:"muted_until"`
	PermissionsAllow int64      `gorm:"not null;default:0" json:"permissions_allow"`
	PermissionsDeny  int64      `gorm:"not null;default:0" json:"permissions_deny"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
