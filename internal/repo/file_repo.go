package repo

import (
	"context"

	"meshchat-server/internal/model"

	"gorm.io/gorm"
)

type FileRepo struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

func (r *FileRepo) Create(ctx context.Context, file *model.File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *FileRepo) GetByCID(ctx context.Context, cid string) (*model.File, error) {
	var file model.File
	if err := r.db.WithContext(ctx).First(&file, "cid = ?", cid).Error; err != nil {
		return nil, err
	}
	return &file, nil
}
