package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// x/sys/windows wraps neither of these.
var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procAttachConsole      = kernel32.NewProc("AttachConsole")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
)

// codePageUTF8 is CP_UTF8. A console left on its default code page reads the
// UTF-8 this program writes as single bytes, so a profile named "Ethernet —
// Atelier" comes out as "Ethernet ÔÇö Atelier".
const codePageUTF8 = 65001

// attachConsole points standard output at the console the executable was
// launched from.
//
// The binary is linked as a GUI application so that double-clicking it does not
// flash a console window; the cost is that it starts with no standard output at
// all, and "--list" would print into the void. Launched from Explorer there is
// no parent console and nothing to attach to, which is why failing here is
// silent rather than fatal.
func attachConsole() {
	const attachParentProcess = ^uintptr(0) // (DWORD)-1

	// Output already redirected — piped into another command, or captured to a
	// file — works as it stands. Replacing it with the console would send the
	// answer somewhere nobody is reading.
	if handle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && handle != 0 && handle != windows.InvalidHandle {
		return
	}

	if ret, _, _ := procAttachConsole.Call(attachParentProcess); ret == 0 {
		return
	}

	console, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	os.Stdout = console
	os.Stderr = console

	// Profile names carry accents and em dashes; without this the console
	// renders the UTF-8 they are written in as one mangled character per byte.
	_, _, _ = procSetConsoleOutputCP.Call(codePageUTF8)
}
