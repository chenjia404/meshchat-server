package repo

import (
	"context"
	"errors"
	"strings"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FriendMailboxRepo struct {
	db *gorm.DB
}

func NewFriendMailboxRepo(db *gorm.DB) *FriendMailboxRepo {
	return &FriendMailboxRepo{db: db}
}

func (r *FriendMailboxRepo) Create(ctx context.Context, row *model.FriendMailboxRequest) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *FriendMailboxRepo) GetByRequestID(ctx context.Context, requestID string) (*model.FriendMailboxRequest, error) {
	var row model.FriendMailboxRequest
	if err := r.db.WithContext(ctx).First(&row, "request_id = ?", requestID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListForPeer 返回与该 peer 相关的记录（发件或收件），按更新时间倒序。
func (r *FriendMailboxRepo) ListForPeer(ctx context.Context, peerID string, limit int) ([]model.FriendMailboxRequest, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []model.FriendMailboxRequest
	q := r.db.WithContext(ctx).
		Where("from_peer_id = ? OR to_peer_id = ?", peerID, peerID).
		Order("updated_at DESC").
		Limit(limit)
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *FriendMailboxRepo) UpdateStateIfToPeer(ctx context.Context, requestID, toPeerID, newState string) (*model.FriendMailboxRequest, error) {
	var row model.FriendMailboxRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&row, "request_id = ?", requestID).Error; err != nil {
			return err
		}
		if strings.TrimSpace(row.ToPeerID) != strings.TrimSpace(toPeerID) {
			return ErrNotMailboxRecipient
		}
		if row.State != model.FriendMailboxStatePending {
			return ErrMailboxNotPending
		}
		row.State = newState
		return tx.Save(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

var (
	ErrNotMailboxRecipient = errors.New("not_mailbox_recipient")
	ErrMailboxNotPending   = errors.New("mailbox_request_not_pending")
)
