# gearup

Opinionated, open-source macOS developer-machine bootstrap CLI.

Status: Phase 2 — in development. Supports `brew`, `curl-pipe-sh`, and `shell` step types on macOS.

See `docs/superpowers/specs/2026-04-15-gearup-design.md` for the design.

## Usage

    gearup run --profile ./examples/profiles/dev-stack.yaml

Requires macOS. If Homebrew is not installed, the profile's first step installs it via the official installer. Subsequent brew steps in the same run require `brew` on PATH — if Homebrew was just installed, open a new shell and re-run so PATH picks up `/opt/homebrew/bin` (or `/usr/local/bin` on Intel).

## Phase 1 verification

Manual end-to-end test (against real Homebrew):

1. `go build -o gearup ./cmd/gearup`
2. `./gearup run --profile ./examples/profiles/example.yaml` — installs jq
3. `./gearup run --profile ./examples/profiles/example.yaml` — skips jq (idempotent)

All unit tests: `go test ./...`
