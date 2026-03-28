package repo

import (
	"context"
	"time"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupRepo struct {
	db *gorm.DB
}

func NewGroupRepo(db *gorm.DB) *GroupRepo {
	return &GroupRepo{db: db}
}

func (r *GroupRepo) DB() *gorm.DB {
	return r.db
}

func (r *GroupRepo) CreateWithOwner(ctx context.Context, group *model.Group, ownerMember *model.GroupMember) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		ownerMember.GroupID = group.ID
		return tx.Create(ownerMember).Error
	})
}

func (r *GroupRepo) GetByGroupID(ctx context.Context, groupID string) (*model.Group, error) {
	var group model.Group
	if err := r.db.WithContext(ctx).Preload("OwnerUser").First(&group, "group_id = ?", groupID).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) List(ctx context.Context, limit, offset int) ([]model.Group, error) {
	var groups []model.Group
	query := r.db.WithContext(ctx).Preload("OwnerUser").Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *GroupRepo) GetByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*model.Group, error) {
	var group model.Group
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) Update(ctx context.Context, group *model.Group) error {
	return r.db.WithContext(ctx).Model(group).Updates(map[string]any{
		"title":                    group.Title,
		"about":                    group.About,
		"avatar_cid":               group.AvatarCID,
		"member_list_visibility":   group.MemberListVisibility,
		"join_mode":                group.JoinMode,
		"default_permissions":      group.DefaultPermissions,
		"message_ttl_seconds":      group.MessageTTLSeconds,
		"message_retract_seconds":  group.MessageRetractSeconds,
		"message_cooldown_seconds": group.MessageCooldownSeconds,
		"settings_version":         group.SettingsVersion,
		"status":                   group.Status,
	}).Error
}

func (r *GroupRepo) GetMember(ctx context.Context, groupDBID, userID uint64) (*model.GroupMember, error) {
	var member model.GroupMember
	if err := r.db.WithContext(ctx).Preload("User").First(&member, "group_id = ? AND user_id = ?", groupDBID, userID).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *GroupRepo) GetMemberForUpdate(ctx context.Context, tx *gorm.DB, groupDBID, userID uint64) (*model.GroupMember, error) {
	var member model.GroupMember
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("User").
		First(&member, "group_id = ? AND user_id = ?", groupDBID, userID).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *GroupRepo) GetMemberByGroupIdentifier(ctx context.Context, groupID string, userID uint64) (*model.GroupMember, error) {
	var member model.GroupMember
	if err := r.db.WithContext(ctx).
		Joins("JOIN groups ON groups.id = group_members.group_id").
		Preload("User").
		Where("groups.group_id = ? AND group_members.user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *GroupRepo) ListMembers(ctx context.Context, groupDBID uint64) ([]model.GroupMember, error) {
	var members []model.GroupMember
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("group_id = ?", groupDBID).
		Order("joined_at ASC").
		Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *GroupRepo) UpdateMember(ctx context.Context, member *model.GroupMember) error {
	return r.db.WithContext(ctx).Model(member).Updates(map[string]any{
		"role":              member.Role,
		"status":            member.Status,
		"title":             member.Title,
		"muted_until":       member.MutedUntil,
		"permissions_allow": member.PermissionsAllow,
		"permissions_deny":  member.PermissionsDeny,
	}).Error
}

func (r *GroupRepo) IncrementLastSeq(ctx context.Context, tx *gorm.DB, groupID uint64) (uint64, error) {
	group, err := r.GetByIDForUpdate(ctx, tx, groupID)
	if err != nil {
		return 0, err
	}
	group.LastMessageSeq++
	if err := tx.WithContext(ctx).Model(group).Update("last_message_seq", group.LastMessageSeq).Error; err != nil {
		return 0, err
	}
	return group.LastMessageSeq, nil
}

func (r *GroupRepo) UpdateMessagePolicyTx(ctx context.Context, tx *gorm.DB, group *model.Group) error {
	return tx.WithContext(ctx).Model(group).Updates(map[string]any{
		"message_ttl_seconds":      group.MessageTTLSeconds,
		"message_retract_seconds":  group.MessageRetractSeconds,
		"message_cooldown_seconds": group.MessageCooldownSeconds,
		"settings_version":         group.SettingsVersion,
	}).Error
}

func (r *GroupRepo) SetMuteUntil(ctx context.Context, member *model.GroupMember, until *time.Time) error {
	member.MutedUntil = until
	return r.UpdateMember(ctx, member)
}
