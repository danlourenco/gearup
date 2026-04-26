# TS Port — Phase 2 Run + Install + Elevation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `gearup run` (install dispatch) to the TypeScript port. Adds install handlers for all 5 step types, `post_install` hooks, and an elevation pause banner using `@clack/prompts`. Single-file configs only (no `extends:` — that's Phase 3). No XDG logging yet (Phase 3). Fail-fast on first error.

**Architecture:** Extends the Phase 1 pipeline with install dispatch. Each handler now exposes `install(step, ctx)` alongside `check(step, ctx)`. The runner orchestrates: check → skip if installed, else run elevation banner if needed → install → run `post_install` → next step. `@clack/prompts` is brought forward from Phase 4 (one dep) for the elevation confirmation banner; the rest of the styled UI (spinners, intro/outro) remains in Phase 4.

**Tech Stack:** Same as Phase 1 (Bun, citty, Valibot, execa, bun:test) plus `@clack/prompts` for the elevation banner.

---

## Preflight

This plan extends the Phase 1 codebase. Phase 1 is merged to `main`. The Go code is still alongside (deletion is Phase 4).

**Before Task 1, create a worktree for Phase 2:**

```bash
cd /Users/dlo/Dev/gearup
git worktree add .worktrees/phase-2-run -b ts-port/phase-2-run-and-elevation
cd .worktrees/phase-2-run
```

All subsequent tasks assume you are in `.worktrees/phase-2-run/`. The branch merges back to `main` at the end of Phase 2.

**Reference files from Go that Phase 2 mirrors (for behavior parity):**
- `internal/installer/brew/brew.go` — `Install`: `brew install <formula>`
- `internal/installer/brewcask/brewcask.go` — `Install`: `brew install --cask <cask>`
- `internal/installer/curlpipe/curlpipe.go` — `Install`: `curl -fsSL <url> | <shell>` with optional `-s -- <args>`
- `internal/installer/gitclone/gitclone.go` — `Install`: `git clone [--branch <ref>] <repo> <dest>` with parent dir creation and `~/` expansion
- `internal/installer/shell/shell.go` — `Install`: runs `step.install` directly
- `internal/runner/runner.go` — orchestration: `runLive`, partition by elevation, `runPostInstall`, fail-fast
- `internal/elevation/elevation.go` — `Acquire`: banner + confirm

**Behavior contract reminders:**
- **Fail-fast:** First step that fails halts the run. Match Go's behavior exactly.
- **Idempotency:** Always run `check` first; skip if installed.
- **post_install:** Only runs when an actual install happened (not when skipped). Sequential, fail-fast within `post_install` too.
- **Elevation:** Pre-checks all elevation-required steps. If any need install, shows the banner once before running ANY of them. If all are installed, banner is suppressed. Non-elevation steps run after.
- **Spec update:** This plan brings `@clack/prompts` from Phase 4 → Phase 2 (just for the confirm banner). Phase 4 still adds the picker and spinners. Update `docs/superpowers/specs/2026-04-24-typescript-port-design.md` to reflect this in the final cleanup task.

---

## File Structure (additions and modifications in Phase 2)

```
gearup/
├── package.json                      MODIFY — add @clack/prompts
├── src/
│   ├── cli.ts                        MODIFY — add `run` to subCommands map
│   ├── commands/
│   │   ├── run.ts                    NEW — gearup run command
│   │   └── run.test.ts               NEW
│   ├── steps/
│   │   ├── types.ts                  MODIFY — make Handler.install required (no longer optional)
│   │   ├── brew.ts                   MODIFY — add installBrew
│   │   ├── brew.test.ts              MODIFY — add install tests
│   │   ├── brew-cask.ts              MODIFY — add installBrewCask
│   │   ├── brew-cask.test.ts         MODIFY — add install tests
│   │   ├── curl-pipe-sh.ts           MODIFY — add installCurlPipe
│   │   ├── curl-pipe-sh.test.ts      MODIFY — add install tests
│   │   ├── git-clone.ts              MODIFY — add installGitClone
│   │   ├── git-clone.test.ts         MODIFY — add install tests
│   │   ├── shell.ts                  MODIFY — add installShell
│   │   ├── shell.test.ts             MODIFY — add install tests
│   │   ├── post-install.ts           NEW — runs step.post_install commands
│   │   ├── post-install.test.ts      NEW
│   │   └── index.ts                  MODIFY — registry adds install function per handler
│   ├── elevation/
│   │   ├── acquire.ts                NEW — banner + clack confirm
│   │   └── acquire.test.ts           NEW
│   └── runner/
│       ├── run.ts                    MODIFY — add runInstall (full execution path); refactor dispatchCheck if needed
│       └── run.test.ts               MODIFY — add runInstall tests
└── tests/
    └── fixtures/
        ├── safe-install.jsonc        NEW — uses shell type with safe touch/rm commands for e2e
        └── elevation-required.jsonc  NEW — has a step with requires_elevation: true
```

---

## Task 1: Add @clack/prompts dependency

Bring in `@clack/prompts` ahead of Phase 4 for the elevation confirm. One dep, no usage yet.

**Files:** Modify `package.json`

- [ ] **Step 1: Add the dep**

```bash
bun add @clack/prompts@^0.7.0
```

This updates both `package.json` and `bun.lock`.

- [ ] **Step 2: Verify**

```bash
bun run typecheck
```
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add package.json bun.lock
git commit -m "chore: add @clack/prompts for elevation banner (brought forward from Phase 4)"
```

---

## Task 2: brew install handler

Adds `installBrew` to `src/steps/brew.ts`. Runs `brew install <formula>`. Returns `InstallResult`.

**Files:** Modify `src/steps/brew.ts`, `src/steps/brew.test.ts`

- [ ] **Step 1: Append failing tests**

Append to `src/steps/brew.test.ts`:

```ts
import { installBrew } from "./brew"

describe("installBrew", () => {
  it("runs `brew install <formula>` and returns ok on exit 0", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "jq", formula: "jq" }
    const result = await installBrew(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["brew", "install", "jq"])
    expect(exec.calls[0]?.shell).toBeFalsy()
  })

  it("returns ok=false with error when exit code is non-zero", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "Error: No available formula" })
    const ctx = makeContext({ exec })

    const step: BrewStep = { type: "brew", name: "nonexistent", formula: "nonexistent" }
    const result = await installBrew(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("brew install nonexistent")
      expect(result.error).toContain("exit 1")
      expect(result.error).toContain("Error: No available formula")
    }
  })
})
```

- [ ] **Step 2: Run (FAIL — installBrew not exported)**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun test src/steps/brew.test.ts
```
Expected: FAIL — `installBrew` not exported.

