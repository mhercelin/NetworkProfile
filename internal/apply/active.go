package apply

import (
	"strings"

	"networkprofile/internal/network"
	"networkprofile/internal/profile"
)

// ActiveIDs reports every profile whose configuration matches what the adapters
// currently report.
//
// There is deliberately no single "active profile": profiles target adapters,
// and a machine has several. An Ethernet profile and a Wi-Fi profile are active
// at the same time, each on its own adapter, and answering with one of them
// would leave the other looking unapplied.
//
// It lives here rather than in the interface because three surfaces need the
// same answer — the window's list, the notification area menu and the command
// line — and three implementations of "is this one in effect" would drift.
//
// DNS is deliberately left out of the comparison: Windows reorders and
// supplements resolvers on its own, which would make the match flicker.
func ActiveIDs(profiles []profile.Profile, adapters []network.Interface) []string {
	active := make([]string, 0, len(profiles))
	for _, p := range profiles {
		if matches(p, adapters) {
			active = append(active, p.ID)
		}
	}
	return active
}

func matches(p profile.Profile, adapters []network.Interface) bool {
	if len(p.Targets) == 0 {
		return false
	}

	for _, target := range p.Targets {
		adapter, found := adapterNamed(target.Interface, adapters)
		if !found {
			return false
		}

		// A profile that names a network is only in effect when the adapter is
		// on that network. Without this a Wi-Fi profile was reported as applied
		// while the adapter sat connected to nothing at all.
		if target.SSID != "" && !strings.EqualFold(adapter.SSID, target.SSID) {
			return false
		}

		if target.Mode == profile.ModeDHCP {
			if !adapter.DHCP {
				return false
			}
			continue
		}

		if adapter.DHCP ||
			adapter.Address != target.Address ||
			adapter.Mask != target.Mask ||
			adapter.Gateway != target.Gateway {
			return false
		}
	}
	return true
}

func adapterNamed(name string, adapters []network.Interface) (network.Interface, bool) {
	for _, adapter := range adapters {
		if adapter.Name == name {
			return adapter, true
		}
	}
	return network.Interface{}, false
}
