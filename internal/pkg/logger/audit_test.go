package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
)

func TestAuditIgnoresConfiguredLevel(t *testing.T) {
	if err := Init("error", false); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	prev := audit
	audit = newAudit(&buf)
	defer func() { audit = prev }()

	Audit("api.request", zap.String("path", "/api/v1/vms"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %q", err, buf.String())
	}
	if got["audit"] != true || got["message"] != "api.request" || got["path"] != "/api/v1/vms" {
		t.Fatalf("unexpected record: %v", got)
	}
}
