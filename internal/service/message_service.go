package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"meshchat-server/internal/events"
	"meshchat-server/internal/ipfs"
	"meshchat-server/internal/model"
	"meshchat-server/internal/redisx"
	"meshchat-server/internal/repo"
	"meshchat-server/pkg/apperrors"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type MessageService struct {
	groups    *repo.GroupRepo
	messages  *repo.MessageRepo
	redis     *redis.Client
	ipfs      ipfs.Client
	publisher EventPublisher
}

func NewMessageService(groups *repo.GroupRepo, messages *repo.MessageRepo, redis *redis.Client, ipfs ipfs.Client, publisher EventPublisher) *MessageService {
	return &MessageService{
		groups:    groups,
		messages:  messages,
		redis:     redis,
		ipfs:      ipfs,
		publisher: publisher,
	}
}

func (s *MessageService) ListMessages(ctx context.Context, userID uint64, groupID string, beforeSeq uint64, limit int) ([]MessageView, error) {
	group, member, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	_ = member
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	items, err := s.messages.ListGroupMessages(ctx, group.ID, beforeSeq, limit)
	if err != nil {
		return nil, err
	}

	views := make([]MessageView, 0, len(items))
	now := time.Now().UTC()
	for i := len(items) - 1; i >= 0; i-- {
		if !isVisibleByTTL(group.MessageTTLSeconds, items[i].CreatedAt, now) {
			continue
		}
		view, err := s.toMessageViewForUser(ctx, userID, *group, items[i])
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *MessageService) ListMessagesForAdmin(ctx context.Context, groupID string, beforeSeq uint64, limit int) ([]MessageView, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	items, err := s.messages.ListGroupMessages(ctx, group.ID, beforeSeq, limit)
	if err != nil {
		return nil, err
	}

	views := make([]MessageView, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		view, err := s.toMessageViewForAdmin(ctx, *group, items[i])
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *MessageService) SendMessage(ctx context.Context, userID uint64, groupID string, input SendMessageInput) (*MessageView, error) {
	group, member, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if member.Status == model.MemberStatusBanned {
		return nil, apperrors.New(403, "member_banned", "member is banned")
	}
	if member.MutedUntil != nil && member.MutedUntil.After(time.Now().UTC()) {
		return nil, apperrors.New(403, "member_muted", "member is muted")
	}
	if input.ReplyToMessageID != nil && input.ForwardFromMessageID != nil {
		return nil, apperrors.New(400, "invalid_reference", "reply and forward cannot be used together")
	}

	perms := model.EffectivePermissions(member.Role, group.DefaultPermissions, member.PermissionsAllow, member.PermissionsDeny)
	if err := s.enforceSendPermission(input.ContentType, perms); err != nil {
		return nil, err
	}
	if err := s.enforceSlowMode(ctx, *group, *member, perms); err != nil {
		return nil, err
	}

	payloadJSON, err := s.validatePayload(ctx, input.ContentType, input.Payload)
	if err != nil {
		return nil, err
	}
	if err := s.validateReferences(ctx, userID, *group, input.ReplyToMessageID, input.ForwardFromMessageID, perms); err != nil {
		return nil, err
	}

	message := &model.GroupMessage{
		GroupID:              group.ID,
		MessageID:            uuid.NewString(),
		SenderUserID:         userID,
		ContentType:          input.ContentType,
		PayloadJSON:          payloadJSON,
		ReplyToMessageID:     input.ReplyToMessageID,
		ForwardFromMessageID: input.ForwardFromMessageID,
		Status:               model.MessageStatusNormal,
		Signature:            input.Signature,
	}

	if err := s.groups.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		seq, err := s.groups.IncrementLastSeq(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		message.Seq = seq
		return s.messages.Create(ctx, tx, message)
	}); err != nil {
		return nil, err
	}

	if group.MessageCooldownSeconds > 0 && !hasPermission(perms, model.PermBypassSlowmode) {
		_ = s.redis.Set(ctx, redisx.CooldownKey(group.GroupID.String(), userID), "1", time.Duration(group.MessageCooldownSeconds)*time.Second).Err()
	}

	saved, err := s.messages.GetByGroupAndMessageID(ctx, group.ID, message.MessageID)
	if err != nil {
		return nil, err
	}
	view, err := s.toMessageViewForUser(ctx, userID, *group, *saved)
	if err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:      events.EventGroupMessageCreated,
		GroupID:   group.GroupID.String(),
		MessageID: message.MessageID,
		At:        time.Now().UTC(),
	})
	return &view, nil
}

