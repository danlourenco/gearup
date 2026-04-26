# TypeScript Port — Architectural Overview

> **Companion to** [`superpowers/specs/2026-04-24-typescript-port-design.md`](./superpowers/specs/2026-04-24-typescript-port-design.md). This doc is the quick architectural reference; the spec is the full design with rationale, phasing, and migration detail.

## The 30-second pitch

Gearup is a config-driven runner with a replaceable handler registry. A user writes a config file listing tools they want installed. Gearup parses it, validates it into typed step records, and hands each step to a dedicated handler that knows how to (a) check if it's already there and (b) install it if not. Everything flows one direction; every layer is swappable for testing.

## The pipeline

```
  argv                      "plan --config backend.jsonc"
    │
    ▼
┌──────────────────┐
│  CLI router      │  Routes argv to a command handler.
│  (citty/similar) │  Knows nothing about steps — just commands + flags.
└──────────────────┘
    │
    ▼
┌──────────────────┐
│  Config loader   │  Reads the file. Picks parser by extension (JSONC/YAML/TOML).
│  (c12 + confbox) │  Returns a plain untyped object. Resolves `extends:` (Phase 3).
└──────────────────┘
    │
    ▼
┌──────────────────┐
│  Validator       │  Parses the object against the schema.
│  (Valibot)       │  Output: typed `Config { steps: Step[] }` where each Step
│                  │  is narrowed to its variant (BrewStep, ShellStep, etc.)
└──────────────────┘
    │
    ▼
┌──────────────────┐
│  Runner          │  Iterates steps. For each one, looks up its handler
│                  │  in the registry and calls `check` (and later `install`).
│                  │  Knows nothing about specific step types.
└──────────────────┘
    │
    ▼
┌──────────────────┐
│  Handler         │  `brew.ts`, `shell.ts`, etc. One per step type.
│  (registry)      │  Each exposes `check(step, ctx)` and `install(step, ctx)`.
│                  │  Handlers don't talk to each other.
└──────────────────┘
    │
    ▼
┌──────────────────┐
│  exec (Context)  │  The one place subprocesses get spawned.
│                  │  Swapped for a fake in tests.
└──────────────────┘
    │
    ▼
  real commands    `brew list --formula jq`, `curl ... | bash`, etc.
```

## What each piece is, and why it's its own piece

| Layer | What it does | Why it's separate |
|---|---|---|
| **CLI router** | Parses `argv`, figures out which command + flags | CLI parsing is tedious and has nothing to do with installing software. Isolating it means the rest of the codebase never sees `process.argv`. |
| **Config loader** | File → raw object, format-agnostic | Lets us support JSONC/YAML/TOML without the rest of the code caring. File-format complexity stops here. |
| **Validator** | Raw object → typed `Config` | The whole codebase downstream can trust the data. No "maybe a string, maybe undefined" questions. Also: this is where the user gets their error messages when their config is wrong. |
| **Runner** | Orchestrates step-by-step execution | This is the only place that knows "we do checks first, skip if already installed, otherwise install, and run post_install if install succeeded." That flow lives in one file and nowhere else. |
| **Handler registry** | Maps `step.type` → the functions that handle it | Adding a new step type = add a file. Changing how `brew` works = edit one file. No central switch statement to grow. |
| **Context / exec** | The subprocess seam | Handlers don't import `child_process` or `Bun.spawn` directly — they take a `Context` that has `exec` on it. Tests pass a fake; no mocking library needed. |

## Why this shape vs alternatives

