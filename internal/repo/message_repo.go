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
		First(&message, "message_id = ?", messageID).Error; err != nil {
		return nil, err
	}
	if err := r.attachGroupByInternalID(ctx, r.db, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *MessageRepo) GetByMessageIDForUpdate(ctx context.Context, tx *gorm.DB, messageID string) (*model.GroupMessage, error) {
	var message model.GroupMessage
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Sender").
		First(&message, "message_id = ?", messageID).Error; err != nil {
		return nil, err
	}
	if err := r.attachGroupByInternalID(ctx, tx, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// attachGroupByInternalID 按 groups.id 加载 Group（含 OwnerUser），避免 Preload("Group") 与 Group.group_id（UUID）列同名冲突。
func (r *MessageRepo) attachGroupByInternalID(ctx context.Context, db *gorm.DB, message *model.GroupMessage) error {
	if message.GroupID == 0 {
		return nil
	}
	var g model.Group
	if err := db.WithContext(ctx).Preload("OwnerUser").First(&g, "id = ?", message.GroupID).Error; err != nil {
		return err
	}
	message.Group = g
	return nil
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