func (s *MessageService) EditMessage(ctx context.Context, userID uint64, groupID, messageID string, input EditMessageInput) (*MessageView, error) {
	group, _, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}

	var updated model.GroupMessage
	if err := s.messages.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		message, err := s.messages.GetByMessageIDForUpdate(ctx, tx, messageID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(404, "message_not_found", "message not found")
			}
			return err
		}
		if message.GroupID != group.ID {
			return apperrors.New(404, "message_not_found", "message not found")
		}
		if message.SenderUserID != userID {
			return apperrors.New(403, "forbidden", "only sender can edit the message")
		}
		if message.Status == model.MessageStatusDeleted {
			return apperrors.New(400, "message_deleted", "deleted message cannot be edited")
		}

		newPayload, err := s.validatePayload(ctx, message.ContentType, input.Payload)
		if err != nil {
			return err
		}

		edit := &model.GroupMessageEdit{
			GroupID:        group.ID,
			MessageID:      message.MessageID,
			EditorUserID:   userID,
			OldPayloadJSON: message.PayloadJSON,
			NewPayloadJSON: newPayload,
		}
		now := time.Now().UTC()
		message.PayloadJSON = newPayload
		message.EditCount++
		message.LastEditedAt = &now
		message.LastEditedByUserID = &userID
		if err := s.messages.CreateEdit(ctx, tx, edit); err != nil {
			return err
		}
		if err := s.messages.Save(ctx, tx, message); err != nil {
			return err
		}
		updated = *message
		return nil
	}); err != nil {
		return nil, err
	}

	view, err := s.toMessageViewForUser(ctx, userID, *group, updated)
	if err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:      events.EventGroupMessageEdited,
		GroupID:   groupID,
		MessageID: messageID,
		At:        time.Now().UTC(),
	})
	return &view, nil
}

func (s *MessageService) RetractMessage(ctx context.Context, userID uint64, groupID, messageID string) (*MessageView, error) {
	return s.deleteMessage(ctx, userID, groupID, messageID, true)
}

func (s *MessageService) DeleteMessage(ctx context.Context, userID uint64, groupID, messageID string) (*MessageView, error) {
	return s.deleteMessage(ctx, userID, groupID, messageID, false)
}

func (s *MessageService) BuildMessageEventForUser(ctx context.Context, viewerID uint64, eventType, messageID string) (*RealtimeEnvelope, error) {
	message, err := s.messages.GetByMessageID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	group, member, err := s.requireActiveMembership(ctx, viewerID, message.Group.GroupID.String())
	if err != nil {
		if apperrors.HTTPStatus(err) == 403 || apperrors.HTTPStatus(err) == 404 {
			return nil, nil
		}
		return nil, err
	}
	_ = member

	view, err := s.toMessageViewForUser(ctx, viewerID, *group, *message)
	if err != nil {
		return nil, err
	}
	return &RealtimeEnvelope{
		Type: eventType,
		Data: view,
	}, nil
}

func (s *MessageService) requireActiveMembership(ctx context.Context, userID uint64, groupID string) (*model.Group, *model.GroupMember, error) {
	group, err := s.groups.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, apperrors.New(404, "group_not_found", "group not found")
		}
		return nil, nil, err
	}
	member, err := s.groups.GetMember(ctx, group.ID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, apperrors.New(403, "not_group_member", "user is not an active group member")
		}
		return nil, nil, err
	}
	if member.Status != model.MemberStatusActive {
		return nil, nil, apperrors.New(403, "member_inactive", "member is not active in this group")
	}
	if group.Status != model.GroupStatusActive {
		return nil, nil, apperrors.New(403, "group_closed", "group is closed")
	}
	return group, member, nil
}

