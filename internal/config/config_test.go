package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"gearup/internal/config"
)

func TestConfigUnmarshal(t *testing.T) {
	const src = `
version: 1
name: "Backend"
description: "for tests"
platform:
  os: [darwin]
  arch: [arm64, amd64]
sources:
  - path: ~/src/my-configs
extends:
  - base
  - jvm
steps:
  - name: "Inline step"
    type: brew
    formula: jq
`
	var c config.Config
	if err := yaml.Unmarshal([]byte(src), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Version != 1 {
		t.Errorf("Version = %d, want 1", c.Version)
	}
	if c.Name != "Backend" {
		t.Errorf("Name = %q", c.Name)
	}
	if got := c.Platform.OS; len(got) != 1 || got[0] != "darwin" {
		t.Errorf("Platform.OS = %v", got)
	}
	if len(c.Sources) != 1 || c.Sources[0].Path != "~/src/my-configs" {
		t.Errorf("Sources = %+v", c.Sources)
	}
	if len(c.Extends) != 2 || c.Extends[0] != "base" || c.Extends[1] != "jvm" {
		t.Errorf("Extends = %v", c.Extends)
	}
	if len(c.Steps) != 1 || c.Steps[0].Formula != "jq" {
		t.Errorf("Steps = %+v", c.Steps)
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
	var c config.Config
	if err := yaml.Unmarshal([]byte(src), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Elevation == nil {
		t.Fatal("Elevation is nil")
	}
	if c.Elevation.Message == "" {
		t.Error("Elevation.Message empty")
	}
	if c.Elevation.Duration != 180*time.Second {
		t.Errorf("Elevation.Duration = %v, want 180s", c.Elevation.Duration)
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
	var c config.Config
	if err := yaml.Unmarshal([]byte(src), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(c.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(c.Steps))
	}
	cp := c.Steps[0]
	if cp.Type != "curl-pipe-sh" || cp.URL != "https://example.com/install.sh" {
		t.Errorf("curl-pipe step = %+v", cp)
	}
	if cp.Shell != "bash" {
		t.Errorf("Shell = %q", cp.Shell)
	}
	if len(cp.Args) != 1 || cp.Args[0] != "--quiet" {
		t.Errorf("Args = %v", cp.Args)
	}
	sh := c.Steps[1]
	if sh.Type != "shell" || sh.Install == "" {
		t.Errorf("shell step = %+v", sh)
	}
}

func TestStepUnmarshal_CaskGitClonePostInstall(t *testing.T) {
	const src = `
version: 1
name: "Test"
steps:
  - name: "iTerm2"
    type: brew-cask
    cask: iterm2
  - name: "Dotfiles"
    type: git-clone
    repo: https://github.com/example/dotfiles.git
    dest: ~/.dotfiles
    ref: main
    post_install:
      - ~/.dotfiles/install.sh
`
	var c config.Config
	if err := yaml.Unmarshal([]byte(src), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(c.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(c.Steps))
	}
	if c.Steps[0].Cask != "iterm2" {
		t.Errorf("Steps[0].Cask = %q, want iterm2", c.Steps[0].Cask)
	}
	if c.Steps[1].Repo != "https://github.com/example/dotfiles.git" {
		t.Errorf("Steps[1].Repo = %q", c.Steps[1].Repo)
	}
	if c.Steps[1].Dest != "~/.dotfiles" {
		t.Errorf("Steps[1].Dest = %q", c.Steps[1].Dest)
	}
	if c.Steps[1].Ref != "main" {
		t.Errorf("Steps[1].Ref = %q", c.Steps[1].Ref)
	}
	if len(c.Steps[1].PostInstall) != 1 || c.Steps[1].PostInstall[0] != "~/.dotfiles/install.sh" {
		t.Errorf("Steps[1].PostInstall = %v", c.Steps[1].PostInstall)
	}
}

// Helper to write fixture files.
func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.yaml", `
version: 1
name: "Test"
extends:
  - sample
`)
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Name != "Test" {
		t.Errorf("Name = %q", c.Name)
	}
	if len(c.Extends) != 1 || c.Extends[0] != "sample" {
		t.Errorf("Extends = %v", c.Extends)
	}
}

func TestResolve_ExtendsFromSameDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "base.yaml", `
version: 1
name: base
steps:
  - name: "Git"
    type: brew
    formula: git
`)
	path := writeFile(t, dir, "backend.yaml", `
version: 1
name: "Backend"
extends:
  - base
steps:
  - name: "jq"
    type: brew
    formula: jq
`)
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := config.Resolve(c, dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(plan.Steps))
	}
	if plan.Steps[0].Formula != "git" {
		t.Errorf("Steps[0].Formula = %q, want git (from extended base)", plan.Steps[0].Formula)
	}
	if plan.Steps[1].Formula != "jq" {
		t.Errorf("Steps[1].Formula = %q, want jq (inline)", plan.Steps[1].Formula)
	}
}

func TestResolve_ExtendsFromDeclaredSource(t *testing.T) {
	root := t.TempDir()
	shared := filepath.Join(root, "shared")
	writeFile(t, shared, "base.yaml", `
version: 1
name: base
steps:
  - name: "Git"
    type: brew
    formula: git
`)
	path := writeFile(t, root, "backend.yaml", `
version: 1
name: "Backend"
sources:
  - path: ./shared
extends:
  - base
`)
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := config.Resolve(c, root)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Formula != "git" {
		t.Errorf("Steps = %+v", plan.Steps)
	}
}

func TestResolve_TransitiveExtends(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "core.yaml", `
version: 1
name: core
steps:
  - name: "Homebrew"
    type: curl-pipe-sh
    url: https://example.com/install.sh
    check: command -v brew
`)
	writeFile(t, dir, "base.yaml", `
version: 1
name: base
extends:
  - core
steps:
  - name: "Git"
    type: brew
    formula: git
`)
	path := writeFile(t, dir, "backend.yaml", `
version: 1
name: "Backend"
extends:
  - base
steps:
  - name: "jq"
    type: brew
    formula: jq
`)
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := config.Resolve(c, dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// core(Homebrew) + base(Git) + backend(jq)
	if len(plan.Steps) != 3 {
		t.Fatalf("Steps len = %d, want 3", len(plan.Steps))
	}
	if plan.Steps[0].Name != "Homebrew" {
		t.Errorf("Steps[0] = %q, want Homebrew (transitive from core)", plan.Steps[0].Name)
	}
	if plan.Steps[1].Name != "Git" {
		t.Errorf("Steps[1] = %q, want Git (from base)", plan.Steps[1].Name)
	}
	if plan.Steps[2].Name != "jq" {
		t.Errorf("Steps[2] = %q, want jq (inline)", plan.Steps[2].Name)
	}
}

func TestResolve_CircularExtendsDetected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", `
version: 1
name: a
extends: [b]
`)
	writeFile(t, dir, "b.yaml", `
version: 1
name: b
extends: [a]
`)
	c, err := config.Load(filepath.Join(dir, "a.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	_, err = config.Resolve(c, dir)
	if err == nil {
		t.Error("want error for circular extends, got nil")
	}
}

func TestResolve_DeduplicatesByStepName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "base.yaml", `
version: 1
name: base
steps:
  - name: "Git"
    type: brew
    formula: git
`)
	// backend extends base AND has its own Git step — should be deduped.
	path := writeFile(t, dir, "backend.yaml", `
version: 1
name: "Backend"
extends:
  - base
steps:
  - name: "Git"
    type: brew
    formula: git-custom
  - name: "jq"
    type: brew
    formula: jq
`)
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := config.Resolve(c, dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2 (Git deduped)", len(plan.Steps))
	}
	// First occurrence wins — base's Git, not backend's.
	if plan.Steps[0].Formula != "git" {
		t.Errorf("Steps[0].Formula = %q, want git (first occurrence from base)", plan.Steps[0].Formula)
	}
}

func TestResolve_ConfigNotFound(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "backend.yaml", `
version: 1
name: "Backend"
extends:
  - does-not-exist
`)
	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	_, err = config.Resolve(c, dir)
	if err == nil {
		t.Error("want error for missing config, got nil")
	}
}

func TestResolve_BackendFixture(t *testing.T) {
	c, err := config.Load("../../examples/configs/backend.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := config.Resolve(c, "../../examples/configs")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := len(plan.Steps); got != 12 {
		t.Fatalf("Steps len = %d, want 12", got)
	}
	if plan.Steps[0].Name != "Homebrew" || plan.Steps[0].Type != "curl-pipe-sh" {
		t.Errorf("Steps[0] = %+v, want Homebrew curl-pipe-sh", plan.Steps[0])
	}
	if plan.Steps[11].Name != "nvm" || plan.Steps[11].Type != "curl-pipe-sh" {
		t.Errorf("Steps[11] = %+v, want nvm curl-pipe-sh", plan.Steps[11])
	}
	if c.Elevation == nil {
		t.Fatal("backend config has no Elevation block")
	}

	// kubectl uses canonical formula
	var kubectl *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "kubectl" {
			kubectl = &plan.Steps[i]
			break
		}
	}
	if kubectl == nil {
		t.Fatal("did not find kubectl step")
	}
	if kubectl.Formula != "kubernetes-cli" {
		t.Errorf("kubectl.Formula = %q, want kubernetes-cli", kubectl.Formula)
	}

	// Docker Compose is shell type
	var compose *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "Docker Compose (CLI plugin)" {
			compose = &plan.Steps[i]
			break
		}
	}
	if compose == nil {
		t.Fatal("did not find Docker Compose step")
	}
	if compose.Type != "shell" {
		t.Errorf("compose.Type = %q, want shell", compose.Type)
	}

	// JVM symlink requires elevation
	var jvmLink *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "Link OpenJDK 21 for system Java discovery" {
			jvmLink = &plan.Steps[i]
			break
		}
	}
	if jvmLink == nil {
		t.Fatal("did not find JVM symlink step")
	}
	if !jvmLink.RequiresElevation {
		t.Error("jvm symlink should have RequiresElevation:true")
	}
}

func TestResolve_FrontendFixture(t *testing.T) {
	c, err := config.Load("../../examples/configs/frontend.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	plan, err := config.Resolve(c, "../../examples/configs")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := len(plan.Steps); got != 4 {
		t.Fatalf("Steps len = %d, want 4", got)
	}
	// Last step is nvm
	if plan.Steps[3].Name != "nvm" {
		t.Errorf("Steps[3] = %q, want nvm", plan.Steps[3].Name)
	}
}
