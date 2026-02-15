package pageinsight

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
	"time"
)

var errBlockedAddress = errors.New("request to private/reserved network address is not allowed")

// This code is added to address:
// - https://snyk.io/articles/how-to-avoid-ssrf-vulnerability-in-go-applications/
// - https://logoi.dny.dev/2022/12/02/implementing-ssrf-protections-in-golang/

// reservedPrefixes are CIDR ranges not covered by the netip.Addr helper methods
// (IsLoopback, IsPrivate, IsLinkLocalUnicast, IsLinkLocalMulticast, IsUnspecified).
var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),   // Carrier-grade NAT (RFC 6598)
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments (RFC 6890)
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1 (RFC 5737)
	netip.MustParsePrefix("198.18.0.0/15"),   // Benchmarking (RFC 2544)
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2 (RFC 5737)
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3 (RFC 5737)
}

// safeDialer returns a net.Dialer whose Control function rejects connections
// to private, loopback, link-local, and other reserved IP ranges. The check
// runs at dial time (after DNS resolution), which also prevents DNS-rebinding.
func safeDialer() *net.Dialer {
	return &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   blockPrivateAddresses,
	}
}

func blockPrivateAddresses(_ string, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("%w: %w", errBlockedAddress, err)
	}

	if isBlockedIP(addrPort.Addr()) {
		return fmt.Errorf("%w: %s", errBlockedAddress, addrPort.Addr())
	}

	return nil
}

func isBlockedIP(addr netip.Addr) bool {
	// Unmap IPv4-in-IPv6 (e.g. ::ffff:127.0.0.1 → 127.0.0.1) so that
	// mapped addresses cannot bypass IPv4 checks.
	addr = addr.Unmap()

	// Block private addresses or everything that isn't globally routable unicast.
	if !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return true
	}

	for _, p := range reservedPrefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