func (s *MessageService) enforceSendPermission(contentType string, perms int64) error {
	required := map[string]int64{
		model.MessageContentTypeText:    model.PermSendText,
		model.MessageContentTypeImage:   model.PermSendImage,
		model.MessageContentTypeVideo:   model.PermSendVideo,
		model.MessageContentTypeVoice:   model.PermSendVoice,
		model.MessageContentTypeFile:    model.PermSendFile,
		model.MessageContentTypeForward: model.PermForward,
	}
	perm, ok := required[contentType]
	if !ok {
		return apperrors.New(400, "invalid_content_type", "unsupported content_type")
	}
	if !hasPermission(perms, perm) {
		return apperrors.New(403, "forbidden", "missing permission for this message type")
	}
	return nil
}

func (s *MessageService) enforceSlowMode(ctx context.Context, group model.Group, member model.GroupMember, perms int64) error {
	if group.MessageCooldownSeconds <= 0 || member.Role == model.RoleOwner || hasPermission(perms, model.PermBypassSlowmode) {
		return nil
	}
	ttl, err := s.redis.TTL(ctx, redisx.CooldownKey(group.GroupID.String(), member.UserID)).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if ttl > 0 {
		return apperrors.New(429, "slow_mode", fmt.Sprintf("slow mode active, retry after %d seconds", int(ttl.Seconds())+1))
	}
	return nil
}

func (s *MessageService) validateReferences(ctx context.Context, userID uint64, group model.Group, replyTo, forwardFrom *string, perms int64) error {
	now := time.Now().UTC()
	if replyTo != nil {
		if !hasPermission(perms, model.PermReply) {
			return apperrors.New(403, "forbidden", "missing reply permission")
		}
		replyMessage, err := s.messages.GetByGroupAndMessageID(ctx, group.ID, *replyTo)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(400, "invalid_reply_target", "reply target does not exist")
			}
			return err
		}
		if replyMessage.Status == model.MessageStatusDeleted || !isVisibleByTTL(group.MessageTTLSeconds, replyMessage.CreatedAt, now) {
			return errRecordNotVisible("reply target")
		}
	}

	if forwardFrom != nil {
		if !hasPermission(perms, model.PermForward) {
			return apperrors.New(403, "forbidden", "missing forward permission")
		}
		source, err := s.messages.GetByMessageID(ctx, *forwardFrom)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(400, "invalid_forward_target", "forward source does not exist")
			}
			return err
		}
		if source.Status == model.MessageStatusDeleted || !isVisibleByTTL(source.Group.MessageTTLSeconds, source.CreatedAt, now) {
			return errRecordNotVisible("forward source")
		}
		if _, _, err := s.requireActiveMembership(ctx, userID, source.Group.GroupID.String()); err != nil {
			return errRecordNotVisible("forward source")
		}
	}
	return nil
}

