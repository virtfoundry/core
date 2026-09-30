package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/virtfoundry/core/internal/platform"
)

const testTicketSecret = "test-console-ticket-secret-32b!!"

func TestConsoleTicketRedeemReturnsSubject(t *testing.T) {
	s := NewConsoleTicketStore(DefaultConsoleTicketTTL, []byte(testTicketSecret))

	token, expiresAt, err := s.Issue(ConsoleTicket{
		UserID: "u-1", Username: "operator", Role: platform.RoleUser,
		TenantID: "t-1", VMName: "vm-a",
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if token == "" {
		t.Fatal("Issue returned an empty ticket")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expiresAt = %v, want a future time", expiresAt)
	}

	got, err := s.Redeem(token)
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if got.TenantID != "t-1" || got.VMName != "vm-a" || got.UserID != "u-1" {
		t.Fatalf("ticket = %+v, want vm-a in t-1 for u-1", got)
	}
}

func TestConsoleTicketIsSingleUseOnSameReplica(t *testing.T) {
	s := NewConsoleTicketStore(DefaultConsoleTicketTTL, []byte(testTicketSecret))
	token, _, err := s.Issue(ConsoleTicket{TenantID: "t-1", VMName: "vm-a"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := s.Redeem(token); err != nil {
		t.Fatalf("first Redeem: %v", err)
	}
	if _, err := s.Redeem(token); !errors.Is(err, ErrConsoleTicketInvalid) {
		t.Fatalf("second Redeem error = %v, want %v", err, ErrConsoleTicketInvalid)
	}
}

func TestConsoleTicketRedeemableOnSiblingReplica(t *testing.T) {
	secret := []byte(testTicketSecret)
	a := NewConsoleTicketStore(DefaultConsoleTicketTTL, secret)
	b := NewConsoleTicketStore(DefaultConsoleTicketTTL, secret)

	token, _, err := a.Issue(ConsoleTicket{
		UserID: "u-1", Username: "op", Role: platform.RoleUser,
		TenantID: "t-1", VMName: "vm-a",
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	got, err := b.Redeem(token)
	if err != nil {
		t.Fatalf("sibling Redeem: %v", err)
	}
	if got.VMName != "vm-a" || got.TenantID != "t-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestConsoleTicketExpires(t *testing.T) {
	s := NewConsoleTicketStore(30*time.Second, []byte(testTicketSecret))
	now := time.Now()
	s.now = func() time.Time { return now }

	token, _, err := s.Issue(ConsoleTicket{TenantID: "t-1", VMName: "vm-a"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	now = now.Add(31 * time.Second)
	if _, err := s.Redeem(token); !errors.Is(err, ErrConsoleTicketInvalid) {
		t.Fatalf("Redeem error = %v, want %v", err, ErrConsoleTicketInvalid)
	}
}

func TestConsoleTicketRejectsEmptyAndUnknown(t *testing.T) {
	s := NewConsoleTicketStore(DefaultConsoleTicketTTL, []byte(testTicketSecret))

	for _, token := range []string{"", "not-a-ticket"} {
		if _, err := s.Redeem(token); !errors.Is(err, ErrConsoleTicketInvalid) {
			t.Fatalf("Redeem(%q) error = %v, want %v", token, err, ErrConsoleTicketInvalid)
		}
	}
}

func TestConsoleTicketRejectsEventsPurpose(t *testing.T) {
	s := NewConsoleTicketStore(DefaultConsoleTicketTTL, []byte(testTicketSecret))
	token, _, err := s.IssueEvents(EventsTicket{
		UserID: "u-1", Username: "op", Role: platform.RoleUser, TenantID: "t-1",
	})
	if err != nil {
		t.Fatalf("IssueEvents: %v", err)
	}
	if _, err := s.Redeem(token); !errors.Is(err, ErrConsoleTicketInvalid) {
		t.Fatalf("console Redeem(events) = %v, want %v", err, ErrConsoleTicketInvalid)
	}
}

func TestEventsTicketRedeem(t *testing.T) {
	s := NewConsoleTicketStore(DefaultConsoleTicketTTL, []byte(testTicketSecret))
	token, _, err := s.IssueEvents(EventsTicket{
		UserID: "u-1", Username: "op", Role: platform.RoleUser, TenantID: "t-1",
	})
	if err != nil {
		t.Fatalf("IssueEvents: %v", err)
	}
	got, err := s.RedeemEvents(token)
	if err != nil {
		t.Fatalf("RedeemEvents: %v", err)
	}
	if got.UserID != "u-1" || got.TenantID != "t-1" {
		t.Fatalf("got %+v", got)
	}
	if _, err := s.RedeemEvents(token); !errors.Is(err, ErrEventsTicketInvalid) {
		t.Fatalf("second RedeemEvents = %v", err)
	}
}

func TestConsoleTicketsAreUnique(t *testing.T) {
	s := NewConsoleTicketStore(DefaultConsoleTicketTTL, []byte(testTicketSecret))
	seen := make(map[string]bool, 64)

	for i := 0; i < 64; i++ {
		token, _, err := s.Issue(ConsoleTicket{TenantID: "t-1", VMName: "vm-a"})
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		if seen[token] {
			t.Fatalf("duplicate ticket generated: %s", token)
		}
		seen[token] = true
	}
}

func TestViewerHasNoConsolePermission(t *testing.T) {
	if HasPermission(TenantViewerPermissions, PermVMsConsole) {
		t.Fatal("tenant viewer must not hold vms:console")
	}
	for _, perms := range [][]string{TenantOperatorPermissions, TenantAdminPermissions} {
		if !HasPermission(perms, PermVMsConsole) {
			t.Fatalf("%v must hold vms:console", perms)
		}
	}
	if HasPermission(LegacyRolePermissions(platform.RoleUser), PermVMsConsole) {
		t.Fatal("legacy user role must not hold vms:console")
	}
}
