package migrations

import (
	"context"
	"fmt"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
)

// Run executes the schema migration required by the current application version.
func Run(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
			return err
		}

		if err := tx.AutoMigrate(
			&model.ServerUser{},
			&model.File{},
		); err != nil {
			return err
		}
		if err := ensureCanonicalUserAvatarColumn(tx); err != nil {
			return err
		}

		if err := ensureCanonicalGroupTables(tx); err != nil {
			return err
		}
		if err := ensureGroupLastMessageAtColumn(tx); err != nil {
			return err
		}

		return nil
	})
}

func ensureCanonicalGroupTables(tx *gorm.DB) error {
	if err := renameLegacyGroupAvatarColumn(tx); err != nil {
		return err
	}

	legacyGroupSchema, err := hasLegacyGroupIdentifier(tx)
	if err != nil {
		return err
	}
	if legacyGroupSchema {
		// Drop the group-related tables when an older schema created the
		// external identifier column with the wrong type.
		if err := tx.Migrator().DropTable(
			&model.GroupMessageEdit{},
			&model.GroupMessage{},
			&model.GroupMember{},
			&model.Group{},
		); err != nil {
			return err
		}
	}

	// Create the two core tables explicitly so group_id is guaranteed to be
	// UUID even when GORM's inferred DDL differs across environments.
	groupStatements := []string{
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
			"last_message_at" bigint NOT NULL DEFAULT 0,
			"settings_version" bigint NOT NULL DEFAULT 1,
			"status" varchar(32) NOT NULL DEFAULT 'active',
			"created_at" timestamptz,
			"updated_at" timestamptz
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_groups_group_id" ON "groups" ("group_id")`,
		`CREATE INDEX IF NOT EXISTS "idx_groups_owner_user_id" ON "groups" ("owner_user_id")`,
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
		`CREATE TABLE IF NOT EXISTS "group_messages" (
			"id" bigserial PRIMARY KEY,
			"group_id" bigint NOT NULL,
			"message_id" text NOT NULL,
			"sender_user_id" bigint NOT NULL,
			"seq" bigint NOT NULL,
			"content_type" varchar(32) NOT NULL,
			"payload_json" jsonb NOT NULL,
			"reply_to_message_id" text,
			"forward_from_message_id" text,
			"status" varchar(32) NOT NULL DEFAULT 'normal',
			"edit_count" integer NOT NULL DEFAULT 0,
			"last_edited_at" timestamptz,
			"last_edited_by_user_id" bigint,
			"deleted_at" timestamptz,
			"deleted_by_user_id" bigint,
			"delete_reason" varchar(64),
			"signature" text,
			"created_at" timestamptz,
			"updated_at" timestamptz
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_group_message_message_id" ON "group_messages" ("group_id", "message_id")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_group_message_seq" ON "group_messages" ("group_id", "seq")`,
		`CREATE INDEX IF NOT EXISTS "idx_group_created_at" ON "group_messages" ("group_id", "created_at")`,
		`CREATE INDEX IF NOT EXISTS "idx_group_sender_created_at" ON "group_messages" ("group_id", "sender_user_id", "created_at" DESC)`,
		`CREATE INDEX IF NOT EXISTS "idx_group_reply" ON "group_messages" ("group_id", "reply_to_message_id")`,
		`CREATE INDEX IF NOT EXISTS "idx_group_messages_forward_from_message_id" ON "group_messages" ("forward_from_message_id")`,
		`CREATE TABLE IF NOT EXISTS "group_message_edits" (
			"id" bigserial PRIMARY KEY,
			"group_id" bigint NOT NULL,
			"message_id" text NOT NULL,
			"editor_user_id" bigint NOT NULL,
			"old_payload_json" jsonb NOT NULL,
			"new_payload_json" jsonb NOT NULL,
			"created_at" timestamptz
		)`,
		`CREATE INDEX IF NOT EXISTS "idx_group_message_edits_group_id" ON "group_message_edits" ("group_id")`,
		`CREATE INDEX IF NOT EXISTS "idx_group_message_edits_message_id" ON "group_message_edits" ("message_id")`,
		`CREATE INDEX IF NOT EXISTS "idx_group_message_edits_editor_user_id" ON "group_message_edits" ("editor_user_id")`,
	}

	for _, stmt := range groupStatements {
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("ensure canonical group tables: %w", err)
		}
	}

	return nil
}

