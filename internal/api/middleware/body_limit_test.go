package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBodyBytesRejectsOversizedContentLength(t *testing.T) {
	h := MaxBodyBytes(64)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run for oversized Content-Length")
	}))

	body := strings.Repeat("x", 128)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestMaxBodyBytesAllowsWithinLimit(t *testing.T) {
	var got string
	h := MaxBodyBytes(64)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		got = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"ok":true}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func TestMaxBodyBytesReturns413WhenStreamExceedsLimit(t *testing.T) {
	h := MaxBodyBytes(32)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err == nil {
			t.Fatal("expected read error for oversized body")
		}
		var maxErr *http.MaxBytesError
		if !errors.As(err, &maxErr) {
			t.Fatalf("read error = %v, want *http.MaxBytesError", err)
		}
		http.Error(w, `{"error":"request body too large"}`, http.StatusRequestEntityTooLarge)
	}))

	body := strings.Repeat("y", 128)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = -1 // unknown length — exercise MaxBytesReader
	req.Header.Set("Transfer-Encoding", "chunked")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%q", rr.Code, http.StatusRequestEntityTooLarge, rr.Body.String())
	}
}

func TestDefaultMaxBodyBytesIsOneMiB(t *testing.T) {
	if DefaultMaxBodyBytes != 1<<20 {
		t.Fatalf("DefaultMaxBodyBytes = %d, want %d", DefaultMaxBodyBytes, 1<<20)
	}
}
