package service

type PublicChannelAvatar struct {
	FileName string `json:"file_name,omitempty"`
	MIMEType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	BlobID   string `json:"blob_id,omitempty"`
	URL      string `json:"url,omitempty"`
}

type PublicChannelFile struct {
	FileID   string `json:"file_id,omitempty"`
	FileName string `json:"file_name"`
	MIMEType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	BlobID   string `json:"blob_id,omitempty"`
	URL      string `json:"url,omitempty"`
}

type PublicChannelMessageContent struct {
	Text  string              `json:"text,omitempty"`
	Files []PublicChannelFile `json:"files,omitempty"`
}

type PublicChannelProfileView struct {
	ChannelID               string              `json:"channel_id"`
	OwnerPeerID             string              `json:"owner_peer_id"`
	OwnerVersion            int64               `json:"owner_version"`
	Name                    string              `json:"name"`
	Avatar                  PublicChannelAvatar `json:"avatar"`
	Bio                     string              `json:"bio"`
	MessageRetentionMinutes int                 `json:"message_retention_minutes"`
	ProfileVersion          int64               `json:"profile_version"`
	CreatedAt               int64               `json:"created_at"`
	UpdatedAt               int64               `json:"updated_at"`
	Signature               string              `json:"signature"`
}

type PublicChannelHeadView struct {
	ChannelID      string `json:"channel_id"`
	OwnerPeerID    string `json:"owner_peer_id"`
	OwnerVersion   int64  `json:"owner_version"`
	LastMessageID  int64  `json:"last_message_id"`
	ProfileVersion int64  `json:"profile_version"`
	LastSeq        int64  `json:"last_seq"`
	UpdatedAt      int64  `json:"updated_at"`
	Signature      string `json:"signature"`
}

type PublicChannelMessageView struct {
	ChannelID     string                      `json:"channel_id"`
	MessageID     int64                       `json:"message_id"`
	Version       int64                       `json:"version"`
	Seq           int64                       `json:"seq"`
	OwnerVersion  int64                       `json:"owner_version"`
	CreatorPeerID string                      `json:"creator_peer_id"`
	AuthorPeerID  string                      `json:"author_peer_id"`
	CreatedAt     int64                       `json:"created_at"`
	UpdatedAt     int64                       `json:"updated_at"`
	IsDeleted     bool                        `json:"is_deleted"`
	MessageType   string                      `json:"message_type"`
	Content       PublicChannelMessageContent `json:"content"`
	Signature     string                      `json:"signature"`
}

type PublicChannelChangeView struct {
	ChannelID      string `json:"channel_id"`
	Seq            int64  `json:"seq"`
	ChangeType     string `json:"change_type"`
	MessageID      *int64 `json:"message_id,omitempty"`
	Version        *int64 `json:"version,omitempty"`
	IsDeleted      *bool  `json:"is_deleted,omitempty"`
	ProfileVersion *int64 `json:"profile_version,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	ProviderPeerID string `json:"provider_peer_id,omitempty"`
}

type PublicChannelSyncStateView struct {
	ChannelID             string `json:"channel_id"`
	LastSeenSeq           int64  `json:"last_seen_seq"`
	LastSyncedSeq         int64  `json:"last_synced_seq"`
	LatestLoadedMessageID int64  `json:"latest_loaded_message_id"`
	OldestLoadedMessageID int64  `json:"oldest_loaded_message_id"`
	UnreadCount           int    `json:"unread_count"`
	Subscribed            bool   `json:"subscribed"`
	UpdatedAt             int64  `json:"updated_at"`
}

type PublicChannelProviderView struct {
	ChannelID     string `json:"channel_id"`
	PeerID        string `json:"peer_id"`
	Source        string `json:"source"`
	UpdatedAt     int64  `json:"updated_at"`
	LastSuccessAt int64  `json:"last_success_at"`
	LastFailureAt int64  `json:"last_failure_at"`
	SuccessCount  int64  `json:"success_count"`
	FailureCount  int64  `json:"failure_count"`
}

type PublicChannelSummaryView struct {
	Profile PublicChannelProfileView   `json:"profile"`
	Head    PublicChannelHeadView      `json:"head"`
	Sync    PublicChannelSyncStateView `json:"sync"`
}

type PublicChannelGetChangesResponse struct {
	ChannelID      string                    `json:"channel_id"`
	CurrentLastSeq int64                     `json:"current_last_seq"`
	HasMore        bool                      `json:"has_more"`
	NextAfterSeq   int64                     `json:"next_after_seq"`
	Items          []PublicChannelChangeView `json:"items"`
}

type PublicChannelSubscribeResult struct {
	Profile   PublicChannelProfileView   `json:"profile"`
	Head      PublicChannelHeadView      `json:"head"`
	Messages  []PublicChannelMessageView `json:"messages"`
	Providers []PublicChannelProviderView `json:"providers,omitempty"`
}

type CreatePublicChannelInput struct {
	Name                    string              `json:"name"`
	Bio                     string              `json:"bio"`
	Avatar                  PublicChannelAvatar `json:"avatar"`
	MessageRetentionMinutes int                 `json:"message_retention_minutes"`
}

type UpdatePublicChannelProfileInput struct {
	Name                    string               `json:"name"`
	Bio                     string               `json:"bio"`
	Avatar                  PublicChannelAvatar  `json:"avatar"`
	MessageRetentionMinutes *int                 `json:"message_retention_minutes,omitempty"`
}

type UpsertPublicChannelMessageInput struct {
	MessageType string              `json:"message_type,omitempty"`
	Text        string              `json:"text"`
	Files       []PublicChannelFile `json:"files"`
}

type SubscribePublicChannelInput struct {
	LastSeenSeq int64 `json:"last_seen_seq"`
}
