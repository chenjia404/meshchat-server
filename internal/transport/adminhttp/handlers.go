package adminhttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"meshchat-server/internal/middleware"
	"meshchat-server/internal/service"
	"meshchat-server/pkg/apperrors"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(adminPageHTML))
}

func (h *Handler) postLogin(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeBody(w, r, &request) {
		return
	}
	response, err := h.admin.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	username, _ := middleware.AdminUsernameFromContext(r.Context())
	writeJSON(w, http.StatusOK, service.AdminMeView{Username: username})
}

func (h *Handler) getUsers(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r)
	response, err := h.admin.ListUsers(r.Context(), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getGroups(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r)
	response, err := h.admin.ListGroups(r.Context(), limit, offset)
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
	response, err := h.admin.CreateGroup(r.Context(), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request) {
	response, err := h.admin.GetGroup(r.Context(), chi.URLParam(r, "group_id"))
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
	response, err := h.admin.UpdateGroup(r.Context(), chi.URLParam(r, "group_id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) postGroupDissolve(w http.ResponseWriter, r *http.Request) {
	response, err := h.admin.DissolveGroup(r.Context(), chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getGroupMembers(w http.ResponseWriter, r *http.Request) {
	response, err := h.admin.ListMembers(r.Context(), chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	beforeSeq, _ := strconv.ParseUint(r.URL.Query().Get("before_seq"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	response, err := h.admin.ListMessages(r.Context(), chi.URLParam(r, "group_id"), beforeSeq, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
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
