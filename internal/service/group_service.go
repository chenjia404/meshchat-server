package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"meshchat-server/internal/events"
	"meshchat-server/internal/ipfs"
	"meshchat-server/internal/model"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventPublisher interface {
	Publish(ctx context.Context, event events.Envelope) error
}

type GroupService struct {
	groups    *repo.GroupRepo
	users     *repo.UserRepo
	publisher EventPublisher
	ipfs      ipfs.Client
	admins    *ServerAdminService
	mode      string
}

func NewGroupService(groups *repo.GroupRepo, users *repo.UserRepo, publisher EventPublisher, ipfs ipfs.Client, admins *ServerAdminService, mode string) *GroupService {
	return &GroupService{
		groups:    groups,
		users:     users,
		publisher: publisher,
		ipfs:      ipfs,
		admins:    admins,
		mode:      strings.ToLower(strings.TrimSpace(mode)),
	}
}

func (s *GroupService) CreateGroup(ctx context.Context, userID uint64, input CreateGroupInput) (*GroupView, error) {
	if err := s.requireCreateGroupPermission(ctx, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, apperrors.New(400, "invalid_title", "title is required")
	}
	if input.AvatarCID != "" {
		if err := s.ipfs.ValidateCID(input.AvatarCID); err != nil {
			return nil, payloadValidationError("avatar_cid is invalid")
		}
	}
	if input.MemberListVisibility == "" {
		input.MemberListVisibility = model.MemberListVisibilityVisible
	}
	if input.JoinMode == "" {
		input.JoinMode = model.JoinModeInviteOnly
	}
	if input.DefaultPermissions == 0 {
		input.DefaultPermissions = model.DefaultMemberPermissions
	}

	group := &model.Group{
		GroupID:                uuid.New(),
		Title:                  strings.TrimSpace(input.Title),
		About:                  strings.TrimSpace(input.About),
		AvatarCID:              strings.TrimSpace(input.AvatarCID),
		OwnerUserID:            userID,
		MemberListVisibility:   input.MemberListVisibility,
		JoinMode:               input.JoinMode,
		DefaultPermissions:     input.DefaultPermissions,
		MessageTTLSeconds:      0,
		MessageRetractSeconds:  300,
		MessageCooldownSeconds: 0,
		SettingsVersion:        1,
		Status:                 model.GroupStatusActive,
	}
	member := &model.GroupMember{
		UserID:           userID,
		Role:             model.RoleOwner,
		Status:           model.MemberStatusActive,
		JoinedAt:         time.Now().UTC(),
		PermissionsAllow: 0,
		PermissionsDeny:  0,
	}

	if err := s.groups.CreateWithOwner(ctx, group, member); err != nil {
		return nil, err
	}

	view, err := s.GetGroup(ctx, userID, group.GroupID.String())
	if err != nil {
		return nil, err
	}
	return view, nil
}

func (s *GroupService) GetGroup(ctx context.Context, userID uint64, groupID string) (*GroupView, error) {
	group, member, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	view := s.toGroupView(*group, *member)
	return &view, nil
}

