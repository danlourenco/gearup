// Package gitclone implements the "git-clone" step type.
package gitclone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
)

// Installer is the git-clone step installer.
type Installer struct {
	Runner gearexec.Runner
}

// Check reports whether the destination directory already exists.
func (i *Installer) Check(_ context.Context, step config.Step) (bool, error) {
	if step.Dest == "" {
		return false, fmt.Errorf("git-clone step %q missing dest", step.Name)
	}
	dest := expandHome(step.Dest)
	info, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// Install runs `git clone [--branch <ref>] <repo> <dest>`.
func (i *Installer) Install(ctx context.Context, step config.Step) error {
	if step.Repo == "" {
		return fmt.Errorf("git-clone step %q missing repo", step.Name)
	}
	if step.Dest == "" {
		return fmt.Errorf("git-clone step %q missing dest", step.Name)
	}
	dest := expandHome(step.Dest)

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("git-clone %s: create parent dir: %w", step.Name, err)
	}

	cmd := fmt.Sprintf("git clone %s %s", step.Repo, dest)
	if step.Ref != "" {
		cmd = fmt.Sprintf("git clone --branch %s %s %s", step.Ref, step.Repo, dest)
	}
	res, err := i.Runner.RunQuiet(ctx, cmd)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("git clone %s failed (exit %d): %s", step.Name, res.ExitCode, res.Stderr)
	}
	return nil
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
