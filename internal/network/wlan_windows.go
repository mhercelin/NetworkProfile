//go:build windows

package network

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Windows keeps one XML document per known wireless network, filed under the
// adapter that learned it. Reading them is independent of the display language,
// unlike the output of "netsh wlan show profiles"; the folder is readable
// because the application runs elevated.
func wlanProfileRoot() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "Microsoft", "Wlansvc", "Profiles", "Interfaces")
}

// knownWiFiNetworks lists the networks Windows already holds credentials for on
// the given adapter. An adapter with no folder has simply never joined one.
func knownWiFiNetworks(adapterID string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(wlanProfileRoot(), adapterID))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		doc, err := os.ReadFile(filepath.Join(wlanProfileRoot(), adapterID, entry.Name()))
		if err != nil {
			continue
		}
		name, err := profileNameFromXML(doc)
		if err != nil {
			continue
		}
		names = append(names, name)
	}

	slices.Sort(names)
	return names, nil
}