- [ ] **Step 3: Implement installBrew**

Append to `src/steps/brew.ts` (keep existing `checkBrew` unchanged, add new `import` for `InstallResult`):

```ts
import type { InstallResult } from "./types"

export async function installBrew(step: BrewStep, ctx: Context): Promise<InstallResult> {
  const result = await ctx.exec.run({ argv: ["brew", "install", step.formula] })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `brew install ${step.formula} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
```

- [ ] **Step 4: Run (PASS — 5 tests total in this file: 3 check + 2 install)**

```bash
bun test src/steps/brew.test.ts
```
Expected: PASS — 5 tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/steps/brew.ts src/steps/brew.test.ts
git commit -m "feat(steps): add brew install handler"
```

---

## Task 3: brew-cask install handler

Adds `installBrewCask`. Runs `brew install --cask <cask>`.

**Files:** Modify `src/steps/brew-cask.ts`, `src/steps/brew-cask.test.ts`

- [ ] **Step 1: Append failing tests**

Append to `src/steps/brew-cask.test.ts`:

```ts
import { installBrewCask } from "./brew-cask"

describe("installBrewCask", () => {
  it("runs `brew install --cask <cask>` and returns ok on exit 0", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "iTerm2", cask: "iterm2" }
    const result = await installBrewCask(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["brew", "install", "--cask", "iterm2"])
    expect(exec.calls[0]?.shell).toBeFalsy()
  })

  it("returns ok=false with error when exit code is non-zero", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "Cask not found" })
    const ctx = makeContext({ exec })

    const step: BrewCaskStep = { type: "brew-cask", name: "missing", cask: "missing" }
    const result = await installBrewCask(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("brew install --cask missing")
      expect(result.error).toContain("exit 1")
      expect(result.error).toContain("Cask not found")
    }
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/steps/brew-cask.test.ts
```

- [ ] **Step 3: Implement installBrewCask**

Append to `src/steps/brew-cask.ts`:

```ts
import type { InstallResult } from "./types"

export async function installBrewCask(step: BrewCaskStep, ctx: Context): Promise<InstallResult> {
  const result = await ctx.exec.run({ argv: ["brew", "install", "--cask", step.cask] })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `brew install --cask ${step.cask} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
```

- [ ] **Step 4: Run (PASS — 5 tests in this file)**

```bash
bun test src/steps/brew-cask.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/steps/brew-cask.ts src/steps/brew-cask.test.ts
git commit -m "feat(steps): add brew-cask install handler"
```

---

## Task 4: curl-pipe-sh install handler

Adds `installCurlPipe`. Runs `curl -fsSL <url> | <shell>` with optional `-s -- <args>`. Default shell is `bash`.

**Files:** Modify `src/steps/curl-pipe-sh.ts`, `src/steps/curl-pipe-sh.test.ts`

- [ ] **Step 1: Append failing tests**

Append to `src/steps/curl-pipe-sh.test.ts`:

```ts
import { installCurlPipe } from "./curl-pipe-sh"

describe("installCurlPipe", () => {
  it("runs `curl -fsSL <url> | bash` by default", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "Homebrew",
      url: "https://example.com/install.sh",
      check: "command -v brew",
    }
    const result = await installCurlPipe(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual(["curl -fsSL https://example.com/install.sh | bash"])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("uses step.shell when provided", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "rust",
      url: "https://sh.rustup.rs",
      shell: "sh",
      check: "command -v rustc",
    }
    await installCurlPipe(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["curl -fsSL https://sh.rustup.rs | sh"])
  })

  it("appends `-s -- <args>` when args is non-empty", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "rust",
      url: "https://sh.rustup.rs",
      shell: "sh",
      args: ["-y", "--default-toolchain", "stable"],
      check: "command -v rustc",
    }
    await installCurlPipe(step, ctx)

    expect(exec.calls[0]?.argv).toEqual([
      "curl -fsSL https://sh.rustup.rs | sh -s -- -y --default-toolchain stable",
    ])
  })

  it("returns ok=false with error on non-zero exit", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 22, stderr: "404 Not Found" })
    const ctx = makeContext({ exec })

    const step: CurlPipeStep = {
      type: "curl-pipe-sh",
      name: "missing",
      url: "https://example.com/404.sh",
      check: "command -v missing",
    }
    const result = await installCurlPipe(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("curl-pipe-sh missing")
      expect(result.error).toContain("exit 22")
    }
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/steps/curl-pipe-sh.test.ts
```

- [ ] **Step 3: Implement installCurlPipe**

Append to `src/steps/curl-pipe-sh.ts`:

```ts
import type { InstallResult } from "./types"

