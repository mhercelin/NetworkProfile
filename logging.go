package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"networkprofile/internal/gui"
)

// logPath is where both halves write. The elevated helper runs as another
// account entirely, so it is told the path rather than deriving one — otherwise
// its side of a failure would land in the administrator's profile, where nobody
// would think to look.
func logPath() (string, error) {
	dir, err := gui.DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "networkprofile.log"), nil
}

// startLogging sends the standard logger to a file. A window has no console, so
// without this every diagnostic the application produces is lost — which is
// exactly the situation an operator is in when a profile silently fails to
// apply.
func startLogging(path, prefix string) io.Closer {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		// Nowhere to report this to; carry on rather than refuse to start.
		return io.NopCloser(nil)
	}

	log.SetOutput(file)
	log.SetPrefix(prefix + " ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	return file
}

// trimLog keeps the file from growing without end. It is read by a person
// chasing a failure that just happened, so only the recent past matters.
func trimLog(path string) {
	const maxBytes = 512 * 1024

	info, err := os.Stat(path)
	if err != nil || info.Size() <= maxBytes {
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Drop the older half, then the partial line it starts on.
	kept := content[len(content)/2:]
	if cut := indexNewline(kept); cut >= 0 {
		kept = kept[cut+1:]
	}
	_ = os.WriteFile(path, kept, 0o644)
}

func indexNewline(b []byte) int {
	for i, c := range b {
		if c == '\n' {
			return i
		}
	}
	return -1
}
