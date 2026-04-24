# TS Port — Phase 1 Tracer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port gearup's `plan` command (dry-run verification) from Go to TypeScript on Bun. End-to-end pipeline (argv → config load → schema validate → dispatch check handlers → print report → exit 0/10) against single-file configs in JSONC / YAML / TOML. No installation, no elevation, no logging, no picker.

**Architecture:** One-directional pipeline with replaceable layers. argv → citty → c12+confbox → Valibot → runner → handler registry → Exec (execa wrapped behind an interface). All I/O through a `Context` object that carries the `Exec`. Handlers are one file per step type, gathered in a registry with `satisfies` for compile-time completeness.

**Tech Stack:** Bun, TypeScript, citty (CLI), c12 + confbox (config loader), Valibot (schema), execa (subprocess), bun:test (test runner).

---

## Preflight

This plan adds TypeScript code alongside the existing Go code. The Go codebase is not touched. Delete Go is a Phase 4 task once the TS port reaches full parity.

**Before Task 1, create a worktree for this work:**

```bash
git worktree add ../gearup-ts-phase1 -b ts-port/phase-1-tracer
cd ../gearup-ts-phase1
```

All subsequent tasks assume you are in the worktree directory. The branch merges back to `main` at the end of Phase 1.

**Reference files from Go that Phase 1 mirrors:**
- `internal/config/config.go` — the `Step` struct (canonical field list)
- `internal/installer/brew/brew.go` — brew check default (`brew list --formula <formula>`)
- `internal/installer/brewcask/brewcask.go` — brew-cask check default (`brew list --cask <cask>`)
- `internal/installer/curlpipe/curlpipe.go` — curl-pipe check (requires explicit `check:`)
- `internal/installer/gitclone/gitclone.go` — git-clone check default (destination directory exists)
- `internal/installer/shell/shell.go` — shell check (requires explicit `check:`)

Use these as the source of truth when a behavior question comes up.

---

## File Structure (what gets created in Phase 1)

```
gearup/
├── package.json                 NEW — deps + scripts
├── tsconfig.json                NEW
├── bunfig.toml                  NEW
├── .gitignore                   MODIFY — add node_modules, bin/
├── src/
│   ├── cli.ts                   citty mainCommand, wires plan + version subcommands
│   ├── context.ts               Context type + makeContext() factory
│   ├── commands/
│   │   ├── plan.ts              plan command definition (citty defineCommand)
│   │   ├── plan.test.ts         integration test via runCommand
│   │   └── version.ts           trivial version command
│   ├── config/
│   │   ├── load.ts              c12 + confbox → validated Config
│   │   └── load.test.ts
│   ├── schema/
│   │   ├── step.ts              per-type step schemas + v.variant union
│   │   ├── config.ts            top-level config schema
│   │   ├── index.ts             re-exports + inferred types
│   │   └── schema.test.ts
│   ├── runner/
│   │   ├── run.ts               iterate steps, dispatch check via registry, return PlanReport
│   │   └── run.test.ts
│   ├── steps/
│   │   ├── types.ts             CheckResult, Handler<S>
│   │   ├── brew.ts              + brew.test.ts
│   │   ├── brew-cask.ts         + brew-cask.test.ts
│   │   ├── curl-pipe-sh.ts      + curl-pipe-sh.test.ts
│   │   ├── git-clone.ts         + git-clone.test.ts
│   │   ├── shell.ts             + shell.test.ts
│   │   └── index.ts             handlers registry with `satisfies`
│   └── exec/
│       ├── types.ts             Exec interface, RunInput, RunResult
│       ├── execa.ts             ExecaExec
│       ├── fake.ts              FakeExec
│       └── exec.test.ts
└── tests/
    └── fixtures/
        ├── single-brew.jsonc
        ├── single-brew.yaml
        ├── single-brew.toml
        ├── all-five-types.jsonc
        └── all-installed.jsonc
```

---

## Task 1: Project Scaffolding

Bring the TypeScript project to life. Nothing functional yet — just package manifest, TS config, and an empty CLI file that `bun run` can execute.

**Files:**
- Create: `package.json`, `tsconfig.json`, `bunfig.toml`, `src/cli.ts`
- Modify: `.gitignore`

- [ ] **Step 1: Create `package.json`**

```json
{
  "name": "gearup",
  "version": "0.2.0-alpha.1",
  "type": "module",
  "bin": {
    "gearup": "./src/cli.ts"
  },
  "scripts": {
    "dev": "bun run src/cli.ts",
    "test": "bun test",
    "typecheck": "tsc --noEmit",
    "build": "bun build src/cli.ts --compile --outfile=bin/gearup"
  },
  "dependencies": {
    "citty": "^0.1.6",
    "c12": "^2.0.0",
    "confbox": "^0.1.7",
    "valibot": "^0.40.0",
    "execa": "^9.5.0"
  },
  "devDependencies": {
    "typescript": "^5.6.0",
    "@types/bun": "latest"
  }
}
```

- [ ] **Step 2: Create `tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "types": ["bun-types"],
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "allowImportingTsExtensions": true
  },
  "include": ["src/**/*", "scripts/**/*", "tests/**/*"]
}
```

- [ ] **Step 3: Create `bunfig.toml`**

```toml
[test]
preload = []
```

- [ ] **Step 4: Create `src/cli.ts` stub**

```ts
#!/usr/bin/env bun
console.log("gearup (ts): scaffolding in place")
```

- [ ] **Step 5: Update `.gitignore`**

Append to existing `.gitignore`:

```
# TypeScript / Bun
node_modules/
bin/
*.tsbuildinfo
```

- [ ] **Step 6: Install dependencies**

Run: `bun install`
Expected: `bun install` completes; `bun.lockb` is created; `node_modules/` is populated.

- [ ] **Step 7: Smoke-test the stub runs**

Run: `bun run src/cli.ts`
Expected output: `gearup (ts): scaffolding in place`

- [ ] **Step 8: Verify tsc is happy**

Run: `bun run typecheck`
Expected: exits 0 with no output.

- [ ] **Step 9: Commit**

```bash
git add package.json tsconfig.json bunfig.toml src/cli.ts .gitignore bun.lockb
git commit -m "chore: scaffold TypeScript project for port"
```

---

