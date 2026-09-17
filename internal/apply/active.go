package apply

import (
	"networkprofile/internal/network"
	"networkprofile/internal/profile"
)

// Active reports the identifier of the profile whose configuration matches what
// the adapters currently report, or an empty string when none does.
//
// It lives here rather than in the interface because two surfaces need the same
// answer — the window's list and the notification area menu — and two
// implementations of "is this the one" would drift apart.
//
// DNS is deliberately left out of the comparison: Windows reorders and
// supplements resolvers on its own, which would make the match flicker.
func Active(profiles []profile.Profile, adapters []network.Interface) string {
	for _, p := range profiles {
		if matches(p, adapters) {
			return p.ID
		}
	}
	return ""
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
