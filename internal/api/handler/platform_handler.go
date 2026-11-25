package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
)

func nonNilSlice[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

type PlatformHandler struct {
	auth      *auth.Service
	store     store.Repository
	svc       *service.PlatformService
	throttle  *auth.LoginThrottle
	dummyHash string
}

func NewPlatformHandler(authSvc *auth.Service, st store.Repository, svc *service.PlatformService, throttle *auth.LoginThrottle) *PlatformHandler {
	if throttle == nil {
		throttle = auth.NewLoginThrottle(auth.ThrottleParams{})
	}
	// Dummy bcrypt hash (DefaultCost) compared when the username does not
	// exist, so missing-user and wrong-password logins take the same time.
	dummy, err := auth.HashPassword("virtfoundry-login-timing-equalizer")
	if err != nil {
		dummy = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	}
	return &PlatformHandler{auth: authSvc, store: st, svc: svc, throttle: throttle, dummyHash: dummy}
}

func (h *PlatformHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	ip := auth.ClientIP(r)
	if retryAfter, ok := h.throttle.Allow(ip, req.Username); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		http.Error(w, `{"error":"too many login attempts, try again later"}`, http.StatusTooManyRequests)
		return
	}
	user, found := h.store.GetUserByUsername(req.Username)
	hash := h.dummyHash
	if found {
		hash = user.PasswordHash
	}
	valid := auth.CheckPassword(hash, req.Password)
	// State is checked last, after bcrypt, so disabled accounts get the same
	// generic 401 as unknown users and wrong passwords (no enumeration oracle).
	if !found || !valid || user.State == "disabled" {
		h.throttle.Failure(ip, req.Username)
		resp := map[string]interface{}{
			"error":              "invalid credentials",
			"attempts_remaining": h.throttle.Remaining(ip, req.Username),
		}
		if retryAfter, ok := h.throttle.Allow(ip, req.Username); !ok {
			resp["retry_after"] = int(retryAfter.Seconds()) + 1
		}
		respondJSON(w, http.StatusUnauthorized, resp)
		return
	}
	h.throttle.Success(ip, req.Username)
	token, exp, err := h.auth.IssueToken(user)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token, "expires_at": exp,
		"user": h.userPayload(user),
	})
}

func (h *PlatformHandler) userPayload(u *platform.User) map[string]interface{} {
	if u == nil {
		return nil
	}
	actor := h.svc.ActorFromUser(u)
	m := publicUser(u)
	m["permissions"] = actor.Permissions
	return m
}

func (h *PlatformHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	user, _ := h.store.GetUser(claims.UserID)
	payload := map[string]interface{}{
		"id": claims.UserID, "username": claims.Username, "role": claims.Role, "tenant_id": claims.TenantID,
		"email": userEmail(user),
	}
	if actor := middleware.GetActor(r.Context()); actor != nil {
		payload["permissions"] = actor.Permissions
		payload["role_id"] = actor.RoleID
	} else if user != nil {
		payload["permissions"] = h.svc.ActorFromUser(user).Permissions
		payload["role_id"] = user.RoleID
	}
	respondJSON(w, http.StatusOK, payload)
}

const (
	maxVMLogTail      = int64(10_000)
	vmLogFollowMaxAge = 5 * time.Minute
)

func (h *PlatformHandler) DashboardSummary(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	perms := []string{auth.PermAll}
	var role platform.Role
	if actor := middleware.GetActor(r.Context()); actor != nil {
		perms = actor.Permissions
		role = actor.Role
	}
	summary, err := h.svc.DashboardSummary(r.Context(), tid, perms, role)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, summary)
}

func (h *PlatformHandler) Search(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	q := r.URL.Query().Get("q")
	perms := []string{auth.PermAll}
	if actor := middleware.GetActor(r.Context()); actor != nil {
		perms = actor.Permissions
	}
	hits := h.svc.Search(r.Context(), tid, q, perms)
	respondJSON(w, http.StatusOK, map[string]interface{}{"results": nonNilSlice(hits)})
}

func (h *PlatformHandler) Notifications(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	perms := []string{auth.PermAll}
	if actor := middleware.GetActor(r.Context()); actor != nil {
		perms = actor.Permissions
	}
	items := h.svc.Notifications(r.Context(), tid, perms)
	respondJSON(w, http.StatusOK, map[string]interface{}{"notifications": nonNilSlice(items)})
}

func (h *PlatformHandler) tenantID(r *http.Request) (string, error) {
	claims := middleware.GetClaims(r.Context())
	tid := middleware.GetTenantID(r.Context())
	if tid == "" {
		tid = r.URL.Query().Get("tenant_id")
	}
	return h.svc.ResolveTenantID(claims, tid)
}

func publicUser(u *platform.User) map[string]interface{} {
	if u == nil {
		return nil
	}
	return map[string]interface{}{
		"id": u.ID, "username": u.Username, "role": u.Role,
		"role_id": u.RoleID, "tenant_id": u.TenantID, "email": u.Email, "state": u.State,
	}
}

func userEmail(u *platform.User) string {
	if u == nil {
		return ""
	}
	return u.Email
}
