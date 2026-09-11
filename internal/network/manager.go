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
	ID          string // adapter GUID, "{A1B2...}", stable across renames
	Name        string // friendly name, the one netsh takes: "Ethernet 2"
	Description string // adapter model, e.g. "Intel(R) I211 Gigabit"
	Kind        Kind
	Virtual     bool // Hyper-V, WSL and other software adapters
	Up          bool
	DHCP        bool
	Address     string
	Mask        string
	Gateway     string
	DNS         []string
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