export async function installCurlPipe(step: CurlPipeStep, ctx: Context): Promise<InstallResult> {
  const shell = step.shell ?? "bash"
  let cmd = `curl -fsSL ${step.url} | ${shell}`
  if (step.args && step.args.length > 0) {
    cmd = `${cmd} -s -- ${step.args.join(" ")}`
  }

  const result = await ctx.exec.run({ argv: [cmd], shell: true })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `curl-pipe-sh ${step.name} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
```

- [ ] **Step 4: Run (PASS — 6 tests total in this file: 2 check + 4 install)**

```bash
bun test src/steps/curl-pipe-sh.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/steps/curl-pipe-sh.ts src/steps/curl-pipe-sh.test.ts
git commit -m "feat(steps): add curl-pipe-sh install handler"
```

---

## Task 5: git-clone install handler

Adds `installGitClone`. Creates parent dir, then runs `git clone [--branch <ref>] <repo> <dest>`. Expands `~/` and bare `~` using `ctx.env.HOME` (the same `expandHome` helper from check).

**Files:** Modify `src/steps/git-clone.ts`, `src/steps/git-clone.test.ts`

- [ ] **Step 1: Append failing tests**

Append to `src/steps/git-clone.test.ts`:

```ts
import { installGitClone } from "./git-clone"

describe("installGitClone", () => {
  it("creates the parent dir, then runs `git clone <repo> <dest>` (no ref)", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p
    exec.queueResponse({ exitCode: 0 })  // git clone
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "/tmp/dotfiles",
    }
    const result = await installGitClone(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls).toHaveLength(2)
    expect(exec.calls[0]?.argv).toEqual(["mkdir", "-p", "/tmp"])
    expect(exec.calls[1]?.argv).toEqual([
      "git", "clone", "git@github.com:me/dotfiles.git", "/tmp/dotfiles",
    ])
  })

  it("includes --branch when step.ref is set", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p
    exec.queueResponse({ exitCode: 0 })  // git clone
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "thing",
      repo: "git@github.com:me/thing.git",
      dest: "/tmp/thing",
      ref: "main",
    }
    await installGitClone(step, ctx)

    expect(exec.calls[1]?.argv).toEqual([
      "git", "clone", "--branch", "main",
      "git@github.com:me/thing.git", "/tmp/thing",
    ])
  })

  it("expands ~/ in dest using ctx.env.HOME", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p
    exec.queueResponse({ exitCode: 0 })  // git clone
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "dotfiles",
      repo: "git@github.com:me/dotfiles.git",
      dest: "~/.dotfiles",
    }
    await installGitClone(step, ctx)

    expect(exec.calls[0]?.argv).toEqual(["mkdir", "-p", "/Users/test"])
    expect(exec.calls[1]?.argv?.slice(-1)).toEqual(["/Users/test/.dotfiles"])
  })

  it("returns ok=false when git clone fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })  // mkdir -p ok
    exec.queueResponse({ exitCode: 128, stderr: "fatal: repository not found" })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "missing",
      repo: "git@github.com:me/missing.git",
      dest: "/tmp/missing",
    }
    const result = await installGitClone(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("git clone missing")
      expect(result.error).toContain("exit 128")
    }
  })

  it("returns ok=false when mkdir fails", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "Permission denied" })
    const ctx = makeContext({ exec, env: { HOME: "/Users/test" } })

    const step: GitCloneStep = {
      type: "git-clone",
      name: "x",
      repo: "git@github.com:me/x.git",
      dest: "/root/forbidden/x",
    }
    const result = await installGitClone(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("create parent dir")
    }
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/steps/git-clone.test.ts
```

- [ ] **Step 3: Implement installGitClone**

Append to `src/steps/git-clone.ts`. Re-uses the existing `expandHome` helper:

```ts
import type { InstallResult } from "./types"
import path from "node:path"

export async function installGitClone(step: GitCloneStep, ctx: Context): Promise<InstallResult> {
  const home = ctx.env.HOME ?? ""
  const dest = expandHome(step.dest, home)
  const parent = path.dirname(dest)

  // Ensure parent directory exists.
  const mkdirResult = await ctx.exec.run({ argv: ["mkdir", "-p", parent] })
  if (mkdirResult.exitCode !== 0) {
    return {
      ok: false,
      error: `git-clone ${step.name}: create parent dir ${parent} failed (exit ${mkdirResult.exitCode}): ${mkdirResult.stderr.trim()}`,
    }
  }

  // Build clone command.
  const argv = step.ref
    ? ["git", "clone", "--branch", step.ref, step.repo, dest]
    : ["git", "clone", step.repo, dest]

  const cloneResult = await ctx.exec.run({ argv })
  if (cloneResult.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `git clone ${step.name} failed (exit ${cloneResult.exitCode}): ${cloneResult.stderr.trim()}`,
  }
}
```

- [ ] **Step 4: Run (PASS — 10 tests total in this file: 5 check + 5 install)**

```bash
bun test src/steps/git-clone.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/steps/git-clone.ts src/steps/git-clone.test.ts
git commit -m "feat(steps): add git-clone install handler"
```

---

## Task 6: shell install handler

Adds `installShell`. Runs the user's `step.install` command directly through the shell.

**Files:** Modify `src/steps/shell.ts`, `src/steps/shell.test.ts`

- [ ] **Step 1: Append failing tests**

Append to `src/steps/shell.test.ts`:

```ts
import { installShell } from "./shell"

describe("installShell", () => {
  it("runs step.install in shell mode", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "rust",
      install: "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
      check: "command -v rustc",
    }
    const result = await installShell(step, ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls[0]?.argv).toEqual([step.install])
    expect(exec.calls[0]?.shell).toBe(true)
  })

  it("returns ok=false on non-zero exit", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1, stderr: "boom" })
    const ctx = makeContext({ exec })

    const step: ShellStep = {
      type: "shell",
      name: "broken",
      install: "false",
      check: "command -v broken",
    }
    const result = await installShell(step, ctx)

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("shell broken")
      expect(result.error).toContain("exit 1")
    }
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/steps/shell.test.ts
```

- [ ] **Step 3: Implement installShell**

Append to `src/steps/shell.ts`:

```ts
import type { InstallResult } from "./types"

export async function installShell(step: ShellStep, ctx: Context): Promise<InstallResult> {
  const result = await ctx.exec.run({ argv: [step.install], shell: true })
  if (result.exitCode === 0) return { ok: true }
  return {
    ok: false,
    error: `shell ${step.name} failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
  }
}
```

- [ ] **Step 4: Run (PASS — 4 tests total: 2 check + 2 install)**

```bash
bun test src/steps/shell.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/steps/shell.ts src/steps/shell.test.ts
git commit -m "feat(steps): add shell install handler"
```

---

## Task 7: post_install runner

Runs `step.post_install` shell commands sequentially. Fail-fast within the list. Returns `{ ok: true }` if all succeed, or `{ ok: false, error }` on first failure.

**Files:** Create `src/steps/post-install.ts`, `src/steps/post-install.test.ts`

- [ ] **Step 1: Write failing test**

Create `src/steps/post-install.test.ts`:

```ts
import { describe, it, expect } from "bun:test"
import { runPostInstall } from "./post-install"
import { FakeExec } from "../exec/fake"
import { makeContext } from "../context"

