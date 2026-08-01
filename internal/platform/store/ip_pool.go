package store

import (
	"fmt"
	"net"
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

func seedIPRangeLocked(m *Memory, networkID, start, end string) error {
	if start == "" || end == "" {
		return nil
	}
	if net.ParseIP(start) == nil || net.ParseIP(end) == nil {
		return fmt.Errorf("invalid ip pool range")
	}
	for addr := start; ; {
		m.saveIPAddressQuiet(networkID, addr)
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
	if start == "" || end == "" {
		return nil
	}
	for addr := start; ; {
		_, _ = m.db.Exec(`INSERT IGNORE INTO ip_addresses (id, network_id, address, status, created_at) VALUES (?, ?, ?, 'available', ?)`,
			NewID(), networkID, addr, Now())
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
