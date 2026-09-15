package main

import (
	"testing"
	"time"
)

func TestServerTimeoutsConfigured(t *testing.T) {
	if serverReadHeaderTimeout != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want 10s", serverReadHeaderTimeout)
	}
	if serverReadTimeout != 30*time.Second {
		t.Fatalf("ReadTimeout = %v, want 30s", serverReadTimeout)
	}
	if serverIdleTimeout != 120*time.Second {
		t.Fatalf("IdleTimeout = %v, want 120s", serverIdleTimeout)
	}
}
