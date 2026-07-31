package logs

import "testing"

func TestSanitizeLine(t *testing.T) {
	in := `error: pods "virt-launcher-teste-abc" is forbidden: User "system:serviceaccount:nimbus-system:nimbus-api" cannot get resource "pods/log"`
	out := sanitizeLine(in)
	if out == in {
		t.Fatalf("expected sanitization, got same string")
	}
	if containsAny(out, []string{"virt-launcher", "serviceaccount", "pods/log"}) {
		t.Fatalf("k8s details leaked: %q", out)
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