func ensureGroupLastMessageAtColumn(tx *gorm.DB) error {
	var hasUnix, hasMs bool
	if err := tx.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'groups' AND column_name = 'last_message_at'
		)
	`).Scan(&hasUnix).Error; err != nil {
		return err
	}
	if err := tx.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'groups' AND column_name = 'last_message_at_ms'
		)
	`).Scan(&hasMs).Error; err != nil {
		return err
	}
	if !hasUnix {
		if err := tx.Exec(`ALTER TABLE "groups" ADD COLUMN "last_message_at" bigint NOT NULL DEFAULT 0`).Error; err != nil {
			return fmt.Errorf("add groups.last_message_at: %w", err)
		}
		if hasMs {
			if err := tx.Exec(`UPDATE "groups" SET "last_message_at" = "last_message_at_ms" / 1000 WHERE "last_message_at_ms" != 0`).Error; err != nil {
				return fmt.Errorf("migrate last_message_at_ms to last_message_at: %w", err)
			}
			if err := tx.Exec(`ALTER TABLE "groups" DROP COLUMN "last_message_at_ms"`).Error; err != nil {
				return fmt.Errorf("drop groups.last_message_at_ms: %w", err)
			}
		}
	} else if hasMs {
		if err := tx.Exec(`ALTER TABLE "groups" DROP COLUMN "last_message_at_ms"`).Error; err != nil {
			return fmt.Errorf("drop groups.last_message_at_ms: %w", err)
		}
	}

	// 已有历史数据：若 last_message_seq 已递增但时间戳仍为 0，则用最后一条消息的 created_at 回填（Unix 秒）。
	if err := tx.Exec(`
		UPDATE "groups" g
		SET "last_message_at" = (EXTRACT(EPOCH FROM m.max_ca))::bigint
		FROM (
			SELECT "group_id", MAX("created_at") AS max_ca
			FROM "group_messages"
			GROUP BY "group_id"
		) m
		WHERE g."id" = m."group_id"
		  AND g."last_message_seq" > 0
		  AND g."last_message_at" = 0
	`).Error; err != nil {
		return fmt.Errorf("backfill groups.last_message_at: %w", err)
	}
	return nil
}

func ensureCanonicalUserAvatarColumn(tx *gorm.DB) error {
	var hasLegacyColumn bool
	if err := tx.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'server_users'
			  AND column_name = 'avatar_cid'
		)
	`).Scan(&hasLegacyColumn).Error; err != nil {
		return err
	}
	if hasLegacyColumn {
		return nil
	}

	var hasSnakeColumn bool
	if err := tx.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'server_users'
			  AND column_name = 'avatar_c_id'
		)
	`).Scan(&hasSnakeColumn).Error; err != nil {
		return err
	}
	if hasSnakeColumn {
		return tx.Exec(`ALTER TABLE "server_users" RENAME COLUMN "avatar_c_id" TO "avatar_cid"`).Error
	}
	return nil
}

func hasLegacyGroupIdentifier(tx *gorm.DB) (bool, error) {
	var dataType string
	row := tx.Raw(`
		SELECT data_type
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'groups'
		  AND column_name = 'group_id'
	`)
	if err := row.Scan(&dataType).Error; err != nil {
		return false, err
	}
	if dataType == "" {
		return false, nil
	}
	return dataType != "uuid", nil
}

func renameLegacyGroupAvatarColumn(tx *gorm.DB) error {
	var hasLegacyColumn bool
	if err := tx.Raw(`
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
	if err := tx.Raw(`
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
		return tx.Exec(`ALTER TABLE "groups" DROP COLUMN IF EXISTS "avatar_c_id"`).Error
	}
	return tx.Exec(`ALTER TABLE "groups" RENAME COLUMN "avatar_c_id" TO "avatar_cid"`).Error
}
