package repo

import (
	"context"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) DB() *gorm.DB {
	return r.db
}

func (r *UserRepo) GetByID(ctx context.Context, id uint64) (*model.ServerUser, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	var user model.ServerUser
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) GetByPeerID(ctx context.Context, peerID string) (*model.ServerUser, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	var user model.ServerUser
	if err := r.db.WithContext(ctx).First(&user, "peer_id = ?", peerID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) ListByIDs(ctx context.Context, ids []uint64) (map[uint64]model.ServerUser, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	result := make(map[uint64]model.ServerUser, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var users []model.ServerUser
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}

	for _, user := range users {
		result[user.ID] = user
	}
	return result, nil
}

func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]model.ServerUser, error) {
	if err := r.ensureTable(); err != nil {
		return nil, err
	}
	var users []model.ServerUser
	query := r.db.WithContext(ctx).Order("id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) Create(ctx context.Context, user *model.ServerUser) error {
	if err := r.ensureTable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) UpdateProfile(ctx context.Context, user *model.ServerUser) error {
	if err := r.ensureTable(); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(user).Updates(map[string]any{
		"username":        user.Username,
		"display_name":    user.DisplayName,
		"avatar_cid":      user.AvatarCID,
		"bio":             user.Bio,
		"status":          user.Status,
		"profile_version": user.ProfileVersion,
	}).Error
}

func (r *UserRepo) ensureTable() error {
	if r.db.Migrator().HasTable(&model.ServerUser{}) {
		return nil
	}
	return r.db.Migrator().CreateTable(&model.ServerUser{})
}
