package config_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"gearup/internal/config"
)

func TestProfileUnmarshal(t *testing.T) {
	const src = `
version: 1
name: "Example Profile"
description: "for tests"
platform:
  os: [darwin]
  arch: [arm64, amd64]
recipe_sources:
  - path: ~/src/my-recipes
includes:
  - recipe: example-recipe
steps:
  - name: "Inline step"
    type: brew
    formula: jq
`
	var p config.Profile
	if err := yaml.Unmarshal([]byte(src), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Version != 1 {
		t.Errorf("Version = %d, want 1", p.Version)
	}
	if p.Name != "Example Profile" {
		t.Errorf("Name = %q, want %q", p.Name, "Example Profile")
	}
	if got := p.Platform.OS; len(got) != 1 || got[0] != "darwin" {
		t.Errorf("Platform.OS = %v, want [darwin]", got)
	}
	if len(p.RecipeSources) != 1 || p.RecipeSources[0].Path != "~/src/my-recipes" {
		t.Errorf("RecipeSources = %+v", p.RecipeSources)
	}
	if len(p.Includes) != 1 || p.Includes[0].Recipe != "example-recipe" {
		t.Errorf("Includes = %+v", p.Includes)
	}
	if len(p.Steps) != 1 || p.Steps[0].Type != "brew" || p.Steps[0].Formula != "jq" {
		t.Errorf("Steps = %+v", p.Steps)
	}
}

func TestRecipeUnmarshal(t *testing.T) {
	const src = `
version: 1
name: example-recipe
description: "test recipe"
steps:
  - name: "Install jq"
    type: brew
    formula: jq
  - name: "Install git"
    type: brew
    formula: git
`
	var r config.Recipe
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Name != "example-recipe" {
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
	var p config.Profile
	if err := yaml.Unmarshal([]byte(src), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(p.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(p.Steps))
	}
	cp := p.Steps[0]
	if cp.Type != "curl-pipe-sh" || cp.URL != "https://example.com/install.sh" {
		t.Errorf("curl-pipe step = %+v", cp)
	}
	if cp.Shell != "bash" {
		t.Errorf("Shell = %q, want bash", cp.Shell)
	}
	if len(cp.Args) != 1 || cp.Args[0] != "--quiet" {
		t.Errorf("Args = %v", cp.Args)
	}
	sh := p.Steps[1]
	if sh.Type != "shell" || sh.Install == "" {
		t.Errorf("shell step = %+v", sh)
	}
}
