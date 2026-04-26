# TS Port — Phase 3 Extends + Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `extends:` config composition (via c12) and XDG file logging (with captured subprocess output and inline failure printing) to the TypeScript port. Also: re-shape the schema for c12-friendliness (steps as Record keyed by name, drop the `sources:` field) and harden the `curl-pipe-sh` schema (URL validator, shell allowlist, args whitespace).

**Architecture:** Lean on c12 for everything: load the user's config file, recursively follow `extends:` references (relative paths, npm packages, `github:` URLs), deep-merge with defu defaults. The schema change to a Record-of-steps means defu's natural override semantics (current-config-wins) handles step dedup with zero custom merger code. Add a `Logger` interface threaded through `Context`; wrap the production `Exec` in a `LoggingExec` decorator that records every subprocess call to a timestamped file under `$XDG_STATE_HOME/gearup/logs/`.

**Tech Stack:** Phase 1+2 stack (Bun, citty, Valibot, execa, bun:test, @clack/prompts) plus c12 (already declared, now actually used) and pathe (for XDG path normalization).

---

## Preflight

Phase 1 + Phase 2 are merged to `main`. This plan extends the codebase further. Go code remains untouched.

**Before Task 1, create a worktree:**

```bash
cd /Users/dlo/Dev/gearup
git worktree add .worktrees/phase-3-extends -b ts-port/phase-3-extends-and-logging
cd .worktrees/phase-3-extends
```

All subsequent tasks assume you are in `.worktrees/phase-3-extends/`. The branch merges back to `main` at the end.

**Reference files from Go that Phase 3 mirrors:**
- `internal/config/config.go` — `Resolve` (extends recursion), `findConfig` (search path), `dedup`. We DO NOT mirror this byte-for-byte; we use c12 with a Record-keyed schema for cleaner semantics.
- `internal/log/log.go` — XDG log path resolution; matches the format `<YYYYMMDD>-<HHMMSS>-<configname>.log`.
- `internal/runner/runner.go` — log-on-failure UX (`Log: <path>` printed inline).

**Behavior notes & deliberate divergences from Go:**

1. **Schema change:** `steps` is now a Record keyed by step name (was an array). The `name` field disappears from step bodies (it becomes the key). Internal `Step` type stays the same after Valibot's transform injects `name` from the key.
2. **Override semantics:** Current config overrides extended configs on key collision (defu defaults). Go's behavior was "first wins" (extended wins), which was counterintuitive — this is a fix, not a regression.
3. **`sources:` field removed:** Replaced by c12-native path references in `extends:` (`./relative.jsonc`, `~/abs/path.yaml`, `github:owner/repo/path`, etc.).
4. **Bare-name extends no longer supported:** `extends: [base]` (Go syntax) becomes `extends: ["./base"]` (c12 syntax). The `./` prefix is required for local files. Extension is auto-resolved (`./base` finds `base.jsonc`/`base.yaml`/`base.toml`).

---

## File Structure (Phase 3)

```
gearup/
├── package.json                      MODIFY — add pathe; remove c12 if it's still listed unused (it isn't — Phase 2 has it but unused; this phase actually uses it)
├── src/
│   ├── schema/
│   │   ├── step.ts                   MODIFY — drop `name` from step bodies; tighten curl-pipe-sh (URL/shell/args)
│   │   ├── config.ts                 MODIFY — `steps` becomes Record + transform; drop `sources:`
│   │   └── schema.test.ts            MODIFY — update tests for new shape; add hardening tests
│   ├── config/
│   │   ├── load.ts                   REWRITE — c12.loadConfig + cwd/configFile derivation + normalize
│   │   └── load.test.ts              MODIFY — extends fixtures + override semantics + extension resolution
│   ├── log/
│   │   ├── types.ts                  NEW — Logger interface
│   │   ├── xdg.ts                    NEW — XDG path resolution (logDir, timestampedFilename)
│   │   ├── xdg.test.ts               NEW
│   │   ├── file.ts                   NEW — FileLogger (Bun writer-backed)
│   │   ├── file.test.ts              NEW
│   │   ├── fake.ts                   NEW — FakeLogger (in-memory, for tests)
│   │   └── fake.test.ts              NEW
│   ├── exec/
│   │   ├── logging.ts                NEW — LoggingExec decorator
│   │   └── logging.test.ts           NEW
│   ├── context.ts                    MODIFY — add `log: Logger` field
│   ├── context.test.ts               MODIFY — assert log defaults
│   └── commands/
│       └── run.ts                    MODIFY — open FileLogger, wrap Exec, pass to Context, close on finally; print log path on failure
└── tests/
    └── fixtures/
        ├── extends-base.jsonc        NEW — a minimal base config (1 step)
        ├── extends-child.jsonc       NEW — extends extends-base.jsonc; adds 1 step; verifies merge
        ├── extends-override.jsonc    NEW — extends extends-base.jsonc; overrides one of base's steps
        └── (existing fixtures will be MIGRATED to Record-of-steps form in Task 1)
```

---

## Task 1: Schema redesign — Record-keyed steps, drop sources, migrate fixtures and schema tests

This is the foundational change. Steps become a Record keyed by step name; the `name` field disappears from step bodies. Valibot's `transform` normalizes back to `Step[]` so internal types and handlers don't change. The `sources:` field is removed from the config schema.

This is a single atomic commit because all four pieces (schema, schema tests, fixtures, fixture-using tests) need to land together.

**Files:**
- Modify: `src/schema/step.ts`
- Modify: `src/schema/config.ts`
- Modify: `src/schema/schema.test.ts`
- Modify: `tests/fixtures/single-brew.jsonc`, `single-brew.yaml`, `single-brew.toml`, `all-five-types.jsonc`, `all-installed.jsonc`, `never-installed.jsonc`, `safe-install.jsonc`, `elevation-required.jsonc`

- [ ] **Step 1: Modify `src/schema/step.ts` — drop `name` from each variant body**

Replace the entire file with:

```ts
import * as v from "valibot"

const platform = v.object({
  os: v.optional(v.array(v.picklist(["darwin", "linux"]))),
  arch: v.optional(v.array(v.string())),
})

// `check` is optional here so brew/brew-cask/git-clone can fall back to a default;
// curl-pipe-sh and shell override it to required because they have no sensible default.
// `name` is NOT in the body — it lives on the parent map's key.
const baseFields = {
  check: v.optional(v.string()),
  requires_elevation: v.optional(v.boolean()),
  post_install: v.optional(v.array(v.string())),
  platform: v.optional(platform),
}

export const brewStepBody = v.object({
  type: v.literal("brew"),
  ...baseFields,
  formula: v.pipe(v.string(), v.minLength(1)),
})

export const brewCaskStepBody = v.object({
  type: v.literal("brew-cask"),
  ...baseFields,
  cask: v.pipe(v.string(), v.minLength(1)),
})

export const curlPipeStepBody = v.object({
  type: v.literal("curl-pipe-sh"),
  ...baseFields,
  url: v.pipe(v.string(), v.url()),
  shell: v.optional(v.picklist(["bash", "sh", "zsh", "fish"])),
  args: v.optional(v.array(v.pipe(v.string(), v.regex(/^\S+$/)))),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const gitCloneStepBody = v.object({
  type: v.literal("git-clone"),
  ...baseFields,
  repo: v.pipe(v.string(), v.minLength(1)),
  dest: v.pipe(v.string(), v.minLength(1)),
  ref: v.optional(v.string()),
})

export const shellStepBody = v.object({
  type: v.literal("shell"),
  ...baseFields,
  install: v.pipe(v.string(), v.minLength(1)),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const stepBody = v.variant("type", [
  brewStepBody,
  brewCaskStepBody,
  curlPipeStepBody,
  gitCloneStepBody,
  shellStepBody,
])

// Internal types: each step has `name` (injected from the Record key during config parsing).
type WithName<T> = T & { name: string }
export type BrewStep = WithName<v.InferOutput<typeof brewStepBody>>
export type BrewCaskStep = WithName<v.InferOutput<typeof brewCaskStepBody>>
export type CurlPipeStep = WithName<v.InferOutput<typeof curlPipeStepBody>>
export type GitCloneStep = WithName<v.InferOutput<typeof gitCloneStepBody>>
export type ShellStep = WithName<v.InferOutput<typeof shellStepBody>>
export type Step = WithName<v.InferOutput<typeof stepBody>>
```

