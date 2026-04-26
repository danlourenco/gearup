# TypeScript Port — Design Spec

**Date:** 2026-04-24
**Author:** Dan Lourenco
**Status:** Awaiting implementation plan

## Context

Gearup is currently a ~3,700-line Go CLI (Cobra + Charm) that provisions macOS developer machines from YAML configs. It has four commands (`run`, `plan`, `init`, `version`), five step types (`brew`, `brew-cask`, `curl-pipe-sh`, `git-clone`, `shell`), YAML configs with an `extends:` composition primitive, XDG-spec logging, and a `curl`-pipe install script that delivers a cross-compiled binary.

The tool works, but its primary maintainer is not fluent in Go, and the prospective team that will own it long-term is TypeScript-native. Continuing in Go means the team can't meaningfully read or extend the code. That is the motivating problem, and it's sufficient on its own to justify the port.

The current Go stack (Cobra + Charm — bubbletea, lipgloss, huh, bubbles) is excellent and is not being replaced because of any shortcoming. Charm is a mature, capable TUI ecosystem used by lazygit, gh, and many others. The choice to leave it is about language fluency, not library quality.

Two specific things the TypeScript ecosystem does offer that are worth naming:

1. **JSONC + JSON Schema IDE integration.** Publishing `gearup.schema.json` and having users reference it via `"$schema": "..."` in their config gives them autocomplete and inline validation in VS Code / Cursor with zero extensions. Go can emit a JSON Schema too, but the workflow is less native. This is a config-authoring win, not a TUI win.
2. **Ecosystem coherence with the team's stack.** When the team writes TS for everything else, the CLI being in TS means shared linters, formatters, test runners, and LLM-assisted tooling.

Targeting "rivals Astro / Wrangler" remains a reasonable aspiration — but as a polish target for the resulting TS code, not as a dig at Charm.

## Goals

1. **Preserve the user-facing contract exactly.** Command surface (`run`/`plan`/`init`/`version`), flags, exit codes (`0` / `10`), log file layout, and the YAML config schema all remain identical. Existing configs keep working without migration.
2. **Adopt TypeScript idioms and ecosystem libraries** aggressively where they subsume custom code. This is a rewrite, not a transliteration.
3. **Ship a single self-contained binary** per target platform via `bun build --compile`, consumed through the same `curl | bash` install flow users see today.
4. **Team maintainability** as the top non-functional goal. Prefer coherent ecosystem choices (unjs family) and familiar TS idioms over clever ones.
5. **Modern CLI UX**: JSONC-with-`$schema` IDE support, polished interactive prompts (Clack), structured output.

## Non-goals

- Breaking changes to the YAML config schema. Step types, field names, `extends:` semantics, elevation behavior — all identical.
- Cross-platform expansion. macOS remains the only supported target; `platform: [darwin]` stays.
- Performance improvement. Acceptable to be slower than Go as long as UX is not perceptibly worse.
- Migration tooling. There is no existing user base with configs to migrate; the port ships as `v0.2.0` of the same tool.

## Technology stack

| Concern | Choice | Rationale (short) |
|---|---|---|
| Runtime | **Bun** | Best npm compatibility, fastest startup, `bun build --compile` with cross-targets, team familiarity. |
| Port approach | **Idiomatic rewrite, public contract frozen** | Literal port would translate Go-isms into worse TS; full redesign breaks the config. Middle path is correct. |
| Config loader | **c12 + confbox** | unjs-native, supports JSONC/YAML/TOML, `extends:` primitive matches our composition model. |
| Canonical format | **JSONC**, YAML/TOML also parsed | Team knows JSONC. `$schema` IDE experience is best-in-class. YAML/TOML support is nearly free via confbox. |
| Schema validation | **Valibot** + `@valibot/to-json-schema` | Modern mental model (schemas + pipes), first-class `v.variant()` for the step union, official JSON Schema exporter. |
| CLI framework | **citty** | unjs ecosystem coherence, functional `defineCommand` style matches the rest of the codebase, `runCommand`/`renderUsage` cover testing. |
| Subprocess | **execa** behind an `Exec` interface | Battle-tested, rich error objects, shell mode, timeout, streaming — all first-class. Wrapped behind our interface for testability and replaceability. |
| Test runner | **`bun:test`** | Zero config, native Bun, Jest-compatible API, fastest iteration. |
| Elevation banner (Phase 2) + Picker (Phase 4) | **@clack/prompts** | The Astro-originated picker library. Brought forward to Phase 2 for the elevation banner; Phase 4 adds the picker. |
| XDG paths (Phase 3) | **pathe** | unjs cross-platform path normalization. |

