package store

import (
	"fmt"
	"net"

	"github.com/virtfoundry/core/internal/config"
)

func nextIPString(ip string) (string, bool) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", false
	}
	parsed = parsed.To4()
	if parsed == nil {
		return "", false
	}
	for i := len(parsed) - 1; i >= 0; i-- {
		parsed[i]++
		if parsed[i] != 0 {
			break
		}
	}
	return parsed.String(), true
}

func ipGreater(a, b string) bool {
	aa := net.ParseIP(a).To16()
	bb := net.ParseIP(b).To16()
	if aa == nil || bb == nil {
		return false
	}
	for i := range aa {
		if aa[i] > bb[i] {
			return true
		}
		if aa[i] < bb[i] {
			return false
		}
	}
	return false
}

func ipInReserved(addr string, reserved []config.ReservedIPRange) bool {
	if net.ParseIP(addr) == nil {
		return false
	}
	for _, r := range reserved {
		if net.ParseIP(r.Start) == nil || net.ParseIP(r.End) == nil {
			continue
		}
		// addr in [start, end]
		if !ipGreater(addr, r.End) && !ipGreater(r.Start, addr) {
			return true
		}
	}
	return false
}

func seedIPRangeLocked(m *Memory, networkID, start, end string, reserved []config.ReservedIPRange) error {
	if start == "" || end == "" {
		return nil
	}
	if net.ParseIP(start) == nil || net.ParseIP(end) == nil {
		return fmt.Errorf("invalid ip pool range")
	}
	for addr := start; ; {
		if !ipInReserved(addr, reserved) {
			m.saveIPAddressQuiet(networkID, addr)
		}
		if addr == end {
			break
		}
		next, ok := nextIPString(addr)
		if !ok || ipGreater(next, end) {
			break
		}
		addr = next
	}
	return nil
}

func (m *MySQL) SeedIPPool(networkID, start, end string) error {
	return m.SeedIPPoolExcluding(networkID, start, end, nil)
}

func (m *MySQL) SeedIPPoolExcluding(networkID, start, end string, reserved []config.ReservedIPRange) error {
	if start == "" || end == "" {
		return nil
	}
	for addr := start; ; {
		if !ipInReserved(addr, reserved) {
			_, _ = m.db.Exec(`INSERT IGNORE INTO ip_addresses (id, network_id, address, status, created_at) VALUES (?, ?, ?, 'available', ?)`,
				NewID(), networkID, addr, Now())
		}
		if addr == end {
			break
		}
		next, ok := nextIPString(addr)
		if !ok || ipGreater(next, end) {
			break
		}
		addr = next
	}
	return nil
}
