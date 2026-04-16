package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"gearup/internal/ui"
)

func writeConfig(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestDiscoverConfigs_FindsYAML(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "backend.yaml", `
version: 1
name: Backend
description: "Backend toolchain"
`)
	writeConfig(t, dir, "frontend.yaml", `
version: 1
name: Frontend
description: "Frontend toolchain"
`)
	writeConfig(t, dir, "notes.txt", "not a config")

	configs, err := ui.DiscoverConfigs([]string{dir})
	if err != nil {
		t.Fatalf("DiscoverConfigs: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("got %d configs, want 2", len(configs))
	}
}

func TestDiscoverConfigs_EmptyDirs(t *testing.T) {
	dir := t.TempDir()
	configs, err := ui.DiscoverConfigs([]string{dir})
	if err != nil {
		t.Fatalf("DiscoverConfigs: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("got %d configs, want 0", len(configs))
	}
}

func TestDiscoverConfigs_SkipsMissingDirs(t *testing.T) {
	configs, err := ui.DiscoverConfigs([]string{"/nonexistent/path/gearup"})
	if err != nil {
		t.Fatalf("DiscoverConfigs: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("got %d configs, want 0", len(configs))
	}
}

func TestDiscoverConfigs_DeduplicatesByPath(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "backend.yaml", `
version: 1
name: Backend
description: "Backend toolchain"
`)
	configs, err := ui.DiscoverConfigs([]string{dir, dir})
	if err != nil {
		t.Fatalf("DiscoverConfigs: %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("got %d configs, want 1 (deduped)", len(configs))
	}
}
