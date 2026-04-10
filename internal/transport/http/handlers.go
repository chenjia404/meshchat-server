package http

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"meshchat-server/internal/model"
	"meshchat-server/internal/service"
	"meshchat-server/pkg/apperrors"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) postChallenge(w http.ResponseWriter, r *http.Request) {
	var request struct {
		PeerID string `json:"peer_id"`
	}
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.auth.RequestChallenge(r.Context(), request.PeerID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postLogin(w http.ResponseWriter, r *http.Request) {
	var request struct {
		PeerID      string `json:"peer_id"`
		ChallengeID string `json:"challenge_id"`
		Signature   string `json:"signature"`
		PublicKey   string `json:"public_key"`
	}
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.auth.Login(r.Context(), request.PeerID, request.ChallengeID, request.Signature, request.PublicKey)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getServerInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, service.ServerInfoView{
		ServerMode:         h.mode,
		IPFSGatewayPrefix:  h.ipfsGatewayPrefix,
		IPFSGatewayBaseURL: h.ipfsGatewayBaseURL,
	})
}

func (h *Handler) getMyProfile(w http.ResponseWriter, r *http.Request) {
	response, err := h.profile.GetProfile(r.Context(), currentUserID(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getMyGroups(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	response, err := h.groups.ListMyGroups(r.Context(), currentUserID(r), limit, offset)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchUserProfileByPeerID(w http.ResponseWriter, r *http.Request) {
	var request service.UpdateProfileInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.profile.UpdateProfileByPeerID(r.Context(), currentUserID(r), chi.URLParam(r, "peer_id"), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroups(w http.ResponseWriter, r *http.Request) {
	var request service.CreateGroupInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.CreateGroup(r.Context(), currentUserID(r), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.GetGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupJoin(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.JoinGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupLeave(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.LeaveGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchGroup(w http.ResponseWriter, r *http.Request) {
	var request service.UpdateGroupInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.UpdateGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupTransferOwner(w http.ResponseWriter, r *http.Request) {
	var request service.TransferGroupOwnershipInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.TransferOwnership(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupDissolve(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.DissolveGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberInvite(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	response, err := h.groups.InviteMember(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMembersInvite(w http.ResponseWriter, r *http.Request) {
	var request service.InviteGroupMembersInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.InviteMembersByPeerIDs(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchMessagePolicy(w http.ResponseWriter, r *http.Request) {
	var request service.UpdateMessagePolicyInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.UpdateMessagePolicy(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getGroupMembers(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.ListMembers(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberAdmin(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	var request service.SetGroupAdminInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.SetGroupAdmin(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID, request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchMemberPermissions(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	var request service.UpdateMemberPermissionsInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.UpdateMemberPermissions(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID, request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberMute(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	var request service.MuteMemberInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.MuteMember(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID, request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberBan(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	response, err := h.groups.BanMember(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	beforeSeq, _ := strconv.ParseUint(r.URL.Query().Get("before_seq"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	response, err := h.message.ListMessages(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), beforeSeq, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMessages(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.postMessagesMultipart(w, r)
		return
	}

	var request service.SendMessageInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.message.SendMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) postMessagesMultipart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_multipart", "multipart form is invalid"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "file_required", "multipart field 'file' is required"))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_file", "failed to read uploaded file"))
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	} else if parsed, _, parseErr := mime.ParseMediaType(mimeType); parseErr == nil {
		mimeType = parsed
	}

	uploaded, err := h.files.UploadFile(r.Context(), currentUserID(r), service.UploadFileInput{
		FileName: firstNonEmpty(r.FormValue("file_name"), header.Filename),
		MIMEType: mimeType,
		Content:  content,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	messageType, err := resolveUploadedMessageType(strings.TrimSpace(r.FormValue("content_type")), uploaded.MIMEType)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var replyTo *string
	if value := strings.TrimSpace(r.FormValue("reply_to_message_id")); value != "" {
		replyTo = &value
	}
	var forwardFrom *string
	if value := strings.TrimSpace(r.FormValue("forward_from_message_id")); value != "" {
		forwardFrom = &value
	}

	var payload any
	switch messageType {
	case model.MessageContentTypeImage:
		if uploaded.Width == nil || uploaded.Height == nil || *uploaded.Width <= 0 || *uploaded.Height <= 0 {
			h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_payload", "uploaded file is not a valid image"))
			return
		}
		payload = service.ImagePayload{
			CID:          uploaded.CID,
			MIMEType:     uploaded.MIMEType,
			Size:         uploaded.Size,
			Width:        *uploaded.Width,
			Height:       *uploaded.Height,
			Caption:      strings.TrimSpace(r.FormValue("caption")),
			ThumbnailCID: strings.TrimSpace(r.FormValue("thumbnail_cid")),
		}
	case model.MessageContentTypeFile:
		payload = service.FilePayload{
			CID:      uploaded.CID,
			MIMEType: uploaded.MIMEType,
			Size:     uploaded.Size,
			FileName: uploaded.FileName,
			Caption:  strings.TrimSpace(r.FormValue("caption")),
		}
	default:
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_content_type", "multipart messages only support image or file"))
		return
	}

	response, err := h.message.SendMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), service.SendMessageInput{
		ContentType:          messageType,
		Payload:              payload,
		ReplyToMessageID:     replyTo,
		ForwardFromMessageID: forwardFrom,
		Signature:            strings.TrimSpace(r.FormValue("signature")),
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) patchMessage(w http.ResponseWriter, r *http.Request) {
	var request service.EditMessageInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.message.EditMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), chi.URLParam(r, "message_id"), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMessageRetract(w http.ResponseWriter, r *http.Request) {
	response, err := h.message.RetractMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), chi.URLParam(r, "message_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMessageDelete(w http.ResponseWriter, r *http.Request) {
	response, err := h.message.DeleteMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), chi.URLParam(r, "message_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postFiles(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.postFilesMultipart(w, r)
		return
	}

	var request service.RegisterFileInput
	if !h.decodeBody(w, r, &request) {
		return
	}
	response, err := h.files.Register(r.Context(), currentUserID(r), request)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) postFilesMultipart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_multipart", "multipart form is invalid"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "file_required", "multipart field 'file' is required"))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_file", "failed to read uploaded file"))
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	} else if parsed, _, parseErr := mime.ParseMediaType(mimeType); parseErr == nil {
		mimeType = parsed
	}

	response, err := h.files.UploadFile(r.Context(), currentUserID(r), service.UploadFileInput{
		FileName: header.Filename,
		MIMEType: mimeType,
		Content:  content,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) decodeBody(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_json", "request body is invalid"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	writeJSON(w, apperrors.HTTPStatus(err), map[string]any{
		"error": apperrors.PublicWithDetail(err, h.exposeInternalErrorDetail),
	})
}

func resolveUploadedMessageType(requestedType, mimeType string) (string, error) {
	switch requestedType {
	case "":
		if strings.HasPrefix(mimeType, "image/") {
			return model.MessageContentTypeImage, nil
		}
		return model.MessageContentTypeFile, nil
	case model.MessageContentTypeImage:
		if !strings.HasPrefix(mimeType, "image/") {
			return "", apperrors.New(http.StatusBadRequest, "invalid_payload", "content_type=image requires an image file")
		}
		return requestedType, nil
	case model.MessageContentTypeFile:
		return requestedType, nil
	default:
		return "", apperrors.New(http.StatusBadRequest, "invalid_content_type", "multipart messages only support image or file")
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