### Decisions considered and rejected

- **Runtime: Deno.** Narrower npm compat; Clack and several other target libraries have rough edges under Deno. Team familiarity with Node/Bun idioms is higher.
- **Schema: Zod.** Valid alternative; picked Valibot for cleaner teaching model, smaller bundle (negligible here but signals modern), and official JSON Schema exporter. Reversible if we ever need tRPC/Next.js interop.
- **CLI framework: clipanion.** Superior help polish and Wrangler pedigree. Rejected because class-based style is a stylistic island in an otherwise functional codebase, and citty's testing story (`runCommand` + `renderUsage`) is sufficient for a 4-command CLI where the tests concentrate below the CLI layer.
- **CLI framework: commander.** Ubiquitous but weakly-typed and API-dated. Not the "modern CLI" signal we want.
- **Subprocess: Bun.spawn direct / tinyexec.** Bun.spawn means reimplementing 60% of execa over four phases. tinyexec's feature set is narrower than what we need.
- **Test runner: Vitest.** Fine choice; `bun:test` wins on coherence and zero config for a Bun-native project.

## Architecture

See [`docs/ts-port-architecture.md`](../../ts-port-architecture.md) for the full architectural overview. Summary:

The system is a one-directional pipeline from argv to executed subprocess. Every layer has one purpose and a well-defined input/output:

```
argv → CLI router → Config loader → Validator → Runner → Handler → exec
```

- **CLI router** (citty) parses argv and dispatches to a command.
- **Config loader** (c12 + confbox) reads a file in any supported format and produces a raw object. Handles `extends:` composition (Phase 3).
- **Validator** (Valibot) parses the raw object into a typed `Config`, narrowing each step to its concrete variant.
- **Runner** iterates steps and dispatches each to its handler via the registry.
- **Handler registry** maps `step.type` → `{ check, install }` functions. One file per step type.
- **Exec / Context** is the single subprocess seam. Threaded through every handler; swappable for `FakeExec` in tests.

### Data model

The step schema is a Valibot discriminated union keyed on `type`:

```ts
// src/schema/step.ts
import * as v from "valibot"

const baseStep = {
  name: v.pipe(v.string(), v.minLength(1)),
  check: v.optional(v.string()),
  requires_elevation: v.optional(v.boolean()),
  post_install: v.optional(v.array(v.string())),
}

export const brewStep = v.object({ type: v.literal("brew"),         ...baseStep, formula: v.string() })
export const brewCaskStep = v.object({ type: v.literal("brew-cask"), ...baseStep, cask: v.string() })
export const curlPipeStep = v.object({ type: v.literal("curl-pipe-sh"), ...baseStep,
  url: v.string(),
  check: v.string(),                            // required for curl-pipe
})
export const gitCloneStep = v.object({ type: v.literal("git-clone"), ...baseStep,
  repo: v.string(),
  dest: v.string(),
  ref: v.optional(v.string()),
})
export const shellStep = v.object({ type: v.literal("shell"),         ...baseStep,
  install: v.string(),
  check: v.string(),                            // required for shell
})

export const step = v.variant("type", [brewStep, brewCaskStep, curlPipeStep, gitCloneStep, shellStep])
export type Step = v.InferOutput<typeof step>
```

