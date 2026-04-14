package http

import (
	"net/http"
	"strconv"
	"strings"

	"meshchat-server/internal/service"
	"meshchat-server/pkg/apperrors"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getPublicChannels(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	items, err := h.publicChannels.ListChannelsByOwner(r.Context(), currentUserID(r), strings.TrimSpace(r.URL.Query().Get("owner_peer_id")))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) getPublicChannelSubscriptions(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	items, err := h.publicChannels.ListSubscribedChannels(r.Context(), currentUserID(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) postPublicChannels(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	var in service.CreatePublicChannelInput
	if !h.decodeBody(w, r, &in) {
		return
	}
	out, err := h.publicChannels.CreateChannel(r.Context(), currentUserID(r), in)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) getPublicChannel(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	out, err := h.publicChannels.GetChannelSummary(r.Context(), currentUserID(r), chi.URLParam(r, "channel_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) patchPublicChannel(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	var in service.UpdatePublicChannelProfileInput
	if !h.decodeBody(w, r, &in) {
		return
	}
	out, err := h.publicChannels.UpdateChannel(r.Context(), currentUserID(r), chi.URLParam(r, "channel_id"), in)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) postPublicChannelSubscribe(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	var in service.SubscribePublicChannelInput
	if !h.decodeBody(w, r, &in) {
		return
	}
	out, err := h.publicChannels.SubscribeChannel(r.Context(), currentUserID(r), chi.URLParam(r, "channel_id"), in)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getPublicChannelHead(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	out, err := h.publicChannels.GetChannelHead(r.Context(), chi.URLParam(r, "channel_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getPublicChannelChanges(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	afterSeq, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("after_seq")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	out, err := h.publicChannels.ListChannelChanges(r.Context(), chi.URLParam(r, "channel_id"), afterSeq, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getPublicChannelMessages(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	beforeMessageID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("before_message_id")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	out, err := h.publicChannels.ListChannelMessages(r.Context(), chi.URLParam(r, "channel_id"), beforeMessageID, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel_id": chi.URLParam(r, "channel_id"), "items": out})
}

func (h *Handler) getPublicChannelMessage(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	messageID, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "message_id")), 10, 64)
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_message_id", "invalid message_id"))
		return
	}
	out, err := h.publicChannels.GetChannelMessage(r.Context(), chi.URLParam(r, "channel_id"), messageID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) postPublicChannelMessage(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	var in service.UpsertPublicChannelMessageInput
	if !h.decodeBody(w, r, &in) {
		return
	}
	out, err := h.publicChannels.CreateMessage(r.Context(), currentUserID(r), chi.URLParam(r, "channel_id"), in)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) patchPublicChannelMessage(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	messageID, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "message_id")), 10, 64)
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_message_id", "invalid message_id"))
		return
	}
	var in service.UpsertPublicChannelMessageInput
	if !h.decodeBody(w, r, &in) {
		return
	}
	out, err := h.publicChannels.UpdateMessage(r.Context(), currentUserID(r), chi.URLParam(r, "channel_id"), messageID, in)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) deletePublicChannelMessage(w http.ResponseWriter, r *http.Request) {
	if h.publicChannels == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "public channel service unavailable"}})
		return
	}
	messageID, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "message_id")), 10, 64)
	if err != nil {
		h.writeError(w, apperrors.New(http.StatusBadRequest, "invalid_message_id", "invalid message_id"))
		return
	}
	out, err := h.publicChannels.DeleteMessage(r.Context(), currentUserID(r), chi.URLParam(r, "channel_id"), messageID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
