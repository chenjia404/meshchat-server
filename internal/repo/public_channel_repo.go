package repo

import (
	"context"
	"errors"
	"time"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PublicChannelRepo struct {
	db *gorm.DB
}

func NewPublicChannelRepo(db *gorm.DB) *PublicChannelRepo {
	return &PublicChannelRepo{db: db}
}

func (r *PublicChannelRepo) DB() *gorm.DB {
	return r.db
}

func (r *PublicChannelRepo) GetByChannelID(ctx context.Context, channelID string) (*model.PublicChannel, error) {
	var out model.PublicChannel
	if err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PublicChannelRepo) GetByChannelIDForUpdate(ctx context.Context, tx *gorm.DB, channelID string) (*model.PublicChannel, error) {
	var out model.PublicChannel
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("channel_id = ?", channelID).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PublicChannelRepo) CreateChannel(ctx context.Context, channel *model.PublicChannel, sub *model.PublicChannelSubscription) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(channel).Error; err != nil {
			return err
		}
		if sub != nil {
			sub.ChannelDBID = channel.ID
			if err := tx.Create(sub).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PublicChannelRepo) UpdateChannelProfile(ctx context.Context, channelID string, mutate func(*model.PublicChannel, *model.PublicChannelChange) error) (*model.PublicChannel, error) {
	var out *model.PublicChannel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ch, err := r.GetByChannelIDForUpdate(ctx, tx, channelID)
		if err != nil {
			return err
		}
		change := &model.PublicChannelChange{ChannelDBID: ch.ID}
		if err := mutate(ch, change); err != nil {
			return err
		}
		if err := tx.Save(ch).Error; err != nil {
			return err
		}
		if err := tx.Create(change).Error; err != nil {
			return err
		}
		out = ch
		return nil
	})
	return out, err
}

func (r *PublicChannelRepo) ListChannelsByOwnerPeerID(ctx context.Context, ownerPeerID string) ([]model.PublicChannel, error) {
	var rows []model.PublicChannel
	err := r.db.WithContext(ctx).Where("owner_peer_id = ?", ownerPeerID).Order("updated_at_unix DESC, channel_id DESC").Find(&rows).Error
	return rows, err
}

func (r *PublicChannelRepo) ListSubscribedChannelsForUser(ctx context.Context, userID uint64) ([]model.PublicChannel, error) {
	var rows []model.PublicChannel
	err := r.db.WithContext(ctx).
		Joins("JOIN public_channel_subscriptions s ON s.channel_db_id = public_channels.id").
		Where("s.user_id = ? AND s.subscribed = ?", userID, true).
		Order("s.updated_at_unix DESC, public_channels.channel_id DESC").
		Find(&rows).Error
	return rows, err
}

func (r *PublicChannelRepo) GetSubscription(ctx context.Context, channelDBID uint64, userID uint64) (*model.PublicChannelSubscription, error) {
	var out model.PublicChannelSubscription
	if err := r.db.WithContext(ctx).Where("channel_db_id = ? AND user_id = ?", channelDBID, userID).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PublicChannelRepo) UpsertSubscription(ctx context.Context, channelDBID, userID uint64, lastSeenSeq, nowUnix int64, subscribed bool) (*model.PublicChannelSubscription, error) {
	out := &model.PublicChannelSubscription{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sub model.PublicChannelSubscription
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("channel_db_id = ? AND user_id = ?", channelDBID, userID).
			First(&sub).Error
		now := time.Now().UTC()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sub = model.PublicChannelSubscription{
				ChannelDBID:           channelDBID,
				UserID:                userID,
				LastSeenSeq:           lastSeenSeq,
				LastSyncedSeq:         lastSeenSeq,
				LatestLoadedMessageID: 0,
				OldestLoadedMessageID: 0,
				UnreadCount:           0,
				Subscribed:            subscribed,
				UpdatedAtUnix:         nowUnix,
				CreatedAt:             now,
				UpdatedAt:             now,
			}
			if err := tx.Create(&sub).Error; err != nil {
				return err
			}
			*out = sub
			return nil
		}
		if err != nil {
			return err
		}
		if lastSeenSeq > sub.LastSeenSeq {
			sub.LastSeenSeq = lastSeenSeq
		}
		if lastSeenSeq > sub.LastSyncedSeq {
			sub.LastSyncedSeq = lastSeenSeq
		}
		sub.Subscribed = subscribed
		sub.UpdatedAtUnix = nowUnix
		sub.UpdatedAt = now
		if err := tx.Save(&sub).Error; err != nil {
			return err
		}
		*out = sub
		return nil
	})
	return out, err
}