- [ ] **Step 2: Modify `src/schema/config.ts` — Record-keyed steps + transform; drop `sources`**

Replace the entire file with:

```ts
import * as v from "valibot"
import { stepBody, type Step } from "./step"

// Steps are authored as a Record keyed by name; we transform to Step[] with name injected
// from the key. This shape lets defu (via c12) handle override semantics natively when configs
// are merged through `extends:`.
const stepsRecord = v.pipe(
  v.record(v.string(), stepBody),
  v.transform((rec): Step[] =>
    Object.entries(rec).map(([name, body]) => ({ name, ...body } as Step)),
  ),
)

export const config = v.object({
  version: v.literal(1),
  name: v.pipe(v.string(), v.minLength(1)),
  description: v.optional(v.string()),
  platform: v.optional(
    v.object({
      os: v.optional(v.array(v.picklist(["darwin", "linux"]))),
      arch: v.optional(v.array(v.string())),
    }),
  ),
  elevation: v.optional(
    v.object({
      message: v.string(),
      duration: v.optional(v.string()),
    }),
  ),
  extends: v.optional(v.array(v.string())),
  steps: v.optional(stepsRecord),
})

export type Config = v.InferOutput<typeof config>
```

- [ ] **Step 3: Modify `src/schema/schema.test.ts` — update existing tests + add hardening tests**

Replace the entire file with:

```ts
import { describe, it, expect } from "bun:test"
import * as v from "valibot"
import { stepBody } from "./step"
import { config } from "./config"

describe("step body schema", () => {
  it("parses a brew step body (no name field)", () => {
    const parsed = v.parse(stepBody, { type: "brew", formula: "jq" })
    expect(parsed.type).toBe("brew")
    if (parsed.type === "brew") {
      expect(parsed.formula).toBe("jq")
    }
  })

  it("parses a brew-cask step body", () => {
    const parsed = v.parse(stepBody, { type: "brew-cask", cask: "iterm2" })
    expect(parsed.type).toBe("brew-cask")
  })

  it("parses a curl-pipe-sh step body with valid URL", () => {
    const parsed = v.parse(stepBody, {
      type: "curl-pipe-sh",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh with non-URL string for url", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "not a url",
        check: "command -v brew",
      }),
    ).toThrow()
  })

  it("rejects curl-pipe-sh with disallowed shell", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "https://example.com/install.sh",
        shell: "rm",
        check: "command -v brew",
      }),
    ).toThrow()
  })

  it("accepts curl-pipe-sh with allowed shell", () => {
    const parsed = v.parse(stepBody, {
      type: "curl-pipe-sh",
      url: "https://example.com/install.sh",
      shell: "sh",
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh args containing whitespace within an arg", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "https://example.com/install.sh",
        args: ["valid", "has space"],
        check: "command -v brew",
      }),
    ).toThrow()
  })

  it("accepts curl-pipe-sh args that are individually whitespace-free", () => {
    const parsed = v.parse(stepBody, {
      type: "curl-pipe-sh",
      url: "https://example.com/install.sh",
      args: ["-y", "--default-toolchain", "stable"],
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh without check", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "curl-pipe-sh",
        url: "https://example.com/install.sh",
      }),
    ).toThrow()
  })

  it("parses a git-clone step body", () => {
    const parsed = v.parse(stepBody, {
      type: "git-clone",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    })
    expect(parsed.type).toBe("git-clone")
  })

  it("parses a shell step body with required check", () => {
    const parsed = v.parse(stepBody, {
      type: "shell",
      install: "curl ... | sh",
      check: "command -v rustc",
    })
    expect(parsed.type).toBe("shell")
  })

  it("rejects shell step without check", () => {
    expect(() =>
      v.parse(stepBody, {
        type: "shell",
        install: "curl ... | sh",
      }),
    ).toThrow()
  })

  it("rejects an unknown step type", () => {
    expect(() => v.parse(stepBody, { type: "wat" })).toThrow()
  })

  it("accepts optional fields: post_install, requires_elevation, check, platform", () => {
    const parsed = v.parse(stepBody, {
      type: "brew",
      formula: "colima",
      check: "command -v colima",
      requires_elevation: false,
      post_install: ["colima start --cpu 4 --memory 8"],
      platform: { os: ["darwin"] },
    })
    expect(parsed.type).toBe("brew")
    if (parsed.type === "brew") {
      expect(parsed.post_install).toEqual(["colima start --cpu 4 --memory 8"])
    }
  })
})

describe("config schema", () => {
  it("parses a minimal config", () => {
    const parsed = v.parse(config, { version: 1, name: "base" })
    expect(parsed.name).toBe("base")
    expect(parsed.steps).toBeUndefined()
  })

  it("parses a config with steps as a Record (transformed to Step[] with name injected)", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      steps: {
        jq: { type: "brew", formula: "jq" },
        Git: { type: "brew", formula: "git" },
      },
    })
    expect(parsed.steps).toHaveLength(2)
    expect(parsed.steps?.[0]).toEqual({ type: "brew", name: "jq", formula: "jq" })
    expect(parsed.steps?.[1]).toEqual({ type: "brew", name: "Git", formula: "git" })
  })

  it("parses a config with extends", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      extends: ["./base", "./jvm"],
    })
    expect(parsed.extends).toEqual(["./base", "./jvm"])
  })

  it("parses a config with elevation", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "team",
      elevation: { message: "Admin please", duration: "180s" },
    })
    expect(parsed.elevation?.message).toBe("Admin please")
  })

  it("parses a config with both extends and steps coexisting", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "full",
      extends: ["./base"],
      steps: { jq: { type: "brew", formula: "jq" } },
    })
    expect(parsed.extends).toHaveLength(1)
    expect(parsed.steps).toHaveLength(1)
  })

  it("rejects an elevation block missing message", () => {
    expect(() => v.parse(config, { version: 1, name: "x", elevation: {} })).toThrow()
  })

  it("rejects version !== 1", () => {
    expect(() => v.parse(config, { version: 2, name: "x" })).toThrow()
  })

  it("rejects a config with no name", () => {
    expect(() => v.parse(config, { version: 1 })).toThrow()
  })

  it("silently drops the removed `sources` field (Valibot's v.object is non-strict)", () => {
    // The legacy `sources` field is no longer in the schema. Valibot's v.object
    // accepts unknown keys silently rather than rejecting them, so legacy configs
    // won't fail to parse — they just lose the field. If Valibot ever flips to
    // strict-mode by default, this test will fail loudly and we'll know.
    const parsed = v.parse(config, {
      version: 1,
      name: "x",
      sources: [{ path: "./somewhere" }],  // legacy field; should be silently dropped
    } as never)
    expect(parsed.name).toBe("x")
    expect((parsed as Record<string, unknown>).sources).toBeUndefined()
  })
})
```