describe("runPostInstall", () => {
  it("runs each command in shell mode and returns ok when all succeed", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    exec.queueResponse({ exitCode: 0 })
    const ctx = makeContext({ exec })

    const result = await runPostInstall(["echo first", "echo second"], "step-name", ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls).toHaveLength(2)
    expect(exec.calls[0]?.argv).toEqual(["echo first"])
    expect(exec.calls[0]?.shell).toBe(true)
    expect(exec.calls[1]?.argv).toEqual(["echo second"])
  })

  it("returns ok on empty list without calling exec", async () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    const result = await runPostInstall([], "step-name", ctx)

    expect(result.ok).toBe(true)
    expect(exec.calls).toHaveLength(0)
  })

  it("stops at the first failing command and reports which one", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 0 })
    exec.queueResponse({ exitCode: 1, stderr: "boom" })
    // Third response would never be consumed if fail-fast works.
    const ctx = makeContext({ exec })

    const result = await runPostInstall(
      ["echo ok", "false", "echo never"],
      "colima",
      ctx,
    )

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.error).toContain("colima")
      expect(result.error).toContain("post_install[1]")
      expect(result.error).toContain("exit 1")
      expect(result.error).toContain("boom")
    }
    expect(exec.calls).toHaveLength(2)  // third was not run
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/steps/post-install.test.ts
```

- [ ] **Step 3: Implement post-install**

Create `src/steps/post-install.ts`:

```ts
import type { Context } from "../context"
import type { InstallResult } from "./types"

export async function runPostInstall(
  commands: string[],
  stepName: string,
  ctx: Context,
): Promise<InstallResult> {
  for (let i = 0; i < commands.length; i++) {
    const cmd = commands[i]!
    const result = await ctx.exec.run({ argv: [cmd], shell: true })
    if (result.exitCode !== 0) {
      return {
        ok: false,
        error: `${stepName} post_install[${i}] failed (exit ${result.exitCode}): ${result.stderr.trim()}`,
      }
    }
  }
  return { ok: true }
}
```

- [ ] **Step 4: Run (PASS — 3 tests)**

```bash
bun test src/steps/post-install.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/steps/post-install.ts src/steps/post-install.test.ts
git commit -m "feat(steps): add post_install runner (sequential, fail-fast)"
```

---

## Task 8: Tighten Handler.install to required, update registry

Now that all 5 handlers have install functions, update the `Handler<S>` type to require `install` (no longer optional) and update the registry to include each handler's install function.

**Files:** Modify `src/steps/types.ts`, `src/steps/index.ts`, `src/steps/index.test.ts`

- [ ] **Step 1: Update `src/steps/types.ts`**

Replace the `Handler<S>` type:

```ts
export type Handler<S extends Step> = {
  check: (step: S, ctx: Context) => Promise<CheckResult>
  install: (step: S, ctx: Context) => Promise<InstallResult>  // Phase 2: now required
}
```

(Remove the old `// Phase 2: install dispatch...` comment since install is no longer scaffolding.)

- [ ] **Step 2: Update `src/steps/index.ts`**

Add install imports and entries:

```ts
import type { Step } from "../schema"
import type { Handler } from "./types"
import { checkBrew, installBrew } from "./brew"
import { checkBrewCask, installBrewCask } from "./brew-cask"
import { checkCurlPipe, installCurlPipe } from "./curl-pipe-sh"
import { checkGitClone, installGitClone } from "./git-clone"
import { checkShell, installShell } from "./shell"

export const handlers = {
  brew:           { check: checkBrew,     install: installBrew     },
  "brew-cask":    { check: checkBrewCask, install: installBrewCask },
  "curl-pipe-sh": { check: checkCurlPipe, install: installCurlPipe },
  "git-clone":    { check: checkGitClone, install: installGitClone },
  shell:          { check: checkShell,    install: installShell    },
} satisfies { [K in Step["type"]]: Handler<Extract<Step, { type: K }>> }

export type Handlers = typeof handlers
export { type Handler, type CheckResult, type InstallResult } from "./types"
```

- [ ] **Step 3: Update `src/steps/index.test.ts`**

Replace the second test (which asserts install is undefined) with one that asserts install IS defined:

```ts
it("has an install function for every step type", () => {
  const expectedTypes = ["brew", "brew-cask", "curl-pipe-sh", "git-clone", "shell"]
  for (const type of expectedTypes) {
    expect(typeof (handlers as Record<string, { install: unknown }>)[type]?.install).toBe("function")
  }
})
```

(Keep the first test — "has a check function for every step type" — unchanged.)

- [ ] **Step 4: Run typecheck and registry tests**

```bash
bun run typecheck
bun test src/steps/index.test.ts
```
Both must pass.

- [ ] **Step 5: Run full suite to confirm nothing else broke**

```bash
bun test
```
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add src/steps/types.ts src/steps/index.ts src/steps/index.test.ts
git commit -m "feat(steps): tighten Handler.install to required, wire registry"
```

---

## Task 9: Elevation acquire (banner + Clack confirm)

Implements the elevation pause mechanism. Prints a styled banner with the user's `cfg.message`, asks for confirmation via `@clack/prompts`, returns either resolved or aborted.

**Files:** Create `src/elevation/acquire.ts`, `src/elevation/acquire.test.ts`

- [ ] **Step 1: Write failing test**

Create `src/elevation/acquire.test.ts`:

```ts
import { describe, it, expect, mock } from "bun:test"

// We mock @clack/prompts before importing acquire so the import sees the mock.
mock.module("@clack/prompts", () => ({
  confirm: mock(async () => true),
  isCancel: (v: unknown) => v === Symbol.for("clack:cancel"),
  intro: mock(() => undefined),
  outro: mock(() => undefined),
  note: mock(() => undefined),
}))

import { acquireElevation } from "./acquire"
import * as clack from "@clack/prompts"

describe("acquireElevation", () => {
  it("returns ok when the user confirms", async () => {
    ;(clack.confirm as ReturnType<typeof mock>).mockImplementation(async () => true)

    const result = await acquireElevation({ message: "Need admin", duration: "180s" })

    expect(result.ok).toBe(true)
    expect(clack.note).toHaveBeenCalled()       // banner printed
    expect(clack.confirm).toHaveBeenCalled()    // confirm prompted
  })

  it("returns aborted when the user declines", async () => {
    ;(clack.confirm as ReturnType<typeof mock>).mockImplementation(async () => false)

    const result = await acquireElevation({ message: "Need admin" })

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.reason).toContain("aborted")
    }
  })

  it("returns aborted when the user cancels (Ctrl-C)", async () => {
    const cancelSymbol = Symbol.for("clack:cancel")
    ;(clack.confirm as ReturnType<typeof mock>).mockImplementation(async () => cancelSymbol)

    const result = await acquireElevation({ message: "Need admin" })

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.reason).toContain("aborted")
    }
  })

  it("rejects an empty elevation message", async () => {
    const result = await acquireElevation({ message: "" })

    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.reason).toContain("message is required")
    }
  })
})
```

- [ ] **Step 2: Run (FAIL — module doesn't exist)**

```bash
bun test src/elevation/acquire.test.ts
```

- [ ] **Step 3: Implement `src/elevation/acquire.ts`**

```ts
import * as clack from "@clack/prompts"
import type { Config } from "../schema"