func (s *MessageService) validatePayload(ctx context.Context, contentType string, payload any) ([]byte, error) {
	switch contentType {
	case model.MessageContentTypeText:
		var data TextPayload
		if err := decodeStrict(payload, &data); err != nil {
			return nil, payloadValidationError("invalid text payload")
		}
		if strings.TrimSpace(data.Text) == "" {
			return nil, payloadValidationError("text is required")
		}
		return encodeJSON(data)
	case model.MessageContentTypeImage:
		var data ImagePayload
		if err := decodeStrict(payload, &data); err != nil {
			return nil, payloadValidationError("invalid image payload")
		}
		if err := validateMediaCommon(data.CID, data.MIMEType, data.Size); err != nil {
			return nil, err
		}
		if data.Width <= 0 || data.Height <= 0 {
			return nil, payloadValidationError("width and height must be positive")
		}
		if err := s.validateMediaCIDs(ctx, data.CID, data.ThumbnailCID); err != nil {
			return nil, err
		}
		return encodeJSON(data)
	case model.MessageContentTypeVideo:
		var data VideoPayload
		if err := decodeStrict(payload, &data); err != nil {
			return nil, payloadValidationError("invalid video payload")
		}
		if err := validateMediaCommon(data.CID, data.MIMEType, data.Size); err != nil {
			return nil, err
		}
		if data.Width <= 0 || data.Height <= 0 || data.Duration <= 0 {
			return nil, payloadValidationError("width, height and duration must be positive")
		}
		if err := s.validateMediaCIDs(ctx, data.CID, data.ThumbnailCID); err != nil {
			return nil, err
		}
		return encodeJSON(data)
	case model.MessageContentTypeVoice:
		var data VoicePayload
		if err := decodeStrict(payload, &data); err != nil {
			return nil, payloadValidationError("invalid voice payload")
		}
		if err := validateMediaCommon(data.CID, data.MIMEType, data.Size); err != nil {
			return nil, err
		}
		if data.Duration <= 0 {
			return nil, payloadValidationError("duration must be positive")
		}
		if err := s.ipfs.RegisterMetadata(ctx, data.CID); err != nil {
			return nil, payloadValidationError("cid is invalid")
		}
		return encodeJSON(data)
	case model.MessageContentTypeFile:
		var data FilePayload
		if err := decodeStrict(payload, &data); err != nil {
			return nil, payloadValidationError("invalid file payload")
		}
		if err := validateMediaCommon(data.CID, data.MIMEType, data.Size); err != nil {
			return nil, err
		}
		if strings.TrimSpace(data.FileName) == "" {
			return nil, payloadValidationError("file_name is required")
		}
		if err := s.ipfs.RegisterMetadata(ctx, data.CID); err != nil {
			return nil, payloadValidationError("cid is invalid")
		}
		return encodeJSON(data)
	case model.MessageContentTypeForward:
		var data ForwardPayload
		if err := decodeStrict(payload, &data); err != nil {
			return nil, payloadValidationError("invalid forward payload")
		}
		return encodeJSON(data)
	default:
		return nil, apperrors.New(400, "invalid_content_type", "unsupported content_type")
	}
}

func (s *MessageService) validateMediaCIDs(ctx context.Context, mainCID, thumbnailCID string) error {
	if err := s.ipfs.RegisterMetadata(ctx, mainCID); err != nil {
		return payloadValidationError("cid is invalid")
	}
	if thumbnailCID != "" {
		if err := s.ipfs.RegisterMetadata(ctx, thumbnailCID); err != nil {
			return payloadValidationError("thumbnail_cid is invalid")
		}
	}
	return nil
}

func (s *MessageService) toMessageViewForUser(ctx context.Context, viewerID uint64, group model.Group, message model.GroupMessage) (MessageView, error) {
	return s.toMessageView(ctx, viewerID, group, message, false)
}

func (s *MessageService) toMessageViewForAdmin(ctx context.Context, group model.Group, message model.GroupMessage) (MessageView, error) {
	return s.toMessageView(ctx, 0, group, message, true)
}

func (s *MessageService) toMessageView(ctx context.Context, viewerID uint64, group model.Group, message model.GroupMessage, adminView bool) (MessageView, error) {
	payload, err := decodeJSONPayload(message.PayloadJSON)
	if err != nil {
		return MessageView{}, err
	}

	view := MessageView{
		GroupID:              group.GroupID.String(),
		MessageID:            message.MessageID,
		Seq:                  message.Seq,
		ContentType:          message.ContentType,
		ReplyToMessageID:     message.ReplyToMessageID,
		ForwardFromMessageID: message.ForwardFromMessageID,
		Sender:               toPublicUser(message.Sender),
		Status:               message.Status,
		EditCount:            message.EditCount,
		LastEditedAt:         message.LastEditedAt,
		DeleteReason:         message.DeleteReason,
		DeletedAt:            message.DeletedAt,
		CreatedAt:            message.CreatedAt,
	}

	if message.Status != model.MessageStatusDeleted {
		view.Payload = payload
	}

	if message.ContentType == model.MessageContentTypeForward && message.ForwardFromMessageID != nil {
		ref, err := s.resolveForwardReference(ctx, viewerID, *message.ForwardFromMessageID, adminView)
		if err != nil {
			return MessageView{}, err
		}
		view.Forward = ref
	}

	return view, nil
}

