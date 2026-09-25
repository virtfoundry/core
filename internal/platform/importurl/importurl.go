// Package importurl validates tenant-supplied image download URLs before they
// reach the CDI importer.
//
// CDI fetches the URL from a pod inside the cluster, so an unchecked URL turns
// "register an ISO template" into a server-side request forgery primitive: cloud
// metadata endpoints (169.254.169.254), in-cluster services
// (https://kubernetes.default.svc), the node itself, or any RFC1918 host the
// cluster can reach.
//
// Two levels are exposed:
//
//   - CheckSafeTarget enforces the rules that never depend on configuration
//     (https only, no credentials, no private/link-local/in-cluster target).
//   - Policy.Validate additionally requires the host to be on the admin
//     allowlist. This is the tenant-facing check.
package importurl

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrRejected wraps every rejection so callers can map it to a 4xx response.
var ErrRejected = errors.New("image import url rejected")

// DefaultAllowedHosts applies when the admin did not configure an allowlist. It
// covers the public ISO sources documented in docs/VM-TEMPLATES.md plus the
// object storage endpoints used for pre-signed ISO downloads.
var DefaultAllowedHosts = []string{
	// Microsoft evaluation ISOs (Windows Server / Windows 11).
	"go.microsoft.com",
	"software.download.prss.microsoft.com",
	"software-static.download.prss.microsoft.com",
	// Linux install media.
	"releases.ubuntu.com",
	"cdimage.ubuntu.com",
	"cdimage.debian.org",
	"cloud.debian.org",
	"download.fedoraproject.org",
	"download.rockylinux.org",
	"mirrors.almalinux.org",
	// Object storage, typically with a pre-signed URL.
	"s3.amazonaws.com",
	"*.s3.amazonaws.com",
	"*.blob.core.windows.net",
	"storage.googleapis.com",
	"*.r2.cloudflarestorage.com",
}

// blockedSuffixes only resolve inside the cluster, on the node, or on the LAN.
var blockedSuffixes = []string{
	".svc",
	".local",
	".localhost",
	".localdomain",
	".internal",
	".home.arpa",
}

// blockedRanges are the ranges net.IP does not already classify as private,
// loopback or link-local but that must never be an import target.
var blockedRanges = mustParseCIDRs(
	"0.0.0.0/8",          // "this network"
	"100.64.0.0/10",      // RFC6598 shared address space (CGNAT, some CNIs)
	"192.0.0.0/24",       // IETF protocol assignments
	"192.0.2.0/24",       // documentation
	"198.18.0.0/15",      // benchmarking
	"198.51.100.0/24",    // documentation
	"203.0.113.0/24",     // documentation
	"240.0.0.0/4",        // reserved
	"255.255.255.255/32", // broadcast
	"64:ff9b::/96",       // NAT64, can re-encode an IPv4 link-local target
	"64:ff9b:1::/48",     // local-use NAT64
	"100::/64",           // discard-only
	"2001:db8::/32",      // documentation
)

// Policy is an allowlist of hosts CDI may download images from.
type Policy struct {
	rules []hostRule
}

type hostRule struct {
	host string
	// wildcard entries ("*.example.com") match subdomains but not the apex.
	wildcard bool
}

// NewPolicy builds a policy from the admin-configured host list. An empty list
// falls back to DefaultAllowedHosts; use DenyAllPolicy to refuse URL imports
// entirely.
func NewPolicy(allowedHosts []string) *Policy {
	rules := parseRules(allowedHosts)
	if len(rules) == 0 {
		rules = parseRules(DefaultAllowedHosts)
	}
	return &Policy{rules: rules}
}

// DenyAllPolicy rejects every URL. Tenants can still register ISO templates
// from an existing PVC (iso_volume_id).
func DenyAllPolicy() *Policy {
	return &Policy{}
}

