//go:build windows

package network

import (
	"fmt"
	"net"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	ifTypeEthernet = 6  // IF_TYPE_ETHERNET_CSMACD
	ifTypeWiFi     = 71 // IF_TYPE_IEEE80211

	flagDHCPEnabled = 0x00000004 // IP_ADAPTER_DHCP_ENABLED

	operStatusUp = 1 // IfOperStatusUp

	ncfPhysical = 0x4 // NCF_PHYSICAL

	// Adapters of every kind are registered under this class key; the ones
	// whose Characteristics lack NCF_PHYSICAL are software adapters (Hyper-V
	// switches, WSL, VPN clients) that would only clutter the picker.
	netClassKey = `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}`
)

// adapters reports every Ethernet and Wi-Fi adapter with its current IPv4
// configuration.
func adapters() ([]Interface, error) {
	rows, err := adapterAddresses()
	if err != nil {
		return nil, err
	}
	physical, classified := physicalAdapterIDs()
	connected := currentSSIDs()

	var out []Interface
	for _, row := range rows {
		kind, ok := kindOf(row.IfType)
		if !ok {
			continue
		}

		id := windows.BytePtrToString(row.AdapterName)
		iface := Interface{
			ID:          id,
			Name:        windows.UTF16PtrToString(row.FriendlyName),
			Description: windows.UTF16PtrToString(row.Description),
			Kind:        kind,
			Virtual:     classified && !physical[strings.ToLower(id)],
			Up:          row.OperStatus == operStatusUp,
			DHCP:        row.Flags&flagDHCPEnabled != 0,
			SSID:        connected[strings.ToLower(id)],
		}

		for unicast := row.FirstUnicastAddress; unicast != nil; unicast = unicast.Next {
			if !usableIPv4(unicast.Address.IP()) {
				continue
			}
			iface.Address = unicast.Address.IP().String()
			iface.Mask = maskFromPrefixLen(int(unicast.OnLinkPrefixLength))
			break
		}
		if gateway := row.FirstGatewayAddress; gateway != nil {
			if ip := gateway.Address.IP(); ip != nil {
				iface.Gateway = ip.String()
			}
		}
		for dns := row.FirstDnsServerAddress; dns != nil; dns = dns.Next {
			if ip := dns.Address.IP(); ip != nil && ip.To4() != nil {
				iface.DNS = append(iface.DNS, ip.String())
			}
		}

		out = append(out, iface)
	}

	return out, nil
}

func adapterAddresses() ([]*windows.IpAdapterAddresses, error) {
	const flags = windows.GAA_FLAG_INCLUDE_GATEWAYS |
		windows.GAA_FLAG_SKIP_MULTICAST |
		windows.GAA_FLAG_SKIP_ANYCAST

	size := uint32(15000)
	// The adapter set can change between sizing the buffer and filling it, so
	// Windows may ask for a bigger one more than once.
	for attempt := 0; attempt < 5; attempt++ {
		buf := make([]byte, size)
		first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))

		err := windows.GetAdaptersAddresses(windows.AF_INET, flags, 0, first, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("GetAdaptersAddresses: %w", err)
		}

		var rows []*windows.IpAdapterAddresses
		for row := first; row != nil; row = row.Next {
			rows = append(rows, row)
		}
		return rows, nil
	}

	return nil, fmt.Errorf("GetAdaptersAddresses: buffer too small after 5 attempts")
}

// physicalAdapterIDs maps the adapter GUIDs Windows considers physical. The
// second result reports whether the classification could be made at all: when
// the registry cannot be read, no adapter is marked virtual, which is noisier
// but never hides the one the user is looking for.
func physicalAdapterIDs() (map[string]bool, bool) {
	ids := make(map[string]bool)

	class, err := registry.OpenKey(registry.LOCAL_MACHINE, netClassKey, registry.READ)
	if err != nil {
		return nil, false
	}
	defer func() { _ = class.Close() }()

	names, err := class.ReadSubKeyNames(-1)
	if err != nil {
		return nil, false
	}

	for _, name := range names {
		sub, err := registry.OpenKey(class, name, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		id, _, idErr := sub.GetStringValue("NetCfgInstanceId")
		characteristics, _, charErr := sub.GetIntegerValue("Characteristics")
		_ = sub.Close()

		if idErr == nil && charErr == nil && characteristics&ncfPhysical != 0 {
			ids[strings.ToLower(id)] = true
		}
	}

	return ids, true
}

func kindOf(ifType uint32) (Kind, bool) {
	switch ifType {
	case ifTypeEthernet:
		return KindEthernet, true
	case ifTypeWiFi:
		return KindWiFi, true
	default:
		return "", false
	}
}

func maskFromPrefixLen(prefixLen int) string {
	if prefixLen < 0 || prefixLen > 32 {
		return ""
	}
	return net.IP(net.CIDRMask(prefixLen, 32)).String()
}