## Task 2: Exec interface + FakeExec

The subprocess seam. Define the interface handlers depend on, and the fake they use in tests. No execa yet — this establishes the shape.

**Files:**
- Create: `src/exec/types.ts`, `src/exec/fake.ts`, `src/exec/exec.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/exec/exec.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { FakeExec } from "./fake"

describe("FakeExec", () => {
  it("records calls and returns queued responses", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0, stdout: "ok" })

    const result = await exec.run({ argv: ["brew", "list", "--formula", "jq"] })

    expect(result.exitCode).toBe(0)
    expect(result.stdout).toBe("ok")
    expect(exec.calls).toHaveLength(1)
    expect(exec.calls[0]?.argv).toEqual(["brew", "list", "--formula", "jq"])
  })

  it("throws when queue is empty", async () => {
    const exec = new FakeExec()
    await expect(exec.run({ argv: ["brew"] })).rejects.toThrow(/unexpected call/)
  })

  it("fills defaults on queued response", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })

    const result = await exec.run({ argv: ["false"] })

    expect(result.exitCode).toBe(1)
    expect(result.stdout).toBe("")
    expect(result.stderr).toBe("")
    expect(result.timedOut).toBe(false)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/exec/exec.test.ts`
Expected: FAIL — `Cannot find module "./fake"`.

- [ ] **Step 3: Implement `src/exec/types.ts`**

```ts
export type RunInput = {
  argv: string[]
  shell?: boolean
  cwd?: string
  env?: Record<string, string>
  stdin?: string
  timeout?: number
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

- [ ] **Step 4: Implement `src/exec/fake.ts`**

```ts
import type { Exec, RunInput, RunResult } from "./types"

export class FakeExec implements Exec {
  calls: RunInput[] = []
  private queue: RunResult[] = []

  queueResponse(r: Partial<RunResult>): void {
    this.queue.push({
      exitCode: 0,
      stdout: "",
      stderr: "",
      durationMs: 0,
      timedOut: false,
      ...r,
    })
  }

