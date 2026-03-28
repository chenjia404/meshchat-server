package service

import "time"

// PublicUser is the user shape exposed to clients.
type PublicUser struct {
	ID             uint64    `json:"id"`
	PeerID         string    `json:"peer_id,omitempty"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"display_name"`
	AvatarCID      string    `json:"avatar_cid,omitempty"`
	Bio            string    `json:"bio,omitempty"`
	ProfileVersion uint64    `json:"profile_version"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GroupView struct {
	GroupID                string     `json:"group_id"`
	Title                  string     `json:"title"`
	About                  string     `json:"about,omitempty"`
	AvatarCID              string     `json:"avatar_cid,omitempty"`
	OwnerUser              PublicUser `json:"owner_user"`
	MemberListVisibility   string     `json:"member_list_visibility"`
	JoinMode               string     `json:"join_mode"`
	DefaultPermissions     int64      `json:"default_permissions"`
	MessageTTLSeconds      int64      `json:"message_ttl_seconds"`
	MessageRetractSeconds  int64      `json:"message_retract_seconds"`
	MessageCooldownSeconds int64      `json:"message_cooldown_seconds"`
	LastMessageSeq         uint64     `json:"last_message_seq"`
	SettingsVersion        uint64     `json:"settings_version"`
	Status                 string     `json:"status"`
	EffectivePermissions   int64      `json:"effective_permissions"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type GroupMemberView struct {
	User                 PublicUser `json:"user"`
	Role                 string     `json:"role"`
	Status               string     `json:"status"`
	Title                string     `json:"title,omitempty"`
	JoinedAt             time.Time  `json:"joined_at"`
	MutedUntil           *time.Time `json:"muted_until,omitempty"`
	PermissionsAllow     int64      `json:"permissions_allow"`
	PermissionsDeny      int64      `json:"permissions_deny"`
	EffectivePermissions int64      `json:"effective_permissions"`
}

type JoinGroupInput struct {
	// Reserved for future expansion, such as invite tokens.
}

type InviteGroupMemberInput struct {
	UserID uint64 `json:"user_id"`
}

type InviteGroupMembersInput struct {
	PeerIDs []string `json:"peer_ids"`
}

type ForwardReferenceView struct {
	State   string          `json:"state"`
	Message *MessageSummary `json:"message,omitempty"`
	Notice  string          `json:"notice,omitempty"`
}

type MessageSummary struct {
	GroupID     string     `json:"group_id"`
	MessageID   string     `json:"message_id"`
	Seq         uint64     `json:"seq"`
	ContentType string     `json:"content_type"`
	Payload     any        `json:"payload,omitempty"`
	Sender      PublicUser `json:"sender"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

type MessageView struct {
	GroupID              string                `json:"group_id"`
	MessageID            string                `json:"message_id"`
	Seq                  uint64                `json:"seq"`
	ContentType          string                `json:"content_type"`
	Payload              any                   `json:"payload,omitempty"`
	ReplyToMessageID     *string               `json:"reply_to_message_id,omitempty"`
	ForwardFromMessageID *string               `json:"forward_from_message_id,omitempty"`
	Forward              *ForwardReferenceView `json:"forward,omitempty"`
	Sender               PublicUser            `json:"sender"`
	Status               string                `json:"status"`
	EditCount            uint32                `json:"edit_count"`
	LastEditedAt         *time.Time            `json:"last_edited_at,omitempty"`
	DeleteReason         *string               `json:"delete_reason,omitempty"`
	DeletedAt            *time.Time            `json:"deleted_at,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
}

type ChallengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	Challenge   string    `json:"challenge"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type LoginResponse struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

type AdminLoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

type AdminMeView struct {
	Username string `json:"username"`
}

type AdminUserView struct {
	PeerID string `json:"peer_id"`
	PublicUser
}

type ServerInfoView struct {
	ServerMode string `json:"server_mode"`
}

type TextPayload struct {
	Text string `json:"text"`
}

type ImagePayload struct {
	CID          string `json:"cid"`
	MIMEType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Caption      string `json:"caption,omitempty"`
	ThumbnailCID string `json:"thumbnail_cid,omitempty"`
}

type VideoPayload struct {
	CID          string `json:"cid"`
	MIMEType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     int    `json:"duration"`
	Caption      string `json:"caption,omitempty"`
	ThumbnailCID string `json:"thumbnail_cid,omitempty"`
}

type VoicePayload struct {
	CID      string `json:"cid"`
	MIMEType string `json:"mime_type"`
	Size     int64  `json:"size"`
	Duration int    `json:"duration"`
	Waveform string `json:"waveform,omitempty"`
}

type FilePayload struct {
	CID      string `json:"cid"`
	MIMEType string `json:"mime_type"`
	Size     int64  `json:"size"`
	FileName string `json:"file_name"`
	Caption  string `json:"caption,omitempty"`
}

type ForwardPayload struct {
	Comment string `json:"comment,omitempty"`
}

type UpdateProfileInput struct {
	DisplayName string `json:"display_name"`
	AvatarCID   string `json:"avatar_cid"`
	Bio         string `json:"bio"`
	Status      string `json:"status"`
}

type CreateGroupInput struct {
	Title                string `json:"title"`
	About                string `json:"about"`
	AvatarCID            string `json:"avatar_cid"`
	MemberListVisibility string `json:"member_list_visibility"`
	JoinMode             string `json:"join_mode"`
	DefaultPermissions   int64  `json:"default_permissions"`
}

type UpdateGroupInput struct {
	Title                string `json:"title"`
	About                string `json:"about"`
	AvatarCID            string `json:"avatar_cid"`
	MemberListVisibility string `json:"member_list_visibility"`
	JoinMode             string `json:"join_mode"`
	Status               string `json:"status"`
}

type UpdateMessagePolicyInput struct {
	MessageTTLSeconds      int64 `json:"message_ttl_seconds"`
	MessageRetractSeconds  int64 `json:"message_retract_seconds"`
	MessageCooldownSeconds int64 `json:"message_cooldown_seconds"`
}

type UpdateMemberPermissionsInput struct {
	PermissionsAllow int64 `json:"permissions_allow"`
	PermissionsDeny  int64 `json:"permissions_deny"`
}

type SetGroupAdminInput struct {
	IsAdmin bool `json:"is_admin"`
}

type TransferGroupOwnershipInput struct {
	UserID uint64 `json:"user_id"`
}

type MuteMemberInput struct {
	DurationSeconds int64 `json:"duration_seconds"`
}

type SendMessageInput struct {
	ContentType          string  `json:"content_type"`
	Payload              any     `json:"payload"`
	ReplyToMessageID     *string `json:"reply_to_message_id"`
	ForwardFromMessageID *string `json:"forward_from_message_id"`
	Signature            string  `json:"signature"`
}

type EditMessageInput struct {
	Payload any `json:"payload"`
}

type RegisterFileInput struct {
	CID             string `json:"cid"`
	MIMEType        string `json:"mime_type"`
	Size            int64  `json:"size"`
	Width           *int   `json:"width"`
	Height          *int   `json:"height"`
	DurationSeconds *int   `json:"duration_seconds"`
	FileName        string `json:"file_name"`
	ThumbnailCID    string `json:"thumbnail_cid"`
}

type RealtimeEnvelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type GroupLifecycleView struct {
	GroupID         string    `json:"group_id"`
	Status          string    `json:"status"`
	SettingsVersion uint64    `json:"settings_version"`
	UpdatedAt       time.Time `json:"updated_at"`
}
