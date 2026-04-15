# gearup

Opinionated, open-source macOS developer-machine bootstrap CLI.

Status: Phase 1 (tracer bullet) — in development.

See `docs/superpowers/specs/2026-04-15-gearup-design.md` for the design.

## Phase 1 usage

    gearup run --profile ./examples/profiles/example.yaml

Requires macOS with Homebrew installed. Only the `brew` step type is supported in Phase 1.

## Phase 1 verification

Manual end-to-end test (against real Homebrew):

1. `go build -o gearup ./cmd/gearup`
2. `./gearup run --profile ./examples/profiles/example.yaml` — installs jq
3. `./gearup run --profile ./examples/profiles/example.yaml` — skips jq (idempotent)

All unit tests: `go test ./...`