type ElevationConfig = NonNullable<Config["elevation"]>

export type AcquireResult =
  | { ok: true }
  | { ok: false; reason: string }

export async function acquireElevation(cfg: ElevationConfig): Promise<AcquireResult> {
  if (!cfg.message || cfg.message.trim() === "") {
    return { ok: false, reason: "elevation: message is required" }
  }

  // Print the styled banner using Clack's note helper.
  clack.note(cfg.message, "Elevation required")

  const confirmed = await clack.confirm({
    message: "Proceed with elevation-required steps?",
    initialValue: true,
  })

  if (clack.isCancel(confirmed) || confirmed === false) {
    return { ok: false, reason: "elevation aborted by user" }
  }

  return { ok: true }
}
```

- [ ] **Step 4: Run (PASS — 4 tests)**

```bash
bun test src/elevation/acquire.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add src/elevation/acquire.ts src/elevation/acquire.test.ts
git commit -m "feat(elevation): add acquireElevation with banner + Clack confirm"
```

---

## Task 10: runInstall — the full execution loop

Adds `runInstall(config, ctx)` to the runner. This is the heart of `gearup run`:

1. Partition steps into elevation-required vs regular
2. Pre-check all elevation-required steps (only need elevation if any are not installed)
3. If elevation needed, call `acquireElevation`; abort if user declines
4. Run elevation-required steps first, then regular steps (Go's behavior)
5. For each step: check → if installed, skip; else install → run post_install
6. Fail-fast on any error

**Files:** Modify `src/runner/run.ts`, `src/runner/run.test.ts`

- [ ] **Step 1: Append failing tests to `src/runner/run.test.ts`**

Append:

```ts
import { runInstall } from "./run"

describe("runInstall", () => {
  it("checks each step; skips installed; installs missing; runs post_install on success", async () => {
    const exec = new FakeExec()
    // step 1 (jq, brew): check exit 0 → installed, skip install
    exec.queueResponse({ exitCode: 0 })
    // step 2 (rust, shell): check exit 1 → not installed; install exit 0; post_install command exit 0
    exec.queueResponse({ exitCode: 1 })  // check
    exec.queueResponse({ exitCode: 0 })  // install
    exec.queueResponse({ exitCode: 0 })  // post_install[0]

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "mixed",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        {
          type: "shell",
          name: "rust",
          install: "curl ... | sh",
          check: "command -v rustc",
          post_install: ["rustup default stable"],
        },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(true)
    expect(report.steps).toHaveLength(2)
    expect(report.steps[0]).toEqual({ name: "jq", type: "brew", action: "skipped" })
    expect(report.steps[1]).toEqual({ name: "rust", type: "shell", action: "installed" })
  })

  it("fails fast on install failure and reports which step", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })  // jq check: not installed
    exec.queueResponse({ exitCode: 1, stderr: "boom" })  // jq install: fail

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "broken",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        { type: "brew", name: "git", formula: "git" },  // never reached
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(false)
    if (!report.ok) {
      expect(report.failedAt).toBe("jq")
      expect(report.error).toContain("brew install jq")
    }
    // Only 2 calls: jq check + jq install. git never started.
    expect(exec.calls).toHaveLength(2)
  })

  it("fails fast on check failure", async () => {
    const exec = new FakeExec()
    // Simulate exec rejecting the call (e.g., bad arg). FakeExec throws on
    // empty queue; here we queue a result with non-zero, but check itself doesn't
    // distinguish "not installed" from "check error" — both produce installed=false.
    // To test true check failure, throw from exec.
    exec.queueResponse({ exitCode: 0 })  // first step ok, installed
    // For step 2, don't queue → FakeExec throws

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "check-fail",
      steps: [
        { type: "brew", name: "jq", formula: "jq" },
        { type: "brew", name: "git", formula: "git" },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(false)
    if (!report.ok) {
      expect(report.failedAt).toBe("git")
      expect(report.error).toContain("unexpected call")
    }
  })

  it("returns ok with empty steps when config has no steps", async () => {
    const exec = new FakeExec()
    const ctx = makeContext({ exec })

    const config: Config = { version: 1, name: "empty" }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(true)
    expect(report.steps).toHaveLength(0)
  })

  it("fails fast when post_install fails (and reports which command)", async () => {
    const exec = new FakeExec()
    exec.queueResponse({ exitCode: 1 })  // check: not installed
    exec.queueResponse({ exitCode: 0 })  // install: ok
    exec.queueResponse({ exitCode: 1, stderr: "post boom" })  // post_install[0]: fail

    const ctx = makeContext({ exec })

    const config: Config = {
      version: 1,
      name: "post-fail",
      steps: [
        {
          type: "shell",
          name: "thing",
          install: "true",
          check: "false",
          post_install: ["false"],
        },
      ],
    }

    const report = await runInstall(config, ctx)

    expect(report.ok).toBe(false)
    if (!report.ok) {
      expect(report.failedAt).toBe("thing")
      expect(report.error).toContain("post_install")
    }
  })
})
```

- [ ] **Step 2: Run (FAIL — runInstall not exported)**

```bash
bun test src/runner/run.test.ts
```

- [ ] **Step 3: Implement runInstall in `src/runner/run.ts`**

**IMPORTANT:** Do NOT delete or change anything in `src/runner/run.ts` from Phase 1. The existing `PlanReport` type, `runPlan` function, and `dispatchCheck` helper must remain exactly as they are. You are appending new code only.

Add the following imports to the top of the file (the `Step` import already exists; just add what's missing):

```ts
import type { CheckResult, InstallResult } from "../steps/types"
import { runPostInstall } from "../steps/post-install"
import { acquireElevation } from "../elevation/acquire"
```

Append the following types and functions to the END of `src/runner/run.ts`:

```ts
// New types for runInstall reports.
export type StepAction = "installed" | "skipped"

export type InstallStepReport = {
  name: string
  type: Step["type"]
  action: StepAction
}

export type RunReport =
  | {
      ok: true
      configName: string
      steps: InstallStepReport[]
    }
  | {
      ok: false
      configName: string
      steps: InstallStepReport[]   // partial: steps completed before failure
      failedAt: string             // step name
      error: string                // human-readable error message
    }

