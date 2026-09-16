//go:build windows

package ipc

import (
	"strings"
	"testing"
)

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
