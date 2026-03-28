package model

import "time"

// File stores metadata for content addressed by CID in IPFS.
type File struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	CID             string    `gorm:"column:c_id;size:255;uniqueIndex;not null" json:"cid"`
	MIMEType        string    `gorm:"size:255;not null" json:"mime_type"`
	Size            int64     `gorm:"not null" json:"size"`
	Width           *int      `json:"width,omitempty"`
	Height          *int      `json:"height,omitempty"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	FileName        string    `gorm:"size:512" json:"file_name"`
	ThumbnailCID    string    `gorm:"column:thumbnail_c_id;size:255" json:"thumbnail_cid"`
	CreatedByUserID uint64    `gorm:"not null;index" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
}
