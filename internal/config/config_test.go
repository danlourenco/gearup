package config_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"gearup/internal/config"
)

func TestRecipeUnmarshal(t *testing.T) {
	const src = `
version: 1
name: "Example Recipe"
description: "for tests"
platform:
  os: [darwin]
  arch: [arm64, amd64]
ingredient_sources:
  - path: ~/src/my-ingredients
ingredients:
  - example-ingredient
steps:
  - name: "Inline step"
    type: brew
    formula: jq
`
	var r config.Recipe
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Version != 1 {
		t.Errorf("Version = %d, want 1", r.Version)
	}
	if r.Name != "Example Recipe" {
		t.Errorf("Name = %q, want %q", r.Name, "Example Recipe")
	}
	if got := r.Platform.OS; len(got) != 1 || got[0] != "darwin" {
		t.Errorf("Platform.OS = %v, want [darwin]", got)
	}
	if len(r.IngredientSources) != 1 || r.IngredientSources[0].Path != "~/src/my-ingredients" {
		t.Errorf("IngredientSources = %+v", r.IngredientSources)
	}
	if len(r.Ingredients) != 1 || r.Ingredients[0] != "example-ingredient" {
		t.Errorf("Ingredients = %+v", r.Ingredients)
	}
	if len(r.Steps) != 1 || r.Steps[0].Type != "brew" || r.Steps[0].Formula != "jq" {
		t.Errorf("Steps = %+v", r.Steps)
	}
}

func TestIngredientUnmarshal(t *testing.T) {
	const src = `
version: 1
name: example-ingredient
description: "test ingredient"
steps:
  - name: "Install jq"
    type: brew
    formula: jq
  - name: "Install git"
    type: brew
    formula: git
`
	var r config.Ingredient
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Name != "example-ingredient" {
		t.Errorf("Name = %q", r.Name)
	}
	if len(r.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(r.Steps))
	}
	if r.Steps[0].Formula != "jq" || r.Steps[1].Formula != "git" {
		t.Errorf("Steps = %+v", r.Steps)
	}
}

func TestStepUnmarshal_CurlPipeAndShell(t *testing.T) {
	const src = `
version: 1
name: "Mixed"
steps:
  - name: "nvm"
    type: curl-pipe-sh
    url: https://example.com/install.sh
    shell: bash
    args: ["--quiet"]
    check: 'test -f "$HOME/.nvm/nvm.sh"'
  - name: "Custom"
    type: shell
    check: command -v thing
    install: |
      echo "installing thing"
      touch thing
`
	var r config.Recipe
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(r.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(r.Steps))
	}
	cp := r.Steps[0]
	if cp.Type != "curl-pipe-sh" || cp.URL != "https://example.com/install.sh" {
		t.Errorf("curl-pipe step = %+v", cp)
	}
	if cp.Shell != "bash" {
		t.Errorf("Shell = %q, want bash", cp.Shell)
	}
	if len(cp.Args) != 1 || cp.Args[0] != "--quiet" {
		t.Errorf("Args = %v", cp.Args)
	}
	sh := r.Steps[1]
	if sh.Type != "shell" || sh.Install == "" {
		t.Errorf("shell step = %+v", sh)
	}
}

func TestElevationUnmarshal(t *testing.T) {
	const src = `
version: 1
name: "Backend"
elevation:
  message: "Please elevate admin permissions now, then press Enter."
  duration: 180s
`
	var r config.Recipe
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Elevation == nil {
		t.Fatal("Elevation is nil, want non-nil")
	}
	if r.Elevation.Message == "" {
		t.Error("Elevation.Message empty")
	}
	if r.Elevation.Duration != 180*time.Second {
		t.Errorf("Elevation.Duration = %v, want 180s", r.Elevation.Duration)
	}
}
