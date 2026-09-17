package network

import (
	"net"
	"testing"
)

func TestUsableIPv4(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"configured address", "10.56.20.9", true},
		{"private range", "192.168.1.50", true},
		{"public address", "9.9.9.9", true},
		// Windows puts one of these on any adapter without a lease, next to the
		// real address and in no guaranteed order.
		{"link-local autoconfiguration", "169.254.50.233", false},
		{"link-local boundary, low", "169.254.0.1", false},
		{"link-local boundary, high", "169.254.255.254", false},
		{"just outside link-local", "169.253.255.254", true},
		{"IPv6 is not handled here", "2001:db8::1", false},
		{"IPv6 link-local", "fe80::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usableIPv4(net.ParseIP(tt.ip)); got != tt.want {
				t.Fatalf("usableIPv4(%s) = %t, want %t", tt.ip, got, tt.want)
			}
		})
	}
}

func TestUsableIPv4RejectsNothing(t *testing.T) {
	if usableIPv4(nil) {
		t.Fatal("a missing address must not be reported as the adapter's")
	}
}