- **Why not one big file?** A config-file parser, a shell executor, a UI, and a CLI all in one module means any change risks breaking unrelated things. With layers you can rewrite the config loader (swap c12 for something else) without touching handlers.
- **Why handlers in a registry instead of a switch?** Because `brew` and `git-clone` have essentially nothing in common except that they both conform to the same `{ check, install }` interface. Co-locating them in one switch statement spreads one step type's logic across many case branches over time; co-locating in one file per type keeps it cohesive.
- **Why a Context object instead of globals?** Tests. With globals you'd need to monkey-patch `child_process` or use a mocking library. With a Context you construct a fake Context, pass it in, and your handler runs under test conditions with zero magic. Same mechanism production uses.
- **Why split `check` and `install`?** Idempotency. `gearup plan` (dry-run) only needs `check`. `gearup run` needs both. The split is what makes "skip things already installed" a natural property of the system rather than logic grafted on top.
- **Why Record-keyed steps?** Composition via `extends:` works naturally with c12+defu when steps are a Record keyed by name: the deep-merge collapses same-keyed entries with current-config-wins semantics. An array shape would have required a custom merger to dedup. The Record form is also stricter (can't express duplicates in a single config) and gives better error paths (`steps.Homebrew.formula` vs `steps[3].formula`). The internal `Step[]` view is recovered via a Valibot transform that injects `name` from the key.

## Working decisions (as of 2026-04-24)

| Decision | Lives in |
|---|---|
| **Bun** | The runtime everything runs on. Shows up in `exec` (`Bun.spawn`) and the build (`bun build --compile`). |
| **c12 + confbox** | The Config loader layer only. |
| **JSONC canonical** (YAML/TOML accepted) | The Config loader layer + a `gearup.schema.json` file published alongside releases that users reference via `$schema`. |
| **Valibot** | The Validator layer. Defines `Step` as `v.variant("type", [...])`. Also the source-of-truth for the published JSON Schema (`@valibot/to-json-schema` emits it at build time). |
| **Handler registry with `satisfies`** | The Handler layer. The `satisfies` is what ensures "add a step type = update three places" stays enforced by the compiler. |
| **Single Context seam** | Threaded from the CLI router all the way to the handlers. This is the one shared parameter. |

## Phasing

| Phase | Scope |
|---|---|
| **1. Tracer: `plan`** | `gearup plan --config <path>` • c12+confbox parsing (all 3 formats) • single-file configs (no `extends:` yet) • all 5 step types' **check** handlers • stdout report • exit codes 0/10 |
| **2. `run` + full step types** | Install dispatch for all 5 types • `post_install` hooks • elevation pause banner • still stdout, no log file yet |
| **3. Robustness** | `extends:` composition via c12 (paths, packages, github: refs) • Record-keyed steps schema (drops the dedup-merger problem) • `sources:` field removed in favor of explicit paths • curl-pipe-sh schema hardening (URL/shell/args validation) • XDG logging via FileLogger + LoggingExec decorator with captured subprocess output and inline failure printing |
| **4. Polish** | Interactive picker when `--config` omitted (Clack) • animated spinners / styled banners • `gearup init` + embedded default configs • release pipeline (Bun compile matrix + install.sh rewrite) |

Phase 1 builds the whole pipeline but only implements `check` in handlers. Phase 2 fills in `install`. Phase 3 adds `extends:` inside the Config loader and a `log` on the Context. Phase 4 adds a picker before the Config loader and a prettier output layer after the Runner.

## Mental model

> Gearup is **a pipeline** (argv → config → validated → dispatched → executed), where **each handler is a plugin** for one step type, and **the Context is the injection seam** that makes it all testable.

## Locked tech stack

| Concern | Choice |
|---|---|
| Runtime | **Bun** |
| Config loader | **c12 + confbox** — active since Phase 3: loads JSONC/YAML/TOML, resolves `extends:` references recursively (paths, packages, `github:` refs), deep-merges with defu (current config wins on collision) |
| Canonical format | **JSONC** (YAML/TOML parsed but not advocated) |
| Schema validation | **Valibot** (+ `@valibot/to-json-schema` for the published schema) |
| CLI framework | **citty** |
| Subprocess | **execa** behind an `Exec` interface |
| Test runner | **`bun:test`** |
| Elevation banner (Phase 2) + Picker (Phase 4) | **@clack/prompts** |
| XDG paths (Phase 3) | **pathe** |

## File layout

See the full tree in the design spec. High level:

```
src/
├── cli.ts                 citty mainCommand
├── context.ts             Context type + factory
├── commands/              plan.ts, run.ts, init.ts, version.ts
├── config/                c12 integration, extends resolver, embedded access
├── schema/                Valibot step + config schemas; source of truth
├── runner/                the check/install loop
├── steps/                 one file per step type, plus handler registry
├── exec/                  Exec interface, execa impl, FakeExec
├── log/                   (Phase 3) XDG file logger
└── ui/                    (Phase 4) Clack picker, spinner, banner
```

Tests are co-located (`foo.ts` + `foo.test.ts`). Shared fixtures live at `tests/fixtures/`.
