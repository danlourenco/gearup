// Package ui provides interactive terminal elements backed by the Charm
// ecosystem: config picker (Huh), plan preview (Lip Gloss), and execution
// step printer.
package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"
)

// ConfigEntry is a discovered config file with its parsed metadata.
type ConfigEntry struct {
	Path        string
	Name        string
	Description string
}

// DiscoverConfigs scans dirs for *.yaml files, parses their name and
// description fields, and returns a deduplicated list. Missing directories
// are silently skipped.
func DiscoverConfigs(dirs []string) ([]ConfigEntry, error) {
	seen := map[string]bool{}
	var entries []ConfigEntry

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
			entry, err := parseConfigEntry(abs)
			if err != nil {
				continue
			}
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func parseConfigEntry(path string) (ConfigEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ConfigEntry{}, err
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return ConfigEntry{}, err
	}
	if meta.Name == "" {
		meta.Name = filepath.Base(path)
	}
	return ConfigEntry{Path: path, Name: meta.Name, Description: meta.Description}, nil
}

// PickConfig shows an interactive Huh select for the given entries.
// If exactly one entry exists, it is returned without prompting.
// Returns the selected entry or an error if the user aborts.
func PickConfig(entries []ConfigEntry) (ConfigEntry, error) {
	if len(entries) == 0 {
		return ConfigEntry{}, fmt.Errorf("no configs found; pass --config <path> explicitly")
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
		Title("Which config?").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return ConfigEntry{}, fmt.Errorf("config picker: %w", err)
	}
	return entries[selected], nil
}
