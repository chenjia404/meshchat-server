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
	auth                       *service.AuthService
	profile                    *service.ProfileService
	groups                     *service.GroupService
	message                    *service.MessageService
	files                      *service.FileService
	dm                         *service.DMService
	friendMailbox              *service.FriendMailboxService
	ws                         stdhttp.Handler
	mode                       string
	ipfsGatewayPrefix          string
	ipfsGatewayBaseURL         string
	exposeInternalErrorDetail  bool
}

func NewHandler(authService *service.AuthService, profile *service.ProfileService, groupService *service.GroupService, messageService *service.MessageService, fileService *service.FileService, dmService *service.DMService, friendMailbox *service.FriendMailboxService, wsHandler stdhttp.Handler, serverMode string, ipfsGatewayPrefix, ipfsGatewayBaseURL string, exposeInternalErrorDetail bool) *Handler {
	return &Handler{
		auth:                      authService,
		profile:                   profile,
		groups:                    groupService,
		message:                   messageService,
		files:                     fileService,
		dm:                        dmService,
		friendMailbox:             friendMailbox,
		ws:                        wsHandler,
		mode:                      serverMode,
		ipfsGatewayPrefix:         ipfsGatewayPrefix,
		ipfsGatewayBaseURL:        ipfsGatewayBaseURL,
		exposeInternalErrorDetail: exposeInternalErrorDetail,
	}
}

func NewRouter(handler *Handler, jwtManager *auth.JWTManager, recoverer func(stdhttp.Handler) stdhttp.Handler, requestLogger func(stdhttp.Handler) stdhttp.Handler, ipfsGatewayProxy stdhttp.Handler, legacyAPIRoot bool) chi.Router {
	router := chi.NewRouter()
	router.Use(appmiddleware.AllowAnyOriginCORS())
	router.Use(chimiddleware.RequestID)
	router.Use(recoverer)
	router.Use(requestLogger)

	if ipfsGatewayProxy != nil {
		router.Handle("/ipfs/", ipfsGatewayProxy)
		router.Handle("/ipfs/*", ipfsGatewayProxy)
		router.Get("/ipfs", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			stdhttp.Redirect(w, r, "/ipfs/", stdhttp.StatusTemporaryRedirect)
		})
	}

	healthz := func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		writeJSON(w, stdhttp.StatusOK, map[string]any{"status": "ok"})
	}
	router.Get("/healthz", healthz)

	router.Route("/api", func(r chi.Router) {
		r.Get("/healthz", healthz)
		registerHTTPAPI(r, handler, jwtManager)
	})

	// 兼容未使用 /api 前缀的旧客户端（与 /api 下路由等价）。弃用时将 LEGACY_API_ROOT=false 或删除本段与 registerHTTPAPI 的 legacy 调用。
	if legacyAPIRoot {
		registerHTTPAPI(router, handler, jwtManager)
	}

	return router
}

// registerHTTPAPI 注册 MeshChat HTTP API（不含 /healthz）。挂载在 /api 子路由上时为 /api/...，挂在根路由上时为旧版根路径。
func registerHTTPAPI(r chi.Router, handler *Handler, jwtManager *auth.JWTManager) {
	r.Get("/server/info", handler.getServerInfo)

	r.Post("/auth/challenge", handler.postChallenge)
	r.Post("/auth/login", handler.postLogin)
	r.Get("/ws", handler.ws.ServeHTTP)

	r.Group(func(r chi.Router) {
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

		r.Get("/dm/conversations", handler.getDMConversations)
		r.Post("/dm/conversations", handler.postDMConversations)
		r.Get("/dm/conversations/{conversation_id}/messages", handler.getDMMessages)
		r.Post("/dm/conversations/{conversation_id}/messages", handler.postDMMessages)
		r.Post("/dm/messages/{message_id}/ack", handler.postDMAck)

		r.Get("/friend-mailbox/requests", handler.getFriendMailboxRequests)
		r.Post("/friend-mailbox/requests", handler.postFriendMailboxRequest)
		r.Post("/friend-mailbox/requests/{request_id}/accept", handler.postFriendMailboxAccept)
		r.Post("/friend-mailbox/requests/{request_id}/reject", handler.postFriendMailboxReject)
	})
}

func currentUserID(r *stdhttp.Request) uint64 {
	userID, _ := appmiddleware.UserIDFromContext(r.Context())
	return userID
}

func parseUintParam(r *stdhttp.Request, key string) (uint64, error) {
	return strconv.ParseUint(chi.URLParam(r, key), 10, 64)
}
