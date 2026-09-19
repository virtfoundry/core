package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/virtfoundry/core/internal/pkg/logger"
	"go.uber.org/zap"
)

// Login throttle defaults, applied when config leaves values unset.
const (
	DefaultUserMaxFailures = 5
	DefaultIPMaxFailures   = 20
	DefaultWindow          = 10 * time.Minute
	DefaultLockout         = 5 * time.Minute
)

// ThrottleParams configures a LoginThrottle. Non-positive values fall back to defaults.
type ThrottleParams struct {
	UserMaxFailures int
	IPMaxFailures   int
	Window          time.Duration
	Lockout         time.Duration
}

type attempts struct {
	failures     int
	lastFailure  time.Time
	blockedUntil time.Time
}

// LoginThrottle counts login failures per client IP and per username and
// temporarily blocks keys that exceed the configured thresholds. Throttling
// (not permanent lockout) bounds online brute force without letting an
// attacker keep a victim account locked forever.
type LoginThrottle struct {
	mu          sync.Mutex
	byIP        map[string]*attempts
	byUser      map[string]*attempts
	userMax     int
	ipMax       int
	window      time.Duration
	lockout     time.Duration
	stopJanitor chan struct{}
}

func NewLoginThrottle(p ThrottleParams) *LoginThrottle {
	if p.UserMaxFailures <= 0 {
		p.UserMaxFailures = DefaultUserMaxFailures
	}
	if p.IPMaxFailures <= 0 {
		p.IPMaxFailures = DefaultIPMaxFailures
	}
	if p.Window <= 0 {
		p.Window = DefaultWindow
	}
	if p.Lockout <= 0 {
		p.Lockout = DefaultLockout
	}
	t := &LoginThrottle{
		byIP:        make(map[string]*attempts),
		byUser:      make(map[string]*attempts),
		userMax:     p.UserMaxFailures,
		ipMax:       p.IPMaxFailures,
		window:      p.Window,
		lockout:     p.Lockout,
		stopJanitor: make(chan struct{}),
	}
	go t.janitor()
	return t
}

// Allow reports whether a login attempt from ip for username may proceed.
// When blocked, ok is false and retryAfter is the remaining block time.
func (t *LoginThrottle) Allow(ip, username string) (retryAfter time.Duration, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	if a := t.byIP[ip]; a != nil && now.Before(a.blockedUntil) {
		return a.blockedUntil.Sub(now), false
	}
	if a := t.byUser[username]; a != nil && now.Before(a.blockedUntil) {
		return a.blockedUntil.Sub(now), false
	}
	return 0, true
}

// Failure records a failed login attempt for ip and username, blocking each
// key once its threshold is reached.
func (t *LoginThrottle) Failure(ip, username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.record(t.byIP, ip, t.ipMax, "ip", now)
	t.record(t.byUser, username, t.userMax, "username", now)
}

func (t *LoginThrottle) record(m map[string]*attempts, key string, max int, dimension string, now time.Time) {
	a := m[key]
	if a == nil {
		a = &attempts{}
		m[key] = a
	}
	if now.Sub(a.lastFailure) > t.window {
		a.failures = 0
	}
	a.failures++
	a.lastFailure = now
	if a.failures >= max {
		a.blockedUntil = now.Add(t.lockout)
		a.failures = 0
		logger.Warn("login throttled",
			zap.String("dimension", dimension),
			zap.String("key", key),
			zap.Duration("lockout", t.lockout),
		)
	}
}

// Success clears the per-username failure state after a successful login. The
// per-IP counter is intentionally kept so a valid credential cannot be used to
// reset the IP budget while brute-forcing other accounts.
func (t *LoginThrottle) Success(ip, username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.byUser, username)
}

// Stop terminates the background cleanup goroutine.
func (t *LoginThrottle) Stop() {
	close(t.stopJanitor)
}

func (t *LoginThrottle) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopJanitor:
			return
		case now := <-ticker.C:
			t.mu.Lock()
			t.purge(t.byIP, now)
			t.purge(t.byUser, now)
			t.mu.Unlock()
		}
	}
}

func (t *LoginThrottle) purge(m map[string]*attempts, now time.Time) {
	horizon := t.window
	if t.lockout > horizon {
		horizon = t.lockout
	}
	for key, a := range m {
		last := a.lastFailure
		if a.blockedUntil.After(last) {
			last = a.blockedUntil
		}
		if now.Sub(last) > horizon {
			delete(m, key)
		}
	}
}

// ClientIP extracts the client IP honoring the proxy headers set by the UI
// nginx (X-Real-IP) and standard proxies (X-Forwarded-For), falling back to
// RemoteAddr. The UI nginx proxies /api/ to this server, so RemoteAddr alone
// would group all UI users under the nginx address.
func ClientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, _ := strings.Cut(xff, ","); strings.TrimSpace(first) != "" {
			return strings.TrimSpace(first)
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
