package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"meshchat-server/internal/model"

	"github.com/google/uuid"
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
	if err := r.ensureTable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		ownerMember.GroupID = group.ID
		return tx.Create(ownerMember).Error
	})
}

func (r *GroupRepo) GetByGroupID(ctx context.Context, groupID string) (*model.Group, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	parsed, err := uuid.Parse(groupID)
	if err != nil {
		return nil, gorm.ErrRecordNotFound
	}
	var group model.Group
	if err := r.db.WithContext(ctx).Preload("OwnerUser").First(&group, "group_id = ?", parsed).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) List(ctx context.Context, limit, offset int) ([]model.Group, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
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
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	var group model.Group
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) Update(ctx context.Context, group *model.Group) error {
	if err := r.ensureTable(); err != nil {
		return err
	}
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
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	var member model.GroupMember
	if err := r.db.WithContext(ctx).Preload("User").First(&member, "group_id = ? AND user_id = ?", groupDBID, userID).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *GroupRepo) GetMemberForUpdate(ctx context.Context, tx *gorm.DB, groupDBID, userID uint64) (*model.GroupMember, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
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
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	parsed, err := uuid.Parse(groupID)
	if err != nil {
		return nil, gorm.ErrRecordNotFound
	}
	var member model.GroupMember
	if err := r.db.WithContext(ctx).
		Joins("JOIN groups ON groups.id = group_members.group_id").
		Preload("User").
		Where("groups.group_id = ? AND group_members.user_id = ?", parsed, userID).
		First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *GroupRepo) ListMembers(ctx context.Context, groupDBID uint64) ([]model.GroupMember, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
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

func (r *GroupRepo) ListGroupsByMember(ctx context.Context, userID uint64, limit, offset int) ([]model.GroupMember, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	var members []model.GroupMember
	query := r.db.WithContext(ctx).
		Preload("Group.OwnerUser").
		Where("group_members.user_id = ? AND group_members.status = ?", userID, model.MemberStatusActive).
		Order("group_members.joined_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *GroupRepo) UpdateMember(ctx context.Context, member *model.GroupMember) error {
	if err := r.ensureTable(); err != nil {
		return err
	}
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
	if err := r.ensureTable(); err != nil {
		return 0, err
	}
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
	if err := r.ensureTable(); err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(group).Updates(map[string]any{
		"message_ttl_seconds":      group.MessageTTLSeconds,
		"message_retract_seconds":  group.MessageRetractSeconds,
		"message_cooldown_seconds": group.MessageCooldownSeconds,
		"settings_version":         group.SettingsVersion,
	}).Error
}

func (r *GroupRepo) SetMuteUntil(ctx context.Context, member *model.GroupMember, until *time.Time) error {
	if err := r.ensureTable(); err != nil {
		return err
	}
	member.MutedUntil = until
	return r.UpdateMember(ctx, member)
}

func (r *GroupRepo) ensureTable() error {
	legacyGroupSchema, err := r.hasLegacyGroupIdentifier()
	if err != nil {
		return err
	}
	if legacyGroupSchema {
		// Recover from older schemas that created groups.group_id as bigint.
		if err := r.db.Migrator().DropTable(&model.GroupMember{}, &model.Group{}); err != nil {
			return err
		}
	}
	if err := r.renameLegacyGroupAvatarColumn(); err != nil {
		return err
	}
	// Build the dependency chain in order so foreign keys do not fail on
	// partially migrated databases.
	if r.db.Migrator().HasTable(&model.ServerUser{}) &&
		r.db.Migrator().HasTable(&model.Group{}) &&
		r.db.Migrator().HasTable(&model.GroupMember{}) {
		return nil
	}
	if !r.db.Migrator().HasTable(&model.ServerUser{}) {
		if err := r.db.Migrator().CreateTable(&model.ServerUser{}); err != nil {
			return err
		}
	}
	if !r.db.Migrator().HasTable(&model.Group{}) {
		if err := r.ensureCanonicalGroupTable(); err != nil {
			return err
		}
	}
	if !r.db.Migrator().HasTable(&model.GroupMember{}) {
		if err := r.ensureCanonicalGroupMemberTable(); err != nil {
			return err
		}
	}
	return nil
}

func (r *GroupRepo) ensureCanonicalGroupTable() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS "groups" (
			"id" bigserial PRIMARY KEY,
			"group_id" uuid NOT NULL,
			"title" varchar(256) NOT NULL,
			"about" varchar(2048),
			"avatar_cid" varchar(255),
			"owner_user_id" bigint NOT NULL,
			"member_list_visibility" varchar(32) NOT NULL DEFAULT 'visible',
			"join_mode" varchar(32) NOT NULL DEFAULT 'invite_only',
			"default_permissions" bigint NOT NULL DEFAULT 0,
			"message_ttl_seconds" bigint NOT NULL DEFAULT 0,
			"message_retract_seconds" bigint NOT NULL DEFAULT 0,
			"message_cooldown_seconds" bigint NOT NULL DEFAULT 0,
			"last_message_seq" bigint NOT NULL DEFAULT 0,
			"settings_version" bigint NOT NULL DEFAULT 1,
			"status" varchar(32) NOT NULL DEFAULT 'active',
			"created_at" timestamptz,
			"updated_at" timestamptz
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_groups_group_id" ON "groups" ("group_id")`,
		`CREATE INDEX IF NOT EXISTS "idx_groups_owner_user_id" ON "groups" ("owner_user_id")`,
	}
	for _, stmt := range statements {
		if err := r.db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("ensure groups table: %w", err)
		}
	}
	return nil
}