  async run(input: RunInput): Promise<RunResult> {
    this.calls.push(input)
    const response = this.queue.shift()
    if (!response) {
      throw new Error(`FakeExec: unexpected call ${input.argv.join(" ")}`)
    }
    return response
  }
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `bun test src/exec/exec.test.ts`
Expected: PASS — 3 tests pass.

- [ ] **Step 6: Commit**

```bash
git add src/exec/types.ts src/exec/fake.ts src/exec/exec.test.ts
git commit -m "feat(exec): add Exec interface and FakeExec"
```

---

## Task 3: ExecaExec — real subprocess implementation

Wire execa behind the `Exec` interface. One contract test proves the real impl honors the interface shape.

**Files:**
- Create: `src/exec/execa.ts`
- Modify: `src/exec/exec.test.ts` (add contract tests)

- [ ] **Step 1: Write the failing test**

Append to `src/exec/exec.test.ts`:

```ts
import { ExecaExec } from "./execa"

describe("ExecaExec", () => {
  it("captures stdout and exit code from a real command", async () => {
    const exec = new ExecaExec()
    const result = await exec.run({ argv: ["/bin/echo", "hello"] })

    expect(result.exitCode).toBe(0)
    expect(result.stdout.trim()).toBe("hello")
    expect(result.stderr).toBe("")
    expect(result.timedOut).toBe(false)
    expect(result.durationMs).toBeGreaterThanOrEqual(0)
  })

  it("reports non-zero exit code without throwing", async () => {
    const exec = new ExecaExec()
    const result = await exec.run({ argv: ["/usr/bin/false"] })

    expect(result.exitCode).toBe(1)
  })

  it("runs through a shell when shell: true", async () => {
    const exec = new ExecaExec()
    const result = await exec.run({ argv: ["echo foo | cat"], shell: true })

    expect(result.exitCode).toBe(0)
    expect(result.stdout.trim()).toBe("foo")
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/exec/exec.test.ts`
Expected: FAIL — `Cannot find module "./execa"`.

- [ ] **Step 3: Implement `src/exec/execa.ts`**

```ts
import { execa } from "execa"
import type { Exec, RunInput, RunResult } from "./types"

export class ExecaExec implements Exec {
  async run(input: RunInput): Promise<RunResult> {
    const start = performance.now()
    const [cmd, ...args] = input.shell
      ? [input.argv.join(" ")]
      : input.argv

    if (!cmd) {
      throw new Error("ExecaExec: empty argv")
    }

    const result = await execa(cmd, args, {
      cwd: input.cwd,
      env: input.env,
      input: input.stdin,
      timeout: input.timeout ?? 60_000,
      shell: input.shell ? "/bin/sh" : false,
      reject: false,
    })

    return {
      exitCode: result.exitCode ?? 0,
      stdout: typeof result.stdout === "string" ? result.stdout : "",
      stderr: typeof result.stderr === "string" ? result.stderr : "",
      durationMs: Math.round(performance.now() - start),
      timedOut: result.timedOut === true,
    }
  }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/exec/exec.test.ts`
Expected: PASS — all 6 tests pass (3 FakeExec + 3 ExecaExec).

- [ ] **Step 5: Commit**

```bash
git add src/exec/execa.ts src/exec/exec.test.ts
git commit -m "feat(exec): implement ExecaExec over the execa library"
```

---

## Task 4: Step schemas (all 5 types + variant)

Valibot schemas for each step type plus the discriminated union. Field set matches Go's `config.Step` struct exactly.

**Files:**
- Create: `src/schema/step.ts`, `src/schema/schema.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/schema/schema.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import * as v from "valibot"
import { step } from "./step"

describe("step schema", () => {
  it("parses a brew step", () => {
    const parsed = v.parse(step, {
      type: "brew",
      name: "jq",
      formula: "jq",
    })
    expect(parsed.type).toBe("brew")
    if (parsed.type === "brew") {
      expect(parsed.formula).toBe("jq")
    }
  })

  it("parses a brew-cask step", () => {
    const parsed = v.parse(step, {
      type: "brew-cask",
      name: "iTerm2",
      cask: "iterm2",
    })
    expect(parsed.type).toBe("brew-cask")
    if (parsed.type === "brew-cask") {
      expect(parsed.cask).toBe("iterm2")
    }
  })

  it("parses a curl-pipe-sh step (check required)", () => {
    const parsed = v.parse(step, {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    })
    expect(parsed.type).toBe("curl-pipe-sh")
  })

  it("rejects curl-pipe-sh without check", () => {
    expect(() =>
      v.parse(step, {
        type: "curl-pipe-sh",
        name: "Homebrew",
        url: "https://example.com/install.sh",
      }),
    ).toThrow()
  })

  it("parses a git-clone step", () => {
    const parsed = v.parse(step, {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    })
    expect(parsed.type).toBe("git-clone")
  })

  it("parses a shell step (check required)", () => {
    const parsed = v.parse(step, {
      type: "shell",
      name: "rust",
      install: "curl ... | sh",
      check: "command -v rustc",
    })
    expect(parsed.type).toBe("shell")
  })

  it("rejects shell step without check", () => {
    expect(() =>
      v.parse(step, {
        type: "shell",
        name: "rust",
        install: "curl ... | sh",
      }),
    ).toThrow()
  })

  it("rejects an unknown step type", () => {
    expect(() =>
      v.parse(step, {
        type: "wat",
        name: "x",
      }),
    ).toThrow()
  })

  it("accepts optional fields: post_install, requires_elevation, check, platform", () => {
    const parsed = v.parse(step, {
      type: "brew",
      name: "colima",
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/schema/schema.test.ts`
Expected: FAIL — `Cannot find module "./step"`.

- [ ] **Step 3: Implement `src/schema/step.ts`**

```ts
import * as v from "valibot"

const platform = v.object({
  os: v.optional(v.array(v.picklist(["darwin", "linux"]))),
  arch: v.optional(v.array(v.string())),
})

const baseFields = {
  name: v.pipe(v.string(), v.minLength(1)),
  check: v.optional(v.string()),
  requires_elevation: v.optional(v.boolean()),
  post_install: v.optional(v.array(v.string())),
  platform: v.optional(platform),
}

export const brewStep = v.object({
  type: v.literal("brew"),
  ...baseFields,
  formula: v.pipe(v.string(), v.minLength(1)),
})

export const brewCaskStep = v.object({
  type: v.literal("brew-cask"),
  ...baseFields,
  cask: v.pipe(v.string(), v.minLength(1)),
})

export const curlPipeStep = v.object({
  type: v.literal("curl-pipe-sh"),
  ...baseFields,
  url: v.pipe(v.string(), v.minLength(1)),
  shell: v.optional(v.string()),
  args: v.optional(v.array(v.string())),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const gitCloneStep = v.object({
  type: v.literal("git-clone"),
  ...baseFields,
  repo: v.pipe(v.string(), v.minLength(1)),
  dest: v.pipe(v.string(), v.minLength(1)),
  ref: v.optional(v.string()),
})

export const shellStep = v.object({
  type: v.literal("shell"),
  ...baseFields,
  install: v.pipe(v.string(), v.minLength(1)),
  check: v.pipe(v.string(), v.minLength(1)),
})

export const step = v.variant("type", [
  brewStep,
  brewCaskStep,
  curlPipeStep,
  gitCloneStep,
  shellStep,
])

export type BrewStep = v.InferOutput<typeof brewStep>
export type BrewCaskStep = v.InferOutput<typeof brewCaskStep>
export type CurlPipeStep = v.InferOutput<typeof curlPipeStep>
export type GitCloneStep = v.InferOutput<typeof gitCloneStep>
export type ShellStep = v.InferOutput<typeof shellStep>
export type Step = v.InferOutput<typeof step>
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/schema/schema.test.ts`
Expected: PASS — 9 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/schema/step.ts src/schema/schema.test.ts
git commit -m "feat(schema): add Valibot step schemas (brew, brew-cask, curl-pipe-sh, git-clone, shell)"
```

---

## Task 5: Config schema + schema index

Top-level config schema wrapping the step union. Export everything through a single `schema/index.ts` entry point.

**Files:**
- Create: `src/schema/config.ts`, `src/schema/index.ts`
- Modify: `src/schema/schema.test.ts` (add config tests)

- [ ] **Step 1: Write the failing test**

Append to `src/schema/schema.test.ts`:

```ts
import { config } from "./config"

describe("config schema", () => {
  it("parses a minimal config", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "base",
    })
    expect(parsed.name).toBe("base")
    expect(parsed.steps).toBeUndefined()
  })

  it("parses a config with steps", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
      ],
    })
    expect(parsed.steps).toHaveLength(1)
  })

  it("parses a config with extends", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "backend",
      extends: ["base", "jvm"],
    })
    expect(parsed.extends).toEqual(["base", "jvm"])
  })

  it("parses a config with elevation", () => {
    const parsed = v.parse(config, {
      version: 1,
      name: "team",
      elevation: {
        message: "Admin please",
        duration: "180s",
      },
    })
    expect(parsed.elevation?.message).toBe("Admin please")
  })

  it("rejects version !== 1", () => {
    expect(() =>
      v.parse(config, { version: 2, name: "x" }),
    ).toThrow()
  })

  it("rejects a config with no name", () => {
    expect(() =>
      v.parse(config, { version: 1 }),
    ).toThrow()
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/schema/schema.test.ts`
Expected: FAIL — `Cannot find module "./config"`.

- [ ] **Step 3: Implement `src/schema/config.ts`**

```ts
import * as v from "valibot"
import { step } from "./step"

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
  sources: v.optional(
    v.array(
      v.object({
        path: v.pipe(v.string(), v.minLength(1)),
      }),
    ),
  ),
  extends: v.optional(v.array(v.string())),
  steps: v.optional(v.array(step)),
})

export type Config = v.InferOutput<typeof config>
```

- [ ] **Step 4: Implement `src/schema/index.ts`**

```ts
export * from "./step"
export * from "./config"
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `bun test src/schema/schema.test.ts`
Expected: PASS — 15 tests total pass (9 step + 6 config).

- [ ] **Step 6: Commit**

```bash
git add src/schema/config.ts src/schema/index.ts src/schema/schema.test.ts
git commit -m "feat(schema): add top-level config schema"
```

---

## Task 6: Context factory

The single object threaded through every handler. Holds the `Exec` plus cwd/env. `makeContext()` fills defaults. Created first so downstream files (`src/steps/types.ts`, handlers) can import `Context` without resolution errors.