func (r *PublicChannelRepo) CreateMessage(ctx context.Context, channelID string, mutate func(*model.PublicChannelMessage, *model.PublicChannel, *model.PublicChannelChange) error) (*model.PublicChannelMessage, *model.PublicChannel, error) {
	var saved model.PublicChannelMessage
	var channel model.PublicChannel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := r.GetByChannelIDForUpdate(ctx, tx, channelID)
		if err != nil {
			return err
		}
		channel = *locked
		msg := &model.PublicChannelMessage{
			ChannelDBID: locked.ID,
		}
		change := &model.PublicChannelChange{
			ChannelDBID: locked.ID,
		}
		if err := mutate(msg, locked, change); err != nil {
			return err
		}
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		if err := tx.Create(change).Error; err != nil {
			return err
		}
		if err := tx.Save(locked).Error; err != nil {
			return err
		}
		saved = *msg
		channel = *locked
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &saved, &channel, nil
}

func (r *PublicChannelRepo) UpdateMessage(ctx context.Context, channelID string, messageID int64, mutate func(*model.PublicChannelMessage, *model.PublicChannel, *model.PublicChannelChange) error) (*model.PublicChannelMessage, *model.PublicChannel, error) {
	var saved model.PublicChannelMessage
	var channel model.PublicChannel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := r.GetByChannelIDForUpdate(ctx, tx, channelID)
		if err != nil {
			return err
		}
		channel = *locked
		var msg model.PublicChannelMessage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("channel_db_id = ? AND message_id = ?", locked.ID, messageID).
			First(&msg).Error; err != nil {
			return err
		}
		change := &model.PublicChannelChange{ChannelDBID: locked.ID}
		if err := mutate(&msg, locked, change); err != nil {
			return err
		}
		if err := tx.Save(&msg).Error; err != nil {
			return err
		}
		if err := tx.Create(change).Error; err != nil {
			return err
		}
		if err := tx.Save(locked).Error; err != nil {
			return err
		}
		saved = msg
		channel = *locked
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &saved, &channel, nil
}

func (r *PublicChannelRepo) GetMessage(ctx context.Context, channelDBID uint64, messageID int64) (*model.PublicChannelMessage, error) {
	var out model.PublicChannelMessage
	if err := r.db.WithContext(ctx).Where("channel_db_id = ? AND message_id = ?", channelDBID, messageID).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PublicChannelRepo) ListMessages(ctx context.Context, channelDBID uint64, beforeMessageID int64, limit int) ([]model.PublicChannelMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Where("channel_db_id = ?", channelDBID).Order("message_id DESC").Limit(limit)
	if beforeMessageID > 0 {
		q = q.Where("message_id < ?", beforeMessageID)
	}
	var rows []model.PublicChannelMessage
	err := q.Find(&rows).Error
	return rows, err
}

func (r *PublicChannelRepo) ListChanges(ctx context.Context, channelDBID uint64, afterSeq int64, limit int) ([]model.PublicChannelChange, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var rows []model.PublicChannelChange
	err := r.db.WithContext(ctx).
		Where("channel_db_id = ? AND seq > ?", channelDBID, afterSeq).
		Order("seq ASC").
		Limit(limit + 1).
		Find(&rows).Error
	return rows, err
}
