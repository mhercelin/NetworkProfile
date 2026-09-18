package network

// Kind tells an Ethernet adapter from a Wi-Fi one.
type Kind string

const (
	KindEthernet Kind = "ethernet"
	KindWiFi     Kind = "wifi"
)

// Interface is the observed state of one network adapter. It is read through
// the Windows IP Helper API rather than parsed out of netsh, whose output is
// translated into the language of the running Windows.
type Interface struct {
	ID          string   `json:"id"`          // adapter GUID, "{A1B2...}", stable across renames
	Name        string   `json:"name"`        // friendly name, the one netsh takes: "Ethernet 2"
	Description string   `json:"description"` // adapter model, e.g. "Intel(R) I211 Gigabit"
	Kind        Kind     `json:"kind"`
	Virtual     bool     `json:"virtual"` // Hyper-V, WSL and other software adapters
	Up          bool     `json:"up"`
	DHCP        bool     `json:"dhcp"`
	Address     string   `json:"address"`
	Mask        string   `json:"mask"`
	Gateway     string   `json:"gateway"`
	DNS         []string `json:"dns"`
	// SSID is the network a wireless adapter is actually connected to, empty
	// when it is on none. Without it, a profile that says "join network X"
	// could be reported as being in effect while the adapter sits connected to
	// nothing at all.
	SSID string `json:"ssid"`
}

// StaticConfig is a fixed address to apply to an interface.
type StaticConfig struct {
	Address string
	Mask    string
	Gateway string
	DNS     []string
}

// Manager drives the adapters. The real implementation talks to Windows; tests
// drive the application layer through Fake.
type Manager interface {
	Interfaces() ([]Interface, error)
	SetStatic(iface string, cfg StaticConfig) error
	SetDHCP(iface string) error
	ConnectWiFi(iface, ssid string) error
	KnownWiFiNetworks(iface string) ([]string, error)
}
