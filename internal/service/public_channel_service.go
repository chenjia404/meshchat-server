package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"meshchat-server/internal/events"
	"meshchat-server/internal/model"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	publicChannelChangeTypeMessage = "message"
	publicChannelChangeTypeProfile = "profile"

	publicChannelMessageTypeText    = "text"
	publicChannelMessageTypeImage   = "image"
	publicChannelMessageTypeVideo   = "video"
	publicChannelMessageTypeAudio   = "audio"
	publicChannelMessageTypeFile    = "file"
	publicChannelMessageTypeSystem  = "system"
	publicChannelMessageTypeDeleted = "deleted"

	publicChannelDefaultPageLimit = 20
	publicChannelDefaultSyncLimit = 100
	publicChannelMaxRetention     = 60 * 24 * 365
)

type PublicChannelService struct {
	channels  *repo.PublicChannelRepo
	users     *repo.UserRepo
	publisher EventPublisher
}

func NewPublicChannelService(channels *repo.PublicChannelRepo, users *repo.UserRepo, publisher EventPublisher) *PublicChannelService {
	return &PublicChannelService{channels: channels, users: users, publisher: publisher}
}

func buildPublicChannelID(ownerPeerID string, channelUUID uuid.UUID) string {
	return strings.TrimSpace(ownerPeerID) + ":" + channelUUID.String()
}

func normalizePublicChannelRetention(minutes int) int {
	if minutes < 0 {
		return 0
	}
	if minutes > publicChannelMaxRetention {
		return publicChannelMaxRetention
	}
	return minutes
}

func validatePublicChannelRetention(minutes int) error {
	if minutes < 0 || minutes > publicChannelMaxRetention {
		return payloadValidationError("message_retention_minutes is out of range")
	}
	return nil
}

func normalizePublicChannelMessageType(content PublicChannelMessageContent, requested string, deleted bool) string {
	if deleted {
		return publicChannelMessageTypeDeleted
	}
	if len(content.Files) == 0 {
		if strings.TrimSpace(requested) == "" {
			return publicChannelMessageTypeText
		}
		return strings.TrimSpace(requested)
	}
	if requested = strings.TrimSpace(requested); requested != "" {
		return requested
	}
	mimeType := strings.ToLower(strings.TrimSpace(content.Files[0].MIMEType))
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return publicChannelMessageTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return publicChannelMessageTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return publicChannelMessageTypeAudio
	default:
		return publicChannelMessageTypeFile
	}
}

func normalizePublicChannelContent(text string, files []PublicChannelFile) PublicChannelMessageContent {
	out := PublicChannelMessageContent{Text: strings.TrimSpace(text)}
	if len(files) == 0 {
		return out
	}
	out.Files = make([]PublicChannelFile, 0, len(files))
	for _, item := range files {
		file := item
		file.FileName = strings.TrimSpace(file.FileName)
		file.MIMEType = strings.TrimSpace(file.MIMEType)
		file.SHA256 = strings.TrimSpace(file.SHA256)
		file.BlobID = strings.TrimSpace(file.BlobID)
		file.URL = strings.TrimSpace(file.URL)
		file.FileID = strings.TrimSpace(file.FileID)
		out.Files = append(out.Files, file)
	}
	return out
}

func publicChannelAvatarJSON(v PublicChannelAvatar) (datatypes.JSON, error) {
	raw, err := json.Marshal(v)
	return datatypes.JSON(raw), err
}

func publicChannelContentJSON(v PublicChannelMessageContent) (datatypes.JSON, error) {
	raw, err := json.Marshal(v)
	return datatypes.JSON(raw), err
}

func decodePublicChannelAvatar(raw datatypes.JSON) (PublicChannelAvatar, error) {
	var out PublicChannelAvatar
	if len(raw) == 0 {
		return out, nil
	}
	err := json.Unmarshal(raw, &out)
	return out, err
}

func decodePublicChannelContent(raw datatypes.JSON) (PublicChannelMessageContent, error) {
	var out PublicChannelMessageContent
	if len(raw) == 0 {
		return out, nil
	}
	err := json.Unmarshal(raw, &out)
	return out, err
}

func (s *PublicChannelService) currentUser(ctx context.Context, userID uint64) (*model.ServerUser, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "user_not_found", "user not found")
		}
		return nil, err
	}
	return user, nil
}

