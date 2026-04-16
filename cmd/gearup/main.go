// Command gearup is an open-source macOS developer-machine bootstrap CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"gearup/configs"
	"gearup/internal/config"
	"gearup/internal/elevation"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer"
	"gearup/internal/installer/brew"
	"gearup/internal/installer/brewcask"
	"gearup/internal/installer/curlpipe"
	"gearup/internal/installer/gitclone"
	installshell "gearup/internal/installer/shell"
	gearlog "gearup/internal/log"
	"gearup/internal/runner"
	"gearup/internal/ui"
)

// version is set by goreleaser via ldflags at build time.
// Falls back to "dev" for local builds.
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "gearup",
		Short: "Open-source macOS developer-machine bootstrap CLI",
	}
	root.AddCommand(runCmd(), planCmd(), initCmd(), versionCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
	var configPath string
	var dryRun, yes bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a provisioning config",
		RunE: func(c *cobra.Command, args []string) error {
			return execute(configPath, dryRun, yes)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to config YAML (omit to pick interactively)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve checks without installing; exit 10 if anything would run")
	cmd.Flags().BoolVar(&yes, "yes", false, "auto-approve elevation confirmations (for scripted use)")
	return cmd
}

func planCmd() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Alias for `run --dry-run`",
		RunE: func(c *cobra.Command, args []string) error {
			return execute(configPath, true, true)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to config YAML (omit to pick interactively)")
	return cmd
}

func execute(configPath string, dryRun, yes bool) error {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "gearup currently supports macOS only")
		os.Exit(4)
	}

	// If no config specified, try to discover and pick one interactively.
	if configPath == "" {
		if !isTerminal(os.Stdin) {
			fmt.Fprintln(os.Stderr, "no --config specified and stdin is not a terminal; cannot show picker")
			os.Exit(3)
		}
		picked, err := discoverAndPick()
		if err != nil {
			return err
		}
		configPath = picked
	}

	// TTY guard: interactive runs (non-dry-run, non-yes) require a terminal.
	if !dryRun && !yes && !isTerminal(os.Stdin) {
		fmt.Fprintln(os.Stderr, "gearup requires an interactive terminal. Use --yes to bypass elevation prompts, or --dry-run to preview.")
		os.Exit(3)
	}

	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}
	rec, err := config.Load(absConfig)
	if err != nil {
		return err
	}
	plan, err := config.Resolve(rec, filepath.Dir(absConfig))
	if err != nil {
		return err
	}

	// Open a per-run log file.
	lf, err := gearlog.Create(rec.Name)
	if err != nil {
		return err
	}
	defer lf.Close()

	shellRunner := &gearexec.ShellRunner{
		StreamOut: os.Stdout,
		StreamErr: os.Stderr,
		LogOut:    lf.Writer(),
	}

	reg := installer.Registry{
		"brew":         &brew.Installer{Runner: shellRunner},
		"brew-cask":    &brewcask.Installer{Runner: shellRunner},
		"curl-pipe-sh": &curlpipe.Installer{Runner: shellRunner},
		"git-clone":    &gitclone.Installer{Runner: shellRunner},
		"shell":        &installshell.Installer{Runner: shellRunner},
	}

	var prompter elevation.Prompter = elevation.HuhPrompter{}
	if yes {
		prompter = elevation.AutoApprovePrompter{}
	}

	printer := ui.NewStepPrinter(os.Stdout)

	r := &runner.Runner{
		Registry:   reg,
		Out:        stdPrinter{},
		Prompter:   prompter,
		Printer:    printer,
		ExecRunner: shellRunner,
		DryRun:     dryRun,
	}

	header := "CONFIG"
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
		fmt.Fprintln(os.Stderr, "\nerror:", err)
		fmt.Fprintln(os.Stderr, "full log:", lf.Path())
		os.Exit(1)
	}
	if !dryRun {
		fmt.Println("\nDone.")
		fmt.Printf("Log: %s\n", lf.Path())
	}
	return nil
}

// discoverAndPick scans well-known directories for config files and
// prompts the user to select one via Huh.
func discoverAndPick() (string, error) {
	dirs := configSearchDirs()

	entries, err := ui.DiscoverConfigs(dirs)
	if err != nil {
		return "", err
	}

	// No configs found — auto-extract embedded defaults.
	if len(entries) == 0 {
		fmt.Println("No configs found. Writing defaults...")
		dir, err := initDefaults(false)
		if err != nil {
			return "", fmt.Errorf("auto-init: %w", err)
		}
		// Re-scan with the new directory included.
		dirs = append(dirs, dir)
		entries, err = ui.DiscoverConfigs(dirs)
		if err != nil {
			return "", err
		}
	}

	picked, err := ui.PickConfig(entries)
	if err != nil {
		return "", err
	}
	return picked.Path, nil
}

func configSearchDirs() []string {
	var dirs []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "gearup", "configs"))
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "gearup", "configs"))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "configs"))
	}
	return dirs
}

// initDefaults writes the embedded default configs to the user's config
// directory. Returns the directory path. Existing files are NOT overwritten
// unless force is true.
func initDefaults(force bool) (string, error) {
	configDir := defaultConfigDir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	entries, err := fs.ReadDir(configs.Defaults, ".")
	if err != nil {
		return "", err
	}
	written := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		dest := filepath.Join(configDir, e.Name())
		if !force {
			if _, err := os.Stat(dest); err == nil {
				continue // don't overwrite existing
			}
		}
		data, err := configs.Defaults.ReadFile(e.Name())
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return "", err
		}
		written++
	}
	if written > 0 {
		fmt.Printf("Wrote %d default config(s) to %s\n", written, configDir)
	}
	return configDir, nil
}

func defaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gearup", "configs")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gearup", "configs")
}

func initCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write default configs to ~/.config/gearup/configs/",
		RunE: func(c *cobra.Command, args []string) error {
			dir, err := initDefaults(force)
			if err != nil {
				return err
			}
			fmt.Printf("Configs ready at %s\n", dir)
			fmt.Println("Run `gearup run` to pick one.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config files")
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

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
func (stdPrinter) Write(p []byte) (int, error)       { return os.Stdout.Write(p) }
