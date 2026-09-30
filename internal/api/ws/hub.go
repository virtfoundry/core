package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/virtfoundry/core/internal/pkg/logger"
)

// Scope decides which tenants' events a client receives. Every client must
// carry a scope: there is no unscoped subscription.
type Scope struct {
	// TenantID restricts delivery to a single tenant.
	TenantID string
	// AllTenants receives every tenant's events and is only granted to platform root.
	AllTenants bool
}

func (s Scope) allows(tenantID string) bool {
	if s.AllTenants {
		return true
	}
	return s.TenantID != "" && s.TenantID == tenantID
}

// Hub broadcasts real-time events to connected UI clients, filtered by the
// tenant each client is scoped to. Channel subscribers (gRPC Watch) share the
// same BroadcastTenant fan-out as WebSocket clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	subs    map[*subscriber]struct{}
}

type Client struct {
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	scope Scope
}

// subscriber is a non-WebSocket consumer (e.g. gRPC WatchInstances).
type subscriber struct {
	ch    chan Event
	scope Scope
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		subs:    make(map[*subscriber]struct{}),
	}
}

// BroadcastTenant delivers an event to clients scoped to tenantID, plus root
// clients subscribed to all tenants. An empty tenantID is dropped rather than
// fanned out, so a caller that forgets to scope an event cannot leak it.
func (h *Hub) BroadcastTenant(tenantID, eventType string, payload interface{}) {
	if tenantID == "" {
		logger.Get().Warn("ws broadcast dropped: missing tenant scope", zap.String("event", eventType))
		return
	}

	msg, err := json.Marshal(Event{Type: eventType, Payload: payload})
	if err != nil {
		logger.Get().Error("ws broadcast marshal", zap.Error(err))
		return
	}

	ev := Event{Type: eventType, Payload: payload}

	h.mu.RLock()
	slow := make([]*Client, 0)
	slowSubs := make([]*subscriber, 0)
	for client := range h.clients {
		if !client.scope.allows(tenantID) {
			continue
		}
		select {
		case client.send <- msg:
		default:
			slow = append(slow, client)
		}
	}
	for sub := range h.subs {
		if !sub.scope.allows(tenantID) {
			continue
		}
		select {
		case sub.ch <- ev:
		default:
			slowSubs = append(slowSubs, sub)
		}
	}
	h.mu.RUnlock()

	if len(slow) > 0 || len(slowSubs) > 0 {
		h.mu.Lock()
		for _, client := range slow {
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		}
		for _, sub := range slowSubs {
			if _, ok := h.subs[sub]; ok {
				delete(h.subs, sub)
				close(sub.ch)
			}
		}
		h.mu.Unlock()
	}
}

// Subscribe registers a buffered channel consumer under scope. cancel must be
// called (typically via defer) to unregister; after cancel the channel is closed.
// A scope that matches no tenant returns a closed channel and a no-op cancel.
func (h *Hub) Subscribe(scope Scope) (<-chan Event, func()) {
	if !scope.AllTenants && scope.TenantID == "" {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}
	sub := &subscriber{ch: make(chan Event, 64), scope: scope}
	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			if _, ok := h.subs[sub]; ok {
				delete(h.subs, sub)
				close(sub.ch)
			}
			h.mu.Unlock()
		})
	}
	return sub.ch, cancel
}

// SubscriberCount returns the number of channel subscribers (tests / diagnostics).
func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}

// Register attaches a connection to the hub under scope. A scope that matches
// no tenant is rejected so an unauthenticated or unresolved caller cannot
// subscribe.
func (h *Hub) Register(conn *websocket.Conn, scope Scope) *Client {
	if !scope.AllTenants && scope.TenantID == "" {
		return nil
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 64), scope: scope}
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
	return client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	h.mu.Unlock()
}

func (c *Client) WritePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