func (s *GroupService) JoinGroup(ctx context.Context, userID uint64, groupID string) (*GroupMemberView, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
	}
	if group.Status != model.GroupStatusActive {
		return nil, apperrors.New(403, "group_closed", "group is closed")
	}
	if group.JoinMode != model.JoinModeOpen {
		return nil, apperrors.New(403, "join_not_allowed", "group does not allow open join")
	}

	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedGroup, err := s.groups.GetByIDForUpdate(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		if lockedGroup.Status != model.GroupStatusActive {
			return apperrors.New(403, "group_closed", "group is closed")
		}
		if lockedGroup.JoinMode != model.JoinModeOpen {
			return apperrors.New(403, "join_not_allowed", "group does not allow open join")
		}

		existing, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, userID)
		if err == nil {
			if existing.Status == model.MemberStatusBanned {
				return apperrors.New(403, "member_banned", "member is banned")
			}
			if existing.Status == model.MemberStatusActive {
				return nil
			}
			existing.Role = model.RoleMember
			existing.Status = model.MemberStatusActive
			existing.JoinedAt = time.Now().UTC()
			existing.MutedUntil = nil
			existing.PermissionsAllow = 0
			existing.PermissionsDeny = 0
			if err := tx.WithContext(ctx).Model(existing).Updates(map[string]any{
				"role":              existing.Role,
				"status":            existing.Status,
				"joined_at":         existing.JoinedAt,
				"muted_until":       existing.MutedUntil,
				"permissions_allow": existing.PermissionsAllow,
				"permissions_deny":  existing.PermissionsDeny,
			}).Error; err != nil {
				return err
			}
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		newMember := &model.GroupMember{
			GroupID:          lockedGroup.ID,
			UserID:           userID,
			Role:             model.RoleMember,
			Status:           model.MemberStatusActive,
			JoinedAt:         time.Now().UTC(),
			PermissionsAllow: 0,
			PermissionsDeny:  0,
		}
		if err := tx.WithContext(ctx).Create(newMember).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	loaded, err := s.groups.GetMember(ctx, group.ID, userID)
	if err != nil {
		return nil, err
	}
	view := s.toMemberView(*group, *loaded)
	return &view, nil
}

func (s *GroupService) InviteMember(ctx context.Context, userID uint64, groupID string, targetUserID uint64) (*GroupMemberView, error) {
	if targetUserID == 0 {
		return nil, apperrors.New(400, "invalid_user_id", "target user_id is required")
	}
	targetUser, err := s.users.GetByID(ctx, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "user_not_found", "target user not found")
		}
		return nil, err
	}
	views, err := s.InviteMembersByPeerIDs(ctx, userID, groupID, InviteGroupMembersInput{PeerIDs: []string{targetUser.PeerID}})
	if err != nil {
		return nil, err
	}
	if len(views) == 0 {
		return nil, apperrors.New(404, "member_not_found", "target member not found")
	}
	return &views[0], nil
}

func (s *GroupService) InviteMembersByPeerIDs(ctx context.Context, userID uint64, groupID string, input InviteGroupMembersInput) ([]GroupMemberView, error) {
	peerIDs := uniqueNonBlankStrings(input.PeerIDs)
	if len(peerIDs) == 0 {
		return nil, apperrors.New(400, "invalid_peer_ids", "peer_ids is required")
	}

	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
	}
	if group.Status != model.GroupStatusActive {
		return nil, apperrors.New(403, "group_closed", "group is closed")
	}

	isServerAdmin, err := s.admins.IsServerAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}

	inviter, err := s.groups.GetMember(ctx, group.ID, userID)
	if err != nil && !isServerAdmin {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(403, "not_group_member", "user is not an active group member")
		}
		return nil, err
	}
	if !isServerAdmin {
		if inviter.Status != model.MemberStatusActive {
			return nil, apperrors.New(403, "member_inactive", "member is not active in this group")
		}
		if inviter.Role != model.RoleOwner && inviter.Role != model.RoleAdmin {
			return nil, apperrors.New(403, "forbidden", "only group owners, admins or server admins can invite members")
		}
	}

	invitedIDs := make([]uint64, 0, len(peerIDs))
	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedGroup, err := s.groups.GetByIDForUpdate(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		if lockedGroup.Status != model.GroupStatusActive {
			return apperrors.New(403, "group_closed", "group is closed")
		}
		if !isServerAdmin {
			lockedInviter, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, userID)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return apperrors.New(403, "not_group_member", "user is not an active group member")
				}
				return err
			}
			if lockedInviter.Status != model.MemberStatusActive {
				return apperrors.New(403, "member_inactive", "member is not active in this group")
			}
			if lockedInviter.Role != model.RoleOwner && lockedInviter.Role != model.RoleAdmin {
				return apperrors.New(403, "forbidden", "only group owners, admins or server admins can invite members")
			}
		}

		for _, peerID := range peerIDs {
			targetUser, created, err := s.ensurePeerUserTx(ctx, tx, peerID)
			if err != nil {
				return err
			}
			if targetUser.ID == userID {
				return apperrors.New(400, "invalid_user_id", "cannot invite yourself")
			}

			targetMember, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, targetUser.ID)
			if err == nil {
				switch targetMember.Status {
				case model.MemberStatusBanned:
					return apperrors.New(403, "member_banned", "member is banned")
				case model.MemberStatusActive:
					invitedIDs = append(invitedIDs, targetUser.ID)
				default:
					targetMember.Role = model.RoleMember
					targetMember.Status = model.MemberStatusActive
					targetMember.JoinedAt = time.Now().UTC()
					targetMember.MutedUntil = nil
					targetMember.PermissionsAllow = 0
					targetMember.PermissionsDeny = 0
					if err := tx.WithContext(ctx).Model(targetMember).Updates(map[string]any{
						"role":              targetMember.Role,
						"status":            targetMember.Status,
						"joined_at":         targetMember.JoinedAt,
						"muted_until":       targetMember.MutedUntil,
						"permissions_allow": targetMember.PermissionsAllow,
						"permissions_deny":  targetMember.PermissionsDeny,
					}).Error; err != nil {
						return err
					}
					invitedIDs = append(invitedIDs, targetUser.ID)
				}
				_ = created
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}

			targetMember = &model.GroupMember{
				GroupID:          lockedGroup.ID,
				UserID:           targetUser.ID,
				Role:             model.RoleMember,
				Status:           model.MemberStatusActive,
				JoinedAt:         time.Now().UTC(),
				PermissionsAllow: 0,
				PermissionsDeny:  0,
			}
			if err := tx.WithContext(ctx).Create(targetMember).Error; err != nil {
				return err
			}
			invitedIDs = append(invitedIDs, targetUser.ID)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	reloadedGroup, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	views := make([]GroupMemberView, 0, len(invitedIDs))
	for _, invitedID := range invitedIDs {
		member, err := s.groups.GetMember(ctx, reloadedGroup.ID, invitedID)
		if err != nil {
			return nil, err
		}
		views = append(views, s.toMemberView(*reloadedGroup, *member))
		_ = s.publisher.Publish(ctx, events.Envelope{
			Type:    events.EventGroupMemberUpdated,
			GroupID: reloadedGroup.GroupID.String(),
			UserID:  invitedID,
			At:      time.Now().UTC(),
		})
	}
	return views, nil
}