The top-level config schema:

```ts
// src/schema/config.ts
export const config = v.object({
  version: v.literal(1),
  name: v.string(),
  description: v.optional(v.string()),
  platform: v.optional(v.object({
    os: v.array(v.picklist(["darwin"])),
  })),
  elevation: v.optional(v.object({
    message: v.string(),
    duration: v.optional(v.string()),          // "180s" etc.
  })),
  extends: v.optional(v.array(v.string())),
  sources: v.optional(v.array(v.object({ path: v.string() }))),
  steps: v.optional(v.array(step)),
})
export type Config = v.InferOutput<typeof config>
```

### Context shape

```ts
// src/context.ts
export type Context = {
  exec: Exec
  cwd: string
  env: Record<string, string>
  log: Logger                                   // Phase 1: console wrapper; Phase 3: XDG file logger
}
```

The `Exec` interface:

```ts
// src/exec/types.ts
export type RunInput = {
  argv: string[]
  shell?: boolean
  cwd?: string
  env?: Record<string, string>
  stdin?: string
  timeout?: number                              // default 60_000
}

export type RunResult = {
  exitCode: number
  stdout: string
  stderr: string
  durationMs: number
  timedOut: boolean
}

export interface Exec {
  run(input: RunInput): Promise<RunResult>
}
```

### Handler registry

```ts
// src/steps/index.ts
import { checkBrew,     installBrew     } from "./brew"
import { checkBrewCask, installBrewCask } from "./brew-cask"
import { checkCurlPipe, installCurlPipe } from "./curl-pipe-sh"
import { checkGitClone, installGitClone } from "./git-clone"
import { checkShell,    installShell    } from "./shell"

export type Handler<S extends Step> = {
  check: (step: S, ctx: Context) => Promise<CheckResult>
  install?: (step: S, ctx: Context) => Promise<InstallResult>   // Phase 1: optional
}

export const handlers = {
  brew:           { check: checkBrew,     install: installBrew     },
  "brew-cask":    { check: checkBrewCask, install: installBrewCask },
  "curl-pipe-sh": { check: checkCurlPipe, install: installCurlPipe },
  "git-clone":    { check: checkGitClone, install: installGitClone },
  shell:          { check: checkShell,    install: installShell    },
} satisfies { [K in Step["type"]]: Handler<Extract<Step, { type: K }>> }
```

The `satisfies` enforces: adding a step type requires (a) a new Valibot schema, (b) an entry in `v.variant`, (c) an entry here. The compiler fails closed on all three.

## Phasing