// AllowedHosts returns the effective allowlist, for startup logging and
// error messages.
func (p *Policy) AllowedHosts() []string {
	out := make([]string, 0, len(p.rules))
	for _, r := range p.rules {
		if r.wildcard {
			out = append(out, "*."+r.host)
			continue
		}
		out = append(out, r.host)
	}
	return out
}

// Validate reports whether raw is an acceptable image import URL: a safe target
// (see CheckSafeTarget) whose host is on the allowlist. Every error wraps
// ErrRejected.
func (p *Policy) Validate(raw string) error {
	host, err := CheckSafeTarget(raw)
	if err != nil {
		return err
	}
	for _, r := range p.rules {
		if r.matches(host) {
			return nil
		}
	}
	if len(p.rules) == 0 {
		return rejectf("url-based image import is disabled; register the ISO from an existing volume instead")
	}
	return rejectf("host %q is not in the image import allowlist (allowed: %s)",
		host, strings.Join(p.AllowedHosts(), ", "))
}

// CheckSafeTarget enforces the transport rules that hold regardless of admin
// configuration and returns the normalized hostname so callers can apply their
// own allowlist on top. Every error wraps ErrRejected.
func CheckSafeTarget(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", rejectf("url is empty")
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", rejectf("url is not parseable")
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return "", rejectf("scheme %q is not allowed, use https", u.Scheme)
	}
	if u.User != nil {
		return "", rejectf("url must not embed credentials")
	}
	if port := u.Port(); port != "" && port != "443" {
		return "", rejectf("port %s is not allowed, use the default https port", port)
	}
	host := normalizeHost(u.Hostname())
	if host == "" {
		return "", rejectf("url has no host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if err := checkIP(ip); err != nil {
			return "", err
		}
		return host, nil
	}
	if err := checkHostname(host); err != nil {
		return "", err
	}
	return host, nil
}

func checkIP(ip net.IP) error {
	// An IPv4-mapped IPv6 literal (::ffff:10.0.0.1) must be judged as IPv4.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	switch {
	case ip.IsLoopback():
		return rejectf("%s is a loopback address", ip)
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return rejectf("%s is a link-local address (cloud metadata)", ip)
	case ip.IsPrivate():
		return rejectf("%s is a private address", ip)
	case ip.IsUnspecified(), ip.IsMulticast(), ip.IsInterfaceLocalMulticast(), !ip.IsGlobalUnicast():
		return rejectf("%s is not a routable public address", ip)
	}
	for _, r := range blockedRanges {
		if r.Contains(ip) {
			return rejectf("%s is in reserved range %s", ip, r)
		}
	}
	return nil
}

func checkHostname(host string) error {
	if host == "localhost" {
		return rejectf("localhost is not a valid import host")
	}
	// A single label resolves through the pod search domains, so "kubernetes"
	// reaches a service in the importer's own namespace.
	if !strings.Contains(host, ".") {
		return rejectf("host %q is a single label and would resolve inside the cluster", host)
	}
	for _, suffix := range blockedSuffixes {
		if strings.HasSuffix(host, suffix) {
			return rejectf("host %q is internal to the cluster or node (%s)", host, suffix)
		}
	}
	return nil
}

func parseRules(entries []string) []hostRule {
	rules := make([]hostRule, 0, len(entries))
	for _, entry := range entries {
		host := normalizeHost(entry)
		wildcard := strings.HasPrefix(host, "*.")
		host = strings.TrimPrefix(host, "*.")
		if host == "" || strings.ContainsAny(host, "/:*") {
			continue
		}
		rules = append(rules, hostRule{host: host, wildcard: wildcard})
	}
	return rules
}

func (r hostRule) matches(host string) bool {
	if r.wildcard {
		return strings.HasSuffix(host, "."+r.host)
	}
	return host == r.host
}

// normalizeHost lowercases and drops the trailing dot of a fully qualified
// name, which is the same DNS name but would defeat suffix matching.
func normalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func rejectf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrRejected, fmt.Sprintf(format, args...))
}

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic("importurl: invalid CIDR " + c + ": " + err.Error())
		}
		out = append(out, n)
	}
	return out
}