func (s *GroupService) ensurePeerUserTx(ctx context.Context, tx *gorm.DB, peerID string) (*model.ServerUser, bool, error) {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return nil, false, apperrors.New(400, "invalid_peer_id", "peer_id is required")
	}

	var user model.ServerUser
	if err := tx.WithContext(ctx).First(&user, "peer_id = ?", peerID).Error; err == nil {
		return &user, false, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	username := buildBootstrapUsername(peerID)
	user = model.ServerUser{
		PeerID:         peerID,
		PublicKey:      fmt.Sprintf("invited:%s", peerID),
		Username:       username,
		DisplayName:    username,
		ProfileVersion: 1,
		Status:         "active",
	}
	if err := tx.WithContext(ctx).Create(&user).Error; err != nil {
		if duplicateConstraintError(err) {
			user.Username = fmt.Sprintf("%s_%s", username, uuid.NewString()[:8])
			user.DisplayName = user.Username
			if err := tx.WithContext(ctx).Create(&user).Error; err != nil {
				return nil, false, err
			}
		} else {
			return nil, false, err
		}
	}
	return &user, true, nil
}

func uniqueNonBlankStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *GroupService) LeaveGroup(ctx context.Context, userID uint64, groupID string) (*GroupMemberView, error) {
	group, member, err := s.requireMembership(ctx, userID, groupID, true)
	if err != nil {
		return nil, err
	}
	if member.Status != model.MemberStatusActive {
		return nil, apperrors.New(403, "member_inactive", "member is not active in this group")
	}
	if group.OwnerUserID == userID || member.Role == model.RoleOwner {
		return nil, apperrors.New(403, "forbidden", "group owner must transfer ownership before leaving")
	}

	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedGroup, err := s.groups.GetByIDForUpdate(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		lockedMember, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, userID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(403, "not_group_member", "user is not an active group member")
			}
			return err
		}
		if lockedMember.Status != model.MemberStatusActive {
			return apperrors.New(403, "member_inactive", "member is not active in this group")
		}
		if lockedGroup.OwnerUserID == userID || lockedMember.Role == model.RoleOwner {
			return apperrors.New(403, "forbidden", "group owner must transfer ownership before leaving")
		}

		lockedMember.Status = model.MemberStatusLeft
		lockedMember.MutedUntil = nil
		if err := tx.WithContext(ctx).Model(lockedMember).Updates(map[string]any{
			"status":      lockedMember.Status,
			"muted_until": lockedMember.MutedUntil,
		}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	leftMember, err := s.groups.GetMember(ctx, group.ID, userID)
	if err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  userID,
		At:      time.Now().UTC(),
	})
	view := s.toMemberView(*group, *leftMember)
	return &view, nil
}

