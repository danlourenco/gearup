// Package log owns per-run log files. Each gearup run opens a timestamped
// file under $XDG_STATE_HOME/gearup/logs/ (falling back to
// ~/.local/state/gearup/logs/) and writes a short header. Installers and
// the runner mirror command output into this file for audit + debugging.
package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File wraps an open log file handle.
type File struct {
	f    *os.File
	path string
}

// Create opens a new per-run log file named after recipe + current timestamp,
// writes a header identifying the recipe and run time, and returns the handle.
func Create(recipeName string) (*File, error) {
	dir, err := logDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	safe := sanitize(recipeName)
	name := fmt.Sprintf("%s-%s.log", time.Now().Format("20060102-150405"), safe)
	path := filepath.Join(dir, name)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	fmt.Fprintf(f, "# gearup run log\n# recipe: %s\n# started: %s\n\n", recipeName, time.Now().Format(time.RFC3339))
	return &File{f: f, path: path}, nil
}

// Writer returns the underlying io.Writer for mirroring command output.
func (l *File) Writer() io.Writer { return l.f }

// Path returns the absolute path of the log file.
func (l *File) Path() string { return l.path }

// Close flushes and closes the log file.
func (l *File) Close() error { return l.f.Close() }

func logDir() (string, error) {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "gearup", "logs"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "gearup", "logs"), nil
}

// sanitize replaces filesystem-unfriendly characters in the recipe name so it
// can appear in a filename. It is intentionally conservative.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unnamed"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
