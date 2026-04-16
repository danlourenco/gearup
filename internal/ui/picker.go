// Package ui provides interactive terminal elements backed by the Charm
// ecosystem: recipe picker (Huh), plan preview (Lip Gloss), and execution
// step printer.
package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"
)

// RecipeEntry is a discovered recipe file with its parsed metadata.
type RecipeEntry struct {
	Path        string
	Name        string
	Description string
}

// DiscoverRecipes scans dirs for *.yaml files, parses their name and
// description fields, and returns a deduplicated list. Missing directories
// are silently skipped.
func DiscoverRecipes(dirs []string) ([]RecipeEntry, error) {
	seen := map[string]bool{}
	var entries []RecipeEntry

	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			return nil, err
		}
		for _, path := range matches {
			abs, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			if seen[abs] {
				continue
			}
			seen[abs] = true
			entry, err := parseRecipeEntry(abs)
			if err != nil {
				continue
			}
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func parseRecipeEntry(path string) (RecipeEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RecipeEntry{}, err
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return RecipeEntry{}, err
	}
	if meta.Name == "" {
		meta.Name = filepath.Base(path)
	}
	return RecipeEntry{Path: path, Name: meta.Name, Description: meta.Description}, nil
}

// PickRecipe shows an interactive Huh select for the given entries.
// If exactly one entry exists, it is returned without prompting.
// Returns the selected entry or an error if the user aborts.
func PickRecipe(entries []RecipeEntry) (RecipeEntry, error) {
	if len(entries) == 0 {
		return RecipeEntry{}, fmt.Errorf("no recipes found; pass --recipe <path> explicitly")
	}
	if len(entries) == 1 {
		return entries[0], nil
	}

	options := make([]huh.Option[int], len(entries))
	for i, e := range entries {
		label := e.Name
		if e.Description != "" {
			label = fmt.Sprintf("%s — %s", e.Name, e.Description)
		}
		options[i] = huh.NewOption(label, i)
	}

	var selected int
	err := huh.NewSelect[int]().
		Title("Which recipe?").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return RecipeEntry{}, fmt.Errorf("recipe picker: %w", err)
	}
	return entries[selected], nil
}
