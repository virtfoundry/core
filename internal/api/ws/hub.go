package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/virtfoundry/core/internal/pkg/logger"
)

// Hub broadcasts real-time events to connected UI clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]struct{})}
}

func (h *Hub) Broadcast(eventType string, payload interface{}) {
	msg, err := json.Marshal(Event{Type: eventType, Payload: payload})
	if err != nil {
		logger.Get().Error("ws broadcast marshal", zap.Error(err))
		return
	}

	h.mu.RLock()
	slow := make([]*Client, 0)
	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			slow = append(slow, client)
		}
	}
	h.mu.RUnlock()

	if len(slow) > 0 {
		h.mu.Lock()
		for _, client := range slow {
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		}
		h.mu.Unlock()
	}
}

func (h *Hub) Register(conn *websocket.Conn) *Client {
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 64)}
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