- [ ] **Step 4: Migrate fixtures to Record-keyed `steps`**

Update each fixture file to use the new shape. Each step's `name` becomes the Record key.

`tests/fixtures/single-brew.jsonc`:

```jsonc
{
  "version": 1,
  "name": "single-brew",
  "steps": {
    "jq": { "type": "brew", "formula": "jq" }
  }
}
```

`tests/fixtures/single-brew.yaml`:

```yaml
version: 1
name: single-brew
steps:
  jq:
    type: brew
    formula: jq
```

`tests/fixtures/single-brew.toml`:

```toml
version = 1
name = "single-brew"

[steps.jq]
type = "brew"
formula = "jq"
```

`tests/fixtures/all-five-types.jsonc`:

```jsonc
{
  "version": 1,
  "name": "all-five-types",
  "steps": {
    "jq":       { "type": "brew",         "formula": "jq" },
    "iTerm2":   { "type": "brew-cask",    "cask": "iterm2" },
    "Homebrew": { "type": "curl-pipe-sh", "url": "https://example.com/install.sh", "check": "command -v brew" },
    "dotfiles": { "type": "git-clone",    "repo": "git@github.com:me/dotfiles.git", "dest": "/tmp/dotfiles" },
    "rust":     { "type": "shell",        "install": "curl ... | sh", "check": "command -v rustc" }
  }
}
```

`tests/fixtures/all-installed.jsonc`:

```jsonc
{
  "version": 1,
  "name": "all-installed",
  "steps": {
    "jq": { "type": "brew", "formula": "jq" }
  }
}
```

`tests/fixtures/never-installed.jsonc`:

```jsonc
{
  "version": 1,
  "name": "never-installed",
  "steps": {
    "always-missing": { "type": "shell", "install": "true", "check": "false" }
  }
}
```

`tests/fixtures/safe-install.jsonc`:

```jsonc
{
  "version": 1,
  "name": "safe-install",
  "steps": {
    "marker": {
      "type": "shell",
      "install": "touch /tmp/gearup-e2e-marker",
      "check": "test -f /tmp/gearup-e2e-marker",
      "post_install": ["echo post-install ran > /tmp/gearup-e2e-post"]
    }
  }
}
```

`tests/fixtures/elevation-required.jsonc`:

```jsonc
{
  "version": 1,
  "name": "elevation-required",
  "elevation": {
    "message": "This run needs admin permissions for one step. Acquire them, then continue.",
    "duration": "180s"
  },
  "steps": {
    "needs-admin": {
      "type": "shell",
      "install": "true",
      "check": "false",
      "requires_elevation": true
    }
  }
}
```

- [ ] **Step 5: Run tests**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun test
```

Expected:
- Schema tests pass with the new shape (~17 tests in schema.test.ts).
- All other tests still pass — they construct `Config` objects inline using the OUTPUT type (`steps: Step[]` after transform), which hasn't changed.
- E2E tests pass because fixtures now load through the same loader; the loader (Task 2) hasn't changed yet, but `loadConfig` parses the new Record-form fixtures into the same `Config` output shape via the transform.

If anything fails outside `schema.test.ts` and the e2e tests, fix it inline — most likely a test was constructing a Config with Record-form OR was checking schema input shape directly.

- [ ] **Step 6: Run typecheck**

```bash
bun run typecheck
```
Expected: exit 0.

- [ ] **Step 7: Commit**

```bash
git add src/schema/ tests/fixtures/
git commit -m "feat(schema): redesign steps as Record-keyed-by-name; harden curl-pipe-sh"
```

---

## Task 2: Switch loader to c12 — extends, deep merge, and path-based references

Replace the direct confbox calls in `src/config/load.ts` with c12's `loadConfig`. c12 owns: file loading, recursive `extends:` resolution, deep merging via defu. We add: `--config <path>` → `{ cwd, configFile }` translation, plus our existing post-validation step (Valibot parse).

**Files:**
- Modify: `package.json` (verify c12 is in dependencies; remove `confbox` from explicit deps if c12 transitively brings it — keep if you prefer the explicit dep)
- Modify: `src/config/load.ts`
- Modify: `src/config/load.test.ts`
- Create: `tests/fixtures/extends-base.jsonc`, `tests/fixtures/extends-child.jsonc`, `tests/fixtures/extends-override.jsonc`

- [ ] **Step 1: Verify c12 dep is present**

Check `package.json`. It should already have `"c12": "^2.0.0"` from Phase 1 (it's been declared but unused). If for some reason it's been removed, run:

```bash
bun add c12@^2.0.0
```

`confbox` may still be a direct dep from Phase 1 — keep it. c12 uses confbox internally; declaring it explicitly does no harm and lets us still import `confbox/jsonc` etc. if we ever need raw parsing.

- [ ] **Step 2: Create the three extends fixtures**

`tests/fixtures/extends-base.jsonc`:

```jsonc
{
  "version": 1,
  "name": "extends-base",
  "steps": {
    "jq": { "type": "brew", "formula": "jq" }
  }
}
```

`tests/fixtures/extends-child.jsonc`:

```jsonc
{
  "version": 1,
  "name": "extends-child",
  "extends": ["./extends-base"],
  "steps": {
    "git": { "type": "brew", "formula": "git" }
  }
}
```

`tests/fixtures/extends-override.jsonc`:

```jsonc
{
  "version": 1,
  "name": "extends-override",
  "extends": ["./extends-base"],
  "steps": {
    "jq": { "type": "shell", "install": "echo override", "check": "false" }
  }
}
```

- [ ] **Step 3: Rewrite `src/config/load.ts` to use c12**

```ts
import { loadConfig as c12LoadConfig } from "c12"
import * as v from "valibot"
import path from "node:path"
import fs from "node:fs/promises"
import { config as configSchema, type Config } from "../schema"

// Known config extensions, in priority order matching c12's resolution.
const KNOWN_EXTENSIONS = [
  ".jsonc",
  ".json",
  ".yaml",
  ".yml",
  ".toml",
  ".ts",
  ".js",
  ".mjs",
  ".cjs",
]

/**
 * Split a user-provided config path into c12's expected (cwd, configFile) pair.
 *
 * c12's `configFile` is a base name without extension; it tries each supported extension
 * in `cwd`. We accept either a path with extension (`backend.jsonc`) or without
 * (`backend`), absolute or relative to process.cwd().
 */
function deriveLoadOptions(configPath: string): { cwd: string; configFile: string } {
  const abs = path.resolve(configPath)
  const dir = path.dirname(abs)
  const base = path.basename(abs)
  const ext = KNOWN_EXTENSIONS.find((e) => base.endsWith(e))
  const stem = ext ? base.slice(0, -ext.length) : base
  return { cwd: dir, configFile: stem }
}

