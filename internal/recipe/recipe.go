// Package recipe loads recipe and ingredient YAML files and flattens them
// into a ResolvedPlan. Phase 1: local path sources only, no transitive
// requires, no deduplication.
package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"gearup/internal/config"
)

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

// LoadIngredient reads and parses an ingredient YAML file.
func LoadIngredient(path string) (*config.Ingredient, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ingredient %s: %w", path, err)
	}
	var ing config.Ingredient
	if err := yaml.Unmarshal(data, &ing); err != nil {
		return nil, fmt.Errorf("parse ingredient %s: %w", path, err)
	}
	return &ing, nil
}

// Resolve walks recipe.Ingredients and inline Steps, producing a ResolvedPlan.
//
// recipeDir is the directory of the recipe file; relative ingredient_source
// paths are resolved against it.
func Resolve(r *config.Recipe, recipeDir string) (*config.ResolvedPlan, error) {
	searchPath := buildSearchPath(r, recipeDir)

	var steps []config.Step
	for _, name := range r.Ingredients {
		ingPath, err := findIngredient(name, searchPath)
		if err != nil {
			return nil, err
		}
		ing, err := LoadIngredient(ingPath)
		if err != nil {
			return nil, err
		}
		steps = append(steps, ing.Steps...)
	}
	steps = append(steps, r.Steps...)
	return &config.ResolvedPlan{Recipe: r, Steps: steps}, nil
}

func buildSearchPath(r *config.Recipe, recipeDir string) []string {
	var paths []string
	for _, src := range r.IngredientSources {
		if src.Path == "" {
			continue
		}
		expanded := expandHome(src.Path)
		if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(recipeDir, expanded)
		}
		paths = append(paths, expanded)
	}
	return paths
}

func findIngredient(name string, searchPath []string) (string, error) {
	for _, dir := range searchPath {
		candidate := filepath.Join(dir, name+".yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("ingredient %q not found in any ingredient source", name)
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
