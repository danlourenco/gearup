package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"gearup/internal/ui"
)

func writeRecipe(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestDiscoverRecipes_FindsYAML(t *testing.T) {
	dir := t.TempDir()
	writeRecipe(t, dir, "backend.yaml", `
version: 1
name: Backend
description: "Backend toolchain"
`)
	writeRecipe(t, dir, "frontend.yaml", `
version: 1
name: Frontend
description: "Frontend toolchain"
`)
	writeRecipe(t, dir, "notes.txt", "not a recipe")

	recipes, err := ui.DiscoverRecipes([]string{dir})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 2 {
		t.Fatalf("got %d recipes, want 2", len(recipes))
	}
}

func TestDiscoverRecipes_EmptyDirs(t *testing.T) {
	dir := t.TempDir()
	recipes, err := ui.DiscoverRecipes([]string{dir})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("got %d recipes, want 0", len(recipes))
	}
}

func TestDiscoverRecipes_SkipsMissingDirs(t *testing.T) {
	recipes, err := ui.DiscoverRecipes([]string{"/nonexistent/path/gearup"})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("got %d recipes, want 0", len(recipes))
	}
}

func TestDiscoverRecipes_DeduplicatesByPath(t *testing.T) {
	dir := t.TempDir()
	writeRecipe(t, dir, "backend.yaml", `
version: 1
name: Backend
description: "Backend toolchain"
`)
	recipes, err := ui.DiscoverRecipes([]string{dir, dir})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 1 {
		t.Errorf("got %d recipes, want 1 (deduped)", len(recipes))
	}
}
