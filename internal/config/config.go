// Package config defines the data types for gearup profiles and recipes.
// It contains no I/O or resolution logic — just struct shapes and YAML tags.
package config

// Profile is the top-level entry point loaded from a profile YAML file.
type Profile struct {
	Version       int            `yaml:"version"`
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description,omitempty"`
	Platform      Platform       `yaml:"platform,omitempty"`
	RecipeSources []RecipeSource `yaml:"recipe_sources,omitempty"`
	Includes      []Include      `yaml:"includes,omitempty"`
	Steps         []Step         `yaml:"steps,omitempty"`
}

// Platform constrains which OS/arch a profile (or step) applies to.
type Platform struct {
	OS   []string `yaml:"os,omitempty"`
	Arch []string `yaml:"arch,omitempty"`
}

// RecipeSource declares a location where recipes can be resolved from.
// Phase 1 supports only local filesystem paths.
type RecipeSource struct {
	Path string `yaml:"path,omitempty"`
}

// Include references a recipe by name; resolved against the search path.
type Include struct {
	Recipe string `yaml:"recipe"`
}

// Recipe is a bundle of steps loaded from a recipe YAML file.
type Recipe struct {
	Version     int    `yaml:"version"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Steps       []Step `yaml:"steps"`
}

// Step is one unit of provisioning work.
// Phase 1 only implements type: brew, so the type-specific fields here
// are minimal; future phases add cask, url, repo, etc.
type Step struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"`
	Check             string   `yaml:"check,omitempty"`
	RequiresElevation bool     `yaml:"requires_elevation,omitempty"`
	Platform          Platform `yaml:"platform,omitempty"`

	// brew-specific
	Formula string `yaml:"formula,omitempty"`
}

// ResolvedPlan is a flattened, ordered list of steps produced by
// resolving a profile's recipe references.
type ResolvedPlan struct {
	Profile *Profile
	Steps   []Step
}
