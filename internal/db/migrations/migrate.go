package migrations

import (
	"context"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
)

// Run executes the schema migration required by the current application version.
func Run(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
			return err
		}

		return tx.AutoMigrate(
			&model.ServerUser{},
			&model.Group{},
			&model.GroupMember{},
			&model.GroupMessage{},
			&model.GroupMessageEdit{},
			&model.File{},
		)
	})
}
