package repo

import (
	"context"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MessageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) DB() *gorm.DB {
	return r.db
}

func (r *MessageRepo) Create(ctx context.Context, tx *gorm.DB, message *model.GroupMessage) error {
	return tx.WithContext(ctx).Create(message).Error
}

func (r *MessageRepo) GetByGroupAndMessageID(ctx context.Context, groupDBID uint64, messageID string) (*model.GroupMessage, error) {
	var message model.GroupMessage
	if err := r.db.WithContext(ctx).
		Preload("Sender").
		First(&message, "group_id = ? AND message_id = ?", groupDBID, messageID).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *MessageRepo) GetByMessageID(ctx context.Context, messageID string) (*model.GroupMessage, error) {
	var message model.GroupMessage
	if err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Group").
		First(&message, "message_id = ?", messageID).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *MessageRepo) GetByMessageIDForUpdate(ctx context.Context, tx *gorm.DB, messageID string) (*model.GroupMessage, error) {
	var message model.GroupMessage
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Sender").
		Preload("Group").
		First(&message, "message_id = ?", messageID).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *MessageRepo) ListGroupMessages(ctx context.Context, groupDBID uint64, beforeSeq uint64, limit int) ([]model.GroupMessage, error) {
	var messages []model.GroupMessage
	query := r.db.WithContext(ctx).
		Preload("Sender").
		Where("group_id = ?", groupDBID).
		Order("seq DESC").
		Limit(limit)
	if beforeSeq > 0 {
		query = query.Where("seq < ?", beforeSeq)
	}
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *MessageRepo) CreateEdit(ctx context.Context, tx *gorm.DB, edit *model.GroupMessageEdit) error {
	return tx.WithContext(ctx).Create(edit).Error
}

func (r *MessageRepo) Save(ctx context.Context, tx *gorm.DB, message *model.GroupMessage) error {
	return tx.WithContext(ctx).Save(message).Error
}
