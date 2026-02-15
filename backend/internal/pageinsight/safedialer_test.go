package pageinsight

import (
	"net/netip"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		// Loopback
		{name: "IPv4 loopback", ip: "127.0.0.1", blocked: true},
		{name: "IPv6 loopback", ip: "::1", blocked: true},

		// Private ranges (RFC 1918)
		{name: "10.x.x.x", ip: "10.0.0.1", blocked: true},
		{name: "172.16.x.x", ip: "172.16.0.1", blocked: true},
		{name: "192.168.x.x", ip: "192.168.1.1", blocked: true},

		// Link-local
		{name: "link-local IPv4", ip: "169.254.1.1", blocked: true},
		{name: "link-local IPv6", ip: "fe80::1", blocked: true},

		// Cloud metadata
		{name: "AWS metadata", ip: "169.254.169.254", blocked: true},

		// Carrier-grade NAT (RFC 6598)
		{name: "CGN low", ip: "100.64.0.1", blocked: true},
		{name: "CGN high", ip: "100.127.255.254", blocked: true},

		// TEST-NET ranges (RFC 5737)
		{name: "TEST-NET-1", ip: "192.0.2.1", blocked: true},
		{name: "TEST-NET-2", ip: "198.51.100.1", blocked: true},
		{name: "TEST-NET-3", ip: "203.0.113.1", blocked: true},

		// IETF protocol assignments
		{name: "IETF 192.0.0.x", ip: "192.0.0.1", blocked: true},

		// Benchmarking (RFC 2544)
		{name: "benchmark 198.18.x.x", ip: "198.18.0.1", blocked: true},
		{name: "benchmark 198.19.x.x", ip: "198.19.255.254", blocked: true},

		// Unspecified
		{name: "unspecified IPv4", ip: "0.0.0.0", blocked: true},
		{name: "unspecified IPv6", ip: "::", blocked: true},

		// IPv4-mapped IPv6 (bypass attempt)
		{name: "mapped loopback", ip: "::ffff:127.0.0.1", blocked: true},
		{name: "mapped private", ip: "::ffff:10.0.0.1", blocked: true},
		{name: "mapped metadata", ip: "::ffff:169.254.169.254", blocked: true},
		{name: "mapped public", ip: "::ffff:8.8.8.8", blocked: false},

		// Public IPs - should NOT be blocked
		{name: "Google DNS", ip: "8.8.8.8", blocked: false},
		{name: "Cloudflare DNS", ip: "1.1.1.1", blocked: false},
		{name: "public IPv4", ip: "93.184.216.34", blocked: false},
		{name: "public range near CGN", ip: "100.63.255.255", blocked: false},
		{name: "public range after CGN", ip: "100.128.0.1", blocked: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := netip.ParseAddr(tt.ip)
			if err != nil {
				t.Fatalf("failed to parse IP %q: %v", tt.ip, err)
			}
			if got := isBlockedIP(addr); got != tt.blocked {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}

func TestBlockPrivateAddresses(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{name: "public address", address: "93.184.216.34:443", wantErr: false},
		{name: "loopback", address: "127.0.0.1:80", wantErr: true},
		{name: "private 10.x", address: "10.0.0.5:6379", wantErr: true},
		{name: "AWS metadata", address: "169.254.169.254:80", wantErr: true},
		{name: "invalid address no port", address: "127.0.0.1", wantErr: true},
		{name: "IPv6 bracket format", address: "[::1]:80", wantErr: true},
		{name: "mapped IPv4 loopback", address: "[::ffff:127.0.0.1]:80", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := blockPrivateAddresses("tcp", tt.address, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("blockPrivateAddresses(%q) error = %v, wantErr %v", tt.address, err, tt.wantErr)
			}
		})
	}
}
