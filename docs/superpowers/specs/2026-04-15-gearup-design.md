# gearup — design spec

**Date:** 2026-04-15
**Status:** Approved via brainstorming; ready for implementation planning
**Working name:** `gearup`

## 1. Overview

`gearup` is an open-source macOS developer-machine bootstrap CLI. An engineer points it at a profile (a YAML file describing their team's tool stack), and the tool interactively plans and executes an idempotent, re-runnable provisioning of that machine.

The tool's value proposition is a **data-driven, stack-oriented provisioning model** with a **polished interactive UX** that remains **generic and safe for open-source distribution**. Teams compose their own stacks using a private recipes directory the tool consumes at runtime; no company-specific content ever lives inside the tool.

### 1.1 Goals

- Replace ad-hoc "bootstrap your laptop" shell scripts with a structured, inspectable, idempotent tool.
- Give teams a clean extensibility seam (recipes on disk) so each team can own its own tool stack without forking `gearup`.
- Provide a first-run experience on par with modern Charm-ecosystem CLIs (`gh`, `glow`): an interactive picker, a legible plan preview, live progress, and clear failure output.
- Ship a single static binary with no runtime dependencies.

### 1.2 Non-goals

- Not a configuration-management system (no Puppet/Ansible scope). Provisioning is one-shot and user-initiated.
- Not a replacement for dotfiles managers (chezmoi, yadm). `gearup` installs tools; your dotfiles manage their configs.
- Not a headless/CI tool. `gearup` requires an interactive TTY.
- Not cross-platform in v1. macOS only. Linux is a deliberate later phase.
- Not a package manager. It orchestrates package managers and installers; it never invents its own package format.

### 1.3 Hard constraints

- **No company-specific content anywhere.** The tool's source, docs, examples, default configs, test fixtures, commit messages, and output strings contain zero references to any specific organization, internal tool name, or proprietary workflow. Organization-specific specifics (elevation scripts, private recipes, internal tooling) are values the user supplies to generic mechanisms. Treat every artifact as publicly visible.
- **Idempotency by construction.** Every step declares a `check` command; the tool never installs without checking first. Re-running is always safe.
- **Stack-oriented, not role-oriented.** Profiles compose stack recipes. The schema has no notion of seniority, title, or team-internal role.

## 2. Architecture

### 2.1 High-level layers

```
┌─────────────────────────────────────────────────────┐
│                    gearup CLI                       │
│   (cobra commands + Huh forms + Lip Gloss styling)  │
└──────────────────────┬──────────────────────────────┘
                       │
         ┌─────────────┴─────────────┐
         │                           │
┌────────▼─────────┐       ┌─────────▼────────┐
│   Config layer   │       │   Runner layer   │
│  (Koanf-based)   │       │                  │
│                  │       │  • orchestrator  │
│  • loader        │       │  • step executor │
│  • merger        │       │  • elevation mgr │
│  • validator     │       │  • check runner  │
│  • resolver      │       │  • TTY streaming │
└────────┬─────────┘       └─────────┬────────┘
         │                           │
         └───────────┬───────────────┘
                     │
          ┌──────────▼──────────┐
          │  Installer plugins  │
          │  (typed step impls) │
          │                     │
          │  • brew             │
          │  • brew-cask        │
          │  • curl-pipe-sh     │
          │  • download-binary  │
          │  • git-clone        │
          │  • shell            │
          └─────────────────────┘
```

### 2.2 Go module layout

```
gearup/
├── cmd/gearup/              # main entrypoint, cobra commands
│   └── main.go
├── internal/
│   ├── cli/                 # command definitions (run, plan, list, init, ...)
│   ├── config/              # Koanf loader, merger, schema structs
│   ├── profile/             # profile + recipe resolution
│   ├── runner/              # orchestration: check → install → verify
│   ├── elevation/           # elevation banner + time-budget tracking
│   ├── installer/           # one sub-pkg per step type
│   │   ├── brew/
│   │   ├── brewcask/
│   │   ├── curlpipe/
│   │   ├── download/
│   │   ├── gitclone/
│   │   └── shell/
│   ├── ui/                  # Huh forms, Lip Gloss styles, Bubbles progress
│   └── exec/                # wrapper around os/exec with TTY streaming
├── examples/profiles/       # sample generic profiles for users to copy
├── docs/
└── go.mod
```

### 2.3 Boundaries and invariants

- **Config layer is pure and has no execution side effects.** It loads, merges, validates, and resolves references, producing a `ResolvedPlan` (ordered flat step list).
- **Runner takes a `ResolvedPlan` and executes.** Has no knowledge of where config came from.
- **Each installer sits behind a small interface:** `Check(ctx, step) (installed bool, err error)` and `Install(ctx, step) error`. Adding a typed installer is a new sub-package plus a registry entry.
- **Elevation is a runner concern, separate from installers.** The runner consults the elevation manager before entering a batch of elevation-requiring steps.
- **UI is isolated behind an event channel.** Runner emits events; UI subscribes. Makes the runner unit-testable.

### 2.4 Libraries

- **Charm stack:** Huh (forms/prompts), Bubbles (spinner, progress), Lip Gloss (styling). No full Bubble Tea unless a later phase demands it.
- **Config:** Koanf (multi-format config loader; Go analog of `unjs/c12`).
- **CLI:** cobra.
- **Process exec:** stdlib `os/exec` with a thin wrapper for TTY passthrough and line-buffered streaming.

## 3. Config schema

### 3.1 Profile file

```yaml
version: 1
name: "Dev Team"
description: "macOS dev machine for the Dev team's stack"

platform:
  os:   [darwin]
  arch: [arm64, amd64]

# Optional — only consulted if any step declares requires_elevation: true.
elevation:
  message: |
    Admin permissions are required for the next steps.
    Please run your elevation process now, then press Enter to continue.
  duration: 180s    # advisory; drives a countdown and time warnings

# Where recipes can be resolved from, in precedence order.
# v1 supports only local paths. Git sources are a deferred phase.
recipe_sources:
  - path: ~/src/my-recipes

# Stack composition. Each entry is a recipe reference or an inline step.
includes:
  - recipe: macos-base
  - recipe: jvm
  - recipe: container-stack

# Inline steps for one-offs that don't warrant a recipe.
steps:
  - name: "Team dotfiles"
    type: git-clone
    repo: git@github.com:your-org/dotfiles.git
    dest: ~/.dotfiles
```

### 3.2 Recipe file

```yaml
# ~/src/my-recipes/container-stack.yaml
version: 1
name: container-stack
description: "Docker CLI + Compose + Colima"

steps:
  - name: Docker CLI
    type: brew
    formula: docker

  - name: Docker Compose
    type: brew
    formula: docker-compose

  - name: Colima
    type: brew
    formula: colima
    post_install:
      - colima start --cpu 4 --memory 8 --disk 60
```

V1 recipes do not support transitive `requires:` between recipes; the profile must list all recipes explicitly. Transitive requires is a later phase.

### 3.3 Step shape — discriminated union by `type`

Shared envelope fields:

```yaml
- name: <string, required, unique within the resolved plan>
  type: <brew | brew-cask | curl-pipe-sh | download-binary | git-clone | shell>
  check: <shell string; optional for typed steps, required for curl-pipe-sh and shell>
  requires_elevation: <bool, default false>
  platform: <optional override; same shape as profile.platform>
  tags: [<free-form strings>]
  post_install: [<list of shell strings, run after install succeeds>]
  # type-specific fields below
```

Per-type fields and auto-derived checks:

| type              | fields                                                                                 | auto-derived check                  |
|-------------------|----------------------------------------------------------------------------------------|-------------------------------------|
| `brew`            | `formula: <string>`                                                                    | `brew list --formula <formula>`     |
| `brew-cask`       | `cask: <string>`                                                                       | `brew list --cask <cask>`           |
| `curl-pipe-sh`    | `url`, `shell` (default `bash`), `args: [<strings>]`                                   | **none — `check` is required**      |
| `download-binary` | `url`, `extract: <none\|tar.gz\|zip>`, `binary`, `install_to` (default `~/.local/bin`) | `command -v <binary>`               |
| `git-clone`       | `repo`, `dest`, `ref` (optional)                                                       | directory exists at `dest`          |
| `shell`           | `install: <multi-line shell string>`                                                   | **none — `check` is required**      |

### 3.4 `post_install` vs `shell` — when to use which

- **Typed step + `post_install`:** the installer is standard and there is a small amount of glue (start a daemon, symlink, copy a default config). Keeps the typed affordances (auto-check, uniform output).
- **`shell` step:** the whole install is custom and does not map to a typed installer.
- **Cutoff:** if `post_install` needs conditionals, multiple meaningful commands, or error handling, promote to a `shell` step.

### 3.5 Resolution algorithm (profile → flat plan)

1. Load profile; validate schema.
2. Build the recipe search path: `--recipes-dir` overrides, then `recipe_sources` (in declared order), then `$XDG_CONFIG_HOME/gearup/recipes`.
3. For each entry in `includes`, resolve `<name>.yaml` by walking the search path; first match wins.
4. Flatten: recipe steps in `includes` order, then profile inline `steps:`, preserving declared order within each.
5. Deduplicate by step name (first occurrence wins).
6. Apply platform filter: drop steps whose `platform` does not match the host.
7. Validate: step names unique, all `curl-pipe-sh` and `shell` steps have explicit `check`, all recipe references resolved.
8. Emit `ResolvedPlan{Steps, Elevation}`.

### 3.6 Config precedence (Koanf merge order, later wins)

1. Embedded defaults.
2. User config at `$XDG_CONFIG_HOME/gearup/config.yaml` (tool-level: default elevation message, cache dir, etc.).
3. Profile file (specified via `--profile` or selected interactively).
4. CLI flags.

## 4. Runtime behavior and UX

### 4.1 Command surface

```
gearup run      [--profile <path>] [--dry-run]
gearup plan     [--profile <path>]                # alias for `run --dry-run`
gearup list     profiles | recipes | steps
gearup show     <recipe-name>
gearup init
gearup version
```

Global flags: `--config <path>`, `--recipes-dir <path>`, `--no-color`, `--log-level`.

### 4.2 Interactive flow (the only flow)

`gearup` requires a TTY. Without one, it prints `gearup requires an interactive terminal` and exits.

1. **Profile picker (Huh select)** if `--profile` is not supplied. Scans `$CWD` and `$XDG_CONFIG_HOME/gearup/profiles/`.
2. **Plan preview (bullet list)** showing each step's resolved status:

   ```
   PROFILE: Dev Team  (11 steps, 9 to run, 1 requires elevation)

     1. Homebrew         will install   [requires elevation]
     2. Git              already installed — skip
     3. OpenJDK          will install
     ...
   ```

3. **Elevation banner** (only if any step requires elevation): renders the profile's `elevation.message` prominently in Lip Gloss and waits for a Huh confirm (`Enter` to proceed, `Ctrl+C` to abort).
4. **Execution** with a Bubbles spinner and per-step line. Installer stdout/stderr streams live under the step line, indented. Expanded on failure.
5. **Summary:** counts of installed / skipped / failed, total time, log file path.

### 4.3 Elevation model

`gearup` does not invoke elevation. It pauses, displays the configured `elevation.message`, and waits for the user to elevate using whatever mechanism their organization provides. On confirmation, a timer starts; all elevation-requiring steps run back-to-back inside the window. If `elevation.duration` is set, the UI shows a countdown and warns near expiry.

If a step requires elevation and the profile has no `elevation:` block, the tool falls back to native `sudo` (the user sees OS password prompts inside the step).

Rationale: `gearup` has no business invoking privileged commands or coupling to any specific elevation mechanism. Making the banner purely informational keeps the tool portable across every MDM, in-house script, biometric prompt, or native sudo flow.

### 4.4 Dry-run

`gearup run --dry-run` (or `gearup plan`) runs every step's `check` but executes no `install`. Output is the plan preview with statuses resolved. Useful for auditing a profile before first run.

### 4.5 Failure behavior

Fail fast. On first step failure:

1. Print the full stdout/stderr of the failed step (expanded, not truncated).
2. Print the exact command and exit code.
3. Print the log file path and the recipe file + line number that defined the step.
4. Exit nonzero. Remaining steps show as `not attempted`.

Re-running `gearup run` resumes naturally: already-installed steps no-op via their `check`, the failed step retries, subsequent steps proceed if it succeeds. No explicit state file.

### 4.6 Exit codes

| Code | Meaning                                                |
|------|--------------------------------------------------------|
| 0    | All steps completed, or dry-run finished successfully   |
| 1    | A step failed during execution                          |
| 2    | Config error (unresolved recipe, schema violation)      |
| 3    | User aborted (Ctrl+C or declined a confirmation)        |
| 4    | Platform mismatch (profile targets wrong OS/arch)       |

### 4.7 Logging

- Per-run log file at `$XDG_STATE_HOME/gearup/logs/<timestamp>-<profile>.log`.
- Contains everything streamed to the TTY plus structured event metadata.
- No rotation in v1.

## 5. Development approach — tracer-bullet phasing

Each phase delivers a thin, vertical, end-to-end slice. Every phase leaves a working tool. No phase builds a layer in isolation.

**Phase 1 — Tracer bullet.**
One profile + one recipe + one `path:` source + one `includes:` entry + one `brew` step. No elevation, no Huh picker (profile via `--profile`), no dry-run, no styling. Proves: config load → source path resolution → recipe lookup → step flatten → installer interface → exec → check → idempotent re-run.

**Phase 2.** Add `curl-pipe-sh` and `shell` step types. Validates that the installer interface generalizes.

**Phase 3.** Elevation flow end-to-end: `requires_elevation` flag, `elevation.message` banner, Huh confirm, `duration` countdown.

**Phase 4.** Interactive UX polish: Huh profile picker, Lip Gloss plan preview, Bubbles progress UI, `--dry-run` / `gearup plan`.

**Phase 5.** Remaining typed step types (`brew-cask`, `download-binary`, `git-clone`) and `post_install` glue.

**Phase 6.** Git-backed `recipe_sources` (clone, cache, `ref:` pinning).

**Phase 7.** Transitive `requires:` between recipes; override precedence policy.

**Phase 8+.** `gearup list`, `gearup show`, `gearup init`, `gearup doctor`; Linux support; TOML / JSONC config; log rotation.

## 6. V1 scope

### 6.1 In v1

- Commands: `gearup run`, `gearup plan`, `gearup version`.
- Step types: `brew`, `curl-pipe-sh`, `shell`.
- One profile loaded from a local YAML file (via `--profile` or a Huh picker across `$CWD` + `$XDG_CONFIG_HOME/gearup/profiles/`).
- `recipe_sources` with local `path:` entries only.
- `includes:` with flat recipe resolution (no transitive `requires:`).
- Per-step `check` command and idempotent re-run.
- `requires_elevation` + `elevation.message` + `elevation.duration` banner.
- Bullet-list plan preview, Bubbles spinner during execution, Huh confirmations.
- Per-run log file.
- TTY required; errors out otherwise.
- Exit codes 0–4.
- macOS only.

### 6.2 Out of v1 (deferred)

- Step types: `brew-cask`, `download-binary`, `git-clone`.
- `post_install` glue.
- Transitive `requires:` between recipes.
- Git-backed `recipe_sources`.
- Embedded OSS recipe library.
- Commands: `gearup list`, `gearup show`, `gearup init`, `gearup doctor`.
- Flags: `--tags`, `--json`, `--yes`, `--keep-going`.
- TOML / JSONC config formats.
- Linux support.
- Log rotation.
- Koanf env-var overrides.

## 7. Open questions (later phases)

1. **Recipe version pinning** when git sources arrive: branch vs tag vs SHA; default behavior; policy for "must pin" vs "floating main is OK."
2. **Step name collisions** across recipes. Current plan: first-match wins. Alternatives: error out, or merge. Defer until recipes are composable via `requires:`.
3. **`post_install` on re-runs** when the main step no-ops via `check`. Likely needs a dedicated `post_install_check` field. Defer with `post_install`.
4. **Parallelism.** Brew itself serializes and parallel UX is confusing; keep sequential unless a real use case surfaces.
5. **Uninstall.** Every step could eventually grow an optional `uninstall:`. Not designed now.
6. **Windows.** Not planned. WSL is "just Linux" for this tool.

## 8. Risks

- **Shell init side effects mid-run.** curl-pipe installers often mutate `.zshrc`. Steps in the same `gearup run` won't see those changes (same process, same env). Mitigation for v1: document; steps depending on a just-installed shell mutation must live in a separate run or use absolute paths.
- **Interactive installer prompts.** Homebrew's installer asks for sudo and `Y/n`. Mitigation: `curl-pipe-sh` streams TTY through; users answer inline. Document that first-time Homebrew is interactive.
- **Slow or side-effectful `check` commands** degrade UX. Mitigation: document best practices (`command -v X` is ideal; `brew list --formula X` for brew; no network calls in checks).
- **Go learning curve.** Tracer-bullet discipline keeps the first working version small; the maintainer grows Go skills incrementally alongside the tool.
