package network

import "strconv"

// This file builds netsh argument lists and nothing else: no process is run
// here, so every command the application can issue is covered by plain unit
// tests. Arguments are passed as separate argv entries, so names holding a
// space ("Ethernet 2") need no quoting of their own.

// validate=no keeps netsh from probing each DNS server before accepting it,
// which otherwise blocks for seconds on a network that is not reachable yet —
// the normal case when the address was only just changed.
const noValidate = "validate=no"

func staticCommands(iface string, cfg StaticConfig) [][]string {
	address := []string{"interface", "ipv4", "set", "address", "name=" + iface, "static", cfg.Address, cfg.Mask}
	if cfg.Gateway != "" {
		address = append(address, cfg.Gateway)
	}

	return append([][]string{address}, dnsCommands(iface, cfg.DNS)...)
}

func dhcpCommands(iface string) [][]string {
	return [][]string{
		{"interface", "ipv4", "set", "address", "name=" + iface, "source=dhcp"},
		{"interface", "ipv4", "set", "dnsservers", "name=" + iface, "source=dhcp"},
	}
}

// dnsCommands always rewrites the DNS servers, clearing them when the profile
// lists none: carrying the previous profile's resolvers over into a new network
// is the kind of surprise that costs an hour of diagnosis.
func dnsCommands(iface string, servers []string) [][]string {
	if len(servers) == 0 {
		return [][]string{{"interface", "ipv4", "set", "dnsservers", "name=" + iface, "source=static", "address=none"}}
	}

	commands := [][]string{
		{"interface", "ipv4", "set", "dnsservers", "name=" + iface, "static", servers[0], "primary", noValidate},
	}
	for i, server := range servers[1:] {
		commands = append(commands, []string{
			"interface", "ipv4", "add", "dnsservers", "name=" + iface, server, "index=" + strconv.Itoa(i+2), noValidate,
		})
	}
	return commands
}

func connectWiFiCommand(iface, ssid string) []string {
	return []string{"wlan", "connect", "name=" + ssid, "interface=" + iface}
}
