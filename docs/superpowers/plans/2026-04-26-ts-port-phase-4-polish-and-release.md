# TS Port — Phase 4 Polish & Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the TypeScript port to feature-parity with Go gearup, then replace it. Adds: idiomatic Clack UX (intro/outro/spinner) on both `plan` and `run`, interactive picker when `--config` is omitted, embedded default configs with `gearup init` + auto-extract on first run, release pipeline (`bun build --compile` matrix, GitHub Actions, install.sh rewrite). Removes the Go codebase. Bumps to `0.3.0`. Rewrites the README.

**Architecture:** Phase 4 is the polish layer — it doesn't change the architecture, it adds presentation and distribution on top. Two new layers join: (a) a thin **progress reporter** interface threaded through `runInstall` so the runner can emit step-level events without knowing about the UI; the run command implements it with `clack.spinner()`. (b) An **embedded configs** module that uses Bun's `import ... with { type: "file" }` to bundle JSONC files into the compiled binary, exposing them via a registry that `gearup init` writes to `~/.config/gearup/configs/`.

**Tech Stack:** Phase 1-3 stack plus heavier use of `@clack/prompts` (intro, outro, cancel, select, spinner, note). No new deps for the binary itself; release pipeline uses `bun build --compile` with platform targets. CI moves from Go to Bun.

---

## Preflight

Phases 1, 2, and 3 are merged to `main`. The Go code still lives alongside the TS code. Phase 4 is the phase that retires Go.

**Before Task 1, create a worktree:**

```bash
cd /Users/dlo/Dev/gearup
git worktree add .worktrees/phase-4-polish -b ts-port/phase-4-polish-and-release
cd .worktrees/phase-4-polish
```

All subsequent tasks assume you are in `.worktrees/phase-4-polish/`. The branch merges back to `main` at the end.

