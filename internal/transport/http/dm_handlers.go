package http

import (
	"net/http"
	"strconv"

	"meshchat-server/internal/service"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getDMConversations(w http.ResponseWriter, r *http.Request) {
	if h.dm == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "dm service unavailable"}})
		return
	}
	list, err := h.dm.ListConversations(r.Context(), currentUserID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) postDMConversations(w http.ResponseWriter, r *http.Request) {
	if h.dm == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "dm service unavailable"}})
		return
	}
	var in service.CreateDMConversationInput
	if !decodeBody(w, r, &in) {
		return
	}
	out, err := h.dm.CreateConversation(r.Context(), currentUserID(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getDMMessages(w http.ResponseWriter, r *http.Request) {
	if h.dm == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "dm service unavailable"}})
		return
	}
	convID := chi.URLParam(r, "conversation_id")
	var beforeSeq, afterSeq uint64
	if v := r.URL.Query().Get("before_seq"); v != "" {
		beforeSeq, _ = strconv.ParseUint(v, 10, 64)
	}
	if v := r.URL.Query().Get("after_seq"); v != "" {
		afterSeq, _ = strconv.ParseUint(v, 10, 64)
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := h.dm.ListMessages(r.Context(), currentUserID(r), convID, beforeSeq, afterSeq, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) postDMMessages(w http.ResponseWriter, r *http.Request) {
	if h.dm == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "dm service unavailable"}})
		return
	}
	var in service.SendDMMessageInput
	if !decodeBody(w, r, &in) {
		return
	}
	convID := chi.URLParam(r, "conversation_id")
	out, err := h.dm.SendMessage(r.Context(), currentUserID(r), convID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) postDMAck(w http.ResponseWriter, r *http.Request) {
	if h.dm == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": map[string]any{"code": "not_implemented", "message": "dm service unavailable"}})
		return
	}
	msgID := chi.URLParam(r, "message_id")
	out, err := h.dm.AckMessage(r.Context(), currentUserID(r), msgID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
