package service

import (
	"context"
	"strings"

	"meshchat-server/internal/ipfs"
	"meshchat-server/internal/model"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"
)

type FileService struct {
	files *repo.FileRepo
	ipfs  ipfs.Client
}

func NewFileService(files *repo.FileRepo, ipfs ipfs.Client) *FileService {
	return &FileService{
		files: files,
		ipfs:  ipfs,
	}
}

func (s *FileService) Register(ctx context.Context, userID uint64, input RegisterFileInput) (*model.File, error) {
	if strings.TrimSpace(input.CID) == "" || strings.TrimSpace(input.MIMEType) == "" || input.Size <= 0 {
		return nil, apperrors.New(400, "invalid_file", "cid, mime_type and size are required")
	}
	if err := s.ipfs.RegisterMetadata(ctx, input.CID); err != nil {
		return nil, apperrors.New(400, "invalid_cid", "cid is invalid")
	}
	if input.ThumbnailCID != "" {
		if err := s.ipfs.RegisterMetadata(ctx, input.ThumbnailCID); err != nil {
			return nil, apperrors.New(400, "invalid_thumbnail_cid", "thumbnail_cid is invalid")
		}
	}

	file := &model.File{
		CID:             strings.TrimSpace(input.CID),
		MIMEType:        strings.TrimSpace(input.MIMEType),
		Size:            input.Size,
		Width:           input.Width,
		Height:          input.Height,
		DurationSeconds: input.DurationSeconds,
		FileName:        strings.TrimSpace(input.FileName),
		ThumbnailCID:    strings.TrimSpace(input.ThumbnailCID),
		CreatedByUserID: userID,
	}
	if err := s.files.Create(ctx, file); err != nil {
		if duplicateConstraintError(err) {
			existing, getErr := s.files.GetByCID(ctx, file.CID)
			return existing, getErr
		}
		return nil, err
	}
	return file, nil
}