**Reference files from Go that Phase 4 mirrors / replaces:**
- `configs/embed.go` — `go:embed` mechanism (replaced by Bun's import-with-file)
- `internal/ui/spinner.go`, `picker.go`, `preview.go` — UX primitives (replaced by Clack)
- `cmd/gearup/main.go` — top-level command wiring with `init` (replaced; we already have citty)
- `install.sh` — curl-pipe installer (rewritten to fetch new artifacts)
- `.goreleaser.yaml`, `.github/workflows/release.yml` — release pipeline (rewritten)

**Behavior contract for picker resolution:**

Configs can live in two places:
1. `${XDG_CONFIG_HOME ?? ~/.config}/gearup/configs/` — user-global
2. `./configs/` — project-local

When `--config` is omitted, the picker shows a flat union of both locations. **On name collision, project-local wins** (closer to the user's `pwd`). This is documented in the README and as a comment in the discovery module.

---

## File Structure (Phase 4)

```
gearup/
├── package.json                       MODIFY — bump to 0.3.0; add build:release script
├── README.md                          REWRITE — Bun-based, new install path, Clack output
├── install.sh                         REWRITE — fetch Bun-compiled artifacts; verify checksum
├── configs/
│   ├── base.jsonc                     CONVERT (was base.yaml)
│   ├── backend.jsonc                  CONVERT
│   ├── frontend.jsonc                 CONVERT
│   ├── jvm.jsonc                      CONVERT
│   ├── containers.jsonc               CONVERT
│   ├── aws-k8s.jsonc                  CONVERT
│   ├── node.jsonc                     CONVERT
│   ├── *.yaml                         DELETE — Go-format configs
│   ├── desktop-apps.yaml              DELETE — redundant with base
│   └── embed.go                       DELETE — replaced by src/configs/embedded.ts
├── scripts/
│   ├── build-release.ts               NEW — bun build --compile for each target + sha256
│   └── (existing scripts kept)
├── src/
│   ├── configs/
│   │   ├── embedded.ts                NEW — registry of embedded config paths via import-with-file
│   │   ├── extract.ts                 NEW — extract embedded configs to a target dir
│   │   └── extract.test.ts            NEW
│   ├── commands/
│   │   ├── init.ts                    NEW — gearup init command
│   │   ├── init.test.ts               NEW
│   │   ├── plan.ts                    REWRITE — Clack intro/outro/note
│   │   ├── plan.test.ts               MODIFY — mock clack
│   │   ├── run.ts                     REWRITE — Clack intro/outro/spinner via ProgressReporter
│   │   └── run.test.ts                MODIFY — mock clack
│   ├── runner/
│   │   ├── progress.ts                NEW — ProgressReporter interface + NoopReporter
│   │   ├── run.ts                     MODIFY — accept optional ProgressReporter; emit step events
│   │   └── run.test.ts                MODIFY — add ProgressReporter assertions
│   ├── ui/
│   │   ├── picker.ts                  NEW — discoverConfigs + pickConfig (Clack select)
│   │   ├── picker.test.ts             NEW
│   │   └── reporter.ts                NEW — createClackReporter (ProgressReporter for Clack spinner)
│   ├── cli.ts                         MODIFY — wire init command; trigger picker when --config omitted
│   ├── log/
│   │   └── fake.ts                    MODIFY — default sentinel becomes "<no log>" not "/dev/null" (Phase 3 cleanup)
│   └── config/
│       └── load.ts                    MODIFY — comment about extends extension requirement (Phase 3 cleanup)
├── tests/
│   └── fixtures/                      (existing; no Phase 4 changes)
├── .github/
│   └── workflows/
│       ├── ci.yml                     REWRITE — run Bun tests, drop Go
│       └── release.yml                REWRITE — bun build --compile matrix, upload to GitHub Release
├── .goreleaser.yaml                   DELETE
├── go.mod, go.sum                     DELETE
├── cmd/gearup/main.go                 DELETE
├── internal/                          DELETE (entire tree)
└── gearup                             DELETE (compiled Go binary if present; gitignored anyway)
```

---

## Task 1: Convert configs to JSONC + new schema

8 existing `configs/*.yaml` files convert to JSONC with the Phase 3 schema (Record-keyed steps, c12-native extends paths with explicit extensions, no `sources:` field). The `desktop-apps.yaml` is **deleted** — it's redundant with iTerm2/Bruno already in `base.yaml`.

**Files:**
- Create: `configs/base.jsonc`, `configs/backend.jsonc`, `configs/frontend.jsonc`, `configs/jvm.jsonc`, `configs/containers.jsonc`, `configs/aws-k8s.jsonc`, `configs/node.jsonc`
- Delete: All `configs/*.yaml` files; `configs/desktop-apps.yaml`; `configs/embed.go`

- [ ] **Step 1: Create `configs/base.jsonc`**

```jsonc
{
  "version": 1,
  "name": "base",
  "description": "Homebrew, core CLI tools, and standard desktop apps",
  "steps": {
    "Homebrew": {
      "type": "curl-pipe-sh",
      "url": "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh",
      "check": "command -v brew"
    },
    "Git": {
      "type": "brew",
      "formula": "git",
      "check": "command -v git"
    },
    "jq": {
      "type": "brew",
      "formula": "jq"
    },
    "iTerm2": {
      "type": "brew-cask",
      "cask": "iterm2",
      "check": "test -d \"/Applications/iTerm.app\""
    },
    "Bruno": {
      "type": "brew-cask",
      "cask": "bruno",
      "check": "test -d \"/Applications/Bruno.app\""
    }
  }
}
```

- [ ] **Step 2: Create `configs/jvm.jsonc`**

```jsonc
{
  "version": 1,
  "name": "jvm",
  "description": "JVM development toolchain (OpenJDK 21) + system Java discovery linkage",
  "steps": {
    "OpenJDK 21": {
      "type": "brew",
      "formula": "openjdk@21"
    },
    "Link OpenJDK 21 for system Java discovery": {
      "type": "shell",
      "requires_elevation": true,
      "check": "test -L /Library/Java/JavaVirtualMachines/openjdk-21.jdk",
      "install": "sudo ln -sfn /opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk /Library/Java/JavaVirtualMachines/openjdk-21.jdk"
    }
  }
}
```

- [ ] **Step 3: Create `configs/containers.jsonc`**

The Docker Compose step has a multi-line install command in the YAML. JSONC can't have raw multi-line strings, so we collapse to a single shell command joined with `&&`:

```jsonc
{
  "version": 1,
  "name": "containers",
  "description": "Local container runtime + tooling (Docker CLI, Compose plugin, Colima)",
  "steps": {
    "Docker CLI": {
      "type": "brew",
      "formula": "docker",
      "check": "command -v docker"
    },
    "Docker Compose (CLI plugin)": {
      "type": "shell",
      "check": "docker compose version >/dev/null 2>&1",
      "install": "set -euo pipefail && DOCKER_CONFIG=\"${DOCKER_CONFIG:-$HOME/.docker}\" && mkdir -p \"$DOCKER_CONFIG/cli-plugins\" && ARCH=\"$(uname -m | sed 's/arm64/aarch64/')\" && curl -fsSL \"https://github.com/docker/compose/releases/download/v2.32.4/docker-compose-$(uname -s | tr '[:upper:]' '[:lower:]')-${ARCH}\" -o \"$DOCKER_CONFIG/cli-plugins/docker-compose\" && chmod +x \"$DOCKER_CONFIG/cli-plugins/docker-compose\""
    },
    "Colima": {
      "type": "brew",
      "formula": "colima"
    }
  }
}
```

- [ ] **Step 4: Create `configs/aws-k8s.jsonc`**

```jsonc
{
  "version": 1,
  "name": "aws-k8s",
  "description": "AWS and Kubernetes CLI tooling",
  "steps": {
    "AWS CLI": {
      "type": "brew",
      "formula": "awscli"
    },
    "aws-iam-authenticator": {
      "type": "brew",
      "formula": "aws-iam-authenticator"
    },
    "kubectl": {
      "type": "brew",
      "formula": "kubernetes-cli"
    }
  }
}
```

- [ ] **Step 5: Create `configs/node.jsonc`**

```jsonc
{
  "version": 1,
  "name": "node",
  "description": "Node.js via nvm (Node Version Manager)",
  "steps": {
    "nvm": {
      "type": "curl-pipe-sh",
      "url": "https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.4/install.sh",
      "check": "[ -s \"$HOME/.nvm/nvm.sh\" ]"
    }
  }
}
```

- [ ] **Step 6: Create `configs/backend.jsonc`**

```jsonc
{
  "version": 1,
  "name": "Backend",
  "description": "Full macOS developer toolchain for backend/infra work",
  "platform": {
    "os": ["darwin"]
  },
  "elevation": {
    "message": "Some steps need admin permissions. Elevate your session now (via your usual mechanism), then press Continue.",
    "duration": "180s"
  },
  "extends": [
    "./base.jsonc",
    "./jvm.jsonc",
    "./containers.jsonc",
    "./aws-k8s.jsonc",
    "./node.jsonc"
  ]
}
```

- [ ] **Step 7: Create `configs/frontend.jsonc`**

```jsonc
{
  "version": 1,
  "name": "Frontend",
  "description": "macOS developer toolchain for frontend work (Node + core CLI)",
  "platform": {
    "os": ["darwin"]
  },
  "extends": [
    "./base.jsonc",
    "./node.jsonc"
  ]
}
```

- [ ] **Step 8: Delete the old YAML configs and embed.go**

```bash
rm configs/*.yaml configs/embed.go
```

- [ ] **Step 9: Verify the new configs parse**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun run src/cli.ts plan --config configs/backend.jsonc 2>&1 | head -30
```

You may see real shell command exits (some checks may fail because tools aren't installed locally) — that's fine. What matters is that the **config parses** without schema errors and you see a styled report. If you see a schema validation error, fix the offending JSONC.

```bash
bun test
```
Expected: 119 tests still pass.

- [ ] **Step 10: Commit**

```bash
git add configs/
git commit -m "feat(configs): convert to JSONC with Phase 3 schema; drop redundant desktop-apps"
```

---

## Task 2: Embed configs into the binary

Use Bun's `import ... with { type: "file" }` to bundle each `configs/*.jsonc` into the compiled binary. Expose a registry mapping config name → embedded file path. Runtime code reads via `Bun.file(embeddedPath).text()`.

**Files:**
- Create: `src/configs/embedded.ts`

- [ ] **Step 1: Implement `src/configs/embedded.ts`**

```ts
// At build time, Bun's bundler sees each `import ... with { type: "file" }` and
// bundles the file contents into the compiled binary. At runtime, the imported
// value is a path string; `Bun.file(path).text()` reads the embedded content.
//
// In dev mode (`bun run`), the path resolves to the source file on disk, so this
// works transparently for both `bun run src/cli.ts` and `bun build --compile`.

import basePath from "../../configs/base.jsonc" with { type: "file" }
import backendPath from "../../configs/backend.jsonc" with { type: "file" }
import frontendPath from "../../configs/frontend.jsonc" with { type: "file" }
import jvmPath from "../../configs/jvm.jsonc" with { type: "file" }
import containersPath from "../../configs/containers.jsonc" with { type: "file" }
import awsK8sPath from "../../configs/aws-k8s.jsonc" with { type: "file" }
import nodePath from "../../configs/node.jsonc" with { type: "file" }

/** Map from config filename → embedded asset path. Order matters: this is the order
 *  they appear in `gearup init` output. */
export const EMBEDDED_CONFIGS: Record<string, string> = {
  "base.jsonc": basePath,
  "backend.jsonc": backendPath,
  "frontend.jsonc": frontendPath,
  "jvm.jsonc": jvmPath,
  "containers.jsonc": containersPath,
  "aws-k8s.jsonc": awsK8sPath,
  "node.jsonc": nodePath,
}

/** Read the contents of an embedded config by filename. */
export async function readEmbeddedConfig(filename: string): Promise<string> {
  const path = EMBEDDED_CONFIGS[filename]
  if (!path) {
    throw new Error(`embedded config not found: ${filename}`)
  }
  return await Bun.file(path).text()
}
```

- [ ] **Step 2: Verify imports resolve and Bun reads the files**

```bash
bun run src/cli.ts version  # smoke-test: anything that imports the entrypoint goes through
```

If the import-with-file syntax errors, fix per Bun's docs at https://bun.sh/docs/runtime/loaders. As of Bun 1.3.x the syntax should be `import x from "..." with { type: "file" }`.

Quick verification that the import gives a readable path:

```bash
bun -e 'import("/Users/dlo/Dev/gearup/.worktrees/phase-4-polish/src/configs/embedded.ts").then(m => m.readEmbeddedConfig("base.jsonc")).then(s => console.log(s.slice(0, 100)))'
```
Expected: prints the first 100 characters of `base.jsonc`.

- [ ] **Step 3: Run typecheck**

```bash
bun run typecheck
```
Expected: exit 0. If TS complains about the `with { type: "file" }` import syntax (it sometimes does with strict tsconfig), add a triple-slash `/// <reference types="bun-types" />` at the top or extend `tsconfig.json`'s `module` setting. Bun's types should already cover it via `@types/bun`.

- [ ] **Step 4: Run full suite (no new tests yet — Task 3 adds them)**

```bash
bun test
```
Expected: 119 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/configs/embedded.ts
git commit -m "feat(configs): embed default configs into the binary via import-with-file"
```

---

## Task 3: `gearup init` command + auto-extract on first run

Two pieces:

1. **`extractConfigs(targetDir, force)`** — writes embedded configs to a directory. Returns `{ written, skipped }`. With `force: false`, existing files are skipped.
2. **`gearup init [--force]`** — citty command that calls `extractConfigs` on `${XDG_CONFIG_HOME ?? ~/.config}/gearup/configs/`, prints a Clack-styled summary.

Auto-extract on first run: covered in Task 5 when discovery runs and finds nothing in the user dir.

**Files:**
- Create: `src/configs/extract.ts`, `src/configs/extract.test.ts`
- Create: `src/commands/init.ts`, `src/commands/init.test.ts`

- [ ] **Step 1: Failing test — `src/configs/extract.test.ts`**

```ts
import { describe, it, expect, afterEach } from "bun:test"
import fs from "node:fs/promises"
import path from "node:path"
import { extractConfigs } from "./extract"

const tmpDir = path.join("/tmp", `gearup-extract-test-${process.pid}`)

afterEach(async () => {
  await fs.rm(tmpDir, { recursive: true, force: true })
})

describe("extractConfigs", () => {
  it("writes all embedded configs to a fresh directory", async () => {
    const result = await extractConfigs(tmpDir, false)

    expect(result.written.length).toBeGreaterThan(0)
    expect(result.skipped).toEqual([])

    const files = await fs.readdir(tmpDir)
    expect(files).toContain("base.jsonc")
    expect(files).toContain("backend.jsonc")

    // Verify content is the actual config (not empty)
    const base = await fs.readFile(path.join(tmpDir, "base.jsonc"), "utf8")
    expect(base).toContain('"name": "base"')
  })

  it("skips existing files when force is false", async () => {
    await fs.mkdir(tmpDir, { recursive: true })
    await fs.writeFile(path.join(tmpDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const result = await extractConfigs(tmpDir, false)

    expect(result.skipped).toContain("base.jsonc")
    expect(result.written).not.toContain("base.jsonc")

    // The custom file is preserved
    const base = await fs.readFile(path.join(tmpDir, "base.jsonc"), "utf8")
    expect(base).toContain("my-custom")
  })

  it("overwrites existing files when force is true", async () => {
    await fs.mkdir(tmpDir, { recursive: true })
    await fs.writeFile(path.join(tmpDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const result = await extractConfigs(tmpDir, true)

    expect(result.written).toContain("base.jsonc")
    expect(result.skipped).toEqual([])

    // The default content is restored
    const base = await fs.readFile(path.join(tmpDir, "base.jsonc"), "utf8")
    expect(base).toContain('"name": "base"')
    expect(base).not.toContain("my-custom")
  })

  it("creates the target directory if it doesn't exist", async () => {
    const deepPath = path.join(tmpDir, "deeply", "nested", "dir")
    await extractConfigs(deepPath, false)

    const files = await fs.readdir(deepPath)
    expect(files.length).toBeGreaterThan(0)
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun test src/configs/extract.test.ts
```

- [ ] **Step 3: Implement `src/configs/extract.ts`**

```ts
import fs from "node:fs/promises"
import path from "node:path"
import { EMBEDDED_CONFIGS, readEmbeddedConfig } from "./embedded"

export type ExtractResult = {
  /** Filenames written to the target dir (relative names like "base.jsonc"). */
  written: string[]
  /** Filenames skipped because they already existed (force was false). */
  skipped: string[]
}

/**
 * Write each embedded config to `targetDir`. With `force: false`, files that already
 * exist are skipped. The target directory is created if it doesn't exist.
 */
export async function extractConfigs(targetDir: string, force: boolean): Promise<ExtractResult> {
  await fs.mkdir(targetDir, { recursive: true })

  const written: string[] = []
  const skipped: string[] = []

  for (const filename of Object.keys(EMBEDDED_CONFIGS)) {
    const dest = path.join(targetDir, filename)

    if (!force) {
      try {
        await fs.access(dest)
        skipped.push(filename)
        continue
      } catch {
        // doesn't exist; fall through to write
      }
    }

    const content = await readEmbeddedConfig(filename)
    await fs.writeFile(dest, content)
    written.push(filename)
  }

  return { written, skipped }
}
```

- [ ] **Step 4: Run (PASS — 4 tests)**

```bash
bun test src/configs/extract.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/configs/extract.ts src/configs/extract.test.ts
git commit -m "feat(configs): add extractConfigs to write embedded configs to a target dir"
```

- [ ] **Step 6: Failing test — `src/commands/init.test.ts`**

```ts
import { describe, it, expect, spyOn, mock, afterEach } from "bun:test"

mock.module("@clack/prompts", () => ({
  intro: mock(),
  outro: mock(),
  note: mock(),
  cancel: mock(),
  isCancel: () => false,
}))

import { runCommand } from "citty"
import path from "node:path"
import fs from "node:fs/promises"
import { initCommand } from "./init"

const tmpStateDir = path.join("/tmp", `gearup-init-cmd-test-${process.pid}`)

afterEach(async () => {
  await fs.rm(tmpStateDir, { recursive: true, force: true })
})

describe("init command", () => {
  it("writes embedded configs to the XDG_CONFIG_HOME path", async () => {
    process.env.XDG_CONFIG_HOME = tmpStateDir
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)

    try {
      const { result } = await runCommand(initCommand, { rawArgs: [] })

      expect(result).toBe(0)

      const targetDir = path.join(tmpStateDir, "gearup", "configs")
      const files = await fs.readdir(targetDir)
      expect(files).toContain("base.jsonc")
      expect(files).toContain("backend.jsonc")
    } finally {
      logSpy.mockRestore()
    }
  })

  it("preserves existing files without --force", async () => {
    process.env.XDG_CONFIG_HOME = tmpStateDir
    const targetDir = path.join(tmpStateDir, "gearup", "configs")
    await fs.mkdir(targetDir, { recursive: true })
    await fs.writeFile(path.join(targetDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    try {
      await runCommand(initCommand, { rawArgs: [] })

      const base = await fs.readFile(path.join(targetDir, "base.jsonc"), "utf8")
      expect(base).toContain("my-custom")
    } finally {
      logSpy.mockRestore()
    }
  })

  it("overwrites existing files with --force", async () => {
    process.env.XDG_CONFIG_HOME = tmpStateDir
    const targetDir = path.join(tmpStateDir, "gearup", "configs")
    await fs.mkdir(targetDir, { recursive: true })
    await fs.writeFile(path.join(targetDir, "base.jsonc"), '{"version":1,"name":"my-custom"}')

    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    try {
      await runCommand(initCommand, { rawArgs: ["--force"] })

      const base = await fs.readFile(path.join(targetDir, "base.jsonc"), "utf8")
      expect(base).toContain('"name": "base"')
      expect(base).not.toContain("my-custom")
    } finally {
      logSpy.mockRestore()
    }
  })
})
```

- [ ] **Step 7: Run (FAIL)**

```bash
bun test src/commands/init.test.ts
```

- [ ] **Step 8: Implement `src/commands/init.ts`**

```ts
import { defineCommand } from "citty"
import * as clack from "@clack/prompts"
import path from "node:path"
import { extractConfigs } from "../configs/extract"

/**
 * Resolve the XDG config dir for gearup. Honors XDG_CONFIG_HOME, falls back to
 * ~/.config. Throws when HOME is also unset.
 */
export function userConfigsDir(env: Record<string, string | undefined>): string {
  if (env.XDG_CONFIG_HOME) {
    return path.join(env.XDG_CONFIG_HOME, "gearup", "configs")
  }
  if (env.HOME) {
    return path.join(env.HOME, ".config", "gearup", "configs")
  }
  throw new Error("userConfigsDir: neither XDG_CONFIG_HOME nor HOME is set")
}

export const initCommand = defineCommand({
  meta: {
    name: "init",
    description: "Write the embedded default configs to ~/.config/gearup/configs/ (or $XDG_CONFIG_HOME)",
  },
  args: {
    force: {
      type: "boolean",
      description: "Overwrite existing files instead of skipping them",
      default: false,
    },
  },
  async run({ args }) {
    clack.intro("gearup init")

    const targetDir = userConfigsDir(process.env)
    const { written, skipped } = await extractConfigs(targetDir, args.force)

    if (written.length > 0) {
      clack.note(written.join("\n"), `Wrote ${written.length} config${written.length === 1 ? "" : "s"}`)
    }
    if (skipped.length > 0) {
      clack.note(
        skipped.join("\n"),
        `Skipped ${skipped.length} (use --force to overwrite)`,
      )
    }

    clack.outro(`Configs at ${targetDir}`)
    return 0
  },
})
```

- [ ] **Step 9: Run (PASS — 3 tests)**

```bash
bun test src/commands/init.test.ts
```

- [ ] **Step 10: Run full suite**

```bash
bun test
bun run typecheck
```

- [ ] **Step 11: Commit**

```bash
git add src/commands/init.ts src/commands/init.test.ts
git commit -m "feat(cli): add gearup init command with --force; resolves XDG_CONFIG_HOME"
```

---

## Task 4: Config discovery (search XDG + ./configs/)

Walk both locations, parse each config's `name` and `description` (without resolving `extends:` — too expensive for a discovery pass), return a flat list with source labels. Project-local wins on name collision.

**Files:**
- Create: `src/ui/picker.ts`, `src/ui/picker.test.ts` (we'll add the picker proper in Task 5; this task is the discovery half)

- [ ] **Step 1: Failing test — `src/ui/picker.test.ts`**

```ts
import { describe, it, expect, afterEach } from "bun:test"
import fs from "node:fs/promises"
import path from "node:path"
import { discoverConfigs } from "./picker"

const userDir = path.join("/tmp", `gearup-discover-user-${process.pid}`)
const projectDir = path.join("/tmp", `gearup-discover-proj-${process.pid}`)
const projectCwd = path.dirname(projectDir)

afterEach(async () => {
  await fs.rm(userDir, { recursive: true, force: true })
  await fs.rm(projectDir, { recursive: true, force: true })
})

async function writeConfig(dir: string, filename: string, name: string, description?: string) {
  await fs.mkdir(dir, { recursive: true })
  const obj: Record<string, unknown> = { version: 1, name }
  if (description) obj.description = description
  await fs.writeFile(path.join(dir, filename), JSON.stringify(obj))
}

describe("discoverConfigs", () => {
  it("returns configs from user dir labeled 'user'", async () => {
    await writeConfig(userDir, "base.jsonc", "base", "Core tools")

    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })

    expect(result).toHaveLength(1)
    expect(result[0]?.name).toBe("base")
    expect(result[0]?.description).toBe("Core tools")
    expect(result[0]?.source).toBe("user")
  })

  it("returns configs from project dir labeled 'project'", async () => {
    await writeConfig(projectDir, "team.jsonc", "Team", "Team setup")

    const result = await discoverConfigs({ userDir: "/nonexistent", projectDir })

    expect(result).toHaveLength(1)
    expect(result[0]?.source).toBe("project")
  })

  it("project-local wins on name collision; user copy is dropped", async () => {
    await writeConfig(userDir, "shared.jsonc", "shared", "User version")
    await writeConfig(projectDir, "shared.jsonc", "shared", "Project version")

    const result = await discoverConfigs({ userDir, projectDir })

    expect(result).toHaveLength(1)
    expect(result[0]?.source).toBe("project")
    expect(result[0]?.description).toBe("Project version")
  })

  it("ignores files without a recognized extension", async () => {
    await fs.mkdir(userDir, { recursive: true })
    await fs.writeFile(path.join(userDir, "README.md"), "not a config")
    await writeConfig(userDir, "real.jsonc", "real")

    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })

    expect(result.map((c) => c.name)).toEqual(["real"])
  })

  it("returns an empty list when both dirs are missing", async () => {
    const result = await discoverConfigs({ userDir: "/nonexistent-a", projectDir: "/nonexistent-b" })
    expect(result).toEqual([])
  })

  it("skips files that fail to parse without crashing the whole discovery", async () => {
    await fs.mkdir(userDir, { recursive: true })
    await fs.writeFile(path.join(userDir, "broken.jsonc"), "{ this is not valid json")
    await writeConfig(userDir, "ok.jsonc", "ok")

    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })

    expect(result.map((c) => c.name)).toEqual(["ok"])
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/ui/picker.test.ts
```

- [ ] **Step 3: Implement `src/ui/picker.ts` (discovery half only — picker UI added in Task 5)**

```ts
import fs from "node:fs/promises"
import path from "node:path"
import { parseJSONC } from "confbox/jsonc"
import { parseYAML } from "confbox/yaml"
import { parseTOML } from "confbox/toml"

export type ConfigSource = "user" | "project"

export type DiscoveredConfig = {
  name: string
  description?: string
  path: string
  source: ConfigSource
}

const CONFIG_EXTENSIONS = [".jsonc", ".json", ".yaml", ".yml", ".toml"]

type DiscoveryDirs = {
  userDir: string
  projectDir: string
}

/**
 * Discover configs by scanning two directories. The picker UX shows a flat union;
 * on name collision, the project-local copy wins (it's closer to the user's pwd
 * and presumably more relevant in-context).
 *
 * Files are parsed lightly: we read `name` and `description` without resolving
 * `extends:` (that's expensive; pickers should be fast). Files that fail to parse
 * are silently skipped so one broken file doesn't break the whole picker.
 */
export async function discoverConfigs(dirs: DiscoveryDirs): Promise<DiscoveredConfig[]> {
  const userConfigs = await scanDir(dirs.userDir, "user")
  const projectConfigs = await scanDir(dirs.projectDir, "project")

  // Project wins on name collision.
  const seen = new Set<string>()
  const result: DiscoveredConfig[] = []

  for (const c of projectConfigs) {
    if (!seen.has(c.name)) {
      seen.add(c.name)
      result.push(c)
    }
  }
  for (const c of userConfigs) {
    if (!seen.has(c.name)) {
      seen.add(c.name)
      result.push(c)
    }
  }

  return result
}

async function scanDir(dir: string, source: ConfigSource): Promise<DiscoveredConfig[]> {
  let entries: string[]
  try {
    entries = await fs.readdir(dir)
  } catch {
    return []  // dir doesn't exist
  }

  const configs: DiscoveredConfig[] = []
  for (const entry of entries) {
    const ext = CONFIG_EXTENSIONS.find((e) => entry.toLowerCase().endsWith(e))
    if (!ext) continue

    const filePath = path.join(dir, entry)
    try {
      const text = await fs.readFile(filePath, "utf8")
      const parsed = parseByExt(ext, text) as { name?: unknown; description?: unknown }
      if (typeof parsed.name === "string") {
        configs.push({
          name: parsed.name,
          description: typeof parsed.description === "string" ? parsed.description : undefined,
          path: filePath,
          source,
        })
      }
    } catch {
      // skip broken files
    }
  }
  return configs
}

function parseByExt(ext: string, text: string): unknown {
  switch (ext) {
    case ".jsonc":
    case ".json":
      return parseJSONC(text)
    case ".yaml":
    case ".yml":
      return parseYAML(text)
    case ".toml":
      return parseTOML(text)
  }
  return null
}
```

- [ ] **Step 4: Run (PASS — 6 tests)**

```bash
bun test src/ui/picker.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/ui/picker.ts src/ui/picker.test.ts
git commit -m "feat(ui): add discoverConfigs (XDG + ./configs/, project wins on collision)"
```

---

## Task 5: Picker UI + auto-extract on first run

Adds the actual `pickConfig` UI that uses Clack's `select` to let the user choose a config. Also: when discovery finds zero configs in the user dir, auto-extract the embedded defaults first, then re-discover. This gives the "just works" first-run experience.

**Files:**
- Modify: `src/ui/picker.ts`, `src/ui/picker.test.ts`

- [ ] **Step 1: Restructure `src/ui/picker.test.ts` — add the clack mock at top, then existing tests, then pickConfig tests**

The Task 4 test file has no clack mock (discoverConfigs doesn't use clack). Add it now, BEFORE the `discoverConfigs` import. The full restructured file:

```ts
import { describe, it, expect, afterEach, mock } from "bun:test"
import fs from "node:fs/promises"
import path from "node:path"

// Mock @clack/prompts BEFORE importing the picker module so the import sees the mock.
const clackSelectMock = mock(async (opts: { options: { value: string }[] }) => opts.options[0]?.value)
const clackIsCancelMock = mock((_v: unknown) => false)
mock.module("@clack/prompts", () => ({
  select: clackSelectMock,
  isCancel: clackIsCancelMock,
  cancel: mock(() => undefined),
}))

import { discoverConfigs, pickConfig } from "./picker"

const userDir = path.join("/tmp", `gearup-discover-user-${process.pid}`)
const projectDir = path.join("/tmp", `gearup-discover-proj-${process.pid}`)

afterEach(async () => {
  await fs.rm(userDir, { recursive: true, force: true })
  await fs.rm(projectDir, { recursive: true, force: true })
  clackSelectMock.mockClear()
  clackIsCancelMock.mockClear()
  clackIsCancelMock.mockImplementation(() => false)
})

async function writeConfig(dir: string, filename: string, name: string, description?: string) {
  await fs.mkdir(dir, { recursive: true })
  const obj: Record<string, unknown> = { version: 1, name }
  if (description) obj.description = description
  await fs.writeFile(path.join(dir, filename), JSON.stringify(obj))
}

describe("discoverConfigs", () => {
  it("returns configs from user dir labeled 'user'", async () => {
    await writeConfig(userDir, "base.jsonc", "base", "Core tools")
    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })
    expect(result).toHaveLength(1)
    expect(result[0]?.name).toBe("base")
    expect(result[0]?.description).toBe("Core tools")
    expect(result[0]?.source).toBe("user")
  })

  it("returns configs from project dir labeled 'project'", async () => {
    await writeConfig(projectDir, "team.jsonc", "Team", "Team setup")
    const result = await discoverConfigs({ userDir: "/nonexistent", projectDir })
    expect(result).toHaveLength(1)
    expect(result[0]?.source).toBe("project")
  })

  it("project-local wins on name collision; user copy is dropped", async () => {
    await writeConfig(userDir, "shared.jsonc", "shared", "User version")
    await writeConfig(projectDir, "shared.jsonc", "shared", "Project version")
    const result = await discoverConfigs({ userDir, projectDir })
    expect(result).toHaveLength(1)
    expect(result[0]?.source).toBe("project")
    expect(result[0]?.description).toBe("Project version")
  })

  it("ignores files without a recognized extension", async () => {
    await fs.mkdir(userDir, { recursive: true })
    await fs.writeFile(path.join(userDir, "README.md"), "not a config")
    await writeConfig(userDir, "real.jsonc", "real")
    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })
    expect(result.map((c) => c.name)).toEqual(["real"])
  })

  it("returns an empty list when both dirs are missing", async () => {
    const result = await discoverConfigs({ userDir: "/nonexistent-a", projectDir: "/nonexistent-b" })
    expect(result).toEqual([])
  })

  it("skips files that fail to parse without crashing the whole discovery", async () => {
    await fs.mkdir(userDir, { recursive: true })
    await fs.writeFile(path.join(userDir, "broken.jsonc"), "{ this is not valid json")
    await writeConfig(userDir, "ok.jsonc", "ok")
    const result = await discoverConfigs({ userDir, projectDir: "/nonexistent" })
    expect(result.map((c) => c.name)).toEqual(["ok"])
  })
})

describe("pickConfig", () => {
  it("auto-selects when only one config is available (no prompt)", async () => {
    const configs = [
      { name: "only", path: "/tmp/only.jsonc", source: "user" as const },
    ]
    const choice = await pickConfig(configs)
    expect(choice).toBe("/tmp/only.jsonc")
    expect(clackSelectMock).not.toHaveBeenCalled()
  })

  it("prompts via clack.select when multiple configs are available", async () => {
    clackSelectMock.mockImplementation(
      async (opts: { options: { value: string }[] }) => opts.options[1]?.value,
    )

    const configs = [
      { name: "first", path: "/tmp/first.jsonc", source: "user" as const },
      { name: "second", path: "/tmp/second.jsonc", source: "project" as const },
    ]
    const choice = await pickConfig(configs)
    expect(choice).toBe("/tmp/second.jsonc")
    expect(clackSelectMock).toHaveBeenCalled()
  })

  it("returns null when the user cancels", async () => {
    const cancelSym = Symbol.for("clack:cancel")
    clackSelectMock.mockImplementation(async () => cancelSym as unknown as string)
    clackIsCancelMock.mockImplementation(() => true)

    const configs = [
      { name: "a", path: "/tmp/a.jsonc", source: "user" as const },
      { name: "b", path: "/tmp/b.jsonc", source: "user" as const },
    ]
    const choice = await pickConfig(configs)
    expect(choice).toBe(null)
  })

  it("throws when configs list is empty", async () => {
    await expect(pickConfig([])).rejects.toThrow(/no configs/i)
  })
})
```

**Note:** This REPLACES the file from Task 4 (which had only the discoverConfigs tests). Task 4's discoverConfigs tests are preserved verbatim above; Task 5's pickConfig tests are appended. The file is now the final state.

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/ui/picker.test.ts
```

- [ ] **Step 3: Append `pickConfig` to `src/ui/picker.ts`**

Add this at the bottom of the file:

```ts
import * as clack from "@clack/prompts"

/**
 * Show a Clack select prompt of available configs. Returns the chosen config's
 * path, or null if the user cancels (Ctrl-C / Escape).
 *
 * If exactly one config is available, returns its path immediately without
 * prompting — picking from a list of one is wasteful.
 */
export async function pickConfig(configs: DiscoveredConfig[]): Promise<string | null> {
  if (configs.length === 0) {
    throw new Error("no configs to pick from")
  }
  if (configs.length === 1) {
    return configs[0]!.path
  }

  const choice = await clack.select({
    message: "Pick a config",
    options: configs.map((c) => ({
      label: c.name,
      value: c.path,
      hint: c.description
        ? `${c.description} [${c.source}]`
        : `[${c.source}]`,
    })),
  })

  if (clack.isCancel(choice)) {
    return null
  }
  return choice as string
}
```

- [ ] **Step 4: Run (PASS — total 10 tests in this file)**

```bash
bun test src/ui/picker.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/ui/picker.ts src/ui/picker.test.ts
git commit -m "feat(ui): add pickConfig (Clack select; auto-select when only one available)"
```

---

## Task 6: Progress reporter for runInstall

Adds a `ProgressReporter` interface that the runner emits step-level events on. The runner takes it as an optional parameter; tests pass nothing or a `FakeReporter`; the run command (Task 7) implements it with Clack's spinner.

**Files:**
- Create: `src/runner/progress.ts`, `src/runner/progress.test.ts`
- Modify: `src/runner/run.ts`, `src/runner/run.test.ts`

- [ ] **Step 1: Create `src/runner/progress.ts`**

```ts
/**
 * ProgressReporter is a thin event surface the runner uses to inform a UI layer
 * about per-step progress without knowing what the UI looks like.
 *
 * - `start(label)` is called when a step begins (before its `check`).
 * - `update(label)` may be called multiple times during the step (e.g., transitioning
 *   from "checking..." to "installing..." to "post-install...").
 * - `finish(label)` is called once when the step ends (success or failure).
 *
 * Implementations are expected to track a single in-flight step at a time.
 */
export interface ProgressReporter {
  start(label: string): void
  update(label: string): void
  finish(label: string): void
}

/** No-op reporter — used in tests by default. */
export class NoopReporter implements ProgressReporter {
  start(_label: string): void {}
  update(_label: string): void {}
  finish(_label: string): void {}
}

/** Reporter that records every call, for use in tests that want to assert on events. */
export class FakeReporter implements ProgressReporter {
  events: { kind: "start" | "update" | "finish"; label: string }[] = []
  start(label: string): void {
    this.events.push({ kind: "start", label })
  }
  update(label: string): void {
    this.events.push({ kind: "update", label })
  }
  finish(label: string): void {
    this.events.push({ kind: "finish", label })
  }
}
```

- [ ] **Step 2: Create `src/runner/progress.test.ts`**

```ts
import { describe, it, expect } from "bun:test"
import { FakeReporter, NoopReporter } from "./progress"

describe("FakeReporter", () => {
  it("records start/update/finish events in order", () => {
    const r = new FakeReporter()
    r.start("a")
    r.update("a-mid")
    r.finish("a-done")
    r.start("b")
    r.finish("b-done")

    expect(r.events).toEqual([
      { kind: "start", label: "a" },
      { kind: "update", label: "a-mid" },
      { kind: "finish", label: "a-done" },
      { kind: "start", label: "b" },
      { kind: "finish", label: "b-done" },
    ])
  })
})

describe("NoopReporter", () => {
  it("does nothing without throwing", () => {
    const r = new NoopReporter()
    r.start("a")
    r.update("b")
    r.finish("c")
    expect(true).toBe(true)
  })
})
```

- [ ] **Step 3: Run progress tests**

```bash
bun test src/runner/progress.test.ts
```
Expected: 2 tests pass.

- [ ] **Step 4: Modify `src/runner/run.ts` — add ProgressReporter parameter to `runInstall`**

The existing `runInstall(config, ctx)` becomes `runInstall(config, ctx, progress?)`. The body emits start/update/finish around each step.

Find the existing `runInstall` function. Replace its signature and add reporter calls. Add this import at the top:

```ts
import type { ProgressReporter } from "./progress"
import { NoopReporter } from "./progress"
```

Replace the function. The KEY change is wrapping each step iteration with reporter calls; the rest of the orchestration (elevation pre-check, dispatch, post_install) is unchanged. Find the existing iteration loop and replace with:

```ts
export async function runInstall(
  config: Config,
  ctx: Context,
  progress: ProgressReporter = new NoopReporter(),
): Promise<RunReport> {
  const allSteps = config.steps ?? []
  const completed: InstallStepReport[] = []

  // Partition into elevation-required vs regular, preserving relative order.
  const elevSteps = allSteps.filter((s) => s.requires_elevation === true)
  const regSteps = allSteps.filter((s) => s.requires_elevation !== true)

  // Pre-check: any elevation step that needs install?
  // Deliberate: if a step has requires_elevation: true but config.elevation is
  // absent, no banner is shown — matches Go's runner.go runLive behavior.
  let needsElevation = false
  if (elevSteps.length > 0 && config.elevation) {
    for (const step of elevSteps) {
      const checked = await safeCheck(step, ctx)
      if (!checked.ok) {
        return {
          ok: false, configName: config.name, steps: completed,
          failedAt: step.name, error: `check failed for ${step.name}: ${checked.error}`,
        }
      }
      if (!checked.result.installed) {
        needsElevation = true
        break
      }
    }
  }

  if (needsElevation && config.elevation) {
    const acquired = await acquireElevation(config.elevation)
    if (!acquired.ok) {
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: "elevation", error: acquired.reason,
      }
    }
  }

  const ordered = needsElevation ? [...elevSteps, ...regSteps] : allSteps
  const total = ordered.length

  for (let i = 0; i < ordered.length; i++) {
    const step = ordered[i]!
    const stepNum = i + 1
    const prefix = `[${stepNum}/${total}] ${step.name}`

    progress.start(`${prefix}: checking…`)

    const checked = await safeCheck(step, ctx)
    if (!checked.ok) {
      progress.finish(`${prefix}: check failed`)
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: step.name, error: `check failed for ${step.name}: ${checked.error}`,
      }
    }

    if (checked.result.installed) {
      progress.finish(`${prefix}: already installed`)
      completed.push({ name: step.name, type: step.type, action: "skipped" })
      continue
    }

    progress.update(`${prefix}: installing…`)

    const installResult = await dispatchInstall(step, ctx)
    if (!installResult.ok) {
      progress.finish(`${prefix}: install failed`)
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: step.name, error: installResult.error,
      }
    }

    if (step.post_install && step.post_install.length > 0) {
      progress.update(`${prefix}: post-install…`)
      const postResult = await runPostInstall(step.post_install, step.name, ctx)
      if (!postResult.ok) {
        progress.finish(`${prefix}: post-install failed`)
        return {
          ok: false, configName: config.name, steps: completed,
          failedAt: step.name, error: postResult.error,
        }
      }
    }

    progress.finish(`${prefix}: installed`)
    completed.push({ name: step.name, type: step.type, action: "installed" })
  }

  return { ok: true, configName: config.name, steps: completed }
}
```

- [ ] **Step 5: Add a runner test verifying ProgressReporter integration**

Append to `src/runner/run.test.ts`:

```ts
import { FakeReporter } from "./progress"

describe("runInstall ProgressReporter integration", () => {
  it("emits start/finish around each step (skip path)", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // step check: installed
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "skip-only",
      steps: [{ type: "brew", name: "jq", formula: "jq" }],
    }

    const reporter = new FakeReporter()
    const report = await runInstall(config, ctx, reporter)

    expect(report.ok).toBe(true)
    expect(reporter.events).toEqual([
      { kind: "start", label: "[1/1] jq: checking…" },
      { kind: "finish", label: "[1/1] jq: already installed" },
    ])
  })

  it("emits start/update/finish around install + post_install", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })  // check: not installed
    exec.queueResponse({ exitCode: 0 })  // install: ok
    exec.queueResponse({ exitCode: 0 })  // post_install: ok
    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "with-post",
      steps: [
        {
          type: "shell",
          name: "rust",
          install: "true",
          check: "false",
          post_install: ["echo done"],
        },
      ],
    }

    const reporter = new FakeReporter()
    await runInstall(config, ctx, reporter)

    const kinds = reporter.events.map((e) => e.kind)
    expect(kinds).toEqual(["start", "update", "update", "finish"])
    expect(reporter.events[0]?.label).toContain("checking")
    expect(reporter.events[1]?.label).toContain("installing")
    expect(reporter.events[2]?.label).toContain("post-install")
    expect(reporter.events[3]?.label).toContain("installed")
  })
})
```

- [ ] **Step 6: Run runner tests + full suite**

```bash
bun test src/runner/run.test.ts
bun test
bun run typecheck
```
All green.

- [ ] **Step 7: Commit**

```bash
git add src/runner/progress.ts src/runner/progress.test.ts src/runner/run.ts src/runner/run.test.ts
git commit -m "feat(runner): add ProgressReporter; emit step lifecycle events from runInstall"
```

---

## Task 7: Clack reporter for the run command

Implements `ProgressReporter` using Clack's `spinner()`. Each step gets one spinner; `start` opens it, `update` changes the message, `finish` stops it with a final label.

**Files:**
- Create: `src/ui/reporter.ts`, `src/ui/reporter.test.ts`

- [ ] **Step 1: Failing test — `src/ui/reporter.test.ts`**

```ts
import { describe, it, expect, mock, beforeEach } from "bun:test"

const spinnerMock = {
  start: mock<(label: string) => void>(() => undefined),
  message: mock<(label: string) => void>(() => undefined),
  stop: mock<(label: string) => void>(() => undefined),
}

mock.module("@clack/prompts", () => ({
  spinner: () => spinnerMock,
}))

import { createClackReporter } from "./reporter"

beforeEach(() => {
  spinnerMock.start.mockClear()
  spinnerMock.message.mockClear()
  spinnerMock.stop.mockClear()
})

describe("createClackReporter", () => {
  it("calls spinner.start when a step starts", () => {
    const r = createClackReporter()
    r.start("[1/3] jq: checking…")
    expect(spinnerMock.start).toHaveBeenCalledWith("[1/3] jq: checking…")
  })

  it("calls spinner.message on update", () => {
    const r = createClackReporter()
    r.start("[1/3] jq: checking…")
    r.update("[1/3] jq: installing…")
    expect(spinnerMock.message).toHaveBeenCalledWith("[1/3] jq: installing…")
  })

  it("calls spinner.stop on finish", () => {
    const r = createClackReporter()
    r.start("[1/3] jq: checking…")
    r.finish("[1/3] jq: installed")
    expect(spinnerMock.stop).toHaveBeenCalledWith("[1/3] jq: installed")
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/ui/reporter.test.ts
```

- [ ] **Step 3: Implement `src/ui/reporter.ts`**

```ts
import * as clack from "@clack/prompts"
import type { ProgressReporter } from "../runner/progress"

/**
 * Build a ProgressReporter that drives a single Clack spinner per step. Steps
 * that arrive before the previous one's `finish` would clobber the spinner;
 * runner.ts is expected to call start/finish in matched pairs.
 */
export function createClackReporter(): ProgressReporter {
  let s: ReturnType<typeof clack.spinner> | null = null

  return {
    start(label) {
      s = clack.spinner()
      s.start(label)
    },
    update(label) {
      s?.message(label)
    },
    finish(label) {
      s?.stop(label)
      s = null
    },
  }
}
```

- [ ] **Step 4: Run (PASS — 3 tests)**

```bash
bun test src/ui/reporter.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/ui/reporter.ts src/ui/reporter.test.ts
git commit -m "feat(ui): add createClackReporter (ProgressReporter using Clack spinner)"
```

---

## Task 8: Wire Clack into the run command

Replaces the run command's plain console.log output with Clack's `intro`/`outro`/`cancel` flow + the spinner reporter.

**Files:**
- Modify: `src/commands/run.ts`, `src/commands/run.test.ts`

- [ ] **Step 1: Replace `src/commands/run.ts`**

```ts
import { defineCommand } from "citty"
import * as clack from "@clack/prompts"
import { loadConfig } from "../config/load"
import { runInstall } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"
import { LoggingExec } from "../exec/logging"
import { openFileLogger } from "../log/file"
import { logFilePath } from "../log/xdg"
import { createClackReporter } from "../ui/reporter"

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

    clack.intro(`gearup run · ${config.name}`)

    try {
      const reporter = createClackReporter()
      const report = await runInstall(config, ctx, reporter)

      if (!report.ok) {
        clack.cancel(`Failed at step: ${report.failedAt}`)
        clack.note(report.error, "error")
        clack.outro(`Log: ${logger.path()}`)
        return 1
      }

      clack.outro(`Done · Log: ${logger.path()}`)
      return 0
    } finally {
      await logger.close()
    }
  },
})
```

- [ ] **Step 2: Modify `src/commands/run.test.ts`**

The existing tests must adapt to the new Clack-driven output. The mock at the top must include `intro`, `outro`, `cancel`, `note`, and `spinner`. Replace the file with:

```ts
import { describe, it, expect, spyOn, mock, afterEach } from "bun:test"

const spinnerMock = {
  start: mock<(label: string) => void>(() => undefined),
  message: mock<(label: string) => void>(() => undefined),
  stop: mock<(label: string) => void>(() => undefined),
}

mock.module("@clack/prompts", () => ({
  intro: mock<(label: string) => void>(() => undefined),
  outro: mock<(label: string) => void>(() => undefined),
  cancel: mock<(label: string) => void>(() => undefined),
  note: mock<(message: string, title?: string) => void>(() => undefined),
  spinner: () => spinnerMock,
  confirm: mock(async () => true),
  isCancel: () => false,
}))

import { runCommand } from "citty"
import path from "node:path"
import fs from "node:fs/promises"
import { runCommand as gearupRunCommand } from "./run"
import * as clack from "@clack/prompts"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")
const tmpStateDir = path.join("/tmp", `gearup-run-cmd-test-${process.pid}`)

process.env.XDG_STATE_HOME = tmpStateDir

afterEach(async () => {
  await fs.rm(tmpStateDir, { recursive: true, force: true })
})

describe("run command (Clack UX)", () => {
  it("emits intro and outro on success; returns 0", async () => {
    const tmpMarker = "/tmp/gearup-test-clack-run-success"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__clack-run-success.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "clack-success",
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
      expect(clack.intro).toHaveBeenCalled()
      expect(clack.outro).toHaveBeenCalled()
      expect(clack.cancel).not.toHaveBeenCalled()
    } finally {
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }
  })

  it("emits cancel + note + outro on failure; returns 1", async () => {
    const fixturePath = path.join(fixtures, "__clack-run-fail.jsonc")
    await fs.writeFile(
      fixturePath,
      JSON.stringify({
        version: 1,
        name: "clack-fail",
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
      expect(clack.cancel).toHaveBeenCalled()
      expect(clack.outro).toHaveBeenCalled()  // log path still surfaced
    } finally {
      await fs.unlink(fixturePath).catch(() => undefined)
    }
  })
})
```

- [ ] **Step 3: Run command tests**

```bash
bun test src/commands/run.test.ts
```
Expected: 2 tests pass.

- [ ] **Step 4: Run full suite**

```bash
bun test
bun run typecheck
```
All green.

- [ ] **Step 5: Commit**

```bash
git add src/commands/run.ts src/commands/run.test.ts
git commit -m "feat(cli): wrap run command in Clack intro/outro/cancel + spinner reporter"
```

---

## Task 9: Wire Clack into the plan command

Plan is fast (no install) so a per-step spinner is overkill — just intro/outro and a `note` block listing the steps.

**Files:**
- Modify: `src/commands/plan.ts`, `src/commands/plan.test.ts`

- [ ] **Step 1: Replace `src/commands/plan.ts`**

```ts
import { defineCommand } from "citty"
import * as clack from "@clack/prompts"
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

    clack.intro(`gearup plan · ${config.name}`)
    const report = await runPlan(config, ctx)

    const lines = report.steps.map((s, i) => {
      const idx = `[${i + 1}/${report.steps.length}]`
      const marker = s.status === "installed" ? "✓" : "·"
      const label = s.status === "installed" ? "already installed" : "would install"
      return `${marker} ${idx} ${s.name}  ${label}`
    })
    if (lines.length > 0) {
      clack.note(lines.join("\n"), `${report.steps.length} step${report.steps.length === 1 ? "" : "s"}`)
    }

    const wouldInstall = report.steps.filter((s) => s.status === "would-install").length
    if (wouldInstall === 0) {
      clack.outro("Machine is up to date")
    } else {
      clack.outro(`${wouldInstall} step${wouldInstall === 1 ? "" : "s"} would install`)
    }
    return report.exitCode
  },
})

// Re-exported for tests' convenience.
export type { StepReport }
```

- [ ] **Step 2: Replace `src/commands/plan.test.ts`**

```ts
import { describe, it, expect, mock } from "bun:test"

mock.module("@clack/prompts", () => ({
  intro: mock<(label: string) => void>(() => undefined),
  outro: mock<(label: string) => void>(() => undefined),
  note: mock<(message: string, title?: string) => void>(() => undefined),
  cancel: mock<(label: string) => void>(() => undefined),
  spinner: () => ({ start: mock(), message: mock(), stop: mock() }),
  confirm: mock(async () => true),
  isCancel: () => false,
}))

import { runCommand } from "citty"
import path from "node:path"
import { planCommand } from "./plan"
import * as clack from "@clack/prompts"

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("plan command (Clack UX)", () => {
  it("returns the runner's exit code (10 when something would install)", async () => {
    const { result } = await runCommand(planCommand, {
      rawArgs: ["--config", path.join(fixtures, "never-installed.jsonc")],
    })
    expect(result).toBe(10)
    expect(clack.intro).toHaveBeenCalled()
    expect(clack.outro).toHaveBeenCalled()
  })

  it("rejects when no config is given (citty raises)", async () => {
    await expect(runCommand(planCommand, { rawArgs: [] })).rejects.toThrow()
  })
})
```

- [ ] **Step 3: Run plan tests + full suite**

```bash
bun test src/commands/plan.test.ts
bun test
bun run typecheck
```

- [ ] **Step 4: Commit**

```bash
git add src/commands/plan.ts src/commands/plan.test.ts
git commit -m "feat(cli): wrap plan command in Clack intro/outro/note"
```

---

## Task 10: Wire init + picker into the CLI entrypoint

Two changes to `src/cli.ts`:
1. Add `init` to the subCommands map.
2. When the user runs `gearup plan` or `gearup run` **without** `--config`, run discovery + picker first; if the picker returns a path, re-dispatch with `--config <picked-path>`.

**Files:**
- Modify: `src/cli.ts`

- [ ] **Step 1: Read current `src/cli.ts`**

```bash
cat src/cli.ts
```

- [ ] **Step 2: Replace `src/cli.ts`**

```ts
#!/usr/bin/env bun
import { type CommandDef, defineCommand, runMain, runCommand } from "citty"
import * as clack from "@clack/prompts"
import path from "node:path"
import { planCommand } from "./commands/plan"
import { runCommand as gearupRunCommand } from "./commands/run"
import { versionCommand } from "./commands/version"
import { initCommand, userConfigsDir } from "./commands/init"
import { discoverConfigs, pickConfig } from "./ui/picker"
import { extractConfigs } from "./configs/extract"
import pkg from "../package.json" with { type: "json" }

const mainCommand = defineCommand({
  meta: {
    name: "gearup",
    version: pkg.version,
    description: "Config-driven macOS developer-machine bootstrap",
  },
  subCommands: {
    plan: planCommand,
    run: gearupRunCommand,
    init: initCommand,
    version: versionCommand,
  },
})

const rawArgs = process.argv.slice(2)

// Dispatch known subcommands via runCommand so numeric exit codes are surfaced.
// citty's runMain discards subcommand return values.
// CommandDef<any> avoids a spurious contravariance error from mismatched ArgsDef shapes.
const subCommands: Record<string, CommandDef<any>> = {
  plan: planCommand,
  run: gearupRunCommand,
  init: initCommand,
  version: versionCommand,
}
const cmdName = rawArgs[0]

const isHelp = rawArgs.includes("--help") || rawArgs.includes("-h")

async function ensureConfigPath(subcommand: string, args: string[]): Promise<string[] | null> {
  // If --config is already provided, nothing to do.
  if (args.includes("--config")) return args

  // Discover available configs. Auto-extract embedded defaults on first run.
  const userDir = userConfigsDir(process.env)
  const projectDir = path.resolve(process.cwd(), "configs")

  let configs = await discoverConfigs({ userDir, projectDir })

  if (configs.length === 0) {
    // First run: extract embedded defaults to user dir, then re-discover.
    await extractConfigs(userDir, false)
    configs = await discoverConfigs({ userDir, projectDir })
    if (configs.length === 0) {
      clack.cancel(`No configs found. Try \`gearup init\`.`)
      return null
    }
  }

  const choice = await pickConfig(configs)
  if (choice === null) {
    clack.cancel("Cancelled")
    return null
  }
  return [...args, "--config", choice]
}

async function main() {
  if (cmdName && cmdName in subCommands && !isHelp) {
    let dispatchArgs = rawArgs.slice(1)

    // For plan and run, if --config is missing, present the picker.
    if ((cmdName === "plan" || cmdName === "run")) {
      const augmented = await ensureConfigPath(cmdName, dispatchArgs)
      if (augmented === null) {
        process.exit(0)
      }
      dispatchArgs = augmented
    }

    const { result } = await runCommand(subCommands[cmdName]!, { rawArgs: dispatchArgs })
    if (typeof result === "number" && result !== 0) {
      process.exit(result)
    }
  } else {
    runMain(mainCommand)
  }
}

main().catch((err) => {
  console.error(err instanceof Error ? err.message : String(err))
  process.exit(1)
})
```

- [ ] **Step 3: Smoke-test**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun run src/cli.ts --help            # lists plan, run, init, version
bun run src/cli.ts version           # prints version
bun run src/cli.ts init --help       # shows --force flag
bun run src/cli.ts plan --config tests/fixtures/single-brew.jsonc  # works as before
```

- [ ] **Step 4: Run full suite**

```bash
bun test
bun run typecheck
```
All green.

- [ ] **Step 5: Commit**

```bash
git add src/cli.ts
git commit -m "feat(cli): wire init command; auto-pick when --config omitted; auto-extract on first run"
```

---

## Task 11: Phase 3 deferred cleanup

Three small items from Phase 3's review:

1. `FakeLogger` default sentinel `/dev/null` → `<no log>` (clearer for debugging test failures)
2. Comment in `load.ts` explaining the explicit-extension requirement for `extends:` array entries
3. Edge-case test for `logDir` with unusual `HOME` values (whitespace, special chars)

**Files:**
- Modify: `src/log/fake.ts`, `src/log/fake.test.ts`
- Modify: `src/config/load.ts`
- Modify: `src/log/xdg.test.ts`

- [ ] **Step 1: Update `FakeLogger` default sentinel**

In `src/log/fake.ts`, change the default from `/fake/log` to `<no log>`:

```ts
constructor(private syntheticPath: string = "<no log>") {}
```

In `src/log/fake.test.ts`, update the test:

```ts
it("returns its synthetic path string", () => {
  const log = new FakeLogger("/fake/path.log")
  expect(log.path()).toBe("/fake/path.log")
})

it("defaults the synthetic path to <no log>", () => {
  const log = new FakeLogger()
  expect(log.path()).toBe("<no log>")
})
```

Add the second test as a new `it` block.

Also check `src/context.ts` — it constructs `new FakeLogger("/dev/null")` as the default. Update that to use the FakeLogger's own default by removing the argument:

```ts
log: input.log ?? new FakeLogger(),
```

(So uninitialized contexts get `<no log>` instead of `/dev/null`.)

- [ ] **Step 2: Add comment in `src/config/load.ts`**

Find the `c12LoadConfig` call. Add this comment immediately above it:

```ts
// NOTE: c12 auto-resolves extensions ONLY for the entry config (this call's
// `configFile` parameter). Inside `extends:` arrays, references to non-JS files
// MUST include their extension — `extends: ["./base.jsonc"]`, not `["./base"]`.
// Bare names like `"base"` won't resolve and the parent's steps will silently
// fail to merge. (See tests/fixtures/__bare-name-extends.jsonc test in load.test.ts.)
const { config: raw } = await c12LoadConfig({
  cwd,
  configFile,
  rcFile: false,
  globalRc: false,
})
```

- [ ] **Step 3: Add edge-case test in `src/log/xdg.test.ts`**

Append:

```ts
describe("logDir edge cases", () => {
  it("handles HOME containing spaces", () => {
    const result = logDir({ HOME: "/Users/Test User" })
    expect(result).toBe("/Users/Test User/.local/state/gearup/logs")
  })

  it("handles XDG_STATE_HOME with trailing slash", () => {
    const result = logDir({ XDG_STATE_HOME: "/var/state/", HOME: "/Users/test" })
    // pathe.join normalizes the trailing slash
    expect(result).toBe("/var/state/gearup/logs")
  })
})
```

- [ ] **Step 4: Run tests**

```bash
bun test
bun run typecheck
```
All green; new tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/log/fake.ts src/log/fake.test.ts src/log/xdg.test.ts src/config/load.ts src/context.ts
git commit -m "chore: address Phase 3 deferred cleanup (FakeLogger sentinel, load.ts comment, HOME edge cases)"
```

---

## Task 12: Release pipeline (build script + GitHub Actions)

Build the `--compile` matrix locally with a script; CI workflow does the same on tag pushes and uploads to GitHub Releases.

**Files:**
- Create: `scripts/build-release.ts`
- Modify: `package.json` (add `build:release` script)
- Create: `.github/workflows/release.yml` (replaces if existing)
- Modify: `.github/workflows/ci.yml` (move from Go to Bun)

- [ ] **Step 1: Create `scripts/build-release.ts`**

```ts
#!/usr/bin/env bun
import { $ } from "bun"
import fs from "node:fs/promises"
import path from "node:path"
import crypto from "node:crypto"

// Targets to build. macOS-only per the design spec; linux can be added later.
const TARGETS = [
  { target: "bun-darwin-arm64", outdir: "dist/darwin-arm64" },
  { target: "bun-darwin-x64", outdir: "dist/darwin-x64" },
] as const

async function sha256(filePath: string): Promise<string> {
  const buf = await fs.readFile(filePath)
  return crypto.createHash("sha256").update(buf).digest("hex")
}

async function main() {
  // Clean dist/
  await fs.rm("dist", { recursive: true, force: true })

  for (const { target, outdir } of TARGETS) {
    console.log(`Building ${target}...`)
    await fs.mkdir(outdir, { recursive: true })
    const outfile = path.join(outdir, "gearup")

    await $`bun build src/cli.ts --compile --target=${target} --outfile=${outfile}`

    const checksum = await sha256(outfile)
    await fs.writeFile(`${outfile}.sha256`, `${checksum}  gearup\n`)

    const stat = await fs.stat(outfile)
    console.log(`  ${outfile} (${(stat.size / 1024 / 1024).toFixed(1)} MB, sha256: ${checksum.slice(0, 16)}...)`)
  }

  console.log("Done.")
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
```

- [ ] **Step 2: Add `build:release` to `package.json`**

In `package.json`'s `"scripts"` section, ensure there's an entry:

```json
"build:release": "bun run scripts/build-release.ts"
```

(There may already be a placeholder from Phase 1; replace it if so.)

- [ ] **Step 3: Smoke-test the build script locally**

```bash
bun run build:release
ls -la dist/darwin-arm64/
```
Expected: `dist/darwin-arm64/gearup` exists (~70-90 MB), `gearup.sha256` exists with a checksum line.

If the build fails for `bun-darwin-x64` on an arm64 host (cross-compilation), that's expected — Bun's cross-compile may need extra setup on some platforms. As a fallback, comment out the x64 target and note it for the CI workflow (which runs on macos-latest and may have it work natively).

- [ ] **Step 4: Replace `.github/workflows/release.yml`**

```yaml
name: release

on:
  push:
    tags:
      - "v*"

jobs:
  release:
    runs-on: macos-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v2
        with:
          bun-version: latest

      - name: Install dependencies
        run: bun install --frozen-lockfile

      - name: Run tests
        run: bun test

      - name: Typecheck
        run: bun run typecheck

      - name: Build release artifacts
        run: bun run build:release

      - name: Upload to GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/darwin-arm64/gearup
            dist/darwin-arm64/gearup.sha256
            dist/darwin-x64/gearup
            dist/darwin-x64/gearup.sha256
          fail_on_unmatched_files: false
          generate_release_notes: true
```

If a different release-publishing action is preferred, swap it. The artifacts and naming pattern matter most.

- [ ] **Step 5: Replace `.github/workflows/ci.yml`**

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v2
        with:
          bun-version: latest

      - name: Install dependencies
        run: bun install --frozen-lockfile

      - name: Typecheck
        run: bun run typecheck

      - name: Test
        run: bun test

      - name: Smoke test the compiled binary
        run: |
          bun build src/cli.ts --compile --outfile=bin/gearup
          ./bin/gearup --help
          ./bin/gearup plan --config tests/fixtures/single-brew.jsonc || EXIT=$?
          # plan exits 0 or 10 — both are acceptable
          if [ "${EXIT:-0}" != "0" ] && [ "${EXIT:-0}" != "10" ]; then
            echo "unexpected exit code: $EXIT"
            exit 1
          fi
```

- [ ] **Step 6: Commit**

```bash
git add scripts/build-release.ts package.json .github/workflows/
git commit -m "feat(release): add bun build --compile matrix; CI runs Bun tests on macOS"
```

---

## Task 13: Rewrite install.sh

The current `install.sh` downloads a Go-compiled binary by arch. Update to download Bun-compiled artifacts (different naming).

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Read current `install.sh`**

```bash
cat install.sh
```

Note the current pattern (arch detection, download URL, install location).

- [ ] **Step 2: Replace `install.sh`**

```sh
#!/usr/bin/env bash
# Install gearup: detect arch, download the latest Bun-compiled binary from
# GitHub Releases, verify checksum, place at $GEARUP_INSTALL_DIR (default
# ~/.local/bin/).
set -euo pipefail

REPO="${GEARUP_REPO:-danlourenco/gearup}"
INSTALL_DIR="${GEARUP_INSTALL_DIR:-$HOME/.local/bin}"

# Detect platform.
OS=""
case "$(uname -s)" in
  Darwin) OS="darwin" ;;
  *) echo "error: gearup currently supports macOS only (Darwin). Detected: $(uname -s)" >&2; exit 1 ;;
esac

ARCH=""
case "$(uname -m)" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64) ARCH="x64" ;;
  *) echo "error: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# Resolve the latest release tag.
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
TAG="$(curl -fsSL "$LATEST_URL" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
if [ -z "$TAG" ]; then
  echo "error: could not resolve latest release tag from $LATEST_URL" >&2
  exit 1
fi

ARTIFACT_BASE="https://github.com/${REPO}/releases/download/${TAG}"
BINARY_URL="${ARTIFACT_BASE}/gearup"  # GitHub Release asset paths flatten directories
SHA_URL="${BINARY_URL}.sha256"

# Older releases used per-platform subdirs; the matrix build flattens or duplicates
# names. Try both patterns.
if ! curl -fsSL --head -o /dev/null "$BINARY_URL" 2>/dev/null; then
  # Try platform-prefixed name
  BINARY_URL="${ARTIFACT_BASE}/gearup-${OS}-${ARCH}"
  SHA_URL="${BINARY_URL}.sha256"
fi

mkdir -p "$INSTALL_DIR"
TARGET="$INSTALL_DIR/gearup"
TMP="$(mktemp)"
TMP_SHA="$(mktemp)"

cleanup() { rm -f "$TMP" "$TMP_SHA"; }
trap cleanup EXIT

echo "Downloading gearup ${TAG} for ${OS}-${ARCH}..."
curl -fsSL "$BINARY_URL" -o "$TMP"

# Optional checksum verification (only if .sha256 exists).
if curl -fsSL "$SHA_URL" -o "$TMP_SHA" 2>/dev/null; then
  ACTUAL="$(shasum -a 256 "$TMP" | awk '{print $1}')"
  EXPECTED="$(awk '{print $1}' "$TMP_SHA")"
  if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "error: checksum mismatch (expected $EXPECTED, got $ACTUAL)" >&2
    exit 1
  fi
  echo "Checksum verified."
fi

chmod +x "$TMP"
mv "$TMP" "$TARGET"

echo
echo "Installed gearup to $TARGET"
"$TARGET" version

cat <<EOF

Add $INSTALL_DIR to your PATH if it isn't already:

  export PATH="$INSTALL_DIR:\$PATH"

Run \`gearup --help\` to get started.
EOF
```

The fallback to platform-prefixed names (`gearup-darwin-arm64`) handles the case where the release pipeline uploads multiple artifacts under different names. If the release workflow uploads BOTH `dist/darwin-arm64/gearup` AND would prefer a single rename, that's a future cleanup.

- [ ] **Step 3: Smoke-test**

```bash
shellcheck install.sh || echo "(shellcheck not installed; skip)"
bash -n install.sh  # syntax check
```

- [ ] **Step 4: Commit**

```bash
git add install.sh
git commit -m "fix(install.sh): fetch Bun-compiled artifacts; verify checksum"
```

---

## Task 14: Remove Go code

Atomic deletion. Single commit. Everything Go-specific goes.

**Files to DELETE:**
- `cmd/` (entire dir)
- `internal/` (entire dir)
- `go.mod`, `go.sum`
- `.goreleaser.yaml`
- `gearup` (compiled binary if present; should be gitignored anyway)

- [ ] **Step 1: Verify nothing in TS code references Go paths**

```bash
grep -rn "gearup/internal\|gearup/cmd" src/ scripts/ docs/ 2>/dev/null
```
Should find nothing meaningful (only mentions in old plan files / spec history are OK; those are historical).

- [ ] **Step 2: Delete the Go tree**

```bash
rm -rf cmd internal go.mod go.sum .goreleaser.yaml
rm -f gearup  # if present at repo root
```

- [ ] **Step 3: Verify TS still works**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun test
bun run typecheck
bun run src/cli.ts --help
```
Everything green.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove Go implementation; TS port is now the canonical gearup"
```

---

## Task 15: Bump version to 0.3.0

**Files:**
- Modify: `package.json`

- [ ] **Step 1: Update `package.json` version**

Find the line `"version": "0.2.0-alpha.1"` and change it to `"version": "0.3.0"`.

- [ ] **Step 2: Verify**

```bash
bun run src/cli.ts version
```
Expected: `0.3.0`

- [ ] **Step 3: Commit**

```bash
git add package.json
git commit -m "chore: bump to 0.3.0 — first non-alpha release of the TypeScript port"
```

---

## Task 16: Rewrite README

Comprehensive rewrite. The README is what new users see first; every section needs review.

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read the current README**

```bash
cat README.md
```

Note which sections need substantive change vs. light touch-up.

- [ ] **Step 2: Replace `README.md`**

```markdown
# gearup

An open-source, config-driven macOS developer-machine bootstrap CLI built with [Bun](https://bun.sh) and the [unjs](https://unjs.io) ecosystem ([c12](https://github.com/unjs/c12), [confbox](https://github.com/unjs/confbox), [citty](https://github.com/unjs/citty)) plus [Valibot](https://valibot.dev), [execa](https://github.com/sindresorhus/execa), and [Clack](https://github.com/natemoo-re/clack).

Define your team's toolchain in a JSONC, YAML, or TOML config, run `gearup run`, and get a fully provisioned dev machine in minutes. Every step is idempotent — re-running skips what's already installed.

## Quick start

### Install (no Bun required)

```bash
curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/install.sh | bash
```

Detects your architecture (`arm64` / `x64`), downloads the latest Bun-compiled binary to `~/.local/bin/`, verifies the SHA256 checksum, and installs. No `sudo` required.

To install to a different location:

```bash
GEARUP_INSTALL_DIR=~/bin curl -fsSL https://raw.githubusercontent.com/danlourenco/gearup/main/install.sh | bash
```

Default configs (`base`, `backend`, `frontend`, etc.) are embedded in the binary. On first run, they're automatically extracted to `~/.config/gearup/configs/`. To reset defaults or refresh them after a release:

```bash
gearup init          # write any missing defaults; preserve customizations
gearup init --force  # overwrite all customizations with the embedded defaults
```

### Pick and run interactively

```bash
gearup run    # presents a picker of available configs, then runs the chosen one
```

Or specify directly:

```bash
gearup run --config ~/.config/gearup/configs/backend.jsonc
gearup plan --config ~/.config/gearup/configs/backend.jsonc   # check-only
```

### Build from source (requires Bun)

```bash
git clone https://github.com/danlourenco/gearup.git
cd gearup
bun install
bun run src/cli.ts version
bun build src/cli.ts --compile --outfile=bin/gearup   # build local binary
```

## How it works

```
argv → CLI router (citty) → Config loader (c12 + confbox) → Validator (Valibot)
       → Runner → Handler registry → exec (execa, wrapped for logging)
```

Configs are JSONC/YAML/TOML files with an `extends:` array for composition. Each config has a `name`, optional `description`, optional `platform` constraints, optional `elevation:` block, and a `steps` map keyed by step name.

When you run `gearup plan` or `gearup run` and the chosen config has `extends:`, [c12](https://github.com/unjs/c12) recursively loads each referenced config and deep-merges them with [defu](https://github.com/unjs/defu) defaults. **On step name collisions, the current (override) config wins.**

### Picker resolution

When `--config` is omitted, gearup discovers configs in two locations:

1. `~/.config/gearup/configs/` (or `$XDG_CONFIG_HOME/gearup/configs/`) — user-global
2. `./configs/` (relative to your current working directory) — project-local

Configs are presented as a flat union. **On name collision, the project-local config wins** (closer to your `pwd`, presumably more relevant). Each entry shows its `name`, `description`, and source label (`[user]` or `[project]`).

If both locations are empty on first run, gearup auto-extracts the embedded defaults to the user dir before prompting.

### Example config (entry-point)

```jsonc
// ~/.config/gearup/configs/backend.jsonc
{
  "version": 1,
  "name": "Backend",
  "description": "Full macOS developer toolchain for backend/infra work",
  "platform": { "os": ["darwin"] },
  "elevation": {
    "message": "Some steps need admin permissions. Elevate now, then continue.",
    "duration": "180s"
  },
  "extends": [
    "./base.jsonc",
    "./jvm.jsonc",
    "./containers.jsonc",
    "./aws-k8s.jsonc",
    "./node.jsonc"
  ]
}
```

Note: extends array entries **must include the file extension** for non-JS configs. `./base.jsonc` works; `./base` does not.

### Example reusable config

```jsonc
// ~/.config/gearup/configs/base.jsonc
{
  "version": 1,
  "name": "base",
  "description": "Homebrew + universal core CLI tools (git, jq)",
  "steps": {
    "Homebrew": {
      "type": "curl-pipe-sh",
      "url": "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh",
      "check": "command -v brew"
    },
    "Git": {
      "type": "brew",
      "formula": "git",
      "check": "command -v git"
    },
    "jq": {
      "type": "brew",
      "formula": "jq"
    }
  }
}
```

`extends:` and `steps:` can coexist — extended configs' steps are merged in first, then the current config's steps are layered on top with override semantics.

## Step types

Each step in `steps` declares a `type` that determines how it's installed and checked.

| Type | Installs via | Auto-check | Explicit `check:` |
|---|---|---|---|
| `brew` | `brew install <formula>` | `brew list --formula <formula>` | Optional override |
| `brew-cask` | `brew install --cask <cask>` | `brew list --cask <cask>` | Optional override |
| `curl-pipe-sh` | `curl -fsSL <url> \| <shell>` | None | **Required** |
| `git-clone` | `git clone [--branch <ref>] <repo> <dest>` | Directory exists at `dest` | Not needed |
| `shell` | User-provided `install:` command | None | **Required** |

Every step is idempotent: the `check` command runs first, and if it exits 0 the install is skipped.

Any step type can include `post_install:` — a list of shell commands that run after a successful install (skipped if the step was already installed):

```jsonc
{
  "Colima": {
    "type": "brew",
    "formula": "colima",
    "post_install": ["colima start --cpu 4 --memory 8"]
  }
}
```

## Commands

```
gearup plan     [--config <path>]
gearup run      [--config <path>]
gearup init     [--force]
gearup version
```

### `gearup plan`

Runs every step's `check` without installing anything. Prints a styled preview showing what would happen.

Exit codes:
- `0` — machine is fully provisioned (nothing would run)
- `10` — one or more steps would install (CI-friendly: "machine not up to date")

### `gearup run`

Runs `check` per step; if a step is not installed, runs its install. Then runs `post_install` commands if any. Stops on first error (fail-fast). Steps with `requires_elevation: true` run first if the config has an `elevation:` block — gearup shows a confirmation banner before they run.

### `gearup init`

Writes the embedded default configs to `~/.config/gearup/configs/`. Existing files are preserved unless `--force` is passed. Useful for resetting to defaults or seeing what configs ship with the binary.

### Flags

| Flag | Description |
|---|---|
| `--config <path>` | Path to config (JSONC/YAML/TOML; extension-less paths are also accepted). Omit to pick interactively. |
| `--force` | (`init` only) Overwrite existing files. |

## Elevation

Steps that need admin permissions declare `requires_elevation: true`. When a config includes an `elevation:` block, gearup shows a styled banner and waits for confirmation before running those steps:

```jsonc
{
  "elevation": {
    "message": "Some steps need admin permissions. Elevate now, then continue.",
    "duration": "180s"
  }
}
```

gearup never invokes elevation itself — it pauses and lets you acquire permissions through whatever mechanism your organization uses (MDM scripts, Touch ID, native sudo, etc.). If no `elevation:` block is set, steps that need sudo prompt natively.

**Smart suppression:** if all elevation-required steps are already installed, the banner is skipped entirely.

## Log files

Every `gearup run` writes a log file at:

```
$XDG_STATE_HOME/gearup/logs/<YYYYMMDD>-<HHMMSS>-<config>.log
```

(Falls back to `~/.local/state/gearup/logs/` if `XDG_STATE_HOME` is unset.)

Each subprocess invocation is captured: argv, exit code, duration, full stdout, full stderr. The terminal stays clean — only step status (via Clack spinner) and the log path are shown. On failure, the relevant captured output is shown alongside the log path.

## Creating your own configs

Run `gearup init` to extract the default configs at `~/.config/gearup/configs/`. Edit them directly, or use them as templates:

1. Pick a directory for your configs (the user dir, or `./configs/` in your project for team-shared).
2. Write one config file per concern (base tools, language runtimes, cloud tooling, etc.) in JSONC, YAML, or TOML.
3. Create an entry-point config that lists `extends:` with explicit paths and extensions:
   ```jsonc
   {
     "version": 1,
     "name": "my-stack",
     "extends": [
       "./base.jsonc",
       "./node.jsonc",
       "github:my-team/configs/aws.jsonc"
     ]
   }
   ```
4. Run `gearup run --config <your-config.jsonc>` (or just `gearup run` and pick from the list).

`extends:` accepts:
- **Relative paths**: `./base.jsonc` (relative to the current config file)
- **Absolute paths**: `/path/to/base.jsonc`
- **`github:` references**: `github:owner/repo/path/to/file.jsonc` (c12 clones via [giget](https://github.com/unjs/giget))
- **`https:` URLs**: `https://example.com/configs/base.jsonc`

For non-JS configs (jsonc/yaml/toml), the **file extension is required** in extends entries.

## Running tests

```bash
bun test
bun run typecheck
```

## License

[MIT](LICENSE)
```

- [ ] **Step 3: Smoke-check the README renders**

```bash
# If you have a markdown previewer:
mdcat README.md | head -60   # or pandoc README.md -o /tmp/readme.html
```
Visual inspection: headers, code blocks, tables look right.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: rewrite README for the TypeScript port (Bun, Clack, c12, Valibot)"
```

---

## Phase 4 Done — Verify and Push

- [ ] **Step 1: Final verification**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun test
bun run typecheck
git log --oneline main..HEAD | wc -l   # ~16 commits
```

- [ ] **Step 2: Smoke-test the full flow end-to-end**

```bash
# Auto-extract on first run + picker + plan
rm -rf /tmp/fake-config-home
XDG_CONFIG_HOME=/tmp/fake-config-home bun run src/cli.ts plan
# (expect Clack picker; pick "base")

# init + force
bun run src/cli.ts init --force
ls ~/.config/gearup/configs/

# run with --config
bun run src/cli.ts run --config tests/fixtures/safe-install.jsonc
# Should show Clack intro/spinner/outro
rm -f /tmp/gearup-e2e-marker /tmp/gearup-e2e-post

# version
bun run src/cli.ts version    # 0.3.0

# Compile a binary and run it
bun run build:release    # or just `bun build src/cli.ts --compile --outfile=bin/gearup`
./bin/gearup version
```

- [ ] **Step 3: Push and create PR**

```bash
git push -u origin ts-port/phase-4-polish-and-release
gh pr create --title "feat: TS port Phase 4 — polish, init, picker, release pipeline; remove Go" --body "..."
```

(Construct the PR body summarizing all 16 tasks: config conversion, embedding, init, discovery+picker, ProgressReporter, Clack UX on plan/run, Phase 3 cleanup, release pipeline, install.sh rewrite, Go removal, version bump, README rewrite.)

---

## Exit Criteria for Phase 4

1. `gearup plan` and `gearup run` (no `--config`) trigger discovery + picker; pick → re-dispatch with `--config`.
2. `gearup init` (and `--force`) writes embedded defaults to `~/.config/gearup/configs/`.
3. First-run auto-extract: with no configs anywhere, picker triggers extract → re-discover → prompt.
4. Clack `intro/outro/spinner/cancel/note` is the user-facing UX for `plan`, `run`, and `init`.
5. `bun build --compile` produces working `bin/gearup` for `darwin-arm64` and `darwin-x64`.
6. `install.sh` fetches a release binary and verifies its checksum.
7. CI workflow runs Bun tests on macOS; release workflow uploads artifacts on tag push.
8. Go code (`cmd/`, `internal/`, `go.mod`, `.goreleaser.yaml`) is removed.
9. `package.json` is at `0.3.0`.
10. README accurately describes the TS port (Bun build, Clack output, JSONC/YAML/TOML configs, picker resolution rules, c12-native extends).
11. Test count holds (~130+); typecheck exit 0.
12. The PR includes a self-test checklist and merges cleanly.

## What's NOT in Phase 4 (deferred / out of scope)

- Linux release artifacts (would be small to add later if demand exists)
- Streamed (vs buffered) subprocess logging
- A `gearup doctor` / sanity check command
- Plugin system for third-party step types
- Windows support