func (s *GroupService) DissolveGroup(ctx context.Context, userID uint64, groupID string) (*GroupLifecycleView, error) {
	if err := s.admins.RequireServerAdmin(ctx, userID); err != nil {
		return nil, err
	}

	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
	}

	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := s.groups.GetByIDForUpdate(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		if locked.Status == model.GroupStatusClosed {
			group = locked
			return nil
		}
		locked.Status = model.GroupStatusClosed
		locked.SettingsVersion++
		group = locked
		return tx.WithContext(ctx).Model(locked).Updates(map[string]any{
			"status":           locked.Status,
			"settings_version": locked.SettingsVersion,
		}).Error
	}); err != nil {
		return nil, err
	}
	group, err = s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupSettingsUpdated,
		GroupID: group.GroupID.String(),
		At:      time.Now().UTC(),
	})

	return &GroupLifecycleView{
		GroupID:         group.GroupID.String(),
		Status:          group.Status,
		SettingsVersion: group.SettingsVersion,
		UpdatedAt:       group.UpdatedAt,
	}, nil
}

func (s *GroupService) TransferOwnership(ctx context.Context, userID uint64, groupID string, input TransferGroupOwnershipInput) (*GroupView, error) {
	group, operator, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if operator.Role != model.RoleOwner || group.OwnerUserID != userID {
		return nil, apperrors.New(403, "forbidden", "only the current group owner can transfer ownership")
	}
	if input.UserID == 0 {
		return nil, apperrors.New(400, "invalid_user_id", "target user_id is required")
	}
	if input.UserID == userID {
		view := s.toGroupView(*group, *operator)
		return &view, nil
	}

	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedGroup, err := s.groups.GetByIDForUpdate(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		if lockedGroup.OwnerUserID != userID {
			return apperrors.New(409, "owner_changed", "group owner changed, please retry")
		}

		currentOwner, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, userID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(404, "member_not_found", "current owner membership not found")
			}
			return err
		}
		target, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, input.UserID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(404, "member_not_found", "target member not found")
			}
			return err
		}
		if target.Status != model.MemberStatusActive {
			return apperrors.New(400, "member_inactive", "only active members can become group owner")
		}

		currentOwner.Role = model.RoleAdmin
		target.Role = model.RoleOwner
		lockedGroup.OwnerUserID = target.UserID
		lockedGroup.SettingsVersion++

		if err := tx.WithContext(ctx).Model(currentOwner).Update("role", currentOwner.Role).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(target).Update("role", target.Role).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(lockedGroup).Updates(map[string]any{
			"owner_user_id":    lockedGroup.OwnerUserID,
			"settings_version": lockedGroup.SettingsVersion,
		}).Error; err != nil {
			return err
		}

		group = lockedGroup
		return nil
	}); err != nil {
		return nil, err
	}

	group, operator, err = s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupSettingsUpdated,
		GroupID: group.GroupID.String(),
		At:      now,
	})
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  userID,
		At:      now,
	})
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  input.UserID,
		At:      now,
	})

	view := s.toGroupView(*group, *operator)
	return &view, nil
}

