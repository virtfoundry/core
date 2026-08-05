package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, err error) {
	var iaErr *iaerrors.IaaSError
	if errors.As(err, &iaErr) {
		msg := iaErr.Message
		if iaErr.Detail != "" {
			msg = msg + ": " + iaErr.Detail
		}
		respondJSON(w, iaErr.HTTPStatus(), map[string]string{"error": sanitizeClientError(msg)})
		return
	}
	status := http.StatusInternalServerError
	msg := sanitizeClientError(err.Error())
	if msg == "tenant not found" || msg == "tenant_id required for root" || msg == "no tenant assigned" {
		status = http.StatusBadRequest
	}
	if strings.Contains(msg, "not found") || strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "required") || strings.Contains(msg, "overlaps") ||
		strings.Contains(msg, "must be") {
		status = http.StatusBadRequest
	}
	respondJSON(w, status, map[string]string{"error": msg})
}

func sanitizeClientError(msg string) string {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "forbidden:") || strings.Contains(lower, "serviceaccount:") {
		return "sem permissão para esta operação — contate o administrador"
	}
	if strings.Contains(lower, "stream instance logs:") || strings.Contains(lower, "instance runtime") {
		return "não foi possível carregar os logs da instância"
	}
	if strings.Contains(lower, "stream pod logs:") || strings.Contains(lower, "pods/log") {
		return "não foi possível carregar os logs da instância"
	}
	return msg
}
