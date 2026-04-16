// Command gearup is an open-source macOS developer-machine bootstrap CLI.
// Phase 1: runs a single recipe with brew steps only.
package main

import (
	"context"
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
	"gearup/internal/installer/shell"
	"gearup/internal/recipe"
	"gearup/internal/runner"
)

const version = "0.0.4-phase3"

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
	var recipePath string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a provisioning recipe",
		RunE: func(c *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				fmt.Fprintln(os.Stderr, "gearup currently supports macOS only")
				os.Exit(4)
			}
			if recipePath == "" {
				return fmt.Errorf("--recipe is required")
			}
			absRecipe, err := filepath.Abs(recipePath)
			if err != nil {
				return err
			}
			r, err := recipe.LoadRecipe(absRecipe)
			if err != nil {
				return err
			}
			plan, err := recipe.Resolve(r, filepath.Dir(absRecipe))
			if err != nil {
				return err
			}

			shellRunner := &gearexec.ShellRunner{Stdout: os.Stdout, Stderr: os.Stderr}
			reg := installer.Registry{
				"brew":         &brew.Installer{Runner: shellRunner},
				"curl-pipe-sh": &curlpipe.Installer{Runner: shellRunner},
				"shell":        &shell.Installer{Runner: shellRunner},
			}
			runner := &runner.Runner{
				Registry: reg,
				Out:      stdPrinter{},
				Prompter: elevation.HuhPrompter{},
			}

			fmt.Printf("RECIPE: %s  (%d steps)\n\n", r.Name, len(plan.Steps))
			if err := runner.Run(context.Background(), plan); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			fmt.Println("\nDone.")
			return nil
		},
	}
	cmd.Flags().StringVar(&recipePath, "recipe", "", "path to recipe YAML (required)")
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
func (stdPrinter) Write(p []byte) (int, error)       { return os.Stdout.Write(p) }
