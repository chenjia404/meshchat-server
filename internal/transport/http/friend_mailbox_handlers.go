package http

import (
	"net/http"
	"strings"

	"meshchat-server/internal/service"
	"meshchat-server/pkg/apperrors"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getFriendMailboxRequests(w http.ResponseWriter, r *http.Request) {
	if h.friendMailbox == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "friend mailbox unavailable"}})
		return
	}
	list, err := h.friendMailbox.List(r.Context(), currentUserID(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) postFriendMailboxRequest(w http.ResponseWriter, r *http.Request) {
	if h.friendMailbox == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "friend mailbox unavailable"}})
		return
	}
	var body struct {
		ToPeerID  string `json:"to_peer_id"`
		IntroText string `json:"intro_text"`
		Nickname  string `json:"nickname"`
		Bio       string `json:"bio"`
		AvatarCID string `json:"avatar_cid"`
	}
	if !h.decodeBody(w, r, &body) {
		return
	}
	out, err := h.friendMailbox.Create(r.Context(), currentUserID(r), service.CreateFriendMailboxInput{
		ToPeerID:  body.ToPeerID,
		IntroText: body.IntroText,
		Nickname:  body.Nickname,
		Bio:       body.Bio,
		AvatarCID: body.AvatarCID,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) postFriendMailboxAccept(w http.ResponseWriter, r *http.Request) {
	if h.friendMailbox == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "friend mailbox unavailable"}})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "request_id"))
	if requestID == "" {
		h.writeError(w, apperrors.New(400, "invalid_request_id", "request_id is required"))
		return
	}
	out, err := h.friendMailbox.Accept(r.Context(), currentUserID(r), requestID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) postFriendMailboxReject(w http.ResponseWriter, r *http.Request) {
	if h.friendMailbox == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "friend mailbox unavailable"}})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "request_id"))
	if requestID == "" {
		h.writeError(w, apperrors.New(400, "invalid_request_id", "request_id is required"))
		return
	}
	if err := h.friendMailbox.Reject(r.Context(), currentUserID(r), requestID); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "rejected"})
}
