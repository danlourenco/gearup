// Package profile loads profile and recipe YAML files and flattens them
// into a ResolvedPlan. Phase 1: local path sources only, no transitive
// requires, no deduplication.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"gearup/internal/config"
)

// LoadProfile reads and parses a profile YAML file.
func LoadProfile(path string) (*config.Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile %s: %w", path, err)
	}
	var p config.Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", path, err)
	}
	return &p, nil
}

// LoadRecipe reads and parses a recipe YAML file.
func LoadRecipe(path string) (*config.Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read recipe %s: %w", path, err)
	}
	var r config.Recipe
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse recipe %s: %w", path, err)
	}
	return &r, nil
}

// Resolve walks profile.Includes and inline Steps, producing a ResolvedPlan.
//
// profileDir is the directory of the profile file; relative recipe_source
// paths are resolved against it.
func Resolve(p *config.Profile, profileDir string) (*config.ResolvedPlan, error) {
	searchPath := buildSearchPath(p, profileDir)

	var steps []config.Step
	for _, inc := range p.Includes {
		recipePath, err := findRecipe(inc.Recipe, searchPath)
		if err != nil {
			return nil, err
		}
		r, err := LoadRecipe(recipePath)
		if err != nil {
			return nil, err
		}
		steps = append(steps, r.Steps...)
	}
	steps = append(steps, p.Steps...)
	return &config.ResolvedPlan{Profile: p, Steps: steps}, nil
}

func buildSearchPath(p *config.Profile, profileDir string) []string {
	var paths []string
	for _, src := range p.RecipeSources {
		if src.Path == "" {
			continue
		}
		expanded := expandHome(src.Path)
		if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(profileDir, expanded)
		}
		paths = append(paths, expanded)
	}
	return paths
}

func findRecipe(name string, searchPath []string) (string, error) {
	for _, dir := range searchPath {
		candidate := filepath.Join(dir, name+".yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("recipe %q not found in any recipe source", name)
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
