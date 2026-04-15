// Package config defines the data types for gearup recipes and ingredients.
// It contains no I/O or resolution logic — just struct shapes and YAML tags.
package config

import "time"

// Recipe is the top-level entry point loaded from a recipe YAML file.
// A recipe composes ingredients plus optional inline steps.
type Recipe struct {
	Version           int                `yaml:"version"`
	Name              string             `yaml:"name"`
	Description       string             `yaml:"description,omitempty"`
	Platform          Platform           `yaml:"platform,omitempty"`
	Elevation         *Elevation         `yaml:"elevation,omitempty"`
	IngredientSources []IngredientSource `yaml:"ingredient_sources,omitempty"`
	Ingredients       []string           `yaml:"ingredients,omitempty"`
	Steps             []Step             `yaml:"steps,omitempty"`
}

// Platform constrains which OS/arch a recipe (or step) applies to.
type Platform struct {
	OS   []string `yaml:"os,omitempty"`
	Arch []string `yaml:"arch,omitempty"`
}

// IngredientSource declares a location where ingredients can be resolved from.
type IngredientSource struct {
	Path string `yaml:"path,omitempty"`
}

// Ingredient is a reusable bundle of steps loaded from an ingredient YAML file,
// referenced by name from a recipe's `ingredients` list.
type Ingredient struct {
	Version     int    `yaml:"version"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Steps       []Step `yaml:"steps"`
}

// Elevation describes a recipe-level request for admin permissions.
// When a recipe has an Elevation block and at least one step declares
// RequiresElevation: true, the runner shows the Message in a banner,
// asks the user to confirm (assumed to have acquired elevation through
// whatever mechanism their org provides), then runs the elevation-required
// steps back-to-back. If Duration is non-zero, the runner warns if a
// subsequent elevation step is about to begin when less than 30s remain.
type Elevation struct {
	Message  string        `yaml:"message"`
	Duration time.Duration `yaml:"duration,omitempty"`
}

// Step is one unit of provisioning work.
type Step struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"`
	Check             string   `yaml:"check,omitempty"`
	RequiresElevation bool     `yaml:"requires_elevation,omitempty"`
	Platform          Platform `yaml:"platform,omitempty"`

	// brew-specific
	Formula string `yaml:"formula,omitempty"`

	// curl-pipe-sh
	URL   string   `yaml:"url,omitempty"`
	Shell string   `yaml:"shell,omitempty"`
	Args  []string `yaml:"args,omitempty"`

	// shell (raw)
	Install string `yaml:"install,omitempty"`
}

// ResolvedPlan is a flattened, ordered list of steps produced by resolving
// a recipe's ingredient references.
type ResolvedPlan struct {
	Recipe *Recipe
	Steps  []Step
}
