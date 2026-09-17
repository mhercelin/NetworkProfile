//go:build windows

package ipc

import (
	"strings"
	"testing"
	"time"

	"github.com/Microsoft/go-winio"
)

// The helper only publishes its pipe once the operator has answered the
// elevation prompt and the process has started, so the first attempts always
// find nothing. winio.DialPipe does not wait for that on its own.
func TestDialWhenPublishedWaitsForALateHelper(t *testing.T) {
	name, err := randomPipeName()
	if err != nil {
		t.Fatal(err)
	}
	path := pipePath(name)

	listening := make(chan struct{})
	go func() {
		time.Sleep(400 * time.Millisecond)
		listener, err := winio.ListenPipe(path, nil)
		if err != nil {
			close(listening)
			return
		}
		defer listener.Close()
		close(listening)

		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	conn, err := dialWhenPublished(path, time.Now().Add(10*time.Second))
	if err != nil {
		t.Fatalf("expected the late pipe to be picked up, got: %v", err)
	}
	conn.Close()
	<-listening
}

func TestDialWhenPublishedGivesUpOnTheDeadline(t *testing.T) {
	name, err := randomPipeName()
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	_, err = dialWhenPublished(pipePath(name), start.Add(300*time.Millisecond))

	if err == nil {
		t.Fatal("expected a pipe nobody ever creates to fail")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("gave up after %s, far past its deadline", elapsed)
	}
}

// The helper's pipe is locked to this SID, so getting it wrong locks the
// interface out of its own helper — which is exactly what happened when the
// token handle was passed as a bare zero.
func TestCurrentUserSID(t *testing.T) {
	sid, err := currentUserSID()
	if err != nil {
		t.Fatalf("currentUserSID: %v", err)
	}

	if !strings.HasPrefix(sid, "S-1-") {
		t.Fatalf("expected a SID in string form, got %q", sid)
	}
}

func TestParseHelperArgs(t *testing.T) {
	tests := []struct {
		name    string
		argv    []string
		wantOK  bool
		wantSID string
	}{
		{
			name:    "full helper invocation",
			argv:    []string{"--helper", "--pipe", "networkprofile-abc", "--owner", "S-1-5-21-1", "--parent", "4242"},
			wantOK:  true,
			wantSID: "S-1-5-21-1",
		},
		{"plain launch", []string{}, false, ""},
		{"flag without its parameters", []string{"--helper"}, false, ""},
		{"missing parent", []string{"--helper", "--pipe", "p", "--owner", "S-1-5-21-1"}, false, ""},
		{"trailing flag with no value", []string{"--helper", "--pipe"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, ok := ParseHelperArgs(tt.argv)

			if ok != tt.wantOK {
				t.Fatalf("expected helper=%t, got %t", tt.wantOK, ok)
			}
			if ok && args.Owner != tt.wantSID {
				t.Fatalf("expected owner %q, got %q", tt.wantSID, args.Owner)
			}
		})
	}
}
