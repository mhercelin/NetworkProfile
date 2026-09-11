//go:build windows

package network

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const netshTimeout = 20 * time.Second

// Windows reads adapter state through the Windows API and writes it through
// netsh, of which only the exit code is relied on — its printed output is
// translated into the language of the running system.
type Windows struct{}

var _ Manager = (*Windows)(nil)

func NewWindows() *Windows { return &Windows{} }

func (w *Windows) Interfaces() ([]Interface, error) {
	return adapters()
}

func (w *Windows) SetStatic(iface string, cfg StaticConfig) error {
	return runAll(staticCommands(iface, cfg))
}

func (w *Windows) SetDHCP(iface string) error {
	return runAll(dhcpCommands(iface))
}

func (w *Windows) ConnectWiFi(iface, ssid string) error {
	return runAll([][]string{connectWiFiCommand(iface, ssid)})
}

func (w *Windows) KnownWiFiNetworks(iface string) ([]string, error) {
	id, err := adapterIDFor(iface)
	if err != nil {
		return nil, err
	}
	return knownWiFiNetworks(id)
}

func adapterIDFor(name string) (string, error) {
	list, err := adapters()
	if err != nil {
		return "", err
	}
	for _, iface := range list {
		if iface.Name == name {
			return iface.ID, nil
		}
	}
	return "", fmt.Errorf("interface %q introuvable", name)
}

func runAll(commands [][]string) error {
	for _, args := range commands {
		if err := runNetsh(args); err != nil {
			return err
		}
	}
	return nil
}

func runNetsh(args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), netshTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "netsh", args...)
	// Without this a console window flashes on screen at every command.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh %s : %w : %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