func (s *GroupService) UpdateGroup(ctx context.Context, userID uint64, groupID string, input UpdateGroupInput) (*GroupView, error) {
	group, member, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if !hasPermission(model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny), model.PermEditGroupInfo) {
		return nil, apperrors.New(403, "forbidden", "missing permission to edit group info")
	}
	if input.AvatarCID != "" {
		if err := s.ipfs.ValidateCID(input.AvatarCID); err != nil {
			return nil, payloadValidationError("avatar_cid is invalid")
		}
	}

	if strings.TrimSpace(input.Title) != "" {
		group.Title = strings.TrimSpace(input.Title)
	}
	group.About = strings.TrimSpace(input.About)
	group.AvatarCID = strings.TrimSpace(input.AvatarCID)
	if input.MemberListVisibility != "" {
		group.MemberListVisibility = input.MemberListVisibility
	}
	if input.JoinMode != "" {
		group.JoinMode = input.JoinMode
	}
	if input.Status != "" {
		group.Status = input.Status
	}
	group.SettingsVersion++

	if err := s.groups.Update(ctx, group); err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupSettingsUpdated,
		GroupID: group.GroupID.String(),
		At:      time.Now().UTC(),
	})

	view := s.toGroupView(*group, *member)
	return &view, nil
}

func (s *GroupService) UpdateMessagePolicy(ctx context.Context, userID uint64, groupID string, input UpdateMessagePolicyInput) (*GroupView, error) {
	group, member, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	perms := model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny)
	if !hasPermission(perms, model.PermManageMessagePolicy) {
		return nil, apperrors.New(403, "forbidden", "missing permission to manage message policy")
	}

	if input.MessageTTLSeconds < 0 || input.MessageRetractSeconds < 0 || input.MessageCooldownSeconds < 0 {
		return nil, apperrors.New(400, "invalid_policy", "policy values must be non-negative")
	}

	group.MessageTTLSeconds = input.MessageTTLSeconds
	group.MessageRetractSeconds = input.MessageRetractSeconds
	group.MessageCooldownSeconds = input.MessageCooldownSeconds
	group.SettingsVersion++

	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := s.groups.GetByIDForUpdate(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		locked.MessageTTLSeconds = group.MessageTTLSeconds
		locked.MessageRetractSeconds = group.MessageRetractSeconds
		locked.MessageCooldownSeconds = group.MessageCooldownSeconds
		locked.SettingsVersion++
		group.SettingsVersion = locked.SettingsVersion
		return s.groups.UpdateMessagePolicyTx(ctx, tx, locked)
	}); err != nil {
		return nil, err
	}

	group, member, err = s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupSettingsUpdated,
		GroupID: group.GroupID.String(),
		At:      time.Now().UTC(),
	})

	view := s.toGroupView(*group, *member)
	return &view, nil
}

func (s *GroupService) ListMembers(ctx context.Context, userID uint64, groupID string) ([]GroupMemberView, error) {
	group, member, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	perms := model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny)
	if group.MemberListVisibility == model.MemberListVisibilityHidden && !hasPermission(perms, model.PermViewMembers) {
		return nil, apperrors.New(403, "members_hidden", "member list is hidden")
	}

	members, err := s.groups.ListMembers(ctx, group.ID)
	if err != nil {
		return nil, err
	}

	views := make([]GroupMemberView, 0, len(members))
	for _, item := range members {
		views = append(views, s.toMemberView(*group, item))
	}
	return views, nil
}

func (s *GroupService) UpdateMemberPermissions(ctx context.Context, userID uint64, groupID string, targetUserID uint64, input UpdateMemberPermissionsInput) (*GroupMemberView, error) {
	group, operator, target, err := s.requireManageableTarget(ctx, userID, groupID, targetUserID, model.PermSetMemberPermissions)
	if err != nil {
		return nil, err
	}

	target.PermissionsAllow = input.PermissionsAllow
	target.PermissionsDeny = input.PermissionsDeny
	if err := s.groups.UpdateMember(ctx, target); err != nil {
		return nil, err
	}
	_ = operator

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  targetUserID,
		At:      time.Now().UTC(),
	})

	view := s.toMemberView(*group, *target)
	return &view, nil
}

