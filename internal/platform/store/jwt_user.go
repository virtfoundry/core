package store

import "github.com/virtfoundry/core/internal/platform"

// ResolveJWTUser loads the user for JWT validation.
// Prefer O(1) GetUserForAuth by claims username, then verify UID matches.
// If username is empty or UID mismatches, fall back to GetUser(userID).
func ResolveJWTUser(st Repository, userID, username string) (*platform.User, bool) {
	if username != "" {
		if u, ok := st.GetUserForAuth(username); ok && u.ID == userID {
			return u, true
		}
	}
	return st.GetUser(userID)
}
