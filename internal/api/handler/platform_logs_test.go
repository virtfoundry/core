package handler

import "testing"

func TestParseVMLogTail(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{"", 200},
		{"0", 200},
		{"-1", 200},
		{"abc", 200},
		{"50", 50},
		{"10000", 10000},
		{"10001", 10000},
		{"999999", 10000},
	}
	for _, tt := range tests {
		if got := parseVMLogTail(tt.raw); got != tt.want {
			t.Errorf("parseVMLogTail(%q) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}