func (s *GroupService) SetGroupAdmin(ctx context.Context, userID uint64, groupID string, targetUserID uint64, input SetGroupAdminInput) (*GroupMemberView, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
	}
	if group.Status != model.GroupStatusActive {
		return nil, apperrors.New(403, "group_closed", "group is closed")
	}
	if err := s.requireAdminAssignmentPermission(ctx, userID, group); err != nil {
		return nil, err
	}

	target, err := s.groups.GetMember(ctx, group.ID, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "member_not_found", "target member not found")
		}
		return nil, err
	}
	if target.Role == model.RoleOwner {
		return nil, apperrors.New(403, "forbidden", "owner role cannot be modified")
	}
	if target.Status != model.MemberStatusActive {
		return nil, apperrors.New(400, "member_inactive", "only active members can be promoted to admin")
	}

	if input.IsAdmin {
		target.Role = model.RoleAdmin
	} else if target.Role == model.RoleAdmin {
		target.Role = model.RoleMember
	}

	if err := s.groups.UpdateMember(ctx, target); err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  targetUserID,
		At:      time.Now().UTC(),
	})

	view := s.toMemberView(*group, *target)
	return &view, nil
}

func (s *GroupService) MuteMember(ctx context.Context, userID uint64, groupID string, targetUserID uint64, input MuteMemberInput) (*GroupMemberView, error) {
	group, _, target, err := s.requireManageableTarget(ctx, userID, groupID, targetUserID, model.PermMuteMembers)
	if err != nil {
		return nil, err
	}

	var mutedUntil *time.Time
	if input.DurationSeconds > 0 {
		value := time.Now().UTC().Add(time.Duration(input.DurationSeconds) * time.Second)
		mutedUntil = &value
	}
	if err := s.groups.SetMuteUntil(ctx, target, mutedUntil); err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  targetUserID,
		At:      time.Now().UTC(),
	})

	view := s.toMemberView(*group, *target)
	return &view, nil
}

func (s *GroupService) BanMember(ctx context.Context, userID uint64, groupID string, targetUserID uint64) (*GroupMemberView, error) {
	group, _, target, err := s.requireManageableTarget(ctx, userID, groupID, targetUserID, model.PermBanMembers)
	if err != nil {
		return nil, err
	}

	target.Status = model.MemberStatusBanned
	target.MutedUntil = nil
	if err := s.groups.UpdateMember(ctx, target); err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  targetUserID,
		At:      time.Now().UTC(),
	})

	view := s.toMemberView(*group, *target)
	return &view, nil
}

func (s *GroupService) BuildSettingsEventForUser(ctx context.Context, viewerID uint64, groupID string) (*RealtimeEnvelope, error) {
	group, member, err := s.requireMembership(ctx, viewerID, groupID, true)
	if err != nil {
		if apperrors.HTTPStatus(err) == 403 || apperrors.HTTPStatus(err) == 404 {
			return nil, nil
		}
		return nil, err
	}
	view := s.toGroupView(*group, *member)
	return &RealtimeEnvelope{
		Type: events.EventGroupSettingsUpdated,
		Data: &view,
	}, nil
}

func (s *GroupService) BuildMemberEventForUser(ctx context.Context, viewerID uint64, groupID string, changedUserID uint64) (*RealtimeEnvelope, error) {
	group, member, err := s.requireActiveMembership(ctx, viewerID, groupID)
	if err != nil {
		if apperrors.HTTPStatus(err) == 403 || apperrors.HTTPStatus(err) == 404 {
			return nil, nil
		}
		return nil, err
	}

	perms := model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny)
	if group.MemberListVisibility == model.MemberListVisibilityHidden && !hasPermission(perms, model.PermViewMembers) {
		return nil, nil
	}

	target, err := s.groups.GetMember(ctx, group.ID, changedUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	view := s.toMemberView(*group, *target)
	return &RealtimeEnvelope{
		Type: events.EventGroupMemberUpdated,
		Data: view,
	}, nil
}

func (s *GroupService) CanAccessGroup(ctx context.Context, userID uint64, groupID string) bool {
	_, _, err := s.requireActiveMembership(ctx, userID, groupID)
	return err == nil
}

