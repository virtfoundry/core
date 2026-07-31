package errors

import (
	"net/http"
	"testing"
)

func TestIaaSError(t *testing.T) {
	err := &IaaSError{Code: ErrCodeNotFound, Message: "not found", Detail: "zone 123"}
	// ErrCodeNotFound = 404
	if err.Error() != "[404] not found: zone 123" {
		t.Errorf("got %s", err.Error())
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("zone", "123")
	if err.Code != ErrCodeNotFound { t.Error("wrong code") }
	if err.HTTPStatusVal != http.StatusNotFound { t.Error("wrong status") }
}

func TestNewBadRequestError(t *testing.T) {
	err := NewBadRequestError("bad param")
	if err.Code != ErrCodeBadRequest { t.Error("wrong code") }
	if err.HTTPStatusVal != http.StatusBadRequest { t.Error("wrong status") }
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError("db fail")
	if err.Code != ErrCodeInternalError { t.Error("wrong code") }
}

func TestNewUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError("bad key")
	if err.Code != ErrCodeUnauthorized { t.Error("wrong code") }
	if err.HTTPStatusVal != http.StatusUnauthorized { t.Error("wrong status") }
}

func TestNewForbiddenError(t *testing.T) {
	err := NewForbiddenError("no perms")
	if err.Code != ErrCodeForbidden { t.Error("wrong code") }
	if err.HTTPStatusVal != http.StatusForbidden { t.Error("wrong status") }
}

func TestNewResourceInUseError(t *testing.T) {
	err := NewResourceInUseError("volume", "has snapshots")
	if err.Code != ErrCodeResourceInUse { t.Error("wrong code") }
	if err.HTTPStatusVal != http.StatusConflict { t.Error("wrong status") }
}

func TestNewAsyncJobPendingError(t *testing.T) {
	err := NewAsyncJobPendingError("job-123")
	if err.Code != ErrCodeAsyncPending { t.Error("wrong code") }
	if err.HTTPStatusVal != http.StatusAccepted { t.Error("wrong status") }
}
