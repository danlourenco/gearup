package recipe_test

import (
	"os"
	"path/filepath"
	"testing"

	"gearup/internal/config"
	"gearup/internal/recipe"
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

func TestLoadRecipe(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "recipe.yaml", `
version: 1
name: "Test"
ingredients:
  - sample
`)
	r, err := recipe.LoadRecipe(path)
	if err != nil {
		t.Fatalf("LoadRecipe: %v", err)
	}
	if r.Name != "Test" {
		t.Errorf("Name = %q", r.Name)
	}
	if len(r.Ingredients) != 1 || r.Ingredients[0] != "sample" {
		t.Errorf("Ingredients = %+v", r.Ingredients)
	}
}

func TestResolve_RecipeFromLocalPath(t *testing.T) {
	root := t.TempDir()
	ingredientsDir := filepath.Join(root, "my-ingredients")

	writeFile(t, ingredientsDir, "sample.yaml", `
version: 1
name: sample
steps:
  - name: "Install jq"
    type: brew
    formula: jq
`)
	recipePath := writeFile(t, root, "recipe.yaml", `
version: 1
name: "Test"
ingredient_sources:
  - path: `+ingredientsDir+`
ingredients:
  - sample
steps:
  - name: "Inline step"
    type: brew
    formula: git
`)

	r, err := recipe.LoadRecipe(recipePath)
	if err != nil {
		t.Fatalf("LoadRecipe: %v", err)
	}
	plan, err := recipe.Resolve(r, filepath.Dir(recipePath))
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
	recipePath := writeFile(t, root, "recipe.yaml", `
version: 1
name: "Test"
ingredients:
  - does-not-exist
`)
	r, err := recipe.LoadRecipe(recipePath)
	if err != nil {
		t.Fatalf("LoadRecipe: %v", err)
	}
	_, err = recipe.Resolve(r, filepath.Dir(recipePath))
	if err == nil {
		t.Error("want error for missing ingredient, got nil")
	}
}

func TestResolve_RelativePathResolvedAgainstRecipeDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ingredients"), "sample.yaml", `
version: 1
name: sample
steps:
  - name: s
    type: brew
    formula: jq
`)
	recipePath := writeFile(t, root, "recipe.yaml", `
version: 1
name: "Test"
ingredient_sources:
  - path: ./ingredients
ingredients:
  - sample
`)
	r, err := recipe.LoadRecipe(recipePath)
	if err != nil {
		t.Fatalf("LoadRecipe: %v", err)
	}
	plan, err := recipe.Resolve(r, filepath.Dir(recipePath))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Formula != "jq" {
		t.Errorf("Steps = %+v", plan.Steps)
	}
}

func TestResolve_BackendFixture(t *testing.T) {
	r, err := recipe.LoadRecipe("../../examples/recipes/backend.yaml")
	if err != nil {
		t.Fatalf("LoadRecipe: %v", err)
	}
	plan, err := recipe.Resolve(r, "../../examples/recipes")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, want := len(plan.Steps), 12; got != want {
		t.Fatalf("Steps len = %d, want %d", got, want)
	}
	// spot-check: first step is Homebrew (curl-pipe-sh), last step is nvm (curl-pipe-sh)
	if plan.Steps[0].Type != "curl-pipe-sh" || plan.Steps[0].Name != "Homebrew" {
		t.Errorf("Steps[0] = %+v, want Homebrew curl-pipe-sh", plan.Steps[0])
	}
	if plan.Steps[11].Type != "curl-pipe-sh" || plan.Steps[11].Name != "nvm" {
		t.Errorf("Steps[11] = %+v, want nvm curl-pipe-sh", plan.Steps[11])
	}
	// kubectl uses the canonical `kubernetes-cli` formula to avoid brew-alias idempotency issues.
	var kubectl *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "kubectl" {
			kubectl = &plan.Steps[i]
			break
		}
	}
	if kubectl == nil {
		t.Fatal("did not find step named 'kubectl' in backend recipe")
	}
	if kubectl.Formula != "kubernetes-cli" {
		t.Errorf("kubectl.Formula = %q, want kubernetes-cli", kubectl.Formula)
	}
	// Docker Compose is installed via the `shell` escape hatch (plugin manual-install).
	var compose *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "Docker Compose (CLI plugin)" {
			compose = &plan.Steps[i]
			break
		}
	}
	if compose == nil {
		t.Fatal("did not find step named 'Docker Compose (CLI plugin)' in backend recipe")
	}
	if compose.Type != "shell" {
		t.Errorf("compose.Type = %q, want shell", compose.Type)
	}
	if compose.Install == "" {
		t.Error("compose.Install is empty")
	}

	// The JVM ingredient includes a symlink step that requires elevation.
	var jvmSymlink *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "Link OpenJDK 21 for system Java discovery" {
			jvmSymlink = &plan.Steps[i]
			break
		}
	}
	if jvmSymlink == nil {
		t.Fatal("did not find JVM symlink step")
	}
	if !jvmSymlink.RequiresElevation {
		t.Error("jvm symlink step should have RequiresElevation:true")
	}

	// Phase 3: the backend recipe declares an elevation block.
	if r.Elevation == nil {
		t.Fatal("backend recipe has no Elevation block")
	}
	if r.Elevation.Duration == 0 {
		t.Error("backend recipe Elevation.Duration is zero")
	}
}

func TestResolve_FrontendFixture(t *testing.T) {
	r, err := recipe.LoadRecipe("../../examples/recipes/frontend.yaml")
	if err != nil {
		t.Fatalf("LoadRecipe: %v", err)
	}
	plan, err := recipe.Resolve(r, "../../examples/recipes")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, want := len(plan.Steps), 4; got != want {
		t.Fatalf("Steps len = %d, want %d (%+v)", got, want, plan.Steps)
	}
	// base: Homebrew, Git, jq — node: nvm
	var nvm *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "nvm" {
			nvm = &plan.Steps[i]
			break
		}
	}
	if nvm == nil {
		t.Fatal("did not find step named 'nvm' in frontend recipe")
	}
	if nvm.Type != "curl-pipe-sh" {
		t.Errorf("nvm.Type = %q, want curl-pipe-sh", nvm.Type)
	}
}