func (s *GroupService) requireActiveMembership(ctx context.Context, userID uint64, groupID string) (*model.Group, *model.GroupMember, error) {
	return s.requireMembership(ctx, userID, groupID, false)
}

func (s *GroupService) requireMembership(ctx context.Context, userID uint64, groupID string, allowClosed bool) (*model.Group, *model.GroupMember, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, nil, err
	}
	member, err := s.groups.GetMember(ctx, group.ID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, apperrors.New(403, "not_group_member", "user is not an active group member")
		}
		return nil, nil, err
	}
	if member.Status != model.MemberStatusActive {
		return nil, nil, apperrors.New(403, "member_inactive", "member is not active in this group")
	}
	if !allowClosed && group.Status != model.GroupStatusActive {
		return nil, nil, apperrors.New(403, "group_closed", "group is closed")
	}
	return group, member, nil
}

func (s *GroupService) requireCreateGroupPermission(ctx context.Context, userID uint64) error {
	switch s.mode {
	case "", "restricted":
		return s.admins.RequireServerAdmin(ctx, userID)
	case "public":
		return nil
	default:
		return s.admins.RequireServerAdmin(ctx, userID)
	}
}

func (s *GroupService) requireAdminAssignmentPermission(ctx context.Context, userID uint64, group *model.Group) error {
	isServerAdmin, err := s.admins.IsServerAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if isServerAdmin {
		return nil
	}
	if group.OwnerUserID == userID {
		return nil
	}
	return apperrors.New(403, "forbidden", "only server admins or the group owner can manage group admins")
}

func (s *GroupService) requireManageableTarget(ctx context.Context, userID uint64, groupID string, targetUserID uint64, requiredPerm int64) (*model.Group, *model.GroupMember, *model.GroupMember, error) {
	group, operator, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, nil, nil, err
	}
	operatorPerms := model.EffectivePermissions(operator.Role, group.DefaultPermissions, operator.PermissionsAllow, operator.PermissionsDeny)
	if !hasPermission(operatorPerms, requiredPerm) {
		return nil, nil, nil, apperrors.New(403, "forbidden", "missing member management permission")
	}

	target, err := s.groups.GetMember(ctx, group.ID, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil, apperrors.New(404, "member_not_found", "target member not found")
		}
		return nil, nil, nil, err
	}
	if target.Role == model.RoleOwner {
		return nil, nil, nil, apperrors.New(403, "forbidden", "owner cannot be modified")
	}
	if model.RoleRank(operator.Role) <= model.RoleRank(target.Role) && operator.Role != model.RoleOwner {
		return nil, nil, nil, apperrors.New(403, "forbidden", "cannot manage equal or higher role")
	}

	return group, operator, target, nil
}

func (s *GroupService) toGroupView(group model.Group, member model.GroupMember) GroupView {
	return GroupView{
		GroupID:                group.GroupID.String(),
		Title:                  group.Title,
		About:                  group.About,
		AvatarCID:              group.AvatarCID,
		OwnerUser:              toPublicUser(group.OwnerUser),
		MemberListVisibility:   group.MemberListVisibility,
		JoinMode:               group.JoinMode,
		DefaultPermissions:     group.DefaultPermissions,
		MessageTTLSeconds:      group.MessageTTLSeconds,
		MessageRetractSeconds:  group.MessageRetractSeconds,
		MessageCooldownSeconds: group.MessageCooldownSeconds,
		LastMessageSeq:         group.LastMessageSeq,
		SettingsVersion:        group.SettingsVersion,
		Status:                 group.Status,
		EffectivePermissions:   model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny),
		CreatedAt:              group.CreatedAt,
		UpdatedAt:              group.UpdatedAt,
	}
}

func (s *GroupService) toMemberView(group model.Group, member model.GroupMember) GroupMemberView {
	return GroupMemberView{
		User:                 toPublicUser(member.User),
		Role:                 member.Role,
		Status:               member.Status,
		Title:                member.Title,
		JoinedAt:             member.JoinedAt,
		MutedUntil:           member.MutedUntil,
		PermissionsAllow:     member.PermissionsAllow,
		PermissionsDeny:      member.PermissionsDeny,
		EffectivePermissions: model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny),
	}
}
