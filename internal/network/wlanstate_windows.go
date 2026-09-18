//go:build windows

package network

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The network an adapter is actually on is read through the WLAN API rather
// than from "netsh wlan show interfaces", whose output is translated into the
// language of the running Windows.
var (
	wlanapi                = windows.NewLazySystemDLL("wlanapi.dll")
	procWlanOpenHandle     = wlanapi.NewProc("WlanOpenHandle")
	procWlanCloseHandle    = wlanapi.NewProc("WlanCloseHandle")
	procWlanEnumInterfaces = wlanapi.NewProc("WlanEnumInterfaces")
	procWlanQueryInterface = wlanapi.NewProc("WlanQueryInterface")
	procWlanFreeMemory     = wlanapi.NewProc("WlanFreeMemory")
)

const (
	wlanCurrentConnection = 7 // wlan_intf_opcode_current_connection
	wlanConnected         = 1 // wlan_interface_state_connected
	dot11SSIDMaxLength    = 32
)

type wlanInterfaceInfo struct {
	InterfaceGUID        windows.GUID
	InterfaceDescription [256]uint16
	State                uint32
}

type wlanInterfaceInfoList struct {
	NumberOfItems uint32
	Index         uint32
	InterfaceInfo [1]wlanInterfaceInfo
}

type dot11SSID struct {
	Length uint32
	SSID   [dot11SSIDMaxLength]byte
}

type wlanAssociationAttributes struct {
	SSID          dot11SSID
	BSSType       uint32
	BSSID         [6]byte
	PhyType       uint32
	PhyIndex      uint32
	SignalQuality uint32
	RxRate        uint32
	TxRate        uint32
}

type wlanConnectionAttributes struct {
	State                 uint32
	Mode                  uint32
	ProfileName           [256]uint16
	AssociationAttributes wlanAssociationAttributes
	SecurityAttributes    [20]byte
}

// currentSSIDs maps an adapter GUID to the network it is connected to. An
// adapter that is not connected simply has no entry.
//
// Failures are silent on purpose: this is decoration on top of the adapter
// list, and a machine with no wireless hardware must not turn that into an
// error.
func currentSSIDs() map[string]string {
	var handle syscall.Handle
	var negotiated uint32

	if ret, _, _ := procWlanOpenHandle.Call(2, 0, uintptr(unsafe.Pointer(&negotiated)), uintptr(unsafe.Pointer(&handle))); ret != 0 {
		return nil
	}
	defer func() { _, _, _ = procWlanCloseHandle.Call(uintptr(handle), 0) }()

	var list *wlanInterfaceInfoList
	if ret, _, _ := procWlanEnumInterfaces.Call(uintptr(handle), 0, uintptr(unsafe.Pointer(&list))); ret != 0 {
		return nil
	}
	defer func() { _, _, _ = procWlanFreeMemory.Call(uintptr(unsafe.Pointer(list))) }()

	found := make(map[string]string)
	interfaces := unsafe.Slice(&list.InterfaceInfo[0], list.NumberOfItems)

	for i := range interfaces {
		info := &interfaces[i]
		if info.State != wlanConnected {
			continue
		}
		if ssid := connectedSSID(handle, &info.InterfaceGUID); ssid != "" {
			found[strings.ToLower(guidString(&info.InterfaceGUID))] = ssid
		}
	}
	return found
}

func connectedSSID(handle syscall.Handle, guid *windows.GUID) string {
	var attributes *wlanConnectionAttributes
	var size uint32

	ret, _, _ := procWlanQueryInterface.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(guid)),
		wlanCurrentConnection,
		0,
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Pointer(&attributes)),
		0,
	)
	if ret != 0 || attributes == nil {
		return ""
	}
	defer func() { _, _, _ = procWlanFreeMemory.Call(uintptr(unsafe.Pointer(attributes))) }()

	ssid := attributes.AssociationAttributes.SSID
	if ssid.Length == 0 || ssid.Length > dot11SSIDMaxLength {
		return ""
	}
	// An SSID is bytes, not text: Windows hands it over without a code page.
	// UTF-8 is what anything modern uses, and what our profiles store.
	return string(ssid.SSID[:ssid.Length])
}

// guidString renders a GUID the way GetAdaptersAddresses reports adapter names,
// so the two can be matched.
func guidString(guid *windows.GUID) string {
	return guid.String()
}
