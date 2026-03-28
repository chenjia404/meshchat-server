package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"meshchat-server/internal/service"
	"meshchat-server/pkg/apperrors"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) postChallenge(w http.ResponseWriter, r *http.Request) {
	var request struct {
		PeerID string `json:"peer_id"`
	}
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.auth.RequestChallenge(r.Context(), request.PeerID)
	if err != nil {
		writeError(w, err)
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
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.auth.Login(r.Context(), request.PeerID, request.ChallengeID, request.Signature, request.PublicKey)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getServerInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, service.ServerInfoView{
		ServerMode: h.mode,
	})
}

func (h *Handler) getMyProfile(w http.ResponseWriter, r *http.Request) {
	response, err := h.profile.GetProfile(r.Context(), currentUserID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getMyGroups(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	response, err := h.groups.ListMyGroups(r.Context(), currentUserID(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchUserProfileByPeerID(w http.ResponseWriter, r *http.Request) {
	var request service.UpdateProfileInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.profile.UpdateProfileByPeerID(r.Context(), currentUserID(r), chi.URLParam(r, "peer_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroups(w http.ResponseWriter, r *http.Request) {
	var request service.CreateGroupInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.CreateGroup(r.Context(), currentUserID(r), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.GetGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupJoin(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.JoinGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupLeave(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.LeaveGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchGroup(w http.ResponseWriter, r *http.Request) {
	var request service.UpdateGroupInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.UpdateGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupTransferOwner(w http.ResponseWriter, r *http.Request) {
	var request service.TransferGroupOwnershipInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.TransferOwnership(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupDissolve(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.DissolveGroup(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberInvite(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	response, err := h.groups.InviteMember(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMembersInvite(w http.ResponseWriter, r *http.Request) {
	var request service.InviteGroupMembersInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.InviteMembersByPeerIDs(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchMessagePolicy(w http.ResponseWriter, r *http.Request) {
	var request service.UpdateMessagePolicyInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.UpdateMessagePolicy(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getGroupMembers(w http.ResponseWriter, r *http.Request) {
	response, err := h.groups.ListMembers(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberAdmin(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	var request service.SetGroupAdminInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.SetGroupAdmin(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) patchMemberPermissions(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	var request service.UpdateMemberPermissionsInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.UpdateMemberPermissions(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberMute(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	var request service.MuteMemberInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.groups.MuteMember(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMemberBan(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := parseUintParam(r, "user_id")
	if err != nil {
		writeError(w, apperrors.New(http.StatusBadRequest, "invalid_user_id", "user_id must be numeric"))
		return
	}
	response, err := h.groups.BanMember(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), targetUserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	beforeSeq, _ := strconv.ParseUint(r.URL.Query().Get("before_seq"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	response, err := h.message.ListMessages(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), beforeSeq, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMessages(w http.ResponseWriter, r *http.Request) {
	var request service.SendMessageInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.message.SendMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) patchMessage(w http.ResponseWriter, r *http.Request) {
	var request service.EditMessageInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.message.EditMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), chi.URLParam(r, "message_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMessageRetract(w http.ResponseWriter, r *http.Request) {
	response, err := h.message.RetractMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), chi.URLParam(r, "message_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postMessageDelete(w http.ResponseWriter, r *http.Request) {
	response, err := h.message.DeleteMessage(r.Context(), currentUserID(r), chi.URLParam(r, "group_id"), chi.URLParam(r, "message_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postFiles(w http.ResponseWriter, r *http.Request) {
	var request service.RegisterFileInput
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.files.Register(r.Context(), currentUserID(r), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func decodeBody(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeError(w, apperrors.New(http.StatusBadRequest, "invalid_json", "request body is invalid"))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, apperrors.HTTPStatus(err), map[string]any{
		"error": apperrors.Public(err),
	})
}
