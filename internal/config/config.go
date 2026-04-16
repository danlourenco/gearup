// Package config defines types for gearup configuration files and provides
// loading and resolution. Every gearup YAML file has the same schema:
// a Config can declare steps directly and/or extend other configs by name.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is a gearup configuration file. It can serve as both an entry point
// (with platform/elevation) and a reusable building block (extended by other
// configs). There is no distinction between the two — every file has the
// same shape.
type Config struct {
	Version     int        `yaml:"version"`
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	Platform    Platform   `yaml:"platform,omitempty"`
	Elevation   *Elevation `yaml:"elevation,omitempty"`
	Sources     []Source   `yaml:"sources,omitempty"`
	Extends     []string   `yaml:"extends,omitempty"`
	Steps       []Step     `yaml:"steps,omitempty"`
}

// Platform constrains which OS/arch a config applies to.
type Platform struct {
	OS   []string `yaml:"os,omitempty"`
	Arch []string `yaml:"arch,omitempty"`
}

// Source declares a directory where extended configs can be resolved from.
type Source struct {
	Path string `yaml:"path,omitempty"`
}

// Elevation describes a config-level request for admin permissions.
type Elevation struct {
	Message  string        `yaml:"message"`
	Duration time.Duration `yaml:"duration,omitempty"`
}

// Step is one unit of provisioning work.
type Step struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"`
	Check             string   `yaml:"check,omitempty"`
	RequiresElevation bool     `yaml:"requires_elevation,omitempty"`
	Platform          Platform `yaml:"platform,omitempty"`

	// brew
	Formula string `yaml:"formula,omitempty"`

	// curl-pipe-sh
	URL   string   `yaml:"url,omitempty"`
	Shell string   `yaml:"shell,omitempty"`
	Args  []string `yaml:"args,omitempty"`

	// shell
	Install string `yaml:"install,omitempty"`

	// post_install — shell commands to run after a successful install
	PostInstall []string `yaml:"post_install,omitempty"`

	// brew-cask
	Cask string `yaml:"cask,omitempty"`

	// git-clone
	Repo string `yaml:"repo,omitempty"`
	Dest string `yaml:"dest,omitempty"`
	Ref  string `yaml:"ref,omitempty"`
}

// ResolvedPlan is a flattened, ordered list of steps produced by resolving
// a config's extends references.
type ResolvedPlan struct {
	Config *Config
	Steps  []Step
}

// Load reads and parses a config YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &c, nil
}

// Resolve walks c.Extends recursively and flattens into a ResolvedPlan.
// configDir is the directory of the config file; it is always prepended
// to the search path so configs in the same directory can extend each
// other without declaring sources.
func Resolve(c *Config, configDir string) (*ResolvedPlan, error) {
	seen := map[string]bool{}
	steps, err := resolveRecursive(c, configDir, seen)
	if err != nil {
		return nil, err
	}
	return &ResolvedPlan{Config: c, Steps: steps}, nil
}

func resolveRecursive(c *Config, configDir string, seen map[string]bool) ([]Step, error) {
	searchPath := buildSearchPath(c, configDir)
	var steps []Step

	for _, name := range c.Extends {
		extPath, err := findConfig(name, searchPath)
		if err != nil {
			return nil, err
		}
		abs, err := filepath.Abs(extPath)
		if err != nil {
			return nil, err
		}
		if seen[abs] {
			return nil, fmt.Errorf("circular extends detected: %s", name)
		}
		seen[abs] = true

		ext, err := Load(extPath)
		if err != nil {
			return nil, err
		}
		extDir := filepath.Dir(extPath)
		extSteps, err := resolveRecursive(ext, extDir, seen)
		if err != nil {
			return nil, err
		}
		steps = append(steps, extSteps...)
	}

	steps = append(steps, c.Steps...)
	return dedup(steps), nil
}

func buildSearchPath(c *Config, configDir string) []string {
	// The config file's own directory is always first in the search path.
	paths := []string{configDir}
	for _, src := range c.Sources {
		if src.Path == "" {
			continue
		}
		expanded := expandHome(src.Path)
		if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(configDir, expanded)
		}
		paths = append(paths, expanded)
	}
	return paths
}

func findConfig(name string, searchPath []string) (string, error) {
	for _, dir := range searchPath {
		candidate := filepath.Join(dir, name+".yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("config %q not found in any source", name)
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

// dedup removes steps with duplicate names, keeping the first occurrence.
func dedup(steps []Step) []Step {
	seen := map[string]bool{}
	var out []Step
	for _, s := range steps {
		if seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		out = append(out, s)
	}
	return out
}
