package http

import (
	stdhttp "net/http"
	"strconv"

	"meshchat-server/internal/auth"
	appmiddleware "meshchat-server/internal/middleware"
	"meshchat-server/internal/service"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	auth    *service.AuthService
	profile *service.ProfileService
	groups  *service.GroupService
	message *service.MessageService
	files   *service.FileService
	ws      stdhttp.Handler
	mode    string
}

func NewHandler(authService *service.AuthService, profile *service.ProfileService, groupService *service.GroupService, messageService *service.MessageService, fileService *service.FileService, wsHandler stdhttp.Handler, serverMode string) *Handler {
	return &Handler{
		auth:    authService,
		profile: profile,
		groups:  groupService,
		message: messageService,
		files:   fileService,
		ws:      wsHandler,
		mode:    serverMode,
	}
}

func NewRouter(handler *Handler, jwtManager *auth.JWTManager, recoverer func(stdhttp.Handler) stdhttp.Handler, requestLogger func(stdhttp.Handler) stdhttp.Handler) chi.Router {
	router := chi.NewRouter()
	router.Use(appmiddleware.AllowAnyOriginCORS())
	router.Use(chimiddleware.RequestID)
	router.Use(recoverer)
	router.Use(requestLogger)

	router.Get("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		writeJSON(w, stdhttp.StatusOK, map[string]any{"status": "ok"})
	})
	router.Get("/server/info", handler.getServerInfo)

	router.Post("/auth/challenge", handler.postChallenge)
	router.Post("/auth/login", handler.postLogin)
	router.Get("/ws", handler.ws.ServeHTTP)

	router.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(jwtManager))
		r.Get("/me/groups", handler.getMyGroups)
		r.Get("/me/profile", handler.getMyProfile)
		r.Patch("/users/{peer_id}/profile", handler.patchUserProfileByPeerID)

		r.Post("/groups", handler.postGroups)
		r.Post("/groups/{group_id}/join", handler.postGroupJoin)
		r.Post("/groups/{group_id}/leave", handler.postGroupLeave)
		r.Get("/groups/{group_id}", handler.getGroup)
		r.Patch("/groups/{group_id}", handler.patchGroup)
		r.Post("/groups/{group_id}/transfer-owner", handler.postGroupTransferOwner)
		r.Post("/groups/{group_id}/dissolve", handler.postGroupDissolve)
		r.Post("/groups/{group_id}/members/invite", handler.postMembersInvite)
		r.Post("/groups/{group_id}/members/{user_id}/invite", handler.postMemberInvite)
		r.Patch("/groups/{group_id}/message-policy", handler.patchMessagePolicy)
		r.Get("/groups/{group_id}/members", handler.getGroupMembers)
		r.Post("/groups/{group_id}/members/{user_id}/admin", handler.postMemberAdmin)
		r.Patch("/groups/{group_id}/members/{user_id}/permissions", handler.patchMemberPermissions)
		r.Post("/groups/{group_id}/members/{user_id}/mute", handler.postMemberMute)
		r.Post("/groups/{group_id}/members/{user_id}/ban", handler.postMemberBan)
		r.Get("/groups/{group_id}/messages", handler.getMessages)
		r.Post("/groups/{group_id}/messages", handler.postMessages)
		r.Patch("/groups/{group_id}/messages/{message_id}", handler.patchMessage)
		r.Post("/groups/{group_id}/messages/{message_id}/retract", handler.postMessageRetract)
		r.Post("/groups/{group_id}/messages/{message_id}/delete", handler.postMessageDelete)
		r.Post("/files", handler.postFiles)
	})

	return router
}

func currentUserID(r *stdhttp.Request) uint64 {
	userID, _ := appmiddleware.UserIDFromContext(r.Context())
	return userID
}

func parseUintParam(r *stdhttp.Request, key string) (uint64, error) {
	return strconv.ParseUint(chi.URLParam(r, key), 10, 64)
}
