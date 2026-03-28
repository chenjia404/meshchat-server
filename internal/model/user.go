package model

import "time"

// ServerUser stores server-local profile data. peer_id never leaves internal APIs.
type ServerUser struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	PeerID         string    `gorm:"size:255;uniqueIndex;not null" json:"-"`
	PublicKey      string    `gorm:"type:text;not null" json:"-"`
	Username       string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	DisplayName    string    `gorm:"size:128;not null" json:"display_name"`
	AvatarCID      string    `gorm:"size:255" json:"avatar_cid"`
	Bio            string    `gorm:"size:1024" json:"bio"`
	ProfileVersion uint64    `gorm:"not null;default:1" json:"profile_version"`
	Status         string    `gorm:"size:32;not null;default:'active'" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
