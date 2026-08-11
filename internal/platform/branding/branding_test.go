package branding

import "testing"

func TestValidateLinuxBridgeName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		wantErr bool
	}{
		{PublicBridgeName, false},
		{BridgeName, false}, // exactly 15
		{"br0", false},
		{"", true},
		{"virtfoundry-pub0", true}, // 16 chars — Multus IFNAMSIZ failure
		{"abcdefghijklmnop", true}, // 16
	}
	for _, tc := range cases {
		err := ValidateLinuxBridgeName(tc.name)
		if tc.wantErr && err == nil {
			t.Fatalf("%q: want error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%q: unexpected error: %v", tc.name, err)
		}
	}
}

func TestBridgeNameLengths(t *testing.T) {
	t.Parallel()
	if err := ValidateLinuxBridgeName(PublicBridgeName); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLinuxBridgeName(BridgeName); err != nil {
		t.Fatal(err)
	}
	if len(PublicBridgeName) > MaxBridgeNameLen || len(BridgeName) > MaxBridgeNameLen {
		t.Fatalf("branding bridge constants must stay ≤%d chars", MaxBridgeNameLen)
	}
}
