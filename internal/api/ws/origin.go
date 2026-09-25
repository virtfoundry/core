package ws

import (
	"net/http"
	"net/url"
	"strings"
)

// OriginChecker builds the CheckOrigin func for the events upgrader. It accepts
// the request host itself (the UI is served same-origin) plus any origin listed
// in security.allowed_origins, and rejects everything else so a third-party page
// cannot open the socket (CSWSH).
//
// A request with no Origin header is allowed: only browsers set Origin, so its
// absence means a non-browser client (CLI, API key consumer) that CSWSH does not
// apply to. Such clients still have to authenticate.
func OriginChecker(allowed []string) func(*http.Request) bool {
	allowSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		if n := normalizeOrigin(a); n != "" {
			allowSet[n] = struct{}{}
		}
	}

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		if strings.EqualFold(u.Host, r.Host) {
			return true
		}
		_, ok := allowSet[normalizeOrigin(origin)]
		return ok
	}
}

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}
