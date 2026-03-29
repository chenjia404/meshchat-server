package service

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"meshchat-server/internal/auth"
	"meshchat-server/internal/events"
	"meshchat-server/internal/ipfs"
	"meshchat-server/internal/model"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminService struct {
	users     *repo.UserRepo
	groups    *repo.GroupRepo
	messages  *MessageService
	ipfs      ipfs.Client
	publisher EventPublisher
	jwt       *auth.AdminJWTManager
	username  string
	password  string
}

func NewAdminService(users *repo.UserRepo, groups *repo.GroupRepo, messages *MessageService, ipfs ipfs.Client, publisher EventPublisher, jwt *auth.AdminJWTManager, username, password string) *AdminService {
	return &AdminService{
		users:     users,
		groups:    groups,
		messages:  messages,
		ipfs:      ipfs,
		publisher: publisher,
		jwt:       jwt,
		username:  strings.TrimSpace(username),
		password:  password,
	}
}

func (s *AdminService) Login(ctx context.Context, username, password string) (*AdminLoginResponse, error) {
	if subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) != 1 || subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) != 1 {
		return nil, apperrors.New(401, "invalid_admin_credentials", "admin username or password is invalid")
	}
	if _, err := s.ensureAdminUser(ctx); err != nil {
		return nil, err
	}
	token, err := s.jwt.IssueToken(s.username)
	if err != nil {
		return nil, err
	}
	return &AdminLoginResponse{Token: token, Username: s.username}, nil
}

func (s *AdminService) Me() *AdminMeView {
	return &AdminMeView{Username: s.username}
}

func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]AdminUserView, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	users, err := s.users.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	views := make([]AdminUserView, 0, len(users))
	for _, user := range users {
		views = append(views, AdminUserView{
			PeerID:     user.PeerID,
			PublicUser: toPublicUser(user),
		})
	}
	return views, nil
}

// UpdateUserProfileByPeerID updates a user profile using peer_id as the stable identifier.
func (s *AdminService) UpdateUserProfileByPeerID(ctx context.Context, peerID string, input UpdateProfileInput) (*PublicUser, error) {
	user, err := s.users.GetByPeerID(ctx, peerID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "user_not_found", "user not found")
		}
		return nil, err
	}

	if err := validateProfileFieldLengths(input.DisplayName, input.Bio); err != nil {
		return nil, err
	}
	if input.AvatarCID != "" {
		if err := s.ipfs.ValidateCID(input.AvatarCID); err != nil {
			return nil, payloadValidationError("avatar_cid is invalid")
		}
	}

	user.DisplayName = strings.TrimSpace(input.DisplayName)
	user.AvatarCID = strings.TrimSpace(input.AvatarCID)
	user.Bio = strings.TrimSpace(input.Bio)
	if input.Status != "" {
		user.Status = strings.TrimSpace(input.Status)
	}
	user.ProfileVersion++

	if err := s.users.UpdateProfile(ctx, user); err != nil {
		return nil, err
	}

	publicUser := toPublicUser(*user)
	return &publicUser, nil
}

func (s *AdminService) ListGroups(ctx context.Context, limit, offset int) ([]GroupView, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	groups, err := s.groups.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	views := make([]GroupView, 0, len(groups))
	fakeMember := model.GroupMember{Role: model.RoleOwner, PermissionsAllow: 0, PermissionsDeny: 0}
	for _, group := range groups {
		views = append(views, s.toAdminGroupView(group, fakeMember))
	}
	return views, nil
}

func (s *AdminService) CreateGroup(ctx context.Context, input CreateGroupInput) (*GroupView, error) {
	adminUser, err := s.ensureAdminUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.createGroupAsOwner(ctx, adminUser.ID, input)
}

func (s *AdminService) GetGroup(ctx context.Context, groupID string) (*GroupView, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
	}
	view := s.toAdminGroupView(*group, model.GroupMember{Role: model.RoleOwner})
	return &view, nil
}

func (s *AdminService) UpdateGroup(ctx context.Context, groupID string, input UpdateGroupInput) (*GroupView, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
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

	view := s.toAdminGroupView(*group, model.GroupMember{Role: model.RoleOwner})
	return &view, nil
}

func (s *AdminService) DissolveGroup(ctx context.Context, groupID string) (*GroupLifecycleView, error) {
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

func (s *AdminService) ListMembers(ctx context.Context, groupID string) ([]GroupMemberView, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
	}
	members, err := s.groups.ListMembers(ctx, group.ID)
	if err != nil {
		return nil, err
	}

	views := make([]GroupMemberView, 0, len(members))
	for _, item := range members {
		views = append(views, s.toAdminMemberView(*group, item))
	}
	return views, nil
}

