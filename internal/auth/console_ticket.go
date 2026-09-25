package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/virtfoundry/core/internal/platform"
)

// DefaultConsoleTicketTTL is short on purpose: a ticket only has to survive the
// round trip between the API response and the browser opening the WebSocket.
const DefaultConsoleTicketTTL = 30 * time.Second

var ErrConsoleTicketInvalid = errors.New("invalid or expired console ticket")

// ConsoleTicket authorizes exactly one VNC session for one VM. It is the only
// credential that may appear in a console URL: it is single use, expires in
// seconds, and carries no permission other than vms:console.
type ConsoleTicket struct {
	UserID    string
	Username  string
	Role      platform.Role
	TenantID  string
	VMName    string
	ExpiresAt time.Time
}

// ConsoleTicketStore issues and redeems console tickets. Tickets live in memory
// only; losing them on restart just means the browser asks for a new one.
type ConsoleTicketStore struct {
	mu  sync.Mutex
	ttl time.Duration
	now func() time.Time
	// Keyed by SHA-256 of the ticket so the raw credential is never retained.
	tickets map[[32]byte]ConsoleTicket
}

func NewConsoleTicketStore(ttl time.Duration) *ConsoleTicketStore {
	if ttl <= 0 {
		ttl = DefaultConsoleTicketTTL
	}
	return &ConsoleTicketStore{
		ttl:     ttl,
		now:     time.Now,
		tickets: make(map[[32]byte]ConsoleTicket),
	}
}

func (s *ConsoleTicketStore) TTL() time.Duration { return s.ttl }

// Issue mints a ticket for the given VM and returns the opaque credential.
func (s *ConsoleTicketStore) Issue(t ConsoleTicket) (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictExpiredLocked()
	t.ExpiresAt = s.now().Add(s.ttl)
	s.tickets[sha256.Sum256([]byte(token))] = t
	return token, t.ExpiresAt, nil
}

// Redeem consumes a ticket. A ticket is always removed on lookup, so a replayed
// or expired credential is rejected.
func (s *ConsoleTicketStore) Redeem(token string) (ConsoleTicket, error) {
	if token == "" {
		return ConsoleTicket{}, ErrConsoleTicketInvalid
	}
	key := sha256.Sum256([]byte(token))

	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[key]
	if !ok {
		return ConsoleTicket{}, ErrConsoleTicketInvalid
	}
	delete(s.tickets, key)
	if s.now().After(t.ExpiresAt) {
		return ConsoleTicket{}, ErrConsoleTicketInvalid
	}
	return t, nil
}

func (s *ConsoleTicketStore) evictExpiredLocked() {
	now := s.now()
	for k, t := range s.tickets {
		if now.After(t.ExpiresAt) {
			delete(s.tickets, k)
		}
	}
}
