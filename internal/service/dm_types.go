package service

import "time"

// DMConversationView 上游 DM 会话列表项（供 mesh-proxy 使用）。
type DMConversationView struct {
	ConversationID string    `json:"conversation_id"`
	PeerID         string    `json:"peer_id"`
	LastMessageSeq uint64    `json:"last_message_seq"`
	LastMessageAt  time.Time `json:"last_message_at"`
}

// DMMessageView 上游 DM 消息。
type DMMessageView struct {
	MessageID        string     `json:"message_id"`
	ConversationID   string     `json:"conversation_id"`
	Seq              uint64     `json:"seq"`
	ContentType      string     `json:"content_type"`
	Payload          any        `json:"payload"`
	SenderUserID     uint64     `json:"sender_user_id"`
	RecipientUserID  uint64     `json:"recipient_user_id"`
	SenderPeerID     string     `json:"sender_peer_id"`
	RecipientPeerID  string     `json:"recipient_peer_id"`
	ClientMsgID      string     `json:"client_msg_id"`
	Status           string     `json:"status"`
	RecipientAckedAt *time.Time `json:"recipient_acked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// CreateDMConversationInput 与对端 peer 建立或返回已有会话。
type CreateDMConversationInput struct {
	PeerID string `json:"peer_id"`
}

// SendDMMessageInput 发送文本（首期仅 text）。
type SendDMMessageInput struct {
	ClientMsgID string `json:"client_msg_id"`
	ContentType string `json:"content_type"`
	Payload     any    `json:"payload"`
}
