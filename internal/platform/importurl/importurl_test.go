package importurl

import (
	"errors"
	"strings"
	"testing"
)

func TestPolicyRejectsSSRFTargets(t *testing.T) {
	p := NewPolicy([]string{"iso.example.com", "*.cdn.example.com"})

	cases := []struct {
		name string
		url  string
	}{
		{"cloud metadata", "https://169.254.169.254/latest/meta-data/"},
		{"cloud metadata ipv6", "https://[fe80::1]/latest/meta-data/"},
		{"gce metadata name", "https://metadata.google.internal/computeMetadata/v1/"},
		{"kubernetes api service", "https://kubernetes.default.svc/api/v1/namespaces"},
		{"kubernetes api fqdn", "https://kubernetes.default.svc.cluster.local/api"},
		{"single label service", "https://kubernetes/api"},
		{"localhost", "https://localhost/iso"},
		{"localhost suffix", "https://api.localhost/iso"},
		{"loopback v4", "https://127.0.0.1/iso"},
		{"loopback v6", "https://[::1]/iso"},
		{"rfc1918 10/8", "https://10.0.0.5/win.iso"},
		{"rfc1918 172.16/12", "https://172.16.4.9/win.iso"},
		{"rfc1918 192.168/16", "https://192.168.1.10/win.iso"},
		{"ipv6 unique local", "https://[fd00::1]/win.iso"},
		{"ipv4 mapped private", "https://[::ffff:10.0.0.5]/win.iso"},
		{"nat64 link local", "https://[64:ff9b::a9fe:a9fe]/latest/meta-data/"},
		{"cgnat", "https://100.64.0.1/win.iso"},
		{"unspecified", "https://0.0.0.0/win.iso"},
		{"mdns lan host", "https://nas.local/win.iso"},
		{"internal suffix", "https://registry.internal/win.iso"},
		{"plain http", "http://iso.example.com/win.iso"},
		{"file scheme", "file:///etc/shadow"},
		{"gopher scheme", "gopher://iso.example.com/_test"},
		{"no scheme", "iso.example.com/win.iso"},
		{"embedded credentials", "https://user:pass@iso.example.com/win.iso"},
		{"non standard port", "https://iso.example.com:9200/win.iso"},
		{"host not allowlisted", "https://evil.example.com/win.iso"},
		{"apex of wildcard entry", "https://cdn.example.com/win.iso"},
		{"allowlisted host as subdomain suffix", "https://iso.example.com.evil.net/win.iso"},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := p.Validate(tc.url)
			if err == nil {
				t.Fatalf("Validate(%q) = nil, want rejection", tc.url)
			}
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("Validate(%q) error %v does not wrap ErrRejected", tc.url, err)
			}
		})
	}
}

func TestPolicyAllowsConfiguredHosts(t *testing.T) {
	p := NewPolicy([]string{"iso.example.com", "*.cdn.example.com", "IsoMirror.Example.ORG."})

	cases := []string{
		"https://iso.example.com/win2022.iso",
		"https://iso.example.com:443/win2022.iso",
		"https://iso.example.com/win2022.iso?sig=abc123",
		"https://eu.cdn.example.com/win2022.iso",
		"https://deep.eu.cdn.example.com/win2022.iso",
		"https://isomirror.example.org/win2022.iso",
		"https://ISO.EXAMPLE.COM/win2022.iso",
		"https://iso.example.com./win2022.iso",
	}

	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if err := p.Validate(raw); err != nil {
				t.Fatalf("Validate(%q) = %v, want nil", raw, err)
			}
		})
	}
}

func TestDefaultPolicyAllowsDocumentedSourcesAndBlocksOthers(t *testing.T) {
	p := NewPolicy(nil)

	// The Windows Server eval flow in docs/VM-TEMPLATES.md keeps working.
	if err := p.Validate("https://go.microsoft.com/fwlink/?linkid=2195280"); err != nil {
		t.Fatalf("documented Windows eval URL rejected: %v", err)
	}
	if err := p.Validate("https://my-bucket.s3.amazonaws.com/win2022.iso?X-Amz-Signature=abc"); err != nil {
		t.Fatalf("pre-signed object storage URL rejected: %v", err)
	}
	if err := p.Validate("https://mirror.mylab.example.net/win2022.iso"); err == nil {
		t.Fatal("host outside the built-in allowlist was accepted")
	}
}

func TestDenyAllPolicyRejectsEverything(t *testing.T) {
	p := DenyAllPolicy()

	err := p.Validate("https://go.microsoft.com/fwlink/?linkid=2195280")
	if err == nil {
		t.Fatal("DenyAllPolicy accepted a URL")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("error %q should explain that URL import is disabled", err)
	}
	if got := p.AllowedHosts(); len(got) != 0 {
		t.Fatalf("AllowedHosts() = %v, want empty", got)
	}
}

func TestCheckSafeTargetIgnoresAllowlist(t *testing.T) {
	// The k8s sink uses CheckSafeTarget so a custom (public) mirror passes even
	// though it is not in any allowlist, while SSRF targets still fail.
	host, err := CheckSafeTarget("https://mirror.mylab.example.net/win2022.iso")
	if err != nil {
		t.Fatalf("CheckSafeTarget public host = %v, want nil", err)
	}
	if host != "mirror.mylab.example.net" {
		t.Fatalf("host = %q, want mirror.mylab.example.net", host)
	}
	if _, err := CheckSafeTarget("https://kubernetes.default.svc/api"); err == nil {
		t.Fatal("CheckSafeTarget accepted an in-cluster service")
	}
}

func TestNewPolicyIgnoresUnusableEntriesAndFallsBackToDefaults(t *testing.T) {
	p := NewPolicy([]string{"  ", "https://iso.example.com/path", "*"})
	if got := p.AllowedHosts(); len(got) != len(DefaultAllowedHosts) {
		t.Fatalf("AllowedHosts() = %v, want the built-in defaults", got)
	}
}

func TestPolicyAllowedHostsRoundTripsWildcards(t *testing.T) {
	p := NewPolicy([]string{"iso.example.com", "*.cdn.example.com"})
	got := strings.Join(p.AllowedHosts(), ",")
	if got != "iso.example.com,*.cdn.example.com" {
		t.Fatalf("AllowedHosts() = %q", got)
	}
}
