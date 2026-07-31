package errors

import (
	"fmt"
	"net/http"
)

type ErrorCode int

const (
	ErrCodeNone             ErrorCode = 200
	ErrCodeUnauthorized     ErrorCode = 401
	ErrCodeForbidden       ErrorCode = 403
	ErrCodeNotFound        ErrorCode = 404
	ErrCodeBadRequest       ErrorCode = 400
	ErrCodeInternalError   ErrorCode = 500
	ErrCodeAsyncPending    ErrorCode = 431
	ErrCodeResourceInUse   ErrorCode = 433
	ErrCodeInvalidParameter ErrorCode = 430
)

type IaaSError struct {
	Code          ErrorCode `json:"code"`
	Message       string    `json:"message"`
	Detail        string    `json:"detail,omitempty"`
	HTTPStatusVal int       `json:"-"`
}

func (e *IaaSError) Error() string { return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail) }
func (e *IaaSError) HTTPStatus() int { return e.HTTPStatusVal }

func NewNotFoundError(resource, id string) *IaaSError {
	return &IaaSError{Code: ErrCodeNotFound, Message: resource + " not found", Detail: "ID: " + id, HTTPStatusVal: http.StatusNotFound}
}

func NewBadRequestError(msg string) *IaaSError {
	return &IaaSError{Code: ErrCodeBadRequest, Message: msg, HTTPStatusVal: http.StatusBadRequest}
}

func NewInternalError(msg string) *IaaSError {
	return &IaaSError{Code: ErrCodeInternalError, Message: "Internal error", Detail: msg, HTTPStatusVal: http.StatusInternalServerError}
}

func NewUnauthorizedError(msg string) *IaaSError {
	return &IaaSError{Code: ErrCodeUnauthorized, Message: "Unauthorized", Detail: msg, HTTPStatusVal: http.StatusUnauthorized}
}

func NewForbiddenError(msg string) *IaaSError {
	return &IaaSError{Code: ErrCodeForbidden, Message: "Forbidden", Detail: msg, HTTPStatusVal: http.StatusForbidden}
}

func NewResourceInUseError(resource, reason string) *IaaSError {
	return &IaaSError{Code: ErrCodeResourceInUse, Message: resource + " in use", Detail: reason, HTTPStatusVal: http.StatusConflict}
}

func NewAsyncJobPendingError(jobID string) *IaaSError {
	return &IaaSError{Code: ErrCodeAsyncPending, Message: "Job is pending", Detail: "Job ID: " + jobID, HTTPStatusVal: http.StatusAccepted}
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
	ErrorCode int    `json:"errorcode"`
	ErrorText string `json:"errortext"`
}