export async function loadConfig(configPath: string): Promise<Config> {
  // Surface a clearer error when the file doesn't exist — c12's error is opaque.
  const abs = path.resolve(configPath)
  try {
    await fs.access(abs)
  } catch {
    throw new Error(`cannot read config ${abs}: file not found`)
  }

  const { cwd, configFile } = deriveLoadOptions(configPath)

  const { config: raw } = await c12LoadConfig({
    cwd,
    configFile,
    // No defaults; rcFile/globalRc disabled for predictability.
    rcFile: false,
    globalRc: false,
  })

  if (raw == null || (typeof raw === "object" && Object.keys(raw).length === 0)) {
    throw new Error(`config ${abs} loaded as empty — c12 may not have found the file`)
  }

  try {
    return v.parse(configSchema, raw)
  } catch (err) {
    if (err instanceof v.ValiError) {
      const issues = err.issues
        .map(
          (i) =>
            `  - ${i.path?.map((p: v.IssuePathItem) => p.key).join(".") ?? "<root>"}: ${i.message}`,
        )
        .join("\n")
      throw new Error(`config ${abs} failed schema validation:\n${issues}`)
    }
    throw err
  }
}
```

- [ ] **Step 4: Rewrite `src/config/load.test.ts`**

Replace the entire file with:

```ts
import { describe, it, expect } from "bun:test"
import { loadConfig } from "./load"
import path from "node:path"
import fs from "node:fs/promises"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("loadConfig", () => {
  it("loads a JSONC file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.jsonc"))
    expect(config.name).toBe("single-brew")
    expect(config.steps).toHaveLength(1)
    expect(config.steps?.[0]?.type).toBe("brew")
    expect(config.steps?.[0]?.name).toBe("jq")
  })

  it("loads a YAML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.yaml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
    expect(config.steps?.[0]?.name).toBe("jq")
  })

  it("loads a TOML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.toml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
    expect(config.steps?.[0]?.name).toBe("jq")
  })

  it("loads a config with all five step types", async () => {
    const config = await loadConfig(path.join(fixtures, "all-five-types.jsonc"))
    expect(config.steps).toHaveLength(5)
    const types = config.steps?.map((s) => s.type)
    expect(types).toEqual([
      "brew",
      "brew-cask",
      "curl-pipe-sh",
      "git-clone",
      "shell",
    ])
  })

  it("throws with a helpful message when the file is not found", async () => {
    await expect(loadConfig("/nope/missing.jsonc")).rejects.toThrow(/missing\.jsonc/)
  })

  it("throws with schema-validation detail when the file is invalid", async () => {
    const tmpPath = path.join(fixtures, "__invalid-tmp.jsonc")
    await fs.writeFile(tmpPath, JSON.stringify({ version: 2, name: "bad" }))
    try {
      await expect(loadConfig(tmpPath)).rejects.toThrow()
    } finally {
      await fs.unlink(tmpPath).catch(() => undefined)
    }
  })

  it("resolves extends and merges step records", async () => {
    const config = await loadConfig(path.join(fixtures, "extends-child.jsonc"))
    // Child extends base; base has jq, child has git → merged: both present
    const names = config.steps?.map((s) => s.name).sort()
    expect(names).toEqual(["git", "jq"])
  })

  it("override semantics: child step with same key overrides base step", async () => {
    const config = await loadConfig(path.join(fixtures, "extends-override.jsonc"))
    // Base has jq as { type: "brew", formula: "jq" }
    // Override has jq as { type: "shell", install: "echo override", check: "false" }
    // defu defaults: child wins on key collision
    const jq = config.steps?.find((s) => s.name === "jq")
    expect(jq?.type).toBe("shell")
    if (jq?.type === "shell") {
      expect(jq.install).toBe("echo override")
    }
  })

  it("works with a config path that has no extension", async () => {
    // c12's auto-extension resolution should find single-brew.jsonc when given just "single-brew"
    const config = await loadConfig(path.join(fixtures, "single-brew"))
    expect(config.name).toBe("single-brew")
  })
})
```

- [ ] **Step 5: Run loader tests**

```bash
bun test src/config/load.test.ts
```
Expected: 9 tests pass.

- [ ] **Step 6: Run full suite**

```bash
bun test
bun run typecheck
```
All green; typecheck exit 0.

- [ ] **Step 7: Commit**

```bash
git add src/config/load.ts src/config/load.test.ts tests/fixtures/extends-*.jsonc package.json bun.lock
git commit -m "feat(config): use c12 for extends + deep-merge; add extends fixtures and tests"
```

---

## Task 3: Add pathe dep + XDG path helpers

Tiny self-contained module that resolves the log directory and timestamped filename. We use `pathe` for cross-platform path normalization (it's part of the unjs ecosystem and a one-line dep).

**Files:**
- Modify: `package.json`
- Create: `src/log/xdg.ts`, `src/log/xdg.test.ts`

- [ ] **Step 1: Add pathe**

```bash
bun add pathe@^1.1.2
```

- [ ] **Step 2: Write failing test — `src/log/xdg.test.ts`**

```ts
import { describe, it, expect } from "bun:test"
import { logDir, timestampedFilename, logFilePath } from "./xdg"

describe("logDir", () => {
  it("uses $XDG_STATE_HOME when set", () => {
    const result = logDir({ XDG_STATE_HOME: "/var/lib/state", HOME: "/Users/test" })
    expect(result).toBe("/var/lib/state/gearup/logs")
  })

  it("falls back to $HOME/.local/state when XDG_STATE_HOME is unset", () => {
    const result = logDir({ HOME: "/Users/test" })
    expect(result).toBe("/Users/test/.local/state/gearup/logs")
  })

  it("throws when neither XDG_STATE_HOME nor HOME is set", () => {
    expect(() => logDir({})).toThrow(/HOME/)
  })
})

describe("timestampedFilename", () => {
  it("formats the filename as YYYYMMDD-HHMMSS-<name>.log", () => {
    const fixed = new Date("2026-04-15T21:15:27Z")
    const result = timestampedFilename("Backend", fixed)
    // The format is local time in Go gearup; we use UTC components for stability in tests.
    // The test uses UTC by reading UTC accessors. Production code uses local accessors.
    // Just assert the shape:
    expect(result).toMatch(/^\d{8}-\d{6}-Backend\.log$/)
  })
})

describe("logFilePath", () => {
  it("composes logDir + timestampedFilename", () => {
    const fixed = new Date("2026-04-15T21:15:27Z")
    const result = logFilePath("Backend", { HOME: "/Users/test" }, fixed)
    expect(result).toMatch(
      /^\/Users\/test\/\.local\/state\/gearup\/logs\/\d{8}-\d{6}-Backend\.log$/,
    )
  })
})
```

- [ ] **Step 3: Run (FAIL)**

```bash
bun test src/log/xdg.test.ts
```

- [ ] **Step 4: Implement `src/log/xdg.ts`**

```ts
import { join } from "pathe"

const APP_NAME = "gearup"
const LOGS_SUBPATH = `${APP_NAME}/logs`

export function logDir(env: Record<string, string | undefined>): string {
  if (env.XDG_STATE_HOME) {
    return join(env.XDG_STATE_HOME, LOGS_SUBPATH)
  }
  if (env.HOME) {
    return join(env.HOME, ".local/state", LOGS_SUBPATH)
  }
  throw new Error("logDir: neither XDG_STATE_HOME nor HOME is set in the environment")
}

const pad = (n: number) => String(n).padStart(2, "0")

