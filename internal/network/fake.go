package network

import "fmt"

// Call records one write requested of the Fake, so a test can assert what the
// application asked Windows to do, and in which order.
type Call struct {
	Op        string // "static", "dhcp" or "wifi"
	Interface string
	Static    StaticConfig
	SSID      string
}

// Fake is an in-memory Manager. The application layer is developed and tested
// against it: applying a profile for real reconfigures the machine, which is
// not something a test suite should do.
type Fake struct {
	Adapters []Interface
	WiFi     map[string][]string // interface name -> known networks
	Calls    []Call

	// FailOn makes every write to that interface fail, covering the paths where
	// Windows refuses a change.
	FailOn string
}

var _ Manager = (*Fake)(nil)

func (f *Fake) Interfaces() ([]Interface, error) {
	return f.Adapters, nil
}

func (f *Fake) SetStatic(iface string, cfg StaticConfig) error {
	return f.record(Call{Op: "static", Interface: iface, Static: cfg})
}

func (f *Fake) SetDHCP(iface string) error {
	return f.record(Call{Op: "dhcp", Interface: iface})
}

func (f *Fake) ConnectWiFi(iface, ssid string) error {
	return f.record(Call{Op: "wifi", Interface: iface, SSID: ssid})
}

func (f *Fake) KnownWiFiNetworks(iface string) ([]string, error) {
	return f.WiFi[iface], nil
}

func (f *Fake) record(call Call) error {
	if f.FailOn != "" && call.Interface == f.FailOn {
		return fmt.Errorf("fake: write refused on %q", call.Interface)
	}
	f.Calls = append(f.Calls, call)
	return nil
}