func (r *GroupRepo) ensureCanonicalGroupMemberTable() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS "group_members" (
			"id" bigserial PRIMARY KEY,
			"group_id" bigint NOT NULL,
			"user_id" bigint NOT NULL,
			"role" varchar(32) NOT NULL,
			"status" varchar(32) NOT NULL,
			"title" varchar(128),
			"joined_at" timestamptz NOT NULL,
			"muted_until" timestamptz,
			"permissions_allow" bigint NOT NULL DEFAULT 0,
			"permissions_deny" bigint NOT NULL DEFAULT 0,
			"created_at" timestamptz,
			"updated_at" timestamptz
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_group_member_user" ON "group_members" ("group_id", "user_id")`,
		`CREATE INDEX IF NOT EXISTS "idx_group_members_group_id" ON "group_members" ("group_id")`,
		`CREATE INDEX IF NOT EXISTS "idx_group_members_user_id" ON "group_members" ("user_id")`,
	}
	for _, stmt := range statements {
		if err := r.db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("ensure group_members table: %w", err)
		}
	}
	return nil
}

func (r *GroupRepo) hasLegacyGroupIdentifier() (bool, error) {
	if !r.db.Migrator().HasTable(&model.Group{}) {
		return false, nil
	}

	var dataType sql.NullString
	row := r.db.Raw(`
		SELECT data_type
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'groups'
		  AND column_name = 'group_id'
	`)
	if err := row.Scan(&dataType).Error; err != nil {
		return false, err
	}
	if !dataType.Valid || dataType.String == "" {
		return false, nil
	}
	return dataType.String != "uuid", nil
}

func (r *GroupRepo) renameLegacyGroupAvatarColumn() error {
	var hasLegacyColumn bool
	if err := r.db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'groups'
			  AND column_name = 'avatar_c_id'
		)
	`).Scan(&hasLegacyColumn).Error; err != nil {
		return err
	}
	if !hasLegacyColumn {
		return nil
	}

	var hasCanonicalColumn bool
	if err := r.db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'groups'
			  AND column_name = 'avatar_cid'
		)
	`).Scan(&hasCanonicalColumn).Error; err != nil {
		return err
	}
	if hasCanonicalColumn {
		return r.db.Exec(`ALTER TABLE "groups" DROP COLUMN IF EXISTS "avatar_c_id"`).Error
	}
	return r.db.Exec(`ALTER TABLE "groups" RENAME COLUMN "avatar_c_id" TO "avatar_cid"`).Error
}
