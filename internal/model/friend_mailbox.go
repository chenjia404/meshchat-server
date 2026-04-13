package model

import "time"

// 好友信箱状态（与 mesh-proxy 请求 state 语义对齐，仅服务端存储）。
const (
	FriendMailboxStatePending   = "pending"
	FriendMailboxStateAccepted  = "accepted"
	FriendMailboxStateRejected  = "rejected"
)

// FriendMailboxRequest 为公网补充通道：缓存「加好友意向」，不替代 mesh-proxy 的 SessionRequest/加密握手。
type FriendMailboxRequest struct {
	ID          uint64    `gorm:"primaryKey" json:"-"`
	RequestID   string    `gorm:"size:36;uniqueIndex;not null" json:"request_id"`
	FromPeerID  string    `gorm:"size:255;not null;index:idx_friend_mailbox_from" json:"from_peer_id"`
	ToPeerID    string    `gorm:"size:255;not null;index:idx_friend_mailbox_to" json:"to_peer_id"`
	State       string    `gorm:"size:32;not null" json:"state"`
	IntroText   string    `gorm:"type:text;not null" json:"intro_text"`
	Nickname    string    `gorm:"size:128" json:"nickname"`
	Bio         string    `gorm:"size:1024" json:"bio"`
	AvatarCID   string    `gorm:"column:avatar_cid;size:255" json:"avatar_cid"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (FriendMailboxRequest) TableName() string {
	return "friend_mailbox_requests"
}
