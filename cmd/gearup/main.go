// Command gearup is an open-source macOS developer-machine bootstrap CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"gearup/internal/elevation"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer"
	"gearup/internal/installer/brew"
	"gearup/internal/installer/curlpipe"
	installshell "gearup/internal/installer/shell"
	gearlog "gearup/internal/log"
	"gearup/internal/recipe"
	"gearup/internal/runner"
)

const version = "0.0.5-phase4a"

func main() {
	root := &cobra.Command{
		Use:   "gearup",
		Short: "Open-source macOS developer-machine bootstrap CLI",
	}
	root.AddCommand(runCmd(), planCmd(), versionCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
	var recipePath string
	var dryRun, yes bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a provisioning recipe",
		RunE: func(c *cobra.Command, args []string) error {
			return execute(recipePath, dryRun, yes)
		},
	}
	cmd.Flags().StringVar(&recipePath, "recipe", "", "path to recipe YAML (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve checks without installing; exit 10 if anything would run")
	cmd.Flags().BoolVar(&yes, "yes", false, "auto-approve elevation confirmations (for scripted use)")
	return cmd
}

func planCmd() *cobra.Command {
	var recipePath string
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Alias for `run --dry-run`",
		RunE: func(c *cobra.Command, args []string) error {
			return execute(recipePath, true, true)
		},
	}
	cmd.Flags().StringVar(&recipePath, "recipe", "", "path to recipe YAML (required)")
	return cmd
}

func execute(recipePath string, dryRun, yes bool) error {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "gearup currently supports macOS only")
		os.Exit(4)
	}
	if recipePath == "" {
		return fmt.Errorf("--recipe is required")
	}

	// TTY guard: interactive runs (non-dry-run, non-yes) require a terminal
	// because the elevation confirm blocks on Huh.
	if !dryRun && !yes && !isTerminal(os.Stdin) {
		fmt.Fprintln(os.Stderr, "gearup requires an interactive terminal. Use --yes to bypass elevation prompts, or --dry-run to preview.")
		os.Exit(3)
	}

	absRecipe, err := filepath.Abs(recipePath)
	if err != nil {
		return err
	}
	rec, err := recipe.LoadRecipe(absRecipe)
	if err != nil {
		return err
	}
	plan, err := recipe.Resolve(rec, filepath.Dir(absRecipe))
	if err != nil {
		return err
	}

	// Open a per-run log file for command output mirroring.
	lf, err := gearlog.Create(rec.Name)
	if err != nil {
		return err
	}
	defer lf.Close()

	// Build the shared ShellRunner: stream to terminal, mirror to log.
	shellRunner := &gearexec.ShellRunner{
		StreamOut: os.Stdout,
		StreamErr: os.Stderr,
		LogOut:    lf.Writer(),
	}

	reg := installer.Registry{
		"brew":         &brew.Installer{Runner: shellRunner},
		"curl-pipe-sh": &curlpipe.Installer{Runner: shellRunner},
		"shell":        &installshell.Installer{Runner: shellRunner},
	}

	var prompter elevation.Prompter = elevation.HuhPrompter{}
	if yes {
		prompter = elevation.AutoApprovePrompter{}
	}

	r := &runner.Runner{
		Registry: reg,
		Out:      stdPrinter{},
		Prompter: prompter,
		DryRun:   dryRun,
	}

	header := "RECIPE"
	if dryRun {
		header = "PLAN (dry-run)"
	}
	fmt.Printf("%s: %s  (%d steps)\n\n", header, rec.Name, len(plan.Steps))

	err = r.Run(context.Background(), plan)
	if errors.Is(err, runner.ErrDryRunPending) {
		fmt.Fprintln(os.Stderr, "\nnote: re-run without --dry-run to apply the pending installs")
		os.Exit(10)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr, "full log:", lf.Path())
		os.Exit(1)
	}
	if !dryRun {
		fmt.Println("\nDone.")
	}
	return nil
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

// isTerminal reports whether f is a character device (interactive TTY).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// stdPrinter adapts os.Stdout to runner.Writer.
type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
func (stdPrinter) Write(p []byte) (int, error)       { return os.Stdout.Write(p) }
