package ws

import (
	"sync"
)

// Renderer builds a viewer-specific event payload for a subscribed user.
type Renderer func(userID uint64) ([]byte, error)

type Client struct {
	UserID    uint64
	Send      chan []byte
	groups    map[string]struct{}
	closeOnce sync.Once
	closeFn   func()
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		if c.closeFn != nil {
			c.closeFn()
		}
	})
}

// Hub keeps track of local websocket clients and room subscriptions.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	groups  map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		groups:  make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = struct{}{}
	if client.groups == nil {
		client.groups = make(map[string]struct{})
	}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, client)
	for groupID := range client.groups {
		h.unsubscribeLocked(client, groupID)
	}
	close(client.Send)
}

func (h *Hub) Subscribe(client *Client, groupIDs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, groupID := range groupIDs {
		if _, ok := h.groups[groupID]; !ok {
			h.groups[groupID] = make(map[*Client]struct{})
		}
		h.groups[groupID][client] = struct{}{}
		client.groups[groupID] = struct{}{}
	}
}

func (h *Hub) Unsubscribe(client *Client, groupIDs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, groupID := range groupIDs {
		h.unsubscribeLocked(client, groupID)
	}
}

func (h *Hub) BroadcastGroup(groupID string, renderer Renderer) {
	h.mu.RLock()
	groupClients := h.groups[groupID]
	clients := make([]*Client, 0, len(groupClients))
	for client := range groupClients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	cache := map[uint64][]byte{}
	for _, client := range clients {
		payload, ok := cache[client.UserID]
		if !ok {
			rendered, err := renderer(client.UserID)
			if err != nil || len(rendered) == 0 {
				continue
			}
			payload = rendered
			cache[client.UserID] = rendered
		}

		select {
		case client.Send <- payload:
		default:
			client.close()
		}
	}
}

func (h *Hub) unsubscribeLocked(client *Client, groupID string) {
	delete(client.groups, groupID)
	if members, ok := h.groups[groupID]; ok {
		delete(members, client)
		if len(members) == 0 {
			delete(h.groups, groupID)
		}
	}
}
