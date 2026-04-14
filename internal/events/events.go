package events

import "time"

const (
	EventGroupMessageCreated  = "group.message.created"
	EventGroupMessageEdited   = "group.message.edited"
	EventGroupMessageDeleted  = "group.message.deleted"
	EventGroupSettingsUpdated = "group.settings.updated"
	EventGroupMemberUpdated   = "group.member.updated"

	EventDMMessageCreated = "dm.message.created"
	EventDMMessageAcked   = "dm.message.acked"

	EventPublicChannelProfileUpdated = "publicchannel.profile.updated"
	EventPublicChannelMessageCreated = "publicchannel.message.created"
	EventPublicChannelMessageUpdated = "publicchannel.message.updated"
	EventPublicChannelMessageDeleted = "publicchannel.message.deleted"
)

// Envelope is the cross-instance event format transported via Redis Pub/Sub.
type Envelope struct {
	Type           string    `json:"type"`
	GroupID        string    `json:"group_id,omitempty"`
	ConversationID string    `json:"conversation_id,omitempty"`
	ChannelID      string    `json:"channel_id,omitempty"`
	MessageID      string    `json:"message_id,omitempty"`
	UserID         uint64    `json:"user_id,omitempty"`
	At             time.Time `json:"at"`
}
