package network

import (
	"encoding/xml"
	"fmt"
)

// wlanProfile is the part of a Windows WLAN profile document that matters here.
// The profile name is what "netsh wlan connect name=..." expects, and it is not
// always identical to the SSID — a profile can be renamed.
type wlanProfile struct {
	Name string `xml:"name"`
}

func profileNameFromXML(doc []byte) (string, error) {
	var profile wlanProfile
	if err := xml.Unmarshal(doc, &profile); err != nil {
		return "", err
	}
	if profile.Name == "" {
		return "", fmt.Errorf("WLAN profile carries no name")
	}
	return profile.Name, nil
}
