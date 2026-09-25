package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// hubServer exposes a hub over a real WebSocket so the tenant filter is
// exercised through the same path production uses. The scope comes from the
// query string to keep the test independent of the auth middleware.
func hubServer(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()

	hub := NewHub()
	up := websocket.Upgrader{CheckOrigin: OriginChecker(nil)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		scope := Scope{TenantID: r.URL.Query().Get("tenant")}
		if r.URL.Query().Get("all_tenants") == "true" {
			scope = Scope{AllTenants: true}
		}
		client := hub.Register(conn, scope)
		if client == nil {
			conn.Close()
			return
		}
		go client.WritePump()
		client.ReadPump()
	}))
	t.Cleanup(srv.Close)
	return hub, srv
}

func dialHub(t *testing.T, srv *httptest.Server, query string) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/events?" + query
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", query, err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readEvent(t *testing.T, conn *websocket.Conn) (Event, error) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return Event{}, err
	}
	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	return ev, nil
}

func TestHubDoesNotLeakEventsAcrossTenants(t *testing.T) {
	hub, srv := hubServer(t)
	conn := dialHub(t, srv, "tenant=tenant-a")

	// Wait for the client to be registered so the broadcast is not simply lost.
	waitForClients(t, hub, 1)

	hub.BroadcastTenant("tenant-b", "vm.created", map[string]string{"name": "payments-prod"})

	if ev, err := readEvent(t, conn); err == nil {
		t.Fatalf("tenant-a received tenant-b event %+v", ev)
	}
}

func TestHubDeliversEventsToSameTenant(t *testing.T) {
	hub, srv := hubServer(t)
	conn := dialHub(t, srv, "tenant=tenant-a")
	waitForClients(t, hub, 1)

	hub.BroadcastTenant("tenant-a", "vm.created", map[string]string{"name": "web-01"})

	ev, err := readEvent(t, conn)
	if err != nil {
		t.Fatalf("tenant-a did not receive its own event: %v", err)
	}
	if ev.Type != "vm.created" {
		t.Fatalf("event type = %q, want vm.created", ev.Type)
	}
}

func TestHubDeliversAllTenantsToRootScope(t *testing.T) {
	hub, srv := hubServer(t)
	conn := dialHub(t, srv, "all_tenants=true")
	waitForClients(t, hub, 1)

	hub.BroadcastTenant("tenant-b", "vm.updated", map[string]string{"name": "web-01"})

	ev, err := readEvent(t, conn)
	if err != nil {
		t.Fatalf("root scope did not receive tenant-b event: %v", err)
	}
	if ev.Type != "vm.updated" {
		t.Fatalf("event type = %q, want vm.updated", ev.Type)
	}
}

func TestHubDropsUnscopedBroadcast(t *testing.T) {
	hub, srv := hubServer(t)
	connA := dialHub(t, srv, "tenant=tenant-a")
	connRoot := dialHub(t, srv, "all_tenants=true")
	waitForClients(t, hub, 2)

	hub.BroadcastTenant("", "vm.created", map[string]string{"name": "orphan"})

	if ev, err := readEvent(t, connA); err == nil {
		t.Fatalf("tenant client received unscoped event %+v", ev)
	}
	if ev, err := readEvent(t, connRoot); err == nil {
		t.Fatalf("root client received unscoped event %+v", ev)
	}
}

func TestHubRejectsRegistrationWithoutScope(t *testing.T) {
	hub := NewHub()

	if client := hub.Register(nil, Scope{}); client != nil {
		t.Fatal("Register accepted a client with no tenant scope")
	}
	if got := clientCount(hub); got != 0 {
		t.Fatalf("hub has %d clients, want 0", got)
	}
}

func clientCount(h *Hub) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func waitForClients(t *testing.T, h *Hub, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if clientCount(h) == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("hub has %d clients, want %d", clientCount(h), want)
}
