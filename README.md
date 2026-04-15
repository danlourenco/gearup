# gearup

Opinionated, open-source macOS developer-machine bootstrap CLI.

Status: Phase 2 — in development. Supports `brew`, `curl-pipe-sh`, and `shell` step types on macOS.

See `docs/superpowers/specs/2026-04-15-gearup-design.md` for the design.

## Usage

    gearup run --recipe ./examples/recipes/backend.yaml

Recipes compose ingredients. An ingredient is a reusable bundle of steps
for one stack concern (e.g. JVM, containers, AWS/K8s). Example recipes
for a `backend` and `frontend` role live in `examples/recipes/`, composed
from shared ingredients in `examples/ingredients/`. Point `ingredient_sources`
at your own path to share ingredients across recipes and teams.

Requires macOS. If Homebrew is not installed, the recipe's first step installs it via the official installer. Subsequent brew steps in the same run require `brew` on PATH — if Homebrew was just installed, open a new shell and re-run so PATH picks up `/opt/homebrew/bin` (or `/usr/local/bin` on Intel).

## Verifying a run

Idempotency check on any recipe:

    ./gearup run --recipe ./examples/recipes/backend.yaml
    ./gearup run --recipe ./examples/recipes/backend.yaml   # every step should skip

All unit tests: `go test ./...`