func (s *PublicChannelService) toProfileView(ch *model.PublicChannel) (PublicChannelProfileView, error) {
	avatar, err := decodePublicChannelAvatar(ch.AvatarJSON)
	if err != nil {
		return PublicChannelProfileView{}, err
	}
	return PublicChannelProfileView{
		ChannelID:               ch.ChannelID,
		OwnerPeerID:             ch.OwnerPeerID,
		OwnerVersion:            ch.OwnerVersion,
		Name:                    ch.Name,
		Avatar:                  avatar,
		Bio:                     ch.Bio,
		MessageRetentionMinutes: ch.MessageRetentionMinutes,
		ProfileVersion:          ch.ProfileVersion,
		CreatedAt:               ch.CreatedAtUnix,
		UpdatedAt:               ch.UpdatedAtUnix,
		Signature:               ch.ProfileSignature,
	}, nil
}

func (s *PublicChannelService) toHeadView(ch *model.PublicChannel) PublicChannelHeadView {
	return PublicChannelHeadView{
		ChannelID:      ch.ChannelID,
		OwnerPeerID:    ch.OwnerPeerID,
		OwnerVersion:   ch.OwnerVersion,
		LastMessageID:  ch.LastMessageID,
		ProfileVersion: ch.ProfileVersion,
		LastSeq:        ch.LastSeq,
		UpdatedAt:      ch.HeadUpdatedAtUnix,
		Signature:      ch.HeadSignature,
	}
}

func (s *PublicChannelService) toMessageView(msg *model.PublicChannelMessage, channelID string) (PublicChannelMessageView, error) {
	content, err := decodePublicChannelContent(msg.ContentJSON)
	if err != nil {
		return PublicChannelMessageView{}, err
	}
	return PublicChannelMessageView{
		ChannelID:     channelID,
		MessageID:     msg.MessageID,
		Version:       msg.Version,
		Seq:           msg.Seq,
		OwnerVersion:  msg.OwnerVersion,
		CreatorPeerID: msg.CreatorPeerID,
		AuthorPeerID:  msg.AuthorPeerID,
		CreatedAt:     msg.CreatedAtUnix,
		UpdatedAt:     msg.UpdatedAtUnix,
		IsDeleted:     msg.IsDeleted,
		MessageType:   msg.MessageType,
		Content:       content,
		Signature:     msg.Signature,
	}, nil
}

func (s *PublicChannelService) toSyncStateView(ch *model.PublicChannel, sub *model.PublicChannelSubscription) PublicChannelSyncStateView {
	out := PublicChannelSyncStateView{ChannelID: ch.ChannelID}
	if sub == nil {
		return out
	}
	out.LastSeenSeq = sub.LastSeenSeq
	out.LastSyncedSeq = sub.LastSyncedSeq
	out.LatestLoadedMessageID = sub.LatestLoadedMessageID
	out.OldestLoadedMessageID = sub.OldestLoadedMessageID
	out.UnreadCount = sub.UnreadCount
	out.Subscribed = sub.Subscribed
	out.UpdatedAt = sub.UpdatedAtUnix
	return out
}

func (s *PublicChannelService) toSummaryView(ctx context.Context, userID uint64, ch *model.PublicChannel) (*PublicChannelSummaryView, error) {
	profile, err := s.toProfileView(ch)
	if err != nil {
		return nil, err
	}
	head := s.toHeadView(ch)
	sub, err := s.channels.GetSubscription(ctx, ch.ID, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &PublicChannelSummaryView{
		Profile: profile,
		Head:    head,
		Sync:    s.toSyncStateView(ch, sub),
	}, nil
}

func (s *PublicChannelService) getChannel(ctx context.Context, channelID string) (*model.PublicChannel, error) {
	ch, err := s.channels.GetByChannelID(ctx, strings.TrimSpace(channelID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "channel_not_found", "channel not found")
		}
		return nil, err
	}
	return ch, nil
}

func (s *PublicChannelService) requireOwner(ctx context.Context, userID uint64, channelID string) (*model.PublicChannel, error) {
	ch, err := s.getChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch.OwnerUserID != userID {
		return nil, apperrors.New(403, "forbidden", "only channel owner can mutate channel")
	}
	return ch, nil
}