async function dispatchInstall(step: Step, ctx: Context): Promise<InstallResult> {
  switch (step.type) {
    case "brew":           return handlers.brew.install(step, ctx)
    case "brew-cask":      return handlers["brew-cask"].install(step, ctx)
    case "curl-pipe-sh":   return handlers["curl-pipe-sh"].install(step, ctx)
    case "git-clone":      return handlers["git-clone"].install(step, ctx)
    case "shell":          return handlers.shell.install(step, ctx)
  }
}

// Try the check; convert any exec exception into a "check failed" result.
async function safeCheck(step: Step, ctx: Context): Promise<{ ok: true; result: CheckResult } | { ok: false; error: string }> {
  try {
    return { ok: true, result: await dispatchCheck(step, ctx) }
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : String(err) }
  }
}

export async function runInstall(config: Config, ctx: Context): Promise<RunReport> {
  const allSteps = config.steps ?? []
  const completed: InstallStepReport[] = []

  // Partition into elevation-required vs regular, preserving relative order.
  const elevSteps = allSteps.filter((s) => s.requires_elevation === true)
  const regSteps = allSteps.filter((s) => s.requires_elevation !== true)

  // Pre-check: any elevation step that needs install?
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

  // Run elevation-required steps first, then regular.
  const ordered = [...elevSteps, ...regSteps]

  for (const step of ordered) {
    const checked = await safeCheck(step, ctx)
    if (!checked.ok) {
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: step.name, error: `check failed for ${step.name}: ${checked.error}`,
      }
    }

    if (checked.result.installed) {
      completed.push({ name: step.name, type: step.type, action: "skipped" })
      continue
    }

    const installResult = await dispatchInstall(step, ctx)
    if (!installResult.ok) {
      return {
        ok: false, configName: config.name, steps: completed,
        failedAt: step.name, error: installResult.error,
      }
    }

    if (step.post_install && step.post_install.length > 0) {
      const postResult = await runPostInstall(step.post_install, step.name, ctx)
      if (!postResult.ok) {
        return {
          ok: false, configName: config.name, steps: completed,
          failedAt: step.name, error: postResult.error,
        }
      }
    }

    completed.push({ name: step.name, type: step.type, action: "installed" })
  }

  return { ok: true, configName: config.name, steps: completed }
}
```

- [ ] **Step 4: Run (PASS — 8 tests total: 3 runPlan + 5 runInstall)**

```bash
bun test src/runner/run.test.ts
```

- [ ] **Step 5: Run full suite**

```bash
bun test
bun run typecheck
```
All green; typecheck exit 0.

- [ ] **Step 6: Commit**

```bash
git add src/runner/run.ts src/runner/run.test.ts
git commit -m "feat(runner): add runInstall (check → install → post_install) with elevation"
```

---

## Task 11: gearup run command

Citty command. Loads config, runs `runInstall`, prints a report, returns exit code (0 success, 1 failure).

**Files:** Create `src/commands/run.ts`, `src/commands/run.test.ts`

- [ ] **Step 1: Write failing test**

Create `src/commands/run.test.ts`:

```ts
import { describe, it, expect, spyOn } from "bun:test"
import { runCommand } from "citty"
import path from "node:path"
import fs from "node:fs/promises"
import { runCommand as runRun } from "./run"  // exported for tests; actual command exported separately

const fixtures = path.resolve(import.meta.dir, "../../tests/fixtures")

describe("run command", () => {
  it("returns 0 on a successful run", async () => {
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)
    const tmpMarker = "/tmp/gearup-test-run-success"
    await fs.unlink(tmpMarker).catch(() => undefined)

    const fixturePath = path.join(fixtures, "__run-success.jsonc")
    await fs.writeFile(fixturePath, JSON.stringify({
      version: 1,
      name: "run-success",
      steps: [
        {
          type: "shell",
          name: "marker",
          install: `touch ${tmpMarker}`,
          check: `test -f ${tmpMarker}`,
        },
      ],
    }))

    try {
      const { result } = await runCommand(runRun, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(0)
      const output = logSpy.mock.calls.map((c) => c.join(" ")).join("\n")
      expect(output).toContain("run-success")
      expect(output).toContain("marker")
    } finally {
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
      await fs.unlink(tmpMarker).catch(() => undefined)
    }
  })

  it("returns 1 on a failed run", async () => {
    const errSpy = spyOn(console, "error").mockImplementation(() => undefined)
    const logSpy = spyOn(console, "log").mockImplementation(() => undefined)

    const fixturePath = path.join(fixtures, "__run-fail.jsonc")
    await fs.writeFile(fixturePath, JSON.stringify({
      version: 1,
      name: "run-fail",
      steps: [
        {
          type: "shell",
          name: "always-fails",
          install: "false",
          check: "false",
        },
      ],
    }))

    try {
      const { result } = await runCommand(runRun, {
        rawArgs: ["--config", fixturePath],
      })
      expect(result).toBe(1)
    } finally {
      errSpy.mockRestore()
      logSpy.mockRestore()
      await fs.unlink(fixturePath).catch(() => undefined)
    }
  })
})
```

- [ ] **Step 2: Run (FAIL)**

```bash
bun test src/commands/run.test.ts
```

- [ ] **Step 3: Implement `src/commands/run.ts`**

```ts
import { defineCommand } from "citty"
import { loadConfig } from "../config/load"
import { runInstall, type InstallStepReport } from "../runner/run"
import { makeContext } from "../context"
import { ExecaExec } from "../exec/execa"

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
    const ctx = makeContext({ exec: new ExecaExec() })
    const report = await runInstall(config, ctx)

    printReport(report.configName, report.steps)

    if (!report.ok) {
      console.error("")
      console.error(`✗ Failed at step: ${report.failedAt}`)
      console.error(`  ${report.error}`)
      return 1
    }

    console.log("")
    console.log("Done.")
    return 0
  },
})

