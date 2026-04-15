package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"gearup/internal/profile"
)

// writeFile is a helper for fixture setup.
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

func TestLoadProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "profile.yaml", `
version: 1
name: "Test"
includes:
  - recipe: sample
`)
	p, err := profile.LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if p.Name != "Test" {
		t.Errorf("Name = %q", p.Name)
	}
	if len(p.Includes) != 1 || p.Includes[0].Recipe != "sample" {
		t.Errorf("Includes = %+v", p.Includes)
	}
}

func TestResolve_RecipeFromLocalPath(t *testing.T) {
	root := t.TempDir()
	recipesDir := filepath.Join(root, "my-recipes")

	writeFile(t, recipesDir, "sample.yaml", `
version: 1
name: sample
steps:
  - name: "Install jq"
    type: brew
    formula: jq
`)
	profilePath := writeFile(t, root, "profile.yaml", `
version: 1
name: "Test"
recipe_sources:
  - path: `+recipesDir+`
includes:
  - recipe: sample
steps:
  - name: "Inline step"
    type: brew
    formula: git
`)

	p, err := profile.LoadProfile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	plan, err := profile.Resolve(p, filepath.Dir(profilePath))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, want := len(plan.Steps), 2; got != want {
		t.Fatalf("Steps len = %d, want %d (%+v)", got, want, plan.Steps)
	}
	if plan.Steps[0].Formula != "jq" {
		t.Errorf("Steps[0].Formula = %q, want jq", plan.Steps[0].Formula)
	}
	if plan.Steps[1].Formula != "git" {
		t.Errorf("Steps[1].Formula = %q, want git", plan.Steps[1].Formula)
	}
}

func TestResolve_RecipeNotFound(t *testing.T) {
	root := t.TempDir()
	profilePath := writeFile(t, root, "profile.yaml", `
version: 1
name: "Test"
includes:
  - recipe: does-not-exist
`)
	p, err := profile.LoadProfile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	_, err = profile.Resolve(p, filepath.Dir(profilePath))
	if err == nil {
		t.Error("want error for missing recipe, got nil")
	}
}

func TestResolve_RelativePathResolvedAgainstProfileDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "recipes"), "sample.yaml", `
version: 1
name: sample
steps:
  - name: s
    type: brew
    formula: jq
`)
	profilePath := writeFile(t, root, "profile.yaml", `
version: 1
name: "Test"
recipe_sources:
  - path: ./recipes
includes:
  - recipe: sample
`)
	p, err := profile.LoadProfile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	plan, err := profile.Resolve(p, filepath.Dir(profilePath))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Formula != "jq" {
		t.Errorf("Steps = %+v", plan.Steps)
	}
}

func TestResolve_ExampleFixture(t *testing.T) {
	p, err := profile.LoadProfile("../../examples/profiles/example.yaml")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	plan, err := profile.Resolve(p, "../../examples/profiles")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("Steps len = %d, want 1 (%+v)", len(plan.Steps), plan.Steps)
	}
	if plan.Steps[0].Type != "brew" || plan.Steps[0].Formula != "jq" {
		t.Errorf("Steps[0] = %+v", plan.Steps[0])
	}
}