func (s *AdminService) ListMessages(ctx context.Context, groupID string, beforeSeq uint64, limit int) ([]MessageView, error) {
	return s.messages.ListMessagesForAdmin(ctx, groupID, beforeSeq, limit)
}

func (s *AdminService) SetGroupAdmin(ctx context.Context, groupID string, targetUserID uint64, input SetGroupAdminInput) (*GroupMemberView, error) {
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
	target, err := s.groups.GetMember(ctx, group.ID, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "member_not_found", "target member not found")
		}
		return nil, err
	}
	if target.Status != model.MemberStatusActive {
		return nil, apperrors.New(400, "member_inactive", "only active members can be promoted to admin")
	}
	if target.Role == model.RoleOwner {
		return nil, apperrors.New(403, "forbidden", "owner role cannot be modified")
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

	view := s.toAdminMemberView(*group, *target)
	return &view, nil
}

func (s *AdminService) TransferGroupOwnership(ctx context.Context, groupID string, targetUserID uint64) (*GroupView, error) {
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
	if targetUserID == 0 {
		return nil, apperrors.New(400, "invalid_user_id", "target user_id is required")
	}
	if targetUserID == group.OwnerUserID {
		owner, err := s.users.GetByID(ctx, group.OwnerUserID)
		if err != nil {
			return nil, err
		}
		group.OwnerUser = *owner
		view := s.toAdminGroupView(*group, model.GroupMember{Role: model.RoleOwner})
		return &view, nil
	}

	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedGroup, err := s.groups.GetByIDForUpdate(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		currentOwner, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, lockedGroup.OwnerUserID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(404, "member_not_found", "current owner membership not found")
			}
			return err
		}
		target, err := s.groups.GetMemberForUpdate(ctx, tx, group.ID, targetUserID)
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

	group, err = s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupSettingsUpdated,
		GroupID: group.GroupID.String(),
		At:      time.Now().UTC(),
	})
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  group.OwnerUserID,
		At:      time.Now().UTC(),
	})
	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:    events.EventGroupMemberUpdated,
		GroupID: group.GroupID.String(),
		UserID:  targetUserID,
		At:      time.Now().UTC(),
	})

	owner, err := s.users.GetByID(ctx, group.OwnerUserID)
	if err != nil {
		return nil, err
	}
	group.OwnerUser = *owner
	view := s.toAdminGroupView(*group, model.GroupMember{Role: model.RoleOwner})
	return &view, nil
}

func (s *AdminService) createGroupAsOwner(ctx context.Context, ownerUserID uint64, input CreateGroupInput) (*GroupView, error) {
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
		OwnerUserID:            ownerUserID,
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
		UserID:           ownerUserID,
		Role:             model.RoleOwner,
		Status:           model.MemberStatusActive,
		JoinedAt:         time.Now().UTC(),
		PermissionsAllow: 0,
		PermissionsDeny:  0,
	}

	if err := s.groups.CreateWithOwner(ctx, group, member); err != nil {
		return nil, err
	}

	ownerUser, err := s.users.GetByID(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	group.OwnerUser = *ownerUser
	ownerView := s.toAdminGroupView(*group, *member)
	return &ownerView, nil
}

func (s *AdminService) ensureAdminUser(ctx context.Context) (*model.ServerUser, error) {
	peerID := s.adminPeerID()
	user, err := s.users.GetByPeerID(ctx, peerID)
	if err == nil {
		return user, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	created := &model.ServerUser{
		PeerID:         peerID,
		PublicKey:      fmt.Sprintf("admin:%s", s.username),
		Username:       s.username,
		DisplayName:    s.username,
		ProfileVersion: 1,
		Status:         "active",
	}
	if err := s.users.Create(ctx, created); err != nil {
		if duplicateConstraintError(err) {
			created.Username = fmt.Sprintf("%s_%s", s.username, uuid.NewString()[:8])
			created.DisplayName = s.username
			if err := s.users.Create(ctx, created); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return created, nil
}

func (s *AdminService) adminPeerID() string {
	return "admin:" + s.username
}

func (s *AdminService) toAdminGroupView(group model.Group, member model.GroupMember) GroupView {
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
		LastMessageTimestamp:   group.LastMessageAt,
		SettingsVersion:        group.SettingsVersion,
		Status:                 group.Status,
		EffectivePermissions:   model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny),
		CreatedAt:              group.CreatedAt,
		UpdatedAt:              group.UpdatedAt,
	}
}

func (s *AdminService) toAdminMemberView(group model.Group, member model.GroupMember) GroupMemberView {
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