**Files:**
- Create: `src/context.ts`, `src/context.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/context.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { makeContext } from "./context"
import { FakeExec } from "./exec/fake"

describe("makeContext", () => {
  it("defaults cwd to process.cwd() and env to process.env", () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    expect(ctx.cwd).toBe(process.cwd())
    expect(ctx.env).toBe(process.env as Record<string, string>)
    expect(ctx.exec).toBe(exec)
  })

  it("accepts explicit overrides", () => {
    const exec = new FakeExec()
    const ctx = makeContext({
      exec,
      cwd: "/tmp",
      env: { FOO: "bar" },
    })

    expect(ctx.cwd).toBe("/tmp")
    expect(ctx.env).toEqual({ FOO: "bar" })
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/context.test.ts`
Expected: FAIL — `Cannot find module "./context"`.

- [ ] **Step 3: Implement `src/context.ts`**

```ts
import type { Exec } from "./exec/types"

export type Context = {
  exec: Exec
  cwd: string
  env: Record<string, string>
}

type MakeContextInput = {
  exec: Exec
  cwd?: string
  env?: Record<string, string>
}

export function makeContext(input: MakeContextInput): Context {
  return {
    exec: input.exec,
    cwd: input.cwd ?? process.cwd(),
    env: input.env ?? (process.env as Record<string, string>),
  }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/context.test.ts`
Expected: PASS — 2 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/context.ts src/context.test.ts
git commit -m "feat: add Context type and makeContext factory"
```

---

## Task 7: CheckResult + Handler types

Shared types that handlers and runner both depend on. Small but load-bearing. Depends on `Context` from Task 6.

**Files:**
- Create: `src/steps/types.ts`

- [ ] **Step 1: Create `src/steps/types.ts`**

```ts
import type { Step } from "../schema"
import type { Context } from "../context"

export type CheckResult =
  | { installed: true }
  | { installed: false; reason?: string }

export type InstallResult =
  | { ok: true }
  | { ok: false; error: string }

export type Handler<S extends Step> = {
  check: (step: S, ctx: Context) => Promise<CheckResult>
  install?: (step: S, ctx: Context) => Promise<InstallResult>
}
```

- [ ] **Step 2: Verify it compiles**

Run: `bun run typecheck`
Expected: exits 0.

- [ ] **Step 3: Commit**

No runtime test for this yet — it's pure types, and the types get exercised immediately in Task 8. The `typecheck` step above verifies the imports resolve.

```bash
git add src/steps/types.ts
git commit -m "feat(steps): add CheckResult, InstallResult, and Handler types"
```

---

## Task 8: brew check handler

Runs `brew list --formula <formula>` by default; if `step.check` is set, runs that in a shell instead. Exit 0 means installed.

**Files:**
- Create: `src/steps/brew.ts`, `src/steps/brew.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/steps/brew.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { checkBrew } from "./brew"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { BrewStep } from "../schema"

