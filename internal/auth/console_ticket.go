package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/virtfoundry/core/internal/platform"
)

// DefaultConsoleTicketTTL is short on purpose: a ticket only has to survive the
// round trip between the API response and the browser opening the WebSocket.
const DefaultConsoleTicketTTL = 30 * time.Second

const (
	ticketPurposeConsole = "console"
	ticketPurposeEvents  = "events"
)

var (
	ErrConsoleTicketInvalid = errors.New("invalid or expired console ticket")
	ErrEventsTicketInvalid  = errors.New("invalid or expired events ticket")
)

// ConsoleTicket authorizes exactly one VNC session for one VM. It is the only
// credential that may appear in a console URL: it is short-lived and carries
// no permission other than vms:console.
type ConsoleTicket struct {
	UserID    string
	Username  string
	Role      platform.Role
	TenantID  string
	VMName    string
	ExpiresAt time.Time
}

// EventsTicket authorizes a /ws/events WebSocket for one actor. Short-lived so
// the browser never puts the session JWT in the query string.
type EventsTicket struct {
	UserID    string
	Username  string
	Role      platform.Role
	TenantID  string
	ExpiresAt time.Time
}

type ticketClaims struct {
	Purpose  string        `json:"purpose"`
	UserID   string        `json:"user_id"`
	Username string        `json:"username"`
	Role     platform.Role `json:"role"`
	TenantID string        `json:"tenant_id,omitempty"`
	VMName   string        `json:"vm_name,omitempty"`
	jwt.RegisteredClaims
}

// ConsoleTicketStore issues HMAC-signed short-lived tickets that any API
// replica sharing JWT_SECRET can redeem (core#133). A local jti burn map gives
// best-effort single-use on the minting replica; cross-replica replay is
// bounded by the short TTL.
type ConsoleTicketStore struct {
	mu     sync.Mutex
	ttl    time.Duration
	secret []byte
	now    func() time.Time
	// burned JTIs (best-effort single-use on this process).
	burned map[string]time.Time
}

func NewConsoleTicketStore(ttl time.Duration, secret []byte) *ConsoleTicketStore {
	if ttl <= 0 {
		ttl = DefaultConsoleTicketTTL
	}
	if len(secret) == 0 {
		panic("console ticket store requires a non-empty signing secret")
	}
	return &ConsoleTicketStore{
		ttl:    ttl,
		secret: append([]byte(nil), secret...),
		now:    time.Now,
		burned: make(map[string]time.Time),
	}
}

func (s *ConsoleTicketStore) TTL() time.Duration { return s.ttl }

// Issue mints a signed console ticket for the given VM.
func (s *ConsoleTicketStore) Issue(t ConsoleTicket) (string, time.Time, error) {
	return s.issue(ticketPurposeConsole, t.UserID, t.Username, t.Role, t.TenantID, t.VMName)
}

// IssueEvents mints a signed events ticket for the actor (no VM pin).
func (s *ConsoleTicketStore) IssueEvents(t EventsTicket) (string, time.Time, error) {
	return s.issue(ticketPurposeEvents, t.UserID, t.Username, t.Role, t.TenantID, "")
}

func (s *ConsoleTicketStore) issue(purpose, userID, username string, role platform.Role, tenantID, vmName string) (string, time.Time, error) {
	jti, err := newJTI()
	if err != nil {
		return "", time.Time{}, err
	}
	exp := s.now().Add(s.ttl)
	claims := ticketClaims{
		Purpose:  purpose,
		UserID:   userID,
		Username: username,
		Role:     role,
		TenantID: tenantID,
		VMName:   vmName,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(s.now()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// Redeem consumes a console ticket (purpose=console).
func (s *ConsoleTicketStore) Redeem(token string) (ConsoleTicket, error) {
	claims, err := s.redeem(token, ticketPurposeConsole, ErrConsoleTicketInvalid)
	if err != nil {
		return ConsoleTicket{}, err
	}
	return ConsoleTicket{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Role:      claims.Role,
		TenantID:  claims.TenantID,
		VMName:    claims.VMName,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

// RedeemEvents consumes an events ticket (purpose=events).
func (s *ConsoleTicketStore) RedeemEvents(token string) (EventsTicket, error) {
	claims, err := s.redeem(token, ticketPurposeEvents, ErrEventsTicketInvalid)
	if err != nil {
		return EventsTicket{}, err
	}
	return EventsTicket{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Role:      claims.Role,
		TenantID:  claims.TenantID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (s *ConsoleTicketStore) redeem(token, wantPurpose string, invalid error) (*ticketClaims, error) {
	if token == "" {
		return nil, invalid
	}
	parsed, err := jwt.ParseWithClaims(token, &ticketClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithTimeFunc(s.now))
	if err != nil || !parsed.Valid {
		return nil, invalid
	}
	claims, ok := parsed.Claims.(*ticketClaims)
	if !ok || claims.Purpose != wantPurpose {
		return nil, invalid
	}
	if claims.ID == "" {
		return nil, invalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictBurnedLocked()
	if _, seen := s.burned[claims.ID]; seen {
		return nil, invalid
	}
	exp := s.now().Add(s.ttl)
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	s.burned[claims.ID] = exp
	return claims, nil
}

func (s *ConsoleTicketStore) evictBurnedLocked() {
	now := s.now()
	for id, exp := range s.burned {
		if now.After(exp) {
			delete(s.burned, id)
		}
	}
}

func newJTI() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
