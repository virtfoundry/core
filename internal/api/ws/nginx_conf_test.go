package ws

import (
	"os"
	"regexp"
	"testing"
)

// The UI proxy must forward the Host header with its port: OriginChecker compares the browser
// Origin (host:port) with the request Host, so "Host $host" (no port) made every WebSocket upgrade
// a 403 when the UI was served on a non-default port, for example through kubectl port-forward.
func TestUINginxKeepsThePortInTheHostHeader(t *testing.T) {
	raw, err := os.ReadFile("../../../docker/nginx-ui.conf")
	if err != nil {
		t.Fatalf("read nginx-ui.conf: %v", err)
	}
	conf := string(raw)
	if regexp.MustCompile(`proxy_set_header\s+Host\s+\$host\s*;`).MatchString(conf) {
		t.Fatal(`nginx-ui.conf sends "Host $host" (no port) to the API; use $http_host`)
	}
	if n := len(regexp.MustCompile(`proxy_set_header\s+Host\s+\$http_host\s*;`).FindAllString(conf, -1)); n < 2 {
		t.Fatalf("expected $http_host in both /api/ and /ws/, found %d", n)
	}
}
