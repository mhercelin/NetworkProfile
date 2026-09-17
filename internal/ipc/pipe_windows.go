//go:build windows

package ipc

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"

	"networkprofile/internal/network"
)

// How long to wait for the helper to publish its pipe. The elevation prompt
// itself is already answered by the time this starts counting, since
// ShellExecute blocks until the operator has dealt with it.
const helperStartTimeout = 30 * time.Second

// ErrElevationRefused reports that the elevation prompt was dismissed. It is
// an ordinary outcome, not a failure to diagnose.
var ErrElevationRefused = errors.New("élévation refusée : un changement de configuration réseau demande les droits administrateur")

func pipePath(name string) string {
	return `\\.\pipe\` + name
}

// randomPipeName keeps the name unguessable, so no other account on the machine
// can be listening under it by the time the helper tries to create it.
func randomPipeName() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "networkprofile-" + hex.EncodeToString(raw[:]), nil
}

func currentUserSID() (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("lecture du compte courant : %w", err)
	}
	return user.User.Sid.String(), nil
}

// StartHelper elevates a second copy of this executable and connects to it.
// This is where the credentials prompt appears — once, and only when a change
// is actually being applied.
func StartHelper() (*Client, error) {
	name, err := randomPipeName()
	if err != nil {
		return nil, err
	}
	sid, err := currentUserSID()
	if err != nil {
		return nil, err
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}

	args := fmt.Sprintf("--helper --pipe %s --owner %s --parent %d", name, sid, os.Getpid())
	if err := elevate(exe, args); err != nil {
		return nil, err
	}

	conn, err := dialWhenPublished(pipePath(name), time.Now().Add(helperStartTimeout))
	if err != nil {
		return nil, fmt.Errorf("connexion à l'assistant élevé : %w", err)
	}
	return NewClient(conn), nil
}

// dialWhenPublished waits for the helper to create its pipe.
//
// winio.DialPipe's own timeout only covers a pipe that exists but has no free
// instance: when the pipe is simply not there yet it fails on the spot. The
// helper is a process that has just been launched through an elevation prompt,
// so "not there yet" is the normal first answer, and the waiting has to happen
// here.
func dialWhenPublished(path string, deadline time.Time) (net.Conn, error) {
	const attempt = 200 * time.Millisecond

	for {
		timeout := attempt
		conn, err := winio.DialPipe(path, &timeout)
		if err == nil {
			return conn, nil
		}
		// Anything other than "not created yet" is a real failure — a refused
		// connection must not be retried for half a minute.
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("l'assistant élevé n'a pas répondu en %s", helperStartTimeout)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func elevate(exe, args string) error {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	argv, err := windows.UTF16PtrFromString(args)
	if err != nil {
		return err
	}

	err = windows.ShellExecute(0, verb, file, argv, nil, windows.SW_HIDE)
	if errors.Is(err, windows.ERROR_CANCELLED) {
		return ErrElevationRefused
	}
	if err != nil {
		return fmt.Errorf("démarrage de l'assistant élevé : %w", err)
	}
	return nil
}

// ServeHelper is the elevated half. It publishes a pipe only the account that
// started it can open, and stops when that account's interface goes away.
func ServeHelper(name, ownerSID string, parentPID int, manager network.Manager) error {
	// Protected DACL, no inheritance: SYSTEM, the administrators group, and the
	// single account that asked for this helper.
	sddl := fmt.Sprintf("D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;%s)", ownerSID)

	listener, err := winio.ListenPipe(pipePath(name), &winio.PipeConfig{SecurityDescriptor: sddl})
	if err != nil {
		return fmt.Errorf("publication du tube %s : %w", name, err)
	}
	defer func() { _ = listener.Close() }()

	// Without this an elevated process would outlive the window that spawned it.
	go exitWithParent(parentPID)

	server := NewServer(manager)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		err = server.Serve(conn)
		_ = conn.Close()

		if errors.Is(err, ErrShutdown) {
			return nil
		}
		// Any other end of connection just means the interface dropped it; wait
		// for it to come back rather than forcing a second credentials prompt.
	}
}

func exitWithParent(pid int) {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)

	_, _ = windows.WaitForSingleObject(handle, windows.INFINITE)
	os.Exit(0)
}

// HelperArgs recognises the elevated invocation of this executable.
type HelperArgs struct {
	Pipe   string
	Owner  string
	Parent int
}

// ParseHelperArgs reports whether this process was started as the helper, and
// with which parameters.
func ParseHelperArgs(argv []string) (HelperArgs, bool) {
	var args HelperArgs
	helper := false

	for i := 0; i < len(argv); i++ {
		switch argv[i] {
		case "--helper":
			helper = true
		case "--pipe":
			if i+1 < len(argv) {
				i++
				args.Pipe = argv[i]
			}
		case "--owner":
			if i+1 < len(argv) {
				i++
				args.Owner = argv[i]
			}
		case "--parent":
			if i+1 < len(argv) {
				i++
				args.Parent, _ = strconv.Atoi(argv[i])
			}
		}
	}

	return args, helper && args.Pipe != "" && args.Owner != "" && args.Parent > 0
}