| Phase | Scope | Exit criteria |
|---|---|---|
| **1. Tracer: `plan`** | `gearup plan --config <path>` • c12+confbox parsing (all 3 formats) • single-file configs (no `extends:`) • all 5 step types' **check** handlers • stdout report • exit codes 0/10 • `gearup version` | `plan` runs end-to-end against fixture configs in all 3 formats; exit codes are correct; all 5 handlers have passing unit tests. |
| **2. `run` + full step types** | Install dispatch for all 5 types • `post_install` hooks • elevation pause banner via `@clack/prompts` (note + confirm) • still stdout, no log file yet | `run` completes an install path on a real macOS machine; elevation banner suppresses when no elevation-required steps remain; `post_install` only runs on successful installs. |
| **3. Robustness** | `extends:` composition (c12's layered merge + name→path pre-resolver) • XDG logging via pathe (`$XDG_STATE_HOME/gearup/logs/`) with captured subprocess output • inline failure printing with log path | Can run existing `backend.yaml` (with its `extends: [base, jvm, ...]`) unchanged; log files are written at the documented path; on step failure the captured stderr is printed alongside `Log: <path>`. |
| **4. Polish** | Interactive picker when `--config` omitted (`@clack/prompts` select) • styled step status lines (spinners, check marks, timings) • `gearup init` with embedded default configs • release pipeline (`bun build --compile` matrix + install.sh rewrite) | Feature parity with current Go gearup; `curl -fsSL .../install.sh | bash` installs the Bun-compiled binary; interactive picker flow matches current UX. |

## Testing strategy

- **Unit tests co-located** with source files (`brew.ts` + `brew.test.ts`). `bun test` discovers them anywhere under the repo.
- **`FakeExec`** is the single test double for subprocess interaction. Handlers receive a `Context` with a `FakeExec`; tests queue responses and assert on the `calls` array. No mocking library; no monkey-patching.
- **Schema tests** parse representative fixtures and assert both success and failure branches. Snapshot-test the emitted JSON Schema output.
- **Integration tests** use `runCommand(planCommand, { rawArgs: [...] })` from citty against fixture configs. Capture output via `spyOn(console, "log")`.
- **Shared fixtures** at `tests/fixtures/` hold `single-brew.jsonc`, `single-brew.yaml`, `single-brew.toml`, and progressively richer configs. All three formats are tested per phase where parsing is exercised.
- **No tests for citty's help rendering in Phase 1**, but `renderUsage(mainCommand)` is trivially snapshot-testable if we decide to add that later.

## Release strategy

- `scripts/build-binary.ts` wraps `bun build --compile` across a target matrix: `bun-darwin-arm64`, `bun-darwin-x64`, and (optionally) `bun-linux-x64` / `bun-linux-arm64` as a low-cost hedge.
- Each binary ships with a `sha256` checksum file.
- `install.sh` is rewritten to detect architecture and download the matching Bun-compiled binary from the GitHub release (same shape as today; just different artifact names).
- `scripts/build-schema.ts` emits `schema/gearup.schema.json` from the Valibot schemas. The JSON file is committed (so users get a stable `$schema` URL) and CI verifies it's in sync.
- GitHub Actions workflow: test on macOS, build the matrix, upload to GitHub Release, publish the schema. Replaces the current goreleaser workflow.

## File layout

```
gearup/
├── package.json
├── tsconfig.json
├── bunfig.toml
├── install.sh                    Rewritten for Bun-compiled binary releases
├── README.md
├── LICENSE
│
├── configs/                      Embedded default configs (JSONC) — Phase 4
│   ├── base.jsonc
│   ├── backend.jsonc
│   ├── frontend.jsonc
│   ├── containers.jsonc
│   ├── jvm.jsonc
│   ├── aws-k8s.jsonc
│   ├── node.jsonc
│   └── desktop-apps.jsonc
│
├── schema/
│   └── gearup.schema.json        Built artifact, committed for stable $schema URL
│
├── scripts/
│   ├── build-schema.ts
│   └── build-binary.ts
│
├── src/
│   ├── cli.ts                    citty mainCommand entrypoint
│   ├── context.ts
│   │
│   ├── commands/
│   │   ├── plan.ts   + .test.ts  Phase 1
│   │   ├── run.ts                Phase 2
│   │   ├── init.ts               Phase 4
│   │   └── version.ts            Phase 1
│   │
│   ├── config/
│   │   ├── load.ts   + .test.ts  c12 + confbox integration
│   │   ├── resolve.ts            Phase 3: name→path resolver for extends:
│   │   └── embedded.ts           Phase 4: access embedded defaults
│   │
│   ├── schema/
│   │   ├── step.ts               v.variant + per-type schemas
│   │   ├── config.ts             top-level v.object
│   │   ├── index.ts              public exports + inferred types
│   │   └── schema.test.ts
│   │
│   ├── runner/
│   │   └── run.ts    + .test.ts  orchestration loop
│   │
│   ├── steps/                    handler registry
│   │   ├── brew.ts          + .test.ts
│   │   ├── brew-cask.ts     + .test.ts
│   │   ├── curl-pipe-sh.ts  + .test.ts
│   │   ├── git-clone.ts     + .test.ts
│   │   ├── shell.ts         + .test.ts
│   │   └── index.ts              handlers registry with `satisfies`
│   │
│   ├── exec/
│   │   ├── types.ts              Exec interface, RunInput, RunResult
│   │   ├── execa.ts              Production impl
│   │   ├── fake.ts               FakeExec for tests
│   │   └── exec.test.ts
│   │
│   ├── log/                      Phase 3
│   │   ├── logger.ts             XDG file logger
│   │   └── logger.test.ts
│   │
│   └── ui/                       Phase 4
│       ├── picker.ts             @clack/prompts select
│       ├── spinner.ts
│       └── banner.ts             Elevation banner
│
└── tests/
    └── fixtures/                 Shared config files across many tests
        ├── single-brew.jsonc
        ├── single-brew.yaml
        ├── single-brew.toml
        ├── mixed-types.jsonc
        └── ...
```

## `package.json` skeleton

```jsonc
{
  "name": "gearup",
  "version": "0.2.0",
  "type": "module",
  "bin": { "gearup": "./src/cli.ts" },
  "scripts": {
    "dev": "bun run src/cli.ts",
    "test": "bun test",
    "typecheck": "tsc --noEmit",
    "build": "bun build src/cli.ts --compile --outfile=bin/gearup",
    "build:schema": "bun run scripts/build-schema.ts",
    "build:release": "bun run scripts/build-binary.ts"
  },
  "dependencies": {
    "citty": "^0.1.6",
    "c12": "^2.0.0",
    "confbox": "^0.1.7",
    "valibot": "^0.40.0",
    "@valibot/to-json-schema": "^0.2.0",
    "execa": "^9.5.0"
  },
  "devDependencies": {
    "typescript": "^5.6.0",
    "@types/bun": "latest"
  }
}
```

Phase 2 adds `@clack/prompts` (elevation banner). Phase 3 adds `pathe`. Phase 4 uses `@clack/prompts` further (picker, spinners). Total runtime dependency count lands at 8 — small, coherent, and unjs-aligned.

## Open questions deferred to implementation

- **`extends:` name resolution rules.** c12 expects paths; existing configs use names (`extends: [base]` → `base.yaml` in the search path). The pre-resolver must replicate current Go behavior exactly, including the `sources:` declaration for non-same-dir configs. Details land in Phase 3.
- **Elevation prompt mechanism in Phase 2.** Uses `@clack/prompts` (`clack.note` for the banner + `clack.confirm` for the prompt). Originally planned as `readline`, but the readline UX would be a regression vs the Charm-based Go version, so Clack was brought forward from Phase 4 — one extra dep that we know we want anyway.
- **`curl-pipe-sh` shell invocation.** Using `execa("sh", ["-c", command])` vs execa's `$` template is a Phase 2 implementation choice, not a design decision.
- **Log format details.** Current Go gearup writes plaintext logs with command output interleaved. The TS port matches this byte-for-byte in Phase 3.

These are all small, well-bounded decisions. None of them change the architecture.

## Success criteria for the port overall

1. All four commands work identically to current Go gearup against the same fixture configs.
2. `curl -fsSL .../install.sh | bash` produces a working binary on `darwin-arm64` and `darwin-x64`.
3. Interactive picker, elevation banner, log output, and exit codes match or exceed current UX.
4. Every handler has a unit test with `FakeExec`. `plan` and `run` have at least one integration test each.
5. `gearup.schema.json` is published; a JSONC config with `"$schema"` gets autocomplete and validation in VS Code / Cursor.
6. A new TS developer can read the codebase and understand the pipeline in under an hour.

---

*This spec captures the output of the brainstorming interview conducted on 2026-04-24. The implementation plan will be written next, decomposing Phase 1 into a sequence of tracer-bullet steps.*