describe("checkBrew", () => {
  it("runs the default `brew list --formula` check when step.check is absent", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "jq", formula: "jq" }
    const result = await checkBrew(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls).toHaveLength(1)
    expect(exec.calls[0]?.argv).toEqual(["brew", "list", "--formula", "jq"])
    expect(exec.calls[0]?.shell).toBeFalsy()
  })

  it("returns installed=false when the default check exits non-zero", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "jq", formula: "jq" }
    const result = await checkBrew(step, ctx)

    expect(result.installed).toBe(false)
  })

  it("runs step.check in a shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewStep = {
      type: "brew",
      name: "git",
      formula: "git",
      check: "command -v git",
    }
    const result = await checkBrew(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["command -v git"])
    expect(exec.calls[0]?.shell).toBe(true)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/steps/brew.test.ts`
Expected: FAIL — `Cannot find module "./brew"`.

- [ ] **Step 3: Implement `src/steps/brew.ts`**

```ts
import type { BrewStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

export async function checkBrew(step: BrewStep, ctx: Context): Promise<CheckResult> {
  const input = step.check
    ? { argv: [step.check], shell: true }
    : { argv: ["brew", "list", "--formula", step.formula] }

  const result = await ctx.exec.run(input)
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/steps/brew.test.ts`
Expected: PASS — 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/steps/brew.ts src/steps/brew.test.ts
git commit -m "feat(steps): add brew check handler"
```

---

## Task 9: brew-cask check handler

Runs `brew list --cask <cask>` by default; honors `step.check` override when provided.

**Files:**
- Create: `src/steps/brew-cask.ts`, `src/steps/brew-cask.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/steps/brew-cask.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { checkBrewCask } from "./brew-cask"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { BrewCaskStep } from "../schema"

describe("checkBrewCask", () => {
  it("runs the default `brew list --cask` check", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "iTerm2", cask: "iterm2" }
    const result = await checkBrewCask(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["brew", "list", "--cask", "iterm2"])
  })

  it("returns installed=false on non-zero exit", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "iTerm2", cask: "iterm2" }
    const result = await checkBrewCask(step, ctx)

    expect(result.installed).toBe(false)
  })

  it("runs step.check in shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = {
      type: "brew-cask",
      name: "iTerm2",
      cask: "iterm2",
      check: 'test -d "/Applications/iTerm.app"',
    }
    const result = await checkBrewCask(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(['test -d "/Applications/iTerm.app"'])
    expect(exec.calls[0]?.shell).toBe(true)
    expect(result.installed).toBe(true)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/steps/brew-cask.test.ts`
Expected: FAIL — `Cannot find module "./brew-cask"`.

- [ ] **Step 3: Implement `src/steps/brew-cask.ts`**

```ts
import type { BrewCaskStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

export async function checkBrewCask(step: BrewCaskStep, ctx: Context): Promise<CheckResult> {
  const input = step.check
    ? { argv: [step.check], shell: true }
    : { argv: ["brew", "list", "--cask", step.cask] }

  const result = await ctx.exec.run(input)
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/steps/brew-cask.test.ts`
Expected: PASS — 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/steps/brew-cask.ts src/steps/brew-cask.test.ts
git commit -m "feat(steps): add brew-cask check handler"
```

---

## Task 10: curl-pipe-sh check handler

Always runs `step.check` in a shell (schema already enforces `check` is present).

**Files:**
- Create: `src/steps/curl-pipe-sh.ts`, `src/steps/curl-pipe-sh.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/steps/curl-pipe-sh.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { checkCurlPipe } from "./curl-pipe-sh"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { CurlPipeStep } from "../schema"

describe("checkCurlPipe", () => {
  it("runs step.check in shell mode", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    }
    const result = await checkCurlPipe(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["command -v brew"])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("returns installed=false when the check fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    }
    const result = await checkCurlPipe(step, ctx)

    expect(result.installed).toBe(false)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/steps/curl-pipe-sh.test.ts`
Expected: FAIL — `Cannot find module "./curl-pipe-sh"`.

- [ ] **Step 3: Implement `src/steps/curl-pipe-sh.ts`**

```ts
import type { CurlPipeStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

export async function checkCurlPipe(step: CurlPipeStep, ctx: Context): Promise<CheckResult> {
  const result = await ctx.exec.run({ argv: [step.check], shell: true })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/steps/curl-pipe-sh.test.ts`
Expected: PASS — 2 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/steps/curl-pipe-sh.ts src/steps/curl-pipe-sh.test.ts
git commit -m "feat(steps): add curl-pipe-sh check handler"
```

---

## Task 11: git-clone check handler

Default check: destination directory exists. Honors `step.check` override. Expands `~/` to `$HOME` before the filesystem check. Uses `test -d` through the `Exec` seam rather than bypassing to `fs`.

**Files:**
- Create: `src/steps/git-clone.ts`, `src/steps/git-clone.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/steps/git-clone.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { checkGitClone } from "./git-clone"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { GitCloneStep } from "../schema"

describe("checkGitClone", () => {
  it("runs `test -d` on the dest by default", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "/tmp/dotfiles",
    }
    const result = await checkGitClone(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["test", "-d", "/tmp/dotfiles"])
  })

  it("expands ~/ using ctx.env.HOME", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    }
    await checkGitClone(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["test", "-d", "/Users/test/.dotfiles"])
  })

  it("returns installed=false when dest does not exist", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "/tmp/missing",
    }
    const result = await checkGitClone(step, ctx)

    expect(result.installed).toBe(false)
  })

  it("runs step.check in shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
      check: "test -f ~/.dotfiles/README.md",
    }
    await checkGitClone(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["test -f ~/.dotfiles/README.md"])
    expect(exec.calls[0]?.shell).toBe(true)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/steps/git-clone.test.ts`
Expected: FAIL — `Cannot find module "./git-clone"`.

- [ ] **Step 3: Implement `src/steps/git-clone.ts`**

```ts
import type { GitCloneStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

function expandHome(p: string, home: string): string {
  if (p.startsWith("~/")) {
    return `${home}/${p.slice(2)}`
  }
  return p
}

export async function checkGitClone(step: GitCloneStep, ctx: Context): Promise<CheckResult> {
  if (step.check) {
    const result = await ctx.exec.run({ argv: [step.check], shell: true })
    return result.exitCode === 0 ? { installed: true } : { installed: false }
  }

  const home = ctx.env.HOME ?? ""
  const dest = expandHome(step.dest, home)
  const result = await ctx.exec.run({ argv: ["test", "-d", dest] })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/steps/git-clone.test.ts`
Expected: PASS — 4 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/steps/git-clone.ts src/steps/git-clone.test.ts
git commit -m "feat(steps): add git-clone check handler"
```

---

## Task 12: shell check handler

Always runs `step.check` in a shell (schema enforces `check` is present).

**Files:**
- Create: `src/steps/shell.ts`, `src/steps/shell.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/steps/shell.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { checkShell } from "./shell"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { ShellStep } from "../schema"

describe("checkShell", () => {
  it("runs step.check in shell mode", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "rust",
      install: "curl ... | sh",
      check: "command -v rustc",
    }
    const result = await checkShell(step, ctx)

    expect(result.installed).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["command -v rustc"])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("returns installed=false when the check fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 127 })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "rust",
      install: "curl ... | sh",
      check: "command -v rustc",
    }
    const result = await checkShell(step, ctx)

    expect(result.installed).toBe(false)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/steps/shell.test.ts`
Expected: FAIL — `Cannot find module "./shell"`.

- [ ] **Step 3: Implement `src/steps/shell.ts`**

```ts
import type { ShellStep } from "../schema"
import type { Context } from "../context"
import type { CheckResult } from "./types"

export async function checkShell(step: ShellStep, ctx: Context): Promise<CheckResult> {
  const result = await ctx.exec.run({ argv: [step.check], shell: true })
  return result.exitCode === 0 ? { installed: true } : { installed: false }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/steps/shell.test.ts`
Expected: PASS — 2 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/steps/shell.ts src/steps/shell.test.ts
git commit -m "feat(steps): add shell check handler"
```

---

## Task 13: Handlers registry

Assemble the handlers into a typed registry. The `satisfies` clause forces every step type variant to have an entry.

**Files:**
- Create: `src/steps/index.ts`, `src/steps/index.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/steps/index.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { handlers } from "./index"

describe("handlers registry", () => {
  it("has a check function for every step type", () => {
    const expectedTypes = ["brew", "brew-cask", "curl-pipe-sh", "git-clone", "shell"]
    for (const type of expectedTypes) {
      expect(handlers).toHaveProperty(type)
      expect(typeof (handlers as Record<string, { check: unknown }>)[type]?.check).toBe("function")
    }
  })

  it("has no install handler yet (Phase 2)", () => {
    for (const key of Object.keys(handlers)) {
      const handler = (handlers as Record<string, { install?: unknown }>)[key]
      expect(handler?.install).toBeUndefined()
    }
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/steps/index.test.ts`
Expected: FAIL — `handlers` is not exported from `./index`.

- [ ] **Step 3: Implement `src/steps/index.ts`**

```ts
import type { Step } from "../schema"
import type { Handler } from "./types"
import { checkBrew } from "./brew"
import { checkBrewCask } from "./brew-cask"
import { checkCurlPipe } from "./curl-pipe-sh"
import { checkGitClone } from "./git-clone"
import { checkShell } from "./shell"

export const handlers = {
  brew:           { check: checkBrew },
  "brew-cask":    { check: checkBrewCask },
  "curl-pipe-sh": { check: checkCurlPipe },
  "git-clone":    { check: checkGitClone },
  shell:          { check: checkShell },
} satisfies { [K in Step["type"]]: Handler<Extract<Step, { type: K }>> }

export type Handlers = typeof handlers
export { type Handler, type CheckResult, type InstallResult } from "./types"
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/steps/index.test.ts`
Expected: PASS — 2 tests pass.

- [ ] **Step 5: Verify `tsc` still passes**

Run: `bun run typecheck`
Expected: exits 0.

- [ ] **Step 6: Commit**

```bash
git add src/steps/index.ts src/steps/index.test.ts
git commit -m "feat(steps): add handlers registry with satisfies-enforced completeness"
```

---

## Task 14: Config loader

Read a config file (JSONC / YAML / TOML) via confbox parsers, validate with Valibot, return a `Config`. Phase 1 uses confbox directly — c12 layers in at Phase 3 when we need its `extends:` primitive. First, set up fixtures used by this and subsequent tasks.

**Files:**
- Create: `tests/fixtures/single-brew.jsonc`, `tests/fixtures/single-brew.yaml`, `tests/fixtures/single-brew.toml`, `tests/fixtures/all-five-types.jsonc`, `tests/fixtures/all-installed.jsonc`
- Create: `src/config/load.ts`, `src/config/load.test.ts`

- [ ] **Step 1: Create fixtures**

Create `tests/fixtures/single-brew.jsonc`:

```jsonc
{
  "version": 1,
  "name": "single-brew",
  "steps": [
    { "type": "brew", "name": "jq", "formula": "jq" }
  ]
}
```

Create `tests/fixtures/single-brew.yaml`:

```yaml
version: 1
name: single-brew
steps:
  - type: brew
    name: jq
    formula: jq
```

Create `tests/fixtures/single-brew.toml`:

```toml
version = 1
name = "single-brew"

[[steps]]
type = "brew"
name = "jq"
formula = "jq"
```

Create `tests/fixtures/all-five-types.jsonc`:

```jsonc
{
  "version": 1,
  "name": "all-five-types",
  "steps": [
    { "type": "brew",           "name": "jq",        "formula": "jq" },
    { "type": "brew-cask",      "name": "iTerm2",    "cask": "iterm2" },
    { "type": "curl-pipe-sh",   "name": "Homebrew",  "url": "https://example.com/install.sh", "check": "command -v brew" },
    { "type": "git-clone",      "name": "dotfiles",  "repo": "git@github.com:me/dotfiles.git", "dest": "/tmp/dotfiles" },
    { "type": "shell",          "name": "rust",      "install": "curl ... | sh", "check": "command -v rustc" }
  ]
}
```

Create `tests/fixtures/all-installed.jsonc`:

```jsonc
{
  "version": 1,
  "name": "all-installed",
  "steps": [
    { "type": "brew", "name": "jq", "formula": "jq" }
  ]
}
```

- [ ] **Step 2: Write the failing test**

Create `src/config/load.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { loadConfig } from "./load"
import path from "node:path"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("loadConfig", () => {
  it("loads a JSONC file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.jsonc"))
    expect(config.name).toBe("single-brew")
    expect(config.steps).toHaveLength(1)
    expect(config.steps?.[0]?.type).toBe("brew")
  })

  it("loads a YAML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.yaml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
  })

  it("loads a TOML file", async () => {
    const config = await loadConfig(path.join(fixtures, "single-brew.toml"))
    expect(config.name).toBe("single-brew")
    expect(config.steps?.[0]?.type).toBe("brew")
  })

  it("loads a config with all five step types", async () => {
    const config = await loadConfig(path.join(fixtures, "all-five-types.jsonc"))
    expect(config.steps).toHaveLength(5)
    const types = config.steps?.map((s) => s.type)
    expect(types).toEqual(["brew", "brew-cask", "curl-pipe-sh", "git-clone", "shell"])
  })

  it("throws with a helpful message when the file is not found", async () => {
    await expect(loadConfig("/nope/missing.jsonc")).rejects.toThrow(/missing\.jsonc/)
  })

  it("throws with schema-validation detail when the file is invalid", async () => {
    // Write an invalid fixture inline for this test
    const fs = await import("node:fs/promises")
    const tmpPath = path.join(fixtures, "__invalid-tmp.jsonc")
    await fs.writeFile(tmpPath, JSON.stringify({ version: 2, name: "bad" }))
    try {
      await expect(loadConfig(tmpPath)).rejects.toThrow()
    } finally {
      await fs.unlink(tmpPath).catch(() => undefined)
    }
  })
})
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `bun test src/config/load.test.ts`
Expected: FAIL — `Cannot find module "./load"`.

- [ ] **Step 4: Implement `src/config/load.ts`**

```ts
import * as v from "valibot"
import path from "node:path"
import fs from "node:fs/promises"
import { parseJSONC } from "confbox/jsonc"
import { parseYAML } from "confbox/yaml"
import { parseTOML } from "confbox/toml"
import { config as configSchema, type Config } from "../schema"

export async function loadConfig(configPath: string): Promise<Config> {
  const abs = path.resolve(configPath)

  let text: string
  try {
    text = await fs.readFile(abs, "utf8")
  } catch (err) {
    throw new Error(`cannot read config ${abs}: ${(err as Error).message}`)
  }

  const raw = parseByExtension(abs, text)

  try {
    return v.parse(configSchema, raw)
  } catch (err) {
    if (err instanceof v.ValiError) {
      const issues = err.issues
        .map((i) => `  - ${i.path?.map((p) => p.key).join(".") ?? "<root>"}: ${i.message}`)
        .join("\n")
      throw new Error(`config ${abs} failed schema validation:\n${issues}`)
    }
    throw err
  }
}

function parseByExtension(filePath: string, text: string): unknown {
  const ext = path.extname(filePath).toLowerCase()
  switch (ext) {
    case ".jsonc":
    case ".json":
      return parseJSONC(text)
    case ".yaml":
    case ".yml":
      return parseYAML(text)
    case ".toml":
      return parseTOML(text)
    default:
      throw new Error(`unsupported config extension "${ext}" (${filePath})`)
  }
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `bun test src/config/load.test.ts`
Expected: PASS — 6 tests pass.

- [ ] **Step 6: Commit**

```bash
git add src/config/load.ts src/config/load.test.ts tests/fixtures/
git commit -m "feat(config): load JSONC/YAML/TOML configs via confbox + Valibot"
```

---

## Task 15: Runner — plan orchestration

Takes a validated `Config` and a `Context`, iterates steps, dispatches `check` via the registry, returns a structured `PlanReport`. No printing here — formatting lives in the command layer.

**Files:**
- Create: `src/runner/run.ts`, `src/runner/run.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/runner/run.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { runPlan } from "./run"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"
import type { Config } from "../schema"

describe("runPlan", () => {
  it("reports each step as either installed or would-install based on check exit code", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // brew installed
    exec.queueResponse({ exitCode: 1 })  // shell not installed
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "mixed",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        { type: "shell", name: "rust", install: "curl ... | sh", check: "command -v rustc" },
      ],
    }

    const report = await runPlan(config, ctx)

    expect(report.steps).toHaveLength(2)
    expect(report.steps[0]).toEqual({ name: "jq", type: "brew", status: "installed" })
    expect(report.steps[1]).toEqual({ name: "rust", type: "shell", status: "would-install" })
    expect(report.exitCode).toBe(10)
  })

  it("returns exit 0 when every step is installed", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "all-good",
      steps: [{ type: "brew", name: "jq", formula: "jq" }],
    }

    const report = await runPlan(config, ctx)

    expect(report.exitCode).toBe(0)
    expect(report.steps[0]?.status).toBe("installed")
  })

  it("returns exit 0 on a config with no steps", async () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    const config: Config = { version: 1, name: "empty" }
    const report = await runPlan(config, ctx)

    expect(report.exitCode).toBe(0)
    expect(report.steps).toHaveLength(0)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/runner/run.test.ts`
Expected: FAIL — `Cannot find module "./run"`.

- [ ] **Step 3: Implement `src/runner/run.ts`**

```ts
import type { Config, Step } from "../schema"
import type { Context } from "../context"
import { handlers } from "../steps"

export type StepStatus = "installed" | "would-install"

export type StepReport = {
  name: string
  type: Step["type"]
  status: StepStatus
}

export type PlanReport = {
  configName: string
  steps: StepReport[]
  exitCode: 0 | 10
}

export async function runPlan(config: Config, ctx: Context): Promise<PlanReport> {
  const steps = config.steps ?? []
  const reports: StepReport[] = []

  for (const step of steps) {
    const handler = handlers[step.type] as { check: (s: typeof step, c: Context) => Promise<{ installed: boolean }> }
    const result = await handler.check(step, ctx)
    reports.push({
      name: step.name,
      type: step.type,
      status: result.installed ? "installed" : "would-install",
    })
  }

  const anyWouldInstall = reports.some((r) => r.status === "would-install")
  return {
    configName: config.name,
    steps: reports,
    exitCode: anyWouldInstall ? 10 : 0,
  }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/runner/run.test.ts`
Expected: PASS — 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/runner/run.ts src/runner/run.test.ts
git commit -m "feat(runner): iterate steps and dispatch check via handler registry"
```

---

## Task 16: `plan` command

Citty command that ties it together: load config, run plan, print report to stdout, return the report's exit code.

**Files:**
- Create: `src/commands/plan.ts`, `src/commands/plan.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/commands/plan.test.ts`:

```ts
import { describe, it, expect, spyOn } from "bun:test"
import { runCommand } from "citty"
import path from "node:path"
import { planCommand } from "./plan"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("plan command", () => {
  it("prints a report and returns exit code 10 when a step would install", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    try {
      const { result } = await runCommand(planCommand, {
        rawArgs: ["--config", path.join(fixtures, "single-brew.jsonc")],
      })
      expect(result).toBe(10)
      const output = logSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(output).toContain("single-brew")
      expect(output).toContain("jq")
    } finally {
      logSpy.mockRestore()
    }
  })

  it("returns exit code 0 when no config path is given", async () => {
    // Phase 1 requires --config; absence is an error (exit code != 0/10).
    // In Phase 4 this becomes an interactive picker.
    const errSpy = spyOn(console, "error").mockImplementation(() => undefined)
    try {
      await expect(runCommand(planCommand, { rawArgs: [] })).rejects.toThrow()
    } finally {
      errSpy.mockRestore()
    }
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test src/commands/plan.test.ts`
Expected: FAIL — `Cannot find module "./plan"`.

- [ ] **Step 3: Implement `src/commands/plan.ts`**

```ts
import { defineCommand } from "citty"
import { loadConfig } from "../config/load"
import { runPlan, type StepReport } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"

export const planCommand = defineCommand({
  meta: {
    name: "plan",
    description: "Check what would be installed without installing anything",
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
    const ctx = makeContext({ exec: new ExecaExec() })
    const report = await runPlan(config, ctx)

    printReport(config.name, report.steps)
    return report.exitCode
  },
})

function printReport(configName: string, steps: StepReport[]): void {
  console.log(`CONFIG: ${configName}  (${steps.length} step${steps.length === 1 ? "" : "s"})`)
  console.log("")
  steps.forEach((step, i) => {
    const idx = `[${i + 1}/${steps.length}]`
    const status = step.status === "installed" ? "✓" : "·"
    const label = step.status === "installed" ? "already installed" : "would install"
    console.log(`  ${status} ${idx} ${step.name}  ${label}`)
  })
  console.log("")
  const wouldInstall = steps.filter((s) => s.status === "would-install").length
  if (wouldInstall === 0) {
    console.log("Machine is up to date.")
  } else {
    console.log(`${wouldInstall} step${wouldInstall === 1 ? "" : "s"} would install.`)
  }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `bun test src/commands/plan.test.ts`
Expected: PASS — 2 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/commands/plan.ts src/commands/plan.test.ts
git commit -m "feat(cli): add plan command"
```

---

## Task 17: `version` command + CLI entrypoint

Wire the main citty command tree. Replaces the stub `cli.ts` from Task 1.

**Files:**
- Create: `src/commands/version.ts`
- Modify: `src/cli.ts`

- [ ] **Step 1: Implement `src/commands/version.ts`**

```ts
import { defineCommand } from "citty"
import pkg from "../../package.json" with { type: "json" }

export const versionCommand = defineCommand({
  meta: {
    name: "version",
    description: "Print the gearup version",
  },
  run() {
    console.log(pkg.version)
  },
})
```

- [ ] **Step 2: Replace `src/cli.ts`**

```ts
#!/usr/bin/env bun
import { defineCommand, runMain } from "citty"
import { planCommand } from "./commands/plan"
import { versionCommand } from "./commands/version"
import pkg from "../package.json" with { type: "json" }

const mainCommand = defineCommand({
  meta: {
    name: "gearup",
    version: pkg.version,
    description: "Config-driven macOS developer-machine bootstrap",
  },
  subCommands: {
    plan: planCommand,
    version: versionCommand,
  },
})

runMain(mainCommand)
```

- [ ] **Step 3: Smoke-test the entrypoint**

Run: `bun run src/cli.ts version`
Expected output: `0.2.0-alpha.1` (or whatever the current `package.json` version is).

Run: `bun run src/cli.ts --help`
Expected output: includes `gearup`, `plan`, `version`, and a USAGE section.

Run: `bun run src/cli.ts plan --config tests/fixtures/single-brew.jsonc`
Expected output: a styled report for `single-brew`, ending with either "Machine is up to date." or "1 step would install." depending on whether `jq` is installed on the host running this test.
Expected exit code: `0` or `10`, depending on host state.

- [ ] **Step 4: Commit**

```bash
git add src/commands/version.ts src/cli.ts
git commit -m "feat(cli): wire citty entrypoint with version and plan subcommands"
```

---

## Task 18: End-to-end integration test

One integration test that runs the full pipeline via a real subprocess, against a real fixture. This is the tracer's acceptance test: if this passes, the port has completed a full vertical slice.

**Files:**
- Create: `tests/e2e.test.ts`

- [ ] **Step 1: Write the failing test**

Create `tests/e2e.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { $ } from "bun"
import path from "node:path"

const repoRoot = path.resolve(import.meta.dir, "..")
const fixtures = path.join(repoRoot, "tests/fixtures")

describe("e2e: gearup plan", () => {
  it("prints a report and exits 0 or 10 for a valid single-brew config", async () => {
    const result = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config ${path.join(fixtures, "single-brew.jsonc")}`.quiet().nothrow()
    const stdout = result.stdout.toString()

    expect([0, 10]).toContain(result.exitCode)
    expect(stdout).toContain("single-brew")
    expect(stdout).toContain("jq")
  })

  it("exits non-zero when config is missing", async () => {
    const result = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config /nope/missing.jsonc`.quiet().nothrow()
    expect(result.exitCode).not.toBe(0)
    expect(result.exitCode).not.toBe(10)
  })

  it("accepts YAML and TOML configs", async () => {
    const yamlResult = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config ${path.join(fixtures, "single-brew.yaml")}`.quiet().nothrow()
    const tomlResult = await $`bun run ${path.join(repoRoot, "src/cli.ts")} plan --config ${path.join(fixtures, "single-brew.toml")}`.quiet().nothrow()

    expect([0, 10]).toContain(yamlResult.exitCode)
    expect([0, 10]).toContain(tomlResult.exitCode)
  })
})
```

- [ ] **Step 2: Run the test**

Run: `bun test tests/e2e.test.ts`
Expected: PASS — 3 tests pass.

- [ ] **Step 3: Run the full test suite one final time**

Run: `bun test`
Expected: all unit + integration + e2e tests pass together. No type errors.

- [ ] **Step 4: Typecheck**

Run: `bun run typecheck`
Expected: exits 0.

- [ ] **Step 5: Commit**

```bash
git add tests/e2e.test.ts
git commit -m "test: add end-to-end plan integration tests (JSONC/YAML/TOML)"
```

---

## Phase 1 Done — Merge Back

- [ ] **Step 1: Final status check**

Run: `git log --oneline main..HEAD`
Expected: ~18 commits (one per task, plus preflight scaffolding).

Run: `bun test`
Expected: all green.

- [ ] **Step 2: Merge back to main**

From the worktree:

```bash
cd ../gearup                    # back to main worktree
git merge ts-port/phase-1-tracer --no-ff -m "feat: TS port Phase 1 tracer (plan command)"
git worktree remove ../gearup-ts-phase1
git branch -d ts-port/phase-1-tracer
```

- [ ] **Step 3: Verify on main**

Run: `bun test` (from main worktree)
Expected: all green.

Run: `bun run src/cli.ts plan --config tests/fixtures/single-brew.jsonc`
Expected: a styled report; exit code 0 or 10 depending on whether `jq` is installed on this machine.

> Note: running against the existing `configs/backend.yaml` will **not** behave correctly in Phase 1. `backend.yaml` is all `extends: [base, jvm, ...]` and carries no inline steps. Phase 1's loader parses the file fine (the schema allows `extends`) but produces an empty step list, so the output will say "Machine is up to date" misleadingly. Phase 3 adds the `extends:` pre-resolver that walks those references and flattens steps. Until then, use single-file configs like the fixtures in `tests/fixtures/`.

---

## Exit Criteria for Phase 1

1. `bun test` passes — all unit, integration, and e2e tests green.
2. `bun run typecheck` exits 0.
3. `bun run src/cli.ts plan --config tests/fixtures/single-brew.jsonc` runs end-to-end and returns exit code 0 or 10 based on host state.
4. JSONC, YAML, and TOML fixtures all parse through the same pipeline.
5. Every step handler has a unit test with `FakeExec`.
6. The `satisfies` clause in `src/steps/index.ts` would fail compilation if a step type were missing from the registry.

## What's NOT in Phase 1 (deferred to later phases)

- `gearup run` (installation) — Phase 2
- `post_install` hooks — Phase 2
- Elevation pause banner — Phase 2
- `extends:` composition — Phase 3
- XDG logging — Phase 3
- Interactive picker when `--config` omitted — Phase 4
- Styled spinners / progress indicators — Phase 4
- `gearup init` + embedded default configs — Phase 4
- Release pipeline (`bun build --compile`) — Phase 4
- Go code removal — Phase 4

Each of these gets its own implementation plan when we're ready to tackle them.