func (s *MessageService) resolveForwardReference(ctx context.Context, viewerID uint64, forwardMessageID string, adminView bool) (*ForwardReferenceView, error) {
	source, err := s.messages.GetByMessageID(ctx, forwardMessageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &ForwardReferenceView{State: "missing", Notice: "原消息已删除或已过期"}, nil
		}
		return nil, err
	}

	if !adminView {
		if _, _, err := s.requireActiveMembership(ctx, viewerID, source.Group.GroupID.String()); err != nil {
			return &ForwardReferenceView{State: "unavailable", Notice: "原消息不可见"}, nil
		}
	}
	if !adminView && (source.Status == model.MessageStatusDeleted || !isVisibleByTTL(source.Group.MessageTTLSeconds, source.CreatedAt, time.Now().UTC())) {
		return &ForwardReferenceView{State: "deleted_or_expired", Notice: sanitizeDeletedNotice(source)}, nil
	}

	payload, err := decodeJSONPayload(source.PayloadJSON)
	if err != nil {
		return nil, err
	}
	return &ForwardReferenceView{
		State: "ok",
		Message: &MessageSummary{
			GroupID:     source.Group.GroupID.String(),
			MessageID:   source.MessageID,
			Seq:         source.Seq,
			ContentType: source.ContentType,
			Payload:     payload,
			Sender:      toPublicUser(source.Sender),
			Status:      source.Status,
			CreatedAt:   source.CreatedAt,
		},
	}, nil
}

func (s *MessageService) deleteMessage(ctx context.Context, userID uint64, groupID, messageID string, selfRetract bool) (*MessageView, error) {
	group, operator, err := s.requireActiveMembership(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	operatorPerms := model.EffectivePermissions(operator.Role, group.DefaultPermissions, operator.PermissionsAllow, operator.PermissionsDeny)

	var updated model.GroupMessage
	if err := s.messages.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		message, err := s.messages.GetByMessageIDForUpdate(ctx, tx, messageID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.New(404, "message_not_found", "message not found")
			}
			return err
		}
		if message.GroupID != group.ID {
			return apperrors.New(404, "message_not_found", "message not found")
		}
		if message.Status == model.MessageStatusDeleted {
			updated = *message
			return nil
		}

		now := time.Now().UTC()
		if selfRetract {
			if message.SenderUserID != userID {
				return apperrors.New(403, "forbidden", "cannot retract other user's message")
			}
			if group.MessageRetractSeconds > 0 && message.CreatedAt.Before(now.Add(-time.Duration(group.MessageRetractSeconds)*time.Second)) {
				return apperrors.New(400, "retract_window_expired", "message retract window has expired")
			}
			reason := model.DeleteReasonSelfRetracted
			message.Status = model.MessageStatusDeleted
			message.DeletedAt = &now
			message.DeletedByUserID = &userID
			message.DeleteReason = &reason
		} else {
			if !hasPermission(operatorPerms, model.PermDeleteMessages) {
				return apperrors.New(403, "forbidden", "missing delete permission")
			}
			senderMember, err := s.groups.GetMember(ctx, group.ID, message.SenderUserID)
			if err != nil {
				return err
			}
			if senderMember.Role == model.RoleOwner {
				return apperrors.New(403, "forbidden", "cannot delete owner messages")
			}
			if model.RoleRank(operator.Role) <= model.RoleRank(senderMember.Role) && operator.Role != model.RoleOwner {
				return apperrors.New(403, "forbidden", "cannot delete equal or higher role message")
			}
			reason := model.DeleteReasonAdminRemoved
			message.Status = model.MessageStatusDeleted
			message.DeletedAt = &now
			message.DeletedByUserID = &userID
			message.DeleteReason = &reason
		}

		if err := s.messages.Save(ctx, tx, message); err != nil {
			return err
		}
		updated = *message
		return nil
	}); err != nil {
		return nil, err
	}

	view, err := s.toMessageViewForUser(ctx, userID, *group, updated)
	if err != nil {
		return nil, err
	}

	_ = s.publisher.Publish(ctx, events.Envelope{
		Type:      events.EventGroupMessageDeleted,
		GroupID:   groupID,
		MessageID: messageID,
		At:        time.Now().UTC(),
	})
	return &view, nil
}