function printReport(configName: string, steps: InstallStepReport[]): void {
  console.log(`CONFIG: ${configName}  (${steps.length} step${steps.length === 1 ? "" : "s"})`)
  console.log("")
  steps.forEach((step, i) => {
    const idx = `[${i + 1}/${steps.length}]`
    const marker = step.action === "skipped" ? "✓" : "✓"  // both are successful outcomes
    const label = step.action === "skipped" ? "already installed" : "installed"
    console.log(`  ${marker} ${idx} ${step.name}  ${label}`)
  })
}
```

- [ ] **Step 4: Run (PASS — 2 tests)**

```bash
bun test src/commands/run.test.ts
```

Note: these tests run real shell commands (`touch`, `test -f`, `false`). They should run on any unix-like system without side effects (the marker file is in `/tmp` and cleaned up).

- [ ] **Step 5: Commit**

```bash
git add src/commands/run.ts src/commands/run.test.ts
git commit -m "feat(cli): add run command (check + install + post_install)"
```

---

## Task 12: Wire `run` into the CLI entrypoint

Add `runCommand` to the `subCommands` map in `src/cli.ts`.

**Files:** Modify `src/cli.ts`

- [ ] **Step 1: Read current `src/cli.ts` to confirm shape**

```bash
cat src/cli.ts
```

It currently has `subCommands: Record<string, CommandDef<any>> = { plan: planCommand, version: versionCommand }`.

- [ ] **Step 2: Update imports and subCommands map**

Update `src/cli.ts` to import and include `runCommand`:

```ts
#!/usr/bin/env bun
import { type CommandDef, defineCommand, runMain, runCommand } from "citty"
import { planCommand } from "./commands/plan"
import { runCommand as gearupRunCommand } from "./commands/run"
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
    run: gearupRunCommand,
    version: versionCommand,
  },
})

const rawArgs = process.argv.slice(2)

// Dispatch known subcommands via runCommand so numeric exit codes are surfaced.
// citty's runMain discards subcommand return values, so it is reserved for
// meta paths (--help, --version, unknown args) that only need help rendering.
// CommandDef<any> avoids a spurious contravariance error from mismatched ArgsDef shapes.
const subCommands: Record<string, CommandDef<any>> = {
  plan: planCommand,
  run: gearupRunCommand,
  version: versionCommand,
}
const cmdName = rawArgs[0]

if (cmdName && cmdName in subCommands) {
  runCommand(subCommands[cmdName]!, { rawArgs: rawArgs.slice(1) })
    .then(({ result }) => {
      if (typeof result === "number" && result !== 0) process.exit(result)
    })
    .catch((err) => {
      console.error(err instanceof Error ? err.message : String(err))
      process.exit(1)
    })
} else {
  runMain(mainCommand)
}
```

(The pattern from Phase 1 cleanup is reused — adding `run` is just one entry in two places: the `mainCommand.subCommands` and the `subCommands` dispatch map.)

- [ ] **Step 3: Smoke-test**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun run src/cli.ts --help
```
Expected: USAGE section listing `plan`, `run`, `version`.

```bash
bun run src/cli.ts run --help
```
Expected: shows `run`'s description and `--config` argument.

- [ ] **Step 4: Run full suite**

```bash
bun test
```
Expected: all tests still pass.

- [ ] **Step 5: Commit**

```bash
git add src/cli.ts
git commit -m "feat(cli): wire run subcommand into entrypoint"
```

---

## Task 13: Create test fixtures for Phase 2 e2e

Two fixtures:
1. `safe-install.jsonc` — uses shell type to safely create and check a marker file. End-to-end testable on any unix system.
2. `elevation-required.jsonc` — has a step with `requires_elevation: true` and an `elevation:` block. Used to verify elevation pre-check logic (note: e2e test will skip the actual confirmation since stdin is closed in a subprocess — see Task 14).

**Files:** Create `tests/fixtures/safe-install.jsonc`, `tests/fixtures/elevation-required.jsonc`

- [ ] **Step 1: Create `tests/fixtures/safe-install.jsonc`**

```jsonc
{
  "version": 1,
  "name": "safe-install",
  "steps": [
    {
      "type": "shell",
      "name": "marker",
      "install": "touch /tmp/gearup-e2e-marker",
      "check": "test -f /tmp/gearup-e2e-marker",
      "post_install": ["echo post-install ran > /tmp/gearup-e2e-post"]
    }
  ]
}
```

- [ ] **Step 2: Create `tests/fixtures/elevation-required.jsonc`**

```jsonc
{
  "version": 1,
  "name": "elevation-required",
  "elevation": {
    "message": "This run needs admin permissions for one step. Acquire them, then continue.",
    "duration": "180s"
  },
  "steps": [
    {
      "type": "shell",
      "name": "needs-admin",
      "install": "true",
      "check": "false",
      "requires_elevation": true
    }
  ]
}
```

- [ ] **Step 3: Commit**

```bash
git add tests/fixtures/safe-install.jsonc tests/fixtures/elevation-required.jsonc
git commit -m "test: add safe-install and elevation-required fixtures"
```

---

## Task 14: End-to-end test for `gearup run`

Spawns `bun run src/cli.ts run` with the safe-install fixture; verifies:
- Exit 0
- Marker file created (install ran)
- Post-install marker created (post_install ran)
- Re-run shows "already installed" (idempotency) and exits 0

**Files:** Modify `tests/e2e.test.ts`

- [ ] **Step 1: Append to `tests/e2e.test.ts`**

