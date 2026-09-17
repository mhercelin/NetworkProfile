package network

import "net"

// usableIPv4 rejects the addresses that are not the adapter's configuration.
//
// Windows adds a link-local 169.254.x.y address of its own whenever an adapter
// has no lease and no fixed address — an unplugged cable is enough. That
// address sits in the same list as the configured one and is not guaranteed to
// come second, so reporting whichever comes first would sometimes show the
// operator an address they never set.
func usableIPv4(ip net.IP) bool {
	return ip != nil && ip.To4() != nil && !ip.IsLinkLocalUnicast()
}
