// Command gearup is an open-source macOS developer-machine bootstrap CLI.
// Phase 1: runs a single profile with brew steps only.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	gearexec "gearup/internal/exec"
	"gearup/internal/installer"
	"gearup/internal/installer/brew"
	"gearup/internal/profile"
	"gearup/internal/runner"
)

const version = "0.0.1-phase1"

func main() {
	root := &cobra.Command{
		Use:   "gearup",
		Short: "Open-source macOS developer-machine bootstrap CLI",
	}
	root.AddCommand(runCmd(), versionCmd())
	if err := root.Execute(); err != nil {
		// cobra already printed the error
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
	var profilePath string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a provisioning profile",
		RunE: func(c *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				fmt.Fprintln(os.Stderr, "gearup currently supports macOS only")
				os.Exit(4)
			}
			if profilePath == "" {
				return fmt.Errorf("--profile is required (Phase 1)")
			}
			absProfile, err := filepath.Abs(profilePath)
			if err != nil {
				return err
			}
			p, err := profile.LoadProfile(absProfile)
			if err != nil {
				return err
			}
			plan, err := profile.Resolve(p, filepath.Dir(absProfile))
			if err != nil {
				return err
			}

			shell := &gearexec.ShellRunner{Stdout: os.Stdout, Stderr: os.Stderr}
			reg := installer.Registry{
				"brew": &brew.Installer{Runner: shell},
			}
			r := &runner.Runner{Registry: reg, Out: stdPrinter{}}

			fmt.Printf("PROFILE: %s  (%d steps)\n\n", p.Name, len(plan.Steps))
			if err := r.Run(context.Background(), plan); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			fmt.Println("\nDone.")
			return nil
		},
	}
	cmd.Flags().StringVar(&profilePath, "profile", "", "path to profile YAML (required)")
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print gearup version",
		Run: func(*cobra.Command, []string) {
			fmt.Printf("gearup %s\n", version)
		},
	}
}

// stdPrinter adapts os.Stdout to runner.Writer.
type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