Add `import fs from "node:fs/promises"` to the imports at the top of the file (it's not currently imported). Then append this `describe` block to the end:

```ts
describe("e2e: gearup run (safe-install)", () => {
  const installMarker = "/tmp/gearup-e2e-marker"
  const postMarker = "/tmp/gearup-e2e-post"
  const fixture = path.join(fixtures, "safe-install.jsonc")
  const cliPath = path.join(repoRoot, "src/cli.ts")

  it("installs a missing step, runs post_install, then is idempotent on re-run", async () => {
    // Arrange: clean state
    await fs.unlink(installMarker).catch(() => undefined)
    await fs.unlink(postMarker).catch(() => undefined)

    try {
      // First run — install path
      const first = await $`bun run ${cliPath} run --config ${fixture}`.quiet().nothrow()
      expect(first.exitCode).toBe(0)
      expect(first.stdout.toString()).toContain("safe-install")
      expect(first.stdout.toString()).toContain("marker")
      expect(first.stdout.toString()).toContain("Done.")

      // Verify side effects
      await fs.access(installMarker)  // throws if missing
      const postContents = await fs.readFile(postMarker, "utf8")
      expect(postContents.trim()).toBe("post-install ran")

      // Second run — should skip (already installed)
      const second = await $`bun run ${cliPath} run --config ${fixture}`.quiet().nothrow()
      expect(second.exitCode).toBe(0)
      expect(second.stdout.toString()).toContain("already installed")
    } finally {
      await fs.unlink(installMarker).catch(() => undefined)
      await fs.unlink(postMarker).catch(() => undefined)
    }
  })

  it("returns non-zero when a step fails", async () => {
    const failFixturePath = path.join(fixtures, "__e2e-fail.jsonc")
    await fs.writeFile(failFixturePath, JSON.stringify({
      version: 1,
      name: "e2e-fail",
      steps: [
        { type: "shell", name: "always-fails", install: "false", check: "false" },
      ],
    }))

    try {
      const result = await $`bun run ${cliPath} run --config ${failFixturePath}`.quiet().nothrow()
      expect(result.exitCode).toBe(1)
      expect(result.stderr.toString() + result.stdout.toString()).toContain("always-fails")
    } finally {
      await fs.unlink(failFixturePath).catch(() => undefined)
    }
  })
})
```

- [ ] **Step 2: Run e2e tests**

```bash
bun test tests/e2e.test.ts
```
Expected: all pass (Phase 1 e2e + Phase 2 e2e). The Phase 2 tests run real shell commands but are scoped to /tmp markers.

- [ ] **Step 3: Run full suite (final acceptance)**

```bash
bun test
bun run typecheck
```
All green; typecheck exit 0. Test count should be in the 75-85 range.

- [ ] **Step 4: Commit**

```bash
git add tests/e2e.test.ts
git commit -m "test: add e2e tests for gearup run (install + idempotency + failure)"
```

---

## Task 15: Update spec to reflect Clack moving to Phase 2

The design spec at `docs/superpowers/specs/2026-04-24-typescript-port-design.md` says `@clack/prompts` is Phase 4. Phase 2 brought it forward for the elevation banner.

**Files:** Modify `docs/superpowers/specs/2026-04-24-typescript-port-design.md`

- [ ] **Step 1: Update the tech stack table**

Find the line:
```
| Picker / TUI (Phase 4) | **@clack/prompts** | The Astro-originated picker library. ... |
```

Change to:
```
| Elevation banner (Phase 2) + Picker (Phase 4) | **@clack/prompts** | The Astro-originated picker library. Brought forward to Phase 2 for the elevation banner; Phase 4 adds the picker. |
```

- [ ] **Step 2: Update the "Open questions deferred to implementation" entry**

Find:
```
- **Elevation prompt mechanism in Phase 2.** Uses Node's built-in `readline` for a minimal confirmation (~10 lines, zero deps). Phase 4 replaces this with `@clack/prompts` for the styled banner. Keeping the dep graph flat in Phase 2 is worth more than prettier output that gets replaced next phase.
```

Change to:
```
- **Elevation prompt mechanism in Phase 2.** Uses `@clack/prompts` (`clack.note` for the banner + `clack.confirm` for the prompt). Originally planned as `readline`, but the readline UX would be a regression vs the Charm-based Go version, so Clack was brought forward from Phase 4 — one extra dep that we know we want anyway.
```

- [ ] **Step 3: Update the Phasing table — Phase 2 row**

Find the Phase 2 row and update its Scope to mention `@clack/prompts/confirm`:

Change:
```
Install dispatch for all 5 types • `post_install` hooks • elevation pause banner (simple prompt for confirmation) • still stdout, no log file yet
```

to:

```
Install dispatch for all 5 types • `post_install` hooks • elevation pause banner via `@clack/prompts` (note + confirm) • still stdout, no log file yet
```

- [ ] **Step 4: Update the architecture doc tech stack table**

Open `docs/ts-port-architecture.md`. Find:
```
| Picker / TUI (Phase 4) | **@clack/prompts** |
```

Change to:
```
| Elevation banner (Phase 2) + Picker (Phase 4) | **@clack/prompts** |
```

- [ ] **Step 5: Verify nothing else references the old Phase 4 placement**

```bash
grep -rn "Phase 4.*clack\|clack.*Phase 4\|readline" docs/ src/ 2>/dev/null
```

If anything else references readline-for-elevation or Clack-as-Phase-4-only, fix it.

- [ ] **Step 6: Commit**

```bash
git add docs/superpowers/specs/2026-04-24-typescript-port-design.md docs/ts-port-architecture.md
git commit -m "docs: reflect @clack/prompts moving to Phase 2 for elevation banner"
```

---

## Phase 2 Done — Verify and Merge Back

- [ ] **Step 1: Final verification**

```bash
export PATH="$HOME/.bun/bin:$PATH"
bun test         # all pass
bun run typecheck    # exit 0
git log --oneline main..HEAD | wc -l    # ~16 commits
```

- [ ] **Step 2: Smoke-test the new run command end-to-end**

```bash
bun run src/cli.ts --help                     # Lists plan, run, version
bun run src/cli.ts run --config tests/fixtures/safe-install.jsonc; echo $?  # 0; marker created
bun run src/cli.ts run --config tests/fixtures/safe-install.jsonc; echo $?  # 0; "already installed"
rm /tmp/gearup-e2e-marker /tmp/gearup-e2e-post 2>/dev/null  # cleanup
```

- [ ] **Step 3: Push and create PR**

```bash
git push -u origin ts-port/phase-2-run-and-elevation
gh pr create --title "feat: TS port Phase 2 — run command + install + elevation" --body "..."
```

(Construct the PR body when you get there — summarize the 5 install handlers, post_install runner, elevation acquire, runInstall orchestration, and the run command, with a test plan that covers `safe-install` and `elevation-required` fixtures.)

---

## Exit Criteria for Phase 2

1. `gearup run --config <path>` works end-to-end: parses config, runs check → install → post_install per step, partitions elevation steps, shows the Clack banner when needed, fail-fast on first error.
2. All 5 step types have install handlers with FakeExec-driven unit tests.
3. `Handler.install` is now required in the type — `satisfies` would fail compilation if any handler dropped it.
4. `post_install` runner is sequential and fail-fast, with its own tests.
5. Elevation banner shows only when at least one elevation-required step is not installed.
6. End-to-end test exercises the full pipeline against `safe-install.jsonc` and verifies idempotency on re-run.
7. Test count in the 75-85 range; typecheck exit 0; existing Phase 1 tests still pass.

## What's NOT in Phase 2 (still deferred)

- `extends:` composition — Phase 3
- XDG logging with captured subprocess output — Phase 3
- Interactive picker when `--config` omitted — Phase 4
- Animated spinners during install — Phase 4
- Styled `intro/outro` flow — Phase 4
- `gearup init` + embedded default configs — Phase 4
- Release pipeline (`bun build --compile`) — Phase 4
- Go code removal — Phase 4
