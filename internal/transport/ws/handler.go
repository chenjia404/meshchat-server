package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"meshchat-server/internal/auth"
	"meshchat-server/internal/redisx"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type GroupAccessChecker interface {
	CanAccessGroup(ctx context.Context, userID uint64, groupID string) bool
}

// DMAccessChecker 校验用户是否可订阅某 DM 会话。
type DMAccessChecker interface {
	CanAccessConversation(ctx context.Context, userID uint64, conversationID string) bool
}

type Handler struct {
	hub          *Hub
	jwt          *auth.JWTManager
	redis        *redis.Client
	log          *slog.Logger
	checker      GroupAccessChecker
	dmChecker    DMAccessChecker
	sendBuffer   int
	writeWait    time.Duration
	pongWait     time.Duration
	pingInterval time.Duration
	upgrader     websocket.Upgrader
	onlineTTL    time.Duration
}

func NewHandler(hub *Hub, jwt *auth.JWTManager, redis *redis.Client, checker GroupAccessChecker, dmChecker DMAccessChecker, logger *slog.Logger, sendBuffer int, writeWait, pongWait, pingInterval, onlineTTL time.Duration) *Handler {
	return &Handler{
		hub:          hub,
		jwt:          jwt,
		redis:        redis,
		log:          logger,
		checker:      checker,
		dmChecker:    dmChecker,
		sendBuffer:   sendBuffer,
		writeWait:    writeWait,
		pongWait:     pongWait,
		pingInterval: pingInterval,
		onlineTTL:    onlineTTL,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(header, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		}
	}
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwt.ParseToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("failed to upgrade websocket", slog.Any("error", err))
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	bufferSize := h.sendBuffer
	if bufferSize <= 0 {
		bufferSize = 128
	}

	client := &Client{
		UserID: claims.UserID,
		Send:   make(chan []byte, bufferSize),
		groups: make(map[string]struct{}),
		dmRooms: make(map[string]struct{}),
		closeFn: func() {
			cancel()
			_ = conn.Close()
		},
	}
	h.hub.Register(client)
	defer h.hub.Unregister(client)
	defer client.close()

	_ = h.touchUserPresence(ctx, client.UserID)

	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(h.pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(h.pongWait))
		_ = h.touchUserPresence(ctx, client.UserID)
		return nil
	})

	go h.writePump(ctx, conn, client)
	h.readPump(ctx, conn, client)
}

type command struct {
	Action            string   `json:"action"`
	GroupIDs          []string `json:"group_ids"`
	ConversationIDs   []string `json:"conversation_ids"`
}

func (h *Handler) readPump(ctx context.Context, conn *websocket.Conn, client *Client) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var cmd command
		if err := conn.ReadJSON(&cmd); err != nil {
			return
		}

		switch cmd.Action {
		case "subscribe":
			allowed := make([]string, 0, len(cmd.GroupIDs))
			for _, groupID := range cmd.GroupIDs {
				if h.checker.CanAccessGroup(ctx, client.UserID, groupID) {
					allowed = append(allowed, groupID)
					_ = h.touchGroupPresence(ctx, client.UserID, groupID)
				}
			}
			if len(allowed) > 0 {
				h.hub.Subscribe(client, allowed)
				_ = conn.WriteJSON(map[string]any{
					"type": "subscription.updated",
					"data": map[string]any{"group_ids": allowed},
				})
			}
		case "unsubscribe":
			h.hub.Unsubscribe(client, cmd.GroupIDs)
			_ = conn.WriteJSON(map[string]any{
				"type": "subscription.updated",
				"data": map[string]any{"group_ids": []string{}},
			})
		case "subscribe_dm":
			if h.dmChecker == nil {
				continue
			}
			allowed := make([]string, 0, len(cmd.ConversationIDs))
			for _, convID := range cmd.ConversationIDs {
				if h.dmChecker.CanAccessConversation(ctx, client.UserID, convID) {
					allowed = append(allowed, convID)
				}
			}
			if len(allowed) > 0 {
				h.hub.SubscribeDM(client, allowed)
				_ = conn.WriteJSON(map[string]any{
					"type": "subscription.updated",
					"data": map[string]any{"dm_conversation_ids": allowed},
				})
			}
		case "unsubscribe_dm":
			h.hub.UnsubscribeDM(client, cmd.ConversationIDs)
			_ = conn.WriteJSON(map[string]any{
				"type": "subscription.updated",
				"data": map[string]any{"dm_conversation_ids": []string{}},
			})
		}
	}
}

func (h *Handler) writePump(ctx context.Context, conn *websocket.Conn, client *Client) {
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-client.Send:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			_ = h.touchUserPresence(ctx, client.UserID)
		}
	}
}

func (h *Handler) touchUserPresence(ctx context.Context, userID uint64) error {
	return h.redis.Set(ctx, redisx.OnlineUserKey(userID), "1", h.onlineTTL).Err()
}

func (h *Handler) touchGroupPresence(ctx context.Context, userID uint64, groupID string) error {
	return h.redis.Set(ctx, redisx.OnlineGroupKey(groupID, userID), "1", h.onlineTTL).Err()
}

func MarshalEnvelope(envelope any) ([]byte, error) {
	return json.Marshal(envelope)
}
