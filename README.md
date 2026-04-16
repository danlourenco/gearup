# gearup

Opinionated, open-source macOS developer-machine bootstrap CLI.

Status: Phase 3 — in development. Supports three step types, per-step `requires_elevation` with a configurable banner + confirmation, on macOS.

See `docs/superpowers/specs/2026-04-15-gearup-design.md` for the design.

## Usage

    gearup run --recipe ./examples/recipes/backend.yaml

Recipes compose ingredients. An ingredient is a reusable bundle of steps
for one stack concern (e.g. JVM, containers, AWS/K8s). Example recipes
for a `backend` and `frontend` role live in `examples/recipes/`, composed
from shared ingredients in `examples/ingredients/`. Point `ingredient_sources`
at your own path to share ingredients across recipes and teams.

### Elevation

Steps that need admin permissions (e.g., writing to `/Library/...`) declare
`requires_elevation: true`. A recipe can include a top-level `elevation:`
block whose `message` is shown in a styled banner before such steps run —
e.g., "Please elevate admin permissions now, then press Continue." gearup
doesn't invoke elevation itself; it pauses and waits. If no `elevation:`
block is set, steps that need sudo prompt for a password natively as they
run.

See `examples/recipes/backend.yaml` for a full example, and
`examples/ingredients/jvm.yaml` for a step that requires elevation.

Requires macOS. If Homebrew is not installed, the recipe's first step installs it via the official installer. Subsequent brew steps in the same run require `brew` on PATH — if Homebrew was just installed, open a new shell and re-run so PATH picks up `/opt/homebrew/bin` (or `/usr/local/bin` on Intel).

## Verifying a run

Idempotency check on any recipe:

    ./gearup run --recipe ./examples/recipes/backend.yaml
    ./gearup run --recipe ./examples/recipes/backend.yaml   # every step should skip

All unit tests: `go test ./...`
