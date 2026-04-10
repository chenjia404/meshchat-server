package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"meshchat-server/internal/auth"
	"meshchat-server/internal/config"
	"meshchat-server/internal/db"
	"meshchat-server/internal/db/migrations"
	"meshchat-server/internal/events"
	"meshchat-server/internal/ipfs"
	appmiddleware "meshchat-server/internal/middleware"
	"meshchat-server/internal/repo"
	"meshchat-server/internal/service"
	admintransport "meshchat-server/internal/transport/adminhttp"
	httptransport "meshchat-server/internal/transport/http"
	wstransport "meshchat-server/internal/transport/ws"

	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg             config.Config
	logger          *slog.Logger
	httpServer      *http.Server
	adminHTTPServer *http.Server
	redisBus        *events.RedisBus
	hub             *wstransport.Hub
	messages        *service.MessageService
	groups          *service.GroupService
	dm              *service.DMService
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	postgres, err := db.OpenPostgres(cfg.DatabaseURL, logger)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if cfg.AutoMigrate {
		if err := migrations.Run(context.Background(), postgres); err != nil {
			return nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	ipfsClient, err := ipfs.NewLocalClient(cfg.IPFSAPIURL)
	if err != nil {
		return nil, fmt.Errorf("init ipfs client: %w", err)
	}

	userRepo := repo.NewUserRepo(postgres)
	groupRepo := repo.NewGroupRepo(postgres)
	messageRepo := repo.NewMessageRepo(postgres)
	fileRepo := repo.NewFileRepo(postgres)
	dmRepo := repo.NewDMRepo(postgres)

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpiration)
	adminJWTManager := auth.NewAdminJWTManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpiration)
	redisBus := events.NewRedisBus(redisClient, logger)
	serverAdmins := service.NewServerAdminService(userRepo, cfg.ServerAdminPeerIDs)

	authService := service.NewAuthService(userRepo, redisClient, jwtManager, auth.NewLibP2PVerifier(), cfg.ChallengeTTL)
	profileService := service.NewProfileService(userRepo, ipfsClient)
	groupService := service.NewGroupService(groupRepo, userRepo, redisBus, ipfsClient, serverAdmins, cfg.ServerMode)
	messageService := service.NewMessageService(groupRepo, messageRepo, redisClient, ipfsClient, redisBus)
	dmService := service.NewDMService(dmRepo, userRepo, redisBus)
	fileService := service.NewFileService(fileRepo, ipfsClient)
	adminService := service.NewAdminService(userRepo, groupRepo, messageService, ipfsClient, redisBus, adminJWTManager, cfg.AdminUsername, cfg.AdminPassword)

	hub := wstransport.NewHub()
	wsHandler := wstransport.NewHandler(hub, jwtManager, redisClient, groupService, dmService, logger, cfg.WSSendBuffer, cfg.WSWriteWait, cfg.WSPongWait, cfg.WSPingInterval, cfg.OnlineTTL)
	ipfsGatewayPrefix := ""
	if cfg.IPFSGatewayUpstreamURL != "" {
		ipfsGatewayPrefix = "/ipfs"
	}
	httpHandler := httptransport.NewHandler(authService, profileService, groupService, messageService, fileService, dmService, wsHandler, cfg.ServerMode, ipfsGatewayPrefix, cfg.IPFSGatewayBaseURL, cfg.ExposeInternalErrorDetail)

	var ipfsGatewayProxy http.Handler
	if cfg.IPFSGatewayUpstreamURL != "" {
		upstream, err := url.Parse(cfg.IPFSGatewayUpstreamURL)
		if err != nil {
			return nil, fmt.Errorf("parse IPFS_GATEWAY_UPSTREAM: %w", err)
		}
		if upstream.Scheme == "" || upstream.Host == "" {
			return nil, fmt.Errorf("IPFS_GATEWAY_UPSTREAM must be an absolute URL with scheme and host")
		}
		ipfsGatewayProxy = httptransport.NewIPFSGatewayProxy(upstream)
	}

	router := httptransport.NewRouter(httpHandler, jwtManager, appmiddleware.Recoverer(logger), appmiddleware.RequestLogger(logger), ipfsGatewayProxy, cfg.LegacyAPIRoot)
	adminHandler := admintransport.NewHandler(adminService)
	adminRouter := admintransport.NewRouter(adminHandler, adminJWTManager, appmiddleware.Recoverer(logger), appmiddleware.RequestLogger(logger))

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	adminHTTPServer := &http.Server{
		Addr:              cfg.AdminHTTPAddr,
		Handler:           adminRouter,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{
		cfg:             cfg,
		logger:          logger,
		httpServer:      httpServer,
		adminHTTPServer: adminHTTPServer,
		redisBus:        redisBus,
		hub:             hub,
		messages:        messageService,
		groups:          groupService,
		dm:              dmService,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	consumeCtx, cancelConsume := context.WithCancel(context.Background())
	defer cancelConsume()

	go func() {
		if err := a.redisBus.Consume(consumeCtx, a.handleEvent); err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Error("redis event consumer stopped", slog.Any("error", err))
		}
	}()

	serverErr := make(chan error, 2)
	go func() {
		a.logger.Info("starting http server", slog.String("addr", a.cfg.HTTPAddr))
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()
	go func() {
		a.logger.Info("starting admin http server", slog.String("addr", a.cfg.AdminHTTPAddr))
		if err := a.adminHTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	shutdownServers := func() error {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		adminErr := a.adminHTTPServer.Shutdown(shutdownCtx)
		httpErr := a.httpServer.Shutdown(shutdownCtx)
		if adminErr != nil && !errors.Is(adminErr, http.ErrServerClosed) {
			return adminErr
		}
		if httpErr != nil && !errors.Is(httpErr, http.ErrServerClosed) {
			return httpErr
		}
		return nil
	}

	select {
	case <-ctx.Done():
		cancelConsume()
		return shutdownServers()
	case err := <-serverErr:
		cancelConsume()
		if shutdownErr := shutdownServers(); shutdownErr != nil && err == nil {
			return shutdownErr
		}
		return err
	}
}

func (a *App) handleEvent(ctx context.Context, event events.Envelope) error {
	switch event.Type {
	case events.EventGroupMessageCreated, events.EventGroupMessageEdited, events.EventGroupMessageDeleted:
		a.hub.BroadcastGroup(event.GroupID, func(userID uint64) ([]byte, error) {
			envelope, err := a.messages.BuildMessageEventForUser(ctx, userID, event.Type, event.MessageID)
			if err != nil || envelope == nil {
				return nil, err
			}
			return wstransport.MarshalEnvelope(envelope)
		})
	case events.EventGroupSettingsUpdated:
		a.hub.BroadcastGroup(event.GroupID, func(userID uint64) ([]byte, error) {
			envelope, err := a.groups.BuildSettingsEventForUser(ctx, userID, event.GroupID)
			if err != nil || envelope == nil {
				return nil, err
			}
			return wstransport.MarshalEnvelope(envelope)
		})
	case events.EventGroupMemberUpdated:
		a.hub.BroadcastGroup(event.GroupID, func(userID uint64) ([]byte, error) {
			envelope, err := a.groups.BuildMemberEventForUser(ctx, userID, event.GroupID, event.UserID)
			if err != nil || envelope == nil {
				return nil, err
			}
			return wstransport.MarshalEnvelope(envelope)
		})
	case events.EventDMMessageCreated, events.EventDMMessageAcked:
		if a.dm == nil || event.ConversationID == "" {
			break
		}
		a.hub.BroadcastDM(event.ConversationID, func(userID uint64) ([]byte, error) {
			envelope, err := a.dm.BuildRealtimeDMEvent(ctx, userID, event)
			if err != nil || envelope == nil {
				return nil, err
			}
			return wstransport.MarshalEnvelope(envelope)
		})
	}
	return nil
}
