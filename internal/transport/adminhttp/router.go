package adminhttp

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
	admin *service.AdminService
}

func NewHandler(adminService *service.AdminService) *Handler {
	return &Handler{admin: adminService}
}

func NewRouter(handler *Handler, jwtManager *auth.AdminJWTManager, recoverer func(stdhttp.Handler) stdhttp.Handler, requestLogger func(stdhttp.Handler) stdhttp.Handler) chi.Router {
	router := chi.NewRouter()
	router.Use(appmiddleware.AllowAnyOriginCORS())
	router.Use(chimiddleware.RequestID)
	router.Use(recoverer)
	router.Use(requestLogger)

	router.Get("/", handler.getIndex)
	router.Get("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		writeJSON(w, stdhttp.StatusOK, map[string]any{"status": "ok"})
	})

	router.Post("/admin/login", handler.postLogin)

	router.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAdminAuth(jwtManager))
		r.Get("/admin/me", handler.getMe)
		r.Get("/admin/users", handler.getUsers)
		r.Get("/admin/groups", handler.getGroups)
		r.Post("/admin/groups", handler.postGroups)
		r.Get("/admin/groups/{group_id}", handler.getGroup)
		r.Patch("/admin/groups/{group_id}", handler.patchGroup)
		r.Post("/admin/groups/{group_id}/dissolve", handler.postGroupDissolve)
		r.Get("/admin/groups/{group_id}/members", handler.getGroupMembers)
		r.Get("/admin/groups/{group_id}/messages", handler.getMessages)
	})

	return router
}

func parseLimitOffset(r *stdhttp.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return limit, offset
}