export function timestampedFilename(configName: string, now: Date = new Date()): string {
  const ts =
    `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}` +
    `-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  return `${ts}-${configName}.log`
}

export function logFilePath(
  configName: string,
  env: Record<string, string | undefined>,
  now: Date = new Date(),
): string {
  return join(logDir(env), timestampedFilename(configName, now))
}
```

- [ ] **Step 5: Run (PASS — 5 tests)**

```bash
bun test src/log/xdg.test.ts
```

- [ ] **Step 6: Commit**

```bash
git add src/log/xdg.ts src/log/xdg.test.ts package.json bun.lock
git commit -m "feat(log): add XDG path helpers and pathe dep"
```

---

## Task 4: Logger interface + FakeLogger

The Logger contract that handlers, runner, and the LoggingExec decorator depend on. FakeLogger is the in-memory test double.

**Files:**
- Create: `src/log/types.ts`, `src/log/fake.ts`, `src/log/fake.test.ts`

- [ ] **Step 1: Write failing test — `src/log/fake.test.ts`**

```ts
import { describe, it, expect } from "bun:test"
import { FakeLogger } from "./fake"

describe("FakeLogger", () => {
  it("records each line in the order they're written", () => {
    const log = new FakeLogger()
    log.log("first")
    log.log("second")
    log.log("third")

    expect(log.lines).toEqual(["first", "second", "third"])
  })

  it("returns its synthetic path string", () => {
    const log = new FakeLogger("/fake/path.log")
    expect(log.path()).toBe("/fake/path.log")
  })

  it("close() resolves and is idempotent", async () => {
    const log = new FakeLogger()
    await log.close()
    await log.close()
    expect(log.closed).toBe(true)
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/log/fake.test.ts
```

- [ ] **Step 3: Implement `src/log/types.ts`**

```ts
export interface Logger {
  /** Append a line of text to the log. The implementation adds the trailing newline. */
  log(line: string): void
  /** Path to the underlying log destination, for printing on failure. */
  path(): string
  /** Flush and close. Idempotent. */
  close(): Promise<void>
}
```

- [ ] **Step 4: Implement `src/log/fake.ts`**

```ts
import type { Logger } from "./types"

export class FakeLogger implements Logger {
  lines: string[] = []
  closed = false

  constructor(private syntheticPath: string = "/fake/log") {}

  log(line: string): void {
    if (this.closed) {
      throw new Error("FakeLogger: log() called after close()")
    }
    this.lines.push(line)
  }

  path(): string {
    return this.syntheticPath
  }

  async close(): Promise<void> {
    this.closed = true
  }
}
```

- [ ] **Step 5: Run (PASS — 3 tests)**

```bash
bun test src/log/fake.test.ts
```

- [ ] **Step 6: Commit**

```bash
git add src/log/types.ts src/log/fake.ts src/log/fake.test.ts
git commit -m "feat(log): add Logger interface and FakeLogger"
```

---

## Task 5: FileLogger (Bun writer-backed)

Production logger. Creates the log directory if missing, opens a Bun file writer, appends lines, closes cleanly.

**Files:**
- Create: `src/log/file.ts`, `src/log/file.test.ts`

- [ ] **Step 1: Write failing test — `src/log/file.test.ts`**

```ts
import { describe, it, expect, afterEach } from "bun:test"
import fs from "node:fs/promises"
import path from "node:path"
import { FileLogger, openFileLogger } from "./file"

const tmpDir = path.join("/tmp", `gearup-file-logger-test-${process.pid}`)

afterEach(async () => {
  await fs.rm(tmpDir, { recursive: true, force: true })
})

describe("FileLogger", () => {
  it("creates the parent directory if it doesn't exist and writes lines to the file", async () => {
    const filePath = path.join(tmpDir, "subdir", "test.log")
    const logger = await openFileLogger(filePath)

    logger.log("first")
    logger.log("second")
    await logger.close()

    const contents = await fs.readFile(filePath, "utf8")
    expect(contents).toBe("first\nsecond\n")
  })

  it("path() returns the file path", async () => {
    const filePath = path.join(tmpDir, "p.log")
    const logger = await openFileLogger(filePath)
    try {
      expect(logger.path()).toBe(filePath)
    } finally {
      await logger.close()
    }
  })

  it("close() is idempotent", async () => {
    const filePath = path.join(tmpDir, "i.log")
    const logger = await openFileLogger(filePath)
    await logger.close()
    await logger.close()  // must not throw
    expect(true).toBe(true)
  })

  it("log() after close() throws", async () => {
    const filePath = path.join(tmpDir, "afterclose.log")
    const logger = await openFileLogger(filePath)
    await logger.close()
    expect(() => logger.log("nope")).toThrow(/closed/)
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/log/file.test.ts
```

- [ ] **Step 3: Implement `src/log/file.ts`**

```ts
import fs from "node:fs/promises"
import path from "node:path"
import type { Logger } from "./types"

/**
 * FileLogger writes lines to a file using Bun.file's writer. Buffered, not streamed —
 * each `log()` call accumulates in the writer's internal buffer, flushed on close().
 */
export class FileLogger implements Logger {
  private writer: ReturnType<ReturnType<typeof Bun.file>["writer"]> extends infer W ? W : never
  private closed = false

  constructor(private filePath: string) {
    this.writer = Bun.file(filePath).writer()
  }

  log(line: string): void {
    if (this.closed) {
      throw new Error(`FileLogger: log() called after closed (${this.filePath})`)
    }
    this.writer.write(`${line}\n`)
  }

  path(): string {
    return this.filePath
  }

  async close(): Promise<void> {
    if (this.closed) return
    this.closed = true
    await this.writer.end()
  }
}

/**
 * Convenience: ensure the parent directory exists, then open a FileLogger.
 */
export async function openFileLogger(filePath: string): Promise<FileLogger> {
  await fs.mkdir(path.dirname(filePath), { recursive: true })
  return new FileLogger(filePath)
}
```

- [ ] **Step 4: Run (PASS — 4 tests)**

```bash
bun test src/log/file.test.ts
```

If `Bun.file().writer()`'s type isn't directly assignable to a stable type (Bun's API surface may use private types), simplify the field's annotation:

```ts
private writer: { write(s: string): number; end(): Promise<number> }
```

(That's the structural shape we actually use.) Adjust if the strict type fails.

- [ ] **Step 5: Commit**

```bash
git add src/log/file.ts src/log/file.test.ts
git commit -m "feat(log): add FileLogger backed by Bun.file writer"
```

---

## Task 6: LoggingExec decorator

Wraps any `Exec` with logging. Records each subprocess invocation (argv, exit code, duration, stdout, stderr) to the supplied Logger before returning the result.

**Files:**
- Create: `src/exec/logging.ts`, `src/exec/logging.test.ts`

- [ ] **Step 1: Write failing test — `src/exec/logging.test.ts`**

```ts
import { describe, it, expect } from "bun:test"
import { LoggingExec } from "./logging"
import { FakeExec } from "./fake"
import { FakeLogger } from "../log/fake"

describe("LoggingExec", () => {
  it("logs argv, exit code, duration, stdout, stderr around each call", async () => {
    const inner = new FakeExec()
    inner.queueResponse({
      exitCode: 0,
      stdout: "hello\nworld",
      stderr: "warning",
      durationMs: 42,
    })

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    const result = await exec.run({ argv: ["echo", "hi"] })

    expect(result.exitCode).toBe(0)
    expect(inner.calls).toHaveLength(1)

    // Lines should include the command, exit/duration, stdout block, stderr block.
    const joined = log.lines.join("\n")
    expect(joined).toContain("> echo hi")
    expect(joined).toContain("(exit 0")
    expect(joined).toContain("hello")
    expect(joined).toContain("world")
    expect(joined).toContain("warning")
  })

  it("does not log stdout/stderr blocks when they are empty", async () => {
    const inner = new FakeExec()
    inner.queueResponse({ exitCode: 0, stdout: "", stderr: "", durationMs: 5 })

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    await exec.run({ argv: ["true"] })

    const joined = log.lines.join("\n")
    expect(joined).toContain("> true")
    expect(joined).toContain("(exit 0, 5ms)")
    expect(joined).not.toContain("stdout:")
    expect(joined).not.toContain("stderr:")
  })

  it("logs even when the inner exec throws (and rethrows)", async () => {
    const inner = new FakeExec()  // empty queue → next run() throws

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    await expect(exec.run({ argv: ["whatever"] })).rejects.toThrow(/unexpected call/)
    const joined = log.lines.join("\n")
    expect(joined).toContain("> whatever")
    expect(joined).toContain("threw:")
  })

  it("annotates timed-out runs", async () => {
    const inner = new FakeExec()
    inner.queueResponse({ exitCode: 124, stdout: "", stderr: "killed", timedOut: true, durationMs: 60_000 })

    const log = new FakeLogger()
    const exec = new LoggingExec(inner, log)

    await exec.run({ argv: ["sleep", "infinity"], timeout: 60_000 })
    const joined = log.lines.join("\n")
    expect(joined).toContain("(timed out)")
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/exec/logging.test.ts
```

- [ ] **Step 3: Implement `src/exec/logging.ts`**

```ts
import type { Exec, RunInput, RunResult } from "./types"
import type { Logger } from "../log/types"

export class LoggingExec implements Exec {
  constructor(
    private readonly inner: Exec,
    private readonly logger: Logger,
  ) {}

  async run(input: RunInput): Promise<RunResult> {
    this.logger.log(`> ${input.argv.join(" ")}${input.shell ? "  (shell)" : ""}`)
    let result: RunResult
    try {
      result = await this.inner.run(input)
    } catch (err) {
      this.logger.log(`threw: ${err instanceof Error ? err.message : String(err)}`)
      this.logger.log("")
      throw err
    }

    const timeoutSuffix = result.timedOut ? "  (timed out)" : ""
    this.logger.log(`(exit ${result.exitCode}, ${result.durationMs}ms)${timeoutSuffix}`)
    if (result.stdout) {
      this.logger.log("stdout:")
      this.logger.log(result.stdout)
    }
    if (result.stderr) {
      this.logger.log("stderr:")
      this.logger.log(result.stderr)
    }
    this.logger.log("")
    return result
  }
}
```

- [ ] **Step 4: Run (PASS — 4 tests)**

```bash
bun test src/exec/logging.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/exec/logging.ts src/exec/logging.test.ts
git commit -m "feat(exec): add LoggingExec decorator (records each subprocess call)"
```

---

## Task 7: Add `log` to Context

Threads the Logger through the runtime. `makeContext` defaults to a no-op logger so existing tests don't break.

**Files:**
- Modify: `src/context.ts`, `src/context.test.ts`

- [ ] **Step 1: Modify `src/context.ts` to add `log: Logger`**

Replace the file with:

```ts
import type { Exec } from "./exec/types"
import type { Logger } from "./log/types"
import { FakeLogger } from "./log/fake"

export type Context = {
  exec: Exec
  cwd: string
  env: Record<string, string | undefined>
  log: Logger
}

type MakeContextInput = {
  exec: Exec
  cwd?: string
  env?: Record<string, string | undefined>
  log?: Logger
}

export function makeContext(input: MakeContextInput): Context {
  return {
    exec: input.exec,
    cwd: input.cwd ?? process.cwd(),
    env: input.env ?? process.env,
    log: input.log ?? new FakeLogger("/dev/null"),
  }
}
```

(Note: `env` type is also widened to `Record<string, string | undefined>` here — a deferred Phase 1 cleanup that's free to fold in now.)

- [ ] **Step 2: Update `src/context.test.ts`**

Replace:

```ts
import { describe, it, expect } from "bun:test"
import { makeContext } from "./context"
import { FakeExec } from "./exec/fake"
import { FakeLogger } from "./log/fake"

describe("makeContext", () => {
  it("defaults cwd to process.cwd(), env to process.env, log to a no-op FakeLogger", () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    expect(ctx.cwd).toBe(process.cwd())
    expect(ctx.env).toBe(process.env)
    expect(ctx.exec).toBe(exec)
    expect(typeof ctx.log.log).toBe("function")
    expect(typeof ctx.log.path).toBe("function")
  })

  it("accepts explicit overrides", () => {
    const exec = new FakeExec()
    const log = new FakeLogger("/tmp/explicit.log")
    const ctx = makeContext({
      exec,
      cwd: "/tmp",
      env: { FOO: "bar" },
      log,
    })

    expect(ctx.cwd).toBe("/tmp")
    expect(ctx.env).toEqual({ FOO: "bar" })
    expect(ctx.log).toBe(log)
  })
})
```

- [ ] **Step 3: Run full suite**

```bash
bun test
bun run typecheck
```
Expected: all tests pass. The `env` type widening from `Record<string, string>` to `Record<string, string | undefined>` may surface call sites that assume non-undefined values; `git-clone.ts` already uses `ctx.env.HOME ?? ""` so it's fine. If anything else fails to typecheck, fix with a `?? "<default>"` or a guard.

- [ ] **Step 4: Commit**

```bash
git add src/context.ts src/context.test.ts
git commit -m "feat: add Logger to Context; widen env to string | undefined"
```

---

## Task 8: Wire FileLogger + LoggingExec into the run command

Production assembly: open a FileLogger at the XDG path, wrap ExecaExec in LoggingExec, pass both to Context, close the logger in `finally`. Print the log path on failure.

**Files:**
- Modify: `src/commands/run.ts`
- Modify: `src/commands/run.test.ts`

- [ ] **Step 1: Update `src/commands/run.ts`**

Replace the file with:

```ts
import { defineCommand } from "citty"
import { loadConfig } from "../config/load"
import { runInstall, type InstallStepReport } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"
import { LoggingExec } from "../exec/logging"
import { openFileLogger } from "../log/file"
import { logFilePath } from "../log/xdg"

export const runCommand = defineCommand({
  meta: {
    name: "run",
    description: "Install all configured tools (running each step's install if not already installed)",
  },
  args: {
    config: {
      type: "string",
      description: "Path to config file (JSONC, YAML, or TOML)",
      required: true,
    },
  },
  async run({ args }) {
    const config = await loadConfig(args.config)
    const path = logFilePath(config.name, process.env)
    const logger = await openFileLogger(path)
    const exec = new LoggingExec(new ExecaExec(), logger)
    const ctx = makeContext({ exec, log: logger })

    try {
      const report = await runInstall(config, ctx)

      printReport(report.configName, report.steps)

      if (!report.ok) {
        console.error("")
        console.error(`✗ Failed at step: ${report.failedAt}`)
        console.error(`  ${report.error}`)
        console.error("")
        console.error(`Log: ${logger.path()}`)
        return 1
      }

      console.log("")
      console.log("Done.")
      console.log(`Log: ${logger.path()}`)
      return 0
    } finally {
      await logger.close()
    }
  },
})

function printReport(configName: string, steps: InstallStepReport[]): void {
  console.log(`CONFIG: ${configName}  (${steps.length} step${steps.length === 1 ? "" : "s"})`)
  console.log("")
  steps.forEach((step, i) => {
    const idx = `[${i + 1}/${steps.length}]`
    const marker = "✓"  // both skipped and installed are successful outcomes
    const label = step.action === "skipped" ? "already installed" : "installed"
    console.log(`  ${marker} ${idx} ${step.name}  ${label}`)
  })
}
```

- [ ] **Step 2: Update `src/commands/run.test.ts`**

The existing tests construct ad-hoc fixture files and run the command. They must still work but the command now ALSO writes a real log file under `$XDG_STATE_HOME/gearup/logs/` (or `~/.local/state/gearup/logs/`). To avoid polluting the user's filesystem during tests, set `XDG_STATE_HOME` to a tmp dir within each test.

Replace the file with:

```ts
import { describe, it, expect, spyOn, mock, afterEach } from "bun:test"

mock.module("@clack/prompts", () => ({
  confirm: mock(async () => true),
  isCancel: () => false,
  intro: mock(),
  outro: mock(),
  note: mock(),
}))

import { runCommand } from "citty"
import path from "node:path"
import fs from "node:fs/promises"
import { runCommand as gearupRunCommand } from "./run"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")
const tmpStateDir = path.join("/tmp", `gearup-run-cmd-test-${process.pid}`)
const originalXdgState = process.env.XDG_STATE_HOME

process.env.XDG_STATE_HOME = tmpStateDir

afterEach(async () => {
  await fs.rm(tmpStateDir, { recursive: true, force: true })
})

describe("run command", () => {
  it("returns 0 on a successful run", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    const tmpMarker = "/tmp/gearup-test-run-success"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__run-success.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "run-success",
        steps: {
          marker: {
            type: "shell",
            install: `touch ${tmpMarker}`,
            check: `test -f ${tmpMarker}`,
          },
        },
      }),
    )

    try {
      const { result } = await runCommand(gearupRunCommand, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(0)
      const output = logSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(output).toContain("run-success")
      expect(output).toContain("marker")
      expect(output).toContain("Log:")  // log path printed on success
    } finally {
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }
  })

  it("returns 1 on a failed run and prints the log path", async () => {
    const errSpy = spyOn(console, "error").mockImplementation(() => undefined)
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)

    const fixturePath = path.join(fixtures, "__run-fail.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "run-fail",
        steps: {
          "always-fails": {
            type: "shell",
            install: "false",
            check: "false",
          },
        },
      }),
    )

    try {
      const { result } = await runCommand(gearupRunCommand, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(1)
      const errOutput = errSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(errOutput).toContain("Failed at step")
      expect(errOutput).toContain("always-fails")
      expect(errOutput).toContain("Log:")
    } finally {
      errSpy.mockRestore()
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
    }
  })

  it("creates a real log file containing the subprocess output", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    const tmpMarker = "/tmp/gearup-test-real-log"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__run-log-content.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "real-log",
        steps: {
          marker: {
            type: "shell",
            install: `touch ${tmpMarker}`,
            check: `test -f ${tmpMarker}`,
          },
        },
      }),
    )

    try {
      await runCommand(gearupRunCommand, { rawArgs: ["--config", fixturePath] })
    } finally {
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }

    // Verify a log file exists under tmpStateDir/gearup/logs/ and contains the touch invocation.
    const logsDir = path.join(tmpStateDir, "gearup", "logs")
    const entries = await fs.readdir(logsDir)
    expect(entries.length).toBeGreaterThanOrEqual(1)
    const logContent = await fs.readFile(path.join(logsDir, entries[0]!), "utf8")
    expect(logContent).toContain(`touch ${tmpMarker}`)
    expect(logContent).toContain("(exit")
  })
})
```

- [ ] **Step 3: Run the command tests**

```bash
bun test src/commands/run.test.ts
```
Expected: 3 tests pass.

- [ ] **Step 4: Run full suite**

```bash
bun test
bun run typecheck
```
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add src/commands/run.ts src/commands/run.test.ts
git commit -m "feat(cli): wire FileLogger and LoggingExec into run; print log path on success/failure"
```

---

## Task 9: E2E test for `extends` and logging end-to-end

Verifies the full pipeline against a real subprocess: extends fixture loaded, step from base + step from child both run, log file written under the user's actual XDG state path (or a redirected one for test safety).

**Files:**
- Modify: `tests/e2e.test.ts`

- [ ] **Step 1: Append to `tests/e2e.test.ts`**

Append (the import for `fs` is already there from Phase 2):

```ts
describe("e2e: extends + logging", () => {
  const tmpStateDir = path.join("/tmp", `gearup-e2e-extends-${process.pid}`)
  const tmpMarkerJq = "/tmp/gearup-e2e-extends-jq"
  const tmpMarkerGit = "/tmp/gearup-e2e-extends-git"

  it("loads a config that extends another, runs steps from both, writes a log file", async () => {
    // Write a base + child fixture pair where steps are safe shell commands
    const baseFixture = path.join(fixtures, "__e2e-extends-base.jsonc")
    const childFixture = path.join(fixtures, "__e2e-extends-child.jsonc")

    await fs.writeFile(
      baseFixture,
      JSON.stringify({
        version: 1,
        name: "e2e-extends-base",
        steps: {
          "fake-jq": {
            type: "shell",
            install: `touch ${tmpMarkerJq}`,
            check: `test -f ${tmpMarkerJq}`,
          },
        },
      }),
    )
    await fs.writeFile(
      childFixture,
      JSON.stringify({
        version: 1,
        name: "e2e-extends-child",
        extends: ["./__e2e-extends-base"],
        steps: {
          "fake-git": {
            type: "shell",
            install: `touch ${tmpMarkerGit}`,
            check: `test -f ${tmpMarkerGit}`,
          },
        },
      }),
    )

    // Clean state
    await fs.unlink(tmpMarkerJq).catch(() => undefined)
    await fs.unlink(tmpMarkerGit).catch(() => undefined)
    await fs.rm(tmpStateDir, { recursive: true, force: true })

    try {
      const result = await $`XDG_STATE_HOME=${tmpStateDir} bun run ${path.join(repoRoot, "src/cli.ts")} run --config ${childFixture}`.quiet().nothrow()

      expect(result.exitCode).toBe(0)
      expect(result.stdout.toString()).toContain("e2e-extends-child")
      expect(result.stdout.toString()).toContain("fake-jq")
      expect(result.stdout.toString()).toContain("fake-git")

      // Both side effects happened
      await fs.access(tmpMarkerJq)
      await fs.access(tmpMarkerGit)

      // Log file exists under the redirected XDG state dir
      const logsDir = path.join(tmpStateDir, "gearup", "logs")
      const entries = await fs.readdir(logsDir)
      expect(entries.length).toBeGreaterThanOrEqual(1)
      const logContent = await fs.readFile(path.join(logsDir, entries[0]!), "utf8")
      expect(logContent).toContain(`touch ${tmpMarkerJq}`)
      expect(logContent).toContain(`touch ${tmpMarkerGit}`)
    } finally {
      await fs.unlink(baseFixture).catch(() => undefined)
      await fs.unlink(childFixture).catch(() => undefined)
      await fs.unlink(tmpMarkerJq).catch(() => undefined)
      await fs.unlink(tmpMarkerGit).catch(() => undefined)
      await fs.rm(tmpStateDir, { recursive: true, force: true })
    }
  })
})
```

- [ ] **Step 2: Run e2e tests**

```bash
bun test tests/e2e.test.ts
```
Expected: all e2e tests pass (Phase 1 + Phase 2 + this new one).

- [ ] **Step 3: Run full suite (final acceptance)**

```bash
bun test
bun run typecheck
```
All green; typecheck exit 0. Test count target: 110+.

- [ ] **Step 4: Commit**

```bash
git add tests/e2e.test.ts
git commit -m "test: e2e for extends composition + log file content"
```

---

## Task 10: Update spec and architecture docs to reflect Phase 3 changes

Schema and behavior changed in Phase 3. The spec must follow.

**Files:**
- Modify: `docs/superpowers/specs/2026-04-24-typescript-port-design.md`
- Modify: `docs/ts-port-architecture.md`

- [ ] **Step 1: Update the spec**

Find the **Data model** section in `docs/superpowers/specs/2026-04-24-typescript-port-design.md`. The schema example currently shows steps as an array. Replace the relevant block with the Record-keyed shape and explain the override semantics:

```ts
// src/schema/step.ts
import * as v from "valibot"

const baseFields = {
  check: v.optional(v.string()),
  requires_elevation: v.optional(v.boolean()),
  post_install: v.optional(v.array(v.string())),
}
// Step bodies do NOT include `name` — that becomes the Record key.
export const brewStepBody = v.object({ type: v.literal("brew"), ...baseFields, formula: v.string() })
// ... (other variants)
export const stepBody = v.variant("type", [brewStepBody, brewCaskStepBody, ...])

// In the config schema, steps is a Record<name, body>; a transform injects `name` from
// the key so internal code sees the same Step[] shape as before.
const stepsRecord = v.pipe(
  v.record(v.string(), stepBody),
  v.transform((rec) => Object.entries(rec).map(([name, body]) => ({ name, ...body })))
)
```

Then add a paragraph above or below explaining:

> **Why Record-keyed?** When configs compose via `extends:`, c12+defu naturally deep-merge two records keyed by the same string (current overrides extended on collision). With an array shape, we'd need a custom merger to dedup-and-override. The Record shape is also more explicit about step name uniqueness (the schema can't even represent two steps with the same name in a single config) and produces clearer validation error paths (`steps.Homebrew.formula` vs `steps[3].formula`).

Find the **Phasing** table — Phase 3 row. The current Scope says:

```
`extends:` composition (c12's layered merge + a name→path pre-resolver) • XDG logging with captured subprocess output • inline failure printing
```

Replace with:

```
`extends:` composition via c12 (paths, packages, github: refs) • Record-keyed steps schema (drops the dedup-merger problem) • `sources:` field removed in favor of explicit paths • curl-pipe-sh schema hardening (URL/shell/args validation) • XDG logging via FileLogger + LoggingExec decorator with captured subprocess output and inline failure printing
```

Find the **Open questions deferred to implementation** section. Mark `extends:` and the curl-pipe-sh hardening as resolved. If there's an "extends name resolution rules" entry, replace its body with:

```
**Resolved.** Phase 3 dropped Go's name-based search (`extends: [base]`) in favor of c12-native references (`extends: ["./base"]`, `extends: ["github:owner/repo"]`, etc.). The `./` prefix is required for local files; extension is auto-resolved. The `sources:` field was removed — paths are explicit, including absolute paths and remote refs.
```

If any references to `readline` for elevation remain (Phase 2 already updated them), leave them.

- [ ] **Step 2: Update the architecture doc**

In `docs/ts-port-architecture.md`, find the **Locked tech stack** table. Update the row for `Config loader` to reflect that c12's role is now active:

```
| Config loader | **c12 + confbox** (loads JSONC/YAML/TOML; resolves `extends:` recursively; deep-merges with defu) |
```

Find the **Phasing** table — Phase 3 row. Update similarly to the spec.

In the **Why this shape vs alternatives** section (or wherever it fits), add a note:

```
- **Why Record-keyed steps?** Composition via `extends:` works naturally with c12+defu when steps are a Record keyed by name: the deep-merge collapses same-keyed entries with current-config-wins semantics. An array shape would have required a custom merger to dedup. The Record form is also stricter (can't express duplicates in a single config) and gives better error paths. The internal `Step[]` view is recovered via a Valibot transform that injects `name` from the key.
```

- [ ] **Step 3: Verify nothing else references stale schema**

```bash
grep -rn "sources:\|name-based extends\|first-wins" docs/
```

Anything that pops up that's not the plan file or this Phase 3 plan should be updated to match the new behavior.

- [ ] **Step 4: Commit**

```bash
git add docs/
git commit -m "docs: reflect Phase 3 schema redesign, c12 extends, and XDG logging"
```

---

## Phase 3 Done — Verify and Push

- [ ] **Step 1: Final verification**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun test
bun run typecheck
git log --oneline main..HEAD | wc -l    # ~10 commits
```

- [ ] **Step 2: Smoke-test the new behavior end-to-end**

```bash
# A standalone run still works (no extends)
bun run src/cli.ts run --config tests/fixtures/safe-install.jsonc; echo $?
rm -f /tmp/gearup-e2e-marker /tmp/gearup-e2e-post

# Extends composition works
bun run src/cli.ts plan --config tests/fixtures/extends-child.jsonc; echo $?
# Should show steps from BOTH extends-base (jq) and extends-child (git).

# Plan command shows expected output
bun run src/cli.ts plan --config tests/fixtures/single-brew.jsonc; echo $?
```

- [ ] **Step 3: Push and create PR**

```bash
git push -u origin ts-port/phase-3-extends-and-logging
gh pr create --title "feat: TS port Phase 3 — extends, logging, schema redesign" --body "..."
```

(Construct the PR body summarizing: schema redesign to Record-keyed steps, c12 integration for extends, XDG file logging, curl-pipe-sh hardening, removed `sources:` field, updated docs. Test plan should cover plan/run against single-file and extends fixtures plus log file inspection.)

---

## Exit Criteria for Phase 3

1. `gearup plan --config <child>` and `gearup run --config <child>` both work end-to-end against a config that uses `extends: ["./base"]`. Steps from both files are present in the report.
2. Override semantics: a step in the child with the same key as one in the base wins (defu defaults).
3. `gearup run` writes a log file under `$XDG_STATE_HOME/gearup/logs/<timestamp>-<configname>.log` (or `~/.local/state/gearup/logs/...` fallback) containing every subprocess invocation, exit code, duration, and captured stdio.
4. On failure, `gearup run` prints `✗ Failed at step: <name>` followed by the error and `Log: <path>`.
5. `curl-pipe-sh` schema rejects: non-URL strings for `url`, disallowed shells (anything outside `bash`/`sh`/`zsh`/`fish`), args containing whitespace within an element.
6. The `sources:` field is no longer accepted by the schema (silently dropped, since Valibot's `v.object` is non-strict).
7. Test count grows to 110+; typecheck exit 0.
8. `c12` and `pathe` are real, used dependencies.

## What's NOT in Phase 3

- Interactive picker when `--config` omitted — Phase 4
- Animated spinners during install — Phase 4
- Styled `intro/outro` flow — Phase 4
- `gearup init` + embedded default configs — Phase 4
- Release pipeline (`bun build --compile`) — Phase 4
- Go code removal — Phase 4
- Streamed (vs buffered) subprocess logging — possible future enhancement; not needed for Phase 3 UX