func (s *PublicChannelService) CanAccessChannel(ctx context.Context, userID uint64, channelID string) bool {
	if userID == 0 || strings.TrimSpace(channelID) == "" {
		return false
	}
	_, err := s.channels.GetByChannelID(ctx, strings.TrimSpace(channelID))
	return err == nil
}

func (s *PublicChannelService) CreateChannel(ctx context.Context, userID uint64, input CreatePublicChannelInput) (*PublicChannelSummaryView, error) {
	user, err := s.currentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, payloadValidationError("name is required")
	}
	input.MessageRetentionMinutes = normalizePublicChannelRetention(input.MessageRetentionMinutes)
	if err := validatePublicChannelRetention(input.MessageRetentionMinutes); err != nil {
		return nil, err
	}
	chID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	nowUnix := now.Unix()
	avatarJSON, err := publicChannelAvatarJSON(input.Avatar)
	if err != nil {
		return nil, err
	}
	channel := &model.PublicChannel{
		ChannelID:               buildPublicChannelID(user.PeerID, chID),
		OwnerUserID:             userID,
		OwnerPeerID:             user.PeerID,
		OwnerVersion:            1,
		Name:                    name,
		AvatarJSON:              avatarJSON,
		Bio:                     strings.TrimSpace(input.Bio),
		MessageRetentionMinutes: input.MessageRetentionMinutes,
		ProfileVersion:          1,
		LastMessageID:           0,
		LastSeq:                 0,
		CreatedAtUnix:           nowUnix,
		UpdatedAtUnix:           nowUnix,
		HeadUpdatedAtUnix:       nowUnix,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	sub := &model.PublicChannelSubscription{
		UserID:                userID,
		LastSeenSeq:           0,
		LastSyncedSeq:         0,
		LatestLoadedMessageID: 0,
		OldestLoadedMessageID: 0,
		UnreadCount:           0,
		Subscribed:            true,
		UpdatedAtUnix:         nowUnix,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.channels.CreateChannel(ctx, channel, sub); err != nil {
		return nil, err
	}
	return s.toSummaryView(ctx, userID, channel)
}

func (s *PublicChannelService) UpdateChannel(ctx context.Context, userID uint64, channelID string, input UpdatePublicChannelProfileInput) (*PublicChannelSummaryView, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, payloadValidationError("name is required")
	}
	ch, err := s.requireOwner(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	avatarJSON, err := publicChannelAvatarJSON(input.Avatar)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	nowUnix := now.Unix()
	_, err = s.channels.UpdateChannelProfile(ctx, ch.ChannelID, func(locked *model.PublicChannel, change *model.PublicChannelChange) error {
		retention := locked.MessageRetentionMinutes
		if input.MessageRetentionMinutes != nil {
			retention = normalizePublicChannelRetention(*input.MessageRetentionMinutes)
			if err := validatePublicChannelRetention(retention); err != nil {
				return err
			}
		}
		locked.Name = strings.TrimSpace(input.Name)
		locked.Bio = strings.TrimSpace(input.Bio)
		locked.AvatarJSON = avatarJSON
		locked.MessageRetentionMinutes = retention
		locked.ProfileVersion++
		locked.LastSeq++
		locked.UpdatedAtUnix = nowUnix
		locked.HeadUpdatedAtUnix = nowUnix
		locked.UpdatedAt = now
		change.Seq = locked.LastSeq
		change.ChangeType = publicChannelChangeTypeProfile
		change.ProfileVersion = &locked.ProfileVersion
		change.CreatedAtUnix = nowUnix
		change.CreatedAt = now
		change.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}
	updated, err := s.getChannel(ctx, ch.ChannelID)
	if err != nil {
		return nil, err
	}
	if err := s.publishPublicChannelProfileEvent(ctx, updated, events.EventPublicChannelProfileUpdated); err != nil {
		return nil, err
	}
	return s.toSummaryView(ctx, userID, updated)
}

func (s *PublicChannelService) validateUpsertMessageInput(input UpsertPublicChannelMessageInput) (PublicChannelMessageContent, string, error) {
	content := normalizePublicChannelContent(input.Text, input.Files)
	if strings.TrimSpace(content.Text) == "" && len(content.Files) == 0 {
		return PublicChannelMessageContent{}, "", payloadValidationError("message text or files are required")
	}
	for _, file := range content.Files {
		if strings.TrimSpace(file.FileName) == "" {
			return PublicChannelMessageContent{}, "", payloadValidationError("file_name is required")
		}
	}
	msgType := normalizePublicChannelMessageType(content, input.MessageType, false)
	return content, msgType, nil
}

func (s *PublicChannelService) CreateMessage(ctx context.Context, userID uint64, channelID string, input UpsertPublicChannelMessageInput) (*PublicChannelMessageView, error) {
	ch, err := s.requireOwner(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	content, messageType, err := s.validateUpsertMessageInput(input)
	if err != nil {
		return nil, err
	}
	contentJSON, err := publicChannelContentJSON(content)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	nowUnix := now.Unix()
	msg, _, err := s.channels.CreateMessage(ctx, ch.ChannelID, func(out *model.PublicChannelMessage, locked *model.PublicChannel, change *model.PublicChannelChange) error {
		locked.LastMessageID++
		locked.LastSeq++
		locked.UpdatedAtUnix = nowUnix
		locked.HeadUpdatedAtUnix = nowUnix
		locked.UpdatedAt = now
		out.MessageID = locked.LastMessageID
		out.Version = 1
		out.Seq = locked.LastSeq
		out.OwnerVersion = locked.OwnerVersion
		out.CreatorPeerID = locked.OwnerPeerID
		out.AuthorPeerID = locked.OwnerPeerID
		out.CreatedAtUnix = nowUnix
		out.UpdatedAtUnix = nowUnix
		out.IsDeleted = false
		out.MessageType = messageType
		out.ContentJSON = contentJSON
		out.CreatedAt = now
		out.UpdatedAt = now
		change.Seq = out.Seq
		change.ChangeType = publicChannelChangeTypeMessage
		change.MessageID = &out.MessageID
		change.Version = &out.Version
		change.IsDeleted = &out.IsDeleted
		change.CreatedAtUnix = nowUnix
		change.CreatedAt = now
		change.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.toMessageView(msg, ch.ChannelID)
	if err != nil {
		return nil, err
	}
	if err := s.publishPublicChannelMessageEvent(ctx, ch.ChannelID, msg.MessageID, events.EventPublicChannelMessageCreated); err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *PublicChannelService) UpdateMessage(ctx context.Context, userID uint64, channelID string, messageID int64, input UpsertPublicChannelMessageInput) (*PublicChannelMessageView, error) {
	ch, err := s.requireOwner(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	content, messageType, err := s.validateUpsertMessageInput(input)
	if err != nil {
		return nil, err
	}
	contentJSON, err := publicChannelContentJSON(content)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	nowUnix := now.Unix()
	msg, _, err := s.channels.UpdateMessage(ctx, ch.ChannelID, messageID, func(out *model.PublicChannelMessage, locked *model.PublicChannel, change *model.PublicChannelChange) error {
		if out.IsDeleted {
			return apperrors.New(400, "message_deleted", "message has been deleted")
		}
		out.Version++
		locked.LastSeq++
		locked.UpdatedAtUnix = nowUnix
		locked.HeadUpdatedAtUnix = nowUnix
		locked.UpdatedAt = now
		out.Seq = locked.LastSeq
		out.OwnerVersion = locked.OwnerVersion
		out.AuthorPeerID = locked.OwnerPeerID
		out.UpdatedAtUnix = nowUnix
		out.MessageType = messageType
		out.ContentJSON = contentJSON
		out.UpdatedAt = now
		change.Seq = out.Seq
		change.ChangeType = publicChannelChangeTypeMessage
		change.MessageID = &out.MessageID
		change.Version = &out.Version
		change.IsDeleted = &out.IsDeleted
		change.CreatedAtUnix = nowUnix
		change.CreatedAt = now
		change.UpdatedAt = now
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "message_not_found", "message not found")
		}
		return nil, err
	}
	view, err := s.toMessageView(msg, ch.ChannelID)
	if err != nil {
		return nil, err
	}
	if err := s.publishPublicChannelMessageEvent(ctx, ch.ChannelID, msg.MessageID, events.EventPublicChannelMessageUpdated); err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *PublicChannelService) DeleteMessage(ctx context.Context, userID uint64, channelID string, messageID int64) (*PublicChannelMessageView, error) {
	ch, err := s.requireOwner(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	current, err := s.channels.GetMessage(ctx, ch.ID, messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "message_not_found", "message not found")
		}
		return nil, err
	}
	if current.IsDeleted {
		view, viewErr := s.toMessageView(current, ch.ChannelID)
		if viewErr != nil {
			return nil, viewErr
		}
		return &view, nil
	}
	now := time.Now().UTC()
	nowUnix := now.Unix()
	msg, _, err := s.channels.UpdateMessage(ctx, ch.ChannelID, messageID, func(out *model.PublicChannelMessage, locked *model.PublicChannel, change *model.PublicChannelChange) error {
		contentJSON, err := publicChannelContentJSON(PublicChannelMessageContent{})
		if err != nil {
			return err
		}
		out.Version++
		out.IsDeleted = true
		out.MessageType = publicChannelMessageTypeDeleted
		out.ContentJSON = contentJSON
		out.AuthorPeerID = locked.OwnerPeerID
		out.OwnerVersion = locked.OwnerVersion
		out.UpdatedAtUnix = nowUnix
		out.UpdatedAt = now
		locked.LastSeq++
		locked.UpdatedAtUnix = nowUnix
		locked.HeadUpdatedAtUnix = nowUnix
		locked.UpdatedAt = now
		out.Seq = locked.LastSeq
		change.Seq = out.Seq
		change.ChangeType = publicChannelChangeTypeMessage
		change.MessageID = &out.MessageID
		change.Version = &out.Version
		change.IsDeleted = &out.IsDeleted
		change.CreatedAtUnix = nowUnix
		change.CreatedAt = now
		change.UpdatedAt = now
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "message_not_found", "message not found")
		}
		return nil, err
	}
	view, err := s.toMessageView(msg, ch.ChannelID)
	if err != nil {
		return nil, err
	}
	if err := s.publishPublicChannelMessageEvent(ctx, ch.ChannelID, msg.MessageID, events.EventPublicChannelMessageDeleted); err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *PublicChannelService) GetChannelSummary(ctx context.Context, userID uint64, channelID string) (*PublicChannelSummaryView, error) {
	ch, err := s.getChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return s.toSummaryView(ctx, userID, ch)
}

func (s *PublicChannelService) GetChannelHead(ctx context.Context, channelID string) (*PublicChannelHeadView, error) {
	ch, err := s.getChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	head := s.toHeadView(ch)
	return &head, nil
}

func (s *PublicChannelService) GetChannelMessage(ctx context.Context, channelID string, messageID int64) (*PublicChannelMessageView, error) {
	ch, err := s.getChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	msg, err := s.channels.GetMessage(ctx, ch.ID, messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(404, "message_not_found", "message not found")
		}
		return nil, err
	}
	view, err := s.toMessageView(msg, ch.ChannelID)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *PublicChannelService) ListChannelMessages(ctx context.Context, channelID string, beforeMessageID int64, limit int) ([]PublicChannelMessageView, error) {
	ch, err := s.getChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = publicChannelDefaultPageLimit
	}
	rows, err := s.channels.ListMessages(ctx, ch.ID, beforeMessageID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]PublicChannelMessageView, 0, len(rows))
	for i := range rows {
		item, err := s.toMessageView(&rows[i], ch.ChannelID)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *PublicChannelService) ListChannelChanges(ctx context.Context, channelID string, afterSeq int64, limit int) (*PublicChannelGetChangesResponse, error) {
	ch, err := s.getChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = publicChannelDefaultSyncLimit
	}
	rows, err := s.channels.ListChanges(ctx, ch.ID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	resp := &PublicChannelGetChangesResponse{
		ChannelID:      ch.ChannelID,
		CurrentLastSeq: ch.LastSeq,
	}
	for _, row := range rows {
		item := PublicChannelChangeView{
			ChannelID:  ch.ChannelID,
			Seq:        row.Seq,
			ChangeType: row.ChangeType,
			CreatedAt:  row.CreatedAtUnix,
		}
		if row.MessageID != nil {
			v := *row.MessageID
			item.MessageID = &v
		}
		if row.Version != nil {
			v := *row.Version
			item.Version = &v
		}
		if row.IsDeleted != nil {
			v := *row.IsDeleted
			item.IsDeleted = &v
		}
		if row.ProfileVersion != nil {
			v := *row.ProfileVersion
			item.ProfileVersion = &v
		}
		resp.Items = append(resp.Items, item)
	}
	if len(resp.Items) > limit {
		resp.HasMore = true
		resp.Items = resp.Items[:limit]
	}
	if len(resp.Items) == 0 {
		resp.NextAfterSeq = afterSeq
	} else {
		resp.NextAfterSeq = resp.Items[len(resp.Items)-1].Seq
	}
	return resp, nil
}

func (s *PublicChannelService) ListChannelsByOwner(ctx context.Context, userID uint64, ownerPeerID string) ([]PublicChannelSummaryView, error) {
	rows, err := s.channels.ListChannelsByOwnerPeerID(ctx, strings.TrimSpace(ownerPeerID))
	if err != nil {
		return nil, err
	}
	out := make([]PublicChannelSummaryView, 0, len(rows))
	for i := range rows {
		item, err := s.toSummaryView(ctx, userID, &rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *PublicChannelService) ListSubscribedChannels(ctx context.Context, userID uint64) ([]PublicChannelSummaryView, error) {
	rows, err := s.channels.ListSubscribedChannelsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PublicChannelSummaryView, 0, len(rows))
	for i := range rows {
		item, err := s.toSummaryView(ctx, userID, &rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *PublicChannelService) SubscribeChannel(ctx context.Context, userID uint64, channelID string, input SubscribePublicChannelInput) (*PublicChannelSubscribeResult, error) {
	ch, err := s.getChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if input.LastSeenSeq < 0 {
		input.LastSeenSeq = 0
	}
	sub, err := s.channels.UpsertSubscription(ctx, ch.ID, userID, input.LastSeenSeq, time.Now().Unix(), true)
	if err != nil {
		return nil, err
	}
	items, err := s.ListChannelMessages(ctx, ch.ChannelID, 0, publicChannelDefaultPageLimit)
	if err != nil {
		return nil, err
	}
	profile, err := s.toProfileView(ch)
	if err != nil {
		return nil, err
	}
	return &PublicChannelSubscribeResult{
		Profile:  profile,
		Head:     s.toHeadView(ch),
		Messages: items,
		Providers: []PublicChannelProviderView{
			{
				ChannelID:     ch.ChannelID,
				PeerID:        "meshchat-server",
				Source:        "meshchat-server",
				UpdatedAt:     sub.UpdatedAtUnix,
				LastSuccessAt: sub.UpdatedAtUnix,
			},
		},
	}, nil
}

func (s *PublicChannelService) BuildRealtimePublicChannelEvent(ctx context.Context, userID uint64, env events.Envelope) (*RealtimeEnvelope, error) {
	if strings.TrimSpace(env.ChannelID) == "" {
		return nil, nil
	}
	switch env.Type {
	case events.EventPublicChannelProfileUpdated:
		summary, err := s.GetChannelSummary(ctx, userID, env.ChannelID)
		if err != nil {
			return nil, err
		}
		return &RealtimeEnvelope{
			Type: env.Type,
			Data: map[string]any{
				"channel_id": env.ChannelID,
				"profile":    summary.Profile,
				"head":       summary.Head,
			},
		}, nil
	case events.EventPublicChannelMessageCreated, events.EventPublicChannelMessageUpdated, events.EventPublicChannelMessageDeleted:
		if strings.TrimSpace(env.MessageID) == "" {
			return nil, nil
		}
		var msgID int64
		if _, err := fmt.Sscan(env.MessageID, &msgID); err != nil {
			return nil, err
		}
		msg, err := s.GetChannelMessage(ctx, env.ChannelID, msgID)
		if err != nil {
			return nil, err
		}
		return &RealtimeEnvelope{
			Type: env.Type,
			Data: map[string]any{
				"channel_id": env.ChannelID,
				"message":    msg,
			},
		}, nil
	default:
		return nil, nil
	}
}

func (s *PublicChannelService) publishPublicChannelProfileEvent(ctx context.Context, ch *model.PublicChannel, typ string) error {
	if s.publisher == nil {
		return nil
	}
	return s.publisher.Publish(ctx, events.Envelope{
		Type:      typ,
		ChannelID: ch.ChannelID,
		At:        time.Now().UTC(),
	})
}

func (s *PublicChannelService) publishPublicChannelMessageEvent(ctx context.Context, channelID string, messageID int64, typ string) error {
	if s.publisher == nil {
		return nil
	}
	return s.publisher.Publish(ctx, events.Envelope{
		Type:      typ,
		ChannelID: channelID,
		MessageID: fmt.Sprintf("%d", messageID),
		At:        time.Now().UTC(),
	})
}
