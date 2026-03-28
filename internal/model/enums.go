package model

import "math"

const (
	RoleOwner      = "owner"
	RoleAdmin      = "admin"
	RoleMember     = "member"
	RoleRestricted = "restricted"
)

const (
	MemberStatusActive = "active"
	MemberStatusLeft   = "left"
	MemberStatusKicked = "kicked"
	MemberStatusBanned = "banned"
)

const (
	GroupStatusActive = "active"
	GroupStatusClosed = "closed"
)

const (
	MemberListVisibilityVisible = "visible"
	MemberListVisibilityHidden  = "hidden"
)

const (
	JoinModeInviteOnly = "invite_only"
	JoinModeOpen       = "open"
)

const (
	MessageContentTypeText    = "text"
	MessageContentTypeImage   = "image"
	MessageContentTypeVideo   = "video"
	MessageContentTypeVoice   = "voice"
	MessageContentTypeFile    = "file"
	MessageContentTypeForward = "forward"
)

const (
	MessageStatusNormal  = "normal"
	MessageStatusDeleted = "deleted"
)

const (
	DeleteReasonSelfRetracted = "self_retracted"
	DeleteReasonAdminRemoved  = "admin_removed"
)

const (
	PermSendText int64 = 1 << iota
	PermSendImage
	PermSendVideo
	PermSendVoice
	PermSendFile
	PermReply
	PermForward
	PermEditOwnMessages
	PermDeleteOwnMessages
	PermDeleteMessages
	PermViewMembers
	PermBypassSlowmode
	PermManageMessagePolicy
	PermEditGroupInfo
	PermMuteMembers
	PermBanMembers
	PermSetMemberPermissions
)

const AllPermissions int64 = math.MaxInt64

const DefaultMemberPermissions = PermSendText |
	PermSendImage |
	PermSendVideo |
	PermSendVoice |
	PermSendFile |
	PermReply |
	PermForward |
	PermEditOwnMessages |
	PermDeleteOwnMessages |
	PermViewMembers

// RoleRank returns a sortable privilege ranking.
func RoleRank(role string) int {
	switch role {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleMember:
		return 2
	case RoleRestricted:
		return 1
	default:
		return 0
	}
}

// BasePermissions returns the baseline permissions granted by a role.
func BasePermissions(role string, groupDefault int64) int64 {
	switch role {
	case RoleOwner:
		return AllPermissions
	case RoleAdmin:
		return groupDefault |
			PermDeleteMessages |
			PermViewMembers |
			PermBypassSlowmode |
			PermManageMessagePolicy |
			PermEditGroupInfo |
			PermMuteMembers |
			PermBanMembers |
			PermSetMemberPermissions
	case RoleMember:
		return groupDefault
	case RoleRestricted:
		return 0
	default:
		return 0
	}
}

// EffectivePermissions calculates the final permission bitset for a member.
func EffectivePermissions(role string, groupDefault, allow, deny int64) int64 {
	if role == RoleOwner {
		return AllPermissions
	}
	return (BasePermissions(role, groupDefault) | allow) &^ deny
}
